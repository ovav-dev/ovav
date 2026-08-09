/**
 * OVAV MEMORY v3 — HarnessInjector
 * Interface for injecting Cells into the pi.dev agent runtime.
 */
import { type CellStore } from "./cellstore.js";
/**
 * HarnessInjector translates a Cell into a pi.dev tool-call injection.
 * pi.dev calls the `before_agent_start` and `tool_call` hooks to integrate.
 */
export interface InjectionContext {
    sessionId: string;
    projectPath: string;
    harnessScope: string;
}
export interface InjectionResult {
    injected: boolean;
    cellId: string;
    content: string;
    tokens: number;
}
/**
 * The HarnessInjector interface — implemented by the pi.dev extension adapter.
 */
export interface IHarnessInjector {
    /**
     * Called before the agent starts a new turn.
     * Returns content to prepend to the system prompt / context.
     */
    beforeAgentStart(ctx: InjectionContext): Promise<string>;
    /**
     * Called after a tool call completes — returns injection if signal detected.
     */
    afterToolCall(ctx: InjectionContext, tool: string, file: string | undefined, result: unknown, error: string | undefined): Promise<InjectionResult | null>;
    /**
     * Record whether the last injection was helpful.
     */
    recordOutcome(ctx: InjectionContext, cellId: string, helped: boolean): Promise<void>;
}
/**
 * Default HarnessInjector using CellStore and Detector.
 */
export declare class HarnessInjector implements IHarnessInjector {
    private readonly cellStore;
    private readonly getDetector;
    constructor(cellStore: CellStore, getDetector: () => {
        decide: (e: import("./detector.js").ToolCallEvent) => import("./detector.js").InjectionDecision;
    });
    beforeAgentStart(_ctx: InjectionContext): Promise<string>;
    afterToolCall(ctx: InjectionContext, tool: string, file: string | undefined, result: unknown, error: string | undefined): Promise<InjectionResult | null>;
    recordOutcome(ctx: InjectionContext, cellId: string, helped: boolean): Promise<void>;
    private formatCell;
    private estimateTokens;
}
