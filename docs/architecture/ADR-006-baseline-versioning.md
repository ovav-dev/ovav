# ADR-006: Versioned Runtime Integrity Baseline

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 1 anti-drift core)
**Decider:** Thavren + CEO

## Context

Runtime integrity baseline (`.ovav/integrity_backups/baseline.json`) is currently:
1. **Gitignored** — never versioned
2. **Per-worktree** — each worktree has different baseline
3. **Mutable** — regenerated on demand via `ovav integrity baseline --write`
4. **Unaudited** — no record of when/who/why baseline changed

This creates drift risk: a worktree's local baseline may differ from the
"last known good" baseline that protected surfaces were tested against.

## Decision

Adopt **two-file model** for integrity baseline:

| File | Purpose | Tracked? | Writable by |
|------|---------|----------|-------------|
| `.ovav/integrity_backups/baseline.json` | Current operational baseline | **YES** (git) | Worktree integrity commands |
| `.ovav/integrity_backups/baseline.pinned.json` | Last approved baseline | **YES** (git) | Only via `ovav integrity pin --approve` (CEO waiver) |

The pinned baseline represents the **last CEO-approved protected surface
state**. Drift detection = current baseline vs pinned baseline.

## Architecture

### Drift detection (3 levels)

1. **L1 — Core files drift** (existing `runtime_integrity` validator)
   Compares actual file hashes vs `baseline.json`.
2. **L2 — Baseline freshness** (NEW `integrity_baseline_fresh` validator)
   Checks if `baseline.json` is > 7 days old (configurable).
3. **L3 — Pinned drift** (NEW `pinned_baseline_drift` validator)
   Compares `baseline.json` vs `baseline.pinned.json`.
   FAIL if any pinned file's expected hash != current.

### Auto-update ceremony

Every commit that touches a protected surface MUST also update `baseline.json`.
Implemented via:

1. **Pre-commit hook** — refuses commit if:
   - Changed files include any in `coreFiles`
   - AND `baseline.json` was NOT updated in same commit
2. **Worktree validator** — `integrity_baseline_fresh` warns if > 7 days
3. **CI gate** — runs pre-commit logic + freshness on every PR

### Pin workflow

```
# After fixing a bug or merging a feature:
git worktree add feature/foo
# ... make changes, update baseline.json ...
go run -C go-runtime ./cmd/ovav/ integrity pin  # creates baseline.pinned.json

# CEO approves:
go run -C go-runtime ./cmd/ovav/ integrity pin --approve  # promoted to default
```

## Consequences

### Positive

- **Auditable history** — git log shows every baseline change
- **CEO control** — pinned baseline requires explicit approval
- **Drift visibility** — 3-level detection catches regressions
- **CI-ready** — pre-commit hook + freshness validator = automated enforcement

### Negative

- **Discipline required** — every surface change needs baseline update
- **Two-file complexity** — operators must understand pinned vs current
- **CEO bottleneck** — pinning requires CEO approval

### Mitigations

- Hooks make discipline automatic (can't forget)
- Clear docs separate pinned vs current roles
- `ovav integrity pin --approve` can be batched via CEO session waiver

## References

- ADR-005 Phase 1 (anti-drift core)
- `go-runtime/internal/validators/runtime_integrity.go`
- `go-runtime/cmd/ovav/integrity_cli.go`
