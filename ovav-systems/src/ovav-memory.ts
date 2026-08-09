/**
 * OVAV MEMORY v3 — Main Extension Entry Point
 *
 * This is the module that pi.dev loads as `ovav-systems`.
 * It exports the hooks: before_agent_start, tool_call, turn_end, project_trust
 * and the tool registry for governance tools.
 */

import { CellStore } from "./ovav-memory/cellstore.js";
import { Detector, type ToolCallEvent } from "./ovav-memory/detector.js";
import { HarnessInjector } from "./ovav-memory/injector.js";
import { AuditLogger } from "./ovav-audit/logger.js";
import { PermissionGate } from "./ovav-permissions/gate.js";
import {
  runOvavValidate,
  runOvavDaily,
  runOvavNextWork,
  runOvavCheckIntegrity,
} from "./ovav-governance/registry.js";

export { CellStore } from "./ovav-memory/cellstore.js";
export { Detector } from "./ovav-memory/detector.js";
export { HarnessInjector } from "./ovav-memory/injector.js";
export { AuditLogger } from "./ovav-audit/logger.js";
export { PermissionGate } from "./ovav-permissions/gate.js";

// ── Extension State ─────────────────────────────────────────────────────────────

interface ExtensionState {
  cellStore: CellStore;
  detector: Detector;
  injector: HarnessInjector;
  audit: AuditLogger;
  permissions: PermissionGate;
  initialized: boolean;
}

function createState(basePath: string): ExtensionState {
  const cellStore = new CellStore(basePath);
  const detector = new Detector(cellStore);
  const audit = new AuditLogger(basePath);
  const permissions = new PermissionGate(basePath);

  const getDetector = () => detector;
  const injector = new HarnessInjector(cellStore, getDetector);

  return { cellStore, detector, injector, audit, permissions, initialized: false };
}

// ── Hook Handlers ─────────────────────────────────────────────────────────────

/**
 * before_agent_start hook — inject relevant cells before agent starts.
 */
export async function onBeforeAgentStart(
  ctx: { sessionId: string; projectPath: string; harnessScope?: string }
): Promise<{ content: string; state: ExtensionState }> {
  const state = createState(ctx.projectPath);

  if (!state.initialized) {
    await state.cellStore.load();
    await state.audit.init();
    state.initialized = true;
  }

  await state.audit.log({
    event: "session_start",
    sessionId: ctx.sessionId,
    scope: ctx.harnessScope ?? "pi",
  });

  const content = await state.injector.beforeAgentStart({
    sessionId: ctx.sessionId,
    projectPath: ctx.projectPath,
    harnessScope: ctx.harnessScope ?? "pi",
  });

  return { content, state };
}

/**
 * tool_call hook — analyze each tool call for struggle signals.
 */
export async function onToolCall(
  ctx: { sessionId: string; projectPath: string },
  event: ToolCallEvent
): Promise<{ injection: import("./ovav-memory/injector.js").InjectionResult | null; state: ExtensionState }> {
  const state = createState(ctx.projectPath);

  if (!state.initialized) {
    await state.cellStore.load();
    await state.audit.init();
    state.initialized = true;
  }

  const result = await state.injector.afterToolCall(
    {
      sessionId: ctx.sessionId,
      projectPath: ctx.projectPath,
      harnessScope: "pi",
    },
    event.tool,
    event.file,
    event.result,
    event.error
  );

  if (result?.injected) {
    await state.audit.logCellInject(
      result.cellId,
      "tool_call_signal",
      0, // weight fetched from cell
      true
    );
  }

  return { injection: result, state };
}

/**
 * turn_end hook — record outcomes and finalize.
 */
export async function onTurnEnd(
  ctx: { sessionId: string; projectPath: string },
  outcome: { cellId?: string; helped?: boolean }
): Promise<void> {
  const state = createState(ctx.projectPath);

  if (!state.initialized) return;

  await state.audit.log({
    event: "turn_end",
    sessionId: ctx.sessionId,
    cellId: outcome.cellId,
    data: { helped: outcome.helped },
  });

  if (outcome.cellId && outcome.helped !== undefined) {
    await state.cellStore.recordOutcome(outcome.cellId, outcome.helped);
  }
}

/**
 * project_trust hook — evaluate trust level for a project.
 */
export function onProjectTrust(
  projectPath: string,
  requestedPermission: string,
  detail?: string
): { allowed: boolean; reason: string } {
  const permissions = new PermissionGate(projectPath);

  if (requestedPermission === "bash" && detail) {
    return permissions.evaluateCommand(detail);
  }

  if (requestedPermission === "external_directory" && detail) {
    return permissions.evaluateExternalDirectory(detail);
  }

  if (requestedPermission === "env" && detail) {
    return permissions.evaluateEnvAccess(detail);
  }

  return { allowed: true, reason: "default_allow" };
}

// ── Tool Handlers (for ovav_* tools) ──────────────────────────────────────────

export async function handleTool(
  toolName: string,
  basePath: string,
  args?: Record<string, unknown>
): Promise<unknown> {
  switch (toolName) {
    case "ovav_validate": {
      const scope = (args?.scope as "all" | "f0" | "f1" | "f2" | "f3" | "f4" | "f5") ?? "all";
      return runOvavValidate(basePath, scope);
    }
    case "ovav_daily":
      return runOvavDaily(basePath);
    case "ovav_next_work":
      return runOvavNextWork(basePath);
    case "ovav_check_integrity":
      return runOvavCheckIntegrity(basePath);
    default:
      return { error: `Unknown tool: ${toolName}` };
  }
}
