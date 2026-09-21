#!/usr/bin/env bash
# Bashrc emergency restore — append missing zoxide/fzf init to OVAV block
set -euo pipefail

BASHRC="$HOME/.bashrc"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/manual-${TS}"
mkdir -p "$BACKUP"
cp -p "$BASHRC" "$BACKUP/bashrc-pre-restore.bak"

# Find OVAV block boundaries
START=$(grep -n "^# ── OVAV WORKSTATION 2026 ──" "$BASHRC" | head -1 | cut -d: -f1)
END=$(grep -n "^# ── END OVAV WORKSTATION ──" "$BASHRC" | head -1 | cut -d: -f1)

# Build awk script that:
# 1. Inside OVAV block, INSERT zoxide init AFTER Atuin block (before "# ZOXIDE" header)
# 2. Inside OVAV block, INSERT fzf init AFTER ZOXIDE section (before "# FZF" header)
# 3. Replace 'ovv validate' with 'ovv doctor'

awk -v start="$START" -v end="$END" '
  BEGIN { in_ovav = 0; ovav_line = 0; fix_zoxide = 0; fix_fzf = 0; fix_ovv = 0 }
  {
    ovav_line++
    # Track OVAV block
    if (ovav_line == start) in_ovav = 1
    if (ovav_line == end) {
      # Process last line
      print
      in_ovav = 0
      next
    }
    # Fix ovv alias to use doctor
    if (in_ovav && /^alias ovv=/) {
      sub(/ovav validate/, "ovav doctor")
    }
    # Insert zoxide init after "# ZOXIDE" header if no init exists
    if (in_ovav && /^#  ZOXIDE — navigation$/) {
      print
      print ""
      # Check if next lines have zoxide init or not... but awk reads one at a time
      # We know the OVAV block in current state is missing zoxide init
      print "if command -v zoxide >/dev/null 2>&1; then"
      print "  eval \"$(zoxide init bash 2>/dev/null)\""
      print "fi"
      print ""
      next
    }
    # Skip the empty line after # ZOXIDE header (we just printed our own)
    if (in_ovav && fix_zoxide == 1 && /^$/) {
      fix_zoxide = 0
      next
    }
    # Insert fzf init after "# FZF" header if no init exists
    if (in_ovav && /^#  FZF — fuzzy primitives/) {
      print
      # Print remaining FZF header lines (Ctrl-R comment block)
      need_fzf_inserted = 1
    }
    if (need_fzf_inserted && in_ovav && /^# ──/) {
      # End of FZF section header, insert BEFORE this
      print ""
      print "if command -v fzf >/dev/null 2>&1; then"
      print "  eval \"$(fzf --bash 2>/dev/null)\""
      print "fi"
      need_fzf_inserted = 0
    }
    if (need_fzf_inserted && in_ovav) {
      print
      next
    }
    if (in_ovav && /^# ──/ && need_fzf_inserted == 0) {
      print
      need_fzf_inserted = 0
    }
    print
  }
' "$BASHRC" > "$BASHRC.new"

mv "$BASHRC.new" "$BASHRC"

echo "── restore complete ──"
echo "  backup: $BACKUP/bashrc-pre-restore.bak"
echo "  Atuin init:    $(grep -c 'atuin init' $BASHRC)"
echo "  Starship init: $(grep -c 'starship init' $BASHRC)"
echo "  fzf --bash:    $(grep -c 'fzf --bash' $BASHRC)"
echo "  zoxide init:   $(grep -c 'zoxide init' $BASHRC)"
echo "  MIMOCODE refs: $(grep -c MIMOCODE $BASHRC || echo 0)"
echo ""
echo "── OVAV block: zoxide + fzf sections ──"
awk '/^#  ZOXIDE/{p=1} /^#  FZF/{p=1; max=8} p && max>0 {print; max--}' "$BASHRC"