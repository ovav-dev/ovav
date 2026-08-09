/**
 * OVAV MEMORY v3 — pi Coding Agent Extension
 *
 * Installs OVAV MEMORY as a pi extension:
 *   - before_agent_start: injects Cell memory before each turn
 *   - tool_call: feeds error/retry patterns to LiveProfiler
 *   - tool_result: tracks success/failure for weight updates
 *   - turn_end: triggers signal classification
 *   - session_start: loads CellStore + AuditLogger
 *   - session_shutdown: persists CellStore + flushes audit
 *
 * Install: pi install git:github.com/ovav-systems/ovav
 * Or:     copy to ~/.pi/agent/extensions/ovav-memory.ts
 */
import { OVAVMemory } from "../OVAVMemory.js";
export default function (pi) {
    // Initialize OVAV MEMORY — loads CellStore, AuditLogger, Detector
    const ovav = new OVAVMemory({
        projectPath: process.cwd(),
        harnessScope: "pi",
    });
    // ── Session lifecycle ──────────────────────────────────────
    pi.on("session_start", async (event, ctx) => {
        await ovav.onSessionStart({
            sessionId: ctx.sessionManager.getSessionId() ?? "unknown",
            reason: event.reason ?? "startup",
        });
        ctx.ui.notify(`OVAV MEMORY v3 active — ${ovav.cellCount()} cells loaded`, "info");
    });
    pi.on("session_shutdown", async (event) => {
        await ovav.onSessionShutdown({
            reason: event.reason ?? "quit",
        });
    });
    // ── Agent lifecycle ───────────────────────────────────────
    pi.on("before_agent_start", async (event, ctx) => {
        const injection = await ovav.onBeforeAgentStart({
            prompt: event.prompt ?? "",
            systemPrompt: event.systemPrompt ?? "",
        });
        if (!injection)
            return undefined;
        return {
            message: {
                customType: "ovav-memory-inject",
                content: injection.content,
                display: false, // hidden from user, goes only to LLM
            },
            systemPrompt: event.systemPrompt,
        };
    });
    // ── Tool lifecycle ─────────────────────────────────────────
    pi.on("tool_call", async (event, ctx) => {
        const result = await ovav.onToolCall({
            toolName: event.toolName,
            input: event.input,
        });
        if (result?.blocked) {
            return {
                block: true,
                reason: result.reason ?? "Blocked by OVAV PermissionGate",
            };
        }
        return undefined;
    });
    pi.on("tool_result", async (event) => {
        await ovav.onToolResult({
            toolName: event.toolName,
            input: event.input,
            result: event.content,
            isError: event.isError ?? false,
        });
    });
    pi.on("turn_end", async (event, ctx) => {
        await ovav.onTurnEnd({
            turnIndex: event.turnIndex,
            message: event.message,
        });
        // Update status widget
        const stats = ovav.getStats();
        if (stats.activeSignals > 0) {
            ctx.ui.setStatus("ovav", `OVAV: ${stats.activeSignals} signal(s) · ${stats.cellsInjected} injected`);
        }
        else {
            ctx.ui.setStatus("ovav", "OVAV: idle");
        }
    });
    // ── Governance tools ─────────────────────────────────────────
    pi.registerTool({
        name: "ovav_validate",
        label: "OVAV Validate",
        description: "Run OVAV F0-F5 governance validators on the current project. " +
            "Returns: architecture, infrastructure, roles, security, integration scores.",
        parameters: {
            type: "object",
            properties: {
                scope: {
                    type: "string",
                    enum: ["all", "f0", "f1", "f2", "f3", "f4", "f5"],
                    description: "Validation scope: all (full F0-F5), or specific gate (f0-f5)",
                },
            },
        },
        async execute(toolCallId, params) {
            const raw = params?.scope;
            const scope = (typeof raw === "string" && ["all", "f0", "f1", "f2", "f3", "f4", "f5"].includes(raw))
                ? raw
                : "all";
            const result = await ovav.runValidate(scope);
            return {
                content: [
                    {
                        type: "text",
                        text: JSON.stringify(result, null, 2),
                    },
                ],
                details: { scope: params.scope ?? "all" },
            };
        },
    });
    pi.registerTool({
        name: "ovav_next_work",
        label: "OVAV Next Work",
        description: "Returns the next recommended work item from the OVAV decision ledger, " +
            "based on priority, recent decisions, and open tasks.",
        parameters: { type: "object", properties: {} },
        async execute(toolCallId) {
            const result = await ovav.runNextWork();
            return {
                content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
                details: {},
            };
        },
    });
    pi.registerTool({
        name: "ovav_daily",
        label: "OVAV Daily Brief",
        description: "Returns a daily governance brief: validator health, open decisions, " +
            "recent memory injections, and system status.",
        parameters: { type: "object", properties: {} },
        async execute(toolCallId) {
            const result = await ovav.runDailyBrief();
            return {
                content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
                details: {},
            };
        },
    });
    pi.registerTool({
        name: "ovav_check_integrity",
        label: "OVAV Integrity Check",
        description: "Checks OVAV system integrity: workspace safety, secrets hygiene, " +
            "permission policy drift, and contract freshness.",
        parameters: { type: "object", properties: {} },
        async execute(toolCallId) {
            const result = await ovav.runIntegrityCheck();
            return {
                content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
                details: {},
            };
        },
    });
    // ── Memory tools ──────────────────────────────────────────────
    pi.registerTool({
        name: "memory_recall",
        label: "Memory Recall",
        description: "Queries the OVAV memory system for relevant context using a natural language query.",
        parameters: {
            type: "object",
            properties: {
                query: { type: "string", description: "Natural language query to search memory" },
            },
            required: ["query"],
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_recall", { query: params.query });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_store",
        label: "Memory Store",
        description: "Stores a memory card in the OVAV memory system with tags, summary, and payload.",
        parameters: {
            type: "object",
            properties: {
                id: { type: "string", description: "Unique memory card ID" },
                summary: { type: "string", description: "Brief summary of the memory" },
                tags: { type: "array", items: { type: "string" }, description: "Tags for categorization" },
                payload: { type: "string", description: "Full memory content" },
            },
            required: ["id", "summary", "tags", "payload"],
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_store", {
                id: params.id,
                summary: params.summary,
                tags: params.tags,
                payload: params.payload,
            });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_stats",
        label: "Memory Stats",
        description: "Returns OVAV memory statistics: total cards, active signals, session injections.",
        parameters: { type: "object", properties: {} },
        async execute(toolCallId) {
            const result = await ovav.callMemoryTool("memory_stats", {});
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_recent",
        label: "Memory Recent",
        description: "Returns the most recently accessed or injected memory cards.",
        parameters: {
            type: "object",
            properties: {
                limit: { type: "number", description: "Maximum number of cards to return (default 10)" },
            },
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_recent", { limit: params.limit ?? 10 });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_verify",
        label: "Memory Verify",
        description: "Verifies the authenticity and integrity of a memory card by its ID.",
        parameters: {
            type: "object",
            properties: {
                cardId: { type: "string", description: "Memory card ID to verify" },
            },
            required: ["cardId"],
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_verify", { cardId: params.cardId });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_search_decisions",
        label: "Memory Search Decisions",
        description: "Searches governance decision records from OVAV memory.",
        parameters: {
            type: "object",
            properties: {
                query: { type: "string", description: "Search query for governance decisions" },
                scope: { type: "string", description: "Scope filter: f0, f1, f2, f3, f4, f5, all" },
            },
            required: ["query"],
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_search_decisions", {
                query: params.query,
                scope: params.scope ?? "all",
            });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
    pi.registerTool({
        name: "memory_search_errors",
        label: "Memory Search Errors",
        description: "Searches error recovery knowledge and past error patterns from OVAV memory.",
        parameters: {
            type: "object",
            properties: {
                query: { type: "string", description: "Error pattern or keyword to search" },
                limit: { type: "number", description: "Maximum results to return" },
            },
            required: ["query"],
        },
        async execute(toolCallId, params) {
            const result = await ovav.callMemoryTool("memory_search_errors", {
                query: params.query,
                limit: params.limit ?? 20,
            });
            return { content: [{ type: "text", text: result }], details: {} };
        },
    });
}
//# sourceMappingURL=ovav-memory.js.map