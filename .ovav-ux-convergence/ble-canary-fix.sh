#!/bin/bash
# Fix: ble.sh source check using BASH_VERSION + file existence (more reliable than $- check)

cat > /tmp/opencode/canary-bashrc.sh <<'EOF'
export TERM=xterm-256color
export COLORTERM=truecolor
export STARSHIP_CONFIG=/home/braka/Systems/ovav/workstation/configs/starship/starship.toml

# Pre-seed history
history -s "pnpm dev --host 0.0.0.0"
history -s "git status"
history -s "ls -la /tmp"

# 1. Atuin pty-proxy
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin pty-proxy init bash 2>/dev/null)"
fi

# 2. Atuin init
if command -v atuin >/dev/null 2>&1; then
  eval "$(atuin init bash --disable-up-arrow 2>/dev/null)"
fi

# 3. zoxide
if command -v zoxide >/dev/null 2>&1; then
  eval "$(zoxide init bash 2>/dev/null)"
fi

# 4. fzf
if command -v fzf >/dev/null 2>&1; then
  eval "$(fzf --bash 2>/dev/null)"
fi

# 5. ble.sh — load unconditionally if file exists.
# ble.sh checks interactivity itself and prints a warning if non-interactive.
# For interactive shells it loads fully.
if [ -f ~/.local/share/blesh/ble.sh ]; then
  source ~/.local/share/blesh/ble.sh
  if [ -f /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template ]; then
    source /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template
  fi
fi

# 6. Starship LAST
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init bash 2>/dev/null)"
fi

# Diagnostic echo
echo "[canary] ble.sh BLE_VERSION=${BLE_VERSION:-NOT_LOADED}" >&2
EOF
echo "Fix applied: removed interactivity check, ble.sh will load if file exists"