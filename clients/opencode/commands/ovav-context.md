

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

# ovav-context

Prepare the right OVAV context before doing work.

## Current Purpose

Use this when the session needs orientation: current baseline, service area, evidence, blocked surfaces, launch-readiness status and next safe move.

OVAV context loading is governed by current runtime enforcement. Do not load internal files until the Service Area Router resolves the service area and the Context Gateway allows the source/path.

Use `.ovav/service_areas/shared/context_economy_contract.yaml` for context tiers:

| Tier | Use |
|---|---|
| `T0_none` | acknowledgement or trivial conversational turn |
| `T1_tiny` | greetings, simple consultations, identity clarification |
| `T2_compact` | direct diagnosis, command explanation, small context card |
| `T3_focused` | small implementation, scoped OpenCode update, validator repair |
| `T4_full_scoped` | multi-file implementation, cross-area handoff, high-risk scoped change |
| `T5_closure_grade` | final validation, commit readiness, safety review |

## Active Context Gateway Rule

1. Run the Service Area Router first: `tools/agent_runtime/service_area_router.py`.
2. Classify the request context and establish T0–T5 budget via `tools/agent_runtime/context_gateway.py`.
3. For every internal file/source, classify and decide with `tools/agent_runtime/context_gateway.py` before reading.
4. If context crosses service areas, require a sanitized handoff from `tools/agent_runtime/handoff_protocol.py`.

| Task type | Context behavior |
|---|---|
| Greeting/simple explanation | Tiny context only |
| Platform Engineering implementation | Scoped repo/runtime/registry/OpenCode context |
| Closure | Current handoff, validators, evidence, artifact drift and git status |
| Launch readiness | Product docs, launch docs, install/rollback, security, privacy, support, compatibility and release map |
| Research Intelligence | External/public/shared-governance context by default |
| Internal OVAV review by Research | Explicit scoped permission or sanitized Platform handoff required |

## Guardrails

- Do not load full catalogs for simple tasks.
- Do not use raw chat history as source of truth.
- Do not treat historical segment evidence as current product identity.
- Deny repo root, `.opencode`, `.ovav/context`, raw snapshots, install artifacts and git history to Research Intelligence by default.
- Sensitive/execution context requires explicit grant even for Platform Engineering.
