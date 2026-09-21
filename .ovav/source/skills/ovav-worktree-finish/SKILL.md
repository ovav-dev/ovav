---
name: ovav-worktree-finish
description: Merge worktree back to develop, run validation, cleanup.
---

# ovav-worktree-finish

Closes a feature worktree: verify → merge to develop → cleanup.

## When to use

- Phase complete with 100% criteria met
- Safe Stop Report ready
- No blockers pending

## Workflow

```bash
# 1. From inside the worktree
cd .ovav/worktrees/feature-<task-name>

# 2. Run full validation pipeline
ovav worktree owv

# 3. Verify commit messages and status
git log --oneline develop..HEAD
git status --short

# 4. Merge + cleanup
ovav worktree owd
```

## Rules

- `owd` runs full compliance: secrets, forbidden, owv, conflict, hygiene, GPG, reviewer
- If any gate fails → fix issue, do NOT bypass
- Returns HEAD to develop on success
- Removes branch and worktree directory

## Expected output

```
✓ Validate pipeline passed
✓ Merge to develop
✓ Cleanup worktree + branch
✓ Push to origin (if config)
HEAD now at <sha>
```

## Failure modes

- "validation failed" → `ovav validate` for details
- "conflict" → `ovav worktree owa` abort, manual merge
- "branch protected" → ensure target is develop, not main
- "GPG signature missing" → re-sign commits with `git commit --amend --no-edit -S`
