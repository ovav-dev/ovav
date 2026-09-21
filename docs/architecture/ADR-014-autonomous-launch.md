# ADR-014: Zero-Touch Launch Assistant

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-013 (GA promotion), ADR-003 (launch verification), ADR-005 (Phase 4)
**Decider:** Thavren + CEO

## Context

Previous design (ADR-013) required CEO to remember and run multiple commands in
sequence. Per CEO directive: "process must be autonomous, secure, no commands
to memorize, no tech debt."

Risk: human-in-the-loop creates friction → CEO skips steps → tech debt
accumulates → system drift → problems for OVAV devs and users.

## Decision

Replace multi-command ceremony with **ONE smart entry point**: `ovav launch`
(no subcommand = interactive wizard).

The wizard:
1. **Detects** current state automatically (no CEO knowledge needed)
2. **Executes** safe actions automatically (evidence capture, status checks)
3. **Prompts** only for decisions that REQUIRE CEO consent (with clear reasoning)
4. **Validates** post-state (rollback if anything went wrong)
5. **Logs** every step to audit trail

### Architecture

```
ovav launch  (no args)
   │
   ▼
[Smart state detection]
   │  - validate status (passing?)
   │  - drift check (clean?)
   │  - pinned baseline (exists?)
   │  - tag (exists?)
   │  - evidence (captured?)
   │
   ▼
[Build readiness report]
   │  Each gate: ✅/⏳/❌
   │
   ▼
[Execute auto-actions]
   │  - Capture evidence (if missing)
   │  - Generate reports
   │  - All safe, no CEO input needed
   │
   ▼
[Prompt for CEO decisions]
   │  Only gates that REQUIRE consent:
   │  - Pin approval
   │  - GA verification
   │  - Tag push
   │  Each prompt shows: what, why, reversible?, expected outcome
   │
   ▼
[Validate post-state]
   │  Re-run state detection
   │  If anything regressed → automatic rollback
   │
   ▼
[Final summary + audit log]
```

### Smart defaults

| CEO intent | System action |
|------------|----------------|
| "¿Está listo?" | `ovav launch` → readiness report |
| "¿Cómo voy?" | `ovav launch --status` (CI-friendly) |
| "Auto-prep todo" | `ovav launch --prepare` (no prompts) |
| "Hazlo todo" | `ovav launch --all --reason="..."` (full ceremony) |
| "Solo info" | `ovav launch --info` (read-only) |

### Safety guarantees

1. **No memory burden** — single command, smart defaults
2. **Reversible by default** — every action has `--undo`
3. **Audit trail** — every decision logged with reason
4. **Auto-rollback** — if post-state validation fails
5. **No silent changes** — every step prints what it did
6. **CEO-gated irreversible** — only `verify`/`tag-push`/`pin-approve` require explicit consent
7. **Idempotent** — running twice produces same result
8. **Self-documenting** — `--help` explains entire flow

## Consequences

### Positive
- **Zero memorization** — single entry point
- **No tech debt** — system self-maintains
- **No skipped steps** — wizard enforces completeness
- **No rushed decisions** — CEO sees full context before consenting
- **No silent failures** — every step validated

### Negative
- **More code** — wizard logic
- **Slower than `--all`** — interactive pauses (but `--all` is available for CI)

### Mitigations
- All auto-actions are also exposed as flags for power users
- Audit log available via `ovav audit show`

## References
- ADR-003 launch verification
- ADR-005 Phase 4 (D1-D4)
- ADR-013 GA promotion (superseded for ceremony flow)
- CRIT-006 (CEO decisions stay human)
