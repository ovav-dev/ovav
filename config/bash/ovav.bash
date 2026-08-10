# ══════════════════════════════════════════════════════════════════════════
# OVAV Bash Functions v1.0
# ══════════════════════════════════════════════════════════════════════════
# Source this file from ~/.bashrc or ~/.bash_profile:
#   [ -f ~/Systems/OVAV/config/bash/ovav.bash ] && source ~/Systems/OVAV/config/bash/ovav.bash
#
# Worktree shortcuts: owc/obc/ows/owb/owd/owx
# Retrocompatibles:  wt/tsk (mapped to owc/obc)
# ══════════════════════════════════════════════════════════════════════════

OVAV_ROOT="${OVAV_ROOT:-$HOME/Systems/OVAV}"
SCRIPT="$OVAV_ROOT/tools/agent_runtime/position.py"

# ── OVAV smart dispatch ──────────────────────────────────────────────────
# ovav with no args → cd to OVAV root
# ovav with args    → invoke the ovav binary/command
ovav() {
    if [[ $# -eq 0 ]]; then
        cd "$OVAV_ROOT"
    else
        command ovav "$@"
    fi
}

# ── Worktree shortcuts (v3.2 standard) ─────────────────────────────────

owc() {
    # ovav worktree create <feature>
    local output
    output=$(python3 "$SCRIPT" "$@" 2>&1)
    echo "$output"
    local wt_path
    wt_path=$(echo "$output" | grep -oP 'CDPATH:\K.*' | head -1)
    if [[ -n "$wt_path" && -d "$wt_path" ]]; then
        cd "$wt_path"
    fi
}

obc() {
    # ovav branch create <feature> (solo rama, sin worktree)
    python3 "$SCRIPT" --tsk "$@"
}

ows() {
    # ovav worktree status
    python3 "$SCRIPT"
}

owb() {
    # ovav worktree back (volver a develop)
    local output
    output=$(python3 "$SCRIPT" back develop 2>&1)
    echo "$output"
    local cd_path
    cd_path=$(echo "$output" | grep -oP 'CDPATH:\K.*' | head -1)
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
    fi
}

owd() {
    # ovav worktree done (merge -> develop + cleanup)
    local output
    output=$(python3 "$SCRIPT" done 2>&1)
    echo "$output"
    local cd_path
    cd_path=$(echo "$output" | grep -oP 'CDPATH:\K.*' | head -1)
    if [[ -n "$cd_path" && -d "$cd_path" ]]; then
        cd "$cd_path"
    fi
}

owx() {
    # ovav worktree clean (limpiar huerfanos)
    python3 "$SCRIPT" clean --execute
}

# ── Retrocompatibles (deprecados, mapean a owc/obc) ────────────────────

wt() {
    # [retro] -> owc
    owc "$@"
}

tsk() {
    # [retro] -> obc
    obc "$@"
}
