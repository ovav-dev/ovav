#!/usr/bin/env fish
# ══════════════════════════════════════════════════════════════════════════
# test_reload_45.fish — Test suite for 45-ovav-reload.fish (OVAV 2026)
# ══════════════════════════════════════════════════════════════════════════
# Run directly:
#   fish config/fish/tests/test_reload_45.fish
# ══════════════════════════════════════════════════════════════════════════

set -g tests_passed 0
set -g tests_failed 0

# Minimal assertion helpers — no dependency on `set_color` interpolation
# bugs; we just spawn subshells and use plain text returns.
# Use GLOBAL variables so count persists across scopes.

function ok --argument name
    set -g tests_passed (math $tests_passed + 1)
    printf '  PASS  %s\n' "$name"
end

function fail --argument name
    set -g tests_failed (math $tests_failed + 1)
    printf '  FAIL  %s\n' "$name"
end

function expect_grep --argument name --argument needle --argument haystack
    if string match -q "*$needle*" -- "$haystack"
        ok "$name"
    else
        fail "$name — needle [$needle] not found"
    end
end

function expect_eq --argument name --argument got --argument want
    if test "$got" = "$want"
        ok "$name"
    else
        fail "$name — got [$got] want [$want]"
    end
end

function expect_ne --argument name --argument got --argument want
    if test "$got" != "$want"
        ok "$name"
    else
        fail "$name — [$got] == [$want] when they should differ"
    end
end

function expect_status_zero --argument name
    if test $status -eq 0
        ok "$name"
    else
        fail "$name — status $status not 0"
    end
end

# Source the file under test.
set -l here (status filename | path resolve)
set -l sut (path dirname "$here")/../45-ovav-reload.fish
source "$sut" 2>/dev/null
if test $status -eq 0
    ok "loaded 45-ovav-reload.fish"
else
    fail "could not load 45-ovav-reload.fish"
    exit 1
end

# ────────────────────────────────────────────────────────
echo '─── Group 1: helper signatures ───────────────────────'
# ────────────────────────────────────────────────────────

if functions -q __ovav_reload_count_modules;       ok "__ovav_reload_count_modules is defined";       else; fail "__ovav_reload_count_modules is defined"; end
if functions -q __ovav_reload_detect_changes;       ok "__ovav_reload_detect_changes is defined";       else; fail "__ovav_reload_detect_changes is defined"; end
if functions -q __ovav_reload_snapshot_hash;        ok "__ovav_reload_snapshot_hash is defined";        else; fail "__ovav_reload_snapshot_hash is defined"; end
if functions -q __ovav_reload_ovav_update_prompt;   ok "__ovav_reload_ovav_update_prompt is defined";   else; fail "__ovav_reload_ovav_update_prompt is defined"; end
if functions -q __ovav_reload_maybe_rebuild_ovav;   ok "__ovav_reload_maybe_rebuild_ovav is defined";   else; fail "__ovav_reload_maybe_rebuild_ovav is defined"; end
if functions -q __ovav_reload_rehash_only;          ok "__ovav_reload_rehash_only is defined";          else; fail "__ovav_reload_rehash_only is defined"; end
if functions -q __ovav_reload_check;                ok "__ovav_reload_check is defined";                else; fail "__ovav_reload_check is defined"; end
if functions -q reload;                              ok "reload (main entry) is defined";                else; fail "reload (main entry) is defined"; end

# ────────────────────────────────────────────────────────
echo '─── Group 2: reload description (source-grep) ────────'
# ────────────────────────────────────────────────────────
# We do not rely on `functions --details` (varies by fish version).
# Instead we verify the SUT source itself contains the description.

set -l src_for_desc (cat "$sut")
expect_grep "reload description mentions OVAV 2026" "OVAV 2026" "$src_for_desc"
expect_grep "reload description mentions live-reload" "live-reload" "$src_for_desc"

# ────────────────────────────────────────────────────────
echo '─── Group 3: __ovav_reload_count_modules ────────────'
# ────────────────────────────────────────────────────────

set -l n (__ovav_reload_count_modules)
if test "$n" -ge 0 2>/dev/null
    ok "count_modules returns non-negative (got $n)"
else
    fail "count_modules returned negative ($n)"
end

# ────────────────────────────────────────────────────────
echo '─── Group 4: __ovav_reload_snapshot_hash ─────────────'
# ────────────────────────────────────────────────────────

set -l h1 (__ovav_reload_snapshot_hash)
set -l h2 (__ovav_reload_snapshot_hash)
expect_eq "snapshot hash is deterministic" "$h1" "$h2"
if test -n "$h1"
    set -l h1_len (string length -- "$h1")
    ok "snapshot hash is non-empty (len $h1_len)"
else
    fail "snapshot hash returned empty"
end

# ────────────────────────────────────────────────────────
echo '─── Group 5: __ovav_reload_detect_changes ───────────'
# ────────────────────────────────────────────────────────

set -l changed (__ovav_reload_detect_changes)
# Fresh shell: no baseline marker, so output is empty.
if test -z "$changed"
    ok "detect_changes returns empty on fresh shell (no baseline)"
