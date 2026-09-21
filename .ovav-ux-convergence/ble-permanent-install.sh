#!/bin/bash
# PHASE 3 - BLE.SH PERMANENT INSTALL
# Surgical merge into ~/.bashrc — add ble.sh source line in correct position.
# Backup was already created in PHASE 1.

set -e

BASHRC=~/.bashrc
WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"

# 1. Install ~/.blerc from template (BLERC template is in worktree)
cp "$DEST/blerc.template" ~/.blerc
echo "[1/4] Installed ~/.blerc"
ls -la ~/.blerc

# 2. Surgical merge into ~/.bashrc
# Insert AFTER fzf --bash line (line 228 in original), BEFORE starship init (line 235)
# Use a clear marker block so future removal is trivial.

MARKER="# >>> OVAV ble.sh ghost suggestion — minimal config >>>"
END_MARKER="# <<< OVAV ble.sh ghost suggestion <<<"

# Build the insertion block
cat > /tmp/opencode/ble-insertion.sh <<EOF
$MARKER
# ble.sh v0.3.4 — MINIMAL config (history-based ghost suggestion only)
# Source: ~/.local/share/blesh/ble.sh
# Config: ~/.blerc
# Atuin keeps Ctrl+R. Starship owns prompt. fzf keeps Ctrl-T/Alt-C.
if [ -f ~/.local/share/blesh/ble.sh ]; then
  source ~/.local/share/blesh/ble.sh
  [ -f ~/.blerc ] && source ~/.blerc
  # Ctrl+Z = undo in edit mode (does NOT affect job control for running processes)
  ble-bind -s 'C-z' 'undo' 2>/dev/null || true
fi
$END_MARKER
EOF

# Use awk to insert the block after the fzf line (which has 'fzf --bash' string)
# Verify insertion point first
FZF_LINE=$(grep -n 'fzf --bash' "$BASHRC" | head -1 | cut -d: -f1)
if [ -z "$FZF_LINE" ]; then
  echo "ERROR: fzf --bash line not found in $BASHRC"
  exit 1
fi
echo "[2/4] Found fzf --bash at line $FZF_LINE"

# Insert AFTER fzf line
INSERT_AT=$((FZF_LINE + 2))  # +1 for next line, +1 for 1-indexed awk
awk -v insert_at="$INSERT_AT" -v block_file=/tmp/opencode/ble-insertion.sh '
  NR == insert_at {
    while ((getline line < block_file) > 0) print line
    close(block_file)
  }
  { print }
' "$BASHRC" > /tmp/opencode/bashrc.new

# Verify the new bashrc
echo ""
echo "[3/4] Verify insertion in new bashrc:"
grep -n "OVAV ble.sh ghost suggestion" /tmp/opencode/bashrc.new | head -5
echo ""

# Diff before/after (visual)
echo "=== Diff stats ==="
diff "$BASHRC" /tmp/opencode/bashrc.new | head -30

# Apply: replace ~/.bashrc with new version (atomic via mv)
mv /tmp/opencode/bashrc.new "$BASHRC"
echo "[4/4] ~/.bashrc updated (atomic mv)"

# Verify
echo ""
echo "=== POST-INSTALL: ~/.bashrc lines 220-260 ==="
sed -n '220,260p' "$BASHRC"

# SHA256
echo ""
echo "=== SHA256 of updated ~/.bashrc ==="
sha256sum "$BASHRC"

# Also save a copy of new bashrc to worktree for traceability
cp "$BASHRC" "$DEST/bashrc.after-ble-install"
echo ""
echo "New bashrc also saved to: $DEST/bashrc.after-ble-install"