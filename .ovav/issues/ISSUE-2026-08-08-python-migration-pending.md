# ISSUE: Python Migration Pending — tools/ directory

**Date**: 2026-08-08  
**Severity**: HIGH  
**Owner**: thavren (Platform Engineering Lead)  
**Status**: OPEN

## Summary

Python migration from tools/ directory to Go is **NOT COMPLETE**. According to caps.yaml v102.3, governance surface migration is only 30% complete.

## Files Pending Migration (52 files, ~450K LOC)

### Critical Path — Core Governance

| Tool | Files | Status | Priority |
|------|-------|--------|----------|
| **education/** | 14 | pending | HIGH |
| **research/** | 6 | pending | HIGH |
| **visual/** | 5 | pending | MEDIUM |
| **knowledge/** | 3 | pending | MEDIUM |
| **web/** | 5 | pending | MEDIUM |
| **workstation/** | 2 | pending | LOW |
| **validators/** | 1 | pending | LOW |
| **health/** | 2 | pending | MEDIUM |
| **agent_runtime/** | 7 | pending | HIGH |
| **git/** | 1 | pending | MEDIUM |
| **security/** | 2 | pending | MEDIUM |
| **permissions/** | 3 | pending | MEDIUM |
| **memory/** | 1 | pending | MEDIUM |

### NOT In Scope (Deployed Systems)

| System | Reason |
|--------|--------|
| `web/backend/` | Deployed to ovav.dev, separate deployment pipeline |

## Migration Status Matrix

```
Migration Progress: 30% (gov surface) / 98% (product surface)
Remaining Python LOC: ~150,000
Estimated completion: TBD
```

## Excluded from Migration (Legacy/Test)

| File | Reason |
|------|--------|
| `root: subprocess_test.py` | Test file, candidates for deletion |
| `root: test_bash.py` | Test file, candidates for deletion |

## Acceptance Criteria

1. All tools/ Python migrated to Go equivalents in go-runtime/internal/
2. web/backend/ remains as Python (deployed separately)
3. Test files in root removed or moved to tests/
4. Migration documented in caps.yaml changelog

## Related Documents

- `.ovav/plan/caps.yaml` v102.3 — migration status
- `go-runtime/internal/` — target for Go migrations
- `.github/workflows/` — CI pipeline for migration validation

## Changelog

- 2026-08-08: Issue created, 52 files identified for migration
