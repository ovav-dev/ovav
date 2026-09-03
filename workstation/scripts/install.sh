#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Workstation Installer
#  Idempotent. Surgical. Backed up. Auditable.
#  Rule #39: Backup timestamped. Rollback per-layer.
# ─────────────────────────────────────────────────────────────
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
WORKSTATION="$OVAV_ROOT/workstation"
FISH_CONFIG="${FISH_CONFIG:-$HOME/.config/fish/config.fish}"
FISH_NORMALIZER_SRC="$WORKSTATION/scripts/normalize-fish-session.sh"
FISH_TMUX_SRC="$OVAV_ROOT/config/fish/05-ovav-tmux-session.fish"
FISH_TMUX_DEST="$HOME/.config/fish/conf.d/05-ovav-tmux-session.fish"
WIN_PS_PROFILE_DIR="/mnt/c/Users/Alexa/OneDrive/Documentos/PowerShell"
WIN_PS_PROFILE="$WIN_PS_PROFILE_DIR/Microsoft.PowerShell_profile.ps1"
TMUX_SRC="$WORKSTATION/configs/tmux/tmux.conf"
TMUX_DEST="$HOME/.tmux.conf"
ALACRITTY_SRC="$WORKSTATION/configs/alacritty/keybindings.toml"
ALACRITTY_CONFIG="${ALACRITTY_CONFIG:-/mnt/c/Users/Alexa/AppData/Roaming/alacritty/alacritty.toml}"
OPENCODE_LAUNCHER_SRC="$WORKSTATION/scripts/opencode-resume-wrapper.sh"
OPENCODE_LAUNCHER="$HOME/.opencode/bin/opencode"
CLIPBOARD_BRIDGE_SRC="$WORKSTATION/scripts/ovav-clipboard-bridge.sh"
XCLIP_BIN="$HOME/.local/bin/xclip"

TS="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="$HOME/.ovav-backups/$TS"
mkdir -p "$BACKUP_DIR"

log()  { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; }

# ─── Pre-flight ─────────────────────────────────────────────
log "Pre-flight checks"
for cmd in bash cp perl; do
  command -v "$cmd" >/dev/null || { err "missing: $cmd"; exit 1; }
done
ok "tools available"

# ─── 1. Backup existing config ─────────────────────────────
log "Backing up existing configs → $BACKUP_DIR"
[ -f "$HOME/.bashrc" ] && cp -p "$HOME/.bashrc" "$BACKUP_DIR/bashrc.bak" && ok "bashrc"
[ -f "$HOME/.inputrc" ] && cp -p "$HOME/.inputrc" "$BACKUP_DIR/inputrc.bak" && ok "inputrc"
[ -f "$HOME/.config/starship.toml" ] && cp -p "$HOME/.config/starship.toml" "$BACKUP_DIR/starship.toml.bak" && ok "starship"
mkdir -p "$HOME/.config/atuin"
[ -f "$HOME/.config/atuin/config.toml" ] && cp -p "$HOME/.config/atuin/config.toml" "$BACKUP_DIR/atuin-config.toml.bak" && ok "atuin"
[ -f "$WIN_PS_PROFILE" ] && cp -p "$WIN_PS_PROFILE" "$BACKUP_DIR/powershell-profile.ps1.bak" && ok "PS profile"
[ -f "$TMUX_DEST" ] && cp -p "$TMUX_DEST" "$BACKUP_DIR/tmux.conf.bak" && ok "tmux"
[ -f "$ALACRITTY_CONFIG" ] && cp -p "$ALACRITTY_CONFIG" "$BACKUP_DIR/alacritty.toml.bak" && ok "Alacritty"
[ -f "$FISH_CONFIG" ] && cp -p "$FISH_CONFIG" "$BACKUP_DIR/fish-config.fish.bak" && ok "Fish config"
[ -f "$FISH_TMUX_DEST" ] && cp -p "$FISH_TMUX_DEST" "$BACKUP_DIR/fish-tmux-session.fish.bak" && ok "Fish tmux policy"
[ -f "$HOME/.config/opencode/tui.json" ] && cp -p "$HOME/.config/opencode/tui.json" "$BACKUP_DIR/opencode-tui.json.bak" && ok "OpenCode TUI"
[ -f "$OPENCODE_LAUNCHER" ] && cp -p "$OPENCODE_LAUNCHER" "$BACKUP_DIR/opencode-launcher.bak" && ok "OpenCode launcher"
[ -f "$XCLIP_BIN" ] && ! "$XCLIP_BIN" -version >/dev/null 2>&1 && cp -p "$XCLIP_BIN" "$BACKUP_DIR/xclip.bak" && ok "broken xclip"

