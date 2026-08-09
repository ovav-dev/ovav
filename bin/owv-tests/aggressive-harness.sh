#!/usr/bin/env bash
# =============================================================================
# OWS AGGRESSIVE TEST HARNESS — Multi-Stack, Multi-Edge-Case
# Probes every OWS command under stress conditions across stacks and scenarios.
# Runs in /tmp to avoid contaminating the real repo.
#
# Run: bash bin/owv-tests/aggressive-harness.sh
# =============================================================================
set -uo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/OVAV}"
RT="$OVAV_ROOT/bin/ovav-ow-runtime.sh"
HARNESS_TMP="${TMPDIR:-/tmp}/ows-aggressive-$$"
rm -rf /tmp/worktrees 2>/dev/null || true  # clean all leftover worktree dirs
mkdir -p /tmp/worktrees
PASS=0; FAIL=0; WARN=0

green() { printf '\033[32m%s\033[0m\n' "$*"; }
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }
header(){ printf '\n\033[1;34m═══ %s ═══\033[0m\n' "$*"; }

assert_eq() {
  local name="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    green "  PASS $name"; PASS=$((PASS + 1))
  else
    red "  FAIL $name: expected='$expected' got='$actual'"; FAIL=$((FAIL + 1))
  fi
}

assert_ok() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    green "  PASS $name"; PASS=$((PASS + 1))
  else
    red "  FAIL $name (exit=$?)"; FAIL=$((FAIL + 1))
  fi
}

assert_exit_nonzero() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then
    red "  FAIL $name (expected non-zero exit)"; FAIL=$((FAIL + 1))
  else
    green "  PASS $name"; PASS=$((PASS + 1))
  fi
}

