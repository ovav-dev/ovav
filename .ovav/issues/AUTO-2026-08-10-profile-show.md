# AUTO-2026-08-10-profile-show.md

## Issue: `ovav profile show` returns usage error

**Trace ID**: AUTOTEST-LOOP-2026-08-10
**Command**: `go run -C go-runtime ./cmd/ovav/ profile show`
**Exit Code**: 2

### Evidence
```
Usage: ovav profile <list|apply|remove> [args]
exit status 2
```

### Analysis
`profile show` is not a valid subcommand. Available subcommands are `list`, `apply`, `remove`. The command may have been renamed or removed.

### Expected behavior
Either:
- `ovav profile list` to list profiles
- `ovav profile apply <profile>` to apply a profile

### Severity: LOW (CLI UX issue)

### Fix Applied
Added `show` as alias for `list` in `cmdProfile()` at `main.go:647`.

### Status: FIXED
