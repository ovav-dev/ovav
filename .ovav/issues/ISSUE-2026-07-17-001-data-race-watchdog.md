# ISSUE-2026-07-17-001: Data Race in Watchdog Test

**Date:** 2026-07-17
**Severity:** HIGH
**Status:** FIXED
**Package:** `internal/watchdog`
**Test:** `TestT13SetNotifyCallback`

## Problem
Data race on `called` variable in test. Callback runs in goroutine but test reads `called` without synchronization.

## Root Cause
The `notify()` method spawns a goroutine that calls the callback, but the test reads the `called` flag without waiting for the goroutine to complete. `time.Sleep(50ms)` is not sufficient synchronization.

## Fix Applied
- Replaced `bool` with `atomic.Bool` for thread-safe read/write
- Added `sync/atomic` import
- Test now passes with `-race` flag

## Verification
```
=== RUN   TestT13SetNotifyCallback
--- PASS: TestT13SetNotifyCallback (0.08s)
PASS
ok  github.com/ovav/ovav/internal/watchdog  1.568s
```

## Prevention
All tests using goroutine-shared variables must use atomic operations or sync primitives. Add `-race` to CI.
