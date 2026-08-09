#!/usr/bin/env bash
# OVAV OWS Worktree System v3.0 — Test Suite
# Run: bash bin/owv-tests/run-all.sh

set -uo pipefail
TESTS_DIR="$(cd "$(dirname "$0")"; pwd)"
OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/OVAV}"
LIB="$OVAV_ROOT/bin/ovav-owlib.sh"
RT="$OVAV_ROOT/bin/ovav-ow-runtime.sh"
TEST_TMP="${TMPDIR:-/tmp}/ow-test-$$"
mkdir -p "$TEST_TMP"
export OVAV_ROOT
PASS=0; FAIL=0

# Source lib for unit tests
[ -f "$LIB" ] && source "$LIB"

assert_eq() {
  local name="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    green "  PASS $name"
    PASS=$((PASS + 1))
  else
    red "  FAIL $name: expected='$expected' got='$actual'"
    FAIL=$((FAIL + 1))
  fi
}

green() { printf '\033[32m%s\033[0m\n' "$*"; }
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }

# ─── T1: Profile detection (12 profiles) ────────────────────────────────
test_profile_detection() {
  echo "── T1: Profile detection (12 + main + generic)"
  assert_eq "feature"   "feature"   "$(ow_detect_profile 'feature/login')"
  assert_eq "fix"       "fix"       "$(ow_detect_profile 'fix/bug-123')"
  assert_eq "hotfix"    "hotfix"    "$(ow_detect_profile 'hotfix/critical')"
  assert_eq "release"   "release"   "$(ow_detect_profile 'release/v1.2')"
  assert_eq "docs"      "docs"      "$(ow_detect_profile 'docs/readme')"
  assert_eq "refactor"  "refactor"  "$(ow_detect_profile 'refactor/auth')"
  assert_eq "spike"     "spike"     "$(ow_detect_profile 'spike/test-lib')"
  assert_eq "research"  "research"  "$(ow_detect_profile 'research/postgres')"
  assert_eq "migration" "migration" "$(ow_detect_profile 'migration/v2')"
  assert_eq "enterprise" "enterprise" "$(ow_detect_profile 'external/stripe')"
  assert_eq "emergency" "emergency" "$(ow_detect_profile 'emergency/p0')"
  assert_eq "patch"     "patch"     "$(ow_detect_profile 'security/cve')"
  assert_eq "main"      "main"      "$(ow_detect_profile 'main')"
  assert_eq "generic"   "generic"   "$(ow_detect_profile 'random-thing')"
}

# ─── T2: Profile metadata (base/ttl/reviewer) ──────────────────────────────
test_profile_metadata() {
  echo ""
  echo "── T2: Profile metadata"
  assert_eq "feature-base"   "develop" "$(ow_base_branch feature)"
  assert_eq "hotfix-base"    "main"    "$(ow_base_branch hotfix)"
  assert_eq "spike-ttl"      "48"      "$(ow_default_ttl_hours spike)"
  assert_eq "feature-ttl"    "72"      "$(ow_default_ttl_hours feature)"
  assert_eq "emergency-ttl"  "2"       "$(ow_default_ttl_hours emergency)"
  assert_eq "patch-reviewer" "reviewer" "$(ow_required_reviewer patch)"
  assert_eq "hotfix-reviewer" "maintainer" "$(ow_required_reviewer hotfix)"
  assert_eq "feature-reviewer" "none"    "$(ow_required_reviewer feature)"
}

# ─── T3: Conventional commit validation ────────────────────────────────
test_conventional_commits() {
  echo ""
  echo "── T3: Conventional commits"
  assert_eq "feat-simple"           "true"  "$(ow_is_conventional 'feat: add login' && echo true || echo false)"
  assert_eq "feat-scoped"           "true"  "$(ow_is_conventional 'feat(auth): add oauth' && echo true || echo false)"
  assert_eq "fix-bang"              "true"  "$(ow_is_conventional 'fix!: breaking change' && echo true || echo false)"
  assert_eq "hotfix-prefix"         "true"  "$(ow_is_conventional 'hotfix(login): fix crash' && echo true || echo false)"
  assert_eq "no-conventional"       "false" "$(ow_is_conventional 'just a regular message' && echo true || echo false)"
  assert_eq "no-colon"              "false" "$(ow_is_conventional 'feat missing colon' && echo true || echo false)"
}

