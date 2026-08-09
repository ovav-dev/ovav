/**
 * OVAV MEMORY v3 — LiveProfiler
 * Sliding-window event profiler for detecting struggle signals.
 */
export declare enum SignalType {
    ERROR_LOOP = "error_loop",
    EDIT_BURST = "edit_burst",
    EMPTY_RESULT = "empty_result",
    CONTEXT_PRESS = "context_pressure",
    RETRY_LOOP = "retry_loop",
    STUCK = "stuck_in_loop"
}
interface ProfiledEvent {
    ts: Date;
    data: unknown;
}
/**
 * LiveProfiler maintains a 5-minute sliding window of tool events
 * and classifies patterns into SignalType signals.
 */
export declare class LiveProfiler {
    private readonly windowMs;
    private events;
    constructor(windowMinutes?: number);
    /**
     * Feed a new event into the profiler.
     * @param file  - identifier of the file/tool context
     * @param operation - type of operation (error, write, read, etc.)
     * @param data  - arbitrary event data
     */
    feed(file: string, operation: string, data: unknown): void;
    /**
     * Returns all current signal classifications from the window.
     */
    classifyAll(): Array<{
        key: string;
        signal: SignalType;
        count: number;
    }>;
    /**
     * Returns the dominant signal type if one exists, null otherwise.
     */
    classify(): SignalType | null;
    /**
     * Returns events for a specific file:operation key.
     */
    getEvents(file: string, operation: string): ProfiledEvent[];
    /**
     * Clears all profiled events.
     */
    reset(): void;
    private makeKey;
    private classifyWindow;
}
export {};
