"""OVAV Research Mesh — Web Research Infrastructure.

Layers:
  1. search_gateway  — Multi-engine search (Brave, Tavily, DDG, SearXNG)
  2. fetch_orchestrator — Parallel multi-URL fetch with content extraction
  3. content_extractor — HTML → clean text (stdlib only)
  4. research_cache — TTL-based caching with dedup

Security:
  - Query sanitizer (strips internal OVAV context before external calls)
  - Exfil guard (F0.6 monitors every outbound query)
  - Rate limiter (respects free tier limits)
  - Fallback chain (graceful degradation)

Usage:
  from tools.web.search_gateway import search
  results = search("latest AI governance architectures 2026", max_results=10)
"""

__version__ = "1.0.0"
