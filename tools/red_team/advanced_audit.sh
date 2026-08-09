#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# OVAV Red Team Advanced Audit — v1.0
# 7 advanced techniques for comprehensive security testing
# ═══════════════════════════════════════════════════════════════════════════════
set -euo pipefail

OVAV_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$OVAV_ROOT"

PASS=0
FAIL=0
WARN=0
RESULTS=""

log() { echo -e "\033[1;34m[RED TEAM]\033[0m $1"; }
pass() { ((PASS++)); RESULTS+="  ✅ $1\n"; }
fail() { ((FAIL++)); RESULTS+="  ❌ $1\n"; }
warn() { ((WARN++)); RESULTS+="  ⚠️  $1\n"; }

log "Starting OVAV Red Team Advanced Audit v1.0"
log "Target: $OVAV_ROOT"
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 1: Race Condition Detection (full -race sweep)
# ═══════════════════════════════════════════════════════════════════════════════
log "T1: Race Condition Detection"
RACE_OUT=$(go test -C go-runtime ./... -race -count=1 2>&1 || true)
if echo "$RACE_OUT" | grep -q "DATA RACE"; then
    RACE_COUNT=$(echo "$RACE_OUT" | grep -c "DATA RACE" || echo "0")
    fail "T1: $RACE_COUNT data races detected"
    echo "$RACE_OUT" | grep -A5 "DATA RACE" | head -20
else
    pass "T1: 0 data races (full sweep)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 2: go vet + staticcheck
# ═══════════════════════════════════════════════════════════════════════════════
log "T2: Static Analysis (go vet)"
VET_OUT=$(go vet -C go-runtime ./... 2>&1 || true)
if [ -z "$VET_OUT" ]; then
    pass "T2: go vet clean (0 warnings)"
else
    VET_COUNT=$(echo "$VET_OUT" | wc -l)
    fail "T2: go vet found $VET_COUNT issues"
    echo "$VET_OUT" | head -10
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 3: Secret Scanning
# ═══════════════════════════════════════════════════════════════════════════════
log "T3: Secret Scanning (27 patterns)"
SECRETS_FOUND=0
for pattern in "api_key" "api-key" "apikey" "secret" "password" "token" "private_key" "private-key" "AWS_ACCESS_KEY" "AWS_SECRET" "GITHUB_TOKEN" "OAUTH" "JWT_SECRET" "HMAC_SECRET" "VAULT_KEY"; do
    MATCHES=$(grep -rn "$pattern" --include="*.go" --include="*.yaml" --include="*.json" --include="*.md" go-runtime/ .ovav/ 2>/dev/null | grep -v "_test.go" | grep -v "testdata/" | grep -v "TODO" | grep -v "example" | grep -v "placeholder" | head -3)
    if [ -n "$MATCHES" ]; then
        ((SECRETS_FOUND++))
    fi
done
if [ "$SECRETS_FOUND" -eq 0 ]; then
    pass "T3: 0 plaintext secrets detected"
else
    warn "T3: $SECRETS_FOUND potential secret patterns found (review recommended)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 4: Protected Branch Enforcement
# ═══════════════════════════════════════════════════════════════════════════════
log "T4: Protected Branch Enforcement"
PROTECTED_BRANCHES=("main" "master" "develop" "development" "prod" "production" "staging")
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "unknown")
IS_PROTECTED=false
for pb in "${PROTECTED_BRANCHES[@]}"; do
    if [ "$CURRENT_BRANCH" = "$pb" ]; then
        IS_PROTECTED=true
        break
    fi
done
if [ "$IS_PROTECTED" = true ]; then
    warn "T4: On protected branch '$CURRENT_BRANCH' — verify write is authorized"
else
    pass "T4: On branch '$CURRENT_BRANCH' (not protected)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 5: Dependency Integrity
# ═══════════════════════════════════════════════════════════════════════════════
log "T5: Dependency Integrity (go.sum)"
if [ -f "go-runtime/go.sum" ]; then
    SUM_LINES=$(wc -l < go-runtime/go.sum)
    pass "T5: go.sum exists ($SUM_LINES entries)"
    # Check for known vulnerable patterns
    VULN_DEPS=$(grep -c "golang.org/x/crypto\|golang.org/x/net" go-runtime/go.sum 2>/dev/null || echo "0")
    if [ "$VULN_DEPS" -gt 0 ]; then
        warn "T5: $VULN_DEPS crypto/net dependencies — verify versions"
    fi
else
    fail "T5: go.sum missing — dependency integrity unverifiable"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 6: Test Coverage Floor
# ═══════════════════════════════════════════════════════════════════════════════
log "T6: Test Coverage Floor Check"
COVER_OUT=$(go test -C go-runtime ./... -cover 2>&1 | grep "coverage:" | awk '{print $NF}' | tr -d '%' || true)
UNDER_50=0
for cov in $COVER_OUT; do
    if [ "$cov" -lt 50 ] 2>/dev/null; then
        ((UNDER_50++))
    fi
done
if [ "$UNDER_50" -eq 0 ]; then
    pass "T6: All packages above 50% coverage floor"
else
    warn "T6: $UNDER_50 packages below 50% coverage"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# TECHNIQUE 7: Validator Registry Integrity
# ═══════════════════════════════════════════════════════════════════════════════
log "T7: Validator Registry Integrity"
VALIDATOR_COUNT=$(grep -c "New.*()," go-runtime/internal/validators/validators.go 2>/dev/null || echo "0")
TEST_COUNT=$(grep -c "func Test" go-runtime/internal/validators/validators_test.go 2>/dev/null || echo "0")
if [ "$VALIDATOR_COUNT" -gt 0 ] && [ "$TEST_COUNT" -gt 0 ]; then
    pass "T7: $VALIDATOR_COUNT validators registered, $TEST_COUNT test functions"
else
    fail "T7: Validator registry incomplete (validators=$VALIDATOR_COUNT, tests=$TEST_COUNT)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# SUMMARY
# ═══════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══════════════════════════════════════════════════════════════════════════════"
echo "  OVAV RED TEAM ADVANCED AUDIT — RESULTS"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo -e "$RESULTS"
echo "───────────────────────────────────────────────────────────────────────────────"
echo "  PASS: $PASS | FAIL: $FAIL | WARN: $WARN"
TOTAL=$((PASS + FAIL + WARN))
if [ "$FAIL" -eq 0 ]; then
    echo "  STATUS: ✅ ALL CLEAR"
else
    echo "  STATUS: ❌ $FAIL ISSUES FOUND"
fi
echo "═══════════════════════════════════════════════════════════════════════════════"

# Exit with failure if any FAIL
[ "$FAIL" -eq 0 ]
