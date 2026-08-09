/**
 * OVAV Memory HTTP Client
 *
 * Calls cPanel's memory MCP relay at http://localhost:5858
 * Uses Node.js 20+ native fetch (no new dependencies)
 * 10s timeout on calls
 */

export class OVAVMemoryClient {
  constructor(private baseUrl = process.env["OVAV_CPANEL_URL"] ?? "http://localhost:5858") {}

  async callTool(toolName: string, args: Record<string, unknown> = {}): Promise<string> {
    const resp = await fetch(`${this.baseUrl}/api/v1/memory/mcp/relay`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: Date.now(), method: "tools/call", params: { name: toolName, arguments: args } }),
      signal: AbortSignal.timeout(10_000),
    });
    if (!resp.ok) return `❌ cPanel error ${resp.status}`;
    const data = (await resp.json()) as { result?: { content?: Array<{ text?: string }> }; error?: { message?: string } };
    if (data.error) return `❌ MCP error: ${data.error.message}`;
    return data.result?.content?.[0]?.text ?? "No content";
  }
}