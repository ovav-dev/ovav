# ══════════════════════════════════════════════════════════════════════
# OVAV Worktree Commands v6.0 — Go Native
# ══════════════════════════════════════════════════════════════════════
#
# 12 comandos. Go OWS runtime. Python workflow ELIMINADO.
#
# owc, owd, owv, owl, owu, owa, owr, owx, owlk, owm, owcp, ows.
# owclean (limpieza), wt (deprecado).
#
# Instalación: copiar a ~/.config/fish/conf.d/35-ovav-wt-tsk.fish
# Reemplaza el archivo existente COMPLETO.
# ══════════════════════════════════════════════════════════════════════

# ── Go OWS helpers ───────────────────────────────────────────────────

function __ovav_ws
    ovav worktree $argv
end

# ══════════════════════════════════════════════════════════════════════
# owc — Crear worktree desde develop (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owc --description 'Crear worktree (Go OWS)'
    if test (count $argv) -lt 1
        echo "owc <name> [--profile <type>]  — crea worktree desde develop"
        echo "  Profiles: feature, hotfix, release, patch, emergency, refactor, docs, spike, research, migration, enterprise"
        return 1
    end
    __ovav_ws create $argv
end

# ══════════════════════════════════════════════════════════════════════
# owd — Verify → merge → push → cleanup (Go OWS, CEO-gated)
# ══════════════════════════════════════════════════════════════════════

function owd --description 'Merge + cleanup (Go OWS)'
    set current (git branch --show-current 2>/dev/null)
    if test -z "$current"
        echo (set_color red)"owd: no estás en ninguna rama (detached HEAD?)"(set_color normal)
        return 1
    end
    __ovav_ws done $argv
end

# ══════════════════════════════════════════════════════════════════════
# owv — Validación completa pre-merge (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owv --description 'Validación pre-merge (go test + vet + fmt + validators)'
    __ovav_ws verify $argv
end

# ══════════════════════════════════════════════════════════════════════
# owl — Lista worktrees con estado, ownership, conflictos (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owl --description 'Lista worktrees (Go OWS)'
    __ovav_ws list $argv
end

# ══════════════════════════════════════════════════════════════════════
# owu — Fetch + rebase worktree (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owu --description 'Fetch + rebase worktree (Go OWS)'
    __ovav_ws update $argv
end

# ══════════════════════════════════════════════════════════════════════
# owa — Abortar operación en curso (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owa --description 'Abortar operación (Go OWS)'
    __ovav_ws abort $argv
end

# ══════════════════════════════════════════════════════════════════════
# owr — Recuperar ramas/worktrees del reflog (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owr --description 'Rescue del reflog (Go OWS)'
    __ovav_ws rescue $argv
end

# ══════════════════════════════════════════════════════════════════════
# owx — Cherry-pick / patch / hotfix / emergency routing (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owx --description 'Route commits: cherry-pick, patch, hotfix, emergency (Go OWS)'
    __ovav_ws route $argv
end

# ══════════════════════════════════════════════════════════════════════
# owlk — Bloquear worktree (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owlk --description 'Lock worktree (Go OWS)'
    __ovav_ws lock $argv
end

# ══════════════════════════════════════════════════════════════════════
# owm — Mover worktree (Go OWS)
# ══════════════════════════════════════════════════════════════════════

function owm --description 'Mover worktree (Go OWS)'
    __ovav_ws move $argv
end

# ══════════════════════════════════════════════════════════════════════
# owclean — Limpiar worktrees huérfanas (Git nativo, reemplaza viejo owx)
# ══════════════════════════════════════════════════════════════════════

function owclean --description 'Limpiar worktrees huérfanas'
    git worktree prune 2>/dev/null
    echo (set_color green)"✓"(set_color normal)" worktrees prune ejecutado"
    echo
    echo (set_color brblack)"  Ramas de trabajo sin worktree:"(set_color normal)
    for prefix in task feature fix hotfix release patch emergency docs spike research migration refactor enterprise
        git branch --list "$prefix/*" 2>/dev/null | string trim | while read -l branch
            set wt (git worktree list 2>/dev/null | grep -F "$branch")
            if test -z "$wt"
                echo "    "(set_color yellow)"$branch"(set_color normal)" — ¿eliminar? git branch -d $branch"
            end
        end
    end
end

# ══════════════════════════════════════════════════════════════════════
# owcp — Cherry-pick un commit al bus develop (fish puro)
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
# ows — Status visual unificado de TODAS las worktrees (fish puro)
# ══════════════════════════════════════════════════════════════════════

function ows --description 'Status visual de worktrees'
    set current (git branch --show-current)

    # Header
    echo
    echo (set_color brwhite)"  OVAV Worktrees — "(set_color cyan)"$current"(set_color brwhite)" · develop"(set_color normal)

    # Separator
    echo (set_color brblack)"  ─────────────────────────────────────────────────────────"(set_color normal)

    git worktree list 2>/dev/null | while read -l line
        set parts (string split ' ' $line)
        set path $parts[1]
        set hash $parts[2]
        set branch (string replace '[' '' (string replace ']' '' $parts[3]))

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
# wt — [deprecado] Alias de owc.
# ══════════════════════════════════════════════════════════════════════

function wt --description '[deprecado] → owc'
    echo (set_color yellow)"⚠️  'wt' está deprecado. Usá 'owc'."(set_color normal)
    owc $argv
end
