---
name: ovav-worktree-route
description: Route commits between branches (cherry-pick, hotfix, patch, emergency).
---

# ovav-worktree-route

Transfers commits between branches with explicit routing modes.

## Modes

| Mode | Use case |
|---|---|
| `cherry-pick` | Selective commit transfer (default) |
| `patch` | All commits from source to target |
| `hotfix` | Main + develop simultaneously |
| `emergency` | Bypass policies with CEO waiver |

## Workflow

```bash
# Single commit
ovav worktree owx <target-branch> cherry-pick <commit-sha>

# Multiple commits
ovav worktree owx <target-branch> cherry-pick <sha1> <sha2> <sha3>

# Hotfix (main + develop)
ovav worktree owx main hotfix <commit-sha>

# Emergency (requires waiver)
ovav worktree owx develop emergency <commit-sha>
```

## Rules

- Use `owx <target> <mode> <commits>` — there is NO `owcp` alias (use `owx <target> cherry-pick`)
- Abort in-progress: `ovav worktree owa`
- Detect conflicts before commit
- Track routing in audit log

## Expected output

```
Cherry-pick <sha> from <source> to <target>
✓ No conflicts
✓ Audit entry recorded
```

## Failure modes

- "conflict in <file>" → resolve, then `git cherry-pick --continue`
- "marker file present" → `ovav worktree owa` to abort
- "denied by policy" → use `emergency` mode with CEO waiver