# ─── 2. Install bashrc additions ───────────────────────────
log "Installing OVAV bashrc"
mkdir -p "$HOME/.config"
OVAV_MARKER='# ── OVAV WORKSTATION 2026 ── managed by workstation/scripts/install.sh'
if ! grep -qF "$OVAV_MARKER" "$HOME/.bashrc" 2>/dev/null; then
  {
    echo ""
    echo "$OVAV_MARKER"
    cat "$WORKSTATION/configs/bashrc/ovav.bashrc"
    echo "# ── END OVAV WORKSTATION ──"
  } >> "$HOME/.bashrc"
  ok "bashrc updated (idempotent append)"
else
  ok "bashrc already has OVAV block (skipping)"
fi

# ─── 2b. Install readline config (~/.inputrc) ───────────────
log "Installing OVAV readline config (inputrc)"
OVAV_INPUTRC="$HOME/.inputrc"
INPUTRC_SRC="$WORKSTATION/configs/inputrc/ovav.inputrc"
OVAV_INPUTRC_MARKER='OVAV readline config — sourced automatically by bash'
if [ ! -f "$OVAV_INPUTRC" ]; then
  cp -p "$INPUTRC_SRC" "$OVAV_INPUTRC"
  ok "inputrc installed (new)"
elif ! grep -qF "$OVAV_INPUTRC_MARKER" "$OVAV_INPUTRC" 2>/dev/null; then
  # Existing user inputrc — append OVAV bindings without clobbering user prefs.
  # Backup first.
  [ -f "$BACKUP_DIR/inputrc.bak" ] || cp -p "$OVAV_INPUTRC" "$BACKUP_DIR/inputrc.bak"
  {
    echo ""
    echo "# $OVAV_INPUTRC_MARKER"
    cat "$INPUTRC_SRC"
  } >> "$OVAV_INPUTRC"
  ok "inputrc appended (user prefs preserved, backed up)"
else
  ok "inputrc already has OVAV block"
fi

# ─── 2c. Install tmux stability config ──────────────────────
log "Installing tmux keyboard/clipboard config"
if [ -f "$TMUX_SRC" ]; then
  cp -p "$TMUX_SRC" "$TMUX_DEST"
  ok "tmux.conf installed (backup available)"
  # A running tmux server does not reread ~/.tmux.conf automatically.  Apply
  # the policy in-place so mouse selection and clipboard behavior change
  # without terminating the user's active OpenCode session.
  if command -v tmux >/dev/null 2>&1 && tmux list-sessions >/dev/null 2>&1; then
    tmux source-file "$TMUX_DEST"
    ok "tmux server configuration reloaded (session preserved)"
  fi
else
  warn "tmux source config not found: $TMUX_SRC"
fi

# ─── 2d. Install Alacritty Shift+Enter bridge ────────────────
log "Installing Alacritty keyboard bridge"
ALACRITTY_MARKER='OVAV Alacritty keyboard bridge'
if [ ! -f "$ALACRITTY_CONFIG" ]; then
  warn "Alacritty config not found at $ALACRITTY_CONFIG"
elif ! grep -qF "$ALACRITTY_MARKER" "$ALACRITTY_CONFIG" 2>/dev/null; then
  {
    printf '\n'
    cat "$ALACRITTY_SRC"
  } >> "$ALACRITTY_CONFIG"
  ok "Alacritty Shift+Enter bridge appended (backup available)"
else
  ok "Alacritty keyboard bridge already installed"
fi

# ─── 2e. Install isolated Fish tmux sessions ─────────────────
log "Isolating new Alacritty tmux sessions"
if [ -f "$FISH_CONFIG" ] && [ -x "$FISH_NORMALIZER_SRC" ]; then
  bash "$FISH_NORMALIZER_SRC" "$FISH_CONFIG"
  mkdir -p "${FISH_TMUX_DEST%/*}"
  cp -p "$FISH_TMUX_SRC" "$FISH_TMUX_DEST"
  chmod 0644 "$FISH_TMUX_DEST"
  if command -v fish >/dev/null 2>&1; then
    fish -n "$FISH_CONFIG"
    fish -n "$FISH_TMUX_DEST"
  fi
  ok "Fish startup checked (isolated tmux session installed)"
else
  warn "Fish config/normalizer unavailable; no startup changes made"
fi

# ─── 3. Install Starship ───────────────────────────────────
log "Installing Starship config"
cp -p "$WORKSTATION/configs/starship/starship.toml" "$HOME/.config/starship.toml"
ok "starship.toml"

# ─── 4. Install Atuin ─────────────────────────────────────
log "Installing Atuin config"
mkdir -p "$HOME/.config/atuin"
cp -p "$WORKSTATION/configs/atuin/config.toml" "$HOME/.config/atuin/config.toml"
ok "atuin config.toml"

