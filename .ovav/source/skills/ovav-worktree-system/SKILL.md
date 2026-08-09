---
name: ovav-worktree-system
description: OVAV Worktree System (OWS) — full worktree lifecycle with governance. Use when creating, listing, verifying, syncing, moving, locking, rescuing, or finalizing worktrees. Triggers: "worktree", "owc", "owd", "owl", "owv", "branch isolation".
---

# OVAV Worktree System (OWS)

Wraps 11 shell abbreviations (`owc`, `owd`, `owl`, `owv`, `ows`, `owclean`, `owm`, `owx`, `owa`, `owr`, `owlk`) with governance.

## When to use

- Any worktree operation (create, list, finalize, verify, sync, etc.)
- Want F0–F5 validators before merge
- Need audit trail of worktree ops
- Multi-agent coordination on same repo

## Available Commands

- `/ovav-owc <name>` — Create
- `/ovav-owd` — Finalize + publish
- `/ovav-owl` — List + conflicts
- `/ovav-owv` — Verify (no merge)
- `/ovav-ows` — Sync + prune
- `/ovav-owclean` — Cleanup
- `/ovav-owm <path>` — Move
- `/ovav-owx` — Cross-branch
- `/ovav-owa` — Abort
- `/ovav-owr` — Rescue
- `/ovav-owlk` — Lock

See `OVS_USER_GUIDE.md` for full details.
