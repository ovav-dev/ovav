#!/bin/bash
# Deep debug: run canary with verbose ble.sh loading diagnostics
cat > /tmp/opencode/debug-bashrc.sh <<'EOF'
export TERM=xterm-256color
export COLORTERM=truecolor
export STARSHIP_CONFIG=/home/braka/Systems/ovav/workstation/configs/starship/starship.toml

# DIAGNOSTIC: log rcfile execution
exec 2>/tmp/opencode/canary-rcfile-debug.log
echo "=== RCFILE START $(date +%s.%N) ==="
echo "BASH_VERSION=$BASH_VERSION"
echo "$-=$-"
echo "BASH_SOURCE[0]=${BASH_SOURCE[0]}"
echo ""

history -s "pnpm dev --host 0.0.0.0"
echo "history -s done"

if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin pty-proxy init bash 2>/dev/null)"
  echo "atuin pty-proxy init done"
fi
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin init bash --disable-up-arrow 2>/dev/null)"
  echo "atuin init done"
fi
if command -v zoxide >/dev/null 2>&1; then
  eval "$(zoxide init bash 2>/dev/null)"
  echo "zoxide init done"
fi
if command -v fzf >/dev/null 2>&1; then
  eval "$(fzf --bash 2>/dev/null)"
  echo "fzf init done"
fi

echo ""
echo "=== Loading ble.sh ==="
if [ -f ~/.local/share/blesh/ble.sh ]; then
  source ~/.local/share/blesh/ble.sh 2>&1 | head -3
  echo "ble.sh sourced, BLE_VERSION=[$BLE_VERSION]"
  echo "BLE_BASE=[$BLE_BASE]"
  if [ -f /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template ]; then
    source /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template 2>&1 | head -3
  fi
else
  echo "ble.sh NOT FOUND"
fi

echo ""
echo "=== Loading starship ==="
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init bash 2>/dev/null)"
  echo "starship init done"
fi

echo "=== RCFILE END ==="
exec 2>&1  # restore stderr
EOF
echo "Debug bashrc ready"