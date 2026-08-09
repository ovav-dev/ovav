# Changelog — @ovav/pi-memory

## 3.1.0 (2026-08-06)

### Added
- `OVAVMemoryClient` HTTP client for cPanel memory MCP relay at `http://localhost:5858` (`ovav-memory/http_client.ts`)
- `callMemoryTool(toolName, args)` method on `OVAVMemory` — bridges to cPanel relay
- 7 new memory tools registered on the pi extension:
  - `memory_recall` — queries memory with natural language
  - `memory_store` — stores a memory card
  - `memory_stats` — memory statistics
  - `memory_recent` — recent memory cards
  - `memory_verify` — verify card authenticity
  - `memory_search_decisions` — search governance decisions
  - `memory_search_errors` — search error recovery knowledge

### Changed
- `OVAVMemory` now wires in `OVAVMemoryClient` on construction

## 3.0.0 (2026-07-18)

### Added
- OVAV MEMORY v3 full release: CellStore, LiveProfiler, Detector, HarnessInjector
- F0–F5 governance validators
- PermissionGate for bash/shell commands
- AuditLogger for session events
- pi extension integration via `before_agent_start`, `tool_call`, `tool_result`, `turn_end`, `session_start`, `session_shutdown`
