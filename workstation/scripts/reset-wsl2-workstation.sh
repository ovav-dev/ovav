#!/usr/bin/env bash
# Rebuild the active Windows/WSL2 workstation from canonical OVAV sources.
# Destructive mode requires both --apply and --purge-history.
set -euo pipefail

APPLY=0
PURGE=0
OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
HOME_ROOT="${OVAV_HOME:-$HOME}"
FISH_DIR="${FISH_DIR:-$HOME_ROOT/.config/fish}"
FISH_SOURCE="$OVAV_ROOT/config/fish"
TMUX_DEST="${TMUX_DEST:-$HOME_ROOT/.tmux.conf}"
TMUX_SOURCE="$OVAV_ROOT/workstation/configs/tmux/tmux.conf"
ALACRITTY_CONFIG="${ALACRITTY_CONFIG:-/mnt/c/Users/Alexa/AppData/Roaming/alacritty/alacritty.toml}"
ALACRITTY_SOURCE="$OVAV_ROOT/workstation/configs/alacritty/alacritty.toml"
OPENCODE_CONFIG="${OPENCODE_CONFIG:-$HOME_ROOT/.config/opencode}"
OPENCODE_DATA="${OPENCODE_DATA:-$HOME_ROOT/.local/share/opencode}"
OPENCODE_LAUNCHER="${OPENCODE_LAUNCHER:-$HOME_ROOT/.opencode/bin/opencode}"
BACKUP_ROOT="${OVAV_BACKUPS:-$HOME_ROOT/.ovav-backups}"

usage() {
  cat <<'EOF'
Usage: reset-wsl2-workstation.sh --apply --purge-history

  --dry-run          Print the reset plan (default)
  --apply            Apply config reset and stop OpenCode processes
  --purge-history    Required with --apply; permanently remove quarantined history
EOF
}

for arg in "$@"; do
  case "$arg" in
    --apply) APPLY=1 ;;
    --purge-history) PURGE=1 ;;
    --dry-run) ;;
    --help|-h) usage; exit 0 ;;
    *) printf 'unknown option: %s\n' "$arg" >&2; usage >&2; exit 2 ;;
  esac
done

BACKUP_DIR="$BACKUP_ROOT/reset-$(date +%Y%m%d-%H%M%S)"

printf 'OVAV WSL2 reset plan\n'
printf '  Fish:      %s\n' "$FISH_DIR"
printf '  tmux:      %s\n' "$TMUX_DEST"
printf '  Alacritty: %s\n' "$ALACRITTY_CONFIG"
printf '  OpenCode:  %s + %s\n' "$OPENCODE_CONFIG" "$OPENCODE_DATA"
printf '  Backup:    %s\n' "$BACKUP_DIR"

if [ "$APPLY" -eq 0 ]; then
  printf 'dry-run: no changes made\n'
  exit 0
fi
if [ "$PURGE" -eq 0 ]; then
  printf '%s\n' '--purge-history is required with --apply' >&2
  exit 2
fi

for required in "$FISH_SOURCE/config.fish" "$FISH_SOURCE/05-ovav-tmux-session.fish" \
  "$TMUX_SOURCE" "$ALACRITTY_SOURCE" "$OVAV_ROOT/opencode.json"; do
  [ -e "$required" ] || { printf 'missing canonical source: %s\n' "$required" >&2; exit 1; }
done

mkdir -p "$BACKUP_DIR"

backup_path() {
  local source="$1" name="$2"
  if [ -e "$source" ] || [ -L "$source" ]; then
    cp -a "$source" "$BACKUP_DIR/$name"
  fi
}

quarantine_path() {
  local source="$1" name="$2"
  if [ -e "$source" ] || [ -L "$source" ]; then
    mv "$source" "$BACKUP_DIR/$name"
  fi
}

backup_path "$FISH_DIR" fish.config
backup_path "$TMUX_DEST" tmux.conf
backup_path "$ALACRITTY_CONFIG" alacritty.toml
backup_path "$OPENCODE_CONFIG" opencode.config
quarantine_path "$OPENCODE_DATA" opencode.data
backup_path "$OPENCODE_LAUNCHER" opencode.launcher

