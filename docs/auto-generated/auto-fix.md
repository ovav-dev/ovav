# Auto-Fix SAFE_FIX Registry

> **Auto-generated** from `internal/validators/auto_fix_registry.go`. DO NOT EDIT MANUALLY.
> Run `ovav docs generate` to refresh.

Per ADR-011 (auto-remediation). Each entry is a validator that
opt-in to `ovav validate --fix`.

**Total entries**: 3

| Validator ID | Description | Risk |
|--------------|-------------|------|
| `bash_readline_bindings` | Add 'deliberately UNBOUND' marker to ~/.inputrc if missing | low |
| `runtime_integrity_baseline_fresh` | Regenerate baseline.json with current file hashes | low |
| `supply_chain` | Regenerate sbom.json to match current tracked files | low |

## Safety guards

1. Snapshot before any fix
2. Rollback on regression (fix introduces new issues)
3. Max 10 fixes per run
4. No fix on protected files (CEO waiver required)
5. JSONL history with operator + timestamp + outcome
