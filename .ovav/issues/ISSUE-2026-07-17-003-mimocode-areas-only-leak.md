# ISSUE-2026-07-17-003: Mimocode AreasOnly Converter Leaking Files

**Date:** 2026-07-17
**Severity:** HIGH
**Status:** FIXED
**Package:** `internal/convert`
**Test:** `TestMimocodeBrain_CompleteValidation`

## Problem
Mimocode converter with `AreasOnly=true` leaks lead and team files into the output:
- 2 lead files leaked (lead-eidren.md, lead-thavren.md)
- 11 team files leaked (team-aric.md, team-doran.md, etc.)
- Expected 11 files (10 areas + ovav.md), got 24

## Root Cause
The `cleanNonAreaAgents()` function in `convert.go` is not properly filtering out `lead-*` and `team-*` files from the mimocode output directory. The AreasOnly contract (`AreasOnly() = true`) should ensure only area-level agents are published.

## Impact
- Mimocode TUI shows all 24 agents instead of 10 areas
- Violates the AreasOnly contract
- Users see agents they shouldn't have access to

## Fix Required
1. Investigate `cleanNonAreaAgents()` in `convert.go`
2. Ensure it removes `lead-*` and `team-*` from mimocode output
3. Add test assertion for exact file count
4. Verify against OpenCode converter (which respects hidden:true)

## Verification (current state)
```
opencode: AreasOnly runtime leaked 2 lead files into picker
opencode: AreasOnly runtime leaked 11 team files into picker
opencode: expected 11 files (10 areas + ovav.md), got 24
```

## Priority
HIGH — security boundary violation (agents leaked to wrong runtime)
