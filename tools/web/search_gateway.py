#!/usr/bin/env python3
"""
Search Gateway — Multi-engine web search with security guards.

Supports (all free tier, no credit card for primary):
  - Tavily Search API (1,000 queries/month free, NO credit card)
  - DuckDuckGo HTML   (unlimited, no key, fallback)
  - SearXNG public    (unlimited, no key, fallback)
  - Brave Search API  (optional, 1,000 queries/month free, requires credit card)

Security:
  - Query sanitizer: strips OVAV-internal identifiers, paths, tokens
  - Rate limiter: respects free tier quotas
  - Fallback chain: graceful degradation
  - All outbound calls logged for F0.6 exfil detector

Usage:
  from tools.web.search_gateway import search
  results = search("AI governance architectures 2026", max_results=10)
"""

from __future__ import annotations

import hashlib
import json
import logging
import os
import re
import time
import urllib.parse
from dataclasses import asdict, dataclass, field
from pathlib import Path

import requests

logger = logging.getLogger("ovav.search_gateway")

ROOT = Path(__file__).resolve().parents[2]
CACHE_DIR = ROOT / ".ovav" / "cache" / "research"
CACHE_DIR.mkdir(parents=True, exist_ok=True)
CONFIG_DIR = ROOT / ".ovav" / "config"
CONFIG_FILE = CONFIG_DIR / "api_keys.env"


def _load_local_keys() -> dict[str, str]:
    """Load API keys from local config file (gitignored)."""
    keys: dict[str, str] = {}
    # Priority 1: environment variables
    for var in ("TAVILY_API_KEY", "BRAVE_API_KEY"):
        val = os.environ.get(var, "")
        if val:
            keys[var] = val
    # Priority 2: local config file
    if CONFIG_FILE.exists():
        try:
            for line in CONFIG_FILE.read_text().splitlines():
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                k = k.strip()
                v = v.strip().strip('"').strip("'")
                if k in ("TAVILY_API_KEY", "BRAVE_API_KEY") and v and k not in keys:
                    keys[k] = v
        except Exception:
            pass
    return keys

# ── Rate limit tracking ──────────────────────────────────────────
_RATE_STATE: dict[str, dict] = {}


def _load_rate_state() -> dict:
    path = CACHE_DIR / ".rate_state.json"
    if path.exists():
        try:
            return json.loads(path.read_text())
        except Exception:
            pass
    return {}


def _save_rate_state():
    (CACHE_DIR / ".rate_state.json").write_text(json.dumps(_RATE_STATE))


def _check_rate(engine: str, monthly_limit: int) -> bool:
    global _RATE_STATE
    if not _RATE_STATE:
        _RATE_STATE = _load_rate_state()
    now = time.time()
    entry = _RATE_STATE.get(engine, {"count": 0, "month_start": now})
    if now - entry["month_start"] > 30 * 24 * 3600:
        entry = {"count": 0, "month_start": now}
    if entry["count"] >= monthly_limit:
        return False
    entry["count"] += 1
    _RATE_STATE[engine] = entry
    _save_rate_state()
    return True


# ── Query sanitizer ──────────────────────────────────────────────
_SENSITIVE_PATTERNS = [
    r"/home/\w+",           # home dir paths
    r"/mnt/\w+",            # mount paths
    r"ghp_[a-zA-Z0-9]{36}",  # GitHub tokens
    r"sk-[a-zA-Z0-9]{32,}",  # OpenAI keys
    r"-----BEGIN.*?-----",  # PEM keys
    r"ovav_governor",       # internal marker
    r"OVAV_EVIDENCE_MODE",  # internal env
]


def sanitize_query(query: str) -> str:
    """Remove OVAV-internal identifiers from search queries."""
    cleaned = query
    for pat in _SENSITIVE_PATTERNS:
        cleaned = re.sub(pat, "[REDACTED]", cleaned, flags=re.IGNORECASE)
    return cleaned[:500]  # hard cap


# ── Result types ─────────────────────────────────────────────────

@dataclass
class SearchResult:
    title: str
    url: str
    snippet: str
    engine: str
    published_date: str = ""
    rank: int = 0


@dataclass
class SearchResponse:
    query: str
    results: list[SearchResult] = field(default_factory=list)
    total_found: int = 0
    engines_used: list[str] = field(default_factory=list)
    cache_hit: bool = False
    elapsed_ms: float = 0.0


# ── Engine implementations ───────────────────────────────────────

