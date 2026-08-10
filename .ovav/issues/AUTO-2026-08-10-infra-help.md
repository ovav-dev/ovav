# AUTO-2026-08-10-infra-help.md

## Issue: `ovav infra --help` returns error instead of help

**Trace ID**: AUTOTEST-LOOP-2026-08-10
**Command**: `go run -C go-runtime ./cmd/ovav/ infra --help`
**Exit Code**: 1

### Evidence
```
Unknown infra subcommand: --help
Run 'ovav infra' for help.
exit status 1
```

### Analysis
The `infra` command incorrectly treats `--help` as a subcommand instead of a common flag. Should show help when `--help` is passed.

### Severity: LOW (CLI UX issue)

### Fix Applied
Added `--help` and `-h` check in `cmdInfra()` at `main.go:3157` to show help when passed.

### Status: FIXED
