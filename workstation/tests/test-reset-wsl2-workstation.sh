#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RESET="$ROOT/workstation/scripts/reset-wsl2-workstation.sh"

plan="$(OVAV_ROOT="$ROOT" bash "$RESET" --dry-run)"
[[ "$plan" == *"Fish:"* ]]
[[ "$plan" == *"Alacritty:"* ]]
[[ "$plan" == *"OpenCode:"* ]]
[[ "$plan" == *"dry-run: no changes made"* ]]

if OVAV_ROOT="$ROOT" bash "$RESET" --apply >/dev/null 2>&1; then
  exit 1
fi

fish -n "$ROOT/config/fish/config.fish" "$ROOT/config/fish/05-ovav-tmux-session.fish"
python3 - "$ROOT/workstation/configs/alacritty/alacritty.toml" <<'PY'
import sys
import tomllib
from pathlib import Path

data = tomllib.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert data["terminal"]["shell"]["args"] == ["-e", "fish", "-l"]
assert any(binding.get("key") == "Return" and binding.get("chars") == "\r" for binding in data["keyboard"]["bindings"])
PY

printf 'reset workstation: PASS\n'
