#!/bin/bash
# Deep IT discovery
echo "=== IT package dir ==="
ls -la /mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/ 2>&1 | head -30
echo ""
echo "=== Settings locations (siblings of WT) ==="
find /mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe -maxdepth 3 -name "settings.json" -o -name "*.json" 2>/dev/null | head -20
echo ""
echo "=== wtcli help ==="
/mnt/c/Users/Alexa/AppData/Local/Microsoft/WindowsApps/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/wtcli.exe --help 2>&1 | head -40
echo ""
echo "=== wta help ==="
/mnt/c/Users/Alexa/AppData/Local/Microsoft/WindowsApps/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/wta.exe --help 2>&1 | head -20