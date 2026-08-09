"""OVAV Forge — OpenCode Agents Adapter.

Projects agents from canonical source (.ovav/source/agents/) to
OpenCode flat format (.opencode/agents/).

Usage:
    from .ovav.forge.adapters.opencode.agents import project
    cleaned, created = project()
"""

import shutil
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]
SOURCE = REPO_ROOT / ".ovav" / "source" / "agents"
TARGET = REPO_ROOT / ".opencode" / "agents"

PREFIX = {"areas": "area", "leads": "lead", "teams": "team"}


def project() -> tuple[int, int]:
    """Project agents from OVAV source to OpenCode target. Returns (cleaned, created)."""
    cleaned = 0
    created = 0

    for pfx in PREFIX.values():
        for f in TARGET.glob(f"{pfx}-*.md"):
            f.unlink()
            cleaned += 1

    for src in sorted(SOURCE.rglob("*.md")):
        rel = src.relative_to(SOURCE)
        category = rel.parts[0]
        if category not in PREFIX:
            continue
        dst = TARGET / f"{PREFIX[category]}-{rel.parts[-1]}"
        shutil.copy2(src, dst)
        created += 1

    return cleaned, created
