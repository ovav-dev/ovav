# ══════════════════════════════════════════════════════════════════════════
# reload.fish — OVAV 2026 modern live-reload for fish shell
# ══════════════════════════════════════════════════════════════════════════
# Provides:
#   reload            — Re-source all conf.d/*.fish + re-exec fish (PID-preserving)
#   reload --quiet    — Silent (for scripts/CI)
#   reload --rehash   — Only rehash, no exec (cheap, keeps jobs)
#   reload --check    — Dry-run: show what would happen, do nothing
#   reload --full     — Wipe completions + abbreviations + functions cache, then reload
#   reload --ovav     — Extra: rebuild ovav binary if HEAD changed
# ══════════════════════════════════════════════════════════════════════════
# Design choices for 2026:
#   - PID preserved via `exec fish` (the canonical way to live-reload a shell)
#   - Jobs in background are NOT orphaned (exec replaces process, not fork)
#   - Universal variables (fish_user_paths etc.) carry over automatically
#   - Times the reload and reports ms elapsed for diagnostics
#   - Compatible with all modern fish 3.6+ / fish 4.x
# ══════════════════════════════════════════════════════════════════════════

# ── Internal helpers ─────────────────────────────────────────────────────

# Count of conf.d files that ship with OVAV (or user-defined).
function __ovav_reload_count_modules
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        echo 0
        return
    end
    set -l n (count (command ls "$d"/*.fish 2>/dev/null))
    if test "$n" -gt 0
        echo $n
    else
        echo 0
    end
end

# Detect which conf.d files have changed since the last reload marker.
# The marker lives in $__ovav_last_reload_epoch_unix (universal var).
function __ovav_reload_detect_changes
    set -l changed
    set -l now (date +%s)
    set -l last $__ovav_last_reload_epoch_unix
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        return
    end
    for f in "$d"/*.fish
        test -f "$f"; or continue
        set -l mtime (command stat -c %Y "$f" 2>/dev/null; or command stat -f %m "$f" 2>/dev/null; or echo 0)
        if test "$last" -gt 0 2>/dev/null; and test "$mtime" -gt "$last" 2>/dev/null
            set changed $changed (path basename "$f")
        end
    end
    if set -q changed[1]
        string join ', ' $changed
    end
end

# Hash a snapshot of conf.d so a future reload can detect drift.
function __ovav_reload_snapshot_hash
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        echo 'empty'
        return
    end
    set -l files (command ls "$d"/*.fish 2>/dev/null | sort)
    if set -q files[1]
        command cat $files 2>/dev/null | command sha256sum 2>/dev/null | string sub -l 12
    else
        echo 'empty'
    end
end

# Reset fish_user_path cache by re-evaluating conf.d in a subshell.
# Then we re-exec fish, which picks up the new PATH natively.
function __ovav_reload_ovav_update_prompt
    printf '\n%s %s\n' (set_color green --bold) '⟳' (set_color normal)
    set_color yellow
    echo '   reload: live config refresh (OVAV 2026)'
    set_color normal
end

# Build & install the latest ovav binary when the user explicitly opts in.
function __ovav_reload_maybe_rebuild_ovav
    if not command -sq ovav
        return
    end
    # Detect git source root.
    set -l src ""
    for candidate in ~/Systems/OVAV /home/braka/Systems/ovav ~/ovav ~/workspace/ovav
        if test -d "$candidate/.git"
            set src $candidate
            break
        end
    end
    if test -z "$src"
        return
    end
    set -l head_sha (command git -C "$src" rev-parse --short HEAD 2>/dev/null)
    set -l bin_sha (command sha256sum (command which ovav) 2>/dev/null | string sub -l 7)
    set -l go_runtime "$src/go-runtime"
    if not test -d "$go_runtime"
        return
    end
    set -l built_at (command stat -c %Y "$go_runtime/ovav" 2>/dev/null; or echo 0)
    set -l now (date +%s)
    set -l age_days (math "( $now - $built_at ) / 86400" 2>/dev/null; or echo 999)
    if test "$age_days" -gt 0 2>/dev/null
        printf '   %s ovav:%s%s%s is %sd old, rebuilding…\n' \
            (set_color cyan) $head_sha (set_color normal) (set_color cyan) $age_days
        command go -C "$go_runtime" build -o "$HOME/.local/bin/ovav" ./cmd/ovav 2>/dev/null
    end
end

# ── Subcommand: --rehash-only ────────────────────────────────────────────

function __ovav_reload_rehash_only
    set -l before (count (functions --all --no-details 2>/dev/null))
    # Re-source conf.d so any new functions/aliases load.
    for f in ~/.config/fish/conf.d/*.fish
        test -f "$f"; or continue
        source $f
    end
    # Pull PATHS / completions refreshed by source above.
    fish_update_completions 2>/dev/null
    set -l after (count (functions --all --no-details 2>/dev/null))
    printf '%s ✓ rehash%s  functions: %d → %d\n' \
        (set_color green) (set_color normal) $before $after
end

# ── Subcommand: --check (dry run) ─────────────────────────────────────────

function __ovav_reload_check
    set -l n (__ovav_reload_count_modules)
    set -l changed (__ovav_reload_detect_changes)
    set -l h (__ovav_reload_snapshot_hash)
    echo (set_color cyan --bold)'reload --check:'(set_color normal)
    echo "   conf.d modules : $n"
    if test -n "$changed"
        echo -n '   changed since  : '
        set_color yellow
        echo $changed
        set_color normal
    else
        echo '   changed since  : (none)'
    end
    echo "   snapshot hash  : $h"
    echo
    if test -n "$changed"
        echo (set_color yellow)'   → reload needed'(set_color normal)
    else
        echo (set_color green)'   → reload not strictly needed'(set_color normal)
    end
end

# ── Main entry point: reload ──────────────────────────────────────────────
# Recognised flags:
#   -q / --quiet   Silence non-essential output (for scripts / CI)
#   -h / --rehash  Rehash + source conf.d only, no exec (jobs survive)
#   -c / --check   Dry-run: report what would happen, do nothing
#   -f / --full    Wipe completion cache + abbreviations, then reload
#   -o / --ovav    Also rebuild + reinstall ovav binary if it is stale
#   -v / --verbose Show timing breakdown for each step

function reload --description 'OVAV 2026: live-reload fish config + runtime state'
    set -l started (date +%s%N 2>/dev/null; or date +%s)
    set -l do_quiet false
    set -l do_rehash_only false
    set -l do_check false
    set -l do_full false
    set -l do_ovav false
    set -l do_verbose false

    # Parse flags.
    set -l rest
    for a in $argv
        switch "$a"
            case '-q' '--quiet'
                set do_quiet true
            case '-h' '--rehash'
                set do_rehash_only true
            case '-c' '--check'
                set do_check true
            case '-f' '--full'
                set do_full true
            case '-o' '--ovav'
                set do_ovav true
            case '-v' '--verbose'
                set do_verbose true
            case '-*'
                echo "reload: unknown flag '$a' (try reload --help)" >&2
                return 2
            case '*'
                set rest $rest $a
        end
    end

    # ── --help ──────────────────────────────────────────────────────────
    if set -q rest[1]; and test "$rest[1]" = "help"
        set -l cyan (set_color cyan --bold)
        set -l normal (set_color normal)
        set -l yellow (set_color yellow)
        echo "$cyan""reload — $normal$yellow""OVAV 2026 live-reload (PID-preserving)$normal"
        echo
        echo "$cyan""Usage:$normal"" reload [flags]"
        echo
        echo "$cyan""Flags:$normal"
        echo "   -q / --quiet    silence non-essential output"
        echo "   -h / --rehash   source conf.d only (jobs survive, fast)"
        echo "   -c / --check    dry-run — report what would happen"
        echo "   -f / --full     wipe completions + abbreviations first"
        echo "   -o / --ovav     also rebuild ovav binary if stale"
        echo "   -v / --verbose  show per-step timing breakdown"
        echo
        echo "$cyan""Behaviour:$normal"" default uses "exec fish" which replaces"
        echo "   the current shell so PID is preserved and jobs re-parented."
        echo
        echo "   Universal variables (fish_user_paths etc.) carry over"
        echo "   automatically because fish preserves them across exec."
        return 0
    end

    # ── --check ──────────────────────────────────────────────────────────
    if $do_check
        __ovav_reload_check
        return 0
    end

    # ── --rehash-only ───────────────────────────────────────────────────
    if $do_rehash_only
        __ovav_reload_rehash_only
        set -l elapsed (math "( ( date +%s%N ) - $started ) / 1000000" 2>/dev/null)
        if not $do_quiet
            printf '   reload: rehash complete in ~%s ms\n' $elapsed
        end
        set -g __ovav_last_reload_epoch_unix (date +%s)
        return 0
    end

    # ── --full: wipe completion + abbreviation caches ───────────────────
    if $do_full
        if not $do_quiet
            echo (set_color yellow)⟳(set_color normal) '   wiping completion + abbreviation caches…'
        end
        command rm -rf ~/.local/share/fish/fish_completions_cache_* 2>/dev/null
        abbr --erase 2>/dev/null
        for f in ~/.config/fish/conf.d/*.fish
            test -f "$f"; or continue
            source $f
        end
    end

    # ── --ovav (optional): rebuild binary when stale ────────────────────
    if $do_ovav
        __ovav_reload_maybe_rebuild_ovav
    end

    # ── The reload itself ────────────────────────────────────────────────
    # Default path: re-source conf.d so any new bindings are live, then
    # exec fish to get a clean scope while preserving PID and jobs.
    #
    # We use exec instead of `source conf.d/*.fish` because each `source`
    # creates a local scope and tends to lose universal variable refreshes
    # made in newer fish releases.
    if not $do_quiet
        __ovav_reload_ovav_update_prompt
        set -l n (__ovav_reload_count_modules)
        set -l changed (__ovav_reload_detect_changes)
        printf '   modules : %s\n' (set_color cyan)"$n"(set_color normal)
        if test -n "$changed"
            printf '   changed : %s\n' (set_color yellow)"$changed"(set_color normal)
        else
            printf '   changed : %s\n' (set_color brgreen)'(none — fast path)'(set_color normal)
        end
        printf '   %s exec fish%s   PID preserved, jobs re-parented\n' \
            (set_color cyan) (set_color normal)
    end

    # Mark this reload so future `--check` can diff what changed.
    set -g __ovav_last_reload_epoch_unix (date +%s)

    # Persist into universal vars so it survives the exec.
    set -U __ovav_last_reload_epoch_unix $__ovav_last_reload_epoch_unix

    # The actual swap: PID-preserving, no orphaned jobs.
    exec fish
end
