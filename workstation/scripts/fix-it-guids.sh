#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  IT Settings GUID Fix — Idempotent replacement of OVAV-* placeholder GUIDs
#  with valid Windows GUIDs (8-4-4-4-12 hex format with braces).
#
#  Required environment variables (no defaults — user must provide):
#    OVAV_IT_SETTINGS  absolute path to settings.json
#
#  Optional environment variables (with documented defaults):
#    OVAV_BACKUP_DIR       backup destination
#                           (default: $HOME/.ovav-backups/manual-<timestamp>)
#    OVAV_GUID_NAMESPACE   uuid5 namespace string
#                           (default: "ovav.intelligent-terminal")
#
#  External dependencies (must be on PATH):
#    bash   ≥ 4.0
#    python3 (required for atomic JSON manipulation)
#
#  This script is registered under .ovav/registry/tool_configs.yaml
#  → ovav_workstation_scripts. It is NEVER auto-run by OVAV.
#
#  Usage:
#    OVAV_IT_SETTINGS="/path/to/settings.json" bash fix-it-guids.sh
# ─────────────────────────────────────────────────────────────
set -euo pipefail

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
TS=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="${OVAV_BACKUP_DIR:-$HOME/.ovav-backups}"
BACKUP="$BACKUP_DIR/manual-${TS}"
GUID_NAMESPACE="${OVAV_GUID_NAMESPACE:-ovav.intelligent-terminal}"
mkdir -p "$BACKUP"

# ── Dependency check ────────────────────────────────────────
command -v python3 >/dev/null 2>&1 || {
  echo "ERROR: python3 is required but not found on PATH." >&2
  exit 3
}

# ── Backup (always, before any modification) ────────────────
cp -p "$OVAV_IT_SETTINGS" "$BACKUP/intel-terminal-settings-pre-guid-fix.bak"
echo "── backup: $BACKUP ──"

# ── Generate deterministic GUIDs (configurable namespace) ───
GUID_OVAV_UBUNTU=$(OVAV_GUID_NAMESPACE="$GUID_NAMESPACE" python3 -c '
import os, uuid
ns = os.environ["OVAV_GUID_NAMESPACE"]
print(str(uuid.uuid5(uuid.NAMESPACE_DNS, f"{ns}.profile.OVAV Ubuntu")))
')
GUID_OPENCODE=$(OVAV_GUID_NAMESPACE="$GUID_NAMESPACE" python3 -c '
import os, uuid
ns = os.environ["OVAV_GUID_NAMESPACE"]
print(str(uuid.uuid5(uuid.NAMESPACE_DNS, f"{ns}.profile.OpenCode Ubuntu")))
')
GUID_OVAV_SCRATCH=$(OVAV_GUID_NAMESPACE="$GUID_NAMESPACE" python3 -c '
import os, uuid
ns = os.environ["OVAV_GUID_NAMESPACE"]
print(str(uuid.uuid5(uuid.NAMESPACE_DNS, f"{ns}.profile.OVAV Scratch")))
')

echo "── deterministic GUIDs (uuid5, namespace='$GUID_NAMESPACE') ──"
echo "  OVAV Ubuntu       → {$GUID_OVAV_UBUNTU}"
echo "  OpenCode Ubuntu   → {$GUID_OPENCODE}"
echo "  OVAV Scratch      → {$GUID_OVAV_SCRATCH}"

# ── Atomic JSON manipulation via python ─────────────────────
OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" \
GUID_OVAV_UBUNTU="$GUID_OVAV_UBUNTU" \
GUID_OPENCODE="$GUID_OPENCODE" \
GUID_OVAV_SCRATCH="$GUID_OVAV_SCRATCH" \
python3 << 'PYEOF'
import json, os

settings_path = os.environ["OVAV_IT_SETTINGS"]
guid_map = {
    "{OVAV-UBUNTU-PROFILE-0001-000000000001}": "{" + os.environ["GUID_OVAV_UBUNTU"] + "}",
    "{OVAV-OPENCODE-PROFILE-0001-000000000002}": "{" + os.environ["GUID_OPENCODE"] + "}",
    "{OVAV-SCRATCH-PROFILE-0001-000000000003}": "{" + os.environ["GUID_OVAV_SCRATCH"] + "}",
}

with open(settings_path) as f:
    settings = json.load(f)

replaced = 0
for prof in settings.get("profiles", {}).get("list", []):
    name = prof.get("name", "")
    guid = prof.get("guid", "")
    if guid in guid_map:
        prof["guid"] = guid_map[guid]
        replaced += 1
        print(f"  fixed: {name}: {guid} → {prof['guid']}")

def patch_json(obj):
    if isinstance(obj, dict):
        for k, v in list(obj.items()):
            if isinstance(v, str) and v in guid_map:
                obj[k] = guid_map[v]
                print(f"  patched {k}: {v} → {obj[k]}")
            else:
                patch_json(v)
    elif isinstance(obj, list):
        for x in obj:
            patch_json(x)

patch_json(settings)

with open(settings_path, "w") as f:
    json.dump(settings, f, indent=1)

print(f"\n── replaced {replaced} invalid GUID(s) ──")
PYEOF

echo ""
echo "── verify ──"
OVAV_IT_SETTINGS="$OVAV_IT_SETTINGS" python3 << 'PYEOF'
import json, os, re
settings_path = os.environ["OVAV_IT_SETTINGS"]
with open(settings_path) as f:
    s = json.load(f)
pat = re.compile(r"^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$")
for p in s["profiles"]["list"]:
    n = p.get("name", "?")
    g = p.get("guid", "")
    print("✓" if pat.match(g) else "✗", n, "→", g)
PYEOF