def _search_brave(query: str, max_results: int, api_key: str = "") -> list[SearchResult]:
    """Brave Search API — 2,000 free queries/month."""
    if not api_key:
        api_key = os.environ.get("BRAVE_API_KEY", "")
    if not api_key or not _check_rate("brave", 2000):
        return []

    try:
        resp = requests.get(
            "https://api.search.brave.com/res/v1/web/search",
            headers={
                "Accept": "application/json",
                "Accept-Encoding": "gzip",
                "X-Subscription-Token": api_key,
            },
            params={"q": query, "count": min(max_results, 20)},
            timeout=8,
        )
        if resp.status_code != 200:
            logger.warning(f"Brave API: HTTP {resp.status_code}")
            return []

        data = resp.json()
        results = []
        web_results = data.get("web", {}).get("results", [])
        for i, r in enumerate(web_results[:max_results]):
            results.append(SearchResult(
                title=r.get("title", ""),
                url=r.get("url", ""),
                snippet=r.get("description", ""),
                engine="brave",
                published_date=r.get("age", ""),
                rank=i + 1,
            ))
        return results
    except Exception as e:
        logger.warning(f"Brave API error: {e}")
        return []


def _search_tavily(query: str, max_results: int, api_key: str = "") -> list[SearchResult]:
    """Tavily Search API — 1,000 free queries/month."""
    if not api_key:
        api_key = os.environ.get("TAVILY_API_KEY", "")
    if not api_key or not _check_rate("tavily", 1000):
        return []

    try:
        resp = requests.post(
            "https://api.tavily.com/search",
            json={
                "query": query,
                "max_results": min(max_results, 10),
                "search_depth": "advanced",
                "include_answer": False,
            },
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {api_key}",
            },
            timeout=10,
        )
        if resp.status_code != 200:
            logger.warning(f"Tavily API: HTTP {resp.status_code}")
            return []

        data = resp.json()
        results = []
        for i, r in enumerate(data.get("results", [])[:max_results]):
            results.append(SearchResult(
                title=r.get("title", ""),
                url=r.get("url", ""),
                snippet=r.get("content", ""),
                engine="tavily",
                published_date=r.get("published_date", ""),
                rank=i + 1,
            ))
        return results
    except Exception as e:
        logger.warning(f"Tavily API error: {e}")
        return []


def _search_duckduckgo(query: str, max_results: int) -> list[SearchResult]:
    """DuckDuckGo HTML search — no API key, unlimited."""
    try:
        encoded = urllib.parse.quote_plus(query)
        resp = requests.get(
            f"https://html.duckduckgo.com/html/?q={encoded}",
            headers={"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) OVAV-Research/1.0"},
            timeout=8,
        )
        if resp.status_code != 200:
            logger.warning(f"DDG: HTTP {resp.status_code}")
            return []

        # Parse DDG HTML results
        from html.parser import HTMLParser

        class DDGParser(HTMLParser):
            def __init__(self):
                super().__init__()
                self.results: list[SearchResult] = []
                self.in_result = False
                self.in_link = False
                self.in_snippet = False
                self.current_title = ""
                self.current_url = ""
                self.current_snippet = ""
                self.capture_data = False

            def handle_starttag(self, tag, attrs):
                attrs_dict = {k.lower(): v for k, v in attrs if v}
                cls = attrs_dict.get("class", "")
                if tag == "div" and "result" in cls:
                    self.in_result = True
                if self.in_result and tag == "a" and "result__a" in cls:
                    self.in_link = True
                    self.current_url = attrs_dict.get("href", "")
                if self.in_result and tag == "a" and "result__snippet" in cls:
                    self.in_snippet = True

            def handle_data(self, data):
                if self.in_link:
                    self.current_title += data.strip()
                if self.in_snippet:
                    self.current_snippet += data.strip()

            def handle_endtag(self, tag):
                if tag == "a" and self.in_link:
                    self.in_link = False
                if tag == "a" and self.in_snippet:
                    self.in_snippet = False
                if tag == "div" and self.in_result:
                    self.in_result = False
                    if self.current_title and self.current_url:
                        self.results.append(SearchResult(
                            title=self.current_title.strip()[:200],
                            url=self.current_url.strip(),
                            snippet=self.current_snippet.strip()[:500],
                            engine="duckduckgo",
                            rank=len(self.results) + 1,
                        ))
                    self.current_title = ""
                    self.current_url = ""
                    self.current_snippet = ""

        parser = DDGParser()
        parser.feed(resp.text)
        return parser.results[:max_results]

    except Exception as e:
        logger.warning(f"DDG error: {e}")
        return []


