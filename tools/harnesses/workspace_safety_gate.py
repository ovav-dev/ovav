#!/usr/bin/env python3
"""
workspace_safety_gate - OVAV Pre-Write Safety Check
====================================================

Checks workspace safety before any write operation.
Integrates with OMARS via HygieneMonitor.

Usage:
    python3 tools/harnesses/workspace_safety_gate.py --mode mutate [--path <file>]
    python3 tools/harnesses/workspace_safety_gate.py --mode check

Exit codes:
    0 = safe to write
    1 = unsafe / blocked
    2 = warning only
"""

import os
import sys
import json
import argparse
from pathlib import Path

REPO_ROOT = Path(__file__).parent.parent.parent.resolve()
MODE = ""


def check_git_status():
    """Check for uncommitted changes in main directories."""
    dangerous_dirs = ["go-runtime/cmd", "tools/", ".ovav/"]
    
    for d in dangerous_dirs:
        dir_path = REPO_ROOT / d
        if dir_path.exists():
            git_status = os.popen(f"cd {REPO_ROOT} && git status --porcelain {d} 2>/dev/null").read()
            if git_status.strip():
                return False, f"Uncommitted changes in {d}"
    
    return True, "No uncommitted changes in critical dirs"


def check_protected_paths(path):
    """Check if path is in protected list."""
    protected = [
        ".git/objects",
        ".git/refs",
        "go-runtime/internal/vault/secrets",
        ".ovav/vault/tokens",
    ]
    
    path_str = str(path)
    for p in protected:
        if p in path_str:
            return False, f"Protected path: {p}"
    
    return True, "Path not protected"


def check_file_size(path):
    """Check if file is too large (potential binary)."""
    try:
        size = os.path.getsize(path)
        max_size = 10 * 1024 * 1024  # 10MB
        
        if size > max_size:
            return False, f"File too large ({size} bytes)"
    except OSError:
        pass
    
    return True, "File size OK"


def check_mode_mutate(args):
    """Pre-write check - is it safe to modify files?"""
    issues = []
    
    # Check git status
    safe, msg = check_git_status()
    if not safe:
        issues.append(f"WARN: {msg}")
    
    # Check specific path if provided
    if args.path:
        path = Path(args.path)
        
        safe, msg = check_protected_paths(path)
        if not safe:
            issues.append(f"BLOCK: {msg}")
        
        if path.exists():
            safe, msg = check_file_size(path)
            if not safe:
                issues.append(f"WARN: {msg}")
    
    if issues:
        print("\n".join(issues))
        # Check for BLOCK level issues
        has_block = any("BLOCK" in i for i in issues)
        return 1 if has_block else 2
    
    print("✅ Workspace safety check passed")
    return 0


def check_mode_check(args):
    """Passive check - show current safety status."""
    print("🔍 OVAV Workspace Safety Check")
    print("=" * 40)
    
    safe, msg = check_git_status()
    status = "✅" if safe else "⚠️"
    print(f"   {status} {msg}")
    
    # Check for broken symlinks
    result = os.popen(f"find {REPO_ROOT} -type l ! -exec test -e {{}} \\; -print 2>/dev/null | head -10").read()
    if result.strip():
        print(f"   ⚠️  Found broken symlinks")
        for link in result.strip().split("\n")[:5]:
            print(f"      - {link}")
    
    return 0


def main():
    parser = argparse.ArgumentParser(description="OVAV Workspace Safety Gate")
    parser.add_argument("--mode", choices=["mutate", "check"], default="check",
                        help="Check mode: mutate (pre-write) or check (passive)")
    parser.add_argument("--path", help="File path to check")
    parser.add_argument("--json", action="store_true", help="JSON output")
    
    args = parser.parse_args()
    
    if args.mode == "mutate":
        exit_code = check_mode_mutate(args)
    else:
        exit_code = check_mode_check(args)
    
    if args.json:
        print(json.dumps({"status": "ok" if exit_code == 0 else "warning", "code": exit_code}))
    
    sys.exit(exit_code)


if __name__ == "__main__":
    main()