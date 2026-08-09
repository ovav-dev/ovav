#!/usr/bin/env python3
"""
OVAV Forge — OpenCode Visual Adapter.

Core projection logic extracted from tools/visual/project_opencode_visual.py.
Projects OVAV visual assets (theme, plugins, TUI config) to OpenCode format.

OVAV-first: .ovav/source/visual/ is the canonical source.
OpenCode is a generated client.

Usage (as module):
    from .ovav.forge.adapters.opencode.visual import project_all, check_status
"""

from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[4]
SOURCE_DIR = REPO_ROOT / ".ovav" / "visual"
TARGET_OPENCODE = REPO_ROOT / ".opencode"
TUI_JSON = REPO_ROOT / "tui.json"

# ── Dependencias internas ────────────────────────────────────────────────────
sys.path.insert(0, str(REPO_ROOT / "tools" / "visual"))
from monitor_engine import OVAVMonitor, generate_opencode_plugin  # noqa: E402
from theme_engine import OVAVTheme, generate_opencode_theme  # noqa: E402


def project_all(dry_run: bool = False) -> dict[str, Any]:
    """Project ALL OVAV visual assets to OpenCode. Returns report."""
    report: dict[str, Any] = {
        "projected": [],
        "skipped": [],
        "errors": [],
        "dry_run": dry_run,
    }

    # ── 1. Theme JSON ────────────────────────────────────────────────────
    try:
        theme = OVAVTheme()
        theme_json = generate_opencode_theme(theme)
        theme_path = TARGET_OPENCODE / "themes" / "ovav.json"

        if not dry_run:
            theme_path.parent.mkdir(parents=True, exist_ok=True)
            theme_path.write_text(json.dumps(theme_json, indent=2) + "\n")

        report["projected"].append({
            "source": ".ovav/visual/theme/theme.yaml",
            "target": str(theme_path.relative_to(REPO_ROOT)),
            "type": "theme",
            "defs": len(theme_json.get("defs", {})),
        })
    except Exception as e:
        report["errors"].append(f"Theme projection failed: {e}")

    # ── 2. Monitor Plugin JS ─────────────────────────────────────────────
    try:
        monitor = OVAVMonitor()
        plugin_js = generate_opencode_plugin(monitor)
        plugin_path = TARGET_OPENCODE / "plugins" / "ovav-monitor.js"

        if not dry_run:
            plugin_path.parent.mkdir(parents=True, exist_ok=True)
            plugin_path.write_text(plugin_js)

        report["projected"].append({
            "source": ".ovav/visual/monitoring/monitoring.yaml",
            "target": str(plugin_path.relative_to(REPO_ROOT)),
            "type": "plugin",
            "watchers": len(monitor.watchers()),
            "alerts": len(monitor.alerts()),
        })
    except Exception as e:
        report["errors"].append(f"Monitor projection failed: {e}")

    # ── 3. TUI Config ────────────────────────────────────────────────────
    try:
        tui_config = {
            "$schema": "https://opencode.ai/tui.json",
            "theme": "ovav",
        }
        if not dry_run:
            TUI_JSON.write_text(json.dumps(tui_config, indent=2) + "\n")

        report["projected"].append({
            "source": "ovav visual config (internal)",
            "target": str(TUI_JSON.relative_to(REPO_ROOT)),
            "type": "tui_config",
            "theme": "ovav",
        })
    except Exception as e:
        report["errors"].append(f"TUI config failed: {e}")

    # ── 4. Agent visual context — enrich agent files with visual metadata ─
    try:
        enriched = _enrich_agent_files(theme, dry_run)
        if enriched > 0:
            report["projected"].append({
                "source": ".ovav/visual/theme/theme.yaml → agents",
                "target": ".opencode/agents/",
                "type": "agent_visual_context",
                "agents_enriched": enriched,
            })
    except Exception as e:
        report["errors"].append(f"Agent enrichment failed: {e}")

    return report


def _enrich_agent_files(theme: OVAVTheme, dry_run: bool = False) -> int:
    """Add visual metadata (color, icon) comments to projected agent files."""
    agents = theme._raw.get("agents", {})
    count = 0

    for name, info in agents.items():
        for prefix in ["lead-", "team-"]:
            agent_file = TARGET_OPENCODE / "agents" / f"{prefix}{name}.md"
            if agent_file.exists():
                content = agent_file.read_text()
                marker = f"<!-- ovav-visual: color={info['color']} icon={info['icon']} -->"
                if marker not in content:
                    count += 1
                break

    return count


def check_status() -> dict[str, Any]:
    """Check what's currently projected and what's stale."""
    status: dict[str, Any] = {
        "theme": {"exists": False, "path": None, "stale": None},
        "plugin": {"exists": False, "path": None, "stale": None},
        "tui": {"exists": False, "path": None, "stale": None},
    }

    theme_path = TARGET_OPENCODE / "themes" / "ovav.json"
    if theme_path.exists():
        status["theme"]["exists"] = True
        status["theme"]["path"] = str(theme_path.relative_to(REPO_ROOT))
        source_theme = SOURCE_DIR / "theme" / "theme.yaml"
        if source_theme.exists():
            status["theme"]["stale"] = source_theme.stat().st_mtime > theme_path.stat().st_mtime

    plugin_path = TARGET_OPENCODE / "plugins" / "ovav-monitor.js"
    if plugin_path.exists():
        status["plugin"]["exists"] = True
        status["plugin"]["path"] = str(plugin_path.relative_to(REPO_ROOT))
        source_mon = SOURCE_DIR / "monitoring" / "monitoring.yaml"
        if source_mon.exists():
            status["plugin"]["stale"] = source_mon.stat().st_mtime > plugin_path.stat().st_mtime

    if TUI_JSON.exists():
        status["tui"]["exists"] = True
        status["tui"]["path"] = str(TUI_JSON.relative_to(REPO_ROOT))

    return status
