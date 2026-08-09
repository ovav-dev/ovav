# ISSUE-2026-07-17-007: Stale Layer Health Values in caps.yaml

**Date:** 2026-07-17
**Severity:** LOW
**Status:** FIXED
**File:** `.ovav/plan/caps.yaml`

## Problem
Two different health assessments exist for layers:
1. `health_by_layer` section (in current_state) — shows L1=80%, L2=72%, etc.
2. `layered_view` section — shows L1=55%, L2=62%, etc.

These are inconsistent. The `layered_view` values appear more current (lower percentages reflect real gaps).

## Root Cause
`health_by_layer` was last updated in Sprint 7. `layered_view` has been updated more recently but still shows gaps.

## Impact
- Misleading health metrics
- Could cause premature Phase 2 activation
- Caps schema validator may flag inconsistency

## Fix Required
1. Reconcile both sections to use same values
2. Prefer `layered_view` values (more conservative, more current)
3. Add auto-validation: `caps_chronos_alignment` should detect stale health values
4. Consider making health values computed from actual test results

## Verification
Current `health_by_layer`:
```
L0_integrity_safety: 94%
L1_architecture: 80%
L2_security: 72%
L3_operations: 80%
L4_quality: 73%
L5_agent_governance: 75%
L6_migration_debt: 55%
```

Current `layered_view`:
```
L0: 92%
L1: 55%
L2: 62%
L3: 78%
L4: 55%
L5: 60%
L6: 45%
```

## Priority
LOW — informational, but should be fixed for accuracy
