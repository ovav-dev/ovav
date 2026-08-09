/**
 * OVAV MEMORY v3 — HarnessInjector
 * Interface for injecting Cells into the pi.dev agent runtime.
 */
/**
 * Default HarnessInjector using CellStore and Detector.
 */
export class HarnessInjector {
    cellStore;
    getDetector;
    constructor(cellStore, getDetector) {
        this.cellStore = cellStore;
        this.getDetector = getDetector;
    }
    async beforeAgentStart(_ctx) {
        // Context pre-load — return nothing by default in v3
        // Future: load high-weight general-purpose cells
        return "";
    }
    async afterToolCall(ctx, tool, file, result, error) {
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
    async recordOutcome(ctx, cellId, helped) {
        await this.cellStore.recordOutcome(cellId, helped);
    }
    formatCell(cell) {
        return [
            `<!-- OVAV_MEMORY_INJECT: ${cell.id} -->`,
            `**${cell.eventSignature}** (weight: ${cell.weight.toFixed(2)})`,
            "",
            cell.summary,
            ""
        ].join("\n");
    }
    estimateTokens(text) {
        // Rough approximation: 1 token ≈ 4 chars for English
        return Math.ceil(text.length / 4);
    }
}
//# sourceMappingURL=injector.js.map