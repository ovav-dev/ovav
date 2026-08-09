

<!-- OVAV_CURRENT_AUTHORITY_START -->
## Current Authority — Final Launch Verification

- Active baseline: B23 Tool Readiness Matrix + Advanced Capability Boundary.
- Current phase: Final Launch Verification / OpenCode smoke testing.
- Latest closed stack: launch pack, runtime enforcement, OpenCode runtime wiring, squad normalization, visual delivery/context economy, and tool readiness boundary.
- User-facing readiness wording: launch verification / launch candidate smoke in progress until final smoke evidence and final tag are complete.
- Do not present old segment labels as current authority.
- Do not use legacy preview, legacy closure, legacy caution-state, retired deployment-claim, retired closure-gate, or retired release-candidate wording as the current product state.
- Historical segment references are archived evidence only, not the answer to current launch status.
- If asked whether OVAV is ready, answer from this phase: validators passed, OpenCode smoke is being verified, and production/global-ready claims remain blocked until final launch verification is closed.
<!-- OVAV_CURRENT_AUTHORITY_END -->

# ovav-validate

Validate OVAV from inside OpenCode and explain the result in human terms.

## Current Purpose

Use this before closing work, after source-local runtime changes, after OpenCode surface changes, after registry/policy edits, or when historical evidence needs proof.

Validation follows current task risk and current runtime enforcement, not an old segment label.

## Validation Routing

| Task class | Validation |
|---|---|
| Small documentation or command-surface edit | Compact diff review + relevant validator |
| OpenCode active surface | Command/skill/identity checks + stale-current-state scan |
| Registry or runtime change | `python3 tools/validators/validate_all.py` + related harness |
| Permission/config drift | `python3 tools/validators/check_permission_policy_drift.py` auto-reconciles repo-local projections from `.ovav/policy/permission_authority.json` |
| Service governance | `python3 tools/validators/check_service_area_governance.py` |
| Agent runtime enforcement | `python3 tools/validators/check_agent_runtime_service_area_router.py` + `python3 tools/validators/check_agent_runtime_enforcement.py` |
| OpenCode runtime wiring | `python3 tools/validators/check_opencode_runtime_wiring.py` |
| Historical canonical review | Current-authority review over archived evidence |
| Closure | Strict validation + artifact drift + snapshot/evidence review + git safety |
| Launch readiness | Launch file map + install/rollback/security/privacy/support/compatibility checks |

## Required Runtime Set

For OpenCode runtime wiring and closure-adjacent work, run:

- `python3 tools/validators/check_service_area_governance.py`
- `python3 tools/validators/check_build18_launch_pack.py` — launch pack validator
- `python3 tools/validators/check_agent_runtime_service_area_router.py`
- `python3 tools/validators/check_agent_runtime_enforcement.py`
- `python3 tools/validators/check_opencode_runtime_wiring.py`
- `python3 tools/validators/check_permission_policy_drift.py`
- `python3 tools/validators/validate_all.py`

## Guardrails

- Never claim production/global-ready until launch readiness gates pass.
- Restore unintended historical evidence drift.
- Never `git add .`.
- Treat stale prompt-only governance as invalid active behavior when runtime primitives exist.
