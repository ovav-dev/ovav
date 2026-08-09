

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

# ovav-status

Show where OVAV is now and what is safe to do next.

## What It Does

Use this when you need a compact status card inside OpenCode: current baseline, latest completed segment, next work, visible role routing and blocked surfaces.

## OpenCode-First Prompt

Summarize OVAV status in plain language first. Show the current layer, next segment, useful evidence pointers and preserved blocked surfaces. Keep terminal details secondary.

## Internal Gates

These backend checks support the OpenCode status card; they are not the primary user experience.

- `go run -C go-runtime ./cmd/ovav status` (daily summary, Go-native)
- `go run -C go-runtime ./cmd/ovav doctor` (system diagnostic, Go-native)

## Output Shape

| Card | Content |
|---|---|
| Current | Latest completed layer and baseline |
| Next | Next segment and visible role |
| Evidence | Most relevant artifact paths |
| Safety | Blocked surfaces preserved |

## Guardrails

- Source-local status only.
- No global OpenCode config.
- No plugin installation.
- No live Engram behavior.
- No real install/apply/backup/rollback.
- No production-ready or global-ready claims.
