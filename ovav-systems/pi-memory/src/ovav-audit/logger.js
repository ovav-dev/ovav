/**
 * OVAV MEMORY v3 — Audit Logger
 *
 * Event stream audit trail for OVAV MEMORY v3.
 * Mirrors the MiMoCode audit pattern using event stream from pi.dev.
 *
 * Log file: .ovav/runtime/logs/ovav-audit.jsonl
 */
import * as fs from "node:fs/promises";
import * as path from "node:path";
const AUDIT_LOG_DIR = ".ovav/runtime/logs";
const AUDIT_LOG_FILE = "ovav-audit.jsonl";
export class AuditLogger {
    basePath;
    logPath;
    fd = null;
    constructor(basePath) {
        this.basePath = basePath;
        this.logPath = path.join(basePath, AUDIT_LOG_DIR, AUDIT_LOG_FILE);
    }
    /**
     * Initialize the audit log directory and file.
     */
    async init() {
        const dir = path.join(this.basePath, AUDIT_LOG_DIR);
        await fs.mkdir(dir, { recursive: true });
        // Create log file if it doesn't exist
        try {
            await fs.access(this.logPath);
        }
        catch {
            await fs.writeFile(this.logPath, "", "utf-8");
        }
    }
    /**
     * Log an audit event.
     * Non-fatal — failures are silently swallowed.
     */
    async log(event) {
        try {
            const entry = {
                timestamp: new Date().toISOString(),
                projectPath: this.basePath,
                ...event,
            };
            const line = JSON.stringify(entry) + "\n";
            await fs.appendFile(this.logPath, line, "utf-8");
        }
        catch {
            // Audit log failure is non-fatal — never crash the runtime
        }
    }
    /**
     * Log a cell injection event.
     */
    async logCellInject(cellId, eventSignature, weight, helped) {
        await this.log({
            event: helped ? "cell_helped" : "cell_not_helped",
            cellId,
            data: { eventSignature, weight },
        });
    }
    /**
     * Log a validator run event.
     */
    async logValidatorRun(validator, valid, errors) {
        await this.log({
            event: valid ? "validator_pass" : "validator_fail",
            data: { validator, errors },
        });
    }
    /**
     * Log a permission blocked event.
     */
    async logPermissionBlocked(command, reason) {
        await this.log({
            event: "permission_blocked",
            command: command.substring(0, 200),
            reason,
        });
    }
    /**
     * Read recent audit events for a session.
     */
    async getRecentEvents(sessionId, limit = 50) {
        try {
            const content = await fs.readFile(this.logPath, "utf-8");
            const lines = content.split("\n").filter(Boolean);
            const events = [];
            for (const line of lines.slice(-limit)) {
                try {
                    const event = JSON.parse(line);
                    if (!sessionId || event.sessionId === sessionId) {
                        events.push(event);
                    }
                }
                catch {
                    // Skip malformed lines
                }
            }
            return events;
        }
        catch {
            return [];
        }
    }
    /**
     * Get audit stats for a project.
     */
    async getStats() {
        try {
            const content = await fs.readFile(this.logPath, "utf-8");
            const lines = content.split("\n").filter(Boolean);
            const byType = {};
            let total = 0;
            for (const line of lines) {
                try {
                    const event = JSON.parse(line);
                    total++;
                    byType[event.event] = (byType[event.event] ?? 0) + 1;
                }
                catch {
                    // Skip
                }
            }
            return { total, byType };
        }
        catch {
            return { total: 0, byType: {} };
        }
    }
}
//# sourceMappingURL=logger.js.map