#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV Workstation Installer
#  Idempotent. Surgical. Backed up. Auditable.
#  Rule #39: Backup timestamped. Rollback per-layer.
# ─────────────────────────────────────────────────────────────
set -euo pipefail

OVAV_ROOT="${OVAV_ROOT:-/home/braka/Systems/ovav}"
WORKSTATION="$OVAV_ROOT/workstation"
INTEL_TERM_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"
INTEL_TERM_STATE="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/state.json"
WIN_PS_PROFILE_DIR="/mnt/c/Users/Alexa/OneDrive/Documentos/PowerShell"
WIN_PS_PROFILE="$WIN_PS_PROFILE_DIR/Microsoft.PowerShell_profile.ps1"

TS="$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="$HOME/.ovav-backups/$TS"
mkdir -p "$BACKUP_DIR"

log()  { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; }

# ─── Pre-flight ─────────────────────────────────────────────
log "Pre-flight checks"
for cmd in jq bash cp; do
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
[ -f "$INTEL_TERM_SETTINGS" ] && cp -p "$INTEL_TERM_SETTINGS" "$BACKUP_DIR/intel-terminal-settings.json.bak" && ok "IT settings"
[ -f "$INTEL_TERM_STATE" ] && cp -p "$INTEL_TERM_STATE" "$BACKUP_DIR/intel-terminal-state.json.bak" && ok "IT state"
[ -f "$WIN_PS_PROFILE" ] && cp -p "$WIN_PS_PROFILE" "$BACKUP_DIR/powershell-profile.ps1.bak" && ok "PS profile"

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

# ─── 6. Intelligent Terminal settings.json merge ──────────
log "Merging Intelligent Terminal settings"
if [ ! -f "$INTEL_TERM_SETTINGS" ]; then
  warn "settings.json not found at $INTEL_TERM_SETTINGS"
  warn "Skipping IT merge. Apply manually: $WORKSTATION/configs/intelligent-terminal/settings-fragment.json"
else
  # Surgical merge using jq
  FRAG="$WORKSTATION/configs/intelligent-terminal/settings-fragment.json"
  cp -p "$INTEL_TERM_SETTINGS" "$BACKUP_DIR/intel-terminal-settings.premerge.json"

  # The merge: combine existing settings with OVAV fragment
  # Uses jq -s (slurp) to merge two JSON objects deeply.
  # IMPORTANT: After `.| ...` the `.` becomes the merged OBJECT, not the
  # slurp array. So we cannot use .[0] / .[1] in subsequent expressions —
  # use $existing / $fragment variables via `as`.
  jq -s '
    .[0] as $existing |
    .[1] as $fragment |
    $existing * $fragment
    | .profiles.list = (($existing.profiles.list // []) + ($fragment.profiles.list // []) | unique_by(.guid))
    | .schemes       = (($existing.schemes       // []) + ($fragment.schemes       // []) | unique_by(.name))
    | .actions       = (($existing.actions       // []) + ($fragment.actions       // []) | unique_by(.name))
  ' "$INTEL_TERM_SETTINGS" "$FRAG" > "$INTEL_TERM_SETTINGS.tmp"

  if jq empty "$INTEL_TERM_SETTINGS.tmp" 2>/dev/null; then
    mv "$INTEL_TERM_SETTINGS.tmp" "$INTEL_TERM_SETTINGS"
    ok "settings.json merged (validated JSON)"
  else
    err "merged JSON invalid — restoring original"
    jq . "$INTEL_TERM_SETTINGS.tmp" 2>&1 | head -20
    mv "$BACKUP_DIR/intel-terminal-settings.json.bak" "$INTEL_TERM_SETTINGS"
    exit 1
  fi
fi

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
    1. Restart Intelligent Terminal (or close+reopen)
    2. Source ~/.bashrc in current shell:   source ~/.bashrc
    3. Verify: ovav status
    4. Run E2E test: bash $WORKSTATION/tests/test-e2e.sh
    5. Run benchmark: bash $WORKSTATION/scripts/benchmark.sh

═══════════════════════════════════════════════════════════
EOF