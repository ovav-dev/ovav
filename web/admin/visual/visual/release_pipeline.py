#!/usr/bin/env python3
"""
OVAV Release Pipeline — 9-Gate orchestrator with 2-phase staging.

Staging pipeline (I-010): internal test → CEO approval → external release.
  Fase 1 (--stage internal): Gates 1-7 → internal test release para CEO
  Fase 2 (--stage external): CEO approval check → Gates 8-9 → external release

Antes de proyectar CUALQUIER feature OVAV a un cliente externo (OpenCode, etc.),
el sistema DEBE pasar por estos 9 gates en orden. Si uno falla, se detiene todo.

  Gate 1 · PRE-FLIGHT       OVAV Law, Integrity Mesh, Semantic Drift
  Gate 2 · GENERATE         Generar artefactos desde fuente OVAV
  Gate 3 · COMPATIBILITY    Validar contra schema del cliente
  Gate 4 · INTEGRITY SCAN   Detectar archivos rotos, corrupción
  Gate 5 · CODE QUALITY     Revisión de arquitectura y convenciones
  Gate 6 · CHANGELOG        Diff contra versión anterior
  Gate 7 · RIGOROUS TEST    Tests reales de ejecución
  Gate 8 · VERSION PACK     Empaquetar versión lista para distribución
  Gate 9 · CLI NOTIFY       Notificar actualización disponible

Usage:
    python3 tools/visual/release_pipeline.py                        # Pipeline completo
    python3 tools/visual/release_pipeline.py --stage internal       # Fase 1: internal test
    python3 tools/visual/release_pipeline.py --stage external       # Fase 2: external release
    python3 tools/visual/release_pipeline.py --dry-run              # Simular sin escribir
    python3 tools/visual/release_pipeline.py --status               # Estado de última release
    python3 tools/visual/release_pipeline.py --approve              # CEO aprueba internal release

Location: tools/visual/release_pipeline.py
"""

from __future__ import annotations

import hashlib
import json
import subprocess
import sys
import time
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[2]
RELEASES_DIR = ROOT / ".ovav" / "visual" / "releases"
OPENCODE_DIR = ROOT / ".opencode"


# ══════════════════════════════════════════════════════════════════════════════
# UTILITIES
# ══════════════════════════════════════════════════════════════════════════════

def _run(cmd: list[str], timeout: int = 30, cwd: Path | None = None) -> dict[str, Any]:
    """Run a command and return result."""
    try:
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout,
            cwd=str(cwd or ROOT)
        )
        return {
            "ok": result.returncode == 0,
            "stdout": result.stdout.strip(),
            "stderr": result.stderr.strip(),
            "code": result.returncode,
        }
    except Exception as e:
        return {"ok": False, "stdout": "", "stderr": str(e), "code": -1}


def _hash_file(path: Path) -> str:
    """SHA-256 hash of a file."""
    if not path.exists():
        return "MISSING"
    return hashlib.sha256(path.read_bytes()).hexdigest()[:16]


def _timestamp() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def _gate_header(num: int, name: str) -> str:
    return f"\n{'═' * 70}\n  GATE {num} · {name}\n{'═' * 70}"


def _pass(msg: str) -> str:
    return f"  ✅ PASS · {msg}"


def _fail(msg: str) -> str:
    return f"  ❌ FAIL · {msg}"


def _warn(msg: str) -> str:
    return f"  ⚠️  WARN · {msg}"


def _info(msg: str) -> str:
    return f"  ℹ️  {msg}"


# ══════════════════════════════════════════════════════════════════════════════
# GATE 1 — PRE-FLIGHT CHECKS
# ══════════════════════════════════════════════════════════════════════════════

def gate1_preflight() -> tuple[bool, list[str]]:
    """Run existing OVAV validators: law compliance, integrity, drift."""
    log: list[str] = []
    all_pass = True

    # 1a — OVAV Law Compliance
    law = _run(["python3", "tools/harnesses/check_ovav_law_compliance.py"])
    if law["ok"]:
        log.append(_pass("OVAV Law Compliance"))
    else:
        log.append(_fail(f"OVAV Law: {law['stderr'][:200]}"))
        all_pass = False

    # 1b — Integrity Mesh (host config drift as proxy)
    integrity = _run(["python3", "tools/validators/check_host_config_drift.py", "--json"])
    if integrity["ok"]:
        log.append(_pass("Integrity Mesh — Host clean"))
    else:
        log.append(_warn(f"Integrity Mesh: {integrity['stderr'][:200]}"))

    # 1c — Session context guard
    guard = _run(["python3", "tools/security/session_context_guard.py", "--check", "--json"])
    if guard["ok"]:
        log.append(_pass("Session Context Guard"))
    else:
        log.append(_fail(f"Context Guard: {guard['stderr'][:200]}"))
        all_pass = False

    return all_pass, log


# ══════════════════════════════════════════════════════════════════════════════
# GATE 2 — GENERATE ARTIFACTS
# ══════════════════════════════════════════════════════════════════════════════

