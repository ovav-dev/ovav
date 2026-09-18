# OVAV Worktree System — canonical fish entrypoints.
# The Go runtime owns the lifecycle; these names are shell shorthands only.

function owc --description 'Create OVAV worktree from develop'
    if test (count $argv) -lt 1
        echo "Usage: owc <name>"
        return 1
    end

    set output (ovav worktree create $argv 2>&1)
    set exit_code $status
    printf '%s\n' $output
    if test $exit_code -ne 0
        return $exit_code
    end

    for line in $output
        if string match -q 'WORKTREE:*' -- "$line"
            set wt_path (string replace 'WORKTREE:' '' -- "$line")
            if test -d "$wt_path"
                cd "$wt_path"
                echo "→ $wt_path"
            end
            break
        end
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
