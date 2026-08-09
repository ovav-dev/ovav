/**
 * OVAV Memory HTTP Client
 *
 * Calls cPanel's memory MCP relay at http://localhost:5858
 * Uses Node.js 20+ native fetch (no new dependencies)
 * 10s timeout on calls
 */
export class OVAVMemoryClient {
    baseUrl;
    constructor(baseUrl = process.env["OVAV_CPANEL_URL"] ?? "http://localhost:5858") {
        this.baseUrl = baseUrl;
    }
    async callTool(toolName, args = {}) {
        const resp = await fetch(`${this.baseUrl}/api/v1/memory/mcp/relay`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ jsonrpc: "2.0", id: Date.now(), method: "tools/call", params: { name: toolName, arguments: args } }),
            signal: AbortSignal.timeout(10_000),
        });
        if (!resp.ok)
            return `❌ cPanel error ${resp.status}`;
        const data = (await resp.json());
        if (data.error)
            return `❌ MCP error: ${data.error.message}`;
        return data.result?.content?.[0]?.text ?? "No content";
    }
}
//# sourceMappingURL=http_client.js.map