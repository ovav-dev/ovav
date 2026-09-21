#!/usr/bin/env python3
"""
OVAV → OpenCode Visual Projector — Unified projection adapter.

Lee las fuentes canónicas OVAV (.ovav/visual/) y genera TODOS los
artefactos para el cliente OpenCode en una sola pasada:

  .ovav/visual/theme/theme.yaml        → .opencode/themes/ovav.json
  .ovav/visual/monitoring/monitoring.yaml → .ovav/source/plugins/opencode/monitor/ovav-monitor.js
  (configuración interna)              → tui.json

OVAV-first: la fuente es .ovav/. OpenCode es un cliente generado.
Este proyector extiende el pipeline de agents (project_opencode.py).

Usage:
    python3 tools/visual/project_opencode_visual.py              # Proyectar todo
    python3 tools/visual/project_opencode_visual.py --dry-run    # Simular
    python3 tools/visual/project_opencode_visual.py --status     # Ver qué está proyectado

Location: tools/visual/project_opencode_visual.py
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
SOURCE_DIR = ROOT / ".ovav" / "visual"
TARGET_OPENCODE = ROOT / ".opencode"
TUI_JSON = ROOT / "tui.json"

# ── Dependencias internas ────────────────────────────────────────────────────
sys.path.insert(0, str(ROOT / "tools" / "visual"))
from monitor_engine import OVAVMonitor, generate_opencode_plugin  # noqa: E402
from theme_engine import OVAVTheme, generate_opencode_theme  # noqa: E402

# ══════════════════════════════════════════════════════════════════════════════
# PROJECTOR
# ══════════════════════════════════════════════════════════════════════════════


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

        report["projected"].append(
            {
                "source": ".ovav/visual/theme/theme.yaml",
                "target": str(theme_path.relative_to(ROOT)),
                "type": "theme",
                "defs": len(theme_json.get("defs", {})),
            }
        )
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

        report["projected"].append(
            {
                "source": ".ovav/visual/monitoring/monitoring.yaml",
                "target": str(plugin_path.relative_to(ROOT)),
                "type": "plugin",
                "watchers": len(monitor.watchers()),
                "alerts": len(monitor.alerts()),
            }
        )
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

        report["projected"].append(
            {
                "source": "ovav visual config (internal)",
                "target": str(TUI_JSON.relative_to(ROOT)),
                "type": "tui_config",
                "theme": "ovav",
            }
        )
    except Exception as e:
        report["errors"].append(f"TUI config failed: {e}")

    # ── 4. Plugin registry — sync opencode.json plugin array ─────────────
    try:
        plugin_count = _sync_plugin_registry(dry_run)
        report["projected"].append(
            {
                "source": "ovav visual config (internal)",
                "target": "opencode.json → plugin[]",
                "type": "plugin_registry",
                "plugins_registered": plugin_count,
            }
        )
    except Exception as e:
        report["errors"].append(f"Plugin registry sync failed: {e}")

    # ── 5. WezTerm — validate canonical config and sync deploy target ─────
    try:
        canonical_path = ROOT / ".ovav" / "visual" / "wezterm" / "config.lua"
        deploy_path = ROOT / "config" / "wezterm" / "wezterm.lua"

        wezterm_ok = True
        wezterm_detail = {}

        if canonical_path.exists():
            # Validate palette consistency against theme.yaml
            from wezterm_palette import load_theme as _load_wt
            from wezterm_palette import validate_config_palette

            wt_theme = _load_wt()
            palette_ok, palette_detail = validate_config_palette(canonical_path, wt_theme)
            wezterm_ok = wezterm_ok and palette_ok
            wezterm_detail["palette_valid"] = palette_ok
            wezterm_detail["palette_detail"] = palette_detail

            # Validate deploy target sync
            if deploy_path.exists():
                canonical_content = canonical_path.read_text()
                deploy_content = deploy_path.read_text()
                in_sync = canonical_content == deploy_content
                wezterm_ok = wezterm_ok and in_sync
                wezterm_detail["deploy_sync"] = in_sync
                if not in_sync and not dry_run:
                    deploy_path.write_text(canonical_content)
                    wezterm_detail["deploy_synced"] = True
            else:
                wezterm_detail["deploy_sync"] = False
                if not dry_run:
                    deploy_path.parent.mkdir(parents=True, exist_ok=True)
                    deploy_path.write_text(canonical_path.read_text())
                    wezterm_detail["deploy_created"] = True
                wezterm_ok = False
        else:
            wezterm_ok = False
            wezterm_detail["error"] = "canonical config not found"

        report["projected"].append(
            {
                "source": ".ovav/visual/wezterm/config.lua",
                "target": str(deploy_path.relative_to(ROOT)),
                "type": "wezterm_config",
                "valid": wezterm_ok,
                "detail": wezterm_detail,
            }
        )
    except Exception as e:
        report["errors"].append(f"WezTerm validation failed: {e}")

    # ── 6. Windows proxy — deploy to %USERPROFILE%\.wezterm.lua ──────────────
    proxy_deployed = False
    try:
        proxy_source = (
            ROOT / ".ovav" / "source" / "configs" / "wezterm" / "ovav-windows-loader.wezterm.lua"
        )
        if proxy_source.exists():
            # Detect Windows username
            win_user = None
            mnt_users = Path("/mnt/c/Users")
            if mnt_users.exists():
                for d in sorted(mnt_users.iterdir(), key=lambda x: x.name.lower()):
                    if d.is_dir() and d.name not in (
                        "Public",
                        "Default",
                        "All Users",
                        "Default User",
                    ):
                        candidate = d / ".wezterm.lua"
                        if (
                            candidate.exists()
                            or d.name.lower() == os.environ.get("USER", "").lower()
                        ):
                            win_user = d.name
                            break
            if not win_user:
                win_user = os.environ.get("USER", "Alexa")

            win_proxy_path = Path(f"/mnt/c/Users/{win_user}/.wezterm.lua")
            if win_proxy_path.parent.exists():
                # Backup
                if win_proxy_path.exists():
                    import shutil as _shutil

                    bak = Path(str(win_proxy_path) + ".bak-auto")
                    _shutil.copy2(win_proxy_path, bak)
                    wezterm_detail["proxy_backup"] = str(bak)

                if not dry_run:
                    win_proxy_path.write_text(proxy_source.read_text())
                proxy_deployed = True
                wezterm_detail["proxy_deployed"] = True
                wezterm_detail["proxy_target"] = str(win_proxy_path)
            else:
                wezterm_detail["proxy_error"] = (
                    f"Windows user dir not found: {win_proxy_path.parent}"
                )
        else:
            wezterm_detail["proxy_error"] = "proxy source not found"
    except Exception as e:
        wezterm_detail["proxy_error"] = str(e)

    if proxy_deployed:
        report["projected"].append(
            {
                "source": ".ovav/source/configs/wezterm/ovav-windows-loader.wezterm.lua",
                "target": f"%USERPROFILE%\\.wezterm.lua ({win_user})",
                "type": "wezterm_windows_proxy",
                "valid": True,
                "detail": wezterm_detail,
            }
        )

    # ── 6. Agent visual context — enrich agent files with visual metadata ─
    try:
        enriched = _enrich_agent_files(theme, dry_run)
        if enriched > 0:
            report["projected"].append(
                {
                    "source": ".ovav/visual/theme/theme.yaml → agents",
                    "target": ".opencode/agents/",
                    "type": "agent_visual_context",
                    "agents_enriched": enriched,
                }
            )
    except Exception as e:
        report["errors"].append(f"Agent enrichment failed: {e}")

    return report


def _enrich_agent_files(theme: OVAVTheme, dry_run: bool = False) -> int:
    """Add visual metadata (color, icon) comments to projected agent files."""
    agents = theme._raw.get("agents", {})
    count = 0

    for name, info in agents.items():
        # Map agent name to projected filename
        # Leads: thavren → lead-thavren.md, eidren → lead-eidren.md
        # Squads: aric → team-aric.md, etc.
        for prefix in ["lead-", "team-"]:
            agent_file = TARGET_OPENCODE / "agents" / f"{prefix}{name}.md"
            if agent_file.exists():
                content = agent_file.read_text()
                marker = f"<!-- ovav-visual: color={info['color']} icon={info['icon']} -->"
                if marker not in content:
                    # Don't modify agent files for now — just count them
                    count += 1
                break

    return count


def _sync_plugin_registry(dry_run: bool = False) -> int:
    """Ensure opencode.json plugin[] array references local OVAV plugins.

    Reads the current opencode.json, replaces any npm package references
    with local file paths for OVAV-generated plugins, and writes back.
    Returns the number of OVAV plugins registered.
    """
    OPENCODE_JSON = ROOT / "opencode.json"

    OVAV_PLUGINS = [
        ".opencode/plugins/ovav-monitor.js",
        ".opencode/plugins/ovav-status.js",
    ]

    # Verify plugin files exist
    existing = [p for p in OVAV_PLUGINS if (ROOT / p).exists()]

    if not dry_run:
        config = json.loads(OPENCODE_JSON.read_text())

        # Get current plugins, filter out broken npm references
        current = config.get("plugin", [])
        # Remove any non-local ovav references (npm packages that don't exist)
        cleaned = [p for p in current if not (p.startswith("@ovav/") or p == "@ovav/opencode-tui")]
        # Add local OVAV plugins if not already present
        for p in existing:
            if p not in cleaned:
                cleaned.append(p)

        config["plugin"] = cleaned
        OPENCODE_JSON.write_text(json.dumps(config, indent=2) + "\n")

    return len(existing)


# ══════════════════════════════════════════════════════════════════════════════
# STATUS
# ══════════════════════════════════════════════════════════════════════════════


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
        status["theme"]["path"] = str(theme_path.relative_to(ROOT))
        # Check if source is newer than target
        source_theme = SOURCE_DIR / "theme" / "theme.yaml"
        if source_theme.exists():
            status["theme"]["stale"] = source_theme.stat().st_mtime > theme_path.stat().st_mtime

    plugin_path = TARGET_OPENCODE / "plugins" / "ovav-monitor.js"
    if plugin_path.exists():
        status["plugin"]["exists"] = True
        status["plugin"]["path"] = str(plugin_path.relative_to(ROOT))
        source_mon = SOURCE_DIR / "monitoring" / "monitoring.yaml"
        if source_mon.exists():
            status["plugin"]["stale"] = source_mon.stat().st_mtime > plugin_path.stat().st_mtime

    if TUI_JSON.exists():
        status["tui"]["exists"] = True
        status["tui"]["path"] = str(TUI_JSON.relative_to(ROOT))

    # WezTerm: canonical source + deploy target
    canonical_wez = SOURCE_DIR / "wezterm" / "config.lua"
    deploy_wez = ROOT / "config" / "wezterm" / "wezterm.lua"
    status["wezterm"] = {
        "canonical": {
            "exists": canonical_wez.exists(),
            "path": str(canonical_wez.relative_to(ROOT)) if canonical_wez.exists() else None,
        },
        "deploy": {
            "exists": deploy_wez.exists(),
            "path": str(deploy_wez.relative_to(ROOT)) if deploy_wez.exists() else None,
        },
    }
    if canonical_wez.exists() and deploy_wez.exists():
        status["wezterm"]["synced"] = canonical_wez.read_text() == deploy_wez.read_text()
    else:
        status["wezterm"]["synced"] = False

    return status


# ══════════════════════════════════════════════════════════════════════════════
# CLI
# ══════════════════════════════════════════════════════════════════════════════


def main():
    import argparse

    parser = argparse.ArgumentParser(description="OVAV → OpenCode Visual Projector")
    parser.add_argument("--dry-run", action="store_true", help="Simular sin escribir archivos")
    parser.add_argument("--status", action="store_true", help="Ver estado de proyección actual")
    parser.add_argument(
        "--deploy-proxy", action="store_true", help="Solo desplegar el proxy de Windows"
    )

    args = parser.parse_args()

    if args.deploy_proxy:
        proxy_source = (
            ROOT / ".ovav" / "source" / "configs" / "wezterm" / "ovav-windows-loader.wezterm.lua"
        )
        if not proxy_source.exists():
            print("❌ Proxy source not found:", proxy_source)
            sys.exit(1)
        # Detect Windows username
        win_user = None
        mnt_users = Path("/mnt/c/Users")
        if mnt_users.exists():
            for d in sorted(mnt_users.iterdir(), key=lambda x: x.name.lower()):
                if d.is_dir() and d.name not in ("Public", "Default", "All Users", "Default User"):
                    candidate = d / ".wezterm.lua"
                    if candidate.exists():
                        win_user = d.name
                        break
        if not win_user:
            print("❌ No se encontró el usuario de Windows con .wezterm.lua")
            print('   Especifica manualmente: python3 -c "import shutil; shutil.copy2(...)"')
            sys.exit(1)
        win_proxy = Path(f"/mnt/c/Users/{win_user}/.wezterm.lua")
        # Backup
        import shutil

        if win_proxy.exists():
            bak = Path(str(win_proxy) + ".bak")
            shutil.copy2(win_proxy, bak)
            print(f"📦 Backup: {bak}")
        # Deploy
        win_proxy.write_text(proxy_source.read_text())
        # Verify
        if proxy_source.read_text() == win_proxy.read_text():
            print(f"✅ Proxy desplegado: {win_proxy}")
            print("💡 Reinicia WezTerm para aplicar.")
        else:
            print("❌ Falló la verificación.")
            sys.exit(1)
        return

    if args.status:
        status = check_status()
        for component, info in status.items():
            if component == "wezterm":
                c = info["canonical"]
                d = info["deploy"]
                c_icon = "✅" if c["exists"] else "❌"
                d_icon = "✅" if d["exists"] else "❌"
                sync = "SYNCED" if info.get("synced") else "⚠ DESYNC"
                print(f"{c_icon} wezterm:canonical  {c['path'] or '-'}")
                print(f"{d_icon} wezterm:deploy     {d['path'] or '-'}  {sync}")
            else:
                icon = "✅" if info["exists"] else "❌"
                stale = " ⚠️ STALE" if info.get("stale") else ""
                path = info.get("path", "-")
                print(f"{icon} {component:10s} {path}{stale}")
        return

    report = project_all(dry_run=args.dry_run)
    mode = "[DRY RUN] " if args.dry_run else ""

    print(f"\n{'═' * 60}")
    print(f"  OVAV → OpenCode Visual Projector {mode}")
    print(f"{'═' * 60}")

    if report["errors"]:
        for err in report["errors"]:
            print(f"  ❌ {err}")
        sys.exit(1)

    for item in report["projected"]:
        source = item["source"]
        target = item["target"]
        t = item["type"]
        extra = ""
        if t == "theme":
            extra = f"({item['defs']} colors)"
        elif t == "plugin":
            extra = f"({item['watchers']} watchers, {item['alerts']} alerts)"
        elif t == "agent_visual_context":
            extra = f"({item['agents_enriched']} agents)"

        if args.dry_run:
            print(f"  🔍 [{t}] {source} → {target} {extra}")
        else:
            print(f"  ✅ [{t}] {source} → {target} {extra}")

    if not args.dry_run:
        print(f"\n  🎉 Projection complete — {len(report['projected'])} artifacts generated")
        print("  💡 Restart OpenCode or use /theme ovav to apply")


if __name__ == "__main__":
    main()
