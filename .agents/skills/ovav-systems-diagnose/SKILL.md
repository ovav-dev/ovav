---
name: ovav-systems-diagnose
description: Diagnose OVAV system issues (login, vault, worktree, runtime).
trigger: doctor, diagnose, troubleshoot, broken
---

# ovav-systems-diagnose

Systematic diagnosis for OVAV issues. Run from inside worktree.

## Workflow

```bash
# 1. Doctor
ovav doctor
ovav doctor --quick  # git + branch only

# 2. Status
ovav status

# 3. Health
ovav health

# 4. Vault health
ovav-vault-secrets health

# 5. Worktree list
ovav worktree owl

# 6. Identity
ovav whoami
```

## Common issues

| Symptom | Root cause | Fix |
|---|---|---|
| `Session expired` after WSL restart | WSL2 machine-id unstable | `--recover-ceo` + new `wslStableMachineID` |
| `Seed does not match` | machine-id changed | `ovav login --recover-ceo` |
| `Identity not recognized` | vault key stale | `ovav login --force` |
| `vault: no .enc files` | vault layout missing | restore from `.ovav/vault/tokens/` |
| `cpanel: ws exhausted` | test timeout | `go test -timeout 60s` |
| `denied: branch protected` | writing to main directly | use worktree, branch from develop |
| `integrity < 100%` | baseline drift | `ovav integrity baseline --write` |
| `validation failed` | git hooks reject | `ovav validate --details` |

## Crash investigation

```bash
# OVAV logs
ls .ovav/runtime/logs/ | tail -5

# Audit log
tail -50 .ovav/runtime/audit.jsonl

# Vault backup
ls .ovav/registry/backups/identity-recovery/

# Worktree diagnostics
ovav worktree --preflight
```

## Rules

- Always run `ovav doctor` first
- Use `--quick` for fast checks
- Read crash logs BEFORE guessing
- Document fixes in `docs/runbooks/`