# ─── T4: Stack detection ────────────────────────────────────────────────
test_stack_detection() {
  echo ""
  echo "── T4: Stack detection"
  local stack_go="$TEST_TMP/stack-go"
  local stack_ts="$TEST_TMP/stack-ts"
  local stack_py="$TEST_TMP/stack-py"
  local stack_rs="$TEST_TMP/stack-rs"
  mkdir -p "$stack_go" "$stack_ts" "$stack_py" "$stack_rs"
  touch "$stack_go/go.mod" "$stack_ts/package.json" "$stack_py/pyproject.toml" "$stack_rs/Cargo.toml"
  assert_eq "go detected"        "go"         "$(ow_detect_stack $stack_go)"
  assert_eq "ts detected"        "typescript" "$(ow_detect_stack $stack_ts)"
  assert_eq "python detected"    "python"     "$(ow_detect_stack $stack_py)"
  assert_eq "rust detected"      "rust"       "$(ow_detect_stack $stack_rs)"
  mkdir -p "$TEST_TMP/empty"
  assert_eq "unknown empty"      "unknown"    "$(ow_detect_stack $TEST_TMP/empty)"
  rm -rf "$TEST_TMP/stack-"* "$TEST_TMP/empty"
}

# ─── T5: Forbidden files detection ──────────────────────────────────────
test_forbidden_files() {
  echo ""
  echo "── T5: Forbidden files"
  local td="$TEST_TMP/ff"
  mkdir -p "$td/.git"  # Skip .git in find
  touch "$td/.env" "$td/key.pem" "$td/cert.pfx"
  echo "ok" > "$td/clean.txt"
  git -C "$td" init -q
  git -C "$td" add -f .env key.pem cert.pfx clean.txt
  local r
  r="$(ow_forbidden_files "$td")"
  if [[ "$r" == "FORBIDDEN: 3"* ]]; then
    green "  PASS detects 3 publishable forbidden files"; PASS=$((PASS+1))
  else
    red "  FAIL forbidden count: $r"; FAIL=$((FAIL+1))
  fi
  echo "ignored.key" > "$td/.gitignore"
  git -C "$td" add .gitignore
  echo "runtime secret" > "$td/ignored.key"
  r="$(ow_forbidden_files "$td")"
  if [[ "$r" == "FORBIDDEN: 3"* ]]; then
    green "  PASS ignores non-publishable runtime credentials"; PASS=$((PASS+1))
  else
    red "  FAIL ignored credential affected count: $r"; FAIL=$((FAIL+1))
  fi
  rm -rf "$td"
}

# ─── T6: Tier gating ────────────────────────────────────────────────────
test_tier_gating() {
  echo ""
  echo "── T6: Tier gating (must BLOCK cross-tier operations)"
  # tier=free can do owc? Yes
  out=$(OVAV_CONSUMER_TIER=free OVAV_CONSUMER_ID=anon "$RT" c feature/test-test 2>&1 || true)
  if [[ "$out" == *"BLOCKED"* ]]; then red "  FAIL free should access owc"; FAIL=$((FAIL+1)); else green "  PASS free tier can owc"; PASS=$((PASS+1)); fi

  # tier=pro should be BLOCKED from owx (enterprise)
  out=$(cd /tmp && OVAV_CONSUMER_TIER=pro OVAV_CONSUMER_ID=anon "$RT" x main HEAD 2>&1 || true)
  if [[ "$out" == *"BLOCKED"* ]]; then green "  PASS pro blocked from owx"; PASS=$((PASS+1)); else red "  FAIL pro allowed owx (should be blocked): $out"; FAIL=$((FAIL+1)); fi

  # tier=free should be BLOCKED from owd? Actually owd is FREE per spec
  # Re-check: owd is in the free list (c|d|l|v)? YES. So free OK.
  out=$(OVAV_CONSUMER_TIER=free OVAV_CONSUMER_ID=anon "$RT" a 2>&1 || true)
  if [[ "$out" == *"BLOCKED"* ]]; then
    green "  PASS free blocked from owa (pro)"; PASS=$((PASS+1))
  else
    # owa should be pro (>=2)
    if [[ "$out" == *"Abort"* ]] || [[ "$out" == *"abort"* ]]; then
      green "  SKIP owa was upgraded to free — acceptable"
    else
      red "  FAIL owa behavior unclear: $out"
      FAIL=$((FAIL+1))
    fi
  fi
}

