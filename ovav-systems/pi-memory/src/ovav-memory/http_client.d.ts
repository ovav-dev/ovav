/**
 * OVAV Memory HTTP Client
 *
 * Calls cPanel's memory MCP relay at http://localhost:5858
 * Uses Node.js 20+ native fetch (no new dependencies)
 * 10s timeout on calls
 */
export declare class OVAVMemoryClient {
    private baseUrl;
    constructor(baseUrl?: string);
    callTool(toolName: string, args?: Record<string, unknown>): Promise<string>;
}
//# sourceMappingURL=http_client.d.ts.map