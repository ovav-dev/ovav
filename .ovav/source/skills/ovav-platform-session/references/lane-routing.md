# Lane-Aware Routing

## Routing Labels

| Routing label | Use when |
|---|---|
| `greeting_identity` | The user greets, asks who is speaking, or starts casually. |
| `workstation_stack` | Terminal, shell, CLI, local development stack or workstation behavior is in scope. |
| `opencode_system` | OpenCode agents, commands, skills or repo-local OpenCode configuration are in scope. |
| `runtime_governance` | OVAV runtime, next-work, validation, handoff, evidence or harness orchestration is in scope. |
| `configuration_engineering` | Repo-local config, permissions, policy or safe config repair is in scope. |
| `source_local_automation` | Source-local automation, scripts or deterministic artifact generation are in scope. |
| `safety_gate` | Blocked surfaces, risk, permissions, install, Engram, UI/TUI, MCP/A2A or external services are in scope. |
| `validation_closure` | The user asks to verify, close, summarize, hand off or prepare evidence. |

## Routing Behavior

- Each label maps to a specific harness family.
- `greeting_identity` → minimal context (T1).
- `workstation_stack`, `configuration_engineering` → standard context (T3).
- `opencode_system`, `runtime_governance`, `source_local_automation` → full context (T4).
- `safety_gate`, `validation_closure` → strict closure context (T5).
