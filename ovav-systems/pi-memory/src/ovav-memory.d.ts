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
export { CellStore } from "./ovav-memory/cellstore.js";
export { Detector } from "./ovav-memory/detector.js";
export { HarnessInjector } from "./ovav-memory/injector.js";
export { AuditLogger } from "./ovav-audit/logger.js";
export { PermissionGate } from "./ovav-permissions/gate.js";
interface ExtensionState {
    cellStore: CellStore;
    detector: Detector;
    injector: HarnessInjector;
    audit: AuditLogger;
    permissions: PermissionGate;
    initialized: boolean;
}
/**
 * before_agent_start hook — inject relevant cells before agent starts.
 */
export declare function onBeforeAgentStart(ctx: {
    sessionId: string;
    projectPath: string;
    harnessScope?: string;
}): Promise<{
    content: string;
    state: ExtensionState;
}>;
/**
 * tool_call hook — analyze each tool call for struggle signals.
 */
export declare function onToolCall(ctx: {
    sessionId: string;
    projectPath: string;
}, event: ToolCallEvent): Promise<{
    injection: import("./ovav-memory/injector.js").InjectionResult | null;
    state: ExtensionState;
}>;
/**
 * turn_end hook — record outcomes and finalize.
 */
export declare function onTurnEnd(ctx: {
    sessionId: string;
    projectPath: string;
}, outcome: {
    cellId?: string;
    helped?: boolean;
}): Promise<void>;
/**
 * project_trust hook — evaluate trust level for a project.
 */
export declare function onProjectTrust(projectPath: string, requestedPermission: string, detail?: string): {
    allowed: boolean;
    reason: string;
};
export declare function handleTool(toolName: string, basePath: string, args?: Record<string, unknown>): Promise<unknown>;
