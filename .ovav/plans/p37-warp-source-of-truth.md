# P37 — Warp Drive NOT source of truth

## Policy

Per plan §37:
- Workflows in Warp Drive are convenience
- Critical logic lives in:
  - Git
  - AGENTS.md
  - .agents/skills/ (or .ovav/source/skills/ canonical)
  - OWS
  - mise.toml
- Warp Drive = interface + sync, NEVER sole source

## Source of truth map

| Concept | Canonical location | Warp Drive role |
|---|---|---|
| Workflow manifest | `.ovav/warp/workflows.json` | Mirror only |
| Skill content | `.ovav/source/skills/*/SKILL.md` | Mirror only |
| Governance rules | `AGENTS.md` (root) | Reference only |
| Agent permissions | `.ovav/policy/permission_authority.json` | Reference only |
| Worktree lifecycle | OWS (`go-runtime/internal/ows/`) | Calls OWS |
| Toolchain pins | `mise.toml` + `mise.lock` | Calls mise |
| Identities | `.ovav/registry/identities.yaml` | Mirror only |
| Audit log | `.ovav/runtime/audit.jsonl` | Mirror only |

## Implication

If Warp Drive is lost, no critical logic is lost:
- Workflows can be re-imported from `.ovav/warp/workflows.json`
- Skills can be re-synced from `.ovav/source/skills/`
- Identity registry from `.ovav/registry/`

If Git is lost, OVAV is broken. Git is THE source of truth.

## Status

✅ P37 100% — source of truth map documented.
