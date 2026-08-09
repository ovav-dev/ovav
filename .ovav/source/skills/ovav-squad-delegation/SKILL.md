---
name: ovav-squad-delegation
description: Use when an OVAV lead needs to delegate work to squad members. Routes intent to correct team subagent via workflow() + agent(). Replaces actor.run which only accepts explore/general types.
license: Apache-2.0
metadata:
  author: thavren (OVAV)
  version: "1.1"
owner_profile: ovav_systems_architect
owner_lane: runtime_governance
status: active
memory_write: scoped
memory_write_scope: runtime_governance_only
risk_level: low
last_updated: 2026-07-28
---

# OVAV Squad Delegation — Workflow-Based Routing

## Purpose

**CRITICAL LIMITATION (v1.1):** MiMoCode's `actor` tool only accepts `explore` and `general` as `subagent_type` values. Any other value — including LEAD names ("eidren") and team IDs ("team-clara", "team-andres") — is silently treated as invalid and falls back to `general`.
The UI then renders: **"GENERAL TASK"** (no LEAD name visible).

Team agents (team-clara, team-andres, etc.) can ONLY be invoked by the user via @mention
or direct selection. Leads CANNOT delegate to their squad via `actor.run`.

This skill provides the correct delegation pattern using MiMoCode's `workflow` tool,
which CAN spawn any registered agent via the `agent()` function.

## Squad Delegation Pattern

**Bug B fix — C5-T3:** El `actor` tool de MiMoCode solo acepta `explore`/`general`.
La solución es invocar el workflow `ovav-delegate` que usa `workflow + agent()` internamente:

```javascript
workflow({
  name: "ovav-delegate",
  args: {
    agent_id: "lead-eidren",   // lead or team ID (see directory below)
    task: "Full task description with context",
    context: "state",          // state=checkpoint summaries, full=all context, none=clean
    model: "opencode-go/deepseek-v4-pro",  // optional
  },
})
```

El workflow `ovav-delegate.js` (`.ovav/source/workflows/ovav-delegate.js`) resuelve
el `agent_id` a su `subagent_type` canónico y usa `agent({ subagent_type: "lead-eidren", ... })`
que el `actor` tool no soporta.

## Complete Squad Directory

| Lead | Squad Members (subagent_type) |
|---|---|
| **Thavren** (platform_engineering) | `team-andres`, `team-lucas`, `team-helena`, `team-irene`, `team-diana`, `team-pablo`, `team-oscar`, `team-nora`, `team-nadia`, `team-mia`, `team-clara`, `team-marco` |
| **Dante** (digital_product) | `team-sergio`, `team-elena-frontend`, `team-uriel-devops` |
| **Elena** (ux_design) | `team-felipe`, `team-beatriz`, `team-rosa`, `team-sara`, `team-teo` |
| **Eidren** (research_intelligence) | `team-carmen`, `team-celia`, `team-fatima`, `team-ines`, `team-kaori`, `team-mei` |
| **Sofía** (commercial_growth) | `team-gabriela`, `team-gael`, `team-leon`, `team-marina`, `team-oliver`, `team-ramiro`, `team-victor` |
| **Uriel** (devops_infrastructure) | `team-camila`, `team-diego` |
| **Renata** (health_performance) | `team-antonio`, `team-bruno`, `team-karina`, `team-luna`, `team-paula`, `team-ryu`, `team-sandra` |
| **Kenji** (adversarial_intelligence) | `team-akiko`, `team-hiroshi`, `team-ruben`, `team-silvia`, `team-tomas` |
| **Valeria** (education_career) | `team-carmen`, `team-hugo`, `team-julian` |

## Cross-LEAD Delegation (LEAD → LEAD)

When a LEAD needs to hand off to a DIFFERENT LEAD (e.g., Thavren → Eidren, Sofía → Valeria):

**Use the file-based YAML handoff** — NOT `actor.run` or `workflow`.

1. Create `.ovav/handoffs/<trace-id>-<topic>.md` with this frontmatter:
```yaml
---
SERVICE_AREA: research_intelligence
VISIBLE_PROFILE: Eidren
LEAD: eidren
TASK_CLASS: complex
RISK_LEVEL: medium
DELEGATION_MODE: lead_only
CONTEXT_MODE: state
ALLOWED_CONTEXT: benchmark_results, caps.yaml_summary
DENIED_CONTEXT: vault_credentials, ceo_session
TRACE_ID: <unique-id>
---
<Task description and context>
```

2. The Orchestrator (CEO/Agente principal) routes the handoff at next session boundary.

**Do NOT use `actor.run(subagent_type="eidren")`** — it silently falls back to `general`.

## Notes

- `actor.run` ONLY works for `explore` and `general` — use `workflow` for squad members.
- Team agents inherit their parent lead's permission scope when spawned via `workflow`.
- Always pass `context: "state"` so the subagent gets checkpoint summaries.
- Results from `workflow` are returned inline (blocks until complete).
- For parallel delegation, use `parallel()` inside workflow scripts.
- Cross-LEAD delegation must use file-based handoff at `.ovav/handoffs/` — not `actor.run`.
- The "GENERAL TASK" label in the UI is the symptom of using `actor.run` with a non-explore/general subagent_type.

## Verificación — Bug B Fix (C5-T3)

```bash
# Catalogo: 10 leads + 60 teams
go run ./go-runtime/cmd/resolve_subagent/ --list | grep -c "lead-"   # → 10

# Resolución correcta
go run ./go-runtime/cmd/resolve_subagent/ eidren        # → lead-eidren ✅
go run ./go-runtime/cmd/resolve_subagent/ team-clara    # → team-clara ✅

# Workflow disponible
ls .mimocode/workflows/ovav-delegate.js  # → existe ✅
```
