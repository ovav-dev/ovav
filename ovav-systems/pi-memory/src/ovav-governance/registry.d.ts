/**
 * OVAV MEMORY v3 — Governance Tool Registry
 *
 * Exposes F0-F5 validators as pi.dev tools with the same
 * naming pattern as the existing MiMoCode plugin:
 *   - ovav_validate      → runs F0-F5 validators
 *   - ovav_daily        → state summary
 *   - ovav_next_work    → plan resolution
 *   - ovav_check_integrity → system integrity score
 */
export interface ToolResult {
    source: string;
    output: unknown;
    error?: string;
}
export declare function runOvavValidate(basePath: string, scope?: "all" | "f0" | "f1" | "f2" | "f3" | "f4" | "f5"): Promise<ToolResult>;
export declare function runOvavDaily(basePath: string): Promise<ToolResult>;
export declare function runOvavNextWork(basePath: string): Promise<ToolResult>;
export declare function runOvavCheckIntegrity(basePath: string): Promise<ToolResult>;
