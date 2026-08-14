#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  IT Settings GUID Audit — Detecta GUIDs inválidos en settings.json
#
#  Required environment variables (no defaults — user must provide):
#    OVAV_IT_SETTINGS  absolute path to settings.json
#
#  Optional environment variables:
#    OVAV_AUDIT_MODE   "audit-only" (default) | "fix"
#
#  External dependencies (must be on PATH):
#    bash   ≥ 4.0
#    python3 (used for JSON inspection; required)
#
#  This script is registered under .ovav/registry/tool_configs.yaml
#  → ovav_workstation_scripts. It is NEVER auto-run by OVAV.
#
#  Usage:
#    OVAV_IT_SETTINGS="/path/to/settings.json" bash audit-it-guids.sh
#    OVAV_AUDIT_MODE=fix OVAV_IT_SETTINGS="..." bash audit-it-guids.sh
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# ── Resolve SCRIPT_DIR (no hardcoded paths) ────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Required env vars ───────────────────────────────────────
if [ -z "${OVAV_IT_SETTINGS:-}" ]; then
  echo "ERROR: OVAV_IT_SETTINGS env var is required (absolute path to settings.json)." >&2
  echo "  Example: OVAV_IT_SETTINGS=\"/mnt/c/Users/.../settings.json\" bash $0" >&2
  exit 2
fi

if [ ! -f "$OVAV_IT_SETTINGS" ]; then
  echo "ERROR: settings.json not found at: $OVAV_IT_SETTINGS" >&2
  exit 1
fi

# ── Optional env vars with documented defaults ─────────────
OVAV_AUDIT_MODE="${OVAV_AUDIT_MODE:-audit-only}"

# ── Dependency check (no implicit assumptions) ──────────────
command -v python3 >/dev/null 2>&1 || {
  echo "ERROR: python3 is required but not found on PATH." >&2
  exit 3
}

# ── Audit ───────────────────────────────────────────────────
echo "═══════════════════════════════════════════════════════════"
echo "  Intelligent Terminal — GUID Audit"
echo "  settings.json: $OVAV_IT_SETTINGS"
echo "═══════════════════════════════════════════════════════════"

INVALID=$(OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 << 'PYEOF'
import json, os, re
settings_path = os.environ["OVAV_IT_SETTINGS"]
with open(settings_path) as f:
    s = json.load(f)
pat = re.compile(r"^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$")
bad = []
for p in s.get("profiles", {}).get("list", []):
    if not pat.match(p.get("guid", "")):
        bad.append((p.get("name", "?"), p.get("guid", "")))
print(json.dumps(bad))
PYEOF
)

if [ "$INVALID" = "[]" ]; then
  echo "  ✅ All GUIDs are valid. IT can read config OK."
  exit 0
fi

echo ""
echo "  ⚠️  Invalid GUIDs detected:"
echo "$INVALID" | OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 -c "
import json, os, sys
items = json.load(sys.stdin)
for n, g in items:
    print(f'    • {n}: {g}')
print(f'  Total: {len(items)}')
"

if [ "$OVAV_AUDIT_MODE" = "audit-only" ]; then
  echo ""
  echo "  Mode: audit-only (no modifications)"
  exit 1
fi

# ── Delegated fix (sibling script via SCRIPT_DIR, no hardcoded path) ──
echo ""
echo "  Delegating to fix-it-guids.sh (sibling script)..."
FIX_SCRIPT="$SCRIPT_DIR/fix-it-guids.sh"
if [ ! -x "$FIX_SCRIPT" ]; then
  echo "ERROR: fix-it-guids.sh not found or not executable at: $FIX_SCRIPT" >&2
  exit 4
fi
OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" bash "$FIX_SCRIPT"
