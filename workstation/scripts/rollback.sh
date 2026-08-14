#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Workstation Rollback
#  Restores from the latest backup directory.
#  Per rule #39: rollback per-layer.
# ─────────────────────────────────────────────────────────────
set -euo pipefail

INTEL_TERM_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"
WIN_PS_PROFILE="/mnt/c/Users/Alexa/OneDrive/Documentos/PowerShell/Microsoft.PowerShell_profile.ps1"

# Find most recent backup
LATEST=$(ls -1t "$HOME/.ovav-backups/" 2>/dev/null | head -1)
if [ -z "$LATEST" ]; then
  echo "✗ no backups found in ~/.ovav-backups/"
  exit 1
fi

BACKUP="$HOME/.ovav-backups/$LATEST"
echo "▸ Rolling back from: $BACKUP"

# bashrc
if [ -f "$BACKUP/bashrc.bak" ]; then
  cp -p "$BACKUP/bashrc.bak" "$HOME/.bashrc"
  echo "✓ ~/.bashrc restored"
fi

# starship
if [ -f "$BACKUP/starship.toml.bak" ]; then
  cp -p "$BACKUP/starship.toml.bak" "$HOME/.config/starship.toml"
  echo "✓ starship.toml restored"
fi

# atuin
if [ -f "$BACKUP/atuin-config.toml.bak" ]; then
  cp -p "$BACKUP/atuin-config.toml.bak" "$HOME/.config/atuin/config.toml"
  echo "✓ atuin/config.toml restored"
fi

# Intelligent Terminal
if [ -f "$BACKUP/intel-terminal-settings.json.bak" ] && [ -f "$INTEL_TERM_SETTINGS" ]; then
  cp -p "$BACKUP/intel-terminal-settings.json.bak" "$INTEL_TERM_SETTINGS"
  echo "✓ Intelligent Terminal settings.json restored"
fi

# PowerShell profile
if [ -f "$BACKUP/powershell-profile.ps1.bak" ] && [ -f "$WIN_PS_PROFILE" ]; then
  cp -p "$BACKUP/powershell-profile.ps1.bak" "$WIN_PS_PROFILE"
  echo "✓ PowerShell profile restored"
fi

# OpenCode themes — only remove if explicitly requested
# (these are net-new files, safe to leave installed)
echo ""
echo "ℹ OpenCode themes installed under ~/.config/opencode/themes/ are net-new"
echo "  and have no rollback target. Remove manually if desired."