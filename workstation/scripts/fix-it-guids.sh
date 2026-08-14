#!/usr/bin/env bash
# IT Settings GUID fix — idempotent replacement of OVAV-* placeholder GUIDs
# with valid Windows GUIDs (8-4-4-4-12 hex format with braces).
set -euo pipefail

IT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/LocalState/settings.json"
TS=$(date +%Y%m%d-%H%M%S)
BACKUP="$HOME/.ovav-backups/manual-${TS}"
mkdir -p "$BACKUP"

if [ ! -f "$IT_SETTINGS" ]; then
  echo "ERROR: $IT_SETTINGS not found" >&2
  exit 1
fi

cp -p "$IT_SETTINGS" "$BACKUP/intel-terminal-settings-pre-guid-fix.bak"
echo "── backup: $BACKUP ──"

# Generate three deterministic GUIDs (so re-runs don't change values)
# using uuid5 with a fixed namespace = "ovav.intelligent-terminal"
GUID_OVAV_UBUNTU=$(python3 -c 'import uuid; print(str(uuid.uuid5(uuid.NAMESPACE_DNS, "ovav.it.profile.OVAV Ubuntu")))')
GUID_OPENCODE=$(python3 -c 'import uuid; print(str(uuid.uuid5(uuid.NAMESPACE_DNS, "ovav.it.profile.OpenCode Ubuntu")))')
GUID_OVAV_SCRATCH=$(python3 -c 'import uuid; print(str(uuid.uuid5(uuid.NAMESPACE_DNS, "ovav.it.profile.OVAV Scratch")))')

echo "── deterministic GUIDs (uuid5) ──"
echo "  OVAV Ubuntu       → {$GUID_OVAV_UBUNTU}"
echo "  OpenCode Ubuntu   → {$GUID_OPENCODE}"
echo "  OVAV Scratch      → {$GUID_OVAV_SCRATCH}"

# Use python for atomic JSON manipulation (preserves formatting better than jq)
python3 << EOF
import json, re, sys
with open("$IT_SETTINGS") as f:
    settings = json.load(f)

guid_map = {
    "{OVAV-UBUNTU-PROFILE-0001-000000000001}": "{$GUID_OVAV_UBUNTU}",
    "{OVAV-OPENCODE-PROFILE-0001-000000000002}": "{$GUID_OPENCODE}",
    "{OVAV-SCRATCH-PROFILE-0001-000000000003}": "{$GUID_OVAV_SCRATCH}",
}

replaced = 0
for prof in settings.get("profiles",{}).get("list",[]):
    name = prof.get("name","")
    guid = prof.get("guid","")
    if guid in guid_map:
        prof["guid"] = guid_map[guid]
        replaced += 1
        print(f"  fixed: {name}: {guid} → {prof['guid']}")

# Also patch keybindings/commands arrays if they reference old GUIDs
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

with open("$IT_SETTINGS", "w") as f:
    json.dump(settings, f, indent=1)

print(f"\n── replaced {replaced} invalid GUID(s) ──")
EOF

echo ""
echo "── verify ──"
python3 -c '
import re, json
with open("$IT_SETTINGS".replace("$IT_SETTINGS", "'"$IT_SETTINGS"'")) as f:
    s = json.load(f)
pat = re.compile(r"^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$")
for p in s["profiles"]["list"]:
    n = p.get("name","?")
    g = p.get("guid","")
    print("✓" if pat.match(g) else "✗", n, "→", g)
'