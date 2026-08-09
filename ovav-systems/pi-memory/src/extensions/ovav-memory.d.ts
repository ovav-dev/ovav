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
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
export default function (pi: ExtensionAPI): void;
//# sourceMappingURL=ovav-memory.d.ts.map