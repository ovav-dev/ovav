#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Bashrc Repair — Idempotent surgical cleanup
#  Removes duplicate init (Starship/Atuin/zoxide/fzf)
#  Removes MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS
#  Preserves: system bashrc, IT shell integration, NVM
# ─────────────────────────────────────────────────────────────
set -euo pipefail

BASHRC="$HOME/.bashrc"
BACKUP_DIR="$HOME/.ovav-backups/manual-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

cp -p "$BASHRC" "$BACKUP_DIR/bashrc-pre-repair.bak"

# Strategy:
#   1. Keep system bashrc lines 1-136 (Ubuntu preamble + IT shell integration)
#   2. Drop legacy user tools block (lines 137-208 had Starship/Atuin/zoxide/fzf/MIMOCODE)
#   3. KEEP the OVAV WORKSTATION 2026 block (lines 210+) but clean its contents
#
# We rebuild the file from scratch with one canonical block each.

# Find boundary markers
START_LINE=$(grep -n "^# ── OVAV WORKSTATION 2026 ──" "$BASHRC" | head -1 | cut -d: -f1)
END_LINE=$(grep -n "^# ── END OVAV WORKSTATION ──" "$BASHRC" | head -1 | cut -d: -f1)

if [ -z "$START_LINE" ] || [ -z "$END_LINE" ]; then
  echo "ERROR: OVAV markers not found in $BASHRC" >&2
  exit 1
fi

# Lines before OVAV block (1 to START_LINE-1)
# Lines after OVAV block (END_LINE+1 to end)
PRE=$(sed -n "1,$((START_LINE-1))p" "$BASHRC")
POST=$(sed -n "$((END_LINE+1)),\$p" "$BASHRC")
OVAV_BLOCK=$(sed -n "${START_LINE},${END_LINE}p" "$BASHRC")

# Strip legacy duplicate block from PRE (lines that init Starship/Atuin/zoxide/fzf
# outside our OVAV block)
PRE_CLEAN=$(echo "$PRE" | \
  awk '
    BEGIN { skip = 0 }
    /^# >>> intelligent-terminal shell-integration >>>/ { print; skip = 1; next }
    skip && /^# <<< intelligent-terminal shell-integration <<</ { print; skip = 0; next }
    skip { print; next }
    /^# Starship prompt/ { skip = 1; next }
    skip && /^# Optional aliases:/ { skip = 0; next }
    skip { next }
    /^if command -v starship/ { skip = 1; next }
    skip && /^fi/ && depth == 0 { print; skip = 0; depth = 0; next }
    skip { depth++; next }
    /^# Atuin integration/ { skip = 1; next }
    skip && /^fi$/ { print; skip = 0; next }
    skip { next }
    /^if command -v atuin/ { skip = 1; next }
    skip && /^fi$/ { print; skip = 0; next }
    skip { next }
    /^if command -v zoxide/ { skip = 1; next }
    skip && /^fi$/ { print; skip = 0; next }
    skip { next }
    /^if command -v fzf/ { skip = 1; next }
    skip && /^fi$/ { print; skip = 0; next }
    skip { next }
    /^# OVAV SYSTEMS — MIMOCODE/ { skip = 1; next }
    skip && (/^export MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS/ || /^# / || /^$/) { next }
    skip { print; skip = 0; next }
    { print }
  ')

# Remove MIMOCODE from PRE_CLEAN defensively
PRE_CLEAN=$(echo "$PRE_CLEAN" | grep -v "MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS")

# Clean OVAV_BLOCK: remove MIMOCODE references, dedupe internal
OVAV_CLEAN=$(echo "$OVAV_BLOCK" | grep -v "MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS")

# Write final bashrc
{
  echo "$PRE_CLEAN"
  echo "$OVAV_CLEAN"
} > "$BASHRC.new"

mv "$BASHRC.new" "$BASHRC"

echo "── bashrc repaired ──"
echo "  backup: $BACKUP_DIR/bashrc-pre-repair.bak"
echo "  lines before: $(wc -l < $BACKUP_DIR/bashrc-pre-repair.bak)"
echo "  lines after:  $(wc -l < $BASHRC)"
echo "  Atuin init count:   $(grep -c 'atuin init' $BASHRC)"
echo "  Starship init count:$(grep -c 'starship init' $BASHRC)"
echo "  fzf init count:     $(grep -c 'fzf --bash' $BASHRC)"
echo "  zoxide init count:  $(grep -c 'zoxide init' $BASHRC)"
echo "  MIMOCODE refs:      $(grep -c MIMOCODE $BASHRC || echo 0)"