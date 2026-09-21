#!/bin/bash
# Use `script` (util-linux) to force a PTY for the canary
# This is the most reliable way to get a TTY in non-interactive environments

# Reset
tmux kill-session -t script-canary 2>/dev/null

# `script` creates a PTY and records the session
SCRIPT_OUT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/SCRIPT-CANARY.txt

# Spawn script wrapping bash with our rcfile
script -q -c "bash --rcfile /tmp/opencode/debug-bashrc.sh -i" "$SCRIPT_OUT" &
SCRIPT_PID=$!
sleep 4

# script doesn't have a tty we can interact with easily from bash tool.
# Let's just check the output it captured.
kill $SCRIPT_PID 2>/dev/null
wait $SCRIPT_PID 2>/dev/null

ls -la "$SCRIPT_OUT"
echo "---"
echo "=== SCRIPT OUTPUT ==="
head -50 "$SCRIPT_OUT" 2>&1
echo "---"
echo "Total lines:"
wc -l "$SCRIPT_OUT"