#!/usr/bin/env bash
set -euo pipefail

# OpenCode's resume path replays the complete transcript in the mini TUI.
# On a large SQLite history this can look frozen and starve the input loop.
# Resume through mini mode with replay disabled; ordinary commands are passed
# through unchanged. Set OPENCODE_RESUME_AUTOFIX=0 for an upstream comparison.

if [[ "${OPENCODE_RESUME_AUTOFIX:-1}" == "0" ]]; then
  exec "$(dirname "$0")/opencode.bin" "$@"
fi

command_name=""
case "${1:-}" in
  attach|run|session|mcp|completion|debug|providers|agent|upgrade|uninstall|serve|web|models|stats|export|import|github|pr|db|plugin)
    command_name="$1"
    ;;
esac

eligible=0
case "$command_name" in
  ""|attach) eligible=1 ;;
esac

resume=0
mini=0
no_replay=0
explicit_replay=0
for arg in "$@"; do
  case "$arg" in
    -s|--session|--session=*) resume=1 ;;
    -c|--continue) resume=1 ;;
    --mini) mini=1 ;;
    --no-replay) no_replay=1 ;;
    --replay) explicit_replay=1 ;;
  esac
done

if ((eligible && resume)); then
  extra=()
  if (( !mini )); then
    extra+=(--mini)
  fi
  if (( !no_replay && !explicit_replay )); then
    extra+=(--no-replay)
  fi
  exec "$(dirname "$0")/opencode.bin" "${extra[@]}" "$@"
fi

exec "$(dirname "$0")/opencode.bin" "$@"
