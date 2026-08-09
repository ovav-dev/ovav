/**
 * OVAV MEMORY v3 — Audit Logger
 *
 * Event stream audit trail for OVAV MEMORY v3.
 * Mirrors the MiMoCode audit pattern using event stream from pi.dev.
 *
 * Log file: .ovav/runtime/logs/ovav-audit.jsonl
 */
export type AuditEventType = "cell_created" | "cell_injected" | "cell_helped" | "cell_not_helped" | "validator_run" | "validator_pass" | "validator_fail" | "permission_blocked" | "permission_allowed" | "session_start" | "session_end" | "turn_start" | "turn_end";
export interface AuditEvent {
    timestamp?: string;
    event: AuditEventType;
    sessionId?: string;
    projectPath?: string;
    actorId?: string;
    cellId?: string;
    weight?: number;
    tool?: string;
    command?: string;
    reason?: string;
    scope?: string;
    data?: Record<string, unknown>;
}
export declare class AuditLogger {
    private basePath;
    private logPath;
    private fd;
    constructor(basePath: string);
    /**
     * Initialize the audit log directory and file.
     */
    init(): Promise<void>;
    /**
     * Log an audit event.
     * Non-fatal — failures are silently swallowed.
     */
    log(event: AuditEvent): Promise<void>;
    /**
     * Log a cell injection event.
     */
    logCellInject(cellId: string, eventSignature: string, weight: number, helped: boolean): Promise<void>;
    /**
     * Log a validator run event.
     */
    logValidatorRun(validator: string, valid: boolean, errors: number): Promise<void>;
    /**
     * Log a permission blocked event.
     */
    logPermissionBlocked(command: string, reason: string): Promise<void>;
    /**
     * Read recent audit events for a session.
     */
    getRecentEvents(sessionId: string, limit?: number): Promise<AuditEvent[]>;
    /**
     * Get audit stats for a project.
     */
    getStats(): Promise<{
        total: number;
        byType: Record<string, number>;
    }>;
}
