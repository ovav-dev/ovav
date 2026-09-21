#!/usr/bin/env bash
set -euo pipefail

ROOT="${OVAV_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
TUI="$ROOT/workstation/configs/opencode/tui.json"
CONFIG="$ROOT/opencode.json"
WRAPPER="$ROOT/workstation/scripts/opencode-resume-wrapper.sh"
CLIPBOARD_BRIDGE="$ROOT/workstation/scripts/ovav-clipboard-bridge.sh"
ALACRITTY="$ROOT/workstation/configs/alacritty/keybindings.toml"

python3 - "$CONFIG" "$TUI" <<'PY'
import json
import sys

config = json.load(open(sys.argv[1], encoding="utf-8"))
tui = json.load(open(sys.argv[2], encoding="utf-8"))
assert config["default_agent"] == "default"
assert "default" in config["agent"]
assert "mcp" not in config or not config["mcp"]
enabled_remote = [
    name for name, server in config.get("mcp", {}).items()
    if server.get("enabled") and server.get("type") == "remote"
]
assert not enabled_remote, enabled_remote
assert tui["keybinds"]["input_paste"] == {
    "key": "ctrl+v",
    "preventDefault": True,
}
print("PASS canonical OpenCode/WSL2 config")
PY

bash -n "$WRAPPER"
bash -n "$CLIPBOARD_BRIDGE"
[ "$("$CLIPBOARD_BRIDGE" -version)" = "ovav-xclip-bridge 1.0 (WSL2 Windows clipboard)" ]
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"; tmux -L "ovav-opencode-wsl2-${$}" kill-server >/dev/null 2>&1 || true' EXIT
mkdir -p "$tmp/bin"
cp "$WRAPPER" "$tmp/bin/opencode"
cat >"$tmp/bin/opencode.bin" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >"$TRACE_FILE"
SH
chmod +x "$tmp/bin/opencode" "$tmp/bin/opencode.bin"
TRACE_FILE="$tmp/resume.args" "$tmp/bin/opencode" --continue
grep -q -- '--mini' "$tmp/resume.args"
grep -q -- '--no-replay' "$tmp/resume.args"
grep -q -- '--continue' "$tmp/resume.args"
TRACE_FILE="$tmp/list.args" "$tmp/bin/opencode" session list
! grep -q -- '--no-replay' "$tmp/list.args"
echo "PASS resume wrapper routing"

cat >"$tmp/bin/powershell.exe" <<'SH'
#!/usr/bin/env bash
if [[ "$*" == *'Get-Clipboard'* ]]; then
  printf '%s\n' 'OVAV_CLIPBOARD_TEST'
else
  cat >"$TRACE_FILE"
fi
SH
chmod +x "$tmp/bin/powershell.exe"
printf '%s\n' 'OVAV_CLIPBOARD_WRITE' | PATH="$tmp/bin:$PATH" TRACE_FILE="$tmp/clipboard.input" "$CLIPBOARD_BRIDGE" -selection clipboard
grep -q 'OVAV_CLIPBOARD_WRITE' "$tmp/clipboard.input"
[ "$(PATH="$tmp/bin:$PATH" "$CLIPBOARD_BRIDGE" -selection clipboard -o)" = "OVAV_CLIPBOARD_TEST" ]
echo "PASS WSL2 clipboard bridge"

socket="ovav-opencode-wsl2-${$}"
tmux -L "$socket" -f "$ROOT/workstation/configs/tmux/tmux.conf" new-session -d -s test
[ "$(tmux -L "$socket" show-options -gqv mouse)" = "off" ]
[ "$(tmux -L "$socket" show-options -gqv set-clipboard)" = "external" ]
echo "PASS tmux selection/clipboard policy"

grep -q '^save_to_clipboard = true$' "$ALACRITTY"
grep -q '^key = "Return"$' "$ALACRITTY"
grep -q '^chars = "\\r"$' "$ALACRITTY"
grep -q '^key = "V"$' "$ALACRITTY"
grep -q '^mods = "Control"$' "$ALACRITTY"
grep -q '^action = "Paste"$' "$ALACRITTY"
grep -q '^mods = "Control|Shift"$' "$ALACRITTY"
grep -q '^action = "Copy"$' "$ALACRITTY"
echo "PASS Alacritty Ctrl+V and drag-copy policy"

if [ -x "${OPENCODE_BIN:-$HOME/.opencode/bin/opencode}" ]; then
  OPENCODE_BIN="${OPENCODE_BIN:-$HOME/.opencode/bin/opencode}"
  help="$($OPENCODE_BIN --help 2>&1)"
  grep -q -- '--session' <<<"$help"
  grep -q -- '--continue' <<<"$help"
  grep -q -- '--no-replay' <<<"$help"
  timeout 30s "$OPENCODE_BIN" debug startup --pure >/dev/null
  echo "PASS installed OpenCode resume flags/startup"
elif [ "${OVAV_SKIP_LIVE_OPENCODE:-0}" = "1" ]; then
  echo "SKIP live OpenCode checks (OVAV_SKIP_LIVE_OPENCODE=1)"
else
  echo "FAIL installed OpenCode is required; set OVAV_SKIP_LIVE_OPENCODE=1 only for source-only CI" >&2
  exit 1
fi
