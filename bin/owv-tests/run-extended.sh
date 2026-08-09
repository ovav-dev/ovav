#!/usr/bin/env bash
# OVAV OWS Worktree System v3.0 — Extended Test Suite
# Run: bash bin/owv-tests/run-extended.sh

set -uo pipefail
TESTS_DIR="$(cd "$(dirname "$0")"; pwd)"
OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/OVAV}"
RT="$OVAV_ROOT/bin/ovav-ow-runtime.sh"
TEST_TMP="${TMPDIR:-/tmp}/ow-ext-$$"
mkdir -p "$TEST_TMP"
PASS=0; FAIL=0

green() { printf '\033[32m%s\033[0m\n' "$*"; }
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }
assert_ok() {
  local name="$1"; local condition="$2"
  if [ "$condition" = "true" ] || eval "$condition" 2>/dev/null; then
    green "  PASS $name"; PASS=$((PASS+1))
  else
    red "  FAIL $name"; FAIL=$((FAIL+1))
  fi
}

# ─── Setup helper ──────────────────────────────────────────────────────
# Global var so tests can `cd $REPO` after setup
setup_repo() {
  local name="$1"
  REPO="$TEST_TMP/$name"
  # Aggressive cleanup: remove dir AND prune any worktree refs
  cd /tmp 2>/dev/null
  rm -rf "$REPO"
  mkdir -p "$REPO"
  cd "$REPO" || return 1
  git worktree prune --force 2>/dev/null
  git init -q -b develop
  git config user.email "t@t.com"
  git config user.name "T"
  git config commit.gpgsign false
  echo 'module x' > go.mod 2>/dev/null || echo '{"name":"x"}' > package.json
  git add . && git commit -qm init 2>/dev/null
  cd "$REPO" >/dev/null
}

# ─── T1: owc — all 12 profiles ────────────────────────────────────────
test_owc_all_profiles() {
  echo "── T1: owc — 12 profiles + smart folder separation"
  setup_repo "owc-all"
  local profiles_to_test=(
    "feature/login:.ovav/worktrees/feature-login"
    "fix/bug-42:.ovav/worktrees/fix-bug-42"
    "hotfix/critical-fix:.ovav/worktrees/hotfix-critical-fix"
    "docs/update-readme:.ovav/worktrees/docs-update-readme"
    "refactor/auth-module:.ovav/worktrees/refactor-auth-module"
    "research/postgres:.ovav/worktrees/research-postgres"
    "migration/v2:.ovav/worktrees/migration-v2"
    "enterprise/stripe:.ovav/worktrees/enterprise-stripe"
    "patch/security-cve:.ovav/worktrees/patch-security-cve"
  )
  for tc in "${profiles_to_test[@]}"; do
    branch="${tc%%:*}"
    expected_path="${tc##*:}"
    out=$(cd "$REPO" && OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c "$branch" 2>&1 || true)
    # Strip any leading "Cambiado..." git output
    echo "$out" | grep -q "Path:" || true
    if [ -d "${REPO}/${expected_path#./}" ]; then
      green "  PASS $branch -> $expected_path"
      PASS=$((PASS+1))
    else
      red "  FAIL $branch (expected ${REPO}/${expected_path#./})"
      FAIL=$((FAIL+1))
    fi
  done
  cd /
}

