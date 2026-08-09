/**
 * OVAV MEMORY v3 — Detector
 * Analyzes tool call patterns and triggers Cell injection when struggle signals fire.
 */

import { LiveProfiler, type SignalType } from "./profiler.js";
import { CellStore } from "./cellstore.js";
import { buildEventSignature, type Cell } from "./cell.js";

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
export class Detector {
  private readonly profiler: LiveProfiler;
  private readonly cellStore: CellStore;
  private readonly minWeight: number;

  constructor(cellStore: CellStore, minWeight = 0.6) {
    this.profiler = new LiveProfiler();
    this.cellStore = cellStore;
    this.minWeight = minWeight;
  }

  /**
   * Called on every tool_call event from pi.dev runtime.
   */
  onToolCall(event: ToolCallEvent): void {
    const file = event.file ?? "unknown";

    if (event.error) {
      this.profiler.feed(file, "error", { tool: event.tool, error: event.error });
    } else if (event.tool === "Write" || event.tool === "Edit") {
      // Check for empty writes
      const result = event.result as Record<string, unknown> | undefined;
      if (result?.empty || result?.created === false) {
        this.profiler.feed(file, "write", { empty: true });
      } else {
        this.profiler.feed(file, "write", {});
      }
    } else if (event.tool === "Read" || event.tool === "Grep") {
      const result = event.result as Record<string, unknown> | undefined;
      if (!result || result.content === "" || result.count === 0) {
        this.profiler.feed(file, "read", { empty: true });
      }
    } else {
      this.profiler.feed(file, "call", { tool: event.tool });
    }
  }

  /**
   * Decide whether to inject a Cell given a tool call result.
   * Returns the Cell and signal if injection is warranted.
   */
  decide(event: ToolCallEvent): InjectionDecision {
    // Feed the event first
    this.onToolCall(event);

    const signal = this.profiler.classify();
    if (!signal) {
      return { inject: null, signal: null, sig: null, reason: "no_signal" };
    }

    const file = event.file ?? "unknown";
    const detail = event.error ?? "noop";
    const sig = buildEventSignature(file, signal, detail);

    const cell = this.cellStore.lookupBest(sig, this.minWeight);

    if (!cell) {
      return { inject: null, signal, sig, reason: "no_matching_cell" };
    }

    return {
      inject: cell,
      signal,
      sig,
      reason: `matched_weight_${cell.weight.toFixed(2)}`
    };
  }

  /**
   * Returns current profiler state for debugging.
   */
  snapshot(): Array<{ key: string; signal: SignalType; count: number }> {
    return this.profiler.classifyAll();
  }

  /**
   * Resets the profiler window.
   */
  reset(): void {
    this.profiler.reset();
  }
}
