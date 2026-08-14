# Bash Readline (`~/.inputrc`) Contract

**Status:** Active (enforced by `bash_readline_bindings` validator, ID=74 in `ovav validate`)
**Fragment path:** `workstation/configs/inputrc/ovav.inputrc`
**Live path:** `~/.inputrc` (per-user, no hardcoded alternative)
**Date:** 2026-08-14

## Why this contract exists

bash 5.x (currently 5.3.9 on Ubuntu 26.04) ships readline with **no default
bindings for shift+arrow** (modifier 2). When the terminal sends `\e[1;2X`
(X = A/B/C/D) for shift+up/down/right/left, readline:

1. Cannot find a binding (not in default `bind -p`)
2. Beeps (which WSL surfaces as the Windows error sound)
3. The final char of the CSI sequence may leak into the input buffer as a
   typed literal — CEO observed "DABC" being typed when pressing
   shift+arrows

This bug was the cause of the persistent "shift+arrow → DABC + Windows
beep" regression CEO reported, AFTER the IT keybindings fragment was
already fixed and deployed.

The `bash_readline_bindings` validator (#74) now enforces this contract on
every `ovav validate` run.

## Rules (enforced by validator)

### Rule 1 — Required: shift+arrow bindings (modifier 2)

```inputrc
"\e[1;2A": "<set-mark + previous-history>"   # shift+up
"\e[1;2B": "<set-mark + next-history>"      # shift+down
"\e[1;2C": "<set-mark + forward-char>"      # shift+right
"\e[1;2D": "<set-mark + backward-char>"     # shift+left
```

These are **required** — failing them means `ovav validate` returns FAIL
and the "shift+arrow types DABC" regression returns.

### Rule 2 (warn) — Recommended: shift+ctrl+arrow bindings (modifier 6)

```inputrc
"\e[1;6C": "<set-mark + forward-word>"       # shift+ctrl+right (word select)
"\e[1;6D": "<set-mark + backward-word>"      # shift+ctrl+left (word select)
```

### Rule 3 (warn) — Recommended: suppress audible bell

```inputrc
set bell-style none
```

Required for WSL — otherwise shift+arrow beeps surface as Windows error
sound.

### Rule 4 (implementation) — Comments don't count

Lines starting with `#` are comments and don't count as bindings. The
validator scans line-by-line and skips comment lines.

## Modifier math (per xterm CSI)

| Modifier | Value | Common use |
|----------|-------|------------|
| 1        | none (plain arrow) | `\e[A`, `\e[B`, `\e[C`, `\e[D` |
| 2        | shift | **NOT bound by default** — OVAV fixes this |
| 3        | alt (meta) | `\e[1;3X` → bash: backward-word/forward-word |
| 4        | shift+alt | `\e[1;4X` → bash: select-to-line-start |
| 5        | ctrl | `\e[1;5X` → bash: backward-word/forward-word |
| 6        | ctrl+shift | **NOT bound by default** — OVAV fixes this |
| 7        | ctrl+alt | rarely used |
| 8        | ctrl+shift+alt | rarely used |

bash 5.x readline ships bindings for modifiers 3 and 5 (alt, ctrl) but
not 2 or 6 (shift, ctrl+shift). The OVAV inputrc closes that gap.

## Deploy pipeline

```
workstation/configs/inputrc/ovav.inputrc  (source-of-truth)
       │
       │ install.sh step 2b
       ▼
   ~/.inputrc  (per-user, no hardcoded alternative)
       │
       │ bash auto-sources on new shell
       ▼
   bash readline  (effective at next prompt)
```

The install step:

1. Backs up existing `~/.inputrc` to `$BACKUP_DIR/inputrc.bak`
2. If `~/.inputrc` doesn't exist → copies OVAV inputrc
3. If `~/.inputrc` exists but lacks OVAV marker → appends OVAV bindings
   (preserves user customizations)
4. If `~/.inputrc` exists AND has OVAV marker → skips (idempotent)

## Reloading `~/.inputrc` mid-session

bash does NOT auto-reload `~/.inputrc` in an active shell. After deploy,
operator must reload manually:

```bash
# Option 1: Reload readline bindings only
bind -f ~/.inputrc

# Option 2: Restart bash entirely
exec bash

# Option 3: Close and reopen Intelligent Terminal
```

## Testing the contract

```bash
# Run only this validator
go run -C go-runtime ./cmd/ovav/ validate bash_readline_bindings

# Run the Go test suite for this validator
go test -C go-runtime ./internal/validators/ -run TestBashReadlineBindings -v

# Live shell verification (after deploy + bind -f)
bind -p | grep -E '1;2[ABCD]'   # should show 4 entries
```

## Why this was missed earlier

The OVAV validator suite had 73 validators passing on 2026-08-14, but
`bash_readline_bindings` did not exist. The IT fragment validator
checked the IT-side of the keybindings but bash's readline side was
unchecked. When IT was correctly configured (no fragment drift) but
bash was missing bindings, the regression surfaced at the shell layer.

**Lesson:** A green validator suite does not mean features work. Every
layer in the keystroke path (terminal → IT → bash → readline) needs
explicit structural validation.

## References

- **Fragment:** `workstation/configs/inputrc/ovav.inputrc`
- **Deploy log:** `workstation/configs/inputrc/DEPLOY_LOG.md`
- **Validator:** `go-runtime/internal/validators/bash_readline_bindings.go`
- **Install step:** `workstation/scripts/install.sh` step 2b
- **Related fixes (commit `bc1fb2b`):** IT fragment keybindings repair
- **Related fixes (commit `fc95bfd`):** IT deploy pipeline
- **Modifier math:** https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_sequences

## Change history

| Date | Change | Commit |
|------|--------|--------|
| 2026-08-14 | Initial inputrc + validator | `b536521`, validator registered in batch 11c |
| 2026-08-14 | Install.sh deploy step | `c02cf79` |
| 2026-08-14 | Initial deployment to `~/.inputrc` | DEPLOY_LOG entry |
