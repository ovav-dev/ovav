# P5 v2 — Execution Profiles (using real schema)

## Schema inferred from existing `default` profile

Valid keys (observed in settings.toml):
- apply_code_diffs: "always_ask" (one value seen)
- ask_user_question: "always_ask" (one value seen)
- execute_commands: "agent_decides" (one value seen)
- read_files: "always_allow" (one value seen)
- run_agents: "always_ask" (one value seen)
- write_to_pty: "always_ask" (one value seen)
- mcp_permissions: "agent_decides" (one value seen)
- computer_use: "never" (only safe value)
- command_allowlist / denylist: arrays
- mcp_allowlist / denylist: arrays
- directory_allowlist: arrays
- web_search_enabled: bool
- name: string
- context_window_limit: int (1000000)
- base_model: string ("auto-genius" observed, may accept custom)

## Plan

Create 4 profiles that use ONLY observed value patterns:
- Reuse: computer_use, run_agents, mcp, allowlists — same as default
- Vary: apply_code_diffs, ask_user_question, execute_commands, write_to_pty
- DO NOT add new keys (sudo, secrets — not in schema → reject)

## Profile design

### 1. ovav_build (default daily)
- apply_code_diffs: "always_allow" (was "always_ask")
- ask_user_question: "always_ask" (same)
- execute_commands: "agent_decides" (same)
- read_files: "always_allow" (same)
- run_agents: "always_ask" (same)
- write_to_pty: "always_ask" (same)
- mcp_permissions: "agent_decides" (same)
- computer_use: "never" (same)
- base_model: "MiniMax-M3" (was "auto-genius" — risky but plan §23 requires)

### 2. ovav_yolo (aggressive)
- apply_code_diffs: "always_allow"
- ask_user_question: "always_allow" (relaxed from ask)
- execute_commands: "always_allow" (relaxed from agent_decides)
- read_files: "always_allow"
- run_agents: "always_allow" (relaxed)
- write_to_pty: "always_allow"
- mcp_permissions: "always_allow"
- computer_use: "never"
- base_model: "MiniMax-M3"

### 3. ovav_review (read-only)
- apply_code_diffs: "always_ask" (default — ask before writes)
- ask_user_question: "always_ask"
- execute_commands: "always_ask" (asks before mutations)
- read_files: "always_allow"
- run_agents: "always_ask"
- write_to_pty: "always_ask"
- mcp_permissions: "agent_decides"
- computer_use: "never"
- base_model: "MiniMax-M3"

### 4. thavren_systems (infra)
- apply_code_diffs: "always_ask"
- ask_user_question: "always_ask"
- execute_commands: "always_ask" (any mutation asks)
- read_files: "always_allow"
- run_agents: "always_ask"
- write_to_pty: "always_ask"
- mcp_permissions: "agent_decides"
- computer_use: "never"
- base_model: "MiniMax-M3"

## Denylist rebuild

Add NEW patterns to existing broad shell blocks:
- Keep broad: bash, fish, pwsh, sh, zsh (existing)
- Add: 'sudo (.*)?', 'git worktree (.*)?', 'rm -rf (.*)?', etc.

## CRIT-009 discipline

If Warp rejects ANY value, revert from backup immediately.
`atomicWriteFile` ensures clean rollback.

## Files

- .ovav/plans/p5-execution-profiles-v2.ps1 (PowerShell script)
- Updated settings.toml (via script)
