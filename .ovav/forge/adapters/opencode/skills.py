"""OVAV Forge — OpenCode Skills Adapter.

Discovers and projects skills from canonical source (.ovav/source/skills/)
to OpenCode skill registry.

Usage:
    from .ovav.forge.adapters.opencode.skills import discover
    inventory = discover()
"""

import re
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[4]
SKILL_FILE = "SKILL.md"
SCAN_PATHS: list[Path] = [
    REPO_ROOT / ".ovav" / "source" / "skills",
    REPO_ROOT / ".opencode" / "skills",
    REPO_ROOT / ".agents" / "skills",
]


def _parse_skill_md(path: Path) -> dict[str, Any]:
    """Extract metadata from a SKILL.md file."""
    try:
        content = path.read_text()
    except OSError:
        return {"name": path.parent.name, "status": "unreadable"}

    skill: dict[str, Any] = {
        "path": str(path.relative_to(REPO_ROOT)),
        "directory": path.parent.name,
    }

    # Try YAML frontmatter
    fm_match = re.match(r'^---\s*\n(.*?)\n---', content, re.DOTALL)
    if fm_match:
        try:
            import yaml
            fm = yaml.safe_load(fm_match.group(1))
            if isinstance(fm, dict):
                skill.update(fm)
        except Exception:
            pass

    if "name" not in skill:
        name_match = re.search(r'^#\s+(.+)$', content, re.MULTILINE)
        if name_match:
            skill["name"] = name_match.group(1).strip()

    if "description" not in skill:
        desc_match = re.search(
            r'(?:description|purpose|Use when)\s*:\s*(.+)$',
            content, re.MULTILINE | re.IGNORECASE,
        )
        if desc_match:
            skill["description"] = desc_match.group(1).strip()

    if "owner_profile" not in skill:
        owner_match = re.search(r'owner_profile\s*:\s*(\S+)', content, re.MULTILINE)
        if owner_match:
            skill["owner_profile"] = owner_match.group(1)

    if "owner_lane" not in skill:
        lane_match = re.search(r'(?:owner_lane|lane)\s*:\s*(\S+)', content, re.MULTILINE)
        if lane_match:
            skill["owner_lane"] = lane_match.group(1)

    has_memory = bool(re.search(r'memory|persist', content, re.IGNORECASE))  # v52.0: system deleted
    has_risk = bool(re.search(r'risk|danger|warning|blocked', content, re.IGNORECASE))
    has_permission = bool(re.search(r'permission|authority|grant', content, re.IGNORECASE))

    skill["flags"] = {
        "mentions_memory": has_memory,
        "mentions_risk": has_risk,
        "mentions_permission": has_permission,
    }

    if "name" not in skill:
        skill["name"] = path.parent.name

    return skill


def discover() -> dict[str, Any]:
    """Discover all skills from configured scan paths."""
    inventory: dict[str, dict[str, Any]] = {}
    sources_scanned: list[str] = []
    errors: list[str] = []

    for scan_path in SCAN_PATHS:
        if not scan_path.exists():
            continue
        sources_scanned.append(str(scan_path.relative_to(REPO_ROOT)))

        for skill_dir in sorted(scan_path.iterdir()):
            if not skill_dir.is_dir():
                continue
            skill_file = skill_dir / SKILL_FILE
            if not skill_file.exists():
                continue

            try:
                skill = _parse_skill_md(skill_file)
                name = skill.get("name", skill_dir.name)
                if name in inventory:
                    continue
                inventory[name] = skill
            except Exception as exc:
                errors.append(f"{skill_dir.name}: {exc}")

    return {
        "skills": inventory,
        "total": len(inventory),
        "sources_scanned": sources_scanned,
        "errors": errors,
    }