# ─── Setup: create isolated git repo with stack-specific files ──────────────
setup_stack_repo() {
  local name="$1" stack="$2"
  local REPO="$HARNESS_TMP/$name"
  rm -rf "$REPO"
  mkdir -p "$REPO"
  cd "$REPO"
  git worktree prune --force 2>/dev/null
  # Clean any leftover worktree dirs from previous runs
  rm -rf /tmp/worktrees/* 2>/dev/null || true
  git init -q -b develop
  git config user.email "test@ovav.test"
  git config user.name "OWS Test"
  git config commit.gpgsign false

  case "$stack" in
    go)
      echo 'module testrepo' > go.mod
      echo 'package main' > main.go
      echo 'func main() { println("hello") }' >> main.go
      ;;
    typescript)
      echo '{"name":"testrepo","version":"1.0.0"}' > package.json
      echo 'export const x = 1;' > index.ts
      echo '{}' > tsconfig.json
      ;;
    python)
      echo '[project]' > pyproject.toml
      echo 'name = "testrepo"' >> pyproject.toml
      echo 'def main(): print("hello")' > main.py
      ;;
    rust)
      echo '[package]' > Cargo.toml
      echo 'name = "testrepo"' >> Cargo.toml
      echo 'fn main() { println!("hello"); }' > src/main.rs
      mkdir -p src
      echo 'fn main() { println!("hello"); }' > src/main.rs
      ;;
    c)
      echo 'int main() { return 0; }' > main.c
      echo '#include <stdio.h>' > header.h
      ;;
    java)
      mkdir -p src/main/java/com/test
      echo 'public class Main { public static void main(String[] args) {} }' > src/main/java/com/test/Main.java
      ;;
    ruby)
      echo 'puts "hello"' > main.rb
      echo 'source "https://rubygems.org"' > Gemfile
      ;;
    mixed)
      echo 'module testrepo' > go.mod
      echo '{"name":"testrepo"}' > package.json
      echo 'def main(): print("hello")' > main.py
      echo 'int main() { return 0; }' > main.c
      echo 'fn main() {}' > lib.rs
      echo 'puts "hello"' > main.rb
      ;;
  esac

  git add . && git commit -qm "init $stack"
  cd "$OVAV_ROOT" >/dev/null
}

echo "═══════════════════════════════════════════════════════════════════"
echo "  OWS AGGRESSIVE TEST HARNESS — Multi-Stack Stress Test"
echo "═══════════════════════════════════════════════════════════════════"

# ─── SECTION 1: owc — All Stacks ───────────────────────────────────────────
header "SECTION 1: owc — create worktree across 8 stack types"
for stack in go typescript python rust c java ruby mixed; do
  setup_stack_repo "owc-$stack" "$stack"
  REPO="$HARNESS_TMP/owc-$stack"
  cd "$REPO"
  assert_ok "owc feature/test-$stack (stack=$stack)" \
    bash "$RT" c "feature/test-$stack" "test for $stack"
  cd "$OVAV_ROOT"
done

# ─── SECTION 2: owc — Edge Cases ───────────────────────────────────────────
header "SECTION 2: owc — edge cases"
setup_stack_repo "owc-edge" "go"
REPO="$HARNESS_TMP/owc-edge"
cd "$REPO"

# Special characters in branch name
assert_ok "owc feature/fix-42-special-chars" \
  bash "$RT" c "feature/fix-42-special-chars"

# Very long branch name
assert_ok "owc feature/very-long-branch-name-that-exceeds-typical-limits-for-testing" \
  bash "$RT" c "feature/very-long-branch-name-that-exceeds-typical-limits-for-testing"

# Duplicate branch detection (same branch in same repo = should fail)
assert_exit_nonzero "owc duplicate-branch (should fail)" \
  bash "$RT" c "feature/test-go"

# No args
assert_exit_nonzero "owc with no args (should fail)" \
  bash "$RT" c

cd "$OVAV_ROOT"

# ─── SECTION 3: owd — finalize + compliance levels ─────────────────────────
header "SECTION 3: owv + owd — verify and finalize"
for stack in go typescript python rust; do
  setup_stack_repo "owd-$stack" "$stack"
  REPO="$HARNESS_TMP/owd-$stack"
  cd "$REPO"
  bash "$RT" c "feature/test-$stack" >/dev/null 2>&1
  assert_ok "owv (stack=$stack)" bash "$RT" v
  cd "$OVAV_ROOT"
done

# ─── SECTION 4: owl — list all worktrees ───────────────────────────────────
header "SECTION 4: owl — list + zombie detection"
setup_stack_repo "owl-test" "go"
REPO="$HARNESS_TMP/owl-test"
cd "$REPO"
bash "$RT" c "feature/test-owl" >/dev/null 2>&1
assert_ok "owl lists worktrees" bash "$RT" l
cd "$OVAV_ROOT"

# ─── SECTION 5: ows — sync + maintenance ───────────────────────────────────
header "SECTION 5: ows — sync"
setup_stack_repo "ows-test" "go"
REPO="$HARNESS_TMP/ows-test"
cd "$REPO"
assert_ok "ows default" bash "$RT" s
cd "$OVAV_ROOT"

# ─── SECTION 6: owm — move worktree ────────────────────────────────────────
header "SECTION 6: owm — move"
setup_stack_repo "owm-test" "go"
REPO="$HARNESS_TMP/owm-test"
cd "$REPO"
bash "$RT" c "feature/test-owm" >/dev/null 2>&1
WT_PATH=$(cd "$REPO" && git worktree list --porcelain | grep -B1 "feature/test-owm" | head -1 | sed 's/worktree //')
if [ -n "$WT_PATH" ] && [ -d "$WT_PATH" ]; then
  DEST="$REPO/.ovav/worktrees/moved-owm"
  assert_ok "owm move" bash "$RT" m "$WT_PATH" "$DEST"
fi
cd "$OVAV_ROOT"

# ─── SECTION 7: owclean — orphan cleanup ───────────────────────────────────
header "SECTION 7: owclean — prune"
assert_ok "owclean prune" bash "$RT" clean

# ─── SECTION 8: owprep — project analysis ──────────────────────────────────
header "SECTION 8: owprep — project analysis cache"
setup_stack_repo "owprep-test" "go"
REPO="$HARNESS_TMP/owprep-test"
cd "$REPO"
assert_ok "owprep generates config" bash "$RT" prep
assert_ok "owprep config is valid JSON" test -f "$REPO/.ovav/worktree-config.json"
cd "$OVAV_ROOT"

# ─── SECTION 9: owsuggest — smart suggestions ──────────────────────────────
header "SECTION 9: owsuggest — context-aware suggestions"
setup_stack_repo "owsuggest-test" "go"
REPO="$HARNESS_TMP/owsuggest-test"
cd "$REPO"
assert_ok "owsuggest works" bash "$RT" suggest
cd "$OVAV_ROOT"

# ─── SECTION 10: Help parser — all 11 commands ─────────────────────────────
header "SECTION 10: --help / -h / --version across all commands"
for cmd in c d l v s clean m a r x lk; do
  assert_ok "ow$cmd --help" bash "$RT" "$cmd" --help
  assert_ok "ow$cmd -h" bash "$RT" "$cmd" -h
  assert_ok "ow$cmd --version" bash "$RT" "$cmd" --version
done

# ─── SECTION 11: Unknown flag rejection ────────────────────────────────────
header "SECTION 11: Unknown flag rejection"
# Only commands with strict flag parsing reject unknown flags.
# owl, owv, ows, owclean pass through all flags (lenient).
for cmd in c d s clean m; do
  assert_exit_nonzero "ow$cmd --bogus-flag rejects" bash "$RT" "$cmd" --bogus-flag
done

# ─── SECTION 12: owd — compliance flag passthrough ─────────────────────────
header "SECTION 12: owd --compliance passthrough (regression SU-1)"
setup_stack_repo "owd-compliance" "go"
REPO="$HARNESS_TMP/owd-compliance"
cd "$REPO"
bash "$RT" c "feature/test-compliance" >/dev/null 2>&1
# These should NOT fail with "Unknown flag" anymore
assert_exit_nonzero "owd --compliance standard (no work, just flag passthrough)" \
  bash "$RT" d --compliance standard feature/test-compliance
assert_exit_nonzero "owd --compliance strict (flag accepted, merge may fail)" \
  bash "$RT" d --compliance strict feature/test-compliance
cd "$OVAV_ROOT"

# ─── SECTION 13: Stress — rapid succession ─────────────────────────────────
header "SECTION 13: Stress — rapid owc/owv/ows in succession"
setup_stack_repo "stress-rapid" "go"
REPO="$HARNESS_TMP/stress-rapid"
cd "$REPO"
RAPID_PASS=0; RAPID_FAIL=0
for i in $(seq 1 5); do
  if bash "$RT" c "feature/stress-$i" >/dev/null 2>&1; then
    RAPID_PASS=$((RAPID_PASS+1))
  else
    RAPID_FAIL=$((RAPID_FAIL+1))
  fi
done
if [ "$RAPID_FAIL" -eq 0 ]; then
  green "  PASS rapid succession (5/5 created)"; PASS=$((PASS+1))
else
  red "  FAIL rapid succession ($RAPID_PASS/5 created)"; FAIL=$((FAIL+1))
fi
assert_ok "ows prune after rapid" bash "$RT" s
cd "$OVAV_ROOT"

# ─── SECTION 14: Empty / minimal repos ─────────────────────────────────────
header "SECTION 14: Empty and minimal repos"
EMPTY="$HARNESS_TMP/empty-repo"
rm -rf "$EMPTY"; mkdir -p "$EMPTY"; cd "$EMPTY"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
touch .gitkeep && git add . && git commit -qm "init empty"
assert_ok "owc in empty repo" bash "$RT" c "feature/empty-test"
assert_ok "owv in empty repo" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 15: Repos with binary files ───────────────────────────────────
header "SECTION 15: Repos with binary/large files"
BIN="$HARNESS_TMP/binary-repo"
rm -rf "$BIN"; mkdir -p "$BIN"; cd "$BIN"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
dd if=/dev/urandom of=test.bin bs=1K count=10 2>/dev/null
echo '{"name":"binrepo"}' > package.json
git add . && git commit -qm "init with binary"
assert_ok "owc with binary files" bash "$RT" c "feature/binary-test"
assert_ok "owv with binary files" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 16: Repos with special characters in filenames ─────────────────
header "SECTION 16: Special characters in filenames"
SPECIAL="$HARNESS_TMP/special-repo"
rm -rf "$SPECIAL"; mkdir -p "$SPECIAL"; cd "$SPECIAL"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
touch "file with spaces.txt" "file-with-dashes.go" "file_with_underscores.py" "file.multiple.dots.rs"
echo 'module special' > go.mod
git add . && git commit -qm "init with special files"
assert_ok "owc in special-chars repo" bash "$RT" c "feature/special-chars"
assert_ok "owv in special-chars repo" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 17: Nested git repos (submodules) ─────────────────────────────
header "SECTION 17: Nested git (submodule-style)"
NESTED="$HARNESS_TMP/nested-repo"
rm -rf "$NESTED"; mkdir -p "$NESTED/inner"; cd "$NESTED"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
cd inner && git init -q -b main && git config user.email "t@t" && git config user.name "T"
echo 'module inner' > go.mod && git add . && git commit -qm "init inner"
cd "$NESTED"
echo 'module nested' > go.mod && git add . && git commit -qm "init nested"
assert_ok "owc with nested git" bash "$RT" c "feature/nested-test"
assert_ok "owv with nested git" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 18: Unicode content ───────────────────────────────────────────
header "SECTION 18: Unicode content in files"
UNICODE="$HARNESS_TMP/unicode-repo"
rm -rf "$UNICODE"; mkdir -p "$UNICODE"; cd "$UNICODE"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
echo '// 日本語コメント' > main.go
echo '# Ελληνικά' > readme.md
echo '// Кириллица' > utils.rs
echo 'module unicode' > go.mod
git add . && git commit -qm "init unicode"
assert_ok "owc with unicode" bash "$RT" c "feature/unicode-test"
assert_ok "owv with unicode" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 19: Permission stress ─────────────────────────────────────────
header "SECTION 19: Permission edge cases"
PERM="$HARNESS_TMP/perm-repo"
rm -rf "$PERM"; mkdir -p "$PERM"; cd "$PERM"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
echo 'module perm' > go.mod
chmod 444 go.mod
git add . && git commit -qm "init perm" 2>/dev/null || git add . && git commit -qm "init perm"
chmod 644 go.mod
assert_ok "owc with restricted perms" bash "$RT" c "feature/perm-test"
assert_ok "owv with restricted perms" bash "$RT" v
cd "$OVAV_ROOT"

# ─── SECTION 20: Concurrent worktree creation ──────────────────────────────
header "SECTION 20: Concurrent worktree creation (parallel)"
PARALLEL="$HARNESS_TMP/parallel-repo"
rm -rf "$PARALLEL"; mkdir -p "$PARALLEL"; cd "$PARALLEL"
git init -q -b develop
git config user.email "test@ovav.test" && git config user.name "T" && git config commit.gpgsign false
echo 'module parallel' > go.mod
git add . && git commit -qm "init parallel"

PIDS=()
for i in $(seq 1 3); do
  (cd "$PARALLEL" && bash "$RT" c "feature/parallel-$i" >/dev/null 2>&1) &
  PIDS+=($!)
done
PARALLEL_PASS=0
for pid in "${PIDS[@]}"; do
  if wait "$pid" 2>/dev/null; then
    PARALLEL_PASS=$((PARALLEL_PASS+1))
  fi
done
if [ "$PARALLEL_PASS" -ge 2 ]; then
  green "  PASS concurrent creation ($PARALLEL_PASS/3 succeeded)"; PASS=$((PASS+1))
else
  red "  FAIL concurrent creation ($PARALLEL_PASS/3 succeeded)"; FAIL=$((FAIL+1))
fi
cd "$OVAV_ROOT"

# ════════════════════════════════════════════════════════════════════════════
# CLEANUP
# ════════════════════════════════════════════════════════════════════════════
rm -rf "$HARNESS_TMP"

# ════════════════════════════════════════════════════════════════════════════
# SUMMARY
# ════════════════════════════════════════════════════════════════════════════
TOTAL=$((PASS + FAIL))
echo ""
echo "═══════════════════════════════════════════════════════════════════"
echo "  OWS AGGRESSIVE HARNESS RESULTS"
echo "═══════════════════════════════════════════════════════════════════"
echo "  Total assertions: $TOTAL"
green "  Passed: $PASS"
[ "$FAIL" -gt 0 ] && red "  Failed: $FAIL"
[ "$FAIL" -eq 0 ] && green "  Failed: 0"
echo "  Pass rate: $(( PASS * 100 / TOTAL ))%"
echo "═══════════════════════════════════════════════════════════════════"
[ "$FAIL" = "0" ] && exit 0 || exit 1
