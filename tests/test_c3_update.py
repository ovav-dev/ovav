"""Tests for C3.3 — ovav update and uninstall commands.

Run: python3 -m pytest tests/test_c3_update.py -v
"""

from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[1]
OVAV_DEV = "1"


def _run_update(args: list[str]) -> subprocess.CompletedProcess:
    """Run ovav_official_update.py with given args."""
    return subprocess.run(
        [sys.executable, str(REPO_ROOT / "tools" / "cli" / "ovav_official_update.py"), *args],
        capture_output=True, text=True,
        cwd=str(REPO_ROOT),
        env={**os.environ, "OVAV_DEV": OVAV_DEV},
    )


def _run_uninstall(args: list[str]) -> subprocess.CompletedProcess:
    """Run ovav_clean_uninstall.py with given args."""
    return subprocess.run(
        [sys.executable, str(REPO_ROOT / "tools" / "install" / "ovav_clean_uninstall.py"), *args],
        capture_output=True, text=True,
        cwd=str(REPO_ROOT),
        env={**os.environ, "OVAV_DEV": OVAV_DEV},
    )


def _run_public_cli(cmd: list[str]) -> subprocess.CompletedProcess:
    """Run ovav_public_cli.py with given args."""
    return subprocess.run(
        [sys.executable, str(REPO_ROOT / "tools" / "cli" / "ovav_public_cli.py"), *cmd],
        capture_output=True, text=True,
        cwd=str(REPO_ROOT),
        env={**os.environ, "OVAV_DEV": OVAV_DEV},
    )


class TestUpdateCheck:
    """Tests for ovav update --check."""

    def test_update_check_returns_ok(self):
        result = _run_update(["--check"])
        assert result.returncode == 0
        assert "status" in result.stdout.lower() or "up-to-date" in result.stdout.lower()

    def test_update_check_json(self):
        result = _run_update(["--check", "--json"])
        assert result.returncode == 0
        import json
        data = json.loads(result.stdout)
        assert data["schema_version"] == "ovav.official_update.v1"
        assert "status" in data
        assert "local_version" in data
        assert "update_available" in data
        assert "can_apply" in data
        # C3.3.2: pipeline_release field should exist (may be null)
        assert "pipeline_release" in data

    def test_update_check_via_public_cli(self):
        result = _run_public_cli(["update", "--check"])
        assert result.returncode == 0
        assert "OVAV Update" in result.stdout

    def test_update_check_short_alias(self):
        result = _run_public_cli(["up", "--check"])
        assert result.returncode == 0


class TestUpdateApply:
    """Tests for ovav update --apply (safety gates, not actual apply)."""

    def test_update_apply_requires_consent(self):
        """Apply without consent should be blocked."""
        result = _run_update(["--apply"])
        # Should fail because consent + risk acceptance missing
        assert result.returncode != 0 or "blocked" in result.stdout.lower() or "consent" in result.stdout.lower()

    def test_update_apply_with_consent_no_risk(self):
        """Apply with consent but without risk acceptance should be blocked."""
        result = _run_update(["--apply", "--consent"])
        assert result.returncode != 0 or "blocked" in result.stdout.lower()


class TestUninstall:
    """Tests for ovav uninstall."""

    def test_uninstall_scan(self):
        result = _run_uninstall(["--scan"])
        assert result.returncode == 0
        # Should detect at least the .opencode symlink
        assert ".opencode" in result.stdout or "artefacto" in result.stdout.lower() or "No se encontraron" in result.stdout

    def test_uninstall_no_args_shows_help(self):
        result = _run_uninstall([])
        assert result.returncode == 0
        assert "--scan" in result.stdout
        assert "--clean" in result.stdout
        assert "--force" in result.stdout

    def test_uninstall_via_public_cli(self):
        result = _run_public_cli(["uninstall", "--scan"])
        assert result.returncode == 0
        assert "OVAV Uninstall" in result.stdout

    def test_uninstall_short_alias(self):
        result = _run_public_cli(["rm", "--scan"])
        assert result.returncode == 0


class TestPublicCLI:
    """Tests for ovav public CLI surface."""

    def test_help_shows_all_commands(self):
        result = _run_public_cli(["help"])
        assert result.returncode == 0
        assert "update" in result.stdout
        assert "uninstall" in result.stdout
        assert "profile" in result.stdout
        assert "status" in result.stdout
        assert "config" in result.stdout

    def test_profile_via_public_cli(self):
        result = _run_public_cli(["profile", "list"])
        assert result.returncode == 0
        assert "PROFILE" in result.stdout

    def test_profile_short_alias(self):
        result = _run_public_cli(["pf", "list"])
        assert result.returncode == 0
        assert "PROFILE" in result.stdout

    def test_config_shows_commands(self):
        result = _run_public_cli(["config"])
        assert result.returncode == 0
        assert "profile" in result.stdout.lower()

    def test_status_returns_ok(self):
        result = _run_public_cli(["status"])
        assert result.returncode == 0
        assert "OVAV" in result.stdout
