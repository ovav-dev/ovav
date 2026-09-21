# OVAV — WezTerm git status integration
# Canonical source: config/fish/25-ovav-wezterm-git.fish
# Deploy target: ~/.config/fish/conf.d/25-ovav-wezterm-git.fish
# Purpose: Show git branch in WezTerm tab title via OSC 0/2 escape sequences.
#          Only activates inside OVAV repo worktrees.

if status is-interactive
    if set -q WEZTERM_PANE
        function __ovav_git_title --on-variable PWD
            set -l git_dir (git rev-parse --git-dir 2>/dev/null)
            if test $status -eq 0
                set -l branch (git branch --show-current 2>/dev/null)
                if test -n "$branch"
                    printf "\033]0;ovav/%s\007" "$branch"
                end
            end
        end
    end
end
