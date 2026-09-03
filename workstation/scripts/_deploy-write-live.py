#!/usr/bin/env python3
"""
Helper for deploy-it-keybindings.sh: write merged JSON directly to
the WSL/DrvFS target.

Background:
  In WSL, both `mv` and `cp -f` from /tmp (Linux tmpfs) to /mnt/c/...
  (Windows DrvFS) silently fail to update the destination file in some
  cases — the destination appears to keep stale content even though
  the operation exits 0. Verified 2026-08-14:
    - mv TMP LIVE: destination keeps stale content ❌
    - cp -f TMP LIVE: destination keeps stale content ❌
    - python open(LIVE, 'w'): destination updates correctly ✅

  This helper uses python's open() to bypass the WSL bug.

Usage:
  deploy-write-live.py <tmp_merged> <live_destination>
"""
import json
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} <tmp_merged> <live_destination>", file=sys.stderr)
        return 2

    tmp_path = Path(sys.argv[1])
    live_path = Path(sys.argv[2])

    if not tmp_path.exists():
        print(f"ERROR: TMP file does not exist: {tmp_path}", file=sys.stderr)
        return 3

    try:
        with tmp_path.open() as f:
            merged = json.load(f)
        n_merged = len(merged.get("keybindings", []))
    except Exception as exc:
        print(f"ERROR: cannot read TMP JSON: {exc}", file=sys.stderr)
        return 4

    try:
        # Use a temp file in the SAME directory as live, then rename.
        # This is the canonical atomic-replace pattern — but we need to
        # avoid the cross-FS mv bug. Solution: write to a sibling temp
        # in the destination directory, then rename within the same FS.
        sibling = live_path.with_suffix(live_path.suffix + ".tmp")
        with sibling.open("w") as f:
            json.dump(merged, f, indent=1)
            f.write("\n")
            f.flush()
        # Same-FS rename — should work even on WSL DrvFS
        sibling.replace(live_path)
    except Exception as exc:
        print(f"ERROR: cannot write LIVE: {exc}", file=sys.stderr)
        return 5

    # Verify the write actually landed
    try:
        with live_path.open() as f:
            check = json.load(f)
        n_check = len(check.get("keybindings", []))
    except Exception as exc:
        print(f"ERROR: cannot read back LIVE: {exc}", file=sys.stderr)
        return 6

    if n_check != n_merged:
        print(
            f"FATAL: LIVE has {n_check} keybindings but merged had "
            f"{n_merged} — write did not land",
            file=sys.stderr,
        )
        return 7

    print(f"wrote {n_check} keybindings to {live_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
