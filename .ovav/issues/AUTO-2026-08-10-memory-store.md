# AUTO-2026-08-10-memory-store.md

## Issue: `ovav memory store` --rule flag not recognized when positional args come first

**Trace ID**: AUTOTEST-LOOP-2026-08-10
**Command**: `go run -C go-runtime ./cmd/ovav/ memory store "TEST" "autonomous test card" --rule "test" --agent thavren`
**Exit Code**: 2

### Evidence
```
ovav memory store: --rule is required
exit status 2
```

### Root Cause
Go's `flag.Parse()` stops parsing flags at the first positional argument. The command was invoked as:
```
memory store "TOPIC" "SUMMARY" --rule "rule text"
```
But Go's flag package sees positional "TOPIC" first and treats everything after as positional, not flags.

### Correct Usage
```
memory store --rule "rule text" --agent thavren "TOPIC" "SUMMARY"
```

### Impact
CLI help and examples show `topic` and `summary` as positional arguments first, which is inconsistent with Go's flag parsing behavior. User confusion likely.

### Fix Required
1. Update help examples to show flags BEFORE positional arguments, OR
2. Pre-process args to reorder them before flag parsing, OR
3. Document this behavior explicitly

### Severity: LOW (CLI UX issue, not a crash)

### Status: OPEN