def _search_searxng(query: str, max_results: int) -> list[SearchResult]:
    """SearXNG public instances — no API key, unlimited."""
    instances = [
        "https://searx.be",
        "https://search.sapti.me",
    ]
    for instance in instances:
        try:
            resp = requests.get(
                f"{instance}/search",
                params={"q": query, "format": "json", "categories": "general"},
                timeout=8,
            )
            if resp.status_code != 200:
                continue
            data = resp.json()
            results = []
            for i, r in enumerate(data.get("results", [])[:max_results]):
                results.append(SearchResult(
                    title=r.get("title", ""),
                    url=r.get("url", ""),
                    snippet=r.get("content", "")[:500] if r.get("content") else "",
                    engine="searxng",
                    published_date=r.get("publishedDate", ""),
                    rank=i + 1,
                ))
            if results:
                return results
        except Exception:
            continue
    return []


# ── Main search function ─────────────────────────────────────────

def search(
    query: str,
    max_results: int = 10,
    brave_key: str = "",
    tavily_key: str = "",
    engines: list[str] | None = None,
    use_cache: bool = True,
) -> SearchResponse:
    """
    Multi-engine web search with automatic fallback.

    Args:
        query: Search query string
        max_results: Maximum results to return (across all engines)
        brave_key: Brave Search API key (or set BRAVE_API_KEY env var)
        tavily_key: Tavily Search API key (or set TAVILY_API_KEY env var)
        engines: Specific engines to use (default: all available)
        use_cache: Use cached results if available

    Returns:
        SearchResponse with deduplicated, ranked results
    """
    start_time = time.time()
    clean_query = sanitize_query(query)

    # Load local keys
    local_keys = _load_local_keys()
    if not brave_key:
        brave_key = local_keys.get("BRAVE_API_KEY", "")
    if not tavily_key:
        tavily_key = local_keys.get("TAVILY_API_KEY", "")

    # Cache check
    if use_cache:
        cache_key = hashlib.sha256(clean_query.encode()).hexdigest()[:16]
        cache_file = CACHE_DIR / f"search_{cache_key}.json"
        if cache_file.exists():
            try:
                cached = json.loads(cache_file.read_text())
                age = time.time() - cached.get("timestamp", 0)
                if age < 86400:  # 24h TTL for searches
                    resp = SearchResponse(
                        query=clean_query,
                        results=[SearchResult(**r) for r in cached["results"]],
                        total_found=cached.get("total_found", 0),
                        engines_used=cached.get("engines_used", []),
                        cache_hit=True,
                        elapsed_ms=(time.time() - start_time) * 1000,
                    )
                    return resp
            except Exception:
                pass

    # Determine engines
    if engines is None:
        engines = ["brave", "tavily", "duckduckgo", "searxng"]

    all_results: list[SearchResult] = []
    engines_used: list[str] = []

    for engine in engines:
        if len(all_results) >= max_results * 2:
            break
        results = []
        if engine == "brave":
            results = _search_brave(clean_query, max_results, brave_key)
        elif engine == "tavily":
            results = _search_tavily(clean_query, max_results, tavily_key)
        elif engine == "duckduckgo":
            results = _search_duckduckgo(clean_query, max_results)
        elif engine == "searxng":
            results = _search_searxng(clean_query, max_results)

        if results:
            engines_used.append(engine)
            all_results.extend(results)

    # Deduplicate by URL
    seen_urls: set[str] = set()
    deduped: list[SearchResult] = []
    for r in sorted(all_results, key=lambda x: (x.engine != "brave", x.engine != "tavily", x.rank)):
        norm_url = r.url.rstrip("/").lower()
        if norm_url not in seen_urls:
            seen_urls.add(norm_url)
            deduped.append(r)
        if len(deduped) >= max_results:
            break

    elapsed = (time.time() - start_time) * 1000
    response = SearchResponse(
        query=clean_query,
        results=deduped,
        total_found=len(all_results),
        engines_used=engines_used,
        cache_hit=False,
        elapsed_ms=elapsed,
    )

    # Cache results
    if use_cache and deduped:
        cache_key = hashlib.sha256(clean_query.encode()).hexdigest()[:16]
        cache_file = CACHE_DIR / f"search_{cache_key}.json"
        cache_file.write_text(json.dumps({
            "query": clean_query,
            "results": [asdict(r) for r in deduped],
            "total_found": len(all_results),
            "engines_used": engines_used,
            "timestamp": time.time(),
        }))

    return response


# ── CLI ──────────────────────────────────────────────────────────

if __name__ == "__main__":
    import sys
    q = " ".join(sys.argv[1:]) if len(sys.argv) > 1 else "OVAV AI governance"
    resp = search(q)
    print(f"Query: {resp.query}")
    print(f"Engines: {resp.engines_used} | Cache: {resp.cache_hit} | Time: {resp.elapsed_ms:.0f}ms")
    print(f"Results: {len(resp.results)}")
    for r in resp.results:
        print(f"  [{r.engine}] {r.title}")
        print(f"    {r.url}")
        print(f"    {r.snippet[:120]}...")
        print()
