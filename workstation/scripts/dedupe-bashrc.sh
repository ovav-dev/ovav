#!/usr/bin/env bash
# Bashrc surgical dedupe — run via install.sh wrapper
set -euo pipefail

BASHRC="$HOME/.bashrc"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/manual-${TS}"
mkdir -p "$BACKUP"
cp -p "$BASHRC" "$BACKUP/bashrc-pre-dedupe.bak"

# Use awk to remove the legacy user tools block (lines that init
# Starship/Atuin/zoxide/fzf outside our OVAV block).
# Keep: system bashrc (1-136), PATH helpers (137-165), EDITOR/VISUAL, aliases
# Drop: legacy Starship init, legacy Atuin/zoxide/fzf init, MIMOCODE

awk '
  BEGIN { skip = 0 }
  /^# Starship prompt$/ { skip = 1; next }
  skip && /^fi$/ { skip = 0; next }
  skip { next }
  /^if command -v starship &>\/dev\/null; then$/ { skip = 1; next }
  skip && /^fi$/ { skip = 0; next }
  skip { next }
  /^# Atuin integration is installed/ { skip = 1; next }
  skip && /^fi$/ { skip = 0; next }
  skip { next }
  /^# Optional interactive integrations/ { skip = 1; next }
  skip && (/^if command -v atuin/ || /^fi$/) { if (/^fi$/) skip = 0; next }
  skip { next }
  /^if command -v zoxide/ { skip = 1; next }
  skip && /^fi$/ { skip = 0; next }
  skip { next }
  /^if command -v fzf/ { skip = 1; next }
  skip && /^fi$/ { skip = 0; next }
  skip { next }
  /^# OVAV SYSTEMS — MIMOCODE TOTAL FREEDOM$/ { skip = 1; next }
  /^export MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS=1$/ { skip = 0; next }
  /^MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS/ { next }
  skip { next }
  { print }
' "$BASHRC" > "$BASHRC.new"

# Defensive: remove any lingering MIMOCODE references
sed -i '/MIMOCODE_DANGEROUSLY_SKIP_PERMISSIONS/d' "$BASHRC.new"

mv "$BASHRC.new" "$BASHRC"

# Report
echo "── bashrc dedupe complete ──"
echo "  backup: $BACKUP/bashrc-pre-dedupe.bak"
echo "  line count: $(wc -l < $BASHRC) (was $(wc -l < $BACKUP/bashrc-pre-dedupe.bak))"
echo "  Atuin init:    $(grep -c 'atuin init' $BASHRC)"
echo "  Starship init: $(grep -c 'starship init' $BASHRC)"
echo "  fzf --bash:    $(grep -c 'fzf --bash' $BASHRC)"
echo "  zoxide init:   $(grep -c 'zoxide init' $BASHRC)"
echo "  MIMOCODE refs: $(grep -c MIMOCODE $BASHRC || true)"