printf 'stopping OpenCode processes\n'
for pid in $(pgrep -x opencode.bin 2>/dev/null || true); do
  kill -TERM "$pid" 2>/dev/null || true
done
for _ in $(seq 1 20); do
  pgrep -x opencode.bin >/dev/null 2>&1 || break
  sleep 0.25
done
for pid in $(pgrep -x opencode.bin 2>/dev/null || true); do
  kill -KILL "$pid" 2>/dev/null || true
done

printf 'removing non-main tmux sessions\n'
if command -v tmux >/dev/null 2>&1; then
  while IFS= read -r session; do
    case "$session" in
      ''|main) ;;
      *) tmux kill-session -t "$session" 2>/dev/null || true ;;
    esac
  done < <(tmux list-sessions -F '#{session_name}' 2>/dev/null || true)
fi

rm -rf "$FISH_DIR" "$OPENCODE_CONFIG" "$OPENCODE_DATA"
mkdir -p "$FISH_DIR/conf.d" "$OPENCODE_CONFIG/themes" "$OPENCODE_DATA"

cp -p "$FISH_SOURCE/config.fish" "$FISH_DIR/config.fish"
cp -p "$FISH_SOURCE/fish_prompt.fish" "$FISH_DIR/fish_prompt.fish"
for fish_file in 05-ovav-tmux-session.fish 10-ovav-color-profile.fish 30-ovav-runtime-tools.fish 41-ovav-syntax-colors.fish ovav.fish; do
  cp -p "$FISH_SOURCE/$fish_file" "$FISH_DIR/conf.d/$fish_file"
done

cp -p "$TMUX_SOURCE" "$TMUX_DEST"
mkdir -p "${ALACRITTY_CONFIG%/*}"
cp -p "$ALACRITTY_SOURCE" "$ALACRITTY_CONFIG"

ln -s "$OVAV_ROOT/opencode.json" "$OPENCODE_CONFIG/opencode.json"
ln -s "$OVAV_ROOT/go-runtime/internal/runtimes/opencode/agents" "$OPENCODE_CONFIG/agents"
ln -s "$OVAV_ROOT/opencode_AGENTS.md" "$OPENCODE_CONFIG/opencode_AGENTS.md"
cp -p "$OVAV_ROOT/workstation/configs/opencode/tui.json" "$OPENCODE_CONFIG/tui.json"
cp -p "$OVAV_ROOT/workstation/configs/opencode/themes/ovav-night.json" "$OPENCODE_CONFIG/themes/ovav-night.json"
cp -p "$OVAV_ROOT/workstation/configs/opencode/themes/ovav-day.json" "$OPENCODE_CONFIG/themes/ovav-day.json"

if [ -f "$BACKUP_DIR/opencode.data/auth.json" ]; then
  cp -p "$BACKUP_DIR/opencode.data/auth.json" "$OPENCODE_DATA/auth.json"
fi
if [ -f "$OVAV_ROOT/workstation/scripts/opencode-resume-wrapper.sh" ] && [ -f "$HOME_ROOT/.opencode/bin/opencode.bin" ]; then
  mkdir -p "${OPENCODE_LAUNCHER%/*}"
  cp -p "$OVAV_ROOT/workstation/scripts/opencode-resume-wrapper.sh" "$OPENCODE_LAUNCHER"
  chmod 0755 "$OPENCODE_LAUNCHER"
fi

fish -n "$FISH_DIR/config.fish" "$FISH_DIR/fish_prompt.fish" "$FISH_DIR/conf.d"/*.fish
python3 - "$ALACRITTY_CONFIG" <<'PY'
import sys
import tomllib
from pathlib import Path

path = Path(sys.argv[1])
data = tomllib.loads(path.read_text(encoding="utf-8"))
assert data["terminal"]["shell"]["program"] == "wsl.exe"
assert data["terminal"]["shell"]["args"] == ["-e", "fish", "-l"]
assert any(binding.get("key") == "Return" and binding.get("chars") == "\r" for binding in data["keyboard"]["bindings"])
PY

if [ "$PURGE" -eq 1 ]; then
  rm -rf "$BACKUP_ROOT"
fi

printf 'reset complete: Fish → isolated tmux → OpenCode\n'
