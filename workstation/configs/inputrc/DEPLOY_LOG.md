# Bash Readline Config Deploy Log

This file tracks when the `workstation/configs/inputrc/ovav.inputrc` was
deployed to `~/.inputrc`. Updated by the deploy pipeline (manual or
automated) to provide an audit trail.

## Deployments

| Date (UTC) | Operator | Live Path | Fragment SHA (HEAD) | Result |
|------------|----------|-----------|---------------------|--------|
| 2026-08-14T13:08:00Z | thavren (initial fix) | `/home/braka/.inputrc` | `b536521` | success — 89-line inputrc deployed, 0 existing inputrc to preserve |

## How this file is updated

Each successful run of the inputrc deploy step (via `workstation/scripts/install.sh`
step 2b) appends a new row above. The deploy is idempotent — re-running on a
system with the OVAV inputrc already present is a no-op.

## Why this file matters

Prior to this log, there was no audit trail of when (or whether) the
shift+arrow bindings in `workstation/configs/inputrc/ovav.inputrc` were
actually applied to the user's `~/.inputrc`. This led to CEO's
persistent "shift+arrow → DABC + Windows beep" regression even AFTER the
IT keybindings fragment was fixed.

The new `bash_readline_bindings` validator (#74) now catches inputrc
regressions on every `ovav validate` run, so future drift is caught
before CEO notices.

## Operator action after deploy

Bash does NOT auto-reload `~/.inputrc` mid-session. After deploy, CEO must:

```bash
# Option 1: Reload in current shell (readline only, doesn't re-source .bashrc)
bind -f ~/.inputrc

# Option 2: Start a new bash session (full reload)
exec bash
```

Or close and reopen the Intelligent Terminal window.

## References

- **Fragment:** `workstation/configs/inputrc/ovav.inputrc`
- **Install step:** `workstation/scripts/install.sh` step 2b
- **Validator #74:** `go-runtime/internal/validators/bash_readline_bindings.go`
- **Related fix (commit `bc1fb2b`):** IT fragment keybindings repair
- **Related fix (commit `fc95bfd`):** IT deploy pipeline
