# OVAV — worktree/branch shortcuts (v3.2 standard: owc/obc/ows/owb/owd/owx)
# Canonical source: config/fish/35-ovav-wt-tsk.fish
# Deploy target: ~/.config/fish/conf.d/35-ovav-wt-tsk.fish
# wt y tsk se mantienen como retrocompatibles, mapean a owc/obc.
set SCRIPT ~/Systems/OVAV/tools/agent_runtime/position.py

# ── PRIMARY (new standard) ──────────────────────────────────────────

function owc --description 'ovav worktree create <feature>'
    set output (python3 $SCRIPT $argv 2>&1)
    echo $output
    set wt_path (echo $output | grep -oP 'CDPATH:\K.*' | head -1)
    if test -n "$wt_path" -a -d "$wt_path"
        cd $wt_path
    end
end

function obc --description 'ovav branch create <feature> (solo rama, sin worktree)'
    python3 $SCRIPT --tsk $argv
end

function ows --description 'ovav worktree status'
    python3 $SCRIPT
end

function owb --description 'ovav worktree back (volver a develop)'
    set output (python3 $SCRIPT back develop 2>&1)
    echo $output
    set cd_path (echo $output | grep -oP 'CDPATH:\K.*' | head -1)
    if test -n "$cd_path" -a -d "$cd_path"
        cd $cd_path
    end
end

function owd --description 'ovav worktree done (merge -> develop + cleanup)'
    set output (python3 $SCRIPT done 2>&1)
    echo $output
    set cd_path (echo $output | grep -oP 'CDPATH:\K.*' | head -1)
    if test -n "$cd_path" -a -d "$cd_path"
        cd $cd_path
    end
end

function owx --description 'ovav worktree clean (limpiar huerfanos)'
    python3 $SCRIPT clean --execute
end

# ── RETROCOMPATIBLES (deprecados, mapean a owc/obc) ──────────────

function wt --description '[retro] -> owc'
    owc $argv
end

function tsk --description '[retro] -> obc'
    obc $argv
end
