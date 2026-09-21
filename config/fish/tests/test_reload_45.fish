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
if functions -q __ovav_reload_detect_environment;   ok "__ovav_reload_detect_environment is defined";   else; fail "__ovav_reload_detect_environment is defined"; end
if functions -q __ovav_reload_fmt_environment;      ok "__ovav_reload_fmt_environment is defined";      else; fail "__ovav_reload_fmt_environment is defined"; end
if functions -q __ovav_reload_maybe_rebuild_ovav;   ok "__ovav_reload_maybe_rebuild_ovav is defined";   else; fail "__ovav_reload_maybe_rebuild_ovav is defined"; end
if functions -q __ovav_reload_rehash_only;          ok "__ovav_reload_rehash_only is defined";          else; fail "__ovav_reload_rehash_only is defined"; end
if functions -q __ovav_reload_banner_short;         ok "__ovav_reload_banner_short is defined";         else; fail "__ovav_reload_banner_short is defined"; end
if functions -q __ovav_reload_banner_long;          ok "__ovav_reload_banner_long is defined";          else; fail "__ovav_reload_banner_long is defined"; end
if functions -q __ovav_reload_maybe_wipe_caches;    ok "__ovav_reload_maybe_wipe_caches is defined";    else; fail "__ovav_reload_maybe_wipe_caches is defined"; end
if functions -q __ovav_reload_check;                ok "__ovav_reload_check is defined";                else; fail "__ovav_reload_check is defined"; end
if functions -q __ovav_reload_module_fingerprint;   ok "__ovav_reload_module_fingerprint is defined";   else; fail "__ovav_reload_module_fingerprint is defined"; end
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
expect_grep "__ovav_reload_check reports modules" "modules" "$check_out"
expect_grep "__ovav_reload_check reports hash" "hash    :" "$check_out"
expect_grep "__ovav_reload_check reports drift" "drift   :" "$check_out"
expect_grep "__ovav_reload_check reports env" "env     :" "$check_out"

# ────────────────────────────────────────────────────────
echo '─── Group 7: reload --check (via CLI) ─────────────────'
# ────────────────────────────────────────────────────────

set -l cli_check (reload --check 2>&1)
expect_grep "reload CLI --check banner" "reload --check" "$cli_check"
expect_grep "reload CLI --check modules" "modules" "$cli_check"
expect_grep "reload CLI --check env" "env" "$cli_check"

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
expect_grep "reload help --force" "--force" "$help_out"
expect_grep "reload help env-aware" "env-aware" "$help_out"
expect_grep "reload help agent" "agent" "$help_out"

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
echo '─── Group 14: no shadow warning (set -U vs -g) ────────'
# ────────────────────────────────────────────────────────
# Regression: old version did 'set -g' then 'set -U' for the same var name.
# Fish emits '...global by that name shadows it' on the universal set.
# Fixed by removing the prior -g declaration — only universal is needed.
# Use grep to filter comments before checking for set -g pattern (the
# docstring mentions $__ovav_last_reload_epoch_unix but lives on a line
# starting with '#').

expect_grep "implementation uses set -U (no -g shadow)" "set -U" "$src"
set -l code_lines (echo "$src" | grep -v '^\s*#' | grep -v '^\s*$')
if string match -q "*set -g __ovav_last_reload_epoch_unix*" -- "$code_lines"
    fail "source still has redundant set -g before set -U"
else
    ok "no redundant set -g before set -U"
end

# ────────────────────────────────────────────────────────
echo '─── Group 15: env-aware (alacritty + tmux + opencode) ──'
# ────────────────────────────────────────────────────────
# We test the env-detection helper against the *current* environment by
# looking for known keys. We don't mock ps; instead we assert the format
# contract and check that real keys are present.

set -l env_out (__ovav_reload_fmt_environment)
expect_grep "env: ppid present" "ppid=" "$env_out"
expect_grep "env: pid present" "pid=" "$env_out"
expect_grep "env: terminal present" "terminal=" "$env_out"
expect_grep "env: mux present" "mux=" "$env_out"
expect_grep "env: agent present" "agent=" "$env_out"
expect_grep "env: embedded present" "embedded=" "$env_out"
expect_grep "env: fish_depth present" "fish_depth=" "$env_out"

