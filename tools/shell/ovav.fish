# ══════════════════════════════════════════════════════════════════════════
# OVAV Unified Shell Config v7.0 — Fish
# ══════════════════════════════════════════════════════════════════════════
#
# Consolidates:
#   35-ovav-wt-tsk.fish (Python v3.2) + ovav.fish (Go OWS v1.0)
#
# Backend: go-runtime/cmd/ovav/ → /home/braka/.local/bin/ovav
# Visual status: fish-pure (no deps beyond git)
#
# Install:
#   cp tools/shell/ovav.fish ~/.config/fish/conf.d/ovav.fish
#   source ~/.config/fish/conf.d/ovav.fish
#
# After installing, REMOVE the old file:
#   rm ~/.config/fish/conf.d/35-ovav-wt-tsk.fish
#
# ┌──────┬──────────┬─────────────────────────────────────────────────┐
# │ Cmd  │ Alias    │ Description                                    │
# ├──────┼──────────┼─────────────────────────────────────────────────┤
# │ owc  │ create   │ Create worktree + auto-cd                      │
# │ owd  │ done     │ Merge → develop + cleanup + auto-cd back       │
# │ owl  │ list     │ List worktrees (--history, --json)             │
# │ owx  │ route    │ Route commits (cherry-pick/patch/hotfix/...)   │
# │ owa  │ abort    │ Abort in-progress operation                    │
# │ owr  │ rescue   │ Rescue lost branches/worktrees from reflog      │
# │ ows  │ sync     │ Sync remotes + maintenance + prune             │
# │ owv  │ verify   │ Full verification pipeline                      │
# │ owu  │ update   │ Fetch + rebase onto origin/develop              │
# │ owlk │ lock     │ Lock worktree (multi-agent coordination)        │
# │ owm  │ move     │ Move worktree to new path                       │
# │owclean│ clean  │ Clean orphaned worktrees + stale branches        │
# ├──────┼──────────┼─────────────────────────────────────────────────┤
# │ owcp │ cherry   │ Cherry-pick → develop (via ovav route)          │
# │ obc  │ branch   │ Create branch only (no worktree)                │
# │ owsv │ vstatus  │ Visual status display (fish-pure)               │
# │ owst │ vstatus  │ Visual status (mnemonic alias → owsv)           │
# │ wt   │ [legacy] │ Deprecated — use owc                            │
# │ tsk  │ [legacy] │ Deprecated — use obc                            │
# └──────┴──────────┴─────────────────────────────────────────────────┘
#
# ══════════════════════════════════════════════════════════════════════════


# ══════════════════════════════════════════════════════════════════════════
# §1  INTERNAL HELPERS
# ══════════════════════════════════════════════════════════════════════════

# ── __ovav_resolve_main_root: find main repo root from any worktree ─────
#
# FIX: Previous implementation used `string replace -r '/\.git$' ''` which
# FAILS when git-common-dir is relative (".git" — no leading /).
# Now uses realpath + dirname, with defensive .git/worktrees/ handling.
#
function __ovav_resolve_main_root --description "Resolve main OVAV repo root (works from any worktree)"
    # git-common-dir always points to the MAIN .git directory, even from worktrees.
    # - In main repo: returns ".git" (relative) or "/path/to/repo/.git" (absolute)
    # - In worktree:  returns "/path/to/main/repo/.git" (always absolute)
    set -l git_common (git rev-parse --git-common-dir 2>/dev/null)
    if test -z "$git_common"
        # Last resort: show-toplevel (may return worktree root, not main)
        git rev-parse --show-toplevel 2>/dev/null
        return
    end

    # Resolve relative paths to absolute (e.g., ".git" → "/home/user/repo/.git")
    if test "$git_common" != "/*"
        set git_common (realpath "$git_common" 2>/dev/null)
    end
    if test -z "$git_common"
        return 1
    end

    # Defensive: if path contains .git/worktrees/<name>, strip to repo root.
    # --git-common-dir SHOULD point to the main .git, but some edge cases
    # (or future git versions) might return worktree-specific paths.
    if string match -q '*/.git/worktrees/*' -- $git_common
        string replace -r '/\.git/worktrees/.*$' '' -- $git_common
    else
        dirname "$git_common"
    end
end


# ── __ovav_ws: thin Go OWS wrapper ──────────────────────────────────────
function __ovav_ws --description "Internal: call ovav worktree subcommand"
    ovav worktree $argv
end


# ══════════════════════════════════════════════════════════════════════════
# §2  CORE 12 COMMANDS (Go OWS backend)
# ══════════════════════════════════════════════════════════════════════════

