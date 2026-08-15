#!/bin/bash
# Validate ble.sh installation
set -e

echo "=== ~/.local/share/blesh/ STRUCTURE ==="
find ~/.local/share/blesh -maxdepth 2 -type d 2>&1 | head -10
echo ""
echo "=== ble.sh main entry ==="
ls -la ~/.local/share/blesh/ble.sh 2>&1
ls -la ~/.local/share/blesh/lib/ 2>&1 | head -15
echo ""
echo "=== ble.sh version (via ble.sh self-report) ==="
# Source ble.sh in a subshell to query version
bash -c '
  source ~/.local/share/blesh/ble.sh 2>/dev/null
  ble/util/print "ble.sh version: ${BLE_VERSION}"
  ble/util/print "BLE_LOAD_PATH: ${BLE_BASE}"
' 2>&1 | head -5

echo ""
echo "=== Cache + tmp dirs (created by install) ==="
ls -la ~/.local/share/blesh/cache.d 2>&1 | head -3
ls -la ~/.local/share/blesh/tmp 2>&1 | head -3