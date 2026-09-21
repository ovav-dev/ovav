# OVAV × Warp Workflows

This directory contains the OVAV workflow manifests for Warp Drive.

## Contents

- `workflows.json` — Manifest with 10 workflows (9 + abort), 3 tab groups, code review gate

## How to apply

**Warp workflows live in SQLite (`warp.sqlite`) and cannot be imported via filesystem.**

### Option A — Warp UI (recommended)

1. Open Warp → **Settings → Drive → Workflows**
2. For each workflow in `workflows.json`:
   - Click **+ New Workflow**
   - Copy `name`, `description`, `command` from manifest
   - Add `inputs` matching the JSON schema
   - Set `working_directory: current`, `shell: fish`
   - Save

### Option B — Run helper script

```powershell
# From WSL
powershell.exe -ExecutionPolicy Bypass -File .ovav/plans/p6-create-warp-workflows.ps1

# Best-effort: writes to %LOCALAPPDATA%\warp\Warp\dev\workflows\
# Warp may not pick this up — restart Warp after running.
```

If script fails (Warp doesn't read from filesystem), fall back to Option A.

## Authority

WARPS workflows call OWS commands — never `git worktree` directly (plan §14).

OWS is the only authority for git worktree lifecycle. Warp is presentation.

## Variables

- `{{task}}` — short feature name (e.g., `feat-minimax-endpoints`)
- `{{profile}}` — one of `feature|refactor|docs|spike|research|migration|enterprise|hotfix|release|patch|emergency`
- `{{commit}}` — git commit SHA

## Dependencies

- Fish 4.x shell (deterministic quoting)
- OWS binary in PATH (`ovav worktree ...`)
- Warp Desktop running

## Code Review gate

Any workflow with `requires_code_review: true` displays a visible badge in Warp UI.
Before `owd` runs, the Code Review must be approved in Warp.

This is enforced at OVAV governance level — Warp cannot bypass it.