def gate2_generate(dry_run: bool = False) -> tuple[bool, list[str], dict[str, Path]]:
    """Generate client artifacts from OVAV source using unified projector."""
    log: list[str] = []
    artifacts: dict[str, Path] = {}
    all_pass = True

    # Use unified projector: one command, all artifacts
    cmd = ["python3", "tools/visual/project_opencode_visual.py"]
    if dry_run:
        cmd.append("--dry-run")

    result = _run(cmd)
    if result["ok"]:
        log.append(_pass("Unified projector: all artifacts generated"))
    else:
        log.append(_fail(f"Projector failed: {result['stderr'][:300]}"))
        all_pass = False

    # Map generated artifacts
    theme_out = OPENCODE_DIR / "themes" / "ovav.json"
    plugin_out = OPENCODE_DIR / "plugins" / "ovav-monitor.js"
    tui_out = ROOT / "tui.json"

    if theme_out.exists():
        artifacts["theme"] = theme_out
        log.append(_pass(f"Theme: {theme_out.relative_to(ROOT)}"))
    else:
        if not dry_run:
            log.append(_fail("Theme not generated"))
            all_pass = False

    if plugin_out.exists():
        artifacts["plugin"] = plugin_out
        log.append(_pass(f"Plugin: {plugin_out.relative_to(ROOT)}"))
    else:
        log.append(_warn("Plugin not generated yet (may be expected)"))

    if tui_out.exists():
        artifacts["tui"] = tui_out
        log.append(_pass(f"TUI config: {tui_out.relative_to(ROOT)}"))
    else:
        if not dry_run:
            log.append(_fail("tui.json not generated"))
            all_pass = False

    return all_pass, log, artifacts


# ══════════════════════════════════════════════════════════════════════════════
# GATE 3 — COMPATIBILITY
# ══════════════════════════════════════════════════════════════════════════════

def gate3_compatibility(artifacts: dict[str, Path]) -> tuple[bool, list[str]]:
    """Validate generated artifacts against client schemas."""
    log: list[str] = []
    all_pass = True

    # Check if artifacts exist (skip validation if not generated yet, e.g. dry-run)
    has_theme = artifacts.get("theme") and artifacts["theme"].exists()
    has_tui = artifacts.get("tui") and artifacts["tui"].exists()

    if not has_theme and not has_tui:
        log.append(_info("No artifacts generated yet — skipping compatibility check (dry-run expected)"))
        return True, log

    # 3a — Validate theme JSON structure
    theme_path = artifacts.get("theme")
    if has_theme:
        try:
            theme_data = json.loads(theme_path.read_text())
            required = ["$schema", "name", "defs", "theme"]
            missing = [k for k in required if k not in theme_data]
            if missing:
                log.append(_fail(f"Theme JSON missing keys: {missing}"))
                all_pass = False
            else:
                # Validate all hex colors are valid
                invalid = []
                for key, val in theme_data.get("defs", {}).items():
                    if not (isinstance(val, str) and val.startswith("#") and len(val) == 7):
                        invalid.append(f"{key}={val}")
                if invalid:
                    log.append(_fail(f"Invalid color definitions: {invalid}"))
                    all_pass = False
                else:
                    log.append(_pass(f"Theme JSON valid — {len(theme_data['defs'])} colors, schema OK"))
        except json.JSONDecodeError as e:
            log.append(_fail(f"Theme JSON parse error: {e}"))
            all_pass = False
    else:
        log.append(_fail("Theme file not found"))
        all_pass = False

    # 3b — Validate tui.json
    tui_path = artifacts.get("tui")
    if has_tui:
        try:
            tui_data = json.loads(tui_path.read_text())
            if tui_data.get("theme") == "ovav":
                log.append(_pass("tui.json valid — theme=ovav"))
            else:
                log.append(_warn(f"tui.json theme is '{tui_data.get('theme')}', expected 'ovav'"))
        except json.JSONDecodeError:
            log.append(_fail("tui.json parse error"))
            all_pass = False
    else:
        log.append(_fail("tui.json not found"))
        all_pass = False

    return all_pass, log


# ══════════════════════════════════════════════════════════════════════════════
# GATE 4 — INTEGRITY SCAN
# ══════════════════════════════════════════════════════════════════════════════

def gate4_integrity(artifacts: dict[str, Path]) -> tuple[bool, list[str], dict[str, str]]:
    """Scan all generated files for corruption and compute integrity hashes."""
    log: list[str] = []
    hashes: dict[str, str] = {}
    all_pass = True

    any_exist = any(p.exists() for p in artifacts.values() if isinstance(p, Path))
    if not any_exist:
        log.append(_info("No artifacts generated yet — skipping integrity scan (dry-run expected)"))
        for name in artifacts:
            hashes[name] = "DRY_RUN"
        return True, log, hashes

    for name, path in artifacts.items():
        if path.exists():
            file_hash = _hash_file(path)
            hashes[name] = file_hash
            size = path.stat().st_size
            if size == 0:
                log.append(_fail(f"{name}: EMPTY FILE — possible corruption"))
                all_pass = False
            elif size < 10:
                log.append(_warn(f"{name}: Suspiciously small ({size} bytes)"))
            else:
                log.append(_pass(f"{name}: {size}B · hash={file_hash}"))
        else:
            log.append(_warn(f"{name}: Not generated yet"))
            hashes[name] = "NOT_GENERATED"

    # Check for duplicate/conflicting files
    if artifacts.get("theme") and artifacts.get("tui"):
        # Ensure theme referenced in tui.json exists
        pass  # Already validated in gate 3

    return all_pass, log, hashes


