# Research Intelligence — Harness-Governed Session Routing

## Principle

Every interaction is governed by the harness intelligence router. The router selects the right harness family automatically based on user intent, visible service category, lane and risk level. The user never sees harness mechanics.

## Context → Harness Families

| Context | Harness families | Response depth |
|---|---|---|
| Greeting | identity_baseline, session_baseline (minimal) | minimal |
| Source verification | identity + session + research_evidence + safety | evidence_first |
| Benchmark / comparison | identity + session + research_evidence | evidence_first |
| Decision / recommendation | identity + session + research_evidence + safety | decision_artifact |
| Validation closure | identity + session + closure_strict + artifact_drift + git_safety + safety | closure_standard |
| Safety / high-risk | identity + session + safety_gate (immediate) | safety_first |

## Progressive Resource Loading

- Basic greetings use the smallest possible harness set.
- Implementation tasks grow to include segment, registry and skill harnesses.
- Closure tasks include strict validation, artifact drift and git-safety checks.
- High-risk contexts escalate to safety gates immediately.
- No existing harness capability is removed.

## Silent Harness Governance

Use runtime gates as backend proof. Do not ask the user to remember commands before giving a useful answer.

Blocked surfaces:

- Global config writes.
- OpenCode global config writes.
- Plugin installation.
- Live Engram reads, writes, configuration or installation.
- Real install, apply, backup or rollback behavior.
- UI/TUI, MCP/A2A and external service behavior.
- Production-ready or global-ready claims.
- New public profiles.