# ─── 5. Install OpenCode themes ───────────────────────────
log "Installing OpenCode TUI themes"
mkdir -p "$HOME/.config/opencode/themes"
cp -p "$WORKSTATION/configs/opencode/tui.json" "$HOME/.config/opencode/tui.json"
cp -p "$WORKSTATION/configs/opencode/themes/ovav-night.json" "$HOME/.config/opencode/themes/ovav-night.json"
cp -p "$WORKSTATION/configs/opencode/themes/ovav-day.json" "$HOME/.config/opencode/themes/ovav-day.json"
ok "opencode tui.json + 2 themes"
if [ -x "${OPENCODE_LAUNCHER%/*}/opencode.bin" ]; then
  cp -p "$OPENCODE_LAUNCHER_SRC" "$OPENCODE_LAUNCHER"
  chmod 0755 "$OPENCODE_LAUNCHER"
  ok "OpenCode resume launcher installed (binary preserved)"
fi

# OpenCode's Linux clipboard backend selects xclip before xsel. In WSL2 an
# xclip binary with an unresolved X11 library fails instead of falling back.
# Replace only that broken user-local binary; preserve a working installation.
if [ -f "$XCLIP_BIN" ] && ! "$XCLIP_BIN" -version >/dev/null 2>&1 && command -v powershell.exe >/dev/null 2>&1; then
  cp -p "$CLIPBOARD_BRIDGE_SRC" "$XCLIP_BIN"
  chmod 0755 "$XCLIP_BIN"
  ok "WSL2 xclip clipboard bridge installed (backup available)"
elif [ -f "$XCLIP_BIN" ] && ! "$XCLIP_BIN" -version >/dev/null 2>&1; then
  warn "broken xclip detected but powershell.exe is unavailable; clipboard bridge not installed"
fi

# ─── 6. Legacy terminal surfaces ────────────────────────────
# Warp and Microsoft Intelligent Terminal are historical artifacts only.
# The active host is Alacritty; never probe or mutate the legacy paths here.
log "Skipping inactive Warp/Intelligent Terminal surfaces"
ok "active host routing is Alacritty → WSL2 → tmux → OpenCode"

# ─── 7. PowerShell profile ────────────────────────────────
log "Installing PowerShell profile"
if [ -d "$WIN_PS_PROFILE_DIR" ] || mkdir -p "$WIN_PS_PROFILE_DIR"; then
  if [ ! -f "$WIN_PS_PROFILE" ]; then
    cp "$WORKSTATION/configs/powershell/Microsoft.PowerShell_profile.ps1" "$WIN_PS_PROFILE"
    ok "PowerShell profile installed (new)"
  elif ! grep -q "OVAV WORKSTATION 2026" "$WIN_PS_PROFILE" 2>/dev/null; then
    {
      echo ""
      echo "# ── OVAV WORKSTATION 2026 ──"
      cat "$WORKSTATION/configs/powershell/Microsoft.PowerShell_profile.ps1"
    } >> "$WIN_PS_PROFILE"
    ok "PowerShell profile appended"
  else
    ok "PowerShell profile already has OVAV block"
  fi
else
  warn "Could not create $WIN_PS_PROFILE_DIR — manual install required"
fi

# ─── 8. Verify installations ───────────────────────────────
log "Verification"
[ -f "$HOME/.config/starship.toml" ] && ok "starship.toml installed"
[ -f "$HOME/.config/atuin/config.toml" ] && ok "atuin config installed"
[ -f "$HOME/.config/opencode/tui.json" ] && ok "opencode tui.json installed"
[ -f "$HOME/.config/opencode/themes/ovav-night.json" ] && ok "ovav-night theme installed"
[ -f "$HOME/.config/opencode/themes/ovav-day.json" ] && ok "ovav-day theme installed"
[ -f "$HOME/.inputrc" ] && ok "inputrc installed"
[ -f "$ALACRITTY_CONFIG" ] && ok "Alacritty config present"

if [ -x "$HOME/.local/bin/ovav" ]; then
  ok "OVAV CLI present"
else
  warn "OVAV CLI not at ~/.local/bin/ovav — install required"
fi

if [ -x "$HOME/.opencode/bin/opencode" ]; then
  ok "OpenCode canonical Linux install present"
else
  warn "OpenCode not at ~/.opencode/bin/opencode — install required"
fi

# ─── Summary ───────────────────────────────────────────────
cat <<EOF

═══════════════════════════════════════════════════════════
  OVAV WORKSTATION INSTALLED
═══════════════════════════════════════════════════════════
  Backup:      $BACKUP_DIR
  Workspace:   $WORKSTATION

  Next steps:
    1. Restart Alacritty only if live reload does not apply the keyboard change
    2. Source ~/.bashrc in current shell:   source ~/.bashrc
    3. Verify: ovav status
    4. Run E2E test: bash $WORKSTATION/tests/test-e2e.sh
    5. Run benchmark: bash $WORKSTATION/scripts/benchmark.sh

═══════════════════════════════════════════════════════════
EOF
