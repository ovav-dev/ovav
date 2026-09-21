#!/usr/bin/env bash
# Bashrc simple restore — append missing inits to OVAV block
set -euo pipefail

BASHRC="$HOME/.bashrc"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/manual-${TS}"
mkdir -p "$BACKUP"
cp -p "$BASHRC" "$BACKUP/bashrc-pre-restore2.bak"

# Read current content
CONTENT=$(cat "$BASHRC")

# 1. Fix ovv alias: 'ovav validate' → 'ovav doctor'
CONTENT=$(echo "$CONTENT" | sed "s|alias ovv='ovav validate'|alias ovv='ovav doctor'|")

# 2. Insert ZOXIDE init right after the ZOXIDE header (line ~196 in current)
# Pattern: "#  ZOXIDE — navigation\n# ───..." then empty, then "#  FZF"
# Insert zoxide init block before "#  FZF" header
CONTENT=$(awk '
  BEGIN { printed = 0 }
  /^#  FZF — fuzzy primitives/ && printed == 0 {
    print "if command -v zoxide >/dev/null 2>&1; then"
    print "  eval \"$(zoxide init bash 2>/dev/null)\""
    print "fi"
    print ""
    print ""
    printed = 1
  }
  /^#  STARSHIP — premium minimal prompt/ && zf_printed == 0 {
    # Insert fzf init before STARSHIP section
    print "if command -v fzf >/dev/null 2>&1; then"
    print "  eval \"$(fzf --bash 2>/dev/null)\""
    print "fi"
    print ""
    print ""
    zf_printed = 1
  }
  { print }
' <<< "$CONTENT")

echo "$CONTENT" > "$BASHRC"

echo "── restore complete ──"
echo "  Atuin init:    $(grep -c 'atuin init' $BASHRC)"
echo "  Starship init: $(grep -c 'starship init' $BASHRC)"
echo "  fzf --bash:    $(grep -c 'fzf --bash' $BASHRC)"
echo "  zoxide init:   $(grep -c 'zoxide init' $BASHRC)"
echo "  MIMOCODE refs: $(grep -c MIMOCODE $BASHRC || echo 0)"
echo ""
echo "── ZOXIDE / FZF sections ──"
awk '/^#  ZOXIDE/,/^#  STARSHIP/{print}' "$BASHRC"