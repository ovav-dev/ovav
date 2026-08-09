#!/bin/bash
# OVAV E2E Install Pipeline Test
# Verifies the full install workflow: plan → backup → apply → verify
# SEG-5 | T4 | Clara — QA Engineer
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

OVAV_BIN="${OVAV_BIN:-./go-runtime/build/ovav}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo -e "${CYAN}=== OVAV E2E Install Pipeline ===${NC}"
echo "Repo: $REPO_ROOT"
echo "Binary: $OVAV_BIN"
echo ""

# Check binary exists
if [ ! -f "$OVAV_BIN" ]; then
    echo -e "${RED}❌ Go binary not found: $OVAV_BIN${NC}"
    echo "Run: cd go-runtime && go build -o build/ovav ./cmd/ovav/"
    exit 1
fi

PASS=0
FAIL=0

run_step() {
    local step="$1"
    local cmd="$2"
    echo -e "${CYAN}[$step]${NC} $cmd"
    if eval "$cmd" 2>&1; then
        echo -e "  ${GREEN}✅ PASS${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}❌ FAIL${NC}"
        FAIL=$((FAIL + 1))
    fi
    echo ""
}

cd "$REPO_ROOT"

# Step 1: Plan
run_step "plan" "$OVAV_BIN plan --pack-id default 2>&1 | head -5"

# Step 2: Backup (dry-run)
run_step "backup" "$OVAV_BIN backup --pack-id default 2>&1 | head -5"

# Step 3: Apply (dry-run — safe, no writes)
run_step "apply" "$OVAV_BIN apply --pack-id default 2>&1 | head -5"

# Step 4: Verify
run_step "verify" "$OVAV_BIN verify --pack-id default 2>&1 | head -5"

# Step 5: Status
run_step "status" "$OVAV_BIN status 2>&1 | head -3"

# Summary
echo -e "${CYAN}========================================${NC}"
echo -e "Results: ${GREEN}$PASS PASS${NC}, ${RED}$FAIL FAIL${NC}"
if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✅ E2E Install Pipeline — ALL PASS${NC}"
    exit 0
else
    echo -e "${RED}❌ E2E Install Pipeline — $FAIL FAILURE(S)${NC}"
    exit 1
fi
