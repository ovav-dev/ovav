#!/usr/bin/env python3
"""
ovav_git_push_gate - OVAV Pre-Push Safety Gate
===============================================

Checks safety before git push.
Integrates with OMARS HygieneMonitor and GitFlow.

Usage:
    python3 tools/github/ovav_git_push_gate.py [--confirm]
    python3 tools/github/ovav_git_push_gate.py --check-protected

Exit codes:
    0 = safe to push
    1 = blocked / needs confirmation
"""

import os
import sys
import json
import subprocess
import argparse
from pathlib import Path

REPO_ROOT = Path(__file__).parent.parent.parent.resolve()
PROTECTED_BRANCHES = {"main", "master", "production", "staging", "develop"}


def get_current_branch():
    """Get current git branch."""
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            cwd=REPO_ROOT, capture_output=True, text=True, timeout=5
        )
        return result.stdout.strip()
    except Exception:
        return "unknown"


def check_branch_safety():
    """Check if current branch is protected or has issues."""
    branch = get_current_branch()
    
    if branch in PROTECTED_BRANCHES:
        # Check for waiver
        waiver_path = REPO_ROOT / ".ovav" / "governance" / "waivers" / f"{branch}.yaml"
        if not waiver_path.exists():
            return False, f"Protected branch '{branch}' has no active waiver"
    
    return True, f"Branch '{branch}' is safe to push"


def check_uncommitted():
    """Check for uncommitted changes."""
    try:
        result = subprocess.run(
            ["git", "status", "--porcelain"],
            cwd=REPO_ROOT, capture_output=True, text=True, timeout=5
        )
        if result.stdout.strip():
            return False, "Uncommitted changes exist"
    except Exception as e:
        return False, f"Git check failed: {e}"
    
    return True, "No uncommitted changes"


def check_locks():
    """Check for stale lock files."""
    locks_dir = REPO_ROOT / ".ovav" / "locks"
    if not locks_dir.exists():
        return True, "No locks directory"
    
    try:
        locks = list(locks_dir.glob("*.lock"))
        if locks:
            stale = []
            for lock in locks:
                age = os.path.getmtime(lock)
                # Check if older than 1 hour
                if os.path.getmtime(lock) < os.time() - 3600:
                    stale.append(lock.name)
            
            if stale:
                return False, f"Stale locks: {', '.join(stale[:3])}"
    except Exception:
        pass
    
    return True, "No stale locks"


def check(args):
    """Main check - exit early with status."""
    issues = []
    
    # Branch safety
    safe, msg = check_branch_safety()
    if not safe:
        issues.append(f"BLOCK: {msg}")
    else:
        print(f"   ✅ {msg}")
    
    # Uncommitted changes
    safe, msg = check_uncommitted()
    if not safe:
        issues.append(f"BLOCK: {msg}")
    else:
        print(f"   ✅ {msg}")
    
    # Lock check
    safe, msg = check_locks()
    if not safe:
        issues.append(f"WARN: {msg}")
    else:
        print(f"   ✅ {msg}")
    
    if issues:
        has_block = any("BLOCK" in i for i in issues)
        if has_block:
            print("\n".join(issues))
            return 1
    
    print("   ✅ All checks passed - safe to push")
    return 0


def confirm_and_push():
    """Interactive push with confirmation."""
    print("⚠️  Git push confirmation required")
    print("   Type 'yes' to confirm: ", end="")
    
    try:
        response = input().strip().lower()
        if response == "yes":
            print("   ✅ Push confirmed")
            return 0
        else:
            print("   ❌ Push cancelled")
            return 1
    except EOFError:
        print("   ❌ Non-interactive terminal")
        return 1


def main():
    parser = argparse.ArgumentParser(description="OVAV Git Push Gate")
    parser.add_argument("--confirm", action="store_true", help="Skip confirmation")
    parser.add_argument("--check-protected", action="store_true", help="Check protected branches only")
    parser.add_argument("--json", action="store_true", help="JSON output")
    
    args = parser.parse_args()
    
    print("🔒 OVAV Git Push Gate")
    print("=" * 40)
    
    # Check protected branches
    if args.check_protected:
        safe, msg = check_branch_safety()
        print(f"   {'✅' if safe else '❌'} {msg}")
        return 0 if safe else 1
    
    # Run checks
    exit_code = check(args)
    
    if exit_code == 0:
        if args.confirm:
            return 0
        return confirm_and_push()
    
    if args.json:
        print(json.dumps({"status": "blocked", "code": exit_code}))
    
    return exit_code


if __name__ == "__main__":
    sys.exit(main())