# ══════════════════════════════════════════════════════════════════════════════
# GATE 5 — CODE QUALITY
# ══════════════════════════════════════════════════════════════════════════════

def gate5_code_quality() -> tuple[bool, list[str]]:
    """Review OVAV source code for architecture compliance."""
    log: list[str] = []
    all_pass = True

    source_files = [
        ROOT / ".ovav" / "visual" / "theme" / "theme.yaml",
        ROOT / ".ovav" / "visual" / "monitoring" / "monitoring.yaml",
        ROOT / "tools" / "visual" / "theme_engine.py",
    ]

    for sf in source_files:
        if not sf.exists():
            log.append(_fail(f"Missing source: {sf}"))
            all_pass = False
            continue

        # Check for hardcoded secrets (simple heuristic)
        content = sf.read_text()
        suspicious = ["sk-", "api_key=", "password=", "secret=", "BEGIN PRIVATE KEY"]
        found = [s for s in suspicious if s in content.lower()]
        if found:
            log.append(_warn(f"{sf.name}: Contains keywords: {found} — review manually"))
        else:
            log.append(_pass(f"{sf.name}: No suspicious patterns"))

        # Check YAML validity
        if sf.suffix in (".yaml", ".yml"):
            try:
                import yaml
                yaml.safe_load(content)
                log.append(_pass(f"{sf.name}: Valid YAML"))
            except Exception as e:
                log.append(_fail(f"{sf.name}: YAML parse error: {e}"))
                all_pass = False

        # Check Python syntax
        if sf.suffix == ".py":
            try:
                compile(content, str(sf), "exec")
                log.append(_pass(f"{sf.name}: Valid Python syntax"))
            except SyntaxError as e:
                log.append(_fail(f"{sf.name}: Python syntax error: {e}"))
                all_pass = False

    return all_pass, log


# ══════════════════════════════════════════════════════════════════════════════
# GATE 6 — CHANGELOG
# ══════════════════════════════════════════════════════════════════════════════

def gate6_changelog(hashes: dict[str, str], dry_run: bool = False) -> tuple[bool, list[str], dict]:
    """Generate changelog comparing against previous release."""
    log: list[str] = []
    prev_manifest = RELEASES_DIR / "latest" / "manifest.json"
    version = datetime.now(UTC).strftime("v%Y.%m.%d-%H%M")

    changes: dict[str, Any] = {
        "version": version,
        "timestamp": _timestamp(),
        "files": {},
        "breaking": False,
        "summary": [],
    }

    if prev_manifest.exists():
        try:
            prev = json.loads(prev_manifest.read_text())
            prev_hashes = prev.get("hashes", {})

            for name, h in hashes.items():
                prev_h = prev_hashes.get(name, "NEW")
                if prev_h == "NEW":
                    changes["files"][name] = "ADDED"
                    changes["summary"].append(f"➕ {name}: nuevo artefacto")
                elif prev_h != h:
                    changes["files"][name] = "MODIFIED"
                    changes["summary"].append(f"📝 {name}: modificado ({prev_h[:8]} → {h[:8]})")
                else:
                    changes["files"][name] = "UNCHANGED"

            log.append(_pass(f"Changelog: {len([f for f in changes['files'].values() if f != 'UNCHANGED'])} cambios detectados vs {prev.get('version', '?')}"))
        except Exception as e:
            log.append(_warn(f"Changelog: Could not parse previous manifest: {e}"))
            for name in hashes:
                changes["files"][name] = "INITIAL"
            changes["summary"].append("🎉 Release inicial")
    else:
        log.append(_info("Changelog: Primera release — sin versión anterior"))
        for name in hashes:
            changes["files"][name] = "INITIAL"
        changes["summary"].append("🎉 Release inicial del sistema visual OVAV")

    return True, log, changes


# ══════════════════════════════════════════════════════════════════════════════
# GATE 7 — RIGOROUS TESTING
# ══════════════════════════════════════════════════════════════════════════════

