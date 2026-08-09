# OVAV Terminal Automation Hook
# Canonical source: config/fish/90-ovav-terminal-auto.fish
# Deploy target: ~/.config/fish/conf.d/90-ovav-terminal-auto.fish
# Purpose: Silent, gated, low-overhead terminal maintenance.
#          Runs auto_maintain.py in background at most once per day.
#          No manual command required.

if status is-interactive
    if test "$OVAV_TERMINAL" = "wezterm" -o -n "$WEZTERM_PANE"
        set -l ovav_auto "$HOME/.local/share/ovav-terminal/auto_maintain.py"
        set -l ovav_stamp "$HOME/.local/state/ovav-terminal/auto-last-run"

        if test -x "$ovav_auto"
            set -l should_run 0

            if not test -f "$ovav_stamp"
                set should_run 1
            else
                set -l now (date +%s)
                set -l last (stat -c %Y "$ovav_stamp" 2>/dev/null; or echo 0)
                set -l age (math "$now - $last")
                if test "$age" -gt 86400
                    set should_run 1
                end
            end

            if test "$should_run" = "1"
                command python3 "$ovav_auto" maintain >/dev/null 2>&1 &
                disown
            end
        end
    end
end