# ── owc: create worktree + auto-cd ──────────────────────────────────────
function owc --description "Create OVAV worktree — auto-cd into it. --profile <type>"
    if test (count $argv) -eq 0
        echo "Usage: owc <task-name> [--profile <type>]"
        echo "  owc task27                              → feature worktree"
        echo "  owc --profile hotfix fix23              → hotfix worktree"
        echo "  owc --profile release v3                → release worktree"
        echo "  owc --profile emergency critical-fix     → emergency worktree"
        echo ""
        echo "Profiles: feature(default) refactor docs spike research migration"
        echo "          enterprise hotfix release patch emergency"
        return 1
    end

    set -l task_name ""
    set -l profile ""

    # Parse --profile/-p flag
    set -l i 1
    while test $i -le (count $argv)
        switch $argv[$i]
            case --profile -p
                set i (math $i + 1)
                if test $i -le (count $argv)
                    set profile $argv[$i]
                end
            case '*'
                if test -z "$task_name"
                    set task_name $argv[$i]
                end
        end
        set i (math $i + 1)
    end

    if test -z "$task_name"
        echo "Error: task name required"
        return 1
    end

    # Execute with optional profile
    set -l output
    if test -n "$profile"
        set output (ovav worktree create "$task_name" --profile "$profile" 2>&1)
    else
        set output (ovav worktree create "$task_name" 2>&1)
    end
    printf '%s\n' $output

    # Extract worktree path from structured output (WORKTREE:<path>)
    set -l wt_path ""
    for line in $output
        set -l match (string match -r '^WORKTREE:(.+)$' -- $line)
        if test (count $match) -ge 2
            set wt_path (string trim $match[2])
            break
        end
    end

    if test -n "$wt_path" -a -d "$wt_path"
        cd "$wt_path"
    end
end


# ── owd: merge + cleanup + auto-cd back to root ─────────────────────────
#
# FIX: Previous implementation cd'd unconditionally after `ovav worktree done`.
# If done fails (conflict, validation error), the worktree directory may be
# invalid or the user may need to stay and resolve issues. Now captures
# exit code before changing directory.
#
function owd --description "Merge worktree → develop + cleanup — auto-cd back to root"
    # Resolve main repo root BEFORE merge (worktree deleted after success)
    set -l main_root (__ovav_resolve_main_root)

    # Execute merge + cleanup — forward any args
    ovav worktree done $argv
    set -l owd_status $status

    # Only cd back on success — stay put on failure
    if test $owd_status -ne 0
        echo (set_color yellow)"⚠️  owd failed (exit $owd_status). Staying in current directory."(set_color normal)
        return $owd_status
    end

    # Auto-cd back to main repo root
    if test -n "$main_root" -a -d "$main_root"
        cd "$main_root"
        echo "📍 Back to root: $main_root"
    end
end


# ── owl: list worktrees ─────────────────────────────────────────────────
function owl --description "List worktrees with state + conflict predictions. --history for audit, --json"
    ovav worktree list $argv
end


# ── owx: route commits ──────────────────────────────────────────────────
function owx --description "Route commits: cherry-pick / patch / hotfix / emergency"
    ovav worktree route $argv
end


# ── owa: abort in-progress operation ─────────────────────────────────────
function owa --description "Abort in-progress cherry-pick / rebase / merge"
    ovav worktree abort $argv
end


# ── owr: rescue lost work ────────────────────────────────────────────────
function owr --description "Rescue lost branches, worktrees, or commits from reflog"
    ovav worktree rescue $argv
end


# ── ows: sync remotes + maintenance + prune ──────────────────────────────
# NOTE: In the legacy Python file, `ows` was "visual status". That semantic
# is now `owsv` / `owst` (§4). `ows` maps to Go-backed sync as designed.
function ows --description "Sync all remotes + git maintenance + prune stale worktrees"
    ovav worktree sync $argv
end


# ── owv: verify worktree ─────────────────────────────────────────────────
function owv --description "Full verification pipeline on current worktree"
    ovav worktree verify $argv
end


# ── owu: update worktree (fetch + rebase) ────────────────────────────────
function owu --description "Update worktree: fetch + rebase onto origin/develop"
    ovav worktree update $argv
end


# ── owlk: lock worktree ──────────────────────────────────────────────────
function owlk --description "Lock worktree to prevent modifications (multi-agent coord)"
    ovav worktree lock $argv
end


# ── owm: move worktree ───────────────────────────────────────────────────
function owm --description "Move worktree to new path (preserves git link)"
    ovav worktree move $argv
end


# ── owclean: clean orphaned worktrees ────────────────────────────────────
function owclean --description "Clean orphaned worktrees + stale branches. --dry-run to preview"
    ovav worktree clean $argv
end


# ══════════════════════════════════════════════════════════════════════════
# §3  LEGACY / CROSSOVER COMMANDS
# ══════════════════════════════════════════════════════════════════════════

# ── owcp: cherry-pick commit → develop (Go route backend) ───────────────
# Replaces the fish-pure cherry-pick from 35-ovav-wt-tsk.fish.
# Now routes through `ovav worktree route` for atomic handling.
function owcp --description "Cherry-pick commit to develop bus (via ovav worktree route)"
    if test (count $argv) -lt 1
        echo "owcp <commit-hash> [...]  — cherry-pick to develop without merging entire branch"
        echo "  owcp abc123            → cherry-pick abc123 to develop"
        echo "  owcp abc123 def456     → cherry-pick multiple commits"
        return 1
    end
    ovav worktree route --mode cherry-pick --target develop $argv
