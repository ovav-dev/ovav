#!/usr/bin/env python3
"""Tests para D2 CLI RC10 — NerveBus Live Experience.

Cubre:
  - Display formatting: _event_to_line(), _pain_bar(), _severity_badge()
  - CLI commands: status, latest (plain/detail), export (json/csv)
  - Watch: non-interactive mode
  - Dashboard: non-interactive component verification
  - Integration: PainScorer health in status, Lockdown in display
"""

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))


passed = 0
failed = 0


def ok(msg: str) -> str:
    return f"  ✅ {msg}"


def fail(msg: str) -> str:
    return f"  ❌ {msg}"


def assert_true(condition: bool, msg: str):
    global passed, failed
    if condition:
        passed += 1
        print(ok(msg))
    else:
        failed += 1
        print(fail(msg))


def assert_in(needle, haystack, msg: str):
    assert_true(needle in haystack, f"{msg}: '{needle}' in output")


def assert_not_in(needle, haystack, msg: str):
    assert_true(needle not in haystack, f"{msg}: '{needle}' NOT in output")


# ══════════════════════════════════════════════════════════════════════════════
# DISPLAY FORMATTING
# ══════════════════════════════════════════════════════════════════════════════

def test_severity_badge():
    """D2.DISP.01: _severity_badge produce salida coloreada."""
    from tools.agent_runtime.nerve_bus import _severity_badge

    b1 = _severity_badge("blockade")
    b2 = _severity_badge("critical")
    b3 = _severity_badge("info")

    assert_in("⛔", b1, "blockade icon")
    assert_in("🔴", b2, "critical icon")
    assert_in("🔵", b3, "info icon")
    assert_in("\033", b1, "ANSI escape codes present")


def test_pain_bar():
    """D2.DISP.02: _pain_bar produce barra proporcional."""
    from tools.agent_runtime.nerve_bus import _pain_bar

    b0 = _pain_bar(0, 10)
    b50 = _pain_bar(50, 10)
    b100 = _pain_bar(100, 10)

    assert_in("█", b100, "full bar has filled blocks")
    assert_in("░", b0, "empty bar has empty blocks")
    # 50 should have roughly half filled
    filled_50 = b50.count("█")
    assert_true(4 <= filled_50 <= 6, f"50% pain = ~5 filled: got {filled_50}")


def test_pain_color():
    """D2.DISP.03: _pain_color asigna colores por rango."""
    from tools.agent_runtime.nerve_bus import _pain_color

    c0 = _pain_color(0)
    c30 = _pain_color(30)
    c80 = _pain_color(80)
    c95 = _pain_color(95)

    assert_true(c0 != c30, "Low pain != medium pain color")
    assert_true(c80 != c30, "High pain != medium pain color")
    assert_in("\033", c95, "All colors have ANSI codes")


def test_event_to_line():
    """D2.DISP.04: _event_to_line formatea evento completo."""
    from tools.agent_runtime.nerve_bus import _event_to_line

    event = {
        "timestamp": "2026-06-04T12:00:00.000Z",
        "severity": "warning",
        "type": "config_drift",
        "source": "integrity_mesh",
        "pain_score": 42,
        "payload": {"message": "AGENTS.md modified", "file": "AGENTS.md"},
    }

    line = _event_to_line(event)
    assert_in("12:00:00", line, "timestamp formatted")
    assert_in("warning", line, "severity shown")
    assert_in("config_drift", line, "type shown")
    assert_in("integrity_mesh", line, "source shown")

    # Detailed mode
    detail = _event_to_line(event, detailed=True)
    assert_in("AGENTS.md", detail, "detail includes payload key")


def test_event_to_line_blockade():
    """D2.DISP.05: Evento blockade tiene formato de alerta máxima."""
    from tools.agent_runtime.nerve_bus import _event_to_line

    event = {
        "timestamp": "2026-06-04T12:00:00.000Z",
        "severity": "blockade",
        "type": "lockdown_active",
        "source": "governor",
        "pain_score": 100,
        "payload": {"reason": "integrity breach"},
    }

    line = _event_to_line(event)
    assert_in("⛔", line, "blockade icon")
    assert_in("100", line, "pain score shown")


# ══════════════════════════════════════════════════════════════════════════════
# CLI COMMANDS
# ══════════════════════════════════════════════════════════════════════════════

def test_status_command():
    """D2.CLI.01: status command runs without error."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "status"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, f"status exit=0: got {result.returncode}")
    assert_in("NerveBus Status", result.stdout, "title present")
    assert_in("Eventos:", result.stdout, "event count present")
    assert_in("Lockdown:", result.stdout, "lockdown status present")


def test_latest_command():
    """D2.CLI.02: latest command runs with rich formatting."""
    import subprocess
    # Publish a test event first
    subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "publish",
         "test_cli", '{"msg":"cli_test"}', "--severity", "info", "--source", "d2_test"],
        capture_output=True, cwd=ROOT, timeout=10,
    )

    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "latest", "--count", "1"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, f"latest exit=0: got {result.returncode}")
    assert_in("d2_test", result.stdout, "source in output")


def test_latest_plain():
    """D2.CLI.03: latest --plain produce formato legacy."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "latest", "--count", "1", "--plain"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, f"plain exit=0: got {result.returncode}")
    assert_not_in("\033", result.stdout, "no ANSI codes in plain mode")


