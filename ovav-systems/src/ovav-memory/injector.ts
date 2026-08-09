/**
 * OVAV MEMORY v3 — HarnessInjector
 * Interface for injecting Cells into the pi.dev agent runtime.
 */

import { type Cell } from "./cell.js";
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
  afterToolCall(
    ctx: InjectionContext,
    tool: string,
    file: string | undefined,
    result: unknown,
    error: string | undefined
  ): Promise<InjectionResult | null>;

  /**
   * Record whether the last injection was helpful.
   */
  recordOutcome(ctx: InjectionContext, cellId: string, helped: boolean): Promise<void>;
}

/**
 * Default HarnessInjector using CellStore and Detector.
 */
export class HarnessInjector implements IHarnessInjector {
  constructor(
    private readonly cellStore: CellStore,
    private readonly getDetector: () => { decide: (e: import("./detector.js").ToolCallEvent) => import("./detector.js").InjectionDecision }
  ) {}

  async beforeAgentStart(_ctx: InjectionContext): Promise<string> {
    // Context pre-load — return nothing by default in v3
    // Future: load high-weight general-purpose cells
    return "";
  }

  async afterToolCall(
    ctx: InjectionContext,
    tool: string,
    file: string | undefined,
    result: unknown,
    error: string | undefined
  ): Promise<InjectionResult | null> {
    const decision = this.getDetector().decide({ tool, file, result, error });

    if (!decision.inject) {
      return null;
    }

    const cell = decision.inject;

    // Build injection content
    const content = this.formatCell(cell);
    const tokens = this.estimateTokens(content);

    return {
      injected: true,
      cellId: cell.id,
      content,
      tokens
    };
  }

  async recordOutcome(ctx: InjectionContext, cellId: string, helped: boolean): Promise<void> {
    await this.cellStore.recordOutcome(cellId, helped);
  }

  private formatCell(cell: Cell): string {
    return [
      `<!-- OVAV_MEMORY_INJECT: ${cell.id} -->`,
      `**${cell.eventSignature}** (weight: ${cell.weight.toFixed(2)})`,
      "",
      cell.summary,
      ""
    ].join("\n");
  }

  private estimateTokens(text: string): number {
    // Rough approximation: 1 token ≈ 4 chars for English
    return Math.ceil(text.length / 4);
  }
}
