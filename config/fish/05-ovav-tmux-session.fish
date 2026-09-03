# OVAV per-window tmux session policy.
# Each new Alacritty window gets tmux without attaching to the shared `main`.

if status is-interactive; and not set -q TMUX; and command -q tmux
    set -l ovav_session "alacritty-$fish_pid"
    exec tmux new-session -s "$ovav_session"
end
