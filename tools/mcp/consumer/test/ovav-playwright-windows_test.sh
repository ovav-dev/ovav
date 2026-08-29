#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
WRAPPER="$ROOT/tools/mcp/consumer/bin/ovav-playwright-windows"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file=$1 expected=$2
  if ! grep -Fq -- "$expected" "$file"; then
    fail "$file does not contain: $expected"
  fi
}

assert_not_contains() {
  local file=$1 forbidden=$2
  if grep -Fiq -- "$forbidden" "$file"; then
    fail "$file contains forbidden text: $forbidden"
  fi
}

make_fakes() {
  local dir=$1
  mkdir -p "$dir/bin" "$dir/profile"
  printf '61673\n/devtools/browser/test-token\n' >"$dir/profile/DevToolsActivePort"

  cat >"$dir/bin/powershell.exe" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$*" >>"$TEST_LOG/powershell.log"
case " $* " in
  *" -Mode Start "*)
    printf 'PID=4242\nSTART_TICKS=638605440000000000\nPROFILE=C:\\fake\\profile\nTOKEN=test-token\nCHROME_PATH=C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe\n'
    ;;
  *" -Mode Stop "*) ;;
  *) exit 64 ;;
esac
EOF

  cat >"$dir/bin/wslpath" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
case "$1" in
  -w) printf 'C:\\repo\\ovav-playwright-windows.ps1\n' ;;
  -u) printf '%s\n' "$TEST_LOG/profile" ;;
  *) exit 64 ;;
esac
EOF

  cat >"$dir/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$*" >>"$TEST_LOG/curl.log"
printf '{"webSocketDebuggerUrl":"ws://127.0.0.1:61673/devtools/browser/test-token"}\n'
EOF

  cat >"$dir/bin/npx" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$*" >>"$TEST_LOG/npx.log"
IFS= read -r request || exit 65
printf '%s\n' "$request" >"$TEST_LOG/npx.stdin"
exit "${TEST_NPX_EXIT:-0}"
EOF

  chmod +x "$dir/bin/"*
}

run_wrapper() {
  local dir=$1
  printf '{"jsonrpc":"2.0","id":1,"method":"initialize"}\n' | \
    env PATH="$dir/bin:$PATH" \
      TEST_LOG="$dir" \
      TEST_NPX_EXIT="${TEST_NPX_EXIT:-0}" \
      OVAV_POWERSHELL_BIN=powershell.exe \
      OVAV_WSLPATH_BIN=wslpath \
      OVAV_CURL_BIN=curl \
      OVAV_NPX_BIN=npx \
      OVAV_STARTUP_TIMEOUT_SECONDS=1 \
      "$WRAPPER"
}

test_static_safety_contract() {
  [[ -x "$WRAPPER" ]] || fail "wrapper is not executable"

  local forbidden
  for forbidden in 'wsl --shutdown' 'wsl.exe --shutdown' 'taskkill' 'Stop-Process -Name' 'Get-Process -Name'; do
    assert_not_contains "$WRAPPER" "$forbidden"
  done

  assert_contains "$WRAPPER" '@playwright/mcp@0.0.79'
}

test_success_uses_exact_identity() {
  local dir
  dir=$(mktemp -d)
  trap 'rm -rf "$dir"' RETURN
  make_fakes "$dir"

  run_wrapper "$dir"

  assert_contains "$dir/npx.log" '-y @playwright/mcp@0.0.79 --cdp-endpoint http://127.0.0.1:61673'
  assert_contains "$dir/npx.stdin" '"method":"initialize"'
  assert_contains "$dir/powershell.log" '-Mode Stop'
  assert_contains "$dir/powershell.log" '-ChromePid 4242'
  assert_contains "$dir/powershell.log" '-StartTimeTicks 638605440000000000'
  assert_contains "$dir/powershell.log" '-Token test-token'
}

test_mcp_failure_is_propagated_and_cleaned() {
  local dir status
  dir=$(mktemp -d)
  trap 'rm -rf "$dir"' RETURN
  make_fakes "$dir"

  set +e
  TEST_NPX_EXIT=23 run_wrapper "$dir"
  status=$?
  set -e

  [[ $status -eq 23 ]] || fail "expected MCP exit 23, got $status"
  assert_contains "$dir/powershell.log" '-Mode Stop'
  assert_contains "$dir/powershell.log" '-ChromePid 4242'
}

test_cdp_timeout_cleans_only_owned_pid() {
  local dir status
  dir=$(mktemp -d)
  trap 'rm -rf "$dir"' RETURN
  make_fakes "$dir"
  rm -f "$dir/profile/DevToolsActivePort"

  set +e
  run_wrapper "$dir" >/dev/null 2>&1
  status=$?
  set -e

  [[ $status -eq 1 ]] || fail "expected startup timeout exit 1, got $status"
  assert_contains "$dir/powershell.log" '-Mode Stop'
  assert_contains "$dir/powershell.log" '-ChromePid 4242'
  assert_not_contains "$dir/powershell.log" '9999'
}

test_static_safety_contract
test_success_uses_exact_identity
test_mcp_failure_is_propagated_and_cleaned
test_cdp_timeout_cleans_only_owned_pid
printf 'PASS: ovav-playwright-windows\n'
