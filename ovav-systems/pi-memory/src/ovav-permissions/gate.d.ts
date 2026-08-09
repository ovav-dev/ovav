/**
 * OVAV MEMORY v3 — Permission Gates
 *
 * Integrates with pi.dev's project_trust hook to enforce OVAV-native
 * permission model. Based on the MiMoCode security plugin pattern
 * but adapted for OVAV's allowlist approach.
 */
export interface PermissionDecision {
    allowed: boolean;
    reason: string;
    gate: "ovav";
}
export declare class PermissionGate {
    private worktree;
    constructor(worktree: string);
    /**
     * Evaluate a bash/shell command against OVAV security policy.
     * Returns PermissionDecision with allowed=false if blocked.
     */
    evaluateCommand(command: string): PermissionDecision;
    /**
     * Evaluate external directory access against OVAV allowlist.
     */
    evaluateExternalDirectory(requestedPath: string): PermissionDecision;
    /**
     * Evaluate environment variable access.
     */
    evaluateEnvAccess(varName: string): PermissionDecision;
}
