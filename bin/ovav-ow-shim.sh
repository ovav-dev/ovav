#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────
# OVAV OWS Shell Wrappers — for REAL auto-cd (eval of PWD_NOW= line)
# These functions intercept calls to ow{c,d,owl,owv,...} in interactive
# shells. They wrap the runtime so that the `cd` actually happens.
#
# Install: source this file in your ~/.bashrc or ~/.zshrc after the OVAV
# symlinks are present in ~/.local/bin.
# ──────────────────────────────────────────────────────────────────────

# Resolve OVAV root (env override, default standard path)
: "${OVAV_ROOT:=/home/braka/Systems/OVAV}"
: "${OVAV_CONSUMER_ROOT:=$PWD}"
: "${OVAV_CONSUMER_TIER:=business}"

ow_eval_pwdnow() {
  # Lines like  "PWD_NOW=/some/path"  → cd there.
  local line
  while IFS= read -r line; do
    if [ "${line%%PWD_NOW=*}" = "" ] && [ -n "$line" ]; then
      # PWD_NOW= stripped from start
      local target="${line#PWD_NOW=}"
      [ -d "$target" ] && cd "$target" || eval "cd \"$target\""
    fi
  done
}

# Generic dispatcher: any unprefixed cmd → bin/ovav-ow-*
# CRITICAL: processes output in current shell (no pipe = no subshell)
# so that cd actually changes the user's working directory.
ow_dispatch() {
  local cmd="$1"; shift
  local out
  out=$(OVAV_CONSUMER_ROOT="$PWD" \
        OVAV_CONSUMER_TIER="$OVAV_CONSUMER_TIER" \
        "$OVAV_ROOT/bin/ovav-ow$cmd" "$@" 2>&1)
  local rc=$?
  local line _cd_target=""
  while IFS= read -r line; do
    case "$line" in
      PWD_NOW=*)
        _cd_target="${line#PWD_NOW=}"
        ;;
      *)
        printf '%s\n' "$line"
        ;;
    esac
  done <<< "$out"
  # Auto-cd AFTER printing (affects current shell, not subshell)
  [ -n "$_cd_target" ] && [ -d "$_cd_target" ] && cd "$_cd_target"
  return $rc
}

# Per-command wrappers
owc()   { ow_dispatch c   "$@"; }
owd()   { ow_dispatch d   "$@"; }
owl()   { ow_dispatch l   "$@"; }
owv()   { ow_dispatch v   "$@"; }
ows()   { ow_dispatch s   "$@"; }
owclean() { ow_dispatch clean "$@"; }
owm()   { ow_dispatch m   "$@"; }
owx()   { ow_dispatch x   "$@"; }
owa()   { ow_dispatch a   "$@"; }
owr()   { ow_dispatch r   "$@"; }
owlk()  { ow_dispatch lk  "$@"; }
owprep()    { ow_dispatch prep    "$@"; }
owsuggest() { ow_dispatch suggest "$@"; }

# Print help when sourced
printf '%s\n' \
  "✓ OVAV OWS Shell Wrappers loaded (from $OVAV_ROOT/bin/ovav-ow-shim.sh)" \
  "" \
  "Available commands (real auto-cd):" \
  "  owc <branch>      create worktree + branch (auto-cd into it)" \
  "  owd [branch|path] finalize + merge + cleanup + auto-cd to develop" \
  "  owl               list worktrees with conflict predictions" \
  "  owl --history     audit trail" \
  "  owl --json        machine-readable" \
  "  owv               verify (no merge)" \
  "  ows [mode]        sync / rebase / full" \
  "  owclean           spike TTL + orphan cleanup" \
  "  owm <src> <dst>   move" \
  "  owx <src> <sha>   cross-branch cherry-pick" \
  "  owa | owr | owlk  abort | rescue | lock" \
  "" \
  "Per-command options (see: bin/ovav-consumer help):" \
  "  Set OVAV_CONSUMER_TIER (free|pro|business|enterprise) for tier gating."
