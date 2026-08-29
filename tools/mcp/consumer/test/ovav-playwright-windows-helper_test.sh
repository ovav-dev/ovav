#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
HELPER="$ROOT/tools/mcp/consumer/bin/ovav-playwright-windows.ps1"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local expected=$1
  grep -Fq -- "$expected" "$HELPER" || fail "helper does not contain: $expected"
}

assert_not_contains() {
  local forbidden=$1
  if grep -Fiq -- "$forbidden" "$HELPER"; then
    fail "helper contains forbidden text: $forbidden"
  fi
}

[[ -f $HELPER ]] || fail 'PowerShell helper is missing'
for forbidden in 'wsl --shutdown' 'wsl.exe --shutdown' 'taskkill' 'Stop-Process -Name' 'Get-Process -Name'; do
  assert_not_contains "$forbidden"
done

assert_contains 'Stop-Process -Id $ChromePid'
assert_contains 'ProcessId = $escapedPid'
assert_contains '.ovav-playwright-owner'
assert_contains '"method":"Browser.close"'
assert_contains '[IO.FileAttributes]::ReparsePoint'
assert_contains '$expectedProfileArgument = "--user-data-dir=`"$OwnedProfile`""'
assert_contains 'A positive Chrome PID, start time, and executable path are required'

if command -v powershell.exe >/dev/null 2>&1 && command -v wslpath >/dev/null 2>&1; then
  PS_FILE_WIN=$(wslpath -w "$HELPER") WSLENV=PS_FILE_WIN powershell.exe \
    -NoLogo -NoProfile -NonInteractive \
    -Command '$tokens=$null; $errors=$null; [void][System.Management.Automation.Language.Parser]::ParseFile($env:PS_FILE_WIN,[ref]$tokens,[ref]$errors); if ($errors.Count -gt 0) { exit 1 }'
fi

printf 'PASS: ovav-playwright-windows-helper\n'