def test_latest_detail():
    """D2.CLI.04: latest --detail muestra payload."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "latest", "--count", "1", "--detail"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, "detail exit=0")


def test_export_json():
    """D2.CLI.05: export --format json produce JSON válido."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "export", "--format", "json", "--hours", "1"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, f"export exit=0: got {result.returncode}")
    try:
        data = json.loads(result.stdout)
        assert_in("events", data, "JSON has events key")
        assert_true(isinstance(data["events"], list), "events is list")
    except json.JSONDecodeError:
        assert_true(False, "Output is valid JSON")


def test_export_csv():
    """D2.CLI.06: export --format csv produce CSV."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "export", "--format", "csv", "--hours", "1"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, "csv export exit=0")
    lines = result.stdout.strip().split("\n")
    assert_true(len(lines) >= 1, "CSV has at least header line")


def test_watch_once():
    """D2.CLI.07: watch --once ejecuta sin error (modo no interactivo)."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "watch", "--once", "--lines", "3", "--refresh", "0.2"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, f"watch exit=0: got {result.returncode}")
    assert_in("NerveBus Live", result.stdout, "header present")


def test_publish_colored():
    """D2.CLI.08: publish produce output coloreado."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "publish",
         "d2_test_colored", '{"msg":"color"}', "--severity", "info", "--source", "d2"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, "publish exit=0")
    assert_in("Evento publicado", result.stdout, "success message")


def test_history_command():
    """D2.CLI.09: history command funciona."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "history", "--since", "2026-01-01T00:00:00"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_true(result.returncode == 0, "history exit=0")


def test_export_to_file():
    """D2.CLI.10: export --output escribe archivo."""
    import os
    import subprocess
    import tempfile
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
        tmp = f.name

    try:
        result = subprocess.run(
            ["python3", "tools/agent_runtime/nerve_bus.py", "export",
             "--format", "json", "--hours", "1", "--output", tmp],
            capture_output=True, text=True, cwd=ROOT, timeout=10,
        )
        assert_true(result.returncode == 0, "export to file exit=0")
        assert_true(Path(tmp).exists(), "output file exists")
        data = json.loads(Path(tmp).read_text())
        assert_true("events" in data, "output file has events")
    finally:
        os.unlink(tmp)


# ══════════════════════════════════════════════════════════════════════════════
# INTEGRATION
# ══════════════════════════════════════════════════════════════════════════════

def test_status_includes_pain_health():
    """D2.INT.01: status incluye health de PainScorer."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "status"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_in("Health:", result.stdout, "health score in status")


def test_status_includes_lockdown():
    """D2.INT.02: status muestra lockdown status."""
    import subprocess
    result = subprocess.run(
        ["python3", "tools/agent_runtime/nerve_bus.py", "status"],
        capture_output=True, text=True, cwd=ROOT, timeout=10,
    )
    assert_in("unlocked", result.stdout, "lockdown status present")


def test_display_helpers_exist():
    """D2.INT.03: Todos los helpers de display son importables."""
    from tools.agent_runtime.nerve_bus import (
        _display_status_bar,
        _event_to_line,
        _pain_bar,
        _pain_color,
        _severity_badge,
    )
    assert_true(callable(_pain_color), "_pain_color callable")
    assert_true(callable(_pain_bar), "_pain_bar callable")
    assert_true(callable(_severity_badge), "_severity_badge callable")
    assert_true(callable(_event_to_line), "_event_to_line callable")
    assert_true(callable(_display_status_bar), "_display_status_bar callable")


# ══════════════════════════════════════════════════════════════════════════════

def run_all():
    global passed, failed
    print("╔══════════════════════════════════════════════════╗")
    print("║       D2 CLI RC10 — Test Suite                   ║")
    print("╚══════════════════════════════════════════════════╝")
    print()

    sections = [
        ("DISPLAY FORMATTING", [
            test_severity_badge, test_pain_bar, test_pain_color,
            test_event_to_line, test_event_to_line_blockade,
        ]),
        ("CLI COMMANDS", [
            test_status_command, test_latest_command, test_latest_plain,
            test_latest_detail, test_export_json, test_export_csv,
            test_watch_once, test_publish_colored, test_history_command,
            test_export_to_file,
        ]),
        ("INTEGRATION", [
            test_status_includes_pain_health, test_status_includes_lockdown,
            test_display_helpers_exist,
        ]),
    ]

    for section_name, tests in sections:
        print(f"\n── {section_name} ──")
        for test_fn in tests:
            try:
                test_fn()
            except Exception as e:
                failed += 1
                print(fail(f"{test_fn.__name__}: EXCEPTION — {e}"))

    print(f"\n{'='*50}")
    print(f"  Results: {passed} passed, {failed} failed")
    print(f"{'='*50}")

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(run_all())
