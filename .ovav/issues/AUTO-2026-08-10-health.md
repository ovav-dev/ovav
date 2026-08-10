# AUTO-2026-08-10-health.md

## Issue: `ovav health` (doctor) reports failures

**Trace ID**: AUTOTEST-LOOP-2026-08-10
**Command**: `go run -C go-runtime ./cmd/ovav/ health`
**Exit Code**: 1

### Evidence
```
🩺 OVAV Doctor — System Diagnostic
================================================
✅ go-runtime             Go go1.26.0 · linux/amd64 · OVAV Go Runtime v5.0
✅ git-available          git is available
✅ git-repo               Repo: OVAV · Branch: develop · HEAD: d117a54b · dirty
⚠️  git-clean              188 file(s) modified
⚠️  branch-safety          Protected branch 'develop' — WAIVER ACTIVE. Writes allowed.
✅ ovav-root              OVAV root: /home/braka/Systems/OVAV — all directories present
✅ authority-contract     current_authority_contract.yaml intentionally absent — replaced by caps.yaml + git HEAD
✅ permission-authority   Permission authority present
✅ registry               Registry intact · auto_triggers present
✅ waiver                 Waiver file exists but inactive
✅ go-version             Go go1.26.0 — compatible with OVAV Go Runtime
✅ disk-space             Repo directory accessible
================================================
10 passed · 2 warnings · 1 failures
🔴 System has failures — review and fix before critical operations
```

### Failures
1. ❌ **git-remote**: Remote `https://github.com/ovav-dev/ovav.git` is not an authorized OVAV remote
   - Fix: `git remote set-url origin https://github.com/ovav-dev/ovav-systems.git`

### Warnings
1. ⚠️ **git-clean**: 188 file(s) modified (dirty working directory)
2. ⚠️ **branch-safety**: Protected branch 'develop' — WAIVER ACTIVE

### Severity: MEDIUM

### Status: OPEN
