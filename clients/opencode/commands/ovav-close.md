

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

# ovav-close

Prepare a safe close from inside OpenCode.

## Current Close Rules

- Confirm current branch.
- Confirm git status.
- Confirm task scope and selected delivery contract.
- Confirm Service Area Router output and Session Capsule boundary for the work being closed.
- Run current runtime enforcement validation before closure: `python3 tools/validators/check_agent_runtime_service_area_router.py` and `python3 tools/validators/check_agent_runtime_enforcement.py`.
- For OpenCode runtime wiring, also run `python3 tools/validators/check_opencode_runtime_wiring.py`.
- Run all other relevant validators.
- Check artifact drift.
- Preserve historical evidence unless current work explicitly owns it.
- Stage exact files only.
- Never `git add .`.
- Commit only with explicit approval.
- Remote/push only if a repo remote exists and the user explicitly approves it.
- Produce or reference a trace-ready closure payload: service area, lead, mode, decision, source/tool/handoff decisions, validation, delivery contract and Safe Stop state if interrupted.

Closure is blocked if current runtime enforcement validation fails or if unresolved diffs, raw snapshots, install artifacts or unsanitized cross-area handoff are present outside the approved scope.