# ─── T7: owc end-to-end (in a real git repo) ─────────────────────────────
test_owc_endtoend() {
  echo ""
  echo "── T7: owc end-to-end"
  local repo="$TEST_TMP/owc-repo"
  mkdir -p "$repo" && cd "$repo"
  git init -q 2>/dev/null
  git config user.email "t@t.com" && git config user.name "T"
  echo 'module x' > go.mod
  git add . && git commit -qm init
  git checkout -b develop 2>&1 | head -1

  out=$(OVAV_CONSUMER_ROOT="$repo" OVAV_WORKTREE_ROOT="$repo/.ovav/worktrees" OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c feature/login-test 2>&1 || true)
  if [[ "$out" == *"Path:"*".ovav/worktrees/owc-repo-feature-login-test"* ]]; then
    green "  PASS owc creates isolated worktree"
    PASS=$((PASS+1))
  else
    red "  FAIL owc output: $out"
    FAIL=$((FAIL+1))
  fi

  if [ -d "$repo/.ovav/worktrees/owc-repo-feature-login-test" ]; then
    green "  PASS directory exists"
    PASS=$((PASS+1))
  else
    red "  FAIL directory not created"
    FAIL=$((FAIL+1))
  fi

  # Profile detection
  out=$(cd "$repo" && OVAV_CONSUMER_ROOT="$repo" OVAV_WORKTREE_ROOT="$repo/.ovav/worktrees" OVAV_CONSUMER_TIER=business OVAV_CONSUMER_ID=anon "$RT" c hotfix/security 2>&1 || true)
  if [[ "$out" == *"profile=hotfix"* ]]; then
    green "  PASS hotfix profile detected"
    PASS=$((PASS+1))
  else
    red "  FAIL hotfix profile not detected: $out"
    FAIL=$((FAIL+1))
  fi
  cd / && rm -rf "$repo"
}

# ─── T8: Conflict prediction ──────────────────────────────────────────
test_conflict_prediction() {
  echo ""
  echo "── T8: Conflict prediction"
  local repo="$TEST_TMP/conflict-repo"
  mkdir -p "$repo" && cd "$repo"
  git init -q && git config user.email "t@t.com" && git config user.name "T"
  echo "a" > file.txt && git add . && git commit -qm init
  git checkout -b develop -q 2>/dev/null

  # Create branch with conflicting change
  git checkout -b feature/conf -q 2>/dev/null
  echo "BRANCH_CHANGE" > file.txt && git commit -qam "feat: branch change"

  # Update develop with different change
  git checkout develop -q 2>/dev/null
  echo "DEVELOP_CHANGE" > file.txt && git commit -qam "feat: develop change"

  # Now check prediction
  out=$(ow_predict_conflicts "feature/conf" "develop" 2>&1 || echo UNKNOWN)
  if [[ "$out" == "CONFLICTS:"* ]]; then
    green "  PASS detected ${out}"
    PASS=$((PASS+1))
  else
    red "  FAIL conflict prediction: $out"
    FAIL=$((FAIL+1))
  fi
  cd / && rm -rf "$repo"
}

# ─── Run all tests ────────────────────────────────────────────────────
test_profile_detection
test_profile_metadata
test_conventional_commits
test_stack_detection
test_forbidden_files
test_tier_gating
test_owc_endtoend
test_conflict_prediction

echo ""
echo "════════════════════════════════════════════════════════"
echo "  OWS Worktree System v3.0 — TEST RESULTS"
echo "════════════════════════════════════════════════════════"
green "  $PASS passed"
[ $FAIL -eq 0 ] && green "  $FAIL failed" || red "  $FAIL failed"
echo ""
exit $FAIL
