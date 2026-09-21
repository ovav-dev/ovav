# OVAV Worktree System — canonical fish entrypoints.
# The Go runtime owns the lifecycle; these names are shell shorthands only.

function owc --description 'Create OVAV worktree from develop'
    if test (count $argv) -lt 1
        echo "Usage: owc <name>"
        return 1
    end

    # Verify ovav is available
    if not command -v ovav > /dev/null 2>&1
        echo "owc: 'ovav' not found — check PATH or reinstall"
        return 1
    end

    # Capture output with 60s timeout to prevent indefinite hangs
    set -l tmpfile (mktemp)
    command timeout 60 ovav worktree create $argv > $tmpfile 2>&1
    set -l exit_code $status

    # Print output regardless of exit code
    if test -s $tmpfile
        cat $tmpfile
    end

    # Handle timeout
    if test $exit_code -eq 124 -o $exit_code -eq 143
        echo "owc: timeout (>60s) — check network or disk"
        rm -f $tmpfile
        return 1
    end

    # Extract WORKTREE path before deleting tmpfile
    set -l wt_path (grep 'WORKTREE:' $tmpfile 2>/dev/null | string replace 'WORKTREE:' '' | string trim)
    rm -f $tmpfile

    if test $exit_code -ne 0
        return $exit_code
    end

    # Change into worktree if it exists
    if test -n "$wt_path" -a -d "$wt_path"
        cd $wt_path
        echo "→ $wt_path"
    else if test -n "$wt_path"
        echo "owc: worktree not found at: $wt_path"
    end
end

function owd --description 'Finalize current OVAV worktree'
    ovav worktree done $argv
end

function owl --description 'List OVAV worktrees'
    ovav worktree list $argv
end

function owv --description 'Verify current OVAV worktree'
    ovav worktree verify $argv
end

function ows --description 'Show intelligent OVAV worktree status'
    set current (git branch --show-current)
    echo
    echo "  OVAV Worktrees — $current · develop"
    echo "  ─────────────────────────────────────────────────────────"

    set wt_path ''
    set wt_branch ''
    for line in (git worktree list --porcelain 2>/dev/null)
        if string match -q 'worktree *' -- "$line"
            if test -n "$wt_path"
                __ovav_ows_render "$wt_path" "$wt_branch"
            end
            set wt_path (string replace 'worktree ' '' -- "$line")
            set wt_branch ''
        else if string match -q 'branch refs/heads/*' -- "$line"
            set wt_branch (string replace 'branch refs/heads/' '' -- "$line")
        else if test "$line" = 'detached HEAD'
            set wt_branch 'HEAD'
        end
    end
    if test -n "$wt_path"
        __ovav_ows_render "$wt_path" "$wt_branch"
    end

    echo "  ─────────────────────────────────────────────────────────"
    echo
end

function __ovav_ows_render
    set path $argv[1]
    set branch $argv[2]
    test -n "$branch"; or set branch 'HEAD'
    set marker ' '
    if test (realpath "$PWD") = (realpath "$path" 2>/dev/null)
        set marker '▶'
    end
    set short_branch (string replace 'task/' '' "$branch")
    set ahead (git -C "$path" rev-list --count develop.."$branch" 2>/dev/null; or echo 0)
    set behind (git -C "$path" rev-list --count "$branch"..develop 2>/dev/null; or echo 0)
    set last (git -C "$path" log --oneline -1 2>/dev/null | string shorten -m 45)
    set dirty_count (git -C "$path" status --short 2>/dev/null | string match -r '.' | count)
    set dirty ''
    if test "$dirty_count" != 0
        set dirty ' ⚠️ dirty'
    end
    printf '  %s%-30s ↑%-2s ↓%-2s  %s%s\n' "$marker" "$short_branch" "$ahead" "$behind" "$last" "$dirty"
end

function owu --description 'Update current OVAV worktree'
    ovav worktree update $argv
end

function owp --description 'Prepare current OVAV worktree'
    ovav worktree prepare $argv
end

function owlk --description 'Lock an OVAV worktree'
    ovav worktree lock $argv
end

function owm --description 'Move an OVAV worktree'
    ovav worktree move $argv
end

function owclean --description 'Clean stale OVAV worktrees'
    ovav worktree clean $argv
end

function owx --description 'Route OVAV worktree changes'
    ovav worktree route $argv
end

function owa --description 'Abort the current OVAV worktree operation'
    ovav worktree abort $argv
end

function owr --description 'Rescue an OVAV worktree'
    ovav worktree rescue $argv
end

function owprep --description 'Prepare OVAV worktree configuration'
    ovav worktree prep $argv
end

function owsuggest --description 'Suggest the next OWS command'
    ovav worktree suggest $argv
end

function own --description 'Nuke an OVAV worktree'
    ovav worktree nuke $argv
end

function wt --description '[deprecated] create an OVAV worktree'
    echo "⚠️  'wt' está deprecado. Usá 'owc'."
    owc $argv
end
