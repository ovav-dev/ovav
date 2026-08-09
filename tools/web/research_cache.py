#!/usr/bin/env python3
"""
Research Cache — TTL-based caching with dedup and stats.

Manages cached search results and fetched documents.
Provides cache stats, cleanup, and invalidation.

TLT defaults:
  - Search results: 24h
  - Fetched documents: 7 days
  - Papers/academic: 30 days

Usage:
  from tools.web.research_cache import get_stats, cleanup, invalidate
"""

from __future__ import annotations

import hashlib
import json
import time
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CACHE_DIR = ROOT / ".ovav" / "cache" / "research"
CACHE_DIR.mkdir(parents=True, exist_ok=True)

TTL = {
    "search": 86400,       # 24 hours
    "fetch": 604800,        # 7 days
    "paper": 2592000,      # 30 days
}


@dataclass
class CacheStats:
    total_entries: int = 0
    search_entries: int = 0
    fetch_entries: int = 0
    expired_entries: int = 0
    total_size_bytes: int = 0
    oldest_entry_age_hours: float = 0.0
    newest_entry_age_hours: float = 0.0


def get_stats() -> CacheStats:
    """Get current cache statistics."""
    stats = CacheStats()
    now = time.time()
    ages: list[float] = []

    for f in CACHE_DIR.glob("*.json"):
        if f.name.startswith("."):
            continue
        try:
            data = json.loads(f.read_text())
            ts = data.get("timestamp", 0)
            age_hours = (now - ts) / 3600
            ages.append(age_hours)

            if f.name.startswith("search_"):
                stats.search_entries += 1
                ttl = TTL["search"]
            else:
                stats.fetch_entries += 1
                ttl = TTL["fetch"]

            stats.total_entries += 1
            stats.total_size_bytes += f.stat().st_size

            if now - ts > ttl:
                stats.expired_entries += 1
        except Exception:
            pass

    if ages:
        stats.oldest_entry_age_hours = max(ages)
        stats.newest_entry_age_hours = min(ages)

    return stats


def cleanup(dry_run: bool = False) -> int:
    """Remove expired cache entries. Returns count of removed files."""
    now = time.time()
    removed = 0

    for f in CACHE_DIR.glob("*.json"):
        if f.name.startswith("."):
            continue
        try:
            data = json.loads(f.read_text())
            ts = data.get("timestamp", 0)
            prefix = "search" if f.name.startswith("search_") else "fetch"
            ttl = TTL.get(prefix, TTL["fetch"])

            if now - ts > ttl:
                if not dry_run:
                    f.unlink()
                removed += 1
        except Exception:
            pass

    return removed


def invalidate_url(url: str) -> bool:
    """Invalidate cache for a specific URL."""
    cache_key = hashlib.sha256(url.encode()).hexdigest()[:16]
    cache_file = CACHE_DIR / f"fetch_{cache_key}.json"
    if cache_file.exists():
        cache_file.unlink()
        return True
    return False


def invalidate_query(query: str) -> bool:
    """Invalidate cache for a specific search query."""
    cache_key = hashlib.sha256(query.encode()).hexdigest()[:16]
    cache_file = CACHE_DIR / f"search_{cache_key}.json"
    if cache_file.exists():
        cache_file.unlink()
        return True
    return False


def clear_all() -> int:
    """Clear entire research cache. Returns count of removed files."""
    count = 0
    for f in CACHE_DIR.glob("*.json"):
        if f.name.startswith("."):
            continue
        f.unlink()
        count += 1
    # Also clear rate state
    rate_file = CACHE_DIR / ".rate_state.json"
    if rate_file.exists():
        rate_file.unlink()
    return count


# ── CLI ──────────────────────────────────────────────────────────

if __name__ == "__main__":
    import sys
    cmd = sys.argv[1] if len(sys.argv) > 1 else "stats"

    if cmd == "stats":
        s = get_stats()
        print(f"Cache: {s.total_entries} entries ({s.expired_entries} expired)")
        print(f"  Search: {s.search_entries} | Fetch: {s.fetch_entries}")
        print(f"  Size: {s.total_size_bytes / 1024:.1f} KB")
        print(f"  Age: {s.newest_entry_age_hours:.1f}h — {s.oldest_entry_age_hours:.1f}h")

    elif cmd == "cleanup":
        n = cleanup(dry_run="--dry" in sys.argv)
        print(f"Removed {n} expired entries")

    elif cmd == "clear":
        n = clear_all()
        print(f"Cleared {n} entries")

    else:
        print("Usage: python3 tools/web/research_cache.py [stats|cleanup|clear]")
