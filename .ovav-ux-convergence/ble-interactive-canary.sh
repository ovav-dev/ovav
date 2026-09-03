#!/bin/bash
# PHASE 3 - ble.sh INTERACTIVE CANARY via tmux
# Spawn an interactive bash with ble.sh, type commands, capture screen, verify ghost suggestion.
# Validates: ghost appearance, Right-Arrow accept, Ctrl+R passthrough, Starship intact, Atuin pty-proxy intact.

set -e

WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"
BLERC="$DEST/blerc.template"
SESSION=ovav-ble-canary
LOG="$DEST/CANARY-INTERACTIVE.log"

# Kill any previous session
tmux kill-session -t $SESSION 2>/dev/null || true

# Create tmux session with a custom bashrc that sources ble.sh + blerc inline
# We do NOT modify the user's ~/.bashrc yet — this is isolated.
cat > /tmp/opencode/canary-bashrc.sh <<'EOF'
# Isolated bashrc for canary — does NOT affect user shell
export TERM=xterm-256color
export COLORTERM=truecolor
export STARSHIP_CONFIG=/home/braka/Systems/ovav/workstation/configs/starship/starship.toml

# Pre-seed history with a known command so ghost suggestion has something to match
history -s "pnpm dev --host 0.0.0.0"
history -s "git status"
history -s "ls -la /tmp"

# 1. Atuin pty-proxy (must come BEFORE atuin init)
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

# 5. ble.sh — CANARY: source it BEFORE starship
# Use [[ $- == *i* ]] check to ensure interactive
if [[ $- == *i* ]] && [ -f ~/.local/share/blesh/ble.sh ]; then
  source ~/.local/share/blesh/ble.sh
  source /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template
fi

# 6. Starship (LAST so it owns PS1)
if command -v starship >/dev/null 2>&1; then
  eval "$(starship init bash 2>/dev/null)"
fi

# PS1 reset by starship — bash shows starship prompt
PS1='$(starship_precmd)'
EOF

# Spawn tmux session
tmux new-session -d -s $SESSION -x 200 -y 50 "bash --rcfile /tmp/opencode/canary-bashrc.sh -i"
sleep 3  # Let bash init fully

# Capture initial screen
echo "=== INITIAL SCREEN ===" > "$LOG"
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Test 1: Send "pnpm" and wait, capture
echo "" >> "$LOG"
echo "=== TEST 1: Type 'pnpm' (should ghost-suggest 'pnpm dev --host 0.0.0.0') ===" >> "$LOG"
tmux send-keys -t $SESSION "pnpm"
sleep 1
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Test 2: Press Right Arrow to accept suggestion
echo "" >> "$LOG"
echo "=== TEST 2: Press Right Arrow (should accept ghost) ===" >> "$LOG"
tmux send-keys -t $SESSION 'Right'
sleep 0.5
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Test 3: Enter to execute (and verify it runs)
echo "" >> "$LOG"
echo "=== TEST 3: Press Enter (suggestion accepted) ===" >> "$LOG"
tmux send-keys -t $SESSION 'Enter'
sleep 1
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Test 4: Press Ctrl+R (should be Atuin, not ble.sh)
echo "" >> "$LOG"
echo "=== TEST 4: Press Ctrl+R (should be Atuin history search, NOT ble.sh) ===" >> "$LOG"
tmux send-keys -t $SESSION 'C-r'
sleep 1
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1
tmux send-keys -t $SESSION 'Escape'
sleep 0.3

# Test 5: Send a NEW command to verify shell still works
echo "" >> "$LOG"
echo "=== TEST 5: Run fresh command (pwd, ls) ===" >> "$LOG"
tmux send-keys -t $SESSION "pwd"
sleep 0.3
tmux send-keys -t $SESSION 'Enter'
sleep 0.5
tmux send-keys -t $SESSION "echo CANARY-OK-\$(date +%s)"
sleep 0.3
tmux send-keys -t $SESSION 'Enter'
sleep 0.5
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Test 6: Atuin pty-proxy check via Can you pty?
echo "" >> "$LOG"
echo "=== TEST 6: Atuin status (verify pty-proxy alive) ===" >> "$LOG"
tmux send-keys -t $SESSION "atuin search pnpm"
sleep 0.3
tmux send-keys -t $SESSION 'Enter'
sleep 1
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1
tmux send-keys -t $SESSION 'q'
sleep 0.3

# Test 7: Verify Starship prompt rendered
echo "" >> "$LOG"
echo "=== TEST 7: Starship prompt visible ===" >> "$LOG"
tmux send-keys -t $SESSION 'Enter'
sleep 0.5
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Capture version info from inside the canary
echo "" >> "$LOG"
echo "=== VERSION INFO FROM INSIDE CANARY ===" >> "$LOG"
tmux send-keys -t $SESSION 'echo "BLE_VERSION=$BLE_VERSION"'
sleep 0.3
tmux send-keys -t $SESSION 'Enter'
sleep 0.5
tmux capture-pane -t $SESSION -p >> "$LOG" 2>&1

# Save and exit
tmux send-keys -t $SESSION 'exit'
sleep 0.3
tmux send-keys -t $SESSION 'Enter'
sleep 0.5

tmux kill-session -t $SESSION 2>/dev/null || true

echo "=== CANARY LOG SAVED: $LOG ==="
wc -l "$LOG"