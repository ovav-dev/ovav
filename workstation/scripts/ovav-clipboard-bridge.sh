#!/usr/bin/env bash
set -euo pipefail

# xclip-compatible bridge for WSL2. OpenCode prefers xclip when it exists;
# a dynamically-linked xclip with missing libXmu makes Ctrl+V fail silently.
# Keep the xclip CLI contract while using the Windows clipboard transport.

read_clipboard=0
for arg in "$@"; do
  case "$arg" in
    -version|-V)
      printf '%s\n' 'ovav-xclip-bridge 1.0 (WSL2 Windows clipboard)'
      exit 0
      ;;
    -o|--out|--output) read_clipboard=1 ;;
  esac
done

if ((read_clipboard)); then
  exec powershell.exe -NoProfile -NonInteractive -Command 'Get-Clipboard -Raw'
fi

exec powershell.exe -NoProfile -NonInteractive -Command '$input | Set-Clipboard'
