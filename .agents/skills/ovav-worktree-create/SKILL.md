---
name: ovav-worktree-create
description: Create a new isolated worktree via OWS.
trigger: worktree, owc, create branch, isolate work
---

# ovav-worktree-create

Creates a new feature worktree from `develop` using the OVAV Worktree System (OWS).

## When to use

- Starting any multi-step implementation
- Isolating experimental work from main branch
- Working on a phase of the OVAV × WARP 2026 plan

## Workflow

```bash
# 1. Audit current state
ovav worktree owl

# 2. Create worktree from develop
ovav worktree owc <task-name>

# 3. Move into worktree
cd .ovav/worktrees/feature-<task-name>

# 4. Verify
git status --short
ovav worktree --preflight
```

## Rules

- Always create from `develop` (never from `main` or feature branches)
- Never commit directly to `develop` — use worktrees
- Worktree names follow: `feat-<short-name>`, `fix-<short-name>`, `align-<short-name>`
- OWS hooks install automatically on every new worktree

## Expected output

```
🌿 feature/<task-name> · develop · standard
WORKTREE:/home/braka/Systems/ovav/.ovav/worktrees/feature-<task-name>
🤖 OVAV agents: 11 files copied to .mimocode/agents/
🔐 Trust store: initialized worktree HEAD (<sha>)
```

## Failure modes

- "uncommitted changes" → commit, stash, or `--carry-uncommitted`
- "branch already exists" → use `ovav worktree owc <other-name>`
- "develop not found" → `ovav worktree ows` to sync
