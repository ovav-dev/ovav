/**
 * OVAV MEMORY v3 — LiveProfiler
 * Sliding-window event profiler for detecting struggle signals.
 */

export enum SignalType {
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

interface EventWindow {
  file: string;
  operation: string;
  events: ProfiledEvent[];
}

/**
 * LiveProfiler maintains a 5-minute sliding window of tool events
 * and classifies patterns into SignalType signals.
 */
export class LiveProfiler {
  private readonly windowMs: number;
  private events: Map<string, EventWindow> = new Map();

  constructor(windowMinutes = 5) {
    this.windowMs = windowMinutes * 60 * 1000;
  }

  /**
   * Feed a new event into the profiler.
   * @param file  - identifier of the file/tool context
   * @param operation - type of operation (error, write, read, etc.)
   * @param data  - arbitrary event data
   */
  feed(file: string, operation: string, data: unknown): void {
    const key = this.makeKey(file, operation);
    const now = Date.now();

    const existing: EventWindow = this.events.get(key) ?? {
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
  classifyAll(): Array<{ key: string; signal: SignalType; count: number }> {
    const results: Array<{ key: string; signal: SignalType; count: number }> = [];

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
  classify(): SignalType | null {
    const all = this.classifyAll();
    if (all.length === 0) return null;

    // Return the signal with highest event count
    all.sort((a, b) => b.count - a.count);
    return all[0].signal;
  }

  /**
   * Returns events for a specific file:operation key.
   */
  getEvents(file: string, operation: string): ProfiledEvent[] {
    const key = this.makeKey(file, operation);
    return this.events.get(key)?.events ?? [];
  }

  /**
   * Clears all profiled events.
   */
  reset(): void {
    this.events.clear();
  }

  private makeKey(file: string, operation: string): string {
    return `${file}:${operation}`;
  }

  private classifyWindow(key: string, window: EventWindow): SignalType | null {
    const count = window.events.length;

    if (count < 3) return null;

    const [, op] = key.split(":");

    if (op === "error") {
      if (count >= 5) return SignalType.ERROR_LOOP;
      return SignalType.RETRY_LOOP;
    }

    if (op === "write") {
      if (count >= 4) return SignalType.EDIT_BURST;
      return SignalType.STUCK;
    }

    if (op === "read" && count >= 3) {
      return SignalType.CONTEXT_PRESS;
    }

    if (count >= 5) return SignalType.STUCK;

    return null;
  }
}
