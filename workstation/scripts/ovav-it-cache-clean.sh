#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  OVAV IT Cache Clean — Full reset sin matar procesos
#  Uso: bash workstation/scripts/ovav-it-cache-clean.sh
# ─────────────────────────────────────────────────────────────
set -euo pipefail

IT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"
IT_STATE="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/state.json"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/it-clean-${TS}"
mkdir -p "$BACKUP"

log()  { printf "\033[1;36m▸\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m✓\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m⚠\033[0m %s\n" "$*"; }
fail() { printf "\033[1;31m✗\033[0m %s\n" "$*" >&2; exit 1; }

# ── 1. Backup completo ─────────────────────────────────────
log "Paso 1: Backup completo (settings + state)"
[ -f "$IT_SETTINGS" ] && cp -p "$IT_SETTINGS" "$BACKUP/settings.json.bak"
[ -f "$IT_STATE" ]    && cp -p "$IT_STATE"    "$BACKUP/state.json.bak"
ok "Backup en $BACKUP"

# ── 2. Validar settings.json actual ────────────────────────
log "Paso 2: Validar settings.json"
if python3 -c "import json,sys; json.load(open('$IT_SETTINGS'))" 2>/dev/null; then
  ok "settings.json JSON válido"
else
  fail "settings.json JSON inválido — restaurar desde backup"
fi

# Check profiles visible
VIS=$(python3 -c "
import json
with open('$IT_SETTINGS') as f: s = json.load(f)
print(sum(1 for p in s.get('profiles',{}).get('list',[]) if not p.get('hidden')))
")
HID=$(python3 -c "
import json
with open('$IT_SETTINGS') as f: s = json.load(f)
print(sum(1 for p in s.get('profiles',{}).get('list',[]) if p.get('hidden')))
")
echo "  Profiles: $VIS visibles, $HID hidden"
[ "$VIS" -ge 1 ] || fail "Cero profiles visibles — restaurar backup o regenerar"

# ── 3. Validar GUIDs ───────────────────────────────────────
log "Paso 3: Validar todos los GUIDs"
INVALID=$(python3 -c "
import json, re
p = '$IT_SETTINGS'
with open(p) as f: s = json.load(f)
pat = re.compile(r'^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$')
bad = [pr['name'] for pr in s['profiles']['list'] if not pat.match(pr.get('guid',''))]
print('|'.join(bad))
")
if [ -n "$INVALID" ]; then
  fail "GUIDs inválidos en: $INVALID"
fi
ok "todos los GUIDs válidos"

# ── 4. Reset settingsHash en state.json ────────────────────
log "Paso 4: Reset settingsHash → forzar IT a re-validar todo"
python3 -c "
import json
p = '$IT_STATE'
with open(p) as f: s = json.load(f)
s['settingsHash'] = ''  # empty hash forces IT to rescan and recompute
with open(p, 'w') as f: json.dump(s, f, indent=1)
print('  settingsHash cleared')
"
ok "state.json con settingsHash=''"

# ── 5. Touch settings.json (mtime update) ──────────────────
log "Paso 5: Touch settings.json (mtime refresh)"
powershell.exe -NoProfile -Command "(Get-Item 'C:\Users\Alexa\AppData\Local\Packages\Microsoft.IntelligentTerminal_8wekyb3d8bbwe\LocalState\settings.json').LastWriteTime = Get-Date" 2>&1 | head -3

# ── 6. Lanzar nueva ventana IT (NO matar las existentes) ──
log "Paso 6: Lanzar nueva ventana IT para forced-reload"
powershell.exe -NoProfile -Command "
Start-Process 'shell:AppsFolder\Microsoft.IntelligentTerminal_8wekyb3d8bbwe!App'
Start-Sleep -Seconds 3
\$running = Get-Process | Where-Object { \$_.Name -match '^WindowsTerminal$' } | Measure-Object
Write-Host \"IT windows alive: \$(\$running.Count)\"
" 2>&1 | head -5
ok "Nueva ventana IT abierta (sin tocar las tuyas)"

# ── 7. Verify final ─────────────────────────────────────────
log "Paso 7: Verificación final"

IT_OK=$(python3 -c "
import json, re
p = '$IT_SETTINGS'
with open(p) as f: s = json.load(f)
pat = re.compile(r'^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$')
n_visible = sum(1 for x in s['profiles']['list'] if not x.get('hidden'))
all_guids_ok = all(pat.match(p.get('guid','')) for p in s['profiles']['list'])
print('OK' if (n_visible >= 1 and all_guids_ok) else 'FAIL')
")
[ "$IT_OK" = "OK" ] && ok "Settings IT verificadas" || fail "Settings IT corruptas"

# ── Summary ─────────────────────────────────────────────────
cat <<EOF

═══════════════════════════════════════════════════════════
  OVAV IT Cache Clean — Done
═══════════════════════════════════════════════════════════
  Backup:        $BACKUP
  Visible:       $VIS
  Hidden:        $HID
  Settings hash: cleared (IT will force-reload)
  Windows alive: NO se mató ninguna ventana existente

  Acción para CEO:
  • Esperá ~30 segundos — IT debería recomputar hash
  • Si todavía ves error en ventana vieja, cerrá SOLO esa ventana
    (no se cierran otras pestañas/ventanas tuyas)
  • El error "All profiles hidden" debería desaparecer al:
      - lanzar nueva ventana IT (ya lanzado), O
      - cerrar la ventana específica que muestra el error

═══════════════════════════════════════════════════════════
EOF