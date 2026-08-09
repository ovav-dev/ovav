#!/usr/bin/env python3
"""Deploy full OVAV actions into Windows Terminal settings.json (merge, not overwrite)."""
import json, shutil
from datetime import datetime

DEPLOYED = "/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json"
TEMPLATE = "/home/braka/Systems/OVAV/.ovav/visual/windows-terminal/settings.json"

# Backup
shutil.copy2(DEPLOYED, DEPLOYED.replace(".json", f".bak-{datetime.now().strftime('%Y%m%d-%H%M')}"))

with open(DEPLOYED) as f:
    cfg = json.load(f)

with open(TEMPLATE) as f:
    tpl = json.load(f)

# Replace actions with full OVAV template
old_count = len(cfg.get("actions", []))
cfg["actions"] = tpl["actions"]
new_count = len(cfg["actions"])

# Remove old-format keybindings (conflicts with actions)
kb_count = len(cfg.get("keybindings", []))
cfg.pop("keybindings", None)

# Add missing beneficial settings from template
cfg["experimental.inputForcePassthrough"] = True
cfg["theme"] = "dark"

with open(DEPLOYED, "w") as f:
    json.dump(cfg, f, indent=4)

print(f"Actions: {old_count} → {new_count} (+{new_count - old_count})")
print(f"Old keybindings removed: {kb_count}")
print(f"inputForcePassthrough: enabled")
print("Restart Windows Terminal.")
