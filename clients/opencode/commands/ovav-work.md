

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

# ovav-work

Start OVAV work inside OpenCode without thinking in terminal commands.

## Current Purpose

- Route the request to the right professional service area before loading context.
- Use the smallest useful context tier and delegation mode.
- Deliver in a human, visual, safe and concise format.

Historical segment gates remain evidence prerequisites where relevant; they are not the current product identity or workflow label.

## Visible Service Areas

| Task shape | Visible service area |
|---|---|
| Runtime, CLI, OpenCode, source-local repair, validation, safety, install governance, launch readiness | OVAV Platform Engineering |
| Research, benchmark, source verification, evidence scoring, contradiction analysis, decision brief | OVAV Research Intelligence |
| Ambiguous request | Ask one compact Spanish clarification |

## Active Runtime Enforcement Path

`ovav-work` is governed by current runtime enforcement, not prompt-only policy.

1. **Service Area Router first** — before loading repo/internal context, resolve the request through `tools/agent_runtime/service_area_router.py` and produce: `service_area`, `visible_profile`, `lead`, `task_class`, `risk_level`, `delegation_mode`, `context_mode`, `validation_mode`, `delivery_contract`.
2. **Context Gateway** — before reading internal files, classify the source/path with `tools/agent_runtime/context_gateway.py`.
3. **Tool Gateway** — before tools/capabilities, evaluate area, mode and risk with `tools/agent_runtime/tool_gateway.py`.
4. **Delegation Router** — keep `lead_only`/`skill_only` for micro/simple tasks; use `focused_squad`, `full_squad` or `critical_squad` only by size/risk.
5. **Handoff Protocol** — any cross-area transfer must be sanitized with purpose, allowed_context, denied_context, scope and trace_id.
6. **Context Economy** — choose `T0_none`, `T1_tiny`, `T2_compact`, `T3_focused`, `T4_full_scoped` or `T5_closure_grade` with an escalation reason.
7. **Work** — execute the smallest safe source-local action allowed by the gateways.
8. **Observability Trace** — non-trivial work must produce a trace event or trace-ready payload with service area, lead, mode, decision, source/tool/handoff decisions, validation and delivery contract.
9. **Validate and close** — run task-appropriate validators and deliver in the selected contract: `consultation`, `diagnosis`, `implementation_delivery`, `research_decision` or `closure`.
10. **Safe Stop** — if the Host Runtime reaches a limit, emit a Safe Stop Report and distinguish Host Runtime from OVAV Runtime.

## Guardrails

- Source-local by default.
- Global/config/install/deploy actions require governed consent, backup, verify and rollback path.
- MCP/A2A/external adapters are gated and scoped, not free access.
- Research Intelligence has no default repo-root/internal OVAV access, even when the user mentions OVAV; use public/external and shared-governance context unless explicit scope or sanitized handoff exists.
- Platform Engineering may perform repo-local implementation under governed scope; sensitive execution still requires explicit grant.
- No production/global-ready claim until launch readiness gates pass.
- Never `git add .`.
