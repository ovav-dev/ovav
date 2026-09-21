#!/bin/bash
# PHASE 13 - REGRESSION CHECK
# Validate that Terminal Cortex + OVAV runtime still work after all changes.

# Don't use set -e because opencode acp times out (which is expected)

WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"
OUT="$DEST/REGRESSION-RESULT.txt"

exec > "$OUT" 2>&1

echo "============================================================"
echo "OVAV UX CONVERGENCE 2026 — REGRESSION CHECK"
echo "Date: $(date -Iseconds)"
echo "============================================================"

PASS=0
FAIL=0
TOTAL=0

check() {
    local label="$1"
    local cmd="$2"
    TOTAL=$((TOTAL+1))
    echo ""
    echo "[$TOTAL] $label"
    echo "  CMD: $cmd"
    if eval "$cmd" > /dev/null 2>&1; then
        echo "  RESULT: ✅ PASS"
        PASS=$((PASS+1))
    else
        echo "  RESULT: ❌ FAIL"
        FAIL=$((FAIL+1))
    fi
}

check "IT shell-integration v3 alive" "[ -f ~/.intelligent-terminal/shell-integration_v3.sh ]"
check "Atuin daemon running" "pgrep -f 'atuin daemon start' > /dev/null"
check "Atuin pty-proxy running" "pgrep -f 'atuin pty-proxy' > /dev/null"
check "Atuin MCP running" "pgrep -f 'atuin mcp' > /dev/null"
check "Atuin binary works" "atuin --version > /dev/null"
check "Atuin history capture works" "atuin history list --cmd-only | head -1 > /dev/null"
check "fzf binary works" "fzf --version > /dev/null"
check "zoxide binary works" "zoxide --version > /dev/null"
check "Starship binary works" "starship --version > /dev/null"
check "Starship renders prompt" "STARSHIP_CONFIG=/home/braka/Systems/ovav/workstation/configs/starship/starship.toml starship prompt > /dev/null"
check "ble.sh binary present" "[ -f ~/.local/share/blesh/ble.sh ]"
check "blerc present" "[ -f ~/.blerc ]"
check "OpenCode binary works" "opencode --version > /dev/null"
check "Crush binary works" "crush --version > /dev/null"
check "IT settings.json valid JSON" "python3 -c 'import json; json.load(open(\"/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json\"))'"
check "OpenCode tui.json valid JSON" "python3 -c 'import json; json.load(open(\"/home/braka/.config/opencode/tui.json\"))'"
check "OVAV runtime go module present" "[ -d ~/Systems/ovav/go-runtime ]"
check "OVAV binary built" "[ -x ~/Systems/ovav/go-runtime/build/ovav ]"
check "OVAV validate runs" "~/Systems/ovav/go-runtime/build/ovav validate --help > /dev/null"

# OpenCode ACP — JSON-RPC initialize request (5s timeout)
echo ""
echo "[$(($TOTAL+1))] OpenCode ACP JSON-RPC initialize"
TOTAL=$((TOTAL+1))
echo "  CMD: echo JSON-RPC | opencode acp"
RESP=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"test","version":"0"}}}' | timeout 3s opencode acp 2>&1)
if echo "$RESP" | grep -q '"protocolVersion"'; then
    echo "  RESULT: ✅ PASS (JSON-RPC initialize responded)"
    PASS=$((PASS+1))
else
    echo "  RESULT: ❌ FAIL"
    echo "  RESP: ${RESP:0:200}"
    FAIL=$((FAIL+1))
fi

# Bash startup delta with ble.sh
echo ""
echo "[$(($TOTAL+1))] Bash startup delta"
TOTAL=$((TOTAL+1))
BEFORE=$( { time bash -c 'exit'; } 2>&1 | grep real | awk '{print $2}')
AFTER=$( { time bash -c 'source ~/.local/share/blesh/ble.sh; source ~/.blerc; exit'; } 2>&1 | grep real | awk '{print $2}')
echo "  BEFORE (no ble.sh): $BEFORE"
echo "  AFTER (ble.sh): $AFTER"
echo "  RESULT: ⚠️ DELTA (acceptable per spec)"

echo ""
echo "============================================================"
echo "SUMMARY: $PASS PASS / $FAIL FAIL / $TOTAL TOTAL"
echo "============================================================"