# OVAV git workflow aliases — Go runtime v3.0
# Replaces deprecated Python owc/owd wrappers (DEG-010)
# Source: go-runtime/internal/gitflow/workflow.go
#
# Install:
#   cp go-runtime/internal/gitflow/ovav.fish ~/.config/fish/conf.d/ovav.fish
#   source ~/.config/fish/conf.d/ovav.fish

# ── owc: create OVAV worktree for a task ──────────────────────────────
function owc --description "Create OVAV worktree from develop for a task"
    if test (count $argv) -eq 0
        echo "Usage: owc <task-name>"
        echo "Example: owc task25  →  ovav git start tasknext-ceo-task25"
        return 1
    end

    set task_name $argv[1]
    # Auto-prefix with CEO naming convention
    ovav git start "tasknext-ceo-$task_name"
end

# ── owd: merge current worktree → develop + cleanup ───────────────────
function owd --description "Merge current worktree to develop and clean up"
    ovav git merge
end
