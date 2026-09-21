---
name: ovav-incident-response
description: OVAV incident response procedure (security, integrity, runtime).
---

# ovav-incident-response

Structured response for OVAV incidents. Escalate via CEO bypass if needed.

## Severity levels

| Level | Description | Response time |
|---|---|---|
| P0 | Token exposure, integrity <50% | Immediate |
| P1 | Login broken, Vault down | < 1 hour |
| P2 | Worktree stuck, validator fail | < 4 hours |
| P3 | Minor drift, deprecated config | < 24 hours |

## Workflow

### P0 — Token exposure

```bash
# 1. Stop all push operations
ovav worktree ows  # sync state

# 2. Rotate exposed token
# Manual: GitHub → Settings → PAT → Revoke + regenerate

# 3. Update remote URL
git remote set-url origin https://<new-token>@github.com/<org>/<repo>.git
# OR: configure SSH
git remote set-url origin git@github.com:<org>/<repo>.git
# OR: use gh auth
gh auth login --with-token

# 4. Notify CEO via HUD
```

### P1 — Login broken

```bash
# 1. Identify root cause
ovav whoami
ovav-vault-secrets health

# 2. If WSL2 machine-id changed:
ovav login --recover-ceo

# 3. If vault key corrupt:
# Restore from backup: .ovav/registry/backups/identity-recovery/
ovav login --force

# 4. Verify
ovav whoami
```

### P2 — Worktree stuck

```bash
# 1. List state
ovav worktree owl

# 2. Diagnose
ovav worktree --preflight

# 3. Recover
ovav worktree owr   # rescue
ovav worktree owa   # abort
ovav worktree owclean  # cleanup
```

## Audit log

All incidents logged to `.ovav/runtime/audit.jsonl`. Format:

```json
{
  "timestamp": "2026-08-19T03:34:56Z",
  "level": "P0|P1|P2|P3",
  "actor": "ceo-alexander",
  "action": "...",
  "outcome": "success|failure"
}
```

## Rules

- Never rotate tokens in transcripts
- Always create backup before destructive ops
- Use `--ceowaiver` ONLY for P0/P1
- Document incident in `docs/incidents/YYYY-MM-DD.md`
