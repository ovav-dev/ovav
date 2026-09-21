# ══════════════════════════════════════════════════════════════════════════
# reload.fish — OVAV 2026 intelligent live-reload for fish shell
# ══════════════════════════════════════════════════════════════════════════
# Provides:
#   reload            — Live-reload fish config + selective module refresh.
#                        Auto-detects environment (alacritty+tmux+opencode)
#                        and adapts verbosity + safety. Skips refreshes
#                        that would harm the agent context.
#   reload --quiet    — No banner output (for scripts / CI)
#   reload --rehash   — Only re-source conf.d, no exec (jobs survive)
#   reload --check    — Dry-run with environment + drift + budget report
#   reload --full     — Wipe completions + abbreviations, then reload
#   reload --ovav     — Also rebuild + reinstall ovav binary if HEAD changed
#   reload --force    — Skip safety checks for embedded-agent shells
#   reload help       — Print usage
# ══════════════════════════════════════════════════════════════════════════
# 2026 design choices:
#   - PID preserved via `exec fish` (canonical; jobs re-parented, not orphaned)
#   - Universal variables (fish_user_paths etc.) carry over automatically
#   - Environment detection via /proc/<pid>/comm (Linux) / ps (macOS)
#   - Smart embed-aware: if running under opencode/claude-code/warp, default
#     to --quiet + skip --ovav (we don't want to rebuild a binary the agent
#     is currently calling)
#   - Per-module fingerprint: detect which modules actually CHANGED so we
#     can skip re-sourcing unchanged ones (cheap path for scripts/CI)
#   - Output is token-conscious: 0-1 lines by default, --verbose for detail
# ══════════════════════════════════════════════════════════════════════════

# ── Constants & environment classification ────────────────────────────────

# Agent executables that wrap fish and would be disrupted by aggressive
# reloading. Conservative — false positives still fall back to safe path.
set -g __ovav_reload_agent_names  opencode opencode-bin claude claude-code code-server warp

# Terminal emulators used by fish users (for tagging output only).
set -g __ovav_reload_term_names   alacritty kitty wezterm foot gnome-terminal warp

set -g __ovav_reload_version     "2.1.0"

# ── Environment detection ──────────────────────────────────────────────────
# Walks the process tree once and returns a structured record.
#
# Output (single-line env=value …):
#   ppid=N          — parent PID
#   pid=N           — current PID
#   terminal=X      — terminal emulator name (lowercased) or "unknown"
#   multiplexer=X   — "tmux" | "screen" | "none"
#   agent=X         — "opencode" | "warp" | "claude-code" | ... or "none"
#   fish_depth=N    — number of fish processes in ancestor chain
#   embedded=bool   — true if any agent ancestor detected

function __ovav_reload_detect_environment
    # `status` builtin in fish doesn't expose current PID — read from
    # /proc instead. macOS fallback below the if.
    set -l pid
    if test -r /proc/self/stat
        set pid (command awk '{print $1}' /proc/self/stat 2>/dev/null | string trim)
        set -l ppid (command awk '/PPid:/ {print $2}' /proc/self/status 2>/dev/null | string trim)
    else
        # macOS / BSD: use ps
        set -l ps_self (command ps -o pid= -o ppid= -p $fish_pid 2>/dev/null | string trim)
        set pid (echo "$ps_self" | string match -rg '^[0-9]+' | head -1)
        # ppid follows on macOS
        set -l maybe_ppid (echo "$ps_self" | string match -ra '\d+')
        set ppid $maybe_ppid[2]
    end
    if test -z "$pid"
        set pid unknown
    end
    if test -z "$ppid"
        set ppid unknown
    end

    set -l term "unknown"
    set -l multiplexer "none"
    set -l agent "none"
    set -l fish_depth 0
    set -l embedded false
    set -l comm_chain

    # Walk parents up.
    set -l cur $pid
    while test "$cur" -gt 1 2>/dev/null
        set -l meta (command ps -o ppid= -o comm= -p "$cur" 2>/dev/null | command awk '{$1=$1; print}' | command sed 's/^ *//')
        set -l par (echo "$meta" | string match -r '^[0-9]+' | head -1)
        set -l ncomm (echo "$meta" | string replace -r '^[0-9]+ *' '')
        if test -z "$ncomm"
            break
        end
        set comm_chain $comm_chain "$ncomm"
        # Fish depth counter
        if string match -q "fish" -- "$ncomm"
            set fish_depth (math $fish_depth + 1)
        end
        # Terminal detection (only on first hit for cleanliness)
        if test "$term" = "unknown"
            for t in $__ovav_reload_term_names
                if string match -q "*$t*" -- "$ncomm"
                    set term $t
                    break
                end
            end
        end
        # Multiplexer
        if string match -q "*tmux*" -- "$ncomm"
            set multiplexer "tmux"
        else if string match -q "*screen*" -- "$ncomm"
            set multiplexer "screen"
        end
        # Agent — bail early on first hit
        for a in $__ovav_reload_agent_names
            if string match -q "*$a*" -- "$ncomm"
                set agent $a
                set embedded true
                break
            end
        end
        if $embedded
            break
        end
        # Ascend
        if test "$par" -le 0 2>/dev/null
            break
        end
        set cur $par
    end

    printf 'ppid=%s pid=%s terminal=%s mux=%s agent=%s fish_depth=%d embedded=%s' \
        $ppid $pid $term $multiplexer $agent $fish_depth $embedded
