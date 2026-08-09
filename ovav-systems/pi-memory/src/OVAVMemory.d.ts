/**
 * OVAV MEMORY v3 — Main Orchestrator for pi Extension
 *
 * Bridges pi extension events to OVAV MEMORY core:
 *   CellStore, LiveProfiler, Detector, HarnessInjector,
 *   GovernanceRegistry, PermissionGate, AuditLogger.
 */
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
export declare class OVAVMemory {
    private cellStore;
    private profiler;
    private detector;
    private injector;
    private auditLogger;
    private permissionGate;
    private client;
    private projectPath;
    private harnessScope;
    private minWeight;
    private sessionId;
    private cellsInjected;
    constructor(config: OVAVMemoryConfig);
    onSessionStart(event: SessionStartEvent): Promise<void>;
    onSessionShutdown(_event: {
        reason: string;
    }): Promise<void>;
    onBeforeAgentStart(event: BeforeAgentStartEvent): Promise<{
        content: string;
    } | null>;
    onToolCall(event: ToolCallEvent): Promise<{
        blocked: boolean;
        reason?: string;
    } | null>;
    onToolResult(event: ToolResultEvent): Promise<void>;
    onTurnEnd(event: TurnEndEvent): Promise<void>;
    runValidate(scope?: "all" | "f0" | "f1" | "f2" | "f3" | "f4" | "f5"): Promise<unknown>;
    runDailyBrief(): Promise<unknown>;
    runNextWork(): Promise<unknown>;
    runIntegrityCheck(): Promise<unknown>;
    callMemoryTool(toolName: string, args?: Record<string, unknown>): Promise<string>;
    cellCount(): number;
    getStats(): {
        cells: number;
        activeSignals: number;
        cellsInjected: number;
        sessionId: string;
    };
}
//# sourceMappingURL=OVAVMemory.d.ts.map