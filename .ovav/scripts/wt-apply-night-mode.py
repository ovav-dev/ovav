#!/usr/bin/env python3
"""Apply OVAV Pure Night theme + zero transparency to Windows Terminal settings."""
import json, shutil, sys
from datetime import datetime

WT_PATH = "/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.WindowsTerminal_8wekyb3d8bbwe/LocalState/settings.json"

# Backup
shutil.copy2(WT_PATH, WT_PATH.replace(".json", f".bak-{datetime.now().strftime('%Y%m%d-%H%M')}"))

with open(WT_PATH) as f:
    cfg = json.load(f)

defaults = cfg.setdefault("profiles", {}).setdefault("defaults", {})

# Zero transparency
defaults["opacity"] = 100
defaults["useAcrylic"] = False
defaults["backgroundImageOpacity"] = 0
defaults["background"] = "#0a0a0a"
defaults["foreground"] = "#e0e0e0"
defaults["colorScheme"] = "OVAV Pure Night"
defaults["selectionBackground"] = "#264f78"

# Add Pure Night scheme if missing
schemes = cfg.setdefault("schemes", [])
if not any(s.get("name") == "OVAV Pure Night" for s in schemes):
    schemes.insert(0, {
        "name": "OVAV Pure Night",
        "background": "#0a0a0a",
        "black": "#0a0a0a",
        "blue": "#569cd6",
        "brightBlack": "#1e1e1e",
        "brightBlue": "#6d9bc3",
        "brightCyan": "#4ec9b0",
        "brightGreen": "#7eb77f",
        "brightPurple": "#d4a0d0",
        "brightRed": "#ff6b6b",
        "brightWhite": "#f0f0f0",
        "brightYellow": "#e5c07b",
        "cursorColor": "#ffffff",
        "cyan": "#4ec9b0",
        "foreground": "#e0e0e0",
        "green": "#6a9955",
        "purple": "#c586c0",
        "red": "#f44747",
        "selectionBackground": "#264f78",
        "white": "#d4d4d4",
        "yellow": "#d4a85c"
    })

# Ensure copyOnSelect
cfg["copyOnSelect"] = True

with open(WT_PATH, "w") as f:
    json.dump(cfg, f, indent=4)

print("OVAV Pure Night applied. Restart Windows Terminal.")
