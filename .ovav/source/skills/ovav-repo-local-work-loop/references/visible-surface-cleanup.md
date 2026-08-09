# Visible Surface Cleanup: Native BUILD/PLAN Containment

## Principle

OpenCode native BUILD and PLAN must not become OVAV's visible professional model. Do not delete or aggressively retire host-native modes when that could break OpenCode. Contain them: keep them hidden/neutralized where the host allows it, and route active OVAV work through Platform Engineering and Research Intelligence.

## Audit Scope

Active visible references in:
- `.opencode/agents` — agent files and their descriptions
- `.opencode/commands` — command prompts and internal gates
- `.opencode/skills` — skill descriptions and content

## Cleanup Rules

1. Replace active BUILD/PLAN routing with professional service area routing.
2. Treat native OpenCode BUILD/PLAN as host runtime compatibility surfaces: hidden/neutralized, not product-facing OVAV areas.
3. Preserve historical/archival references in:
   - `.ovav/artifacts/` (segment evidence)
   - `.ovav/registry/work_ledger.yaml` (ledger events)
   - `.ovav/context/BUILD*.md` (historical handoffs)
   - `docs/` (historical documentation)
   - `CHECKLIST_0_TO_100.md` (segment history)
4. Never delete historical BUILD segment evidence.
5. Never mutate historical evidence.

## Replacement Pattern

| Before | After |
|---|---|
| "Current BUILD/layer" | "Current professional service layer" |
| historical segment wording | current BUILD 17/18 state wording |
| "BUILD 4 baseline" | "controlled_preview_surface_graduation_rc baseline" |

## Checker Coverage

`check_visible_surface_cleanup.py` verifies:
- No active BUILD/PLAN-only routing in OpenCode agents/commands/skills.
- Native BUILD/PLAN, if present, are compatibility-hidden rather than deleted.
- Historical references remain present in archival paths.
- Professional service area routing is used instead.
