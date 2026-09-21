#!/usr/bin/env python3
"""Refined IT settings.json — Phase 5-10 surgical update v2.

Decision tree per CEO spec section 16:
- "If current API doesn't allow 'create only if missing': don't implement fragile logic."
- "Create explicit 'OVAV Workspace' action for initialization"
- "Use Alt+1/2/3 exclusively for navigation"

Implementation:
- Alt+1/2/3 → Terminal.SwitchToTab0/1/2 (navigation ONLY, no creation)
- Alt+A → OVAV.split.smart (splitPane auto, duplicate, same cwd)
- OVAV Workspace → SHELL SCRIPT in $OVAV_ROOT/bin/ovav-workspace that uses wta.exe
- Ctrl+Alt+Shift+W → OVAV Workspace shell command (via 'commandline' action)
"""

import json
from pathlib import Path

WT_ROOT = Path("/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization")
DEST = WT_ROOT / ".ovav-ux-convergence"
SRC = DEST / "IT-settings.json"
OUT = DEST / "IT-settings.json.v2"

with SRC.open() as f:
    cfg = json.load(f)

# Profile GUIDs
profiles_by_name = {p["name"]: p["guid"] for p in cfg["profiles"]["list"]}
OVAV_GUID = profiles_by_name["OVAV"]

# ─────────────────────────────────────────────────────────────────
# SURGICAL CHANGES v2
# ─────────────────────────────────────────────────────────────────

# CHANGE 1: REMOVE unconditional Ctrl+C → Copy
# Restore WT default: copy if selection, else SIGINT
before = len(cfg["keybindings"])
cfg["keybindings"] = [
    kb for kb in cfg["keybindings"]
    if not (kb.get("keys") == "ctrl+c" and kb.get("id") == "Terminal.CopyToClipboard")
]
print(f"CHANGE 1: Removed {before - len(cfg['keybindings'])} Ctrl+C unconditional copy binding(s).")

# CHANGE 2: Remove the invalid OVAV.profile.OVAV (switchToTab with profile doesn't work)
cfg["actions"] = [a for a in cfg["actions"] if a.get("id") != "OVAV.profile.OVAV"]
print(f"CHANGE 2: Removed invalid OVAV.profile.OVAV action (switchToTab needs index, not profile).")

# CHANGE 3a: Replace OVAV.workspace.OVAV to use commandline (shell script)
# WT supports "commandline" action that opens a new tab with a command
# We use this to spawn wta.exe that creates the 3 tabs
ovav_workspace_cmd = (
    "wsl.exe -d Ubuntu-26.04 bash -lc '"
    "OVAV_ROOT=/home/braka/Systems/ovav; "
    "PATH=$OVAV_ROOT/bin:$PATH; "
    "ovav-workspace'"
)
# Replace OVAV.workspace.OVAV with a commandline-based action
cfg["actions"] = [
    a for a in cfg["actions"] if a.get("id") != "OVAV.workspace.OVAV"
]
cfg["actions"].append({
    "command": {
        "action": "newTab",
        "commandline": ovav_workspace_cmd,
        "profile": OVAV_GUID
    },
    "id": "OVAV.workspace.init"
})
print("CHANGE 3a: OVAV.workspace.init now runs 'ovav-workspace' shell script.")

# CHANGE 3b: Add OVAV.split.smart action (splitPane auto, duplicate, same cwd)
# Per CEO: Alt+A → splitPane split=auto splitMode=duplicate
# splitMode="duplicate" preserves the current pane's CWD (CRITICAL for the spec)
cfg["actions"].append({
    "command": {
        "action": "splitPane",
        "split": "auto",
        "splitMode": "duplicate"
    },
    "id": "OVAV.split.smart"
})
print("CHANGE 3b: OVAV.split.smart → splitPane auto, duplicate (preserves CWD).")

# CHANGE 4: Add tabWidthMode compact + Mica (preserve previous work)
cfg["profiles"]["defaults"]["tabWidthMode"] = "compact"
for theme in cfg["themes"]:
    theme["window"]["useMica"] = True
print("CHANGE 4: tabWidthMode=compact, useMica=true in both themes.")

# CHANGE 5: Add OPS, OpenCode, Scratch profiles
ops_guid = "{6f3a8e2c-9b40-4e2e-9d8a-bf5e6c7d8e9f}"
opencode_guid = "{7a4b9f3d-8c5e-4f3a-9c8b-ae6f7d8e9fa0}"
scratch_guid = "{8b5c0a4e-9d6f-4a4b-ad9c-bf7e8f9e0ab1}"

new_profiles = [
    {
        "commandline": "wsl.exe -d Ubuntu-26.04",
        "startingDirectory": "//wsl.localhost/Ubuntu-26.04/home/braka/Systems/ovav",
        "guid": ops_guid,
        "name": "OPS",
        "font": {"face": "Cascadia Mono", "size": 12}
    },
    {
        "commandline": "wsl.exe -d Ubuntu-26.04 /home/braka/.opencode/bin/opencode",
        "startingDirectory": "//wsl.localhost/Ubuntu-26.04/home/braka/Systems/ovav",
        "guid": opencode_guid,
        "name": "OpenCode",
        "font": {"face": "Cascadia Mono", "size": 12}
    },
    {
        "commandline": "wsl.exe -d Ubuntu-26.04",
        "startingDirectory": "//wsl.localhost/Ubuntu-26.04/home/braka",
        "guid": scratch_guid,
        "name": "Scratch",
        "font": {"face": "Cascadia Mono", "size": 12}
    }
]
existing_guids = {p.get("guid") for p in cfg["profiles"]["list"]}
for np in new_profiles:
    if np["guid"] not in existing_guids:
        cfg["profiles"]["list"].append(np)
print(f"CHANGE 5: Added {sum(1 for p in new_profiles if p['guid'] not in existing_guids)} profiles (OPS, OpenCode, Scratch).")

# CHANGE 6: Alt+1/2/3 + Alt+A + Ctrl+Alt+Shift+W keybindings
# Remove any existing Alt+1/2/3 (we'll add fresh)
cfg["keybindings"] = [
    kb for kb in cfg["keybindings"]
    if kb.get("keys") not in ["alt+1", "alt+2", "alt+3", "alt+a"]
]
cfg["keybindings"].extend([
    {"id": "Terminal.SwitchToTab0",     "keys": "alt+1"},
    {"id": "Terminal.SwitchToTab1",     "keys": "alt+2"},
    {"id": "Terminal.SwitchToTab2",     "keys": "alt+3"},
    {"id": "OVAV.split.smart",          "keys": "alt+a"},
    {"id": "OVAV.workspace.init",       "keys": "ctrl+alt+shift+w"},
])
print("CHANGE 6: Added Alt+1/2/3 (tab nav), Alt+A (split), Ctrl+Alt+Shift+W (Workspace init).")

# CHANGE 7: Preserve existing OVAV.tab, HUB.tab, SYS.tab, WIN.tab, close-others actions
# These were already in the file. Verify they're still there.
existing_ids = {a["id"] for a in cfg["actions"]}
for required in ["OVAV.tab", "HUB.tab", "SYS.tab", "WIN.tab", "OVAV.close-others", "OVAV.menu",
                  "OVAV.workspace.init", "OVAV.split.smart"]:
    if required not in existing_ids:
        print(f"WARNING: Missing required action {required}")
    else:
        print(f"  OK: action {required} present")

# Save
with OUT.open("w") as f:
    json.dump(cfg, f, indent=4)
    f.write("\n")

print(f"\n=== v2 saved to: {OUT} ===")
print(f"Actions: {len(cfg['actions'])}")
print(f"Keybindings: {len(cfg['keybindings'])}")
print(f"Profiles: {len(cfg['profiles']['list'])}")
print(f"Themes: {len(cfg['themes'])}")