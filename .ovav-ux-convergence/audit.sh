#!/bin/bash
# OVAV UX Convergence 2026 - AUDIT script (workspace path, allowed)
OUT=/tmp/opencode/ovav-ux-audit.txt
exec > "$OUT" 2>&1

echo "=== TIMESTAMP ==="
date -Iseconds

echo ""
echo "=== TOOL VERSIONS ==="
bash --version | head -1
command -v atuin >/dev/null && atuin --version | head -1
command -v fzf >/dev/null && fzf --version | head -1
command -v zoxide >/dev/null && zoxide --version | head -1
command -v starship >/dev/null && starship --version | head -1
command -v opencode >/dev/null && opencode --version | head -1
command -v crush >/dev/null && crush --version | head -1

echo ""
echo "=== ~/.bashrc ==="
cat ~/.bashrc 2>&1 | head -200

echo ""
echo "=== ~/.bashrc (INIT ORDER) ==="
grep -nE 'atuin|fzf|zoxide|starship|ble.sh|blesh|init|eval|source|export.*PATH' ~/.bashrc 2>&1

echo ""
echo "=== ~/.config/atuin/config.toml ==="
cat ~/.config/atuin/config.toml 2>&1

echo ""
echo "=== ~/.config/opencode/tui.json ==="
cat ~/.config/opencode/tui.json 2>&1

echo ""
echo "=== ~/.config/opencode/opencode.json (KEYS) ==="
jq 'keys' ~/.config/opencode/opencode.json 2>&1 | head -20

echo ""
echo "=== ~/.config/crush/config.json ==="
cat ~/.config/crush/config.json 2>&1

echo ""
echo "=== ~/.config/starship.toml ==="
cat ~/.config/starship.toml 2>&1 | head -80

echo ""
echo "=== BLE.SH CHECK ==="
ls -la ~/.local/share/blesh 2>&1 | head -5
ls -la ~/.blerc 2>&1 | head -3

echo ""
echo "=== ATUIN DAEMON ==="
pgrep -af atuin | head -5

echo ""
echo "=== ENV: TERM/IT/MICA ==="
echo "TERM=$TERM"
echo "TERM_PROGRAM=$TERM_PROGRAM"
echo "WT_SESSION=$WT_SESSION"
env | grep -iE 'term|it|wez|mica|wsl' | head -10

echo ""
echo "=== END ==="
ls -la "$OUT"