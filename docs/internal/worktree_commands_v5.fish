# ══════════════════════════════════════════════════════════════════════
# OVAV Worktree Commands v5.0 — Optimized Set
# ══════════════════════════════════════════════════════════════════════
#
# 5 comandos esenciales + 1 deprecado.
# owc, owcp, ows, owd, owx. Alias: wt (deprecado).
#
# Eliminados: owb, obc, tsk, owl (redundantes o fusionados).
#
# Instalación: copiar a ~/.config/fish/conf.d/35-ovav-wt-tsk.fish
# Reemplaza el archivo existente COMPLETO.
# ══════════════════════════════════════════════════════════════════════

set -g OVAV_REPO_ROOT (git rev-parse --show-toplevel 2>/dev/null)
set -g OVAV_SCRIPT "$OVAV_REPO_ROOT/tools/harnesses/h_git_branch_workflow.py"

# ══════════════════════════════════════════════════════════════════════
# owc — Crear worktree task/<feature> desde develop
# ══════════════════════════════════════════════════════════════════════

function owc --description 'Crear worktree task/<feature>'
    if test (count $argv) -lt 1
        echo "owc <feature>  — crea worktree task/<feature> desde develop"
        return 1
    end
    python3 $OVAV_SCRIPT create $argv
end

# ══════════════════════════════════════════════════════════════════════
# owcp — Cherry-pick un commit al bus develop sin mergear rama entera
# ══════════════════════════════════════════════════════════════════════

function owcp --description 'Cherry-pick al bus develop'
    if test (count $argv) -lt 1
        echo "owcp <commit-hash>  — cherry-pick a develop sin mergear rama entera"
        return 1
    end

    set commit $argv[1]
    set current (git branch --show-current)

    echo (set_color cyan)"▸ OVAV Bus"(set_color normal)"  cherry-pick "(set_color yellow)"$commit"(set_color normal)" → develop"

    git checkout develop 2>/dev/null
    or begin; echo (set_color red)"✗"(set_color normal)" no se pudo checkout develop"; return 1; end

    git pull origin develop 2>/dev/null

    if git cherry-pick $commit 2>/dev/null
        git push origin develop 2>/dev/null
        git checkout $current 2>/dev/null
        echo (set_color green)"✓"(set_color normal)" $commit → develop. Otras worktrees: git pull origin develop"
    else
        echo (set_color red)"✗"(set_color normal)" cherry-pick falló. Conflicto."
        echo "  "(set_color yellow)"↳"(set_color normal)" resolver → git add . → git cherry-pick --continue"
        echo "  "(set_color yellow)"↳"(set_color normal)" abortar → git cherry-pick --abort"
        git checkout $current 2>/dev/null
        return 1
    end
end

# ══════════════════════════════════════════════════════════════════════
# ows — Status visual unificado de TODAS las worktrees
# ══════════════════════════════════════════════════════════════════════

function ows --description 'Status visual de worktrees'
    set current (git branch --show-current)
    set dev_ahead (git rev-list --count develop..origin/develop 2>/dev/null)

    # Header
    echo
    echo (set_color brwhite)"  OVAV Worktrees — "(set_color cyan)"$current"(set_color brwhite)" · develop"(set_color normal)

    # Separator
    echo (set_color brblack)"  ─────────────────────────────────────────────────────────"(set_color normal)

    set count 0
    git worktree list 2>/dev/null | while read -l line
        set parts (string split ' ' $line)
        set path $parts[1]
        set hash $parts[2]
        set branch (string replace '[' '' (string replace ']' '' $parts[3]))

        set count (math $count + 1)

        # Active marker
        set marker " "
        set marker_color (set_color brblack)
        if test (basename $PWD) = (basename $path)
            set marker "▶"
            set marker_color (set_color green)
        end

        # Branch name (short)
        set short_branch (string replace 'task/' '' $branch)

        # Ahead/behind vs develop
        set ahead (git -C $path rev-list --count develop..$branch 2>/dev/null; or echo "0")
        set behind (git -C $path rev-list --count $branch..develop 2>/dev/null; or echo "0")

        # Last commit
        set last (git -C $path log --oneline -1 2>/dev/null | string shorten -m 45)
        if test -z "$last"
            set last "(vacío)"
        end

        # Dirty check
        set dirty ""
        set dirty_count (git -C $path status --short 2>/dev/null | wc -l | string trim)
        if test "$dirty_count" != "0"
            set dirty (set_color yellow)" ⚠️ dirty"(set_color normal)
        end

        # Format: marker branch  ahead/behind  hash  message  dirty
        printf "  %s%s %-20s"(set_color normal)" "(set_color brblack)"↑%-2s ↓%-2s"(set_color normal)"  %s$dirty\n" \
            $marker_color $marker $short_branch $ahead $behind $last
    end

    # Footer
    echo (set_color brblack)"  ─────────────────────────────────────────────────────────"(set_color normal)
    set task_count (git worktree list 2>/dev/null | grep 'task/' | wc -l | string trim)
    echo "  "(set_color brblack)"$task_count task worktrees"(set_color normal)
    echo
end

# ══════════════════════════════════════════════════════════════════════
# owd — Merge → develop + push + cleanup (CEO-gated)
# ══════════════════════════════════════════════════════════════════════

function owd --description 'Merge + cleanup (CEO-gated)'
    set current (git branch --show-current)
    if not string match -q 'task/*' $current
        echo (set_color red)"owd solo funciona desde rama task/*"(set_color normal)
        return 1
    end
    python3 $OVAV_SCRIPT merge $current --confirm
end

# ══════════════════════════════════════════════════════════════════════
# owx — Limpiar worktrees huérfanas
# ══════════════════════════════════════════════════════════════════════

function owx --description 'Limpiar worktrees huérfanas'
    python3 $OVAV_SCRIPT clean --execute
end

# ══════════════════════════════════════════════════════════════════════
# wt — [deprecado] Alias de owc. Mantenido por compatibilidad.
# ══════════════════════════════════════════════════════════════════════

function wt --description '[deprecado] → owc'
    echo (set_color yellow)"⚠️  'wt' está deprecado. Usá 'owc'."(set_color normal)
    owc $argv
end
