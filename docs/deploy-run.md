# ovav deploy run — usage guide

**ADR-008** — auto-deploy fragments to live state with rollback safety.

## Why this exists

After 4 sessions of drift patches that addressed symptoms not root causes
(ADR-005), the missing piece was **automated deployment**. `ovav deploy run`
replaces the manual "edit fragment → run script → restart IT → test" cycle
with one command.

## Commands

```bash
# Auto-detect drift and deploy everything
ovav deploy run

# Deploy specific target only
ovav deploy run --target=bash-inputrc
ovav deploy run --target=it-keybindings

# Dry-run — show what would happen, no changes
ovav deploy run --dry-run

# Skip pre-flight validators (power users)
ovav deploy run --skip-validate

# Disable rollback on failure (production deploys)
ovav deploy run --no-rollback

# Status / history / rollback
ovav deploy status                # last deploy summary
ovav deploy list                  # all recent deploys
ovav deploy history               # detailed JSON
ovav deploy rollback              # rollback last
ovav deploy rollback --to=<id>    # rollback to specific deploy

# List deploy targets
ovav deploy targets
```

## Pipeline (6 steps)

```
1. Pre-flight validators  → quick gate check
2. Drift detection         → ovav drift show infrastructure (ADR-007)
3. Snapshot live state     → .ovav/registry/snapshots/<deploy-id>/
4. Atomic deploy           → sibling temp + Path.replace (WSL-safe)
5. Verify                  → hash match fragment
6. Audit log               → .ovav/registry/deploy_history.jsonl
```

## Atomic writes (WSL-safe)

Per the cross-FS bug discovered in commit `eb066cd`:
- `mv /tmp/foo /mnt/c/.../settings.json` silently fails
- `cp -f /tmp/foo /mnt/c/.../settings.json` sometimes fails
- ✅ `os.Rename(sibling_temp, live_path)` works reliably

The deploy pipeline writes to a sibling temp file (same FS as live) then
renames atomically. Verified by `verifyDeploy` which reads back and hashes.

## Snapshots & rollback

Every deploy creates `.ovav/registry/snapshots/<deploy-id>/<target-id>.json`:

```json
{
  "target_id": "bash-inputrc",
  "live_path": "/home/braka/.inputrc",
  "content": "...",
  "hash": "abc123...",
  "existed": true
}
```

To rollback:

```bash
# Rollback most recent deploy
ovav deploy rollback

# Rollback to specific deploy (from deploy list)
ovav deploy rollback --to=deploy-20260814T194032-fdf7
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Deploy succeeded (or no drift detected) |
| 1 | Deploy failed (rolled back unless --no-rollback) |
| 2 | Invalid usage (bad flags) |

## CI integration

```bash
# In your CI pipeline:
if ! ovav deploy run --target=bash-inputrc; then
  echo "Deploy failed; check .ovav/registry/deploy_history.jsonl"
  exit 1
fi
```

## References

- ADR-008: deploy pipeline architecture
- ADR-007: drift detection (consumed by deploy)
- ADR-005: anti-drift 2026 plan (D1)
