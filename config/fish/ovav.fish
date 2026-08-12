# ══════════════════════════════════════════════════════════════════════════
# OVAV Unified Shell Config v7.1 — Fish Functions (conf.d)
# ══════════════════════════════════════════════════════════════════════════
# This file is auto-sourced from ~/.config/fish/config.fish
# via: for f in ~/.config/fish/conf.d/*.fish; source $f; end
#
# All functions defined here are auto-loaded by fish shell.
# ══════════════════════════════════════════════════════════════════════════


# ══════════════════════════════════════════════════════════════════════════
# §OVAV smart dispatch — MUST be last to override PATH binary
# FIX: alias ovav "cd ~/Systems/OVAV" broke "ovav login" → fish expanded
# to "cd ~/Systems/OVAV login" → "Too many args for cd command".
# Solution: function detects arg count and routes correctly.
# ══════════════════════════════════════════════════════════════════════════
function ovav
    if test (count $argv) -eq 0
        cd ~/Systems/OVAV
    else
        command ovav $argv
    end
end
