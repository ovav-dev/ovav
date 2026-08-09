# OVAV Rollback

## Backup

Before any install or apply operation, OVAV creates an atomic **backup** snapshot of the affected files and configuration.

## Restore

If verification fails post-apply, the system automatically triggers a **restore** from the most recent backup snapshot.

## Rollback

Full **rollback** is available via:
```bash
python3 tools/cli/ovav_install.py --rollback
```

All rollback operations are snapshot-gated and verify integrity before completion.
