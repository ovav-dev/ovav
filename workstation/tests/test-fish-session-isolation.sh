#!/usr/bin/env bash
# Regression test for the live Alacritty → WSL2 → Fish startup path.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NORMALIZER="$ROOT/workstation/scripts/normalize-fish-session.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

fixture="$TMP_DIR/config.fish"
cat > "$fixture" <<'EOF'
set -gx OVAV_ROOT /home/braka/Systems/ovav

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
EOF

chmod 0755 "$NORMALIZER"
bash "$NORMALIZER" "$fixture" >/dev/null
after="$(<"$fixture")"

[[ "$after" == *"set -gx OVAV_ROOT /home/braka/Systems/ovav"* ]]
[[ "$after" != *"attach-session -t main"* ]]
[[ "$after" != *"new-session -s main"* ]]
[[ "$after" != *"Auto-start tmux INSIDE this shell"* ]]

checksum="$(sha256sum "$fixture")"
bash "$NORMALIZER" "$fixture" >/dev/null
[[ "$checksum" == "$(sha256sum "$fixture")" ]]

clean="$TMP_DIR/clean.fish"
printf '%s\n' 'set -gx OVAV_ROOT /home/braka/Systems/ovav' > "$clean"
clean_checksum="$(sha256sum "$clean")"
bash "$NORMALIZER" "$clean" >/dev/null
[[ "$clean_checksum" == "$(sha256sum "$clean")" ]]

if command -v fish >/dev/null 2>&1; then
  fish -n "$fixture"
  fish -n "$clean"
fi

printf 'fish session isolation: PASS\n'
