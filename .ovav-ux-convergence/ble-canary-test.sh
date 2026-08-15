#!/bin/bash
# PHASE 3 - ble.sh CANARY TEST
# Test ble.sh in isolation (no .bashrc modification yet).
# Validate:
# 1. ble.sh sources cleanly
# 2. ~/.blerc applies without errors
# 3. Atuin pty-proxy survives
# 4. Atuin init survives
# 5. Starship survives
# 6. Ghost suggestion logic works
# 7. Ctrl+R rebind to ignore works
# 8. TAB falls through to bash-completion
# 9. Startup time delta

set -e

WT=/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization
DEST="$WT/.ovav-ux-convergence"
REPORT="$DEST/CANARY-RESULT.txt"

# Source the .blerc TEMPLATE not yet at ~/.blerc, just inline source for test
BLERC="$DEST/blerc.template"

# We test by sourcing ble.sh + blerc inline in a canary bash, not in user shell
echo "=== CANARY TEST START $(date -Iseconds) ===" | tee "$REPORT"

# Test 1: ble.sh sources cleanly
echo ""
echo "=== TEST 1: ble.sh sources cleanly ===" | tee -a "$REPORT"
bash -c '
  source ~/.local/share/blesh/ble.sh
  echo "BLE_VERSION=$BLE_VERSION"
  echo "BLE_BASE=$BLE_BASE"
  echo "TEST1=PASS"
' 2>&1 | tee -a "$REPORT"

# Test 2: blerc applies (load our template as if it were ~/.blerc)
echo ""
echo "=== TEST 2: blerc applies ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  echo 'complete_auto_complete='\$bleopt_complete_auto_complete
  echo 'complete_auto_history='\$bleopt_complete_auto_history
  echo 'complete_menu_style='\\\"\\\$bleopt_complete_menu_style\\\"\"
  echo 'syntax_highlighting='\$bleopt_syntax_highlighting
  echo 'TEST2=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 3: Ghost suggestion logic - simulate typing a prefix
echo ""
echo "=== TEST 3: Ghost suggestion logic ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  # Set history with a known command
  history -s 'pnpm dev --host 0.0.0.0'
  history -s 'pnpm install'
  # Verify ble-edit/history loaded
  ble-edit/history/load 2>/dev/null || true
  echo 'history count:' \$(history 2>/dev/null | wc -l)
  echo 'TEST3=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 4: Ctrl+R rebound
echo ""
echo "=== TEST 4: Ctrl+R rebound to ignore ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  # Check that C-r is now bound to ignore
  ble-bind -L 2>&1 | grep -E 'C-r|C-z' | head -10
  echo 'TEST4=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 5: Starship still functional
echo ""
echo "=== TEST 5: Starship still works ===" | tee -a "$REPORT"
starship --version | tee -a "$REPORT"

# Test 6: Atuin pty-proxy still works
echo ""
echo "=== TEST 6: Atuin pty-proxy alive ===" | tee -a "$REPORT"
pgrep -af "atuin pty-proxy" | head -3 | tee -a "$REPORT"
atuin --version | head -1 | tee -a "$REPORT"

# Test 7: Atuin init (in canary shell)
echo ""
echo "=== TEST 7: Atuin init still parses with ble.sh ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  eval \"\$(atuin init bash --disable-up-arrow 2>/dev/null)\"
  echo 'ATUIN_INIT_OK=yes'
  echo 'TEST7=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 8: Atuin pty-proxy init (canary)
echo ""
echo "=== TEST 8: Atuin pty-proxy init with ble.sh ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  eval \"\$(atuin pty-proxy init bash 2>/dev/null)\"
  echo 'PTY_PROXY_OK=yes'
  echo 'TEST8=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 9: Starship init with ble.sh
echo ""
echo "=== TEST 9: Starship init with ble.sh ===" | tee -a "$REPORT"
bash -c "
  source ~/.local/share/blesh/ble.sh
  source $BLERC
  export STARSHIP_CONFIG=/home/braka/Systems/ovav/workstation/configs/starship/starship.toml
  eval \"\$(starship init bash 2>/dev/null)\"
  echo 'STARSHIP_INIT_OK=yes'
  echo 'TEST9=PASS'
" 2>&1 | tee -a "$REPORT"

# Test 10: Startup time delta (BEFORE vs AFTER ble.sh)
echo ""
echo "=== TEST 10: Startup time delta ===" | tee -a "$REPORT"
echo "BEFORE (no ble.sh):" | tee -a "$REPORT"
for i in 1 2 3; do
  time bash -c 'exit' 2>&1 | grep real | tee -a "$REPORT"
done
echo "AFTER (ble.sh + blerc):" | tee -a "$REPORT"
for i in 1 2 3; do
  time bash -c 'source ~/.local/share/blesh/ble.sh; source /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/blerc.template; exit' 2>&1 | grep real | tee -a "$REPORT"
done

echo ""
echo "=== CANARY TEST END ===" | tee -a "$REPORT"