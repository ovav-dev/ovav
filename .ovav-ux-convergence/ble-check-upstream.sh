#!/bin/bash
# Phase 2 - ble.sh CANARY install
# 1. Identify latest STABLE release tag from upstream
# 2. Clone to canary location (NOT ~/.local/share/blesh)
# 3. Build and install to canary prefix
# 4. Validate binary
# 5. Report version

set -e

echo "=== STEP 1: Check upstream ==="
echo "Network connectivity test..."
curl -fsSL --max-time 10 https://api.github.com/repos/akinomyoga/ble.sh/releases/latest 2>&1 | head -20 || echo "FAIL: cannot reach GitHub"

echo ""
echo "=== STEP 2: List recent stable tags ==="
git ls-remote --tags --sort=-v:refname https://github.com/akinomyoga/ble.sh.git 2>&1 | head -15

echo ""
echo "=== STEP 3: Build tools available ==="
for t in gcc make git autoconf; do
  command -v "$t" >/dev/null && echo "OK: $t → $($t --version 2>&1 | head -1)" || echo "MISSING: $t"
done