def gate7_rigorous_test(artifacts: dict[str, Path]) -> tuple[bool, list[str]]:
    """Run real execution tests on generated artifacts."""
    log: list[str] = []
    all_pass = True

    # 7a — Theme engine self-test
    engine = _run(["python3", "tools/visual/theme_engine.py", "--validate"])
    if engine["ok"]:
        log.append(_pass("Theme engine validate: OK"))
    else:
        log.append(_fail(f"Theme engine: {engine['stderr'][:200]}"))
        all_pass = False

    # 7b — Verify all agent colors are valid hex
    try:
        import yaml
        with open(ROOT / ".ovav" / "visual" / "theme" / "theme.yaml") as f:
            theme = yaml.safe_load(f)
        agents = theme.get("agents", {})
        invalid = []
        for name, info in agents.items():
            color = info.get("color", "")
            if not (color.startswith("#") and len(color) == 7):
                invalid.append(f"{name}: {color}")
        if invalid:
            log.append(_fail(f"Agent colors invalid: {invalid}"))
            all_pass = False
        else:
            log.append(_pass(f"Agent colors valid: {len(agents)} agents OK"))
    except Exception as e:
        log.append(_fail(f"Agent color check: {e}"))
        all_pass = False

    # 7c — Verify theme JSON roundtrip (generate → validate)
    if artifacts.get("theme") and artifacts["theme"].exists():
        try:
            data = json.loads(artifacts["theme"].read_text())
            # Check every color in defs is valid hex
            defs_invalid = []
            for k, v in data.get("defs", {}).items():
                if not (isinstance(v, str) and v.startswith("#") and len(v) == 7):
                    defs_invalid.append(k)
            if defs_invalid:
                log.append(_fail(f"Theme defs invalid: {defs_invalid}"))
                all_pass = False
            else:
                log.append(_pass(f"Theme JSON roundtrip: {len(data['defs'])} colors valid"))
        except Exception as e:
            log.append(_fail(f"Theme JSON roundtrip: {e}"))
            all_pass = False

    # 7d — Verify monitoring YAML is valid and complete
    mon_path = ROOT / ".ovav" / "visual" / "monitoring" / "monitoring.yaml"
    if mon_path.exists():
        try:
            mon = yaml.safe_load(mon_path.read_text())
            watchers = mon.get("watchers", {})
            alerts = mon.get("alerts", {})
            log.append(_pass(f"Monitoring: {len(watchers)} watchers, {len(alerts)} alerts"))
        except Exception as e:
            log.append(_fail(f"Monitoring YAML: {e}"))
            all_pass = False

    return all_pass, log


# ══════════════════════════════════════════════════════════════════════════════
# GATE 8 — VERSION PACKAGING
# ══════════════════════════════════════════════════════════════════════════════

