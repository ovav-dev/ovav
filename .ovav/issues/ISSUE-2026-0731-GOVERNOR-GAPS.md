# ISSUE-2026-0731-GOVERNOR-GAPS — Remaining governor coverage gaps

**Severity:** 🟡 Low  
**Status:** Open  
**Detected:** 2026-07-31  
**Affects:** `go-runtime/internal/governor/`

---

## Current State

Governor coverage: **90.6%** ✅ (Sprint 2 target: 85%)

---

## Remaining Gaps (all below 100%)

| Function | File | Coverage | Status |
|---|---|---|---|
| `QuickSelfDiagnosis` | bridge.go | 76.9% | 🟡 Can be improved |
| `maybeRotate` | session_feed.go | 29.4% | 🟡 Needs test |
| `SyncSessionState` | session_feed.go | 66.7% | 🟡 Needs test |
| `Rotate` | session_feed.go | 76.9% | 🟡 Needs edge case |
| `Accept` | delegation_protocol.go | 80.0% | 🟡 Needs edge case |
| `Complete` | delegation_protocol.go | 80.0% | 🟡 Needs edge case |
| `Reject` | delegation_protocol.go | 80.0% | 🟡 Needs edge case |

---

## Recommended Action

All gaps are ≤ 23% missing. For 95%+ coverage, add:

1. `TestQuickSelfDiagnosis_AllChecksPass` — simulate all checks passing
2. `TestQuickSelfDiagnosis_SomeChecksFail` — simulate partial failure
3. `TestMaybeRotate` — test rotation condition logic
4. `TestSyncSessionState` — test state persistence
5. `TestRotate` — test rotation with empty/non-empty feed
6. `TestTaskQueue_Accept/Complete/Reject_EdgeCases` — error paths

---

## Priority

**P3** — Governor is already at 90.6%. These edge cases are minor. Can be addressed in a future cleanup sprint.
