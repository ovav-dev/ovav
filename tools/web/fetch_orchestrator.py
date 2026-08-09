#!/usr/bin/env python3
"""
Fetch Orchestrator — Parallel multi-URL fetch with content extraction.

Fetches up to 20 URLs simultaneously using ThreadPool.
Each result includes extracted text, title, date, and word count.

Usage:
  from tools.web.fetch_orchestrator import fetch_urls
  docs = fetch_urls(urls, max_workers=10, timeout=8)
"""

from __future__ import annotations

import hashlib
import json
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass, field
from pathlib import Path

import requests

from tools.web.content_extractor import extract

ROOT = Path(__file__).resolve().parents[2]
CACHE_DIR = ROOT / ".ovav" / "cache" / "research"
CACHE_DIR.mkdir(parents=True, exist_ok=True)

SESSION = requests.Session()
SESSION.headers.update({
    "User-Agent": "Mozilla/5.0 (X11; Linux x86_64) OVAV-Research/1.0",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.9,es;q=0.8",
})


@dataclass
class FetchResult:
    url: str
    status_code: int = 0
    error: str = ""
    title: str = ""
    text: str = ""
    text_preview: str = ""
    publish_date: str = ""
    author: str = ""
    word_count: int = 0
    fetch_time_ms: float = 0.0
    cache_hit: bool = False


@dataclass
class FetchBatch:
    results: list[FetchResult] = field(default_factory=list)
    total_urls: int = 0
    success_count: int = 0
    error_count: int = 0
    total_words: int = 0
    elapsed_ms: float = 0.0


def _fetch_one(url: str, timeout: int, use_cache: bool, max_text_chars: int) -> FetchResult:
    """Fetch a single URL with caching."""
    start = time.time()

    # Cache check
    cache_key = hashlib.sha256(url.encode()).hexdigest()[:16]
    cache_file = CACHE_DIR / f"fetch_{cache_key}.json"
    if use_cache and cache_file.exists():
        try:
            cached = json.loads(cache_file.read_text())
            age = time.time() - cached.get("timestamp", 0)
            ttl = 7 * 86400  # 7 days for fetched content
            if age < ttl:
                return FetchResult(
                    url=url,
                    status_code=200,
                    title=cached.get("title", ""),
                    text=cached.get("text", ""),
                    text_preview=cached.get("text", "")[:300],
                    publish_date=cached.get("publish_date", ""),
                    author=cached.get("author", ""),
                    word_count=cached.get("word_count", 0),
                    fetch_time_ms=(time.time() - start) * 1000,
                    cache_hit=True,
                )
        except Exception:
            pass

    try:
        resp = SESSION.get(url, timeout=timeout, allow_redirects=True)
        fetch_time = (time.time() - start) * 1000

        if resp.status_code != 200:
            return FetchResult(
                url=url,
                status_code=resp.status_code,
                error=f"HTTP {resp.status_code}",
                fetch_time_ms=fetch_time,
            )

        content_type = resp.headers.get("Content-Type", "")
        if "text/html" not in content_type and "text/plain" not in content_type:
            return FetchResult(
                url=url,
                status_code=resp.status_code,
                error=f"Non-text content: {content_type[:50]}",
                fetch_time_ms=fetch_time,
            )

        extracted = extract(resp.text, url=url)
        text = extracted.text[:max_text_chars]

        result = FetchResult(
            url=url,
            status_code=resp.status_code,
            title=extracted.title,
            text=text,
            text_preview=text[:300],
            publish_date=extracted.publish_date,
            author=extracted.author,
            word_count=extracted.word_count,
            fetch_time_ms=fetch_time,
            cache_hit=False,
        )

        # Cache result
        if use_cache:
            cache_file.write_text(json.dumps({
                "url": url,
                "title": extracted.title,
                "text": text,
                "publish_date": extracted.publish_date,
                "author": extracted.author,
                "word_count": extracted.word_count,
                "timestamp": time.time(),
            }))

        return result

    except requests.Timeout:
        return FetchResult(url=url, error="Timeout", fetch_time_ms=(time.time() - start) * 1000)
    except requests.ConnectionError:
        return FetchResult(url=url, error="Connection failed", fetch_time_ms=(time.time() - start) * 1000)
    except Exception as e:
        return FetchResult(url=url, error=str(e)[:200], fetch_time_ms=(time.time() - start) * 1000)


def fetch_urls(
    urls: list[str],
    max_workers: int = 10,
    timeout: int = 8,
    use_cache: bool = True,
    max_text_chars: int = 50_000,
) -> FetchBatch:
    """
    Fetch multiple URLs in parallel.

    Args:
        urls: List of URLs to fetch
        max_workers: Maximum concurrent fetches
        timeout: Per-URL timeout in seconds
        use_cache: Use cached results if available
        max_text_chars: Maximum characters per extracted text

    Returns:
        FetchBatch with all results
    """
    start = time.time()
    results: list[FetchResult] = []
    unique_urls = list(dict.fromkeys(urls))  # dedup preserving order

    with ThreadPoolExecutor(max_workers=min(max_workers, len(unique_urls) or 1)) as executor:
        futures = {
            executor.submit(_fetch_one, url, timeout, use_cache, max_text_chars): url
            for url in unique_urls
        }
        for future in as_completed(futures):
            try:
                result = future.result()
                results.append(result)
            except Exception as e:
                url = futures[future]
                results.append(FetchResult(url=url, error=str(e)[:200]))

    # Sort by original order
    url_order = {url: i for i, url in enumerate(unique_urls)}
    results.sort(key=lambda r: url_order.get(r.url, 999))

    success = [r for r in results if r.status_code == 200 and r.text]
    errors = [r for r in results if r.error or r.status_code != 200]

    return FetchBatch(
        results=results,
        total_urls=len(unique_urls),
        success_count=len(success),
        error_count=len(errors),
        total_words=sum(r.word_count for r in success),
        elapsed_ms=(time.time() - start) * 1000,
    )


# ── CLI ──────────────────────────────────────────────────────────

if __name__ == "__main__":
    import sys
    urls = [line.strip() for line in sys.stdin if line.strip().startswith("http")]
    if not urls:
        print("Usage: echo 'https://...' | python3 tools/web/fetch_orchestrator.py")
        sys.exit(1)
    batch = fetch_urls(urls)
    print(f"Fetched {batch.total_urls} URLs in {batch.elapsed_ms:.0f}ms")
    print(f"  Success: {batch.success_count} | Errors: {batch.error_count}")
    print(f"  Total words: {batch.total_words:,}")
    for r in batch.results:
        status = "✓" if r.status_code == 200 else "✗"
        print(f"  {status} [{r.fetch_time_ms:.0f}ms] {r.title[:80] or r.url[:80]}")
