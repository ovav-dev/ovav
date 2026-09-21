#!/usr/bin/env bash
# Regression test for the live Alacritty → WSL2 → Fish startup path.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
NORMALIZER="$ROOT/workstation/scripts/normalize-fish-session.sh"
TMUX_POLICY="$ROOT/config/fish/05-ovav-tmux-session.fish"
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

fake_bin="$TMP_DIR/bin"
mkdir -p "$fake_bin"
cat > "$fake_bin/tmux" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*"
EOF
chmod 0755 "$fake_bin/tmux"

tmux_call="$(env -u TMUX PATH="$fake_bin:$PATH" fish -N -ic "source '$TMUX_POLICY'")"
[[ "$tmux_call" =~ ^new-session\ -s\ alacritty-[0-9]+$ ]]
[[ "$tmux_call" != *"main"* ]]

if PATH="$fake_bin:$PATH" TMUX=/tmp/tmux-ignored fish -N -ic "source '$TMUX_POLICY'" | grep -q .; then
  exit 1
fi

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
  fish -n "$TMUX_POLICY"
fi

printf 'fish session isolation: PASS\n'
