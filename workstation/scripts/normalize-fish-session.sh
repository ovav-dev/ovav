#!/usr/bin/env bash
# Remove the legacy Fish hook that attaches every new terminal to tmux main.
# The operation is deliberately exact: unrelated user Fish configuration stays.
set -euo pipefail

config="${1:?usage: normalize-fish-session.sh <config.fish>}"

if [ ! -f "$config" ]; then
  exit 0
fi

tmp="$(mktemp "${config}.ovav.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

perl -0pe '
my $block = <<BLOCK;
# Auto-start tmux INSIDE this shell
# This makes every Alacritty session attach to tmux directly
if status is-interactive
    if not set -q TMUX
        if command -v tmux >/dev/null 2>&1
            # Try to attach to existing session, otherwise create and attach
            if tmux has-session -t main 2>/dev/null
                exec tmux attach-session -t main
            else
                exec tmux new-session -s main
            end
        end
    end
end
BLOCK
s/\Q$block\E\n?//;
' "$config" > "$tmp"

if cmp -s "$config" "$tmp"; then
  printf 'unchanged: %s\n' "$config"
else
  mv "$tmp" "$config"
  printf 'normalized: %s\n' "$config"
fi