end


# ── obc: create branch only (no worktree) ────────────────────────────────
# Legacy command from 35-ovav-wt-tsk.fish. Routes through Go backend with
# --branch-only flag. If the backend doesn't support this yet, it will
# return an error — use owc for full worktree creation in the meantime.
function obc --description "Create branch without worktree (routes through Go OWS)"
    if test (count $argv) -eq 0
        echo "obc <name>  — create branch without worktree"
        return 1
    end
    ovav worktree create --branch-only $argv
end


# ── wt: [DEPRECATED] alias → owc ────────────────────────────────────────
function wt --description "[DEPRECATED] Use owc instead"
    echo (set_color yellow)"⚠️  'wt' is deprecated. Use 'owc'."(set_color normal)
    owc $argv
end


# ── tsk: [DEPRECATED] alias → obc ───────────────────────────────────────
function tsk --description "[DEPRECATED] Use obc instead"
    echo (set_color yellow)"⚠️  'tsk' is deprecated. Use 'obc'."(set_color normal)
    obc $argv
end


# ══════════════════════════════════════════════════════════════════════════
# §4  VISUAL STATUS (fish-pure, no deps beyond git)
# ══════════════════════════════════════════════════════════════════════════

# ── __ovav_visual_status: internal display ───────────────────────────────
# Rich worktree dashboard with ahead/behind, dirty check, active marker.
# Adapted from 35-ovav-wt-tsk.fish with robust parsing.
function __ovav_visual_status --description "Internal: visual worktree status dashboard"
    # Bail out if not in a git repo
    set -l current_branch (git branch --show-current 2>/dev/null)
    if test -z "$current_branch"
        echo (set_color red)"Not in a git repository."(set_color normal)
        return 1
    end

    # Header
    echo
    echo (set_color brwhite)"  OVAV Worktrees — "(set_color cyan)"$current_branch"(set_color brwhite)" · develop"(set_color normal)
    echo (set_color brblack)"  ─────────────────────────────────────────────────────────"(set_color normal)

    # Parse worktree list
    git worktree list 2>/dev/null | while read -l line
        # Skip lines without branch brackets (bare repo entries, etc.)
        set -l has_branch (string match -r '\[.+\]' -- $line)
        if test -z "$has_branch"
            continue
        end

        # Parse: /path  hash  [branch]
        # Using regex for robustness (paths could have varying spacing)
        set -l path (string replace -r '^\s*(\S+)\s+.*' '$1' -- $line)
        set -l branch (string replace -r '.*\[(.+)\].*' '$1' -- $line)

        # Active marker — highlight current worktree
        set -l marker " "
        set -l marker_color (set_color brblack)
        if test (basename $PWD) = (basename $path)
            set marker "▶"
            set marker_color (set_color green)
        end

        # Short branch name — strip known prefixes
        set -l short_branch (string replace -r '^(task|feature|fix|hotfix|release|patch|emergency|docs|spike|research|migration|refactor|enterprise)/' '' -- $branch)

        # Ahead/behind vs develop
        set -l ahead (git -C $path rev-list --count develop..$branch 2>/dev/null; or echo "?")
        set -l behind (git -C $path rev-list --count $branch..develop 2>/dev/null; or echo "?")

        # Last commit (truncated to 45 chars)
        set -l last (git -C $path log --oneline -1 2>/dev/null | string shorten -m 45; or echo "(vacío)")

        # Dirty check
        set -l dirty ""
        set -l dirty_count (git -C $path status --short 2>/dev/null | wc -l | string trim)
        if test -n "$dirty_count" -a "$dirty_count" != "0" -a "$dirty_count" != ""
            set dirty (set_color yellow)" ⚠️"(set_color normal)
        end

        printf "  %s%s %-20s %s↑%s ↓%s%s %s%s\n" \
            $marker_color $marker \
            (set_color brwhite)$short_branch(set_color normal) \
            (set_color brblack) \
            $ahead $behind \
            (set_color normal) \
            $last $dirty
    end

    # Footer
    echo (set_color brblack)"  ─────────────────────────────────────────────────────────"(set_color normal)
    set -l task_count (git worktree list 2>/dev/null | grep -c 'task/' 2>/dev/null; or echo "?")
    set -l task_count (string trim $task_count)
    echo "  "(set_color brblack)"$task_count task worktrees"(set_color normal)
    echo
end


# ── owsv: visual status (primary) ────────────────────────────────────────
# "s" = status, "v" = visual. Rich dashboard with ahead/behind + dirty.
function owsv --description "Visual status of all worktrees (fish-pure dashboard)"
    __ovav_visual_status
end


# ── owst: visual status (mnemonic alias) ─────────────────────────────────
# "st" = status tree. Alias for owsv.
function owst --description "Visual status of all worktrees (alias for owsv)"
    __ovav_visual_status
end
