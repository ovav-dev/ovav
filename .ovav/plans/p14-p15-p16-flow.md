# P14, P15, P16 — Flow Official

## P14 — OWS = worktree authority

```text
Warp = presentation
OWS = authority
```

OWS owns:
- `owc` create
- `owd` done
- `owl` list
- `owv` verify
- `owx` route (cherry-pick, hotfix, etc.)
- `owa` abort
- `owr` rescue
- `owu` update
- `owprep` preflight
- `owclean` cleanup
- `owlk` lock
- `owm` move
- `ows` sync

Warp may NOT:
- Create worktree (denydist rejects `git worktree *`)
- Prune worktrees
- Merge branches
- Move branches

## P15 — Worktree flow

```
owc <task> --profile <wt.profile>
  ↓
isolated worktree
  ↓
(skill) /ovav-verify or /ovav-review
  ↓
implementation
  ↓
owv (full validation)
  ↓
Warp Code Review gate
  ↓
fix
  ↓
owv (re-verify)
  ↓
owd (merge → cleanup → return)
```

For selective integration: `owx <target> cherry-pick <commit>`
For recovery: `owa` or `owr`
For maintenance: `ows` or `owclean --dry-run`

## P16 — Warp Code Review gate

Code Review is part of lifecycle, not just UX:

```
CREATE → IMPLEMENT → VERIFY → REVIEW → INTEGRATE
```

OWS enforces: `owd` requires `requires_code_review: true` before merge.
Warp UI shows visible badge on pending review.

OWS decides WHEN to integrate. Warp provides human review surface.

## Status

✅ P14 + P15 + P16 100% — flow documented.

## Files

- `.ovav/warp/workflows.json` — workflow manifest
- `.opencode/skills/ovav-worktree-*/SKILL.md` — 3 split skills
- `.ovav/integrity_backups/baseline.json` — integrity baseline
- `.ovav/registry/identities.yaml` — identity registry
