# ADR-011: Auto-Remediation (ovav validate --fix)

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 2 / D1)
**Decider:** Thavren + CEO

## Context

After Phase 1, we have 76 validators running on every commit. Many produce
**known issues with known fixes** (e.g., `bash_readline_bindings` warns when
inputrc is missing the marker — fix is to add it). Currently these require:

1. Run `ovav validate` → see warning
2. Read docs / source to understand fix
3. Manually edit file
4. Re-run `ovav validate` to confirm

This is "the patch loop" repeated. Phase 2 eliminates it.

## Decision

Add `ovav validate --fix` that applies auto-corrections for SAFE_FIX
validators. Pattern per validator:

```go
// SAFE_FIX: Add 'deliberately UNBOUND' marker to inputrc
func (v *BashReadlineBindings) Fix() error {
    // ... implementation ...
}
```

Each validator declares an optional `Fix()` method. The orchestrator
collects all `Fix()` implementations, applies them atomically with rollback.

### Architecture

```
ovav validate --fix [--dry-run] [--strategy=atomic|best-effort]
   │
   ▼
[1. Snapshot all affected files]
   │
   ▼
[2. Run validators, collect SAFE_FIX candidates]
   │
   ▼
[3. Group fixes by file (avoid conflicts)]
   │
   ▼
[4. Apply each fix (atomic write)]
   │
   ▼
[5. Verify: re-run validators, check issue resolved + no regressions]
   │
   ▼
[6. On failure → rollback snapshot]
   │
   ▼
[7. Log to .ovav/registry/auto_fix_history.jsonl]
```

### Safety guards (HARD requirements)

1. **Snapshot before any fix** — full file backup per fix target
2. **Rollback on regression** — if a fix introduces NEW issues, restore
3. **Rollback on no-op** — if a fix doesn't resolve the original issue, restore
4. **Max 10 fixes per run** — prevent runaway
5. **No fix on protected files** — `.ovav/plan/caps.yaml`, `AGENTS.md`,
   `permission_authority.json` require CEO waiver
6. **Logged every attempt** — operator + timestamp + outcome

### Phase 2 initial SAFE_FIX validators

| Validator ID | Fix |
|--------------|-----|
| `bash_readline_bindings` | Add 'deliberately UNBOUND' marker to ~/.inputrc |
| `runtime_integrity_baseline_fresh` | Regenerate baseline.json to match current hashes |
| `supply_chain` | Regenerate sbom.json to match current files |

These are LOW RISK because:
- They modify local-only state
- They are idempotent (regenerating produces same output)
- They don't change system behavior (only metadata)

### Operator experience

```bash
# List what would be fixed
ovav validate --fix --list

# Dry-run
ovav validate --fix --dry-run

# Apply (atomic — all-or-nothing)
ovav validate --fix

# Apply (best-effort — keep going on failures)
ovav validate --fix --strategy=best-effort

# Bypass protected-file check (CEO waiver)
ovav validate --fix --ceo-waiver --reason="phase 3 promotion"
```

## Consequences

### Positive

- **Eliminates patch loop** — validators self-heal
- **Reduced operator burden** — no more reading docs to fix simple issues
- **Auditable** — every fix logged
- **Reversible** — snapshot + rollback safety
- **Composable** — `ovav validate --fix && ovav deploy run` as CI gate

### Negative

- **Risk of bad fixes** — validator logic could be wrong (mitigated by safety guards)
- **Snapshot overhead** — every run creates backup files
- **Protected file bypass** — CEO waiver required (proper governance)

### Mitigations

- Pre-flight validators run AFTER fixes to catch regressions
- Snapshot directory in `.ovav/registry/snapshots/fix-<id>/`
- Auto-fix history with operator + git commit hash
- Protected files require explicit `--ceo-waiver`

## References

- ADR-005 Phase 2 / D1
- ADR-006 baseline versioning (related to runtime_integrity fix)
- `.ovav/hooks/pre-commit` (already enforces baseline update)