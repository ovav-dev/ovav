# OVAV — WezTerm OSC7 integration
# Canonical source: config/fish/20-ovav-wezterm-osc7.fish
# Deploy target: ~/.config/fish/conf.d/20-ovav-wezterm-osc7.fish
# Purpose: Tell WezTerm to set OSC 7 (working directory reporting) so that
#          new tabs/panes open in the same directory as the current pane.

if status is-interactive
    if set -q WEZTERM_PANE
        function __ovav_osc7_cwd --on-variable PWD
            printf "\033]7;file://%s%s\033\\" (hostname) (pwd | string escape --style=url)
        end
    end
end
