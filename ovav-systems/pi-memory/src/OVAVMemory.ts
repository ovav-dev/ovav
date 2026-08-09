/**
 * OVAV MEMORY v3 — Main Orchestrator for pi Extension
 *
 * Bridges pi extension events to OVAV MEMORY core:
 *   CellStore, LiveProfiler, Detector, HarnessInjector,
 *   GovernanceRegistry, PermissionGate, AuditLogger.
 */

import { CellStore } from "./ovav-memory/cellstore.js";
import { Detector } from "./ovav-memory/detector.js";
import { LiveProfiler, SignalType } from "./ovav-memory/profiler.js";
import { HarnessInjector } from "./ovav-memory/injector.js";
import { AuditLogger } from "./ovav-audit/logger.js";
import { PermissionGate } from "./ovav-permissions/gate.js";
import { OVAVMemoryClient } from "./ovav-memory/http_client.js";
import {
  runOvavValidate,
  runOvavDaily,
  runOvavNextWork,
  runOvavCheckIntegrity,
} from "./ovav-governance/registry.js";

// ── Types ───────────────────────────────────────────────────

export interface OVAVMemoryConfig {
  projectPath: string;
  harnessScope?: string;
  minWeight?: number;
}

export interface SessionStartEvent {
  sessionId: string;
  reason: string;
}

export interface BeforeAgentStartEvent {
  prompt: string;
  systemPrompt: string;
}

export interface ToolCallEvent {
  toolName: string;
  input: Record<string, unknown>;
}

export interface ToolResultEvent {
  toolName: string;
  input: Record<string, unknown>;
  result: unknown;
  isError: boolean;
}

export interface TurnEndEvent {
  turnIndex: number;
  message: unknown;
}

// ── OVAVMemory ───────────────────────────────────────────────

export class OVAVMemory {
  private cellStore: CellStore;
  private profiler: LiveProfiler;
  private detector: Detector;
  private injector: HarnessInjector;
  private auditLogger: AuditLogger;
  private permissionGate: PermissionGate;
  private client: OVAVMemoryClient;
  private projectPath: string;
  private harnessScope: string;
  private minWeight: number;
  private sessionId = "unknown";
  private cellsInjected = 0;

  constructor(config: OVAVMemoryConfig) {
    this.projectPath = config.projectPath;
    this.harnessScope = config.harnessScope ?? "pi";
    this.minWeight = config.minWeight ?? 0.6;

    this.cellStore = new CellStore(this.projectPath);
    this.profiler = new LiveProfiler(300_000); // 5-minute window
    this.detector = new Detector(this.cellStore, this.minWeight);
    this.injector = new HarnessInjector(this.cellStore, () => this.detector);
    this.auditLogger = new AuditLogger(this.projectPath);
    this.permissionGate = new PermissionGate(this.projectPath);
    this.client = new OVAVMemoryClient();
  }

  // ── Lifecycle ─────────────────────────────────────────────

  async onSessionStart(event: SessionStartEvent): Promise<void> {
    this.sessionId = event.sessionId;
    await this.cellStore.load();
    await this.auditLogger.init();
    this.auditLogger.log({
      event: "session_start",
      sessionId: event.sessionId,
      projectPath: this.projectPath,
    });
  }

  async onSessionShutdown(_event: { reason: string }): Promise<void> {
    this.auditLogger.log({
      event: "session_end",
      sessionId: this.sessionId,
      projectPath: this.projectPath,
    });
  }

  // ── Memory injection ─────────────────────────────────────

  async onBeforeAgentStart(
    event: BeforeAgentStartEvent
  ): Promise<{ content: string } | null> {
    // Run detector to check for matching signals
    const decision = this.detector.decide({
      tool: "agent",
      file: "prompt",
      result: { prompt: event.prompt },
    });

    if (!decision.inject) return null;

    // Build injection content
    const cell = decision.inject;
    const signal = decision.signal ?? "unknown";

    this.cellsInjected++;

    const content = [
      `<ovav_memory_inject>`,
      `<!-- Signal: ${signal} | Cell: ${cell.id} | Weight: ${cell.weight} -->`,
      ``,
      cell.summary,
      ``,
      `<!-- Tags: ${(cell.tags ?? []).join(", ")} -->`,
      `</ovav_memory_inject>`,
    ].join("\n");

    // Log injection
    this.auditLogger.logCellInject(cell.id, cell.eventSignature, cell.weight, true);

    return { content };
  }

