#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV IT Cache Clean — Full Intelligent Terminal cache reset without killing processes.
#
#  Required environment variables (no defaults — user must provide):
#    OVAV_IT_SETTINGS  absolute path to settings.json
#    OVAV_IT_STATE     absolute path to state.json
#
#  Optional environment variables (with documented defaults):
#    OVAV_BACKUP_DIR   backup destination
#                      (default: $HOME/.ovav-backups/it-clean-<timestamp>)
#    OVAV_POWERSHELL   powershell.exe command to invoke on Windows
#                      (default: powershell.exe — must be on PATH)
#
#  External dependencies (must be on PATH):
#    bash       ≥ 4.0
#    python3    (required for JSON validation)
#    powershell (only if running from WSL — used for Windows-side IT touch & relaunch)
#
#  This script is registered under .ovav/registry/tool_configs.yaml
#  → ovav_workstation_scripts. It is NEVER auto-run by OVAV.
#
#  Usage:
#    OVAV_IT_SETTINGS="/mnt/c/.../settings.json" \
#    OVAV_IT_STATE="/mnt/c/.../state.json" \
#    bash ovav-it-cache-clean.sh
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# ── Required env vars ───────────────────────────────────────
if [ -z "${OVAV_IT_SETTINGS:-}" ]; then
  echo "ERROR: OVAV_IT_SETTINGS env var is required." >&2
  exit 2
fi
if [ -z "${OVAV_IT_STATE:-}" ]; then
  echo "ERROR: OVAV_IT_STATE env var is required." >&2
  exit 2
fi

# ── Optional env vars with documented defaults ─────────────
TS=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="${OVAV_BACKUP_DIR:-$HOME/.ovav-backups}"
BACKUP="$BACKUP_DIR/it-clean-${TS}"
POWERSHELL_CMD="${OVAV_POWERSHELL:-powershell.exe}"
mkdir -p "$BACKUP"

# ── Dependency checks (no implicit assumptions) ─────────────
command -v python3 >/dev/null 2>&1 || {
  echo "ERROR: python3 is required but not found on PATH." >&2
  exit 3
}

log()  { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }
fail() { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; exit 1; }

# ── 1. Full backup ──────────────────────────────────────────
log "Step 1: Full backup (settings + state)"
[ -f "$OVAV_IT_SETTINGS" ] && cp -p "$OVAV_IT_SETTINGS" "$BACKUP/settings.json.bak"
[ -f "$OVAV_IT_STATE" ]    && cp -p "$OVAV_IT_STATE"    "$BACKUP/state.json.bak"
ok "Backup at $BACKUP"

# ── 2. Validate settings.json ───────────────────────────────
log "Step 2: Validate settings.json"
if OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "import json,os; json.load(open(os.environ['OVAV_IT_SETTINGS']))" 2>/dev/null; then
  ok "settings.json is valid JSON"
else
  fail "settings.json is invalid JSON — restore from backup"
fi

# Count profiles
VIS=$(OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "
import json, os
with open(os.environ['OVAV_IT_SETTINGS']) as f: s = json.load(f)
print(sum(1 for p in s.get('profiles',{}).get('list',[]) if not p.get('hidden')))
")
HID=$(OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "
import json, os
with open(os.environ['OVAV_IT_SETTINGS']) as f: s = json.load(f)
print(sum(1 for p in s.get('profiles',{}).get('list',[]) if p.get('hidden')))
")
echo "  Profiles: $VIS visible, $HID hidden"
[ "$VIS" -ge 1 ] || fail "Zero visible profiles — restore backup or regenerate"

# ── 3. Validate GUIDs ───────────────────────────────────────
log "Step 3: Validate all GUIDs"
INVALID=$(OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "
import json, os, re
p = os.environ['OVAV_IT_SETTINGS']
with open(p) as f: s = json.load(f)
pat = re.compile(r'^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$')
bad = [pr['name'] for pr in s['profiles']['list'] if not pat.match(pr.get('guid',''))]
print('|'.join(bad))
")
if [ -n "$INVALID" ]; then
  fail "Invalid GUIDs in: $INVALID"
fi
ok "all GUIDs valid"

# ── 4. Reset settingsHash in state.json ────────────────────
log "Step 4: Reset settingsHash → force IT to re-validate everything"
OVAV_IT_STATE="$OVAV_IT_STATE" python3 -c "
import json, os
p = os.environ['OVAV_IT_STATE']
with open(p) as f: s = json.load(f)
s['settingsHash'] = ''
with open(p, 'w') as f: json.dump(s, f, indent=1)
print('  settingsHash cleared')
"
ok "state.json with settingsHash=''"

# ── 5. Touch settings.json (mtime refresh) ──────────────────
log "Step 5: Touch settings.json (mtime refresh)"
if command -v "$POWERSHELL_CMD" >/dev/null 2>&1; then
  "$POWERSHELL_CMD" -NoProfile -Command "
    if (Test-Path '$OVAV_IT_SETTINGS') {
      (Get-Item '$OVAV_IT_SETTINGS').LastWriteTime = Get-Date
      Write-Host '  settings.json mtime refreshed'
    } else {
      Write-Host '  WARN: settings.json not visible from Windows side'
    }
  " 2>&1 | head -5
else
  warn "powershell ($POWERSHELL_CMD) not on PATH — skipping Windows-side touch"
fi

# ── 6. Launch new IT window (do NOT kill existing ones) ────
log "Step 6: Launch new IT window for forced-reload"
if command -v "$POWERSHELL_CMD" >/dev/null 2>&1; then
  "$POWERSHELL_CMD" -NoProfile -Command "
    Start-Process 'shell:AppsFolder\Microsoft.IntelligentTerminal_8wekyb3d8bbwe!App'
    Start-Sleep -Seconds 3
    \$running = Get-Process | Where-Object { \$_.Name -match '^WindowsTerminal$' }
    Write-Host \"IT windows alive: \$(\$running.Count)\"
  " 2>&1 | head -5
  ok "New IT window opened (yours were not touched)"
else
  warn "powershell ($POWERSHELL_CMD) not on PATH — skipping IT window relaunch"
fi

# ── 7. Final verification ───────────────────────────────────
log "Step 7: Final verification"

IT_OK=$(OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "
import json, os, re
p = os.environ['OVAV_IT_SETTINGS']
with open(p) as f: s = json.load(f)
pat = re.compile(r'^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$')
n_visible = sum(1 for x in s['profiles']['list'] if not x.get('hidden'))
all_guids_ok = all(pat.match(p.get('guid','')) for p in s['profiles']['list'])
print('OK' if (n_visible >= 1 and all_guids_ok) else 'FAIL')
")
[ "$IT_OK" = "OK" ] && ok "IT settings verified" || fail "IT settings corrupted"

# ── Summary ─────────────────────────────────────────────────
cat <<EOF

═══════════════════════════════════════════════════════════
  OVAV IT Cache Clean — Done
═══════════════════════════════════════════════════════════
  Backup:        $BACKUP
  Visible:       $VIS
  Hidden:        $HID
  Settings hash: cleared (IT will force-reload)
  Windows alive: NO existing windows were killed

  Action for operator:
  • Wait ~30 seconds — IT should recompute hash
  • If you still see error in old window, close ONLY that window
  • The "All profiles hidden" error should disappear when:
      - new IT window launches (already attempted), OR
      - you close the specific window showing the error

═══════════════════════════════════════════════════════════
EOF
