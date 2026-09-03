#!/bin/bash
# PHASE 5 - Intelligent Terminal settings discovery
# Find settings.json (Windows side, via WSL mount)

set -e

# 1. Search Windows-side for WT/IT settings.json
echo "=== 1. Search for Windows Terminal / Intelligent Terminal settings ==="
find /mnt/c/Users -maxdepth 8 -path "*Microsoft.WindowsTerminal*" -name "settings.json" 2>/dev/null | head -5
find /mnt/c/Users -maxdepth 8 -path "*IntelligentTerminal*" 2>/dev/null | head -5
find /mnt/c/Users -maxdepth 8 -path "*Intelligent*Terminal*" 2>/dev/null | head -10

# 2. Alternative: check for IT shell-integration file (we already saw this in ~/.intelligent-terminal)
echo ""
echo "=== 2. ~/.intelligent-terminal source ==="
ls -la ~/.intelligent-terminal/ 2>&1
head -30 ~/.intelligent-terminal/shell-integration_v3.sh 2>&1

# 3. Check if there are IT-specific config files in /etc or somewhere
echo ""
echo "=== 3. IT-related files ==="
find / -maxdepth 4 -name "*intelligent*" 2>/dev/null | head -10
find / -maxdepth 4 -name "*IT-*.json" 2>/dev/null | head -5

# 4. Check WSL distributions and Windows Terminal config dirs
echo ""
echo "=== 4. WT/IT config dirs ==="
ls -la /mnt/c/Users/Alexa/AppData/Local/Packages/ 2>&1 | grep -iE 'terminal|intelligent' | head -5

# 5. Check IT in PATH or as a CLI
echo ""
echo "=== 5. IT as CLI ==="
command -v intelligent-terminal 2>&1
command -v it 2>&1
which wt 2>&1

# 6. Check WSLENV and WSL interop for IT signals
echo ""
echo "=== 6. WSL env signals ==="
echo "WSLENV=$WSLENV"
echo "INTELLIGENT_TERMINAL=$INTELLIGENT_TERMINAL"

# 7. Check if there's an IT config in $XDG_CONFIG_HOME or similar
echo ""
echo "=== 7. XDG-style IT config ==="
ls -la ~/.config/it 2>&1
ls -la ~/.config/intelligent-terminal 2>&1
ls -la ~/.it 2>&1