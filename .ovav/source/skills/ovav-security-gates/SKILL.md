---
name: ovav-security-gates
description: Use when OVAV security enforcement, command blocking, actor audit, or session validation is needed.
---

# OVAV Security Gates

Security hooks provided by the `ovav-security` plugin. Enforces OVAV permission authority via MiMo Code hooks.

## Active Hooks

| Hook | Layer | What it does |
|------|-------|-------------|
| `tool.execute.before` | Security Gate | Blocks dangerous bash: git push, sudo, force ops, package installs |
| `actor.preStop` | Actor Audit | Logs subagent lifecycle events |
| `actor.postStop` | Actor Outcome | Logs outcome, retries write-capable actors on failure |
| `session.pre` | Session Guard | Validates governance files exist, logs session start |
| `permission.ask` | Permission Override | Enforces deny-by-default for external directories |

## Blocked Commands (static blocklist)

- `git push*` → use governed push gate
- `git push --force*` / `git push -f*` →永远 blocked
- `git branch -D*` / `git branch -d*` → branch delete blocked
- `sudo*` → privilege escalation blocked
- `pip install*`, `npm install*`, `apt install*` → use governed install pipeline
- `gh auth token*`, `gh auth login*` → auth exposure blocked
- `python3 tools/install/*` → use OVAV install pipeline

## Blocked Commands (dynamic from permission_authority.json)

Patterns from `protected_denies.bash` are loaded at plugin init and enforced dynamically.

## Audit Trail

All blocked commands and security events are logged to:
`.ovav/runtime/logs/security_hooks.jsonl`

Format: JSONL with timestamp, event type, command, reason, session ID.

## Interaction with Workflow Engine

The MiMo Code `workflow` tool is available for multi-step orchestration.
OVAV security hooks apply to ALL tools executed within workflows — no bypass.

## Blocked Surfaces

No bypass flags, no force operations, no global config writes, no production claims.
