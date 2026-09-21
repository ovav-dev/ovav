# ADR-008: Auto-Deploy Pipeline

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 1 anti-drift core), ADR-007 (drift detection)
**Decider:** Thavren + CEO

## Context

After 4 sessions of patches that addressed symptoms not root causes (ADR-005),
the missing piece is **automated deployment** of fragments to live state.
Currently every fix requires:
1. Edit fragment in repo
2. Run deploy script manually (often forgotten or fails silently)
3. Restart IT manually
4. Test if it worked

This is the entire "patch cycle" that keeps repeating. The fix is to make
deployment automatic and atomic.

## Decision

Add `ovav deploy run` command that:
1. Detects drift via `drift show` infrastructure (ADR-007)
2. Snapshots live state for rollback (atomic)
3. Deploys each drifted target (parallel where possible)
4. Verifies deploy via hash comparison
5. Optionally triggers IT reload (ADR-009, Phase 1 / D5)
6. Logs to `.ovav/registry/deploy_history.jsonl`

### Architecture

```
ovav deploy run [--target=X] [--dry-run] [--skip-validate] [--no-rollback]
       │
       ▼
[1. Pre-flight: validators gate]
       │
       ▼
[2. Drift detect (ovav drift show --json)]
       │
       ▼
[3. Snapshot live state to .ovav/registry/snapshots/<deploy-id>/]
       │
       ▼
[4. For each drifted target (parallel):
       - Atomic write (temp + rename, WSL-safe)
       - Verify (hash match fragment)
       - If fail → rollback this target
   ]
       │
       ▼
[5. Post-deploy verify (drift show again, should be 0)]
       │
       ▼
[6. Append deploy_history.jsonl with: id, target, status, duration]
```

### Snapshot format

```json
{
  "deploy_id": "deploy-2026-08-14T19-38-00-abc123",
  "timestamp": "2026-08-14T19:38:00Z",
  "operator": "thavren",
  "targets": [
    {
      "id": "it-keybindings",
      "live_path": "/mnt/c/.../settings.json",
      "live_snapshot": "/home/braka/.ovav/snapshots/deploy-.../it-keybindings.json",
      "status": "success",
      "duration_ms": 123
    }
  ],
  "status": "success | partial | failed",
  "duration_ms": 456
}
```

### Atomic write (WSL-safe)

Per the cross-FS bug discovered in commit eb066cd:
- `mv /tmp/foo /mnt/c/.../settings.json` silently fails
- `cp -f /tmp/foo /mnt/c/.../settings.json` silently fails (sometimes)
- **Only `python open(LIVE, 'w')` or same-FS `Path.replace()` work reliably**

The deploy pipeline uses:
1. Compute new content in fragment
2. Write to a sibling temp file (same FS as live)
3. `Path.replace(temp, live)` — atomic, cross-FS safe
4. Verify by reading live back and hashing

### Subcommands

| Command | Purpose |
|---------|---------|
| `ovav deploy run` | Auto-detect drift and deploy |
| `ovav deploy run --target=X` | Deploy only target X |
| `ovav deploy run --dry-run` | Show what would happen |
| `ovav deploy status` | Show last deploy status |
| `ovav deploy rollback` | Rollback to last snapshot |
| `ovav deploy rollback --to=<id>` | Rollback to specific snapshot |
| `ovav deploy list` | List recent deploys |
| `ovav deploy history` | Show deploy history |

### Initial deploy targets

1. `it-keybindings` — uses python atomic write
2. `bash-inputrc` — uses python atomic write
3. `runtime-baseline` — uses ovav integrity baseline --write

## Consequences

### Positive

- **One command fixes drift** — `ovav deploy run` → all drift fixed
- **Rollback-safe** — every deploy has a snapshot
- **Audit trail** — deploy_history.jsonl captures all deploys
- **CI-friendly** — exit codes, JSON output, dry-run
- **Composable** — `ovav deploy run --from-drift-report=path.json`

### Negative

- **Privileges** — some targets (IT) may need filesystem permissions
- **State on remote machines** — IT lives on Windows; WSL mount needed
- **Rollback complexity** — if many targets deployed, rollback is N operations
- **Validation overhead** — pre-flight validators add 1-2 seconds

### Mitigations

- Skip-validate flag (`--skip-validate`) for power users
- Per-target rollback (not all-or-nothing)
- `--no-rollback` for production deploys where rollback is unsafe

## References

- ADR-005 Phase 1 anti-drift (D1)
- ADR-007 drift detection (consumed by deploy)
- ADR-009 IT reload integration (D5, separate)
- `workstation/scripts/deploy-it-keybindings.sh` (existing, will be wrapped)
- `workstation/scripts/_deploy-write-live.py` (existing helper)
