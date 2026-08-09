/**
 * OVAV MEMORY v3 — Main Orchestrator for pi Extension
 *
 * Bridges pi extension events to OVAV MEMORY core:
 *   CellStore, LiveProfiler, Detector, HarnessInjector,
 *   GovernanceRegistry, PermissionGate, AuditLogger.
 */
import { CellStore } from "./ovav-memory/cellstore.js";
import { Detector } from "./ovav-memory/detector.js";
import { LiveProfiler } from "./ovav-memory/profiler.js";
import { HarnessInjector } from "./ovav-memory/injector.js";
import { AuditLogger } from "./ovav-audit/logger.js";
import { PermissionGate } from "./ovav-permissions/gate.js";
import { OVAVMemoryClient } from "./ovav-memory/http_client.js";
import { runOvavValidate, runOvavDaily, runOvavNextWork, runOvavCheckIntegrity, } from "./ovav-governance/registry.js";
// ── OVAVMemory ───────────────────────────────────────────────
export class OVAVMemory {
    cellStore;
    profiler;
    detector;
    injector;
    auditLogger;
    permissionGate;
    client;
    projectPath;
    harnessScope;
    minWeight;
    sessionId = "unknown";
    cellsInjected = 0;
    constructor(config) {
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
    async onSessionStart(event) {
        this.sessionId = event.sessionId;
        await this.cellStore.load();
        await this.auditLogger.init();
        this.auditLogger.log({
            event: "session_start",
            sessionId: event.sessionId,
            projectPath: this.projectPath,
        });
    }
    async onSessionShutdown(_event) {
        this.auditLogger.log({
            event: "session_end",
            sessionId: this.sessionId,
            projectPath: this.projectPath,
        });
    }
    // ── Memory injection ─────────────────────────────────────
    async onBeforeAgentStart(event) {
        // Run detector to check for matching signals
        const decision = this.detector.decide({
            tool: "agent",
            file: "prompt",
            result: { prompt: event.prompt },
        });
        if (!decision.inject)
            return null;
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
    async onToolCall(event) {
        const { toolName, input } = event;
        // Permission gate check
        if (toolName === "bash" || toolName === "shell") {
            const cmd = input.command ?? input.cmd ?? "";
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
        if (toolName === "bash" ||
            toolName === "write" ||
            toolName === "edit") {
            const error = input.error;
            const exitCode = input.exitCode;
            if (error || (exitCode !== undefined && exitCode !== 0)) {
                this.profiler.feed(input.file ?? "unknown", "error", { tool: toolName, error: error ?? `exit ${exitCode}` });
            }
            if (toolName === "write" || toolName === "edit") {
                this.profiler.feed(input.file ?? "unknown", "write", { tool: toolName });
            }
        }
        return null;
    }
    async onToolResult(event) {
        const { toolName, result, isError } = event;
        // Track outcome for weight update
        if (toolName === "bash" && result) {
            const resultStr = String(result);
            if (resultStr.includes("error") ||
                resultStr.includes("Error") ||
                isError) {
                this.profiler.feed(event.input.file ?? "bash", "error", { tool: toolName, result: resultStr.substring(0, 200) });
            }
        }
    }
    async onTurnEnd(event) {
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
    async runValidate(scope = "all") {
        return runOvavValidate(this.projectPath, scope);
    }
    async runDailyBrief() {
        return runOvavDaily(this.projectPath);
    }
    async runNextWork() {
        return runOvavNextWork(this.projectPath);
    }
    async runIntegrityCheck() {
        return runOvavCheckIntegrity(this.projectPath);
    }
    // ── Memory tools via cPanel relay ───────────────────────────
    async callMemoryTool(toolName, args = {}) {
        return this.client.callTool(toolName, args);
    }
    // ── Stats ────────────────────────────────────────────────
    cellCount() {
        return this.cellStore.all().length;
    }
    getStats() {
        return {
            cells: this.cellStore.all().length,
            activeSignals: this.profiler.classifyAll().length,
            cellsInjected: this.cellsInjected,
            sessionId: this.sessionId,
        };
    }
}
//# sourceMappingURL=OVAVMemory.js.map