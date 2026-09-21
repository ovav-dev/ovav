# OVAV per-window tmux session policy.
# Each new Alacritty window gets an isolated tmux session with 3 stable windows.
# ALT+1/2/3 (M-1/2/3) switch by NAME, not index — immune to renumbering.
# Session structure: 0=home, 1=ovav, 2=akrynt — ovav selected by default.

if status is-interactive; and not set -q TMUX; and command -q tmux
    set -l ovav_session "alacritty-$fish_pid"
    exec tmux new-session -s "$ovav_session" -n home -c "$HOME" \; \
         new-window -n ovav -c "$HOME/Systems/ovav" \; \
         new-window -n akrynt -c "$HOME/Systems/projects/work/akrynt-agent" \; \
         select-window -t =ovav
end
