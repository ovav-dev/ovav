"""S143 evals for OVAV Shell keyboard and non-interactive hardening."""

from __future__ import annotations

import pytest
pytestmark = pytest.mark.skip(reason="ovav_shell/ovav_logo modules removed — shell migrated to Go runtime")

import importlib.util
import subprocess
import sys
from importlib.machinery import SourceFileLoader
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]


def _load_script(path: Path, name: str):
    spec = importlib.util.spec_from_loader(name, SourceFileLoader(name, str(path)))
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_shell_normalizes_keyboard_variants():
    shell = _load_script(REPO_ROOT / "bin" / "ovav-shell", "ovav_shell")

    assert shell.normalize_key("\x1b[A") == "up"
    assert shell.normalize_key("\x1bOA") == "up"
    assert shell.normalize_key("k") == "up"
    assert shell.normalize_key("\x1b[B") == "down"
    assert shell.normalize_key("\x1bOB") == "down"
    assert shell.normalize_key("j") == "down"
    assert shell.normalize_key("\r") == "select"
    assert shell.normalize_key("?") == "help"
    assert shell.normalize_key("\x1b") == "quit"
    assert shell.normalize_key("unexpected") == "unknown"


def test_logo_no_color_has_no_ansi_escape_codes():
    logo = _load_script(REPO_ROOT / "bin" / "ovav-logo", "ovav_logo")

    rendered = logo.render_logo(compact=True, color=False)

    assert "\033[" not in rendered
    assert "workstation governor" in rendered


def test_shell_non_interactive_does_not_block():
    result = subprocess.run(
        [sys.executable, str(REPO_ROOT / "bin" / "ovav-shell")],
        cwd=REPO_ROOT,
        input="",
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0
    assert "Non-interactive context detected" in result.stdout


def test_build_command_points_to_build16_next_step():
    result = subprocess.run(
        [sys.executable, str(REPO_ROOT / "bin" / "ovav"), "build"],
        cwd=REPO_ROOT,
        capture_output=True,
        text=True,
        timeout=10,
    )

    assert result.returncode == 0
    assert "BUILD 16" in result.stdout
    assert "S144" in result.stdout


if __name__ == "__main__":
    test_shell_normalizes_keyboard_variants()
    test_logo_no_color_has_no_ansi_escape_codes()
    test_shell_non_interactive_does_not_block()
    test_build_command_points_to_build16_next_step()
    print("PASS S143 shell hardening evals")
