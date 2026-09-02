#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV E2E Test — Rule #36
#  SHELL CONTEXT → AI → ACTION → RESULT → HISTORY
# ─────────────────────────────────────────────────────────────
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
PASS=0
FAIL=0

pass() { printf "\033[1;32m✓\033[0m %s\n" "$*"; PASS=$((PASS+1)); }
fail() { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; FAIL=$((FAIL+1)); }
info() { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }

echo "═══════════════════════════════════════════════════════════"
echo "  OVAV END-TO-END TEST (Rule #36)"
echo "═══════════════════════════════════════════════════════════"

# 1. Open OVAV workspace
info "1. Workspace reachable"
cd "$OVAV_ROOT" && pwd > /dev/null && pass "cwd = $(pwd)"

# 2. Confirm cwd
info "2. CWD confirmation"
if pwd | grep -q "$OVAV_ROOT"; then
  pass "in OVAV project"
else
  fail "not in OVAV project"
fi

# 3. Execute successful command
info "3. Successful command"
if bash -c 'echo "OVAV-TEST-OK"' > /dev/null 2>&1; then
  pass "successful command exit 0"
else
  fail "successful command failed"
fi

# 4. Execute failing command
info "4. Failing command"
if bash -c 'ls /nonexistent-ovav-test' > /dev/null 2>&1; then
  fail "failing command returned 0"
else
  pass "failing command exit != 0 (detection possible)"
fi

# 5. Active terminal stack
info "5. Active terminal stack"
if [ -n "${TMUX:-}" ] && tmux -V >/dev/null 2>&1; then
  pass "tmux active ($(tmux -V))"
else
  fail "tmux is not active"
fi

# 6. Atuin history DB
info "6. Atuin history"
if command -v atuin >/dev/null 2>&1; then
  ATUIN_HIST_COUNT=$(atuin search --limit 100 --format ts '"OVAV"' 2>/dev/null | wc -l || echo 0)
  pass "atuin available (history commands: $ATUIN_HIST_COUNT)"
else
  fail "atuin not installed"
fi

# 6b. tmux workspace routes
info "6b. tmux workspace routes"
if [ -n "${TMUX:-}" ]; then
  for route in \
    "home|/home/braka" \
    "ovav|/home/braka/Systems/ovav" \
    "akrynt|/home/braka/Systems/projects/work/akrynt-agent"; do
    route_name="${route%%|*}"
    route_path="${route#*|}"
    if tmux select-window -t "main:=${route_name}" 2>/dev/null && \
       [ "$(tmux display-message -p '#{window_name}|#{pane_current_path}')" = "$route" ]; then
      pass "Alt route $route_name → $route_path"
    else
      fail "tmux route mismatch: $route"
    fi
  done
  tmux select-window -t 'main:=ovav' 2>/dev/null || true
  for binding in \
    'M-1.*select-window -t =home' \
    'M-2.*select-window -t =ovav' \
    'M-3.*select-window -t =akrynt'; do
    if tmux list-keys -T root | grep -Eq "$binding"; then
      pass "binding present: ${binding%%.*}"
    else
      fail "binding missing: $binding"
    fi
  done
  if bash "$OVAV_ROOT/workstation/tests/test-tmux-routes.sh" >/dev/null; then
    pass "raw Alt bytes route through tmux correctly"
  else
    fail "raw Alt bytes do not route through tmux correctly"
  fi
else
  fail "cannot test tmux routes outside tmux"
fi

# 7. fzf
info "7. fzf"
if command -v fzf >/dev/null 2>&1; then
  FZF_VERSION=$(fzf --version | head -1 | awk '{print $1}')
  pass "fzf $FZF_VERSION"
else
  fail "fzf not installed"
fi

# 8. zoxide
info "8. zoxide"
if command -v zoxide >/dev/null 2>&1; then
  pass "zoxide available"
else
  fail "zoxide not installed"
fi

# 9. Starship
info "9. Starship prompt"
if command -v starship >/dev/null 2>&1; then
  STARSHIP_OUTPUT=$(starship prompt 2>/dev/null || true)
  if [ -n "$STARSHIP_OUTPUT" ]; then
    pass "starship renders prompt ($(echo "$STARSHIP_OUTPUT" | wc -c) chars)"
  else
    fail "starship empty output"
  fi
else
  fail "starship not installed"
fi

# 10. OVAV runtime
info "10. OVAV runtime"
if command -v ovav >/dev/null 2>&1; then
  OVAV_VERSION=$(ovav status 2>&1 | grep -oE "OVAV [0-9.]+" | head -1 || echo "unknown")
  pass "OVAV runtime: $OVAV_VERSION"
else
  fail "OVAV CLI not in PATH"
fi

# 11. OpenCode
info "11. OpenCode ACP backend"
if [ -x "$HOME/.opencode/bin/opencode" ]; then
  OC_VERSION=$("$HOME/.opencode/bin/opencode" --version 2>&1 | head -1)
  pass "opencode $OC_VERSION"
else
  fail "opencode not installed"
fi

# 12. OpenCode themes
info "12. OpenCode OVAV themes"
[ -f "$HOME/.config/opencode/themes/ovav-night.json" ] && pass "ovav-night theme" || fail "ovav-night missing"
[ -f "$HOME/.config/opencode/themes/ovav-day.json" ] && pass "ovav-day theme" || fail "ovav-day missing"

# 13. Alacritty host bridge
info "13. Alacritty host bridge"
ALACRITTY_CONFIG="${ALACRITTY_CONFIG:-/mnt/c/Users/Alexa/AppData/Roaming/alacritty/alacritty.toml}"
if [ -f "$ALACRITTY_CONFIG" ] && grep -qF 'chars = "\u001b[13;2u"' "$ALACRITTY_CONFIG" 2>/dev/null; then
  pass "Alacritty Shift+Enter CSI-u bridge present"
else
  fail "Alacritty Shift+Enter bridge missing — run install.sh"
fi

# 14. PowerShell profile
info "14. PowerShell PSReadLine"
PS_PROFILE="/mnt/c/Users/Alexa/OneDrive/Documentos/PowerShell/Microsoft.PowerShell_profile.ps1"
if [ -f "$PS_PROFILE" ]; then
  if grep -q "OVAV WORKSTATION" "$PS_PROFILE" 2>/dev/null; then
    pass "PowerShell profile has OVAV block"
  else
    fail "PowerShell profile missing OVAV block"
  fi
else
  warn "PowerShell profile not at expected path (may be in different OneDrive sync)"
fi

# 15. PATH sanity
info "15. PATH sanity"
case ":$PATH:" in
  *":/usr/local/bin:"*) pass "/usr/local/bin in PATH" ;;
  *) fail "/usr/local/bin not in PATH" ;;
esac
case ":$PATH:" in
  *":$HOME/.opencode/bin:"*) pass "opencode bin in PATH" ;;
  *) warn "opencode bin NOT in PATH (use full path)" ;;
esac

# ─── Summary ───────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════"
printf "  \033[1;32mPASS: %d\033[0m   \033[1;31mFAIL: %d\033[0m\n" "$PASS" "$FAIL"
echo "═══════════════════════════════════════════════════════════"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