  // ── Tool events ──────────────────────────────────────────

  async onToolCall(
    event: ToolCallEvent
  ): Promise<{ blocked: boolean; reason?: string } | null> {
    const { toolName, input } = event;

    // Permission gate check
    if (toolName === "bash" || toolName === "shell") {
      const cmd = (input.command as string) ?? (input.cmd as string) ?? "";
      const evalResult = this.permissionGate.evaluateCommand(cmd);
      if (!evalResult.allowed) {
        this.auditLogger.logPermissionBlocked(cmd, evalResult.reason ?? "PermissionGate denied");
        return { blocked: true, reason: evalResult.reason };
      }
      this.auditLogger.log({
        event: "permission_allowed",
        sessionId: this.sessionId,
        projectPath: this.projectPath,
        command: cmd.substring(0, 200),
      });
    }

    // Feed error signals to profiler
    if (
      toolName === "bash" ||
      toolName === "write" ||
      toolName === "edit"
    ) {
      const error = input.error as string | undefined;
      const exitCode = input.exitCode as number | undefined;

      if (error || (exitCode !== undefined && exitCode !== 0)) {
        this.profiler.feed(
          (input.file as string) ?? "unknown",
          "error",
          { tool: toolName, error: error ?? `exit ${exitCode}` }
        );
      }

      if (toolName === "write" || toolName === "edit") {
        this.profiler.feed(
          (input.file as string) ?? "unknown",
          "write",
          { tool: toolName }
        );
      }
    }

    return null;
  }

  async onToolResult(event: ToolResultEvent): Promise<void> {
    const { toolName, result, isError } = event;

    // Track outcome for weight update
    if (toolName === "bash" && result) {
      const resultStr = String(result);
      if (
        resultStr.includes("error") ||
        resultStr.includes("Error") ||
        isError
      ) {
        this.profiler.feed(
          (event.input.file as string) ?? "bash",
          "error",
          { tool: toolName, result: resultStr.substring(0, 200) }
        );
      }
    }
  }

  async onTurnEnd(event: TurnEndEvent): Promise<void> {
    const { turnIndex } = event;

    // Classify signals at turn end
    const signal = this.profiler.classify();
    if (signal) {
      this.auditLogger.log({
        event: "turn_end",
        sessionId: this.sessionId,
        projectPath: this.projectPath,
      });
    }
  }

  // ── Governance tools ─────────────────────────────────────

  async runValidate(
    scope: "all" | "f0" | "f1" | "f2" | "f3" | "f4" | "f5" = "all"
  ): Promise<unknown> {
    return runOvavValidate(this.projectPath, scope);
  }

  async runDailyBrief(): Promise<unknown> {
    return runOvavDaily(this.projectPath);
  }

  async runNextWork(): Promise<unknown> {
    return runOvavNextWork(this.projectPath);
  }

  async runIntegrityCheck(): Promise<unknown> {
    return runOvavCheckIntegrity(this.projectPath);
  }

  // ── Memory tools via cPanel relay ───────────────────────────

  async callMemoryTool(toolName: string, args: Record<string, unknown> = {}): Promise<string> {
    return this.client.callTool(toolName, args);
  }

  // ── Stats ────────────────────────────────────────────────

  cellCount(): number {
    return this.cellStore.all().length;
  }

  getStats(): {
    cells: number;
    activeSignals: number;
    cellsInjected: number;
    sessionId: string;
  } {
    return {
      cells: this.cellStore.all().length,
      activeSignals: this.profiler.classifyAll().length,
      cellsInjected: this.cellsInjected,
      sessionId: this.sessionId,
    };
  }
}
