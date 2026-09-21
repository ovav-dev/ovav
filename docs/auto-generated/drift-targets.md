# Drift Detection Targets

> **Auto-generated** from `cmd/ovav/drift_targets.go`. DO NOT EDIT MANUALLY.
> Run `ovav docs generate` to refresh.

Per ADR-007 (drift detection). Each target is a (fragment, live) pair
with a comparison function.

| ID | Fragment | Live | Auto-fixable |
|----|----------|------|--------------|
| `it-keybindings` | `workstation/configs/intelligent-terminal/settings-fragment.json` | `/mnt/c/.../LocalState/settings.json` | ✅ |
| `bash-inputrc` | `workstation/configs/inputrc/ovav.inputrc` | `~/.inputrc` | ✅ |
| `runtime-baseline` | `.ovav/integrity_backups/baseline.json` | (file hashes) | ✅ |
| `pinned-baseline` | `.ovav/integrity_backups/baseline.pinned.json` | (pinned vs current) | ⚠️ CEO approval |
| `tool-configs` | `.ovav/registry/tool_configs.yaml` | `bin/ovav` | ❌ rebuild required |