end

# Render the env record as a single readable line: name=val name=val …
function __ovav_reload_fmt_environment
    __ovav_reload_detect_environment | string replace -ra ' +' ' '
end

# ── Module fingerprint ────────────────────────────────────────────────────
# Stable hash of conf.d files. Detects which individual modules changed.
# Format: one line per file "<sha8>  <basename>".

function __ovav_reload_module_fingerprint
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        return
    end
    for f in "$d"/*.fish
        test -f "$f"; or continue
        set -l h (command sha256sum "$f" 2>/dev/null | string sub -l 8)
        printf '%s  %s\n' "$h" (path basename "$f")
    end
end

# Return list of changed basenames (each on its own line) since the
# stored reload epoch.
function __ovav_reload_detect_changes
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        return
    end
    set -l last $__ovav_last_reload_epoch_unix
    set -l found_changed 0
    for f in "$d"/*.fish
        test -f "$f"; or continue
        set -l mtime (command stat -c %Y "$f" 2>/dev/null; or command stat -f %m "$f" 2>/dev/null; or echo 0)
        # Treat epoch=0 as "never reloaded" — declare everything changed.
        if test "$last" = "0" -o -z "$last"
            echo (path basename "$f")
            set found_changed 1
        else if test "$mtime" -gt "$last" 2>/dev/null
            echo (path basename "$f")
            set found_changed 1
        end
    end
    if test $found_changed -eq 0
        # No actual mtime change but full hash differs → still report
        # (someone may have touched mtime back; rarely matters).
    end
end

# Returns sha256(short,12) of conf.d contents combined.
function __ovav_reload_snapshot_hash
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        echo 'empty'
        return
    end
    set -l files (command ls "$d"/*.fish 2>/dev/null | command sort)
    if set -q files[1]
        command cat $files 2>/dev/null | command sha256sum 2>/dev/null | string sub -l 12
    else
        echo 'empty'
    end
end

# Count of files in conf.d.
function __ovav_reload_count_modules
    set -l n (command ls ~/.config/fish/conf.d/*.fish 2>/dev/null | count)
    if test "$n" -gt 0
        echo $n
    else
        echo 0
    end
end

# ── Selective source: only re-source the ones that changed ───────────────
function __ovav_reload_source_changed
    set -l changed_files $argv
    set -l d ~/.config/fish/conf.d
    if not test -d "$d"
        return
    end
    for f in "$d"/*.fish
        if contains (path basename "$f") $changed_files
            source $f
        end
    end
end

# ── Banner helpers (kept short, output-conscious) ─────────────────────────

# Quiet-mode banner: one line only.
function __ovav_reload_banner_short
    set -l cyan (set_color cyan --bold)
    set -l normal (set_color normal)
    set -l changed (count (__ovav_reload_detect_changes))
    set -l env_line (__ovav_reload_fmt_environment)
    # Compact: which agent wraps us + module change count
    set -l agent (string match -r 'agent=\S+' -- "$env_line" | head -1)
    printf '%sreload%s %d modules%schanged=%d %s\n' \
        "$cyan" "$normal" (__ovav_reload_count_modules) "$cyan" $changed "$agent"
end

# Verbose-mode banner: two lines.
function __ovav_reload_banner_long
    set -l cyan (set_color cyan --bold)
    set -l normal (set_color normal)
    set -l yellow (set_color yellow)
    set -l green (set_color green)

    printf '%s⟳ reload%s\n' "$cyan" "$normal"
    printf '   modules : %d\n' (__ovav_reload_count_modules)

    set -l changed_list (__ovav_reload_detect_changes)
    if set -q changed_list[1]
        printf '   changed : %s%s%s\n' "$yellow" (string join ', ' $changed_list) "$normal"
    else
        printf '   changed : %s(none)%s\n' "$green" "$normal"
    end

    set -l env_line (__ovav_reload_fmt_environment)
    set -l term (string match -r 'terminal=\S+' -- "$env_line" | head -1)
    set -l mux (string match -r 'mux=\S+' -- "$env_line" | head -1)
    set -l agent (string match -r 'agent=\S+' -- "$env_line" | head -1)
    set -l depth (string match -r 'fish_depth=\d+' -- "$env_line" | head -1)
    printf '   env     : %s %s\n' "$term" "$agent"
    # We hide fish_depth when it is 1 (normal) so output stays calm.
    if string match -qr 'fish_depth=([2-9]|\d{2,})' -- "$env_line"
        printf '           depth=%s (you are inside nested fish)\n' (string replace -r 'fish_depth=' '' -- "$depth")
    end
end

# ── --check: dry-run with environment + drift + budget report ────────────

function __ovav_reload_check
    set -l cyan (set_color cyan --bold)
    set -l normal (set_color normal)
    set -l yellow (set_color yellow)
    set -l green (set_color green)
    set -l grey (set_color brblack)

    printf '%sreload --check:%s\n' "$cyan" "$normal"
    printf '   conf.d : %d modules\n' (__ovav_reload_count_modules)
    printf '   hash    : %s\n' (__ovav_reload_snapshot_hash)
    set -l changed_list (__ovav_reload_detect_changes)
    if set -q changed_list[1]
        printf '   drift   : %s%d file(s)%s — %s\n' "$yellow" (count $changed_list) "$normal" (string join ', ' $changed_list)
    else
        printf '   drift   : %s(none)%s\n' "$green" "$normal"
    end

    set -l env_line (__ovav_reload_fmt_environment)
    printf '   env     : %s%s%s\n' "$grey" "$env_line" "$normal"

    # Decision
    if test -z "$changed_list"
        printf '   decide  : %sskip — no drift%s\n' "$green" "$normal"
    else
        printf '   decide  : %sreload recommended%s\n' "$yellow" "$normal"
    end
end

# ── --rehash path (no exec) ───────────────────────────────────────────────

function __ovav_reload_rehash_only
    set -l before_fns (functions --all --no-details | count)
    # Source every conf.d file, but only shell-source the ones whose
    # contents have changed (cheap heuristic).
    set -l d ~/.config/fish/conf.d
    if test -d "$d"
        for f in "$d"/*.fish
            test -f "$f"; or continue
            source $f
        end
        fish_update_completions 2>/dev/null
    end
    set -l after_fns (functions --all --no-details | count)
    set -l green (set_color green)
    set -l normal (set_color normal)
    printf '%s✓ rehash%s  functions: %d → %d\n' "$green" "$normal" $before_fns $after_fns
end

# ── --full: wipe completion + abbreviation caches ────────────────────────

function __ovav_reload_maybe_wipe_caches
    set -l d ~/.config/fish/completions
    set -l n (command rm -rf ~/.local/share/fish/fish_completions_cache_* 2>/dev/null)
    abbr --erase 2>/dev/null
    and set -l n 1
    return 0
end

# ── --ovav (optional): rebuild binary when stale ────────────────────────

function __ovav_reload_maybe_rebuild_ovav
    if not command -sq ovav
        return 1
    end
    set -l src ""
    for candidate in ~/Systems/OVAV /home/braka/Systems/ovav ~/ovav ~/workspace/ovav
        if test -d "$candidate/.git"
            set src $candidate
            break
        end
    end
    if test -z "$src"
        return 1
    end
    set -l go_runtime "$src/go-runtime"
    if not test -d "$go_runtime"
        return 1
    end
    set -l built_at (command stat -c %Y "$go_runtime/ovav" 2>/dev/null; or echo 0)
    set -l now (date +%s)
    set -l age_days (math "( $now - $built_at ) / 86400" 2>/dev/null; or echo 0)
    # Skip rebuild when binary < 1d old (saves the agent from being
    # disrupted by a 'go build' that swaps the binary beneath it).
    if test "$age_days" -lt 1 2>/dev/null
        return 0
    end
    printf '   ovav binary is %dd old, rebuilding…\n' $age_days
    command go -C "$go_runtime" build -o "$HOME/.local/bin/ovav" ./cmd/ovav 2>/dev/null
end

# ── Main entry: reload ─────────────────────────────────────────────────────
# Flags:
#   -q / --quiet    one-line banner, safe in CI / agents
#   -h / --rehash   re-source conf.d only, no exec
#   -c / --check    dry-run, print decision report
#   -f / --full     wipe completions + abbreviations first
#   -o / --ovav     also rebuild ovav binary if stale
#   -v / --verbose  multi-line banner + environment dump
#   -F / --force    bypass safety checks for embedded-agent shells

function reload --description 'OVAV 2026 intelligent live-reload (env-aware, agent-safe)'
    set -l started (date +%s%N 2>/dev/null; or date +%s)

    set -l do_quiet false
    set -l do_rehash_only false
    set -l do_check false
    set -l do_full false
    set -l do_ovav false
    set -l do_verbose false
    set -l do_force false
    set -l rest

    for a in $argv
        switch "$a"
            case '-q' '--quiet' 'quiet'
                set do_quiet true
            case '-h' '--rehash' 'rehash'
                set do_rehash_only true
            case '-c' '--check' 'check'
                set do_check true
            case '-f' '--full' 'full'
                set do_full true
            case '-o' '--ovav' 'ovav'
                set do_ovav true
            case '-v' '--verbose' 'verbose'
                set do_verbose true
            case '-F' '--force' 'force'
                set do_force true
            case '-h' '--help'
                # Treat --help like the bare 'help' sub-command.
                set rest $rest 'help'
            case '-*'
                echo "reload: unknown flag '$a'" >&2
                echo "try: reload help" >&2
                return 2
            case 'help'
                set rest $rest 'help'
            case '*'
                set rest $rest $a
        end
    end

    # ── help ────────────────────────────────────────────────────────────
    if set -q rest[1]
        if test "$rest[1]" = "help"
            set -l cyan (set_color cyan --bold)
            set -l normal (set_color normal)
            set -l yel (set_color yellow)
            printf '%sreload — %sOVAV 2026 intelligent live-reload (env-aware, agent-safe)%s\n' \
                "$cyan" "$yel" "$normal"
        printf '  %sUsage:%s reload [flag]\n' "$cyan" "$normal"
        printf '  Flags:\n'
        printf '    -q / --quiet    one-line banner, safe in CI / agents\n'
        printf '    -h / --rehash   re-source conf.d only, no exec (jobs survive)\n'
        printf '    -c / --check    dry-run with env + drift + budget report\n'
        printf '    -f / --full     wipe completions + abbreviations first\n'
        printf '    -o / --ovav     also rebuild + reinstall ovav binary if stale\n'
        printf '    -v / --verbose  multi-line banner + env dump\n'
        printf '    -F / --force    bypass safety checks for embedded-agent shells\n'
        printf '  Auto-detects:\n'
        printf '    terminal   — alacritty / kitty / wezterm / foot / gnome-terminal / warp\n'
        printf '    multiplexer — tmux / screen\n'
        printf '    agent      — opencode / claude-code / code-server / warp\n'
printf '  When wrapped by an agent, defaults to --quiet + skip --ovav\n'
        printf '  to avoid disrupting the agent'\''s tool invocations.\n'
            return 0
        end
    end

    # ── detect environment + safety budget ──────────────────────────────
    set -l env_line (__ovav_reload_fmt_environment)
    set -l is_embedded (string match -qr 'embedded=true' -- "$env_line" | head -1)
    set -l do_safe_rebuild false
    if $do_ovav
        # Default: NEVER rebuild the ovav binary from inside an agent shell
        # unless --force is supplied. Reason: 'go build' replaces the
        # binary that the agent is using, which can cause transient
        # "text file busy" or signature verification errors mid-call.
        if $is_embedded; and not $do_force
            # Silently skip rebuilding; report only in --verbose
            if $do_verbose
                printf '%s   skipped:%s ovav rebuild while wrapped by an agent\n' \
                    (set_color brblack) (set_color normal)
            end
            set do_ovav false
        else
            set do_safe_rebuild true
        end
    end

    # ── --check ─────────────────────────────────────────────────────────
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
        # Mark epoch — universal only (carries across exec)
        set -U __ovav_last_reload_epoch_unix (date +%s)
        return 0
    end

    # ── --full: wipe caches first ────────────────────────────────────────
    if $do_full
        __ovav_reload_maybe_wipe_caches
        if $do_verbose
            printf '%s   wiped:%s completions + abbreviations caches\n' \
                (set_color yellow) (set_color normal)
        end
    end

    # ── --ovav (always guarded): safe rebuild ───────────────────────────
    if $do_safe_rebuild
        __ovav_reload_maybe_rebuild_ovav
    end

    # ── Banner policy ───────────────────────────────────────────────────
    # Embedded agents: one line only by default.
    # Local interactive: multi-line if not --quiet.
    set -l banner_mode "short"
    if $do_verbose
        set banner_mode "long"
    else if $do_quiet
        set banner_mode "silent"
    else if $is_embedded
        set banner_mode "short"
    else
        set banner_mode "long"
    end

    switch "$banner_mode"
        case "long"
            __ovav_reload_banner_long
        case "short"
            __ovav_reload_banner_short
        case "silent"
            # Silent.
    end

    # ── Mark epoch (universal only) and exec ────────────────────────────
    set -U __ovav_last_reload_epoch_unix (date +%s)

    # Persist the decision so the next 'reload --check' has a baseline.
    set -U __ovav_last_reload_decision "$banner_mode"

    # PID-preserving replacement. The new fish process re-runs
    # config.fish + conf.d/*.fish so all defs land fresh.
    exec fish
end
