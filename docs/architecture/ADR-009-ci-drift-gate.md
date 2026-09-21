# ADR-009: CI Drift Gate

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 1 anti-drift core), ADR-007 (drift detection), ADR-008 (deploy pipeline)
**Decider:** Thavren + CEO

## Context

After implementing `ovav drift show` (D4) and `ovav deploy run` (D1), the
next protection layer is preventing drift from being pushed to develop
without being addressed.

Currently:
1. Operator edits fragment locally
2. Commits + pushes to develop
3. Drift persists in user's live system (unaware)
4. User reports "keybindings broken" weeks later

We need a gate that catches drift BEFORE the push reaches develop.

## Decision

Add a pre-push hook that runs `ovav drift show --json` and blocks the
push if drift is found.

### Architecture

```
git push origin <branch>
       │
       ▼
[pre-push hook triggered]
       │
       ▼
[Run: ovav drift show --json]
       │
       ▼
[Parse JSON: drifted_targets > 0?]
       │
   ┌───┴───┐
   │       │
   No      Yes
   │       │
   ▼       ▼
  Allow  Block (suggest `ovav deploy run`)
```

### Exit codes (drift show)

| Code | Meaning |
|------|---------|
| 0 | No drift → push allowed |
| 1 | Drift found → push blocked |
| 2 | ovav binary missing → push allowed (fallback) |

### Bypass mechanism

```bash
# Skip drift check (NOT recommended)
OVAV_BYPASS_DRIFT_CHECK=1 git push origin feat-branch

# Or skip ALL hooks
git push --no-verify origin feat-branch
```

### Scope (initial)

- Only checks drift when pushing to `develop` (most common)
- Bypasses for `main`, `master`, feature branches (no enforcement)
- Configurable via `.ovav/hooks/pre-push-config.json`

### Integration with CI

The same logic is exposed as `ovav ci drift-check` for CI runners:

```bash
# In .github/workflows/ci.yml or similar:
- name: OVAV drift check
  run: |
    ./bin/ovav ci drift-check
    if [ $? -ne 0 ]; then exit 1; fi
```

Exit codes match the hook.

## Consequences

### Positive

- **Prevents regression** — drift can't land in develop unnoticed
- **Self-documenting** — error message tells user what to do
- **Composable** — same logic in hook + CI
- **Bypassable** — for power users / emergency fixes

### Negative

- **Push friction** — every push to develop now runs validators
- **Hook installation** — operator must run `ovav hooks install-pre-push`
- **false positives** — pinned baseline drift is acceptable

### Mitigations

- `--no-verify` escape hatch
- Config file for scope (which branches to enforce)
- Pinned-baseline drift treated as WARN not FAIL

## References

- ADR-005 Phase 1 D2
- ADR-007 drift detection (consumed)
- ADR-006 baseline versioning (pinned-baseline handled)
- ADR-008 deploy pipeline (suggested fix)
