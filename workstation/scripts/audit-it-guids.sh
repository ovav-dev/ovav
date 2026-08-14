#!/usr/bin/env bash
# IT Settings GUID Audit — Detecta GUIDs inválidos en settings.json de IT
# Uso: bash fix-it-guids.sh              [repara + muestra backups]
#      bash fix-it-guids.sh --audit-only [solo verifica, no modifica]
set -euo pipefail

IT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"

if [ ! -f "$IT_SETTINGS" ]; then
  echo "ERROR: No encuentro el archivo en:"
  echo "  $IT_SETTINGS"
  exit 1
fi

echo "═══════════════════════════════════════════════════════════"
echo "  Intelligent Terminal — GUID Audit"
echo "═══════════════════════════════════════════════════════════"

# Detección con python (jq regex es frágil para este caso)
INVALID=$(python3 << 'PYEOF'
import json, re
with open("/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json") as f:
    s = json.load(f)
pat = re.compile(r"^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$")
bad = []
for p in s.get("profiles",{}).get("list",[]):
    if not pat.match(p.get("guid","")):
        bad.append((p.get("name","?"), p.get("guid","")))
print(json.dumps(bad))
PYEOF
)

if [ "$INVALID" = "[]" ]; then
  echo "  ✅ Todos los GUIDs son válidos. IT puede leer config OK."
  exit 0
fi

echo ""
echo "  ⚠️  GUIDs inválidos detectados:"
echo "$INVALID" | python3 -c "
import json, sys
items = json.load(sys.stdin)
for n, g in items:
    print(f'    • {n}: {g}')
print(f'  Total: {len(items)}')
"

if [ "${1:-}" = "--audit-only" ]; then
  echo ""
  echo "  Modo: --audit-only (no se modifica nada)"
  exit 1
fi

# Si pidió arreglar: delega a fix-it-guids.sh
echo ""
echo "  Aplicando fix-it-guids.sh..."
bash /home/braka/Systems/ovav/workstation/scripts/fix-it-guids.sh