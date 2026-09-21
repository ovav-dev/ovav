# ovav drift show — usage guide

**ADR-007** — visibility into fragment vs live state.

## What is drift?

OVAV stores config sources-of-truth ("fragments") in the repo. They must
be deployed to "live" system locations to take effect. **Drift** is when
fragment and live no longer match.

Examples:
- `workstation/configs/intelligent-terminal/settings-fragment.json` (fragment)
- `/mnt/c/.../LocalState/settings.json` (live IT settings)

- `workstation/configs/inputrc/ovav.inputrc` (fragment)
- `~/.inputrc` (live bash readline)

## Commands

```bash
# Human-readable table
ovav drift show

# JSON output (for CI/dashboards)
ovav drift show --json

# Markdown (for PR comments)
ovav drift show --md

# Specific target only
ovav drift show it-keybindings
ovav drift show bash-inputrc

# List registered targets
ovav drift targets

# Drift history
ovav drift catalog
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | No drift detected |
| 1 | Drift found (CI-friendly) |

## Targets registered (5)

| ID | Auto-fixable | Notes |
|----|--------------|-------|
| `it-keybindings` | ✅ `ovav deploy run` | IT v0.1.4+ |
| `bash-inputrc` | ✅ `ovav deploy run` | bash readline |
| `runtime-baseline` | ✅ `ovav integrity baseline --write` | file hashes |
| `pinned-baseline` | ⚠️ requires CEO approval | drift firewall |
| `tool-configs` | ❌ requires rebuild | bin/ovav stale |

## Example output

```
OVAV Drift Report — 2026-08-14T19:38:00Z
================================================================================
Targets: 5 total, 0 with drift, 0 items

📦 IT Keybindings [it-keybindings]
   Fragment: workstation/configs/intelligent-terminal/settings-fragment.json
   Live:     /mnt/c/.../LocalState/settings.json
   ✅ No drift

📦 Bash inputrc [bash-inputrc]
   Fragment: workstation/configs/inputrc/ovav.inputrc
   Live:     /home/braka/.inputrc
   ✅ No drift
```

## Path overrides (env vars)

| Target | Env var |
|--------|---------|
| `it-keybindings` | `OVAV_LIVE_IT_SETTINGS` |
| `bash-inputrc` | `OVAV_LIVE_INPUTRC` |
| `tool-configs` | `OVAV_BIN_OVAV` |

Use these in CI/tests to point at different live paths.

## Drift catalog

Each `ovav drift show` run appends to `.ovav/registry/drift_catalog.jsonl`:

```bash
$ ovav drift catalog
Drift catalog (.ovav/registry/drift_catalog.jsonl):

  2026-08-14T19:30:00Z — 5 targets, 0 drifted, 0 items
  2026-08-14T19:35:00Z — 5 targets, 2 drifted, 3 items
  2026-08-14T19:38:00Z — 5 targets, 0 drifted, 0 items
```

Useful for: drift trends, regression detection, automated alerts.

## CI integration

```bash
# In your CI pipeline:
ovav drift show --json > drift.json
if ! ./bin/ovav drift show --json > /dev/null; then
  echo "Drift detected:"
  cat drift.json
  exit 1
fi
```

## References

- ADR-007: drift detection architecture
- ADR-005: anti-drift 2026 plan (D4)
