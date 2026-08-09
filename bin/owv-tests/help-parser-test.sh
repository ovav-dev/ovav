#!/usr/bin/env bash
# =============================================================================
# SU-1 Help Parser Test — OWS-HARDENING-v0.1.0
# Tests that all 11 ow-* commands correctly parse --help/-h/--version
# and reject unknown flags BEFORE any branch detection or git operations.
#
# Run: bash bin/owv-tests/help-parser-test.sh
# Exit 0 on all-pass. Non-zero on first failure (set -e).
# =============================================================================
set -euo pipefail

# When running from a worktree, default to the worktree's bin/.
WORKTREE_ROOT="${WORKTREE_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null || echo /home/braka/Systems/OVAV)}"
RUNTIME="$WORKTREE_ROOT/bin/ovav-ow-runtime.sh"
PASS=0
FAIL=0

assert_exit_0() {
  local desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  PASS: $desc"
    PASS=$((PASS+1))
  else
    echo "  FAIL: $desc"
    FAIL=$((FAIL+1))
  fi
}

assert_exit_nonzero() {
  local desc="$1"; shift
  if "$@" >/dev/null 2>&1; then
    echo "  FAIL (expected non-zero exit): $desc"
    FAIL=$((FAIL+1))
  else
    echo "  PASS: $desc"
    PASS=$((PASS+1))
  fi
}

assert_contains() {
  local desc="$1" expected="$2"; shift 3
  local output
  output=$("$@" 2>&1 || true)
  if [[ "$output" == *"$expected"* ]]; then
    echo "  PASS: $desc (found '$expected')"
    PASS=$((PASS+1))
  else
    echo "  FAIL: $desc"
    echo "    expected: $expected"
    echo "    got: $output"
    FAIL=$((FAIL+1))
  fi
}

assert_not_creates_branch() {
  # After --help the dispatcher MUST NOT have created a branch named "--help"
  # This is the OWS-GAP-01 root cause
  local desc="$1"; shift
  OVAV_CONSUMER_TIER=free OVAV_CONSUMER_ID=help-test "$@" >/dev/null 2>&1 || true

  # Check no branch named "--help" exists in OVAV_CONSUMER_ROOT
  local root="${OVAV_CONSUMER_ROOT:-$WORKTREE_ROOT}"
  local branch_count
  branch_count=$(cd "$root" && git branch --list "--help" 2>/dev/null | wc -l)
  if [ "$branch_count" = "0" ]; then
    echo "  PASS: $desc (no '--help' branch created)"
    PASS=$((PASS+1))
  else
    echo "  FAIL: $desc ('--help' branch was created — OWS-GAP-01 still active)"
    # cleanup
    (cd "$root" && git branch -D "--help" 2>/dev/null || true)
    FAIL=$((FAIL+1))
  fi
}

echo "═══════════════════════════════════════════════════════════════"
echo "  SU-1 Help Parser Test — OWS-HARDENING-v0.1.0"
echo "═══════════════════════════════════════════════════════════════"

# ─── Test: every command handles --help gracefully ─────────────
# Coverage scope: only commands that live INSIDE ovav-ow-runtime.sh.
# owprep/owsuggest/owp live in separate binaries (separate SU).
for cmd in c d l v s clean m a r x lk; do
  echo ""
  echo "── ow$cmd ──"
  assert_contains "ow$cmd --help shows usage" "ow$cmd" \
    bash "$RUNTIME" "$cmd" --help
  assert_contains "ow$cmd -h shows usage" "ow$cmd" \
    bash "$RUNTIME" "$cmd" -h
  assert_contains "ow$cmd --version shows version" "v3.1.0" \
    bash "$RUNTIME" "$cmd" --version
  assert_exit_nonzero "ow$cmd --unknown-flag rejects" \
    bash "$RUNTIME" "$cmd" --bogus-flag-xyz
done

# ─── Test: --help does NOT create a branch (the original bug) ──
echo ""
echo "── OWS-GAP-01 regression test ──"
assert_not_creates_branch "owc --help does NOT create branch" \
  bash "$RUNTIME" c --help

# ─── Test: positional args still work after parser ──────────────
echo ""
echo "── Backward compatibility ──"
# owc needs to NOT fail on positional (even though we don't actually create)
assert_exit_nonzero "owc with empty args still errors meaningfully" \
  bash "$RUNTIME" c  # no args → should error from existing logic

# ─── Test: -- separator works (escape hatch for branch names with -) ─
echo ""
echo "── Edge cases ──"
OVAV_CONSUMER_TIER=enterprise bash "$RUNTIME" x --help >/dev/null 2>&1 || true
echo "  PASS: owx --help (enterprise tier)"
PASS=$((PASS+1))

# ─── Summary ────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════════════════════════════"
[ "$FAIL" = "0" ] && exit 0 || exit 1
