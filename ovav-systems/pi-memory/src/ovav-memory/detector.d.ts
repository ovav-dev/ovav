/**
 * OVAV MEMORY v3 — Detector
 * Analyzes tool call patterns and triggers Cell injection when struggle signals fire.
 */
import { type SignalType } from "./profiler.js";
import { CellStore } from "./cellstore.js";
import { type Cell } from "./cell.js";
export interface InjectionDecision {
    inject: Cell | null;
    signal: SignalType | null;
    sig: string | null;
    reason: string;
}
export interface ToolCallEvent {
    tool: string;
    file?: string;
    result: unknown;
    error?: string;
}
/**
 * Detector watches tool_call events from pi.dev and decides when to inject a Cell.
 *
 * Pattern:
 *   1. LiveProfiler receives tool call events
 *   2. If a SignalType is classified (≥3 events in window)
 *   3. Build event signature → CellStore lookup
 *   4. If a Cell with weight ≥ 0.6 exists → return it for injection
 */
export declare class Detector {
    private readonly profiler;
    private readonly cellStore;
    private readonly minWeight;
    constructor(cellStore: CellStore, minWeight?: number);
    /**
     * Called on every tool_call event from pi.dev runtime.
     */
    onToolCall(event: ToolCallEvent): void;
    /**
     * Decide whether to inject a Cell given a tool call result.
     * Returns the Cell and signal if injection is warranted.
     */
    decide(event: ToolCallEvent): InjectionDecision;
    /**
     * Returns current profiler state for debugging.
     */
    snapshot(): Array<{
        key: string;
        signal: SignalType;
        count: number;
    }>;
    /**
     * Resets the profiler window.
     */
    reset(): void;
}