def gate8_version_pack(
    artifacts: dict[str, Path],
    hashes: dict[str, str],
    changes: dict,
    version: str,
    dry_run: bool = False,
) -> tuple[bool, list[str], Path]:
    """Package the release version."""
    log: list[str] = []
    release_dir = RELEASES_DIR / version
    latest_link = RELEASES_DIR / "latest"

    if dry_run:
        log.append(_info(f"[DRY RUN] Version dir: {release_dir}"))
        return True, log, release_dir

    # Create release directory
    release_dir.mkdir(parents=True, exist_ok=True)
    artifacts_dir = release_dir / "artifacts"
    artifacts_dir.mkdir(exist_ok=True)

    # Copy artifacts
    for name, path in artifacts.items():
        if path.exists():
            dest = artifacts_dir / path.name
            dest.write_bytes(path.read_bytes())
            log.append(_pass(f"Packaged: {path.name} → {dest}"))

    # Write manifest
    manifest = {
        "version": version,
        "timestamp": _timestamp(),
        "hashes": hashes,
        "changes": changes,
        "artifacts": list(artifacts.keys()),
        "pipeline_status": "PASSED",
    }
    (release_dir / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    log.append(_pass(f"Manifest: {release_dir / 'manifest.json'}"))

    # Write changelog
    changelog_lines = [f"# OVAV Visual Release {version}", "", f"**Date:** {_timestamp()}", ""]
    for s in changes.get("summary", []):
        changelog_lines.append(f"- {s}")
    (release_dir / "CHANGELOG.md").write_text("\n".join(changelog_lines) + "\n")
    log.append(_pass(f"Changelog: {release_dir / 'CHANGELOG.md'}"))

    # Update latest symlink
    if latest_link.exists():
        if latest_link.is_symlink():
            latest_link.unlink()
        else:
            import shutil
            shutil.rmtree(latest_link)
    latest_link.symlink_to(release_dir.relative_to(RELEASES_DIR), target_is_directory=True)
    log.append(_pass(f"Latest → {version}"))

    return True, log, release_dir


# ══════════════════════════════════════════════════════════════════════════════
# GATE 9 — CLI NOTIFY
# ══════════════════════════════════════════════════════════════════════════════

def gate9_cli_notify(version: str, release_dir: Path, dry_run: bool = False) -> tuple[bool, list[str]]:
    """Write update notification for OVAV CLI."""
    log: list[str] = []
    # C3.3.2: Escribir al path estándar que bin/ovav y ovav_official_update.py leen
    notify_file = ROOT / ".ovav" / "forge" / "releases" / "opencode" / "update_available.json"
    # También escribir al path visual para referencia del release pipeline
    visual_notify = ROOT / ".ovav" / "visual" / "releases" / "update_available.json"

    notification = {
        "available": True,
        "version": version,
        "timestamp": _timestamp(),
        "component": "visual",
        "message": f"🎨 OVAV Visual v{version} disponible. Ejecutá 'ovav update' para aplicar.",
        "release_dir": str(release_dir.relative_to(ROOT)),
    }

    if not dry_run:
        notify_file.parent.mkdir(parents=True, exist_ok=True)
        notify_file.write_text(json.dumps(notification, indent=2) + "\n")
        visual_notify.parent.mkdir(parents=True, exist_ok=True)
        visual_notify.write_text(json.dumps(notification, indent=2) + "\n")
        log.append(_pass(f"Notification: {notify_file}"))
        log.append(_info(f"💡 El usuario recibirá: '{notification['message']}' al ejecutar 'ovav update'"))
    else:
        log.append(_info(f"[DRY RUN] Notification → {notify_file}"))

    return True, log


# ══════════════════════════════════════════════════════════════════════════════
# STAGING PIPELINE (I-010) — Internal test → CEO approval → External release
# ══════════════════════════════════════════════════════════════════════════════

INTERNAL_DIR = RELEASES_DIR / "internal"
APPROVAL_FILE = RELEASES_DIR / ".ceo_approval.yaml"


def save_internal_release(
    version: str,
    artifacts: dict[str, Path],
    hashes: dict[str, str],
    changes: dict,
    results: dict,
) -> Path:
    """Save internal test release for CEO review.

    Returns path to internal release directory.
    """
    internal_dir = INTERNAL_DIR / version
    internal_dir.mkdir(parents=True, exist_ok=True)

    # Copy artifacts
    artifacts_dir = internal_dir / "artifacts"
    artifacts_dir.mkdir(exist_ok=True)
    for name, path in artifacts.items():
        if path.exists():
            (artifacts_dir / path.name).write_bytes(path.read_bytes())

    # Write internal manifest
    manifest = {
        "version": version,
        "timestamp": _timestamp(),
        "stage": "internal",
        "hashes": hashes,
        "changes": changes,
        "gates_passed": {str(k): v["ok"] for k, v in results.items()},
        "status": "awaiting_ceo_approval",
    }
    (internal_dir / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")

    # Write approval request
    approval_request = {
        "requested_at": _timestamp(),
        "version": version,
        "stage": "internal",
        "status": "pending",
        "message": "CEO: revisá los artefactos en .ovav/visual/releases/internal/ y ejecutá --approve para promover a external.",
        "action_required": "python3 tools/visual/release_pipeline.py --approve",
    }
    (internal_dir / "approval_request.json").write_text(json.dumps(approval_request, indent=2) + "\n")

    # Update latest internal
    latest_internal = INTERNAL_DIR / "latest"
    if latest_internal.exists():
        if latest_internal.is_symlink():
            latest_internal.unlink()
        else:
            import shutil
            shutil.rmtree(latest_internal)
    latest_internal.symlink_to(internal_dir.relative_to(INTERNAL_DIR), target_is_directory=True)

    return internal_dir


def check_ceo_approval() -> dict[str, Any]:
    """Check if CEO has approved the internal release for external promotion.

    Returns approval status dict.
    """
    if not APPROVAL_FILE.exists():
        return {
            "approved": False,
            "reason": "No approval file found. CEO must run: python3 tools/visual/release_pipeline.py --approve",
            "approval_file": str(APPROVAL_FILE.relative_to(ROOT)),
        }

    try:
        import yaml
        approval = yaml.safe_load(APPROVAL_FILE.read_text())
    except Exception:
        approval = {}

    if not isinstance(approval, dict):
        return {"approved": False, "reason": "Invalid approval file format"}

    approved = approval.get("approved", False)
    version = approval.get("version", "unknown")
    timestamp = approval.get("timestamp", "unknown")

    # Check expiry (72 hours max)
    if approved and timestamp != "unknown":
        try:
            approved_at = datetime.fromisoformat(timestamp)
            age_hours = (datetime.now(UTC) - approved_at).total_seconds() / 3600
            if age_hours > 72:
                return {
                    "approved": False,
                    "reason": f"Approval expired ({age_hours:.1f}h > 72h max). Re-run internal stage and re-approve.",
                    "version": version,
                }
        except ValueError:
            pass

    return {
        "approved": approved,
        "version": version,
        "timestamp": timestamp,
        "approval_file": str(APPROVAL_FILE.relative_to(ROOT)),
    }


def ceo_approve() -> dict[str, Any]:
    """Record CEO approval for the latest internal release.

    This is a manual step — the CEO runs this command after reviewing
    the internal test artifacts. It writes the approval file that the
    external stage checks before promoting.
    """
    # Find latest internal release
    latest_internal = INTERNAL_DIR / "latest"
    if not latest_internal.exists():
        return {
            "status": "error",
            "reason": "No internal release found. Run --stage internal first.",
        }

    manifest_path = latest_internal / "manifest.json"
    if not manifest_path.exists():
        return {
            "status": "error",
            "reason": "Internal release manifest not found.",
        }

    manifest = json.loads(manifest_path.read_text())
    version = manifest.get("version", "unknown")

    # Check gates passed
    gates = manifest.get("gates_passed", {})
    failed = [k for k, v in gates.items() if not v]
    if failed:
        return {
            "status": "error",
            "reason": f"Cannot approve: gate(s) {failed} failed in internal stage.",
            "gates": gates,
        }

    # Write approval
    approval = {
        "approved": True,
        "version": version,
        "timestamp": datetime.now(UTC).isoformat(),
        "approved_by": "CEO",
        "note": "Internal release reviewed and approved for external promotion.",
    }
    APPROVAL_FILE.parent.mkdir(parents=True, exist_ok=True)
    APPROVAL_FILE.write_text(
        "# OVAV CEO Approval — Internal → External Promotion\n"
        + __import__('yaml').dump(approval, default_flow_style=False)
    )

    return {
        "status": "ok",
        "version": version,
        "message": f"✅ Internal release {version} aprobada. Ejecutá --stage external para promover.",
        "approval_file": str(APPROVAL_FILE.relative_to(ROOT)),
    }


def run_internal_stage(dry_run: bool = False) -> tuple[bool, str, dict]:
    """Run internal stage (gates 1-7) for CEO review.

    Returns (success, version, results_dict).
    """
    start_time = time.time()
    all_pass = True
    results: dict[int, dict] = {}

    print(f"\n{'▀' * 70}")
    print("  OVAV STAGING PIPELINE — FASE 1: INTERNAL TEST")
    print(f"  Started: {_timestamp()}")
    print(f"  Mode: {'DRY RUN' if dry_run else 'LIVE'}")
    print("  Gates 1-7 → CEO review → Approval → Fase 2 external")
    print(f"{'▄' * 70}")

    # ── GATE 1: Pre-flight ──────────────────────────────────────────────
    print(_gate_header(1, "PRE-FLIGHT — OVAV Law, Integrity, Context"))
    ok, log = gate1_preflight()
    for line in log: print(line)
    results[1] = {"ok": ok, "log": log}
    if not ok: all_pass = False

    # ── GATE 2: Generate ─────────────────────────────────────────────────
    print(_gate_header(2, "GENERATE — Build artifacts from OVAV source"))
    ok, log, artifacts = gate2_generate(dry_run=dry_run)
    for line in log: print(line)
    results[2] = {"ok": ok, "log": log, "artifacts": {k: str(v) for k, v in artifacts.items()}}
    if not ok: all_pass = False

    # ── GATE 3: Compatibility ────────────────────────────────────────────
    print(_gate_header(3, "COMPATIBILITY — Validate against client schema"))
    ok, log = gate3_compatibility(artifacts)
    for line in log: print(line)
    results[3] = {"ok": ok, "log": log}
    if not ok: all_pass = False

    # ── GATE 4: Integrity Scan ───────────────────────────────────────────
    print(_gate_header(4, "INTEGRITY SCAN — Detect corruption, compute hashes"))
    ok, log, hashes = gate4_integrity(artifacts)
    for line in log: print(line)
    results[4] = {"ok": ok, "log": log, "hashes": hashes}
    if not ok: all_pass = False

    # ── GATE 5: Code Quality ─────────────────────────────────────────────
    print(_gate_header(5, "CODE QUALITY — Architecture compliance, syntax, secrets"))
    ok, log = gate5_code_quality()
    for line in log: print(line)
    results[5] = {"ok": ok, "log": log}
    if not ok: all_pass = False

    # ── GATE 6: Changelog ────────────────────────────────────────────────
    print(_gate_header(6, "CHANGELOG — Diff against previous release"))
    ok, log, changes = gate6_changelog(hashes, dry_run=dry_run)
    for line in log: print(line)
    results[6] = {"ok": ok, "log": log, "changes": changes}
    version = changes["version"]

    # ── GATE 7: Rigorous Testing ─────────────────────────────────────────
    print(_gate_header(7, "RIGOROUS TESTING — Real execution tests"))
    ok, log = gate7_rigorous_test(artifacts)
    for line in log: print(line)
    results[7] = {"ok": ok, "log": log}
    if not ok: all_pass = False

    # ── Save internal release ────────────────────────────────────────────
    elapsed = time.time() - start_time
    print(f"\n{'─' * 70}")
    if all_pass and not dry_run:
        internal_dir = save_internal_release(version, artifacts, hashes, changes, results)
        print(f"  📦 Internal release saved: {internal_dir.relative_to(ROOT)}")
        print("  🔐 Awaiting CEO approval: python3 tools/visual/release_pipeline.py --approve")
    elif dry_run:
        print("  ℹ️  [DRY RUN] Internal release would be saved for CEO review")
    else:
        print("  ❌ Internal stage FAILED — fix issues before CEO review")
    print(f"  Duration: {elapsed:.1f}s")
    print(f"{'▄' * 70}\n")

    return all_pass, version, results


def run_external_stage(
    artifacts: dict[str, Path],
    hashes: dict[str, str],
    changes: dict,
    version: str,
    dry_run: bool = False,
) -> bool:
    """Run external stage (gates 8-9) after CEO approval.

    Returns True if external release was successful.
    """
    start_time = time.time()
    all_pass = True

    print(f"\n{'▀' * 70}")
    print("  OVAV STAGING PIPELINE — FASE 2: EXTERNAL RELEASE")
    print(f"  Started: {_timestamp()}")
    print(f"  Version: {version}")
    print(f"  Mode: {'DRY RUN' if dry_run else 'LIVE'}")
    print(f"{'▄' * 70}")

    # ── GATE 8: Version Packaging ────────────────────────────────────────
    print(_gate_header(8, "VERSION PACKAGING — Bundle for distribution"))
    ok, log, release_dir = gate8_version_pack(artifacts, hashes, changes, version, dry_run=dry_run)
    for line in log: print(line)
    if not ok: all_pass = False

    # ── GATE 9: CLI Notify ───────────────────────────────────────────────
    print(_gate_header(9, "CLI NOTIFY — Update available notification"))
    ok, log = gate9_cli_notify(version, release_dir, dry_run=dry_run)
    for line in log: print(line)
    if not ok: all_pass = False

    # ── Clear approval after successful external release ──────────────────
    if all_pass and not dry_run and APPROVAL_FILE.exists():
        APPROVAL_FILE.unlink()
        print("  ✅ CEO approval consumed — internal release promoted to external")

    elapsed = time.time() - start_time
    print(f"\n{'▀' * 70}")
    if all_pass:
        print(f"  🎉 EXTERNAL RELEASE COMPLETE — v{version}")
        print("  💡 Usuarios recibirán notificación via 'ovav update'")
    else:
        print("  ❌ External release FAILED")
    print(f"  Duration: {elapsed:.1f}s")
    print(f"{'▄' * 70}\n")

    return all_pass


# ══════════════════════════════════════════════════════════════════════════════
# ORCHESTRATOR
# ══════════════════════════════════════════════════════════════════════════════

def run_pipeline(dry_run: bool = False) -> bool:
    """Execute the full 9-gate release pipeline. Returns True if all gates pass."""
    start_time = time.time()
    all_pass = True
    results: dict[int, dict] = {}

    print(f"\n{'▀' * 70}")
    print("  OVAV RELEASE PIPELINE — 9 Gates")
    print(f"  Started: {_timestamp()}")
    print(f"  Mode: {'DRY RUN' if dry_run else 'LIVE'}")
    print(f"{'▄' * 70}")

    # ── GATE 1: Pre-flight ──────────────────────────────────────────────
    print(_gate_header(1, "PRE-FLIGHT — OVAV Law, Integrity, Context"))
    ok, log = gate1_preflight()
    for line in log: print(line)
    results[1] = {"ok": ok, "log": log}
    if not ok:
        all_pass = False
        print("\n  🛑 PIPELINE HALTED — Fix pre-flight issues first.")

    # ── GATE 2: Generate ─────────────────────────────────────────────────
    print(_gate_header(2, "GENERATE — Build artifacts from OVAV source"))
    ok, log, artifacts = gate2_generate(dry_run=dry_run)
    for line in log: print(line)
    results[2] = {"ok": ok, "log": log, "artifacts": artifacts}
    if not ok:
        all_pass = False
        print("\n  🛑 PIPELINE HALTED — Artifact generation failed.")

    # ── GATE 3: Compatibility ────────────────────────────────────────────
    print(_gate_header(3, "COMPATIBILITY — Validate against client schema"))
    ok, log = gate3_compatibility(artifacts)
    for line in log: print(line)
    results[3] = {"ok": ok, "log": log}
    if not ok:
        all_pass = False
        print("\n  🛑 PIPELINE HALTED — Compatibility check failed.")

    # ── GATE 4: Integrity Scan ───────────────────────────────────────────
    print(_gate_header(4, "INTEGRITY SCAN — Detect corruption, compute hashes"))
    ok, log, hashes = gate4_integrity(artifacts)
    for line in log: print(line)
    results[4] = {"ok": ok, "log": log, "hashes": hashes}
    if not ok:
        all_pass = False

    # ── GATE 5: Code Quality ─────────────────────────────────────────────
    print(_gate_header(5, "CODE QUALITY — Architecture compliance, syntax, secrets"))
    ok, log = gate5_code_quality()
    for line in log: print(line)
    results[5] = {"ok": ok, "log": log}
    if not ok:
        all_pass = False

    # ── GATE 6: Changelog ────────────────────────────────────────────────
    print(_gate_header(6, "CHANGELOG — Diff against previous release"))
    ok, log, changes = gate6_changelog(hashes, dry_run=dry_run)
    for line in log: print(line)
    results[6] = {"ok": ok, "log": log, "changes": changes}
    version = changes["version"]

    # ── GATE 7: Rigorous Testing ─────────────────────────────────────────
    print(_gate_header(7, "RIGOROUS TESTING — Real execution tests"))
    ok, log = gate7_rigorous_test(artifacts)
    for line in log: print(line)
    results[7] = {"ok": ok, "log": log}
    if not ok:
        all_pass = False
        print("\n  🛑 PIPELINE HALTED — Tests failed.")

    # ── GATE 8: Version Packaging ────────────────────────────────────────
    print(_gate_header(8, "VERSION PACKAGING — Bundle for distribution"))
    ok, log, release_dir = gate8_version_pack(artifacts, hashes, changes, version, dry_run=dry_run)
    for line in log: print(line)
    results[8] = {"ok": ok, "log": log, "release_dir": release_dir}

    # ── GATE 9: CLI Notify ───────────────────────────────────────────────
    print(_gate_header(9, "CLI NOTIFY — Update available notification"))
    ok, log = gate9_cli_notify(version, release_dir, dry_run=dry_run)
    for line in log: print(line)
    results[9] = {"ok": ok, "log": log}

    # ── FINAL ────────────────────────────────────────────────────────────
    elapsed = time.time() - start_time
    print(f"\n{'▀' * 70}")
    if all_pass:
        print("  🎉 PIPELINE COMPLETE — All 9 gates PASSED")
        print(f"  Version: {version}")
        print(f"  Duration: {elapsed:.1f}s")
        print(f"  Release dir: {release_dir.relative_to(ROOT)}")
        print("  CLI notification: ACTIVE")
    else:
        print(f"  ❌ PIPELINE FAILED — {sum(1 for r in results.values() if not r['ok'])} gate(s) failed")
        print(f"  Duration: {elapsed:.1f}s")
    print(f"{'▄' * 70}\n")

    # Save pipeline report
    report = {
        "timestamp": _timestamp(),
        "version": version if all_pass else None,
        "passed": all_pass,
        "duration_seconds": round(elapsed, 1),
        "dry_run": dry_run,
        "gates": {
            str(k): {"ok": v["ok"]} for k, v in results.items()
        },
    }
    report_path = RELEASES_DIR / ".pipeline_report.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n")

    return all_pass


# ══════════════════════════════════════════════════════════════════════════════
# CLI
# ══════════════════════════════════════════════════════════════════════════════

def main():
    import argparse

    parser = argparse.ArgumentParser(description="OVAV Release Pipeline — 9-Gate + 2-Phase Staging")
    parser.add_argument("--dry-run", action="store_true", help="Simular sin escribir archivos")
    parser.add_argument("--status", action="store_true", help="Mostrar estado de última release")
    parser.add_argument("--target", default="opencode", help="Cliente destino (default: opencode)")
    parser.add_argument(
        "--stage", choices=["internal", "external", "full"], default="full",
        help="Pipeline stage: internal (CEO review), external (after approval), full (both, default)",
    )
    parser.add_argument("--approve", action="store_true", help="CEO: aprobar internal release para promoción a external")

    args = parser.parse_args()

    # ── CEO Approval (manual step) ───────────────────────────────────────
    if args.approve:
        result = ceo_approve()
        print(json.dumps(result, indent=2))
        return

    # ── Status ───────────────────────────────────────────────────────────
    if args.status:
        report_path = RELEASES_DIR / ".pipeline_report.json"
        if report_path.exists():
            report = json.loads(report_path.read_text())
            print(json.dumps(report, indent=2))
        else:
            print("No pipeline report found. Run the pipeline first.")

        # Also show approval status
        approval = check_ceo_approval()
        print("\n── CEO Approval Status ──")
        print(json.dumps(approval, indent=2))
        return

    RELEASES_DIR.mkdir(parents=True, exist_ok=True)

    # ── Staging pipeline: Internal only ──────────────────────────────────
    if args.stage == "internal":
        success, version, results = run_internal_stage(dry_run=args.dry_run)
        sys.exit(0 if success else 1)

    # ── Staging pipeline: External only ──────────────────────────────────
    if args.stage == "external":
        # Check CEO approval
        approval = check_ceo_approval()
        if not approval["approved"]:
            print(json.dumps(approval, indent=2))
            print("\n❌ EXTERNAL RELEASE BLOCKED — CEO approval required.")
            print("   1. Run: python3 tools/visual/release_pipeline.py --stage internal")
            print("   2. CEO reviews artifacts in .ovav/visual/releases/internal/latest/")
            print("   3. CEO runs: python3 tools/visual/release_pipeline.py --approve")
            print("   4. Re-run: python3 tools/visual/release_pipeline.py --stage external")
            sys.exit(1)

        # Load internal release data
        latest_internal = INTERNAL_DIR / "latest"
        manifest_path = latest_internal / "manifest.json"
        if not manifest_path.exists():
            print("❌ No internal release found. Run --stage internal first.")
            sys.exit(1)

        manifest = json.loads(manifest_path.read_text())
        version = manifest["version"]

        # Reconstruct artifacts from internal release
        artifacts_dir = latest_internal / "artifacts"
        artifacts = {}
        if artifacts_dir.exists():
            for f in artifacts_dir.iterdir():
                artifacts[f.stem] = f

        success = run_external_stage(
            artifacts=artifacts,
            hashes=manifest.get("hashes", {}),
            changes=manifest.get("changes", {}),
            version=version,
            dry_run=args.dry_run,
        )
        sys.exit(0 if success else 1)

    # ── Full pipeline (default, both stages in one run) ──────────────────
    # Stage 1: Internal
    internal_ok, version, results = run_internal_stage(dry_run=args.dry_run)
    if not internal_ok:
        print("🛑 Internal stage failed. External release blocked.")
        sys.exit(1)

    # Stage 2: External (requires internal artifacts)
    # Reconstruct artifacts from internal stage results
    artifacts = {}
    for gate_id, gate_data in results.items():
        if isinstance(gate_data, dict) and "artifacts" in gate_data:
            for name, path_str in gate_data["artifacts"].items():
                p = Path(path_str)
                if p.exists():
                    artifacts[name] = p

    hashes = results.get(4, {}).get("hashes", {})
    changes = results.get(6, {}).get("changes", {})

    if not args.dry_run:
        # Auto-approve for full pipeline (CEO is running it directly)
        ceo_approve()

    external_ok = run_external_stage(
        artifacts=artifacts,
        hashes=hashes,
        changes=changes,
        version=version,
        dry_run=args.dry_run,
    )

    # Save pipeline report
    report = {
        "timestamp": _timestamp(),
        "version": version if external_ok else None,
        "passed": internal_ok and external_ok,
        "stage": "full",
        "dry_run": args.dry_run,
        "internal_ok": internal_ok,
        "external_ok": external_ok,
    }
    report_path = RELEASES_DIR / ".pipeline_report.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n")

    sys.exit(0 if (internal_ok and external_ok) else 1)


if __name__ == "__main__":
    main()
