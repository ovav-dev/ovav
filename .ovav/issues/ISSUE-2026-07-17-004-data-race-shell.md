# ISSUE-2026-07-17-004: Data Race in Shell Observer

**Date:** 2026-07-17
**Severity:** HIGH
**Status:** FIXED
**Package:** `internal/shell`
**Tests:** `TestT13Emit_Multiple`, `TestT13Run_Success`, `TestT13Run_Failure`

## Problem
Data race detected in shell.go during Emit/Run operations. Multiple goroutines access shared state without synchronization.

## Root Cause
The `Observer.Emit()` method spawns goroutines that access shared event state. The `Run()` method also spawns goroutines. When tests run with `-race`, concurrent access is detected.

## Impact
- Test flakiness under race detector
- Potential runtime corruption in production
- Violates Go memory model

## Fix Required
1. Add mutex or atomic operations to shared state in shell.go
2. Ensure all goroutine-shared variables are properly synchronized
3. Verify with `-race` flag

## Verification (current state)
```
==================
WARNING: DATA RACE
Read at 0x00c0002a4140 by goroutine 38:
  github.com/ovav/ovav/internal/shell.(*Observer).Run()
...
Previous write at 0x00c0002a4140 by goroutine 37:
  github.com/ovav/ovav/internal/shell.(*Observer).Emit()
...
```

## Priority
HIGH — race conditions can cause silent data corruption
