# OVAV Rollback Guide

## Principle

Every governed apply/install/global action must have a rollback path before execution.

## Source-local rollback

For normal repo-local work:

```fish
git status --short
git diff --stat
```

If changes are not committed and should be discarded, review exact files first. Do not use broad destructive commands without explicit decision.

## Commit rollback

To inspect recent changes:

```fish
git log --oneline -5
git show --stat HEAD
```

To revert a committed change safely, prefer a normal revert over history rewriting unless you explicitly control the branch:

```fish
git revert <commit>
```

## Global/config rollback

Global/config rollback requires the original backup artifact. OVAV must not claim rollback safety if no backup exists.

Required rollback evidence:

| Required item | Description |
|---|---|
| Backup path | Where original files were copied |
| Restore command | Exact command to restore |
| Verification | Command proving restored state |
| Risk | What may not be restored |

## Launch rule

If an operation cannot be backed up and rolled back, it cannot be part of the default launch flow.