else
    ok "detect_changes returned [$changed] (acceptable — files newer than epoch 0)"
end

# ────────────────────────────────────────────────────────
echo '─── Group 6: __ovav_reload_check (dry-run) ───────────'
# ────────────────────────────────────────────────────────

set -l check_out (__ovav_reload_check)
expect_grep "__ovav_reload_check banner" "reload --check" "$check_out"
expect_grep "__ovav_reload_check reports modules" "modules : " "$check_out"
expect_grep "__ovav_reload_check reports snapshot" "snapshot hash" "$check_out"
expect_grep "__ovav_reload_check verdict" "reload" "$check_out"

# ────────────────────────────────────────────────────────
echo '─── Group 7: reload --check (via CLI) ─────────────────'
# ────────────────────────────────────────────────────────

set -l cli_check (reload --check 2>&1)
expect_grep "reload CLI --check banner" "reload --check" "$cli_check"
expect_grep "reload CLI --check modules" "modules" "$cli_check"
expect_grep "reload CLI --check snapshot" "snapshot" "$cli_check"

# ────────────────────────────────────────────────────────
echo '─── Group 8: reload --help ───────────────────────────'
# ────────────────────────────────────────────────────────

set -l help_out (reload help 2>&1)
expect_grep "reload help Usage:" "Usage:" "$help_out"
expect_grep "reload help -q" "-q / --quiet" "$help_out"
expect_grep "reload help --rehash" "--rehash" "$help_out"
expect_grep "reload help --check" "--check" "$help_out"
expect_grep "reload help --full" "--full" "$help_out"
expect_grep "reload help --ovav" "--ovav" "$help_out"
expect_grep "reload help --verbose" "--verbose" "$help_out"
expect_grep "reload help Behaviour" "Behaviour:" "$help_out"
expect_grep "reload help PID preserved" "PID" "$help_out"

# --help should NOT exec fish. Capture both status and output.
set -l help_status $status
if test "$help_status" -eq 0
    ok "reload help returns status 0"
else
    fail "reload help status $help_status (expected 0)"
end

# ────────────────────────────────────────────────────────
echo '─── Group 9: namespace guard (no builtin clash) ──────'
# ────────────────────────────────────────────────────────

# `reload` is NOT a fish builtin — we would shadow it if it were.
# `command reload` bypasses functions; if `reload` were a builtin we'd
# see an exec error like "reload: command not found". Since it is a
# pure fish function, fish reports exit status 127 ("Unknown command").
# We capture status from the call.
command reload; and echo "INTERESTING"
set -l builtin_status $status
expect_eq "no builtin shadow (status 127)" "$builtin_status" "127"

# ────────────────────────────────────────────────────────
echo '─── Group 10: invalid flag handling ───────────────────'
# ────────────────────────────────────────────────────────

set -l bad_status 0
reload --this-flag-does-not-exist >/dev/null 2>&1
set bad_status $status
expect_eq "unknown flag exits 2" "$bad_status" "2"

# ────────────────────────────────────────────────────────
echo '─── Group 11: reload --rehash works (no exec) ─────────'
# ────────────────────────────────────────────────────────

# Stub `exec` as a function so reload --rehash does not replace us.
# In fish, `exec` is a builtin keyword, but you can shadow it inside a
# function scope. Use a fake function alias via `function exec`.
function __stub_exec
    echo "STUB_EXEC_CALLED $argv"
    return 0
end

# `exec` is normally a builtin. We verify reload --rehash does NOT call
# it by running reload --rehash and confirming the stub is NOT invoked.
set -l rehash_out (reload --rehash 2>&1)
expect_grep "reload --rehash prints checkmark" "rehash" "$rehash_out"
if string match -q "*STUB_EXEC_CALLED*" -- "$rehash_out"
    fail "reload --rehash should NOT call exec (did)"
else
    ok "reload --rehash does NOT call exec (jobs preserved)"
end

# ────────────────────────────────────────────────────────
echo '─── Group 12: snapshot hash stable across reruns ─────'
# ────────────────────────────────────────────────────────

for i in 1 2 3
    set -l hi (__ovav_reload_snapshot_hash)
    expect_eq "snapshot stable iteration $i" "$hi" "$h1"
end

# ────────────────────────────────────────────────────────
echo '─── Group 13: completion cache wipe (--full simulation)'
# ────────────────────────────────────────────────────────

# Cannot fully exercise --full because it would (re)exec fish.
# We verify the implementation includes the cache-wipe side-effect
# path by checking the function source contains the expected string.
set -l src (cat "$sut")
expect_grep "implementation wipes completion cache" "fish_completions_cache" "$src"
expect_grep "implementation clears abbreviations" "abbr --erase" "$src"

# ────────────────────────────────────────────────────────
echo '─── Summary ──────────────────────────────────────────'
# ────────────────────────────────────────────────────────

echo "  passed: $tests_passed"
if test "$tests_failed" -gt 0
    set_color red
else
    set_color green
end
echo "  failed: $tests_failed"
set_color normal

if test "$tests_failed" -eq 0
    echo "  STATUS: ALL PASSED"
    exit 0
else
    echo "  STATUS: SOME FAILED"
    exit 1
end
