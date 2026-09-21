# P40 — Proceso Operacional Diario

## Inicio de sesión

```text
Warp Desktop (Windows)
  ↓
Warp → New Tab → Ubuntu-26.04 (WSL2 native)
  ↓
Fish 4.x shell (login)
  ↓
OVAV Governor ya activo (auto)
  ↓
mise actua toolchain (Fish hook)
  ↓
ovav worktree owl (status) — opcional
```

## Nueva tarea (CEO)

```text
Ctrl+P → OVAV · Create Worktree
  ↓
Inputs:
  - task: feat-minimax-endpoints
  - profile: wt.feature
  ↓
Workflow: ovav worktree owc feat-minimax-endpoints --profile wt.feature
  ↓
Result: WORKTREE:<path>, auto-cd
```

## Sesión de desarrollo

```bash
# OpenCode primary
ovav worktree owc feat-minimax-endpoints --profile wt.feature
cd .ovav/worktrees/feature-feat-minimax-endpoints

# In OpenCode:
/ovav-verify     # run validation
/ovav-review     # open Code Review gate
```

## Segundo agente (parallel experiments)

```bash
# Crush for second opinion
ovav worktree owc review-minimax --profile wt.spike
cd .ovav/worktrees/feature-review-minimax

# In Crush: parallel experiments
```

## Durante trabajo

- Vertical Tabs (visible status)
- Tab Groups (CORE / AGENTS / DEV / PROTECTED)
- Notifications (manual, CEO trigger)
- Session Navigation (Ctrl+Tab for context switch)

## Finalización

```bash
ovav worktree owv             # validate
# → triggers Warp Code Review (if requires_code_review: true)
ovav worktree owd             # merge → cleanup → return
```

## Recurring patterns

| When | Action |
|---|---|
| Start of day | `ovav status` (verify governor + integrity) |
| Before worktree | `ovav worktree owl` (check state) |
| After commit | `ovav worktree owu` (sync from develop) |
| End of session | `ovav worktree owd` (close feature) |
| Errors | `ovav doctor --quick` (diagnostic) |
| Recovery | `ovav worktree owr` (rescue) |

## Status

✅ P40 100% — process documented.
