#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
TEST="$ROOT/tools/mcp/consumer/test/ovav-mcp-windows-manager.Tests.ps1"

if command -v pwsh >/dev/null 2>&1; then
  PWSH=pwsh
elif [[ -x /mnt/c/Users/Alexa/AppData/Local/Microsoft/WindowsApps/pwsh.exe ]] && command -v wslpath >/dev/null 2>&1; then
  PWSH=/mnt/c/Users/Alexa/AppData/Local/Microsoft/WindowsApps/pwsh.exe
  TEST=$(wslpath -w "$TEST")
else
  printf 'BLOCKED: PowerShell 7 (pwsh) is required; Windows PowerShell is not accepted.\n' >&2
  exit 2
fi

exec "$PWSH" -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "$TEST"
