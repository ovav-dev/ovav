"""Tests for C3.5 — ovav profile commands (list, apply, remove).

Run: python3 -m pytest tests/test_c3_profile.py -v
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parents[1]
PROFILE_SCRIPT = str(REPO_ROOT / "tools" / "cli" / "ovav_profile.py")
OVAV_DEV = "1"
TEST_WORK_DIR = REPO_ROOT / ".ovav" / "runtime" / "test_work"


def _mkdtemp() -> str:
    """Create a temp directory under the repo (required by profile path validation)."""
    TEST_WORK_DIR.mkdir(parents=True, exist_ok=True)
    return tempfile.mkdtemp(dir=str(TEST_WORK_DIR))


def _run(cmd: list[str], cwd: str | None = None) -> subprocess.CompletedProcess:
    """Run ovav_profile.py with given args."""
    env = {**os.environ, "OVAV_DEV": OVAV_DEV}
    return subprocess.run(
        [sys.executable, PROFILE_SCRIPT, *cmd],
        capture_output=True, text=True, cwd=cwd or str(REPO_ROOT), env=env,
    )


class TestProfileList:
    """Tests for ovav profile list."""

    def test_list_shows_profiles(self):
        result = _run(["list"])
        assert result.returncode == 0
        assert "PROFILE" in result.stdout
        assert "platform_engineering" in result.stdout
        assert "research_intelligence" in result.stdout
        assert "health_performance" in result.stdout

    def test_list_json_output(self):
        result = _run(["list", "--json"])
        assert result.returncode == 0
        data = json.loads(result.stdout)
        assert "profiles" in data
        assert len(data["profiles"]) >= 3
        # Verify structure
        for p in data["profiles"]:
            assert "id" in p
            assert "name" in p
            assert "description" in p
            assert "lead" in p

    def test_list_p0_profiles_first(self):
        result = _run(["list", "--json"])
        data = json.loads(result.stdout)
        # P0 profiles should appear first
        p0_ids = [p["id"] for p in data["profiles"] if p["p0"]]
        non_p0_ids = [p["id"] for p in data["profiles"] if not p["p0"]]
        all_ids = [p["id"] for p in data["profiles"]]
        # Every P0 should appear before any non-P0
        if p0_ids and non_p0_ids:
            last_p0_idx = max(all_ids.index(pid) for pid in p0_ids)
            first_non_p0_idx = min(all_ids.index(pid) for pid in non_p0_ids)
            assert last_p0_idx < first_non_p0_idx


class TestProfileApply:
    """Tests for ovav profile apply."""

    def test_apply_help(self):
        result = _run(["apply", "--help"])
        assert result.returncode == 0
        assert "Usage: ovav profile apply" in result.stdout
        assert "--target" in result.stdout
        assert "--dry-run" in result.stdout
        assert "--yes" in result.stdout

    def test_apply_short_help(self):
        result = _run(["apply", "-h"])
        assert result.returncode == 0
        assert "Usage:" in result.stdout

    def test_apply_no_args_shows_help(self):
        result = _run(["apply"])
        assert result.returncode == 2
        assert "Usage:" in result.stdout

    def test_apply_invalid_profile(self):
        result = _run(["apply", "nonexistent_profile_12345"])
        assert result.returncode == 1
        assert "not found" in result.stderr.lower() or "not found" in result.stdout.lower()

    def test_apply_dry_run(self):
        result = _run(["apply", "research_intelligence", "--dry-run"])
        assert result.returncode == 0
        assert "dry-run" in result.stdout.lower()
        assert "no files written" in result.stdout.lower()

    def test_apply_dry_run_shows_files(self):
        result = _run(["apply", "health_performance", "--dry-run"])
        assert result.returncode == 0
        assert "AGENTS.md" in result.stdout
        assert "opencode.json" in result.stdout
        assert ".opencode/agents/" in result.stdout
        assert ".opencode/skills/" in result.stdout

    def test_apply_path_traversal_blocked(self):
        result = _run(["apply", "research_intelligence", "--target", "../outside"])
        assert result.returncode == 1
        err = (result.stderr + result.stdout).lower()
        assert "path traversal" in err or "must be under current directory" in err

    def test_apply_generates_files(self):
        """End-to-end: apply a profile in a temp dir and verify files exist."""
        tmp = _mkdtemp()
        try:
            result = _run(
                ["apply", "research_intelligence", "--target", tmp, "--yes"],
            )
            assert result.returncode == 0
            # Verify generated files
            assert (Path(tmp) / "AGENTS.md").exists()
            assert (Path(tmp) / "opencode.json").exists()
            assert (Path(tmp) / ".opencode" / "agents").is_dir()
            assert (Path(tmp) / ".opencode" / "skills").is_dir()
            # Verify opencode.json is valid JSON with profile key
            config = json.loads((Path(tmp) / "opencode.json").read_text())
            assert "profile" in config
            assert config["profile"]["area"] == "research_intelligence"
            # Verify AGENTS.md has content
            agents_content = (Path(tmp) / "AGENTS.md").read_text()
            assert len(agents_content) > 100
            assert "Research" in agents_content or "research" in agents_content.lower()
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_apply_validation_warnings(self):
        """Apply with --yes should run validation and report warnings if any."""
        tmp = _mkdtemp()
        try:
            result = _run(
                ["apply", "research_intelligence", "--target", tmp, "--yes"],
            )
            # Should succeed even with warnings
            assert result.returncode == 0
            assert "✅" in result.stdout or "applied" in result.stdout.lower()
        finally:
            shutil.rmtree(tmp, ignore_errors=True)


class TestProfileRemove:
    """Tests for ovav profile remove."""

    def test_remove_help(self):
        result = _run(["remove", "--help"])
        assert result.returncode == 0
        assert "Usage: ovav profile remove" in result.stdout
        assert "--target" in result.stdout
        assert "--dry-run" in result.stdout
        assert "--yes" in result.stdout

    def test_remove_short_help(self):
        result = _run(["remove", "-h"])
        assert result.returncode == 0
        assert "Usage:" in result.stdout

    def test_remove_no_args_shows_help(self):
        result = _run(["remove"])
        assert result.returncode == 2
        assert "Usage:" in result.stdout

    def test_remove_invalid_profile(self):
        result = _run(["remove", "nonexistent_profile_12345"])
        assert result.returncode == 1
        assert "not found" in result.stderr.lower() or "not found" in result.stdout.lower()

    def test_remove_dry_run_no_files(self):
        """In a clean dir, remove --dry-run should show nothing to remove."""
        tmp = _mkdtemp()
        try:
            result = _run(
                ["remove", "research_intelligence", "--target", tmp, "--dry-run"],
            )
            # Return code 0 even when nothing to remove
            assert "No profile files found" in result.stdout or result.returncode == 0
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_remove_after_apply(self):
        """End-to-end: apply then remove a profile."""
        tmp = _mkdtemp()
        try:
            # Apply first
            apply_result = _run(
                ["apply", "research_intelligence", "--target", tmp, "--yes"],
            )
            assert apply_result.returncode == 0

            # Now remove
            remove_result = _run(
                ["remove", "research_intelligence", "--target", tmp, "--yes"],
            )
            assert remove_result.returncode == 0
            assert "removed" in remove_result.stdout.lower()

            # Verify files are gone
            assert not (Path(tmp) / "AGENTS.md").exists()
            assert not (Path(tmp) / "opencode.json").exists()
            assert not (Path(tmp) / ".opencode" / "agents").exists()
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_remove_dry_run_preview(self):
        """After apply, remove --dry-run should preview files without deleting."""
        tmp = _mkdtemp()
        try:
            # Apply first
            _run(["apply", "research_intelligence", "--target", tmp, "--yes"])

            # Dry-run remove
            result = _run(
                ["remove", "research_intelligence", "--target", tmp, "--dry-run"],
            )
            assert result.returncode == 0
            assert "dry-run" in result.stdout.lower()
            assert "no files removed" in result.stdout.lower()

            # Files should still exist
            assert (Path(tmp) / "AGENTS.md").exists()
            assert (Path(tmp) / "opencode.json").exists()
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_remove_path_traversal_blocked(self):
        result = _run(["remove", "research_intelligence", "--target", "../outside"])
        assert result.returncode == 1
        err = (result.stderr + result.stdout).lower()
        assert "path traversal" in err or "must be under current directory" in err
