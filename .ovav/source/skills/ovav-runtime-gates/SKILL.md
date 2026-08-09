---
name: ovav-runtime-gates
description: Use when OVAV work needs source-local runtime gates, validation, next-work resolution, close-layer dry-runs or blocked-surface checks.
---

# OVAV Runtime Gates

Use native MiMo Code tools for OVAV governance. Tools are provided by the `ovav-governance` plugin.

## Native Tools

- **ovav_validate** — Run F0-F5 validators + integrity score. Use `scope` param for specific validator.
- **ovav_daily** — Daily state summary: git HEAD, branch, plan phase from caps.yaml.
- **ovav_next_work** — Resolve next work item from OVAV plan.
- **ovav_check_integrity** — System integrity score: governance files, permissions, Go runtime, plugins, skills.

## Additional Native Tools (ovav-status plugin)

- **ovav_status** — Full OVAV system status: governor, memory, integrity, branch, capsule, tokens.
- **ovav_health** — Quick health check: OK/DEGRADED/DOWN with issue list.
- **ovav_monitor** — Session monitor: agent, model, tokens, budget, elapsed time.

## Security Hooks (ovav-security plugin)

- **tool.execute.before** — Blocks dangerous bash (git push, sudo, force ops)
- **actor.preStop/postStop** — Audit trail for subagent lifecycle
- **session.pre** — Workspace validation at session start
- **permission.ask** — Deny-by-default for external directories

See `ovav-security-gates` skill for full details.

## Workflow Engine

MiMo Code's built-in `workflow` tool supports multi-step orchestration.
OVAV security hooks apply to all tools within workflows — no bypass possible.

### OVAV Workflows

- **ovav-deep-research** — Multi-step research: plan → search → extract → crosscheck → report
- **ovav-validate-all** — Comprehensive F0-F5 validation with integrity scoring

Usage: `workflow({ name: "ovav-deep-research", args: { question: "..." } })`

## Fallback: Go Runtime

When native tools are unavailable, use Go runtime directly:
- Session greeting: `go run -C go-runtime ./cmd/session_greeting --json`
- Protected branch check: `go run -C go-runtime ./cmd/session_greeting --check-protected`

## Fallback: Deprecated Python (last resort)

⚠️ Most `python3 tools/ovav_runtime.py` commands are deprecated. Git HEAD is the canonical truth source.
Only `memory` and `install-sandbox` remain active. Use git directly for state queries.

## UX Rule

Prefer native tools over terminal commands. Treat tool output as the primary interface, not backend proof.

## Blocked Surfaces

No global config, plugin installation, live Engram behavior, real install/apply/backup/rollback, UI/TUI, MCP/A2A, external service behavior, production-ready claims or global-ready claims.
