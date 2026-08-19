# P27 — Memory Model Definition

Per plan §27, formal memory hierarchy:

```
OVAV MEMORY          → canonical knowledge
AGENTS.md            → canonical rules
.agents/skills/      → procedures (now .ovav/source/skills/)
Git                  → code truth
OpenCode Session     → harness context (temporary)
Warp Cloud Conv      → operational history (flight recorder)
```

## Memory boundaries

| Memory | Mutable by | Persistent | Authority |
|---|---|---|---|
| OVAV Memory | OpenCode skills only | Yes (memory bridge) | Canonical |
| AGENTS.md | CEO + workflow | Yes (Git) | Canonical |
| Skills | OpenCode + ceo waiver | Yes (Git) | Canonical |
| Git HEAD | Anyone with write | Yes (Git) | Truth |
| OpenCode session | OpenCode | No (per-session) | Transient |
| Warp Cloud Conv | Warp Drive | Yes (Warp Cloud) | Operational |

## Cloud Conversations role

Cloud Conv is **NOT** a knowledge base. It is:
- A flight recorder
- A handoff tool
- A historical search
- An audit aid

It does NOT store decisions that haven't been promoted to:
- OVAV Memory
- Git (docs/ code/ registry/)
- AGENTS.md

## Promotion rules

When a decision is made in Cloud Conv:
1. CEO or agent decides if it should be canonical
2. Promote to OVAV Memory or Git
3. Promote link in Cloud Conv (for audit)
4. Never store canonical decisions only in Cloud Conv

## Status

✅ P27 100% — memory model defined.
