#!/bin/bash
# PHASE 6 - OpenCode keymap investigation
# Search OpenCode source/docs for paste, input_undo, cut primitives

set -e

OC_ROOT="/home/braka/.opencode"
echo "=== OpenCode binary location ==="
command -v opencode
echo ""
echo "=== OpenCode help (look for keymap config) ==="
timeout 10 opencode --help 2>&1 | head -30 || true
echo ""
echo "=== OpenCode keymap-related strings ==="
# Look for "keymap", "paste", "input_undo", "cut" in the opencode binary or related files
grep -lirE 'keymap|paste|input_undo|input_redo|input_cut' "$OC_ROOT" 2>/dev/null | grep -v node_modules | head -10
echo ""
echo "=== OpenCode tui.json docs reference ==="
ls -la /home/braka/.config/opencode/tui.json /home/braka/Systems/ovav/.opencode/tui.json 2>&1