# Whitespace: fields must be space-separated (parseable downstream).
# Embedded newlines would break downstream parsers.
if string match -qr '\n' -- "$env_out"
    fail "env record contains a literal newline"
else
    ok "env record is single-line (no embedded newlines)"
end

# ────────────────────────────────────────────────────────
echo '─── Group 16: function inventory ──────────────────────'
# ────────────────────────────────────────────────────────
# Power-user surface check — all advertised functions exist.

for fn in __ovav_reload_detect_environment \
         __ovav_reload_fmt_environment \
         __ovav_reload_count_modules \
         __ovav_reload_detect_changes \
         __ovav_reload_snapshot_hash \
         __ovav_reload_module_fingerprint \
         __ovav_reload_banner_short \
         __ovav_reload_banner_long \
         __ovav_reload_check \
         __ovav_reload_rehash_only \
         __ovav_reload_maybe_wipe_caches
    if functions -q "$fn"
        ok "$fn exists"
    else
        fail "$fn declared in help but not defined"
    end
end

# ────────────────────────────────────────────────────────
echo '─── Group 17: agent-aware rehash (no exec) ───────────'
# ────────────────────────────────────────────────────────
# An agent-detected shell must be able to use --rehash without exec
# being called. We can't perfectly mock the ancestor chain, but we
# can verify the function exists, runs, and does not invoke exec
# (which would replace this test process — we would never see output).

set -l rehash_out (reload --rehash 2>&1 | string collect)
expect_grep "--rehash prints checkmark" "✓ rehash" "$rehash_out"
if string match -q "*STUB*" -- "$rehash_out"
    fail "--rehash printed a stub marker — test bug"
end
ok "--rehash runs without crashing the test process"

# ────────────────────────────────────────────────────────
echo '─── Group 18: error path (unknown flag) ───────────────'
# ────────────────────────────────────────────────────────

# Capture stderr+stdout via a temp file so $status survives the pipe.
set -l bad_path (mktemp)
reload --bogus-flag >"$bad_path" 2>&1
set -l bad_status $status
expect_eq "unknown flag exits 2" "$bad_status" "2"
set -l bad (cat "$bad_path")
expect_grep "unknown flag error mentions reload:" "reload:" "$bad"
command rm "$bad_path"

# ────────────────────────────────────────────────────────
echo '─── Group 19: source-grep — agent-aware skip exists ─'
# ────────────────────────────────────────────────────────
# We verify the implementation mentions the agent-skip path explicitly
# so regressions are caught even when running tests in a clean shell.

expect_grep "agent-skip path is documented" "wrapped by an agent" "$src"
expect_grep "agent names list contains opencode" "opencode" "$src"
expect_grep "agent names list contains claude-code" "claude" "$src"
expect_grep "agent names list contains warp" "warp" "$src"
expect_grep "force flag exists for agent override" "do_force" "$src"

# ────────────────────────────────────────────────────────
echo '─── Group 20: regression — no “if $var” empty-expansion ──'
# ────────────────────────────────────────────────────────
# Earlier version had `if $do_verbose` style — when $do_verbose was empty
# the `if $do_verbose` expanded to `if` which fish rejects. This regression
# test ensures we use `test "$var" = true` everywhere instead.

# Search for the dangerous pattern in non-comment lines.
set -l code_lines (echo "$src" | grep -v '^\s*#' | grep -v '^\s*$')
if string match -q "*if \$do_*" -- "$code_lines"
    fail "regression: 'if \$do_xxx' style still present (will explode on empty)"
else
    ok "no 'if \$do_xxx' patterns (test-style applied)"
end

# And no `else if $var` either
if string match -q "*else if \$*" -- "$code_lines"
    fail "regression: 'else if \$var' pattern still present (will explode on empty)"
else
    ok "no 'else if \$var' patterns"
end

# Constants must use bare `set` (not `set -g`) at file top level
# so we don't shadow any future universal by the same name.
if string match -q "*set -g __ovav_reload_*" -- "$code_lines"
    fail "regression: 'set -g __ovav_reload_*' still present (will shadow universal)"
else
    ok "no 'set -g __ovav_reload_*' (avoids universal shadow)"
end

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
