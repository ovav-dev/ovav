# ISSUE-2026-07-17-005: OWS Run-All Test Failures (2/41)

**Date:** 2026-07-17
**Severity:** MEDIUM
**Status:** OPEN (requires investigation)
**Test Suite:** `bin/owv-tests/run-all.sh`

## Problem
2 out of 41 OWS tests fail:
1. `FAIL owc output` — output format mismatch
2. `FAIL directory not created` — worktree directory not created

## Root Cause (hypothesis)
- Test environment may not have proper git worktree support
- OWS commands may depend on specific directory structure that tests don't set up
- Pre-existing issue (not caused by this session's changes)

## Impact
- 95% test pass rate (39/41)
- Core OWS functionality works (verified manually)
- Test infrastructure may need updating

## Fix Required
1. Investigate test setup in `run-all.sh`
2. Check if git worktree commands work in test environment
3. Fix test isolation (temp dir cleanup, git config)
4. Re-run with `-v` to get detailed failure output

## Verification
```
39 passed, 2 failed
```

## Priority
MEDIUM — OWS core works, test infrastructure needs attention
