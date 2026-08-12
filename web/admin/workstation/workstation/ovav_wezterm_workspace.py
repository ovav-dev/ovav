#!/usr/bin/env python3
"""OVAV WezTerm workspace isolation helper.

Source-local implementation for repo+branch scoped WezTerm workspaces.
It plans, diagnoses and prints launch commands, but it does not execute WezTerm,
write user home config, write Windows config, mutate Git branches or change remotes.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[2]
DOC = REPO_ROOT / "docs" / "workstation" / "OVAV_WEZTERM_WORKSPACE_ISOLATION.md"
POLICY = REPO_ROOT / "config" / "workstation" / "ovav-wezterm-workspace-isolation.yaml"
WEZTERM_TEMPLATE = (
    REPO_ROOT
    / ".ovav"
    / "source"
    / "configs"
    / "wezterm"
    / "ovav-workspace-isolation.wezterm.lua.example"
)
GOVERNED_WEZTERM_CONFIG = REPO_ROOT / "config" / "wezterm" / "wezterm.lua"
WINDOWS_WEZTERM_LOADER = (
    REPO_ROOT / ".ovav" / "source" / "configs" / "wezterm" / "ovav-windows-loader.wezterm.lua"
)
VALIDATOR = REPO_ROOT / "tools" / "validators" / "check_ovav_wezterm_workspace_isolation.py"
TOOL_CONFIGS = REPO_ROOT / ".ovav" / "registry" / "tool_configs.yaml"

WORKSPACE_PREFIX = "ovav"
MAX_WORKSPACE_LENGTH = 96


def _rel(path: Path) -> str:
    return str(path.relative_to(REPO_ROOT))


def _run_git(args: list[str], cwd: Path) -> str | None:
    try:
        result = subprocess.run(
            ["git", *args],
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    if result.returncode != 0:
        return None
    return result.stdout.strip() or None


def git_root(path: Path) -> Path:
    detected = _run_git(["rev-parse", "--show-toplevel"], path)
    if detected:
        return Path(detected).resolve()
    return path.resolve()


def git_branch(path: Path) -> str:
    branch = _run_git(["branch", "--show-current"], path)
    if branch:
        return branch
    sha = _run_git(["rev-parse", "--short", "HEAD"], path)
    if sha:
        return f"detached-{sha}"
    return "unknown-branch"


def git_dirty(path: Path) -> bool | None:
    try:
        result = subprocess.run(
            ["git", "status", "--short"],
            cwd=path,
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return None
    if result.returncode != 0:
        return None
    return bool(result.stdout.strip())


def slugify(value: str, *, max_length: int = 48) -> str:
    lowered = value.strip().lower()
    slug = re.sub(r"[^a-z0-9._-]+", "-", lowered)
    slug = re.sub(r"-+", "-", slug).strip("-._")
    if not slug:
        slug = "unknown"
    return slug[:max_length].strip("-._") or "unknown"


def root_hash(root: Path) -> str:
    return hashlib.sha256(str(root).encode("utf-8")).hexdigest()[:8]


def workspace_name(repo_path: Path) -> dict[str, Any]:
    root = git_root(repo_path)
    branch = git_branch(root)
    repo_slug = slugify(root.name, max_length=32)
    branch_slug = slugify(branch, max_length=48)
    digest = root_hash(root)
    name = f"{WORKSPACE_PREFIX}-{repo_slug}-{branch_slug}-{digest}"
    if len(name) > MAX_WORKSPACE_LENGTH:
        available_branch = max(
            12, MAX_WORKSPACE_LENGTH - len(f"{WORKSPACE_PREFIX}-{repo_slug}--{digest}")
        )
        branch_slug = slugify(branch, max_length=available_branch)
        name = f"{WORKSPACE_PREFIX}-{repo_slug}-{branch_slug}-{digest}"
    return {
        "workspace": name,
        "repo_slug": repo_slug,
        "branch": branch,
        "branch_slug": branch_slug,
        "root_hash": digest,
        "repo_root": str(root),
        "workspace_scope": "repo_branch_root_hash",
    }


def _launch_env(meta: dict[str, Any]) -> dict[str, str]:
    return {
        "OVAV_WEZTERM_WORKSPACE": meta["workspace"],
        "OVAV_GIT_BRANCH": meta["branch"],
        "OVAV_REPO_ROOT": meta["repo_root"],
    }


def launch_command(repo_path: Path) -> dict[str, Any]:
    meta = workspace_name(repo_path)
    env = _launch_env(meta)
    command = [
        "wezterm",
        "start",
        "--cwd",
        meta["repo_root"],
        "--workspace",
        meta["workspace"],
    ]
    cli_spawn_new_window = [
        "wezterm",
        "cli",
        "spawn",
        "--new-window",
        "--cwd",
        meta["repo_root"],
        "--workspace",
        meta["workspace"],
    ]
    return {
        "status": "pass",
        "mode": "launch-command-dry-run",
        "writes_performed": False,
        "commands_executed": False,
        "workspace": meta,
        "environment": env,
        "recommended": {"env": env, "argv": command},
        "alternate_existing_gui_spawn": {"env": env, "argv": cli_spawn_new_window},
    }


def check_current(repo_path: Path) -> dict[str, Any]:
    meta = workspace_name(repo_path)
    expected_env = _launch_env(meta)
    observed = {
        "OVAV_WEZTERM_WORKSPACE": os.environ.get("OVAV_WEZTERM_WORKSPACE"),
        "OVAV_GIT_BRANCH": os.environ.get("OVAV_GIT_BRANCH"),
        "OVAV_REPO_ROOT": os.environ.get("OVAV_REPO_ROOT"),
    }
    mismatches = [name for name, expected in expected_env.items() if observed.get(name) != expected]
    ok = not mismatches
    return {
        "status": "pass" if ok else "blocked",
        "mode": "check-current-pane",
        "writes_performed": False,
        "workspace": meta,
        "expected_environment": expected_env,
        "observed_environment": observed,
        "mismatches": mismatches,
        "safe_next_action": "Continue in this pane."
        if ok
        else "Open the branch-scoped WezTerm launch command before running branch-sensitive work.",
    }


def build_plan(repo_path: Path) -> dict[str, Any]:
    meta = workspace_name(repo_path)
    return {
        "status": "pass",
        "mode": "dry-run",
        "profile_id": "ovav-wezterm-workspace-isolation",
        "writes_performed": False,
        "workspace": meta,
        "source_templates": {
            "wezterm": _rel(WEZTERM_TEMPLATE),
            "governed_wezterm_config": _rel(GOVERNED_WEZTERM_CONFIG),
            "windows_wezterm_loader": _rel(WINDOWS_WEZTERM_LOADER),
            "policy": _rel(POLICY),
            "doc": _rel(DOC),
            "tool_config_profile": _rel(TOOL_CONFIGS),
        },
        "tailor_surface": {
            "profile_id": "wezterm_workspace_isolation",
            "plan_command": "ovav tools wezterm plan",
            "verify_command": "ovav tools wezterm verify",
            "ovav_installs_wezterm": False,
        },
        "isolation_contract": {
            "workspace_scope": "repo_branch_root_hash",
            "pane_policy": "inherit_current_workspace_only",
            "tab_policy": "inherit_current_workspace_only",
            "cross_branch_attach": "blocked_by_workspace_name_mismatch",
            "visual_boundary": "workspace_and_branch_in_title_tab_status",
        },
        "gates_required_for_real_apply": [
            "explicit_user_approval",
            "backup_existing_wezterm_config",
            "install_or_include_source_template",
            "open_branch_scoped_workspace",
            "verify_visible_title_tab_status",
            "record_rollback_manifest",
        ],
        "blocked_now": [
            "no_user_home_config_writes",
            "no_windows_user_config_writes",
            "no_wezterm_process_launch",
            "no_git_branch_or_remote_mutation",
            "no_plugin_installation",
        ],
    }


def diagnose(repo_path: Path) -> dict[str, Any]:
    root = git_root(repo_path)
    meta = workspace_name(root)
    dirty = git_dirty(root)
    return {
        "status": "pass",
        "mode": "safe-diagnose",
        "writes_performed": False,
        "secret_material_read": False,
        "workspace": meta,
        "git": {
            "branch": meta["branch"],
            "dirty_state_detected": dirty,
            "mutated": False,
        },
        "wezterm": {
            "config_read": False,
            "process_started": False,
            "global_config_written": False,
        },
        "advisory": [
            "Use the computed workspace name before opening panes for this branch.",
            "A different branch must produce a different workspace name.",
        ],
    }


def verify_source() -> dict[str, Any]:
    try:
        result = subprocess.run(
            [sys.executable, str(VALIDATOR)],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            timeout=10,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        return {"status": "fail", "error": str(exc), "writes_performed": False}

    try:
        validator_payload = json.loads(result.stdout)
    except json.JSONDecodeError:
        validator_payload = {"raw_stdout": result.stdout[-1000:]}

    required = [
        DOC,
        POLICY,
        WEZTERM_TEMPLATE,
        GOVERNED_WEZTERM_CONFIG,
        WINDOWS_WEZTERM_LOADER,
        VALIDATOR,
        TOOL_CONFIGS,
    ]
    files = [{"path": _rel(path), "exists": path.exists()} for path in required]
    ok = result.returncode == 0 and all(item["exists"] for item in files)
    return {
        "status": "pass" if ok else "fail",
        "mode": "verify-source",
        "writes_performed": False,
        "files": files,
        "validator": validator_payload,
    }


def blocked_apply() -> dict[str, Any]:
    return {
        "status": "blocked",
        "mode": "apply",
        "writes_performed": False,
        "reason": "Real WezTerm apply is blocked until an explicit install segment is approved.",
        "would_touch_after_approval": [
            "/home/braka/.config/wezterm/wezterm.lua (canonical WSL)",
            "%USERPROFILE%\\.wezterm.lua (Windows proxy entry point)",
            "%USERPROFILE%\\.wezterm-fallback.lua (local fallback cache)",
            "real WezTerm windows/workspaces after operator launch",
        ],
        "safe_next_action": "Review plan output, validate source artifacts, then request a governed real-install segment if desired.",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="OVAV WezTerm workspace isolation helper")
    parser.add_argument(
        "--repo", default=str(REPO_ROOT), help="Repository path to scope the workspace name"
    )
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("plan", help="Print dry-run isolation plan")
    sub.add_parser("diagnose", help="Run safe source/environment diagnosis")
    sub.add_parser("workspace-name", help="Print computed workspace name")
    sub.add_parser("launch-command", help="Print dry-run launch command without executing WezTerm")
    sub.add_parser(
        "check-current", help="Check current pane env against the expected branch-scoped workspace"
    )
    sub.add_parser("verify-source", help="Validate source-local artifacts")
    sub.add_parser("apply", help="Blocked placeholder for future governed real apply")
    args = parser.parse_args()

    repo_path = Path(args.repo).resolve()
    if args.command == "plan":
        payload = build_plan(repo_path)
    elif args.command == "diagnose":
        payload = diagnose(repo_path)
    elif args.command == "workspace-name":
        payload = {
            "status": "pass",
            "mode": "workspace-name",
            "writes_performed": False,
            **workspace_name(repo_path),
        }
    elif args.command == "launch-command":
        payload = launch_command(repo_path)
    elif args.command == "check-current":
        payload = check_current(repo_path)
    elif args.command == "verify-source":
        payload = verify_source()
    else:
        payload = blocked_apply()

    print(json.dumps(payload, indent=2, sort_keys=True))
    return 0 if payload["status"] in {"pass", "blocked"} else 1


if __name__ == "__main__":
    sys.exit(main())
