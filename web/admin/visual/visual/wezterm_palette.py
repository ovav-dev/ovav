#!/usr/bin/env python3
"""
OVAV WezTerm Color Generator — lee theme.yaml y genera paleta Lua.

Usage:
    python3 tools/visual/wezterm_palette.py              # stdout: Lua table
    python3 tools/visual/wezterm_palette.py --write      # Write to config/wezterm/ovav-palette.lua
    python3 tools/visual/wezterm_palette.py --preview    # Preview mappings
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[2]
THEME_PATH = ROOT / ".ovav" / "visual" / "theme" / "theme.yaml"
PALETTE_PATH = ROOT / "config" / "wezterm" / "ovav-palette.lua"


def load_theme() -> dict:
    with open(THEME_PATH) as f:
        return yaml.safe_load(f)


def generate_palette_lua(theme: dict) -> str:
    """Generate a WezTerm Lua color palette from OVAV theme.yaml."""
    surfaces = theme.get("surfaces", {}).get("dark", {})
    semantic = theme.get("semantic", {})
    brand = theme.get("brand", {})
    syntax = theme.get("syntax", {})
    status = theme.get("status", {})

    lines = []
    lines.append(
        "-- ═══════════════════════════════════════════════════════════════════════════════"
    )
    lines.append("-- OVAV WezTerm Palette — Auto-generated from .ovav/visual/theme/theme.yaml")
    lines.append("-- DO NOT EDIT DIRECTLY. Source of truth: .ovav/visual/theme/theme.yaml")
    lines.append("-- Generated via tools/visual/wezterm_palette.py")
    lines.append(
        "-- ═══════════════════════════════════════════════════════════════════════════════"
    )
    lines.append("")
    lines.append("local P = {")
    lines.append("")
    lines.append("  -- ═══ Surface Colors ═══")
    lines.append(f'  bg        = "{surfaces.get("bg_root", "#1d2021")}",')
    lines.append(f'  bg_dark   = "{surfaces.get("bg_panel", "#282828")}",')
    lines.append(f'  bg_alt    = "{surfaces.get("bg_element", "#3c3836")}",')
    lines.append(f'  bg_sel    = "{surfaces.get("bg_hover", "#504945")}",')
    lines.append(f'  fg        = "{surfaces.get("text_primary", "#ebdbb2")}",')
    lines.append(f'  fg_dim    = "{surfaces.get("text_secondary", "#bdae93")}",')
    lines.append(f'  fg_gutter = "{surfaces.get("text_muted", "#7c6f64")}",')
    lines.append("")
    lines.append("  -- ═══ Semantic ═══")
    lines.append(f'  cyan      = "{brand.get("ovav_core", "#458588")}",')
    lines.append(f'  blue      = "{semantic.get("info", "#81a1c1")}",')
    lines.append(f'  green     = "{semantic.get("success", "#a3be8c")}",')
    lines.append(f'  yellow    = "{semantic.get("highlight", "#ebcb8b")}",')
    lines.append(f'  orange    = "{semantic.get("warning", "#d08770")}",')
    lines.append(f'  red       = "{semantic.get("error", "#bf616a")}",')
    lines.append(f'  purple    = "{brand.get("ovav_accent", "#d3869b")}",')
    lines.append(f'  magenta   = "{brand.get("ovav_accent", "#d3869b")}",')
    lines.append(f'  teal      = "{brand.get("thavren", "#83a598")}",')
    lines.append(f'  white     = "{surfaces.get("text_primary", "#ebdbb2")}",')
    lines.append("")
    lines.append("  -- ═══ Element Surfaces ═══")
    lines.append(f'  surface0  = "{surfaces.get("bg_panel", "#282828")}",')
    lines.append(f'  surface1  = "{surfaces.get("bg_element", "#3c3836")}",')
    lines.append(f'  surface2  = "{surfaces.get("bg_hover", "#504945")}",')
    lines.append(f'  overlay0  = "{surfaces.get("bg_selected", "#665c54")}",')
    lines.append(f'  overlay1  = "{surfaces.get("border", "#3c3836")}",')
    lines.append(f'  overlay2  = "{surfaces.get("text_muted", "#7c6f64")}",')
    lines.append("")
    lines.append("  -- ═══ Status ═══")
    lines.append(f'  error     = "{status.get("fail", semantic.get("error", "#bf616a"))}",')
    lines.append(f'  warning   = "{status.get("warn", semantic.get("warning", "#d08770"))}",')
    lines.append(f'  info      = "{status.get("info", semantic.get("info", "#81a1c1"))}",')
    lines.append(f'  hint      = "{status.get("pass", semantic.get("success", "#a3be8c"))}",')
    lines.append(f'  success   = "{status.get("pass", semantic.get("success", "#a3be8c"))}",')
    lines.append("")
    lines.append("  -- ═══ ANSI ═══")
    lines.append(f'  black     = "{surfaces.get("bg_root", "#1d2021")}",')
    lines.append(f'  red_b     = "{semantic.get("error", "#bf616a")}",')
    lines.append(f'  green_b   = "{semantic.get("success", "#a3be8c")}",')
    lines.append(f'  yellow_b  = "{semantic.get("highlight", "#ebcb8b")}",')
    lines.append(f'  blue_b    = "{semantic.get("info", "#81a1c1")}",')
    lines.append(f'  magenta_b = "{syntax.get("keyword", brand.get("ovav_accent", "#d3869b"))}",')
    lines.append(f'  cyan_b    = "{brand.get("ovav_core", "#458588")}",')
    lines.append(f'  white_b   = "{surfaces.get("text_secondary", "#bdae93")}",')
    lines.append("")
    lines.append("  -- ═══ Bright ANSI ═══")
    lines.append(f'  black_h   = "{surfaces.get("bg_element", "#3c3836")}",')
    lines.append(f'  red_h     = "{status.get("fail", "#bf616a")}",')
    lines.append(f'  green_h   = "{status.get("complete", "#a3be8c")}",')
    lines.append(f'  yellow_h  = "{semantic.get("highlight", "#ebcb8b")}",')
    lines.append(f'  blue_h    = "{semantic.get("info", "#81a1c1")}",')
    lines.append(f'  magenta_h = "{brand.get("ovav_accent", "#d3869b")}",')
    lines.append(f'  cyan_h    = "{brand.get("thavren", "#83a598")}",')
    lines.append(f'  white_h   = "{surfaces.get("text_primary", "#ebdbb2")}",')
    lines.append("}")
    lines.append("")
    lines.append("return P")
    lines.append("")

    return "\n".join(lines)


def preview(theme: dict) -> None:
    """Print a color preview showing the mapping."""
    surfaces = theme.get("surfaces", {}).get("dark", {})
    brand = theme.get("brand", {})

    print("OVAV Theme → WezTerm Palette Preview")
    print("=" * 50)
    print(f"{'WezTerm':20s} {'Hex':10s} {'OVAV Source'}")
    print("-" * 50)
    for name, hexval in [
        ("bg", surfaces.get("bg_root")),
        ("fg", surfaces.get("text_primary")),
        ("teal (Thavren)", brand.get("thavren")),
        ("green (Eidren)", brand.get("eidren")),
        ("purple (accent)", brand.get("ovav_accent")),
        ("cyan (core)", brand.get("ovav_core")),
    ]:
        hexstr = hexval or "—"
        print(f"{name:20s} {hexstr:10s} theme.yaml")


def validate_config_palette(config_path: Path, theme: dict) -> tuple[bool, list[str]]:
    """Validate that the palette defined in a WezTerm config.lua matches theme.yaml.

    Returns (is_valid, list_of_mismatches).
    """
    if not config_path.exists():
        return False, [f"Config not found: {config_path}"]

    content = config_path.read_text()

    # Extract the P = { ... } palette table
    import re

    match = re.search(r"local P = \{(.*?)\}", content, re.DOTALL)
    if not match:
        return False, ["No P = {...} palette table found in config"]

    palette_str = match.group(1)

    # Parse key-value pairs from the Lua table
    palette = {}
    for line in palette_str.split("\n"):
        line = line.strip()
        if "=" in line and not line.startswith("--"):
            m = re.match(r'(\w+)\s*=\s*[\'"]?([#0-9a-fA-F]+)[\'"]?', line)
            if m:
                key = m.group(1).strip()
                val = m.group(2).strip()
                if not val.startswith("#"):
                    val = "#" + val
                palette[key] = val.lower()

    # Build expected palette from theme.yaml
    surfaces = theme.get("surfaces", {}).get("dark", {})
    semantic = theme.get("semantic", {})
    brand = theme.get("brand", {})

    expected = {
        "bg": surfaces.get("bg_root", "").lower(),
        "bg_panel": surfaces.get("bg_panel", "").lower(),
        "bg_hover": surfaces.get("bg_hover", "").lower(),
        "fg": surfaces.get("text_primary", "").lower(),
        "fg_dim": surfaces.get("text_secondary", "").lower(),
        "fg_muted": surfaces.get("text_muted", "").lower(),
        "success": semantic.get("success", "").lower(),
        "error": semantic.get("error", "").lower(),
        "warning": semantic.get("warning", "").lower(),
        "info": semantic.get("info", "").lower(),
    }

    mismatches = []
    for key, expected_val in expected.items():
        actual = palette.get(key, "(missing)")
        if actual != expected_val and expected_val:
            mismatches.append(f"  {key}: config={actual} expected={expected_val}")

    return len(mismatches) == 0, mismatches


def main():
    import argparse

    parser = argparse.ArgumentParser(description="OVAV WezTerm Palette Generator")
    parser.add_argument("--write", action="store_true", help="Write palette to config/wezterm/")
    parser.add_argument("--preview", action="store_true", help="Preview color mappings")
    parser.add_argument(
        "--validate",
        action="store_true",
        help="Validate canonical config.lua palette against theme.yaml",
    )
    parser.add_argument(
        "--config",
        type=str,
        default=None,
        help="Path to wezterm config.lua (default: .ovav/visual/wezterm/config.lua)",
    )

    args = parser.parse_args()
    theme = load_theme()

    if args.validate:
        config_path = (
            Path(args.config)
            if args.config
            else ROOT / ".ovav" / "visual" / "wezterm" / "config.lua"
        )
        ok, details = validate_config_palette(config_path, theme)
        if ok:
            print(f"✅ Palette in {config_path.relative_to(ROOT)} matches theme.yaml")
        else:
            print(f"❌ Palette MISMATCH in {config_path.relative_to(ROOT)}:")
            for d in details:
                print(d)
            sys.exit(1)
        return
        preview(theme)
        return

    lua = generate_palette_lua(theme)

    if args.write:
        PALETTE_PATH.write_text(lua)
        print(f"Written: {PALETTE_PATH.relative_to(ROOT)}")
    else:
        print(lua)


if __name__ == "__main__":
    main()
