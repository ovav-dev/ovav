#!/usr/bin/env python3
"""
OVAV Theme Engine — Canonical visual output generator.

Lee .ovav/visual/theme/theme.yaml y expone:
  1. API Python para colores, estados, agentes
  2. Generador de tema OpenCode (.opencode/themes/ovav.json)
  3. Formateo Rich para terminal/dashboard

OVAV-first: este motor es la fuente de verdad visual.
Los clientes (OpenCode, terminal, web) consumen desde aquí.

Usage:
    python3 tools/visual/theme_engine.py --validate      # Validar theme.yaml
    python3 tools/visual/theme_engine.py --generate opencode  # Generar tema OpenCode
    python3 tools/visual/theme_engine.py --generate terminal  # Mostrar paleta en terminal
    python3 tools/visual/theme_engine.py --json          # Exportar como JSON

Location: tools/visual/theme_engine.py
Depends on: .ovav/visual/theme/theme.yaml
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

import yaml

ROOT = Path(__file__).resolve().parents[2]
THEME_PATH = ROOT / ".ovav" / "visual" / "theme" / "theme.yaml"


# ══════════════════════════════════════════════════════════════════════════════
# THEME LOADER
# ══════════════════════════════════════════════════════════════════════════════

class OVAVTheme:
    """Carga y expone el tema visual canónico OVAV."""

    def __init__(self):
        if not THEME_PATH.exists():
            raise FileNotFoundError(f"Theme not found: {THEME_PATH}")
        with open(THEME_PATH) as f:
            self._raw = yaml.safe_load(f)
        self._validate()

    def _validate(self):
        """Validación básica de integridad del tema."""
        required = ["schema", "name", "version", "brand", "semantic", "surfaces"]
        missing = [k for k in required if k not in self._raw]
        if missing:
            raise ValueError(f"Theme missing required keys: {missing}")
        if self._raw["schema"] != "ovav.visual.theme.v1":
            raise ValueError(f"Unknown schema: {self._raw['schema']}")

    # ── Brand Colors ─────────────────────────────────────────────────────
    def brand(self, key: str) -> str:
        return self._raw["brand"].get(key, self._raw["semantic"]["info"])

    # ── Semantic Colors ──────────────────────────────────────────────────
    def semantic(self, key: str) -> str:
        return self._raw["semantic"].get(key, self._raw["semantic"]["info"])

    # ── Surface Colors ───────────────────────────────────────────────────
    def surface(self, key: str, mode: str = "dark") -> str:
        return self._raw["surfaces"][mode].get(key, "#ffffff")

    # ── Agent Colors ─────────────────────────────────────────────────────
    def agent(self, name: str) -> dict[str, str]:
        return self._raw["agents"].get(name, {"color": "#ffffff", "icon": "?", "label": name})

    # ── Status Colors ────────────────────────────────────────────────────
    def status(self, key: str) -> str:
        return self._raw["status"].get(key, self._raw["semantic"]["info"])

    # ── Budget ───────────────────────────────────────────────────────────
    def budget_threshold(self, percent: float) -> dict:
        for name, t in self._raw["budget"]["thresholds"].items():
            if percent <= t["max"]:
                return {"level": name, **t}
        return {"level": "critical", "max": 100, "color": "#bf616a", "icon": "🔴"}

    # ── Raw Access ───────────────────────────────────────────────────────
    def raw(self) -> dict[str, Any]:
        return self._raw


# ══════════════════════════════════════════════════════════════════════════════
# OPENCODE THEME GENERATOR
# ══════════════════════════════════════════════════════════════════════════════

def generate_opencode_theme(theme: OVAVTheme) -> dict[str, Any]:
    """Convierte el tema OVAV al formato de tema JSON de OpenCode."""
    s = theme._raw["surfaces"]["dark"]
    sem = theme._raw["semantic"]
    syntax = theme._raw["syntax"]
    diff = theme._raw["diff"]
    brand = theme._raw["brand"]

    return {
        "$schema": "https://opencode.ai/theme.json",
        "name": "OVAV",
        "defs": {
            # Brand
            "ovav_teal": brand["thavren"],
            "ovav_green": brand["eidren"],
            "ovav_blue": brand["ovav_core"],
            "ovav_rose": brand["ovav_accent"],
            # Semantic
            "ovav_success": sem["success"],
            "ovav_error": sem["error"],
            "ovav_warning": sem["warning"],
            "ovav_info": sem["info"],
            # Surfaces
            "ovav_bg": s["bg_root"],
            "ovav_panel": s["bg_panel"],
            "ovav_element": s["bg_element"],
            "ovav_border": s["border"],
            "ovav_text": s["text_primary"],
            "ovav_text_secondary": s["text_secondary"],
            "ovav_text_muted": s["text_muted"],
            # Syntax
            "ovav_syn_keyword": syntax["keyword"],
            "ovav_syn_string": syntax["string"],
            "ovav_syn_comment": syntax["comment"],
            "ovav_syn_function": syntax["function"],
            "ovav_syn_type": syntax["type"],
            # Diff
            "ovav_diff_added": diff["added"],
            "ovav_diff_removed": diff["removed"],
            "ovav_diff_context": diff["context"],
        },
        "theme": {
            # Primary / Secondary
            "primary": {"dark": "ovav_teal", "light": "ovav_blue"},
            "secondary": {"dark": "ovav_green", "light": "ovav_green"},
            "accent": {"dark": "ovav_rose", "light": "ovav_rose"},
            # Semantic
            "error": {"dark": "ovav_error", "light": "ovav_error"},
            "warning": {"dark": "ovav_warning", "light": "ovav_warning"},
            "success": {"dark": "ovav_success", "light": "ovav_success"},
            "info": {"dark": "ovav_info", "light": "ovav_info"},
            # Text
            "text": {"dark": "ovav_text", "light": "ovav_bg"},
            "textMuted": {"dark": "ovav_text_muted", "light": "ovav_text_muted"},
            # Backgrounds
            "background": {"dark": "ovav_bg", "light": "ovav_text"},
            "backgroundPanel": {"dark": "ovav_panel", "light": "ovav_panel"},
            "backgroundElement": {"dark": "ovav_element", "light": "ovav_element"},
            # Borders
            "border": {"dark": "ovav_border", "light": "ovav_border"},
            "borderActive": {"dark": "ovav_teal", "light": "ovav_blue"},
            "borderSubtle": {"dark": "ovav_element", "light": "ovav_element"},
            # Diff
            "diffAdded": {"dark": "ovav_diff_added", "light": "ovav_diff_added"},
            "diffRemoved": {"dark": "ovav_diff_removed", "light": "ovav_diff_removed"},
            "diffContext": {"dark": "ovav_diff_context", "light": "ovav_diff_context"},
            "diffHunkHeader": {"dark": "ovav_blue", "light": "ovav_blue"},
            "diffHighlightAdded": {"dark": "ovav_diff_added", "light": "ovav_diff_added"},
            "diffHighlightRemoved": {"dark": "ovav_diff_removed", "light": "ovav_diff_removed"},
        },
    }


# ══════════════════════════════════════════════════════════════════════════════
# TERMINAL PALETTE RENDERER
# ══════════════════════════════════════════════════════════════════════════════

def render_palette(theme: OVAVTheme) -> str:
    """Renderiza la paleta OVAV en terminal con colores reales usando Rich."""
    try:
        from rich.console import Console
        from rich.panel import Panel
        from rich.table import Table
        from rich.text import Text

        console = Console()
        out: list[str] = []

        # Brand
        brand_table = Table(title="🎨 OVAV Brand Colors", show_header=True, header_style="bold")
        brand_table.add_column("Token", style="dim")
        brand_table.add_column("Color")
        brand_table.add_column("Hex")
        for key, hex_color in theme._raw["brand"].items():
            brand_table.add_row(
                key,
                Text("████████████████████", style=f"bold {hex_color}"),
                hex_color,
            )
        console.begin_capture()
        console.print(brand_table)

        # Semantic
        sem_table = Table(title="🔴🟡🟢 Semantic Colors", show_header=True, header_style="bold")
        sem_table.add_column("Token", style="dim")
        sem_table.add_column("Color")
        sem_table.add_column("Hex")
        for key, hex_color in theme._raw["semantic"].items():
            sem_table.add_row(
                key,
                Text("████████████████████", style=f"bold {hex_color}"),
                hex_color,
            )

        console.print(sem_table)

        # Agents
        agent_table = Table(title="🧠 Agent Identity Colors", show_header=True, header_style="bold")
        agent_table.add_column("Agent", style="dim")
        agent_table.add_column("Icon")
        agent_table.add_column("Color")
        agent_table.add_column("Hex")
        for name, info in theme._raw["agents"].items():
            agent_table.add_row(
                name,
                info["icon"],
                Text("████████████████████", style=f"bold {info['color']}"),
                info["color"],
            )
        console.print(agent_table)

        # Status
        status_table = Table(title="📊 Status Colors", show_header=True, header_style="bold")
        status_table.add_column("Status", style="dim")
        status_table.add_column("Color")
        status_table.add_column("Hex")
        for key, hex_color in theme._raw["status"].items():
            status_table.add_row(
                key.upper(),
                Text("████████████████████", style=f"bold {hex_color}"),
                hex_color,
            )
        console.print(status_table)

        captured = console.end_capture()
        return captured if isinstance(captured, str) else captured.get()

    except ImportError:
        # Fallback sin Rich
        lines = ["OVAV Theme Palette (no rich — install with: pip install rich)", "=" * 60]
        lines.append("\n🎨 Brand:")
        for k, v in theme._raw["brand"].items():
            lines.append(f"  {k:25s} {v}")
        lines.append("\n🔴🟡🟢 Semantic:")
        for k, v in theme._raw["semantic"].items():
            lines.append(f"  {k:25s} {v}")
        lines.append("\n🧠 Agents:")
        for name, info in theme._raw["agents"].items():
            lines.append(f"  {info['icon']} {name:22s} {info['color']}")
        return "\n".join(lines)


# ══════════════════════════════════════════════════════════════════════════════
# CLI
# ══════════════════════════════════════════════════════════════════════════════

def main():
    import argparse

    parser = argparse.ArgumentParser(description="OVAV Theme Engine")
    parser.add_argument("--validate", action="store_true", help="Validar theme.yaml")
    parser.add_argument("--generate", choices=["opencode", "terminal"], help="Generar salida para cliente")
    parser.add_argument("--json", action="store_true", help="Exportar tema completo como JSON")
    parser.add_argument("--output", type=Path, help="Archivo de salida (si no se especifica, stdout)")

    args = parser.parse_args()

    try:
        theme = OVAVTheme()
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(1)

    if args.validate:
        print(f"✅ Theme valid: {theme._raw['name']} v{theme._raw['version']}")
        print(f"   Schema: {theme._raw['schema']}")
        print(f"   Brand colors: {len(theme._raw['brand'])}")
        print(f"   Semantic colors: {len(theme._raw['semantic'])}")
        print(f"   Agents: {len(theme._raw['agents'])}")
        print(f"   Status colors: {len(theme._raw['status'])}")
        return

    if args.generate == "opencode":
        result = generate_opencode_theme(theme)
        output = json.dumps(result, indent=2)
        if args.output:
            args.output.write_text(output)
            print(f"✅ OpenCode theme written to: {args.output}")
        else:
            print(output)

    elif args.generate == "terminal":
        output = render_palette(theme)
        print(output)

    elif args.json:
        print(json.dumps(theme.raw(), indent=2))


if __name__ == "__main__":
    main()
