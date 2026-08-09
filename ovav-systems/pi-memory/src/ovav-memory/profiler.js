/**
 * OVAV MEMORY v3 — LiveProfiler
 * Sliding-window event profiler for detecting struggle signals.
 */
export var SignalType;
(function (SignalType) {
    SignalType["ERROR_LOOP"] = "error_loop";
    SignalType["EDIT_BURST"] = "edit_burst";
    SignalType["EMPTY_RESULT"] = "empty_result";
    SignalType["CONTEXT_PRESS"] = "context_pressure";
    SignalType["RETRY_LOOP"] = "retry_loop";
    SignalType["STUCK"] = "stuck_in_loop";
})(SignalType || (SignalType = {}));
/**
 * LiveProfiler maintains a 5-minute sliding window of tool events
 * and classifies patterns into SignalType signals.
 */
export class LiveProfiler {
    windowMs;
    events = new Map();
    constructor(windowMinutes = 5) {
        this.windowMs = windowMinutes * 60 * 1000;
    }
    /**
     * Feed a new event into the profiler.
     * @param file  - identifier of the file/tool context
     * @param operation - type of operation (error, write, read, etc.)
     * @param data  - arbitrary event data
     */
    feed(file, operation, data) {
        const key = this.makeKey(file, operation);
        const now = Date.now();
        const existing = this.events.get(key) ?? {
            file,
            operation,
            events: []
        };
        existing.events.push({ ts: new Date(now), data });
        // Prune events outside the sliding window
        const cutoff = now - this.windowMs;
        existing.events = existing.events.filter(e => e.ts.getTime() > cutoff);
        this.events.set(key, existing);
    }
    /**
     * Returns all current signal classifications from the window.
     */
    classifyAll() {
        const results = [];
        for (const [key, window] of this.events) {
            const count = window.events.length;
            const signal = this.classifyWindow(key, window);
            if (signal !== null) {
                results.push({ key, signal, count });
            }
        }
        return results;
    }
    /**
     * Returns the dominant signal type if one exists, null otherwise.
     */
    classify() {
        const all = this.classifyAll();
        if (all.length === 0)
            return null;
        // Return the signal with highest event count
        all.sort((a, b) => b.count - a.count);
        return all[0].signal;
    }
    /**
     * Returns events for a specific file:operation key.
     */
    getEvents(file, operation) {
        const key = this.makeKey(file, operation);
        return this.events.get(key)?.events ?? [];
    }
    /**
     * Clears all profiled events.
     */
    reset() {
        this.events.clear();
    }
    makeKey(file, operation) {
        return `${file}:${operation}`;
    }
    classifyWindow(key, window) {
        const count = window.events.length;
        if (count < 3)
            return null;
        const [, op] = key.split(":");
        if (op === "error") {
            if (count >= 5)
                return SignalType.ERROR_LOOP;
            return SignalType.RETRY_LOOP;
        }
        if (op === "write") {
            if (count >= 4)
                return SignalType.EDIT_BURST;
            return SignalType.STUCK;
        }
        if (op === "read" && count >= 3) {
            return SignalType.CONTEXT_PRESS;
        }
        if (count >= 5)
            return SignalType.STUCK;
        return null;
    }
}
//# sourceMappingURL=profiler.js.map