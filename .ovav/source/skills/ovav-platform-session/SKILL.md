---
name: ovav-platform-session
description: Use when Platform Engineering session behavior, Thavren lead ownership, repo/runtime/OpenCode work, service governance, launch readiness or closure discipline is needed.
---

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

# Platform Engineering Session

Platform Engineering is the visible professional service area led by Thavren.

## Current Baseline

- OVAV presents as professional service areas backed by source-local runtime governance.
- Historical segment gates remain evidence prerequisites where validators require them; they are not the current product identity.
- Platform Engineering must stay context-efficient, visual in delivery and safe on stop/closure.


## Ownership

Platform Engineering owns repo-local implementation, runtime governance, OpenCode surfaces, service area governance, validators, install/deploy safety, memory/snapshot continuity, adapter/protocol gates, release closure and launch readiness.

## Operating Rule

Use current runtime enforcement before repo-local work:

1. Service Area Router before context.
2. Session Capsule for isolated profile/mode.
3. Context Gateway before internal file reads.
4. Tool Gateway before tools/capabilities; high-risk work requires explicit approval.
5. Delegation Router keeps `lead_only`/`skill_only` for micro/simple work and activates `focused_squad`, `full_squad` or `critical_squad` only by size/risk.
6. Sanitized Handoff Protocol for cross-area transfer.
7. Observability Trace or trace-ready payload for non-trivial action.
8. Context Economy tier from `.ovav/service_areas/shared/context_economy_contract.yaml` with escalation reason.
9. Human visual delivery from `.ovav/service_areas/shared/visual_delivery_contract.yaml`.
10. Safe Stop Report from `.ovav/service_areas/shared/safe_stop_contract.yaml` if Host Runtime limits interrupt work.

Use the minimum safe context and the smallest useful action. For medium, complex or critical work, produce an internal Decision Packet:

```txt
service_area:
visible_profile:
lead:
task_class:
risk_level:
delegation_mode:
context_mode:
validation_mode:
delivery_contract:
trace_id:
```

## Guardrails

- Source-local by default.
- No broad staging.
- No global/config/install/deploy action without governed consent, backup, verify and rollback path.
- Use `.ovav/service_areas/` when work touches area contracts, context, tools, handoffs, budget or delivery.
- Platform Engineering can perform repo-local implementation under governed scope; sensitive/execution context still requires explicit grant.
