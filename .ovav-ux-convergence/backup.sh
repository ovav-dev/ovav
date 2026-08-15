#!/bin/bash
# OVAV UX Convergence 2026 - PHASE 2 BACKUP
# Surgical snapshot of all system configs into workspace tree.
# NO modification. NO overwrite of originals.
set -e

TS=$(date +%Y%m%d-%H%M%S)
WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence/backups/$TS"
mkdir -p "$DEST"

echo "BACKUP_ROOT=$DEST"
echo "$DEST" > "$WT/.ovav-ux-convergence/backups/.latest"

# 1. ~/.bashrc
cp ~/.bashrc "$DEST/bashrc.original" && echo "OK: bashrc" || echo "FAIL: bashrc"

# 2. ~/.config/starship.toml (if exists)
[ -f ~/.config/starship.toml ] && cp ~/.config/starship.toml "$DEST/starship.toml.original" && echo "OK: starship.toml"

# 3. ~/.config/starship/ directory (presets, themes)
[ -d ~/.config/starship ] && cp -r ~/.config/starship "$DEST/starship.dir.original" 2>/dev/null && echo "OK: starship dir"

# 4. ~/.config/atuin/
[ -d ~/.config/atuin ] && cp -r ~/.config/atuin "$DEST/atuin.dir.original" && echo "OK: atuin dir"

# 5. ~/.config/opencode/tui.json
[ -f ~/.config/opencode/tui.json ] && cp ~/.config/opencode/tui.json "$DEST/opencode-tui.json.original" && echo "OK: opencode tui.json"

# 6. ~/.config/opencode/opencode.json
[ -f ~/.config/opencode/opencode.json ] && cp ~/.config/opencode/opencode.json "$DEST/opencode-config.json.original" && echo "OK: opencode opencode.json"

# 7. ~/.config/crush/config.json (if exists)
[ -f ~/.config/crush/config.json ] && cp ~/.config/crush/config.json "$DEST/crush-config.json.original" && echo "OK: crush config"

# 8. ~/.local/share/blesh/ (ble.sh, if exists - shouldn't)
[ -d ~/.local/share/blesh ] && cp -r ~/.local/share/blesh "$DEST/blesh.dir.original" 2>/dev/null && echo "OK: blesh (was installed)"

# 9. ~/.blerc (if exists)
[ -f ~/.blerc ] && cp ~/.blerc "$DEST/blerc.original" && echo "OK: blerc"

# 10. Intelligent Terminal settings.json (Windows side)
# Try to find via WSL path
WT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.WindowsTerminal*"
ls $WT_SETTINGS 2>/dev/null | head -3

# 11. ~/.intelligent-terminal/ (IT shell-integration source)
[ -d ~/.intelligent-terminal ] && cp -r ~/.intelligent-terminal "$DEST/intelligent-terminal.dir.original" 2>/dev/null && echo "OK: intelligent-terminal dir"

# 12. wsl.conf / wsl-distribution.conf
[ -f /etc/wsl.conf ] && cp /etc/wsl.conf "$DEST/wsl.conf.original" && echo "OK: wsl.conf"

# 13. Starship canonical config in OVAV repo
STARSHIP_OVAV="$WT/workstation/configs/starship/starship.toml"
[ -f "$STARSHIP_OVAV" ] && cp "$STARSHIP_OVAV" "$DEST/starship-ovav-repo.toml.original" && echo "OK: starship-ovav-repo"

# 14. Compute SHA256 of each backup
echo ""
echo "=== SHA256 of backups ==="
find "$DEST" -type f -exec sha256sum {} \; | tee "$DEST/SHA256SUMS.txt"

echo ""
echo "=== TREE of backups ==="
ls -lR "$DEST" | head -50
echo ""
echo "BACKUP COMPLETE: $DEST"