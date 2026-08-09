# ISSUE-2026-07-17-002: Validator Count Drift (80 → 81)

**Date:** 2026-07-17
**Severity:** MEDIUM
**Status:** FIXED
**Package:** `internal/validators`
**Test:** `TestDefaultRegistry_80Validators`

## Problem
Test expected exactly 80 validators but registry now has 81 (added `AdversarialVerification`).

## Root Cause
When adding a new validator (Batch 10: adversarial_verification), the hardcoded count in the test was not updated.

## Fix Applied
- Renamed test to `TestDefaultRegistry_81Validators`
- Updated expected count from 80 to 81

## Verification
```
--- PASS: TestDefaultRegistry_81Validators (0.00s)
PASS
```

## Prevention
- Use `len(DefaultRegistry().All())` dynamically instead of hardcoded count
- Or: auto-update test when new validators are registered