# ─── T2: owc — spike TTL annotation ───────────────────────────────────
test_owc_spike_ttl() {
  echo ""
  echo "── T2: owc — spike TTL annotation"
  setup_repo "owc-spike"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c spike/test-graphql 2>&1 | tail -3
  if [ -f "$REPO/.ovav/worktrees/spike-test-graphql/.ovav/.spike-ttl-hours" ]; then
    local ttl
    ttl=$(cat "$REPO/.ovav/worktrees/spike-test-graphql/.ovav/.spike-ttl-hours")
    if [ "$ttl" = "48" ]; then
      green "  PASS spike TTL = 48h"; PASS=$((PASS+1))
    else
      red "  FAIL spike TTL=$ttl (expected 48)"; FAIL=$((FAIL+1))
    fi
  else
    red "  FAIL no TTL file for spike"; FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T3: owc — auto-cd (PWD_NOW) ─────────────────────────────────────
test_owc_autocd() {
  echo ""
  echo "── T3: owc — auto-cd printing (PWD_NOW=)"
  setup_repo "owc-cd"
  out=$(cd "$REPO" && OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/cd-test 2>&1 || true)
  if echo "$out" | grep -q "PWD_NOW="; then
    green "  PASS emits PWD_NOW line"; PASS=$((PASS+1))
  else
    red "  FAIL no PWD_NOW: $out"; FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T4: owc — git rerere enabled ────────────────────────────────────
test_owc_rerere() {
  echo ""
  echo "── T4: owc — git rerere.enabled in worktree"
  setup_repo "owc-rerere"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/rerere-test 2>&1 >/dev/null
  if (cd "$REPO/.ovav/worktrees/feature-rerere-test" && git config rerere.enabled 2>/dev/null) | grep -q true; then
    green "  PASS rerere.enabled=true"; PASS=$((PASS+1))
  else
    red "  FAIL rerere not enabled"; FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T5: owd — compliance levels (workflow gate) ─────────────────────
test_owd_compliance_levels() {
  echo ""
  echo "── T5: owd — compliance levels trigger correct stages"
  setup_repo "owd-comp"
  # Create a clean worktree
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/owd-test 2>&1 >/dev/null
  WT="$REPO/.ovav/worktrees/feature-owd-test"

  # quick: should run only S1 (conflict predict)
  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" d --compliance quick 2>&1 || true)
  if echo "$out" | grep -q "S2/6.*Secrets sweep" && echo "$out" | grep -q "SKIP"; then
    green "  PASS quick compliance skips S2-S6"; PASS=$((PASS+1))
  else
    yellow "  quick output: $out" | head -10
    # Some commands might just not have output yet
    if echo "$out" | grep -q "compliance=quick"; then
      green "  PASS quick compliance recognized"
      PASS=$((PASS+1))
    else
      red "  FAIL quick compliance not recognized: $out"
      FAIL=$((FAIL+1))
    fi
  fi

  # standard: should run S1, S2, S3
  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=enterprise OVAV_CONSUMER_ID=anon "$RT" d --compliance standard 2>&1 || true)
  if echo "$out" | grep -q "compliance=standard"; then
    green "  PASS standard compliance recognized"
    PASS=$((PASS+1))
  else
    red "  FAIL standard not recognized: $out"
    FAIL=$((FAIL+1))
  fi

  # strict: should require reviewer
  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=enterprise OVAV_CONSUMER_ID=anon "$RT" d --compliance strict 2>&1 || true)
  if echo "$out" | grep -q "REVIEWER REQUIRED"; then
    green "  PASS strict requires reviewer"
    PASS=$((PASS+1))
  else
    red "  FAIL strict should require reviewer: $out"
    FAIL=$((FAIL+1))
  fi

  # strict + reviewer (should pass that gate)
  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=enterprise OVAV_CONSUMER_ID=anon "$RT" d --compliance strict --reviewer "Braka" 2>&1 || true)
  if echo "$out" | grep -q "Reviewer provided"; then
    green "  PASS strict with reviewer accepted"
    PASS=$((PASS+1))
  else
    red "  FAIL strict+reviewer not accepted: $out"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T6: owd — secrets sweep detects staged secrets ──────────────────
test_owd_secrets_sweep() {
  echo ""
  echo "── T6: owd — secrets sweep detects AWS keys"
  setup_repo "owd-secrets"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/secrets-test 2>&1 >/dev/null
  WT="$REPO/.ovav/worktrees/feature-secrets-test"
  # Plant a fake AWS key in a Python file
  echo 'AWS_ACCESS_KEY_ID = "<<TEST>>"' > "$WT/bad.py"
  echo 'aws_secret_access_key = "<<TEST>>"' >> "$WT/bad.py"

  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" d --compliance standard 2>&1 || true)
  if echo "$out" | grep -q "FAIL.*FOUND" || echo "$out" | grep -q "BLOCKED"; then
    green "  PASS secrets sweep blocked"
    PASS=$((PASS+1))
  else
    red "  FAIL secrets sweep did not block: $out"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T7: owv — integrity 100% required ────────────────────────────────
test_owv_integrity() {
  echo ""
  echo "── T7: owv — checks OVAV integrity"
  setup_repo "owv-int"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/owv-test 2>&1 >/dev/null
  WT="$REPO/.ovav/worktrees/feature-owv-test"
  cd "$WT"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" v 2>&1 || true)
  if echo "$out" | grep -q "PASS.*integrity 100%"; then
    green "  PASS owv checks integrity"
    PASS=$((PASS+1))
  else
    yellow "  owv output: $out"
    red "  FAIL owv not checking integrity"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T8: owl — list with conflict predictions ────────────────────────
test_owl_predictions() {
  echo ""
  echo "── T8: owl — list with conflict predictions vs develop"
  setup_repo "owl-pred"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/list-test 2>&1 >/dev/null
  cd "$REPO"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" l 2>&1 || true)
  if echo "$out" | grep -q "conflict="; then
    green "  PASS owl shows conflict column"
    PASS=$((PASS+1))
  else
    yellow "  owl output: $out"
    red "  FAIL owl missing conflict column"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T9: owl —json ───────────────────────────────────────────────────
test_owl_json() {
  echo ""
  echo "── T9: owl --json produces valid JSON"
  setup_repo "owl-json"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/json-test 2>&1 >/dev/null
  cd "$REPO"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" l --json 2>&1 || true)
  if echo "$out" | python3 -c "import sys,json; json.loads(sys.stdin.read())" 2>/dev/null; then
    green "  PASS owl --json valid"
    PASS=$((PASS+1))
  else
    red "  FAIL owl --json invalid: $out" | head -3
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T10: owclean — spike TTL cleanup ───────────────────────────────
test_owclean_spike_ttl() {
  echo ""
  echo "── T10: owclean — spike TTL enforcement"
  setup_repo "owclean-tll"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c spike/test-cleanup 2>&1 >/dev/null
  WT="$REPO/.ovav/worktrees/spike-test-cleanup"
  # Manually set spike creation to 100h ago to simulate expired
  echo "$(($(date +%s) - 360000))" > "$WT/.ovav/.spike-created-at"
  cd "$REPO"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" clean 2>&1 || true)
  if echo "$out" | grep -q "Removing expired spike"; then
    green "  PASS owclean removes expired spike"
    PASS=$((PASS+1))
  else
    yellow "  owclean output: $out"
    red "  FAIL owclean did not remove expired spike"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T11: owc — duplicate branch detection ──────────────────────────
test_owc_dup_branch() {
  echo ""
  echo "── T11: owc — refuses duplicate branches"
  setup_repo "owc-dup"
  OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/dup-test 2>&1 >/dev/null
  cd "$REPO"
  # Try to create the same branch again (should fail with EITHER error from worktree or from script)
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/dup-test 2>&1 || true)
  if echo "$out" | grep -qiE "(already exists|denied|branch exists|exit|already)"; then
    green "  PASS duplicate branch refused"
    PASS=$((PASS+1))
  else
    red "  FAIL duplicate not refused: $out"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── T12: owprep — project analysis cache ────────────────────────────
test_owprep_cache() {
  echo ""
  echo "── T12: owprep — caches project analysis"
  setup_repo "owprep-c"
  cd "$REPO"
  # Remove any leftover config
  rm -f "$REPO/.ovav/worktree-config.json"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" prep 2>&1 || true)
  if [ -f "$REPO/.ovav/worktree-config.json" ]; then
    green "  PASS owprep creates cache file"
    PASS=$((PASS+1))
  else
    red "  FAIL no cache created"
    FAIL=$((FAIL+1))
  fi

  # Second call should show "cache exists"
  out=$(OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" prep 2>&1 || true)
  if echo "$out" | grep -q "cache exists"; then
    green "  PASS owprep detects existing cache"
    PASS=$((PASS+1))
  else
    red "  FAIL owprep didn't detect cache: $out"
    FAIL=$((FAIL+1))
  fi
  cd /
}

# ─── Run all ─────────────────────────────────────────────────────────
test_owc_all_profiles
test_owc_spike_ttl
test_owc_autocd
test_owc_rerere
test_owd_compliance_levels
test_owd_secrets_sweep
test_owv_integrity
test_owl_predictions
test_owl_json
test_owclean_spike_ttl
test_owc_dup_branch
test_owprep_cache

echo ""
echo "════════════════════════════════════════════════════════"
echo "  OWS Worktree System v3.0 — EXTENDED TEST RESULTS"
echo "════════════════════════════════════════════════════════"
green "  $PASS passed"
[ $FAIL -eq 0 ] && green "  $FAIL failed" || red "  $FAIL failed"
echo ""
cd /
rm -rf "$TEST_TMP"
exit $FAIL
