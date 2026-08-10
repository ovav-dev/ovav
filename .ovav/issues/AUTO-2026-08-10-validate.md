# AUTO-2026-08-10-validate.md

## Issue: `ovav validate` fails with 65/70 validators passed

**Trace ID**: AUTOTEST-LOOP-2026-08-10
**Command**: `go run -C go-runtime ./cmd/ovav/ validate`
**Exit Code**: 1

### Evidence
```
✅ Adversarial Verification       No claims requiring adversarial verification
... [truncated]
65/70 validators passed ❌
exit status 1
```

### Analysis
5 validators are failing. Full output saved to `/home/braka/.local/share/opencode/tool-output/tool_fea46449c001DEZTa6YjZXulkn`

### Severity: MEDIUM
Validators failing means security/quality gates are not fully passing.

### Status: OPEN
