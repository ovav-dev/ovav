---
name: ovav-agent-router
description: Use when OVAV needs to detect the domain of a request and route to the correct lead/agent, or when delegating work to a specific OVAV service area.
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

# OVAV Agent Router

Central router for detecting request domains and loading the correct OVAV agent context.

## Domain Detection Matrix

| Domain | Keywords/Signals | Lead | Area | Skill |
|--------|-----------------|------|------|-------|
| **Platform Engineering** | runtime, Go, CLI, security, validation, git governance, install, deploy, system | Thavren | Platform Engineering | `ovav-platform-session` |
| **Digital Product** | React, TypeScript, frontend, components, UI code, web app, Vite, CSS, state management | Dante | Digital Product | — |
| **UX/UI Design** | design, accessibility, WCAG, prototyping, user research, wireframe, mockup, usability | Elena | UX Design | `ovav-ux-session` |
| **Research Intelligence** | research, evidence, benchmark, source verification, decision brief, comparison | Eidren | Research Intelligence | `ovav-research-session` |
| **Commercial & Growth** | business, monetization, pricing, market, growth, revenue, strategy, partnerships | Sofía | Commercial & Growth | `ovav-business-session` |
| **Health & Performance** | nutrition, exercise, meal plan, supplementation, sleep, recovery, clinical | Renata | Health & Performance | `ovav-health-session` |
| **Education & Career** | learning, curriculum, training, career, tutoring, assessment, teaching | Valeria | Education & Career | `ovav-education-session` |
| **Legal & Compliance** | legal, compliance, contract, regulation, GDPR, terms, liability | Camila | Legal & Compliance | — |
| **DevOps & Infrastructure** | CI/CD, Docker, Kubernetes, cloud, AWS, deployment, infrastructure, monitoring | Uriel | DevOps & Infrastructure | — |
| **Adversarial Intelligence** | security testing, penetration, vulnerability, threat, attack surface | Kenji | Adversarial Intelligence | — |

## Routing Protocol

1. **Detect domain** from user request using keyword matrix above
2. **Identify lead agent** — the CLI runtime loads agent profiles natively from its agents directory (`.opencode/agents/` or `.mimocode/agents/`). Each agent file contains its own OVAV_IDENTITY_GUARD.
3. **Delegate to lead agent** using the CLI's native subagent mechanism (Task tool). The lead's agent file provides full identity, functions, limits, and squad.
4. **Apply session skill** if available (e.g., `ovav-platform-session`)
5. **Handoff** with proper delegation protocol

## Agent Registry (Quick Reference)

| Lead | Color | Domain | Hidden |
|------|-------|--------|--------|
| Thavren | 🔵 `#2563eb` | Platform Engineering | yes (primary) |
| Dante | 🟣 `#7c3aed` | Digital Product | yes |
| Elena | 🩷 `#ec4899` | UX/UI Design | yes |
| Eidren | 🟡 `#f59e0b` | Research Intelligence | yes |
| Sofía | — | Commercial & Growth | yes |
| Renata | — | Health & Performance | yes |
| Valeria | — | Education & Career | yes |
| Camila | — | Legal & Compliance | yes |
| Uriel | — | DevOps & Infrastructure | yes |
| Kenji | — | Adversarial Intelligence | yes |

## Handoff Protocol

When delegating cross-area, use:

```
SERVICE_AREA: {area}
VISIBLE_PROFILE: {profile}
LEAD: {lead}
TASK_CLASS: {micro|simple|medium|complex|critical}
RISK_LEVEL: {low|medium|high|critical}
DELEGATION_MODE: {lead_only|skill_only|focused_squad|full_squad|critical_squad}
CONTEXT_MODE: {none|state|full}
ALLOWED_CONTEXT: {what to share}
DENIED_CONTEXT: {what to protect}
TRACE_ID: {unique id}
```

## Guardrails

- Default to Platform Engineering (Thavren) for ambiguous requests
- Never expose internal squad roles to user
- Cross-area transfer requires sanitized handoff
- Each area has hard boundaries — respect them

## Delegation Paths

### LEAD → SAME LEAD (squad delegation): Use workflow+agent()

For delegating within your own team (Thavren → Clara, Eidren → Carmen), use:
```javascript
workflow({
  script: `
    export const meta = { name: "delegate-to-squad" };
    const result = await agent("Task description", {
      subagent_type: "TEAM_ID",  // e.g., "team-clara", "team-carmen"
      model: "opencode-go/deepseek-v4-pro",
    });
    return result;
  `
})
```

**Do NOT use `actor.run({ subagent_type: "team-clara" })`** — it silently falls back to `general` and renders as **"GENERAL TASK"**.

### LEAD → DIFFERENT LEAD (cross-LEAD handoff): Use file-based YAML handoff

For delegating to a different LEAD (Thavren → Eidren, Sofía → Valeria), write:
`.ovav/handoffs/<trace-id>-<topic>.md` with YAML frontmatter + body, then the Orchestrator routes at next session boundary.

**Do NOT use `actor.run({ subagent_type: "eidren" })`** — it falls back to `general` and the LEAD name is lost.

### Agent-to-Agent (within same agent context): Use @mention

For agents in the same context, @mention the team agent directly in a message.
