#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Workstation Rollback
#  Restores from the latest backup directory.
#  Per rule #39: rollback per-layer.
# ─────────────────────────────────────────────────────────────
set -euo pipefail

ALACRITTY_CONFIG="${ALACRITTY_CONFIG:-/mnt/c/Users/Alexa/AppData/Roaming/alacritty/alacritty.toml}"
TMUX_DEST="$HOME/.tmux.conf"
OPENCODE_TUI_DEST="$HOME/.config/opencode/tui.json"
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

# Alacritty
if [ -f "$BACKUP/alacritty.toml.bak" ] && [ -f "$ALACRITTY_CONFIG" ]; then
  cp -p "$BACKUP/alacritty.toml.bak" "$ALACRITTY_CONFIG"
  echo "✓ Alacritty config restored"
fi

# tmux
if [ -f "$BACKUP/tmux.conf.bak" ]; then
  cp -p "$BACKUP/tmux.conf.bak" "$TMUX_DEST"
  echo "✓ ~/.tmux.conf restored"
fi

# OpenCode TUI
if [ -f "$BACKUP/opencode-tui.json.bak" ] && [ -f "$OPENCODE_TUI_DEST" ]; then
  cp -p "$BACKUP/opencode-tui.json.bak" "$OPENCODE_TUI_DEST"
  echo "✓ OpenCode TUI config restored"
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
