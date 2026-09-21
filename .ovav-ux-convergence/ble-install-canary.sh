#!/bin/bash
# Phase 2 - ble.sh CANARY install (NOT ~/.local/share/blesh yet)
# Target: $OVAV_BLE_CANARY_PREFIX = canary location
# Install to canary → validate → if pass, copy to ~/.local/share/blesh

set -e

CANARY_ROOT=/tmp/opencode/ovav-ble-canary
rm -rf "$CANARY_ROOT"
mkdir -p "$CANARY_ROOT"

# Version pinned: v0.3.4 (latest stable per upstream)
BLE_VERSION="v0.3.4"
BLE_TAG_COMMIT="9da6774f7bc61ed2e354a38d2a57abe4f2847bff"

echo "=== STEP 1: Clone ble.sh $BLE_VERSION ==="
cd "$CANARY_ROOT"
git clone --depth=1 --branch "$BLE_VERSION" https://github.com/akinomyoga/ble.sh.git src 2>&1 | tail -3
cd src
echo "Commit: $(git rev-parse HEAD)"
echo "Tag: $(git describe --tags --exact-match 2>&1)"

echo ""
echo "=== STEP 2: make install prefix=$CANARY_ROOT/install ==="
# ble.sh install with prefix to a writable canary location
PREFIX="$CANARY_ROOT/install"
make install PREFIX="$PREFIX" 2>&1 | tail -10

echo ""
echo "=== STEP 3: Validate binary ==="
BLE_LOAD="$PREFIX/share/blesh/ble.sh"
ls -la "$BLE_LOAD"
echo "File size: $(stat -c%s "$BLE_LOAD") bytes"
echo "Head: $(head -3 "$BLE_LOAD")"

echo ""
echo "=== STEP 4: Save canary info ==="
cat > "$CANARY_ROOT/CANARY-INFO.txt" <<EOF
CANARY_ROOT=$CANARY_ROOT
BLE_VERSION=$BLE_VERSION
BLE_TAG_COMMIT=$BLE_TAG_COMMIT
BLE_LOAD=$BLE_LOAD
PREFIX=$PREFIX
TIMESTAMP=$(date -Iseconds)
EOF

cat "$CANARY_ROOT/CANARY-INFO.txt"

# Move canary info to workspace
cp "$CANARY_ROOT/CANARY-INFO.txt" /home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization/.ovav-ux-convergence/CANARY-INFO.txt