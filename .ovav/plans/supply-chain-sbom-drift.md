# Supply Chain SBOM Drift — Known Issue

**Status:** Known drift, requires investigation
**Detected:** 2026-08-19
**Severity:** MEDIUM (2601 HASH_MISMATCH items, 4 worktree warnings)

---

## Symptom

```
$ ovav validate
❌ Supply Chain Integrity  FAIL — 2601 baseline/security issue(s), 4 worktree warning(s)
   └─ baseline_invalid: CONTENT_IDENTITY_MISMATCH: baseline 8456a146... current 83fde65c...
   └─ baseline_invalid: HASH_MISMATCH: .env.production.example
   └─ baseline_invalid: HASH_MISMATCH: .githooks/post-commit
   ... (2601 items total)
```

## Root cause (hypothesis)

`ovav project sync` regenerates many tracked files (opencode.json, projections, etc.).
Each regeneration changes file hashes, causing baseline drift.

The `sbom.Generate()` function recalculates the content identity and saves to
`.ovav/registry/sbom.json`, but:

1. The `git_identity` field stays the same because the SBOM file itself is excluded
   from the identity calculation.
2. The `core_files` map (2490 files) appears to NOT include new files committed
   today (MEMORY.md, ovav-warp-knowledge/SKILL.md, etc.).

This suggests the SBOM baseline is stale and does not reflect current HEAD.

## Reproduction

```bash
# In any worktree:
ovav validate  # Supply Chain FAIL

# Try to regen:
cd go-runtime
go run ./cmd/regen-sbom/  # "SBOM regenerated"
# But sbom.json still has same git_identity
```

## What's been tried (2026-08-19)

1. ✗ `ovav project sync` — doesn't update SBOM
2. ✗ `ovav integrity baseline --write` — updates integrity baseline, NOT SBOM
3. ✗ Custom `cmd/regen-sbom/main.go` using `sbom.Generate()` + `Save()` — regenerates but
   core_files count stays at 2490

## Proposed fix (next sprint)

1. Add CLI command `ovav sbom regenerate` to make this user-friendly
2. Investigate why `core_files` doesn't include new files
3. Document the sync order: `ovav project sync` → `ovav sbom regenerate` → `ovav validate`
4. Add `ovav sbom verify` to surface the drift as actionable

## Workaround (temporary)

Accept Supply Chain FAIL in validate output. Other 65 validators PASS.

---

*Generated 2026-08-19 — Thavren, CRIT-019 + CRIT-004 enforced.*