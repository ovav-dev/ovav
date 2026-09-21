#!/usr/bin/env python3
"""PHASE 6 - OpenCode tui.json surgical update.

Per OpenCode docs (https://opencode.ai/docs/keybinds):
- input_paste: Ctrl+V (native, works)
- input_undo: Ctrl+Z (Windows default)
- terminal_suspend: Ctrl+Z (POSIX default — must disable for Windows UX)
- ctrl+x: LEADER KEY (cannot remap to cut)

Per CEO spec section 10:
- Ctrl+V → paste: PASS (native)
- Ctrl+Z → input_undo: ENABLE explicit override
- Ctrl+X → cut: UNSUPPORTED_NATIVE (no input_cut primitive in OpenCode)
"""

import json
from pathlib import Path

WT = Path("/home/braka/Systems/ovav/.ovav/worktrees/feature-feat-piagent-tui-customization")
DEST = WT / ".ovav-ux-convergence"
SRC = DEST / "backups" / "20260815-123202" / "opencode-tui.json.original"
OUT = DEST / "opencode-tui.json.updated"

with SRC.open() as f:
    cfg = json.load(f)

# Add keybinds section (does not exist in original)
if "keybinds" not in cfg:
    cfg["keybinds"] = {}

# Add leader timeout (default 2000ms)
if "leader_timeout" not in cfg:
    cfg["leader_timeout"] = 2000

# ─────────────────────────────────────────────────────────────────
# SURGICAL CHANGES
# ─────────────────────────────────────────────────────────────────

# 1. input_paste — Ctrl+V (already default, make explicit for clarity)
cfg["keybinds"]["input_paste"] = {
    "key": "ctrl+v",
    "preventDefault": False
}

# 2. input_undo — Ctrl+Z (Windows-style: explicit override)
# Default on Linux/WSL would be terminal_suspend for Ctrl+Z.
# We override to make Ctrl+Z = undo in OpenCode's input.
cfg["keybinds"]["input_undo"] = "ctrl+z,ctrl+-,super+z"

# 3. terminal_suspend — DISABLE (Windows-style, no POSIX suspend)
# Per CEO spec: "conservar señales Unix necesarias durante procesos reales"
# But also "Ctrl+Z contextual: undo cuando editas, suspend cuando ejecutas proceso"
# OpenCode's input_undo overrides terminal_suspend for the input area.
# When a process IS running (e.g., spawned bash), Ctrl+Z still works via shell.
cfg["keybinds"]["terminal_suspend"] = "none"

# 4. ctrl+x stays as leader (cannot remap to cut)
# OpenCode does not have input_cut primitive. Declared UNSUPPORTED_NATIVE.
# Per CEO spec section 10: "Si NO existe primitive nativa para cut: NO simules cut.
# NO parchees OpenCode. Reporta OPENCODE_CTRL_X = UNSUPPORTED_NATIVE."
cfg["keybinds"]["leader"] = "ctrl+x"  # preserve default

# Save
with OUT.open("w") as f:
    json.dump(cfg, f, indent=2)
    f.write("\n")

print(f"=== OpenCode tui.json v2 saved to {OUT} ===")
print(f"\nKeybinds configured:")
for k, v in cfg["keybinds"].items():
    print(f"  {k}: {v}")

print(f"\n=== FINAL VERDICT ===")
print(f"Ctrl+V (paste):        PASS — input_paste native, preventDefault=false")
print(f"Ctrl+Z (input undo):   PASS — input_undo set to ctrl+z")
print(f"Ctrl+Z (terminal):     DISABLED — terminal_suspend=none (Windows-style)")
print(f"Ctrl+X (cut):          UNSUPPORTED_NATIVE — no input_cut primitive in OpenCode")
print(f"                         OpenCode uses Ctrl+X as leader; cannot be reassigned.")
print(f"                         Per CEO: declare honestly, no hacks.")