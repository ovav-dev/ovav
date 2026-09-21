#!/bin/bash
# Apply IT settings.json.v2 to actual IT location
# Backup is already done. This is the FINAL apply.

set -e

WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"
SRC="$DEST/IT-settings.json.v2"
TARGET="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"
BACKUP="$DEST/IT-settings.json.applied-backup"

# Verify source JSON is valid
python3 -c "import json; json.load(open('$SRC'))" || { echo "FAIL: invalid JSON"; exit 1; }

# Final backup of current settings (idempotent)
cp "$TARGET" "$BACKUP"
echo "Backup: $BACKUP"

# Show before/after SHA
echo "BEFORE: $(sha256sum "$TARGET" | cut -d' ' -f1)"
echo "AFTER will be: $(sha256sum "$SRC" | cut -d' ' -f1)"

# Apply
cp "$SRC" "$TARGET"
echo "✓ Applied to $TARGET"
ls -la "$TARGET"