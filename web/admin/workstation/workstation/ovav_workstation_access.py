#!/usr/bin/env python3
"""OVAV Workstation Access Profile helper.

Source-local implementation for the OVAV/Thavren GitHub SSH profile.
It can plan, diagnose and verify artifacts, but it does not write to user
home, global config, Windows config, SSH config, fish config or Git remotes.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[2]
DOC = REPO_ROOT / "docs" / "workstation" / "OVAV_THAVREN_SSH_PROFILE.md"
ACCESS_DOC = REPO_ROOT / "docs" / "workstation" / "OVAV_WORKSTATION_ACCESS_PROFILE.md"
INSTALL_DOC = REPO_ROOT / "docs" / "workstation" / "OVAV_THAVREN_SSH_INSTALL_PLAN.md"
SSH_TEMPLATE = REPO_ROOT / "config" / "ssh" / "ovav-thavren.ssh.config.example"
FISH_TEMPLATE = REPO_ROOT / "config" / "fish" / "ovav-thavren-ssh-agent.fish.example"
POLICY = REPO_ROOT / "config" / "workstation" / "ovav-thavren-ssh-profile.yaml"
INSTALL_PLAN = REPO_ROOT / "config" / "workstation" / "ovav-thavren-ssh-install-plan.yaml"
VALIDATOR = REPO_ROOT / "tools" / "validators" / "check_ovav_ssh_profile.py"

HOST_ALIAS = "github-ovav-thavren"
EXPECTED_LIFETIME = "24h"


def _rel(path: Path) -> str:
    return str(path.relative_to(REPO_ROOT))


def _git_remote_origin() -> str | None:
    try:
        result = subprocess.run(
            ["git", "remote", "get-url", "origin"],
            cwd=REPO_ROOT,
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


def _redact_remote(remote: str | None) -> str | None:
    if remote is None:
        return None
    if remote.startswith("git@"):
        host, _, path = remote.partition(":")
        repo = path.rsplit("/", 1)[-1] if path else "<repo>"
        return f"{host}:<owner>/{repo}"
    if "github.com" in remote:
        repo = remote.rstrip("/").rsplit("/", 1)[-1]
        return f"https://github.com/<owner>/{repo}"
    return "<remote-redacted>"


def build_plan() -> dict[str, Any]:
    return {
        "status": "pass",
        "mode": "dry-run",
        "profile_id": "ovav-thavren-ssh-profile",
        "host_alias": HOST_ALIAS,
        "agent_lifetime": EXPECTED_LIFETIME,
        "writes_performed": False,
        "source_templates": {
            "ssh": _rel(SSH_TEMPLATE),
            "fish": _rel(FISH_TEMPLATE),
            "policy": _rel(POLICY),
            "install_plan": _rel(INSTALL_PLAN),
        },
        "intended_targets_after_approval": {
            "ssh_config_fragment": "~/.ssh/config.d/ovav-thavren.conf",
            "fish_agent_helper": "~/.config/fish/conf.d/ovav-thavren-ssh-agent.fish",
            "private_key_path": "~/.ssh/ovav_thavren_ed25519",
            "public_key_path": "~/.ssh/ovav_thavren_ed25519.pub",
            "git_remote_shape": "git@github-ovav-thavren:ORG/REPO.git",
        },
        "gates_required_for_real_apply": [
            "explicit_user_approval",
            "backup_existing_targets",
            "confirm_or_create_dedicated_keypair",
            "install_ssh_fragment",
            "install_fish_helper",
            "unlock_key_once_with_24h_lifetime",
            "test_github_alias",
            "migrate_git_remote_after_alias_test",
            "verify_git_fetch",
            "record_rollback_manifest",
        ],
        "blocked_now": [
            "no_user_home_writes",
            "no_global_config_writes",
            "no_key_generation",
            "no_git_remote_mutation",
            "no_secret_storage",
        ],
    }


def diagnose() -> dict[str, Any]:
    remote = _git_remote_origin()
    auth_sock_present = bool(os.environ.get("SSH_AUTH_SOCK"))
    agent_pid_present = bool(os.environ.get("SSH_AGENT_PID"))
    return {
        "status": "pass",
        "mode": "safe-diagnose",
        "writes_performed": False,
        "secret_material_read": False,
        "git_origin": {
            "detected": remote is not None,
            "remote_redacted": _redact_remote(remote),
            "uses_ssh": bool(remote and remote.startswith("git@")),
            "uses_https": bool(remote and remote.startswith("https://")),
            "uses_ovav_alias": bool(remote and f"git@{HOST_ALIAS}:" in remote),
        },
        "agent_environment": {
            "ssh_auth_sock_present": auth_sock_present,
            "ssh_agent_pid_present": agent_pid_present,
            "private_keys_listed": False,
        },
        "advisory": [
            "Current source task does not inspect ~/.ssh or print loaded keys.",
            "If origin_uses_ovav_alias is false, migrate only after alias test passes.",
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
        ACCESS_DOC,
        INSTALL_DOC,
        SSH_TEMPLATE,
        FISH_TEMPLATE,
        POLICY,
        INSTALL_PLAN,
        VALIDATOR,
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
        "reason": "Real workstation apply is blocked until an explicit install segment is approved.",
        "would_touch_after_approval": [
            "~/.ssh/config.d/ovav-thavren.conf",
            "~/.config/fish/conf.d/ovav-thavren-ssh-agent.fish",
            "git remote origin URL",
            "dedicated SSH keypair path if user chooses generation",
        ],
        "safe_next_action": "Review plan output and approve a governed real-install step if desired.",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="OVAV Workstation Access Profile helper")
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("plan", help="Print dry-run install plan")
    sub.add_parser("diagnose", help="Run safe source/environment diagnosis")
    sub.add_parser("verify-source", help="Validate source-local artifacts")
    sub.add_parser("apply", help="Blocked placeholder for future governed real apply")
    args = parser.parse_args()

    if args.command == "plan":
        payload = build_plan()
    elif args.command == "diagnose":
        payload = diagnose()
    elif args.command == "verify-source":
        payload = verify_source()
    else:
        payload = blocked_apply()

    print(json.dumps(payload, indent=2, sort_keys=True))
    return 0 if payload["status"] in {"pass", "blocked"} else 1


if __name__ == "__main__":
    sys.exit(main())
