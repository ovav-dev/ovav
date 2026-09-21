#!/usr/bin/env python3
"""
KC P∞ — Unified Knowledge Compiler
Compiles millions of reasoned data points to ~10KB.
Principles replace facts. Weights replace logs. Connections replace history.

Single unified store consumed by both Thavren and OVAV as views.

Usage:
    python3 tools/knowledge/compiler.py --compile          # Full recompile
    python3 tools/knowledge/compiler.py --ingest FILE.json # Ingest new data
    python3 tools/knowledge/compiler.py --view thavren     # Thavren's view
    python3 tools/knowledge/compiler.py --view ovav        # OVAV's view
    python3 tools/knowledge/compiler.py --stats            # Compression stats
"""

import hashlib
import json
import os
import time
from pathlib import Path

# ── Paths ──────────────────────────────────────────────────────────
ROOT = Path(__file__).resolve().parent.parent.parent
STORE_PATH = ROOT / ".ovav" / "knowledge" / "store.kc"
SYMBOLS_PATH = ROOT / ".ovav" / "knowledge" / "symbols.json"
INGEST_DIR = ROOT / ".ovav" / "knowledge" / "ingest"
STORE_PATH.parent.mkdir(parents=True, exist_ok=True)
INGEST_DIR.mkdir(parents=True, exist_ok=True)

# ── Symbol Table ───────────────────────────────────────────────────
# Maps common terms to 1-byte IDs (0-255). 256 slots total.
# Reserved: 0=NULL, 1-31=domains, 32-255=terms
RESERVED_SYMBOLS = {
    0: "__NULL__",
    1: "security",
    2: "architecture",
    3: "continuity",
    4: "quality",
    5: "delivery",
    6: "relationship",
    7: "governance",
    8: "learning",
    9: "identity",
    10: "ambition",
    11: "compression",
    12: "validation",
    13: "integrity",
    14: "health",
    15: "creator",
    16: "ovav",
    17: "thavren",
    18: "eidren",
    19: "session",
    20: "commit",
    21: "pattern",
    22: "criterion",
    23: "milestone",
    24: "decision",
    25: "error",
    26: "fix",
    27: "feature",
    28: "refactor",
    29: "critical",
    30: "high",
    31: "medium",
}

class SymbolTable:
    """1-byte ID ↔ term mapping. 256 slots. Auto-expands but warns at 200+."""

    def __init__(self):
        self.id_to_term: dict[int, str] = dict(RESERVED_SYMBOLS)
        self.term_to_id: dict[str, int] = {v: k for k, v in RESERVED_SYMBOLS.items()}
        self._next_id = max(RESERVED_SYMBOLS.keys()) + 1 if RESERVED_SYMBOLS else 32

    def encode(self, term: str) -> int:
        """Term → 1-byte ID. Auto-registers new terms."""
        term = term.lower().strip()[:64]  # max 64 chars
        if term in self.term_to_id:
            return self.term_to_id[term]
        if self._next_id >= 256:
            # Fallback: hash to existing slot (lossy but bounded)
            return hash(term) % 256
        sid = self._next_id
        self._next_id += 1
        self.id_to_term[sid] = term
        self.term_to_id[term] = sid
        return sid

    def decode(self, sid: int) -> str:
        return self.id_to_term.get(sid, f"__UNK_{sid}__")

    def to_dict(self) -> dict:
        return {str(k): v for k, v in self.id_to_term.items()}

    @classmethod
    def from_dict(cls, d: dict) -> "SymbolTable":
        st = cls()
        st.id_to_term = {int(k): v for k, v in d.items()}
        st.term_to_id = {v: int(k) for k, v in d.items()}
        st._next_id = max(st.id_to_term.keys()) + 1 if st.id_to_term else 32
        return st

    def save(self):
        with open(SYMBOLS_PATH, "w") as f:
            json.dump(self.to_dict(), f, separators=(",", ":"))

    @classmethod
    def load(cls) -> "SymbolTable":
        if SYMBOLS_PATH.exists():
            with open(SYMBOLS_PATH) as f:
                return cls.from_dict(json.load(f))
        return cls()


# ── Knowledge Entities ─────────────────────────────────────────────

class Principle:
    """A compiled insight: criterion, pattern, or distilled knowledge."""
    __slots__ = ("connections", "created_at", "domain", "evidence_hash", "last_reinforced", "sid", "weight")

    def __init__(self, symbol_id: int, weight: int = 128, connections: list[int] | None = None,
                 evidence: str = "", domain: int = 0):
        self.sid = symbol_id
        self.weight = weight  # 0-255 (uint8)
        self.connections = connections or []
        self.evidence_hash = hashlib.sha256(evidence.encode()).hexdigest()[:8] if evidence else ""
        self.domain = domain
        self.created_at = int(time.time())
        self.last_reinforced = self.created_at

    def reinforce(self, amount: int = 13):
        """Strengthen weight (13 ≈ +0.05 in float)."""
        self.weight = min(255, self.weight + amount)
        self.last_reinforced = int(time.time())

    def decay(self, days: float) -> bool:
        """Time decay. Returns True if principle should be pruned."""
        decay_amount = int(days * 2.5)  # ~1% per day
        self.weight = max(0, self.weight - decay_amount)
        return self.weight < 3  # Below ~1% → prune

    def to_compact(self) -> list:
        """Compact representation: [sid, weight, [conn_ids], evidence_hash_8, domain]"""
        return [self.sid, self.weight, self.connections, self.evidence_hash, self.domain]

    @classmethod
    def from_compact(cls, data: list, created_at: int = 0) -> "Principle":
        p = cls(data[0], data[1], data[2] if len(data) > 2 else [],
                data[3] if len(data) > 3 else "",
                data[4] if len(data) > 4 else 0)
        if created_at:
            p.created_at = created_at
            p.last_reinforced = created_at
        return p


class Connection:
    """Weighted relationship between two principles."""
    __slots__ = ("conn_type", "from_id", "to_id", "weight")

    def __init__(self, from_id: int, to_id: int, weight: int = 128, conn_type: int = 0):
        self.from_id = from_id
        self.to_id = to_id
        self.weight = weight
        self.conn_type = conn_type  # 0=related, 1=depends_on, 2=contradicts, 3=evolved_from

    def to_compact(self) -> list:
        return [self.from_id, self.to_id, self.weight, self.conn_type]

    @classmethod
    def from_compact(cls, data: list) -> "Connection":
        return cls(data[0], data[1], data[2], data[3] if len(data) > 3 else 0)


# ── Core Compiler ──────────────────────────────────────────────────

class KnowledgeCompiler:
    """KC P∞ — compiles raw inputs to extremely compact knowledge store."""

    def __init__(self):
        self.symbols = SymbolTable.load()
        self.principles: dict[int, Principle] = {}  # sid → Principle
        self.connections: list[Connection] = []
        self.total_compiled: int = 0
        self.last_compile: int = 0
        self._load_store()

    def _load_store(self):
        """Load existing compact store if it exists."""
        # Auto-create domain principles (reserved IDs 1-15)
        for sid in range(1, 16):
            if sid in RESERVED_SYMBOLS and sid not in self.principles:
                domain_name = RESERVED_SYMBOLS[sid]
                self.principles[sid] = Principle(
                    symbol_id=sid,
                    weight=50,  # Start weak, strengthen with use
                    domain=sid,
                )

        if not STORE_PATH.exists():
            return
        try:
            with open(STORE_PATH, "rb") as f:
                raw = f.read()
            # Verify integrity
            stored_hash = raw[:32]
            content = raw[32:]
            if hashlib.sha256(content).digest() != stored_hash:
                print("[KC] WARNING: store integrity check failed, starting fresh")
                return
            data = json.loads(content.decode("utf-8"))
            self.total_compiled = data.get("m", {}).get("c", 0)
            self.last_compile = data.get("t", {}).get("last_compile", 0)
            # Restore symbols
            if "s" in data:
                loaded_symbols = SymbolTable.from_dict(data["s"])
                # Merge: keep our reserved auto-created principles, use loaded for everything else
                self.symbols = loaded_symbols
            # Restore principles (overwrites auto-created if they exist in store)
            ts = data.get("t", {}).get("created", int(time.time()))
            for p_data in data.get("p", []):
                p = Principle.from_compact(p_data, ts)
                self.principles[p.sid] = p
            # Restore connections
            for c_data in data.get("x", []):
                self.connections.append(Connection.from_compact(c_data))
        except Exception as e:
            print(f"[KC] WARNING: store load failed: {e}")

    def ingest(self, source: str, event_type: str, payload: dict) -> int:
        """
        Ingest a single knowledge event. Returns classification:
        0=noise, 1=fact(stored), 2=pattern(abstracted), 3=principle(compiled)
        """
        # Classify
        classification = self._classify(source, event_type, payload)
        if classification == 0:
            return 0  # noise

        # Normalize → find or create symbol
        key_terms = self._extract_terms(event_type, payload)
        term_ids = [self.symbols.encode(t) for t in key_terms]
        domain = self._classify_domain(term_ids, payload)

        # Abstract: check if similar principle exists
        primary_term = term_ids[0] if term_ids else self.symbols.encode(event_type)
        existing = self._find_similar(term_ids, domain)
        if existing:
            existing.reinforce()
            self.total_compiled += 1
            # Connect ALL term IDs that already exist as principles to each other
            self._connect_existing_terms(term_ids)
            # Connect to domain principle
            if domain > 0 and domain in self.principles:
                self.principles[domain].reinforce(amount=8)
                self._connect_principles(existing.sid, domain)
            return 2  # pattern reinforced

        # Create new principle
        evidence = json.dumps(payload, sort_keys=True, separators=(",", ":"))
        p = Principle(
            symbol_id=primary_term,
            weight=180 if source == "creator" else 100,
            evidence=evidence,
            domain=domain,
        )
        self.principles[primary_term] = p

        # Connect to domain principle and related terms already in the graph
        if domain > 0 and domain in self.principles:
            self.principles[domain].reinforce(amount=10)
            self._connect_principles(primary_term, domain)
        self._connect_terms(primary_term, term_ids[1:])

        self.total_compiled += 1
        return 3  # new principle compiled

    def _connect_principles(self, from_sid: int, to_sid: int):
        """Create or strengthen a connection between two principle IDs."""
        if from_sid == to_sid:
            return
        for c in self.connections:
            if (c.from_id == from_sid and c.to_id == to_sid) or \
               (c.from_id == to_sid and c.to_id == from_sid):
                c.weight = min(255, c.weight + 25)
                return
        conn = Connection(from_sid, to_sid, weight=100, conn_type=0)
        self.connections.append(conn)
        # Also update the principles' connection lists
        if from_sid in self.principles and to_sid not in self.principles[from_sid].connections:
            self.principles[from_sid].connections.append(to_sid)
        if to_sid in self.principles and from_sid not in self.principles[to_sid].connections:
            self.principles[to_sid].connections.append(from_sid)

    def _connect_existing_terms(self, term_ids: list[int]):
        """Connect all term IDs that are already principles to each other.
        This builds the knowledge graph from co-occurrence patterns."""
        existing_principles = [tid for tid in term_ids if tid in self.principles]
        for i, sid_a in enumerate(existing_principles):
            for sid_b in existing_principles[i + 1:]:
                self._connect_principles(sid_a, sid_b)

    def _connect_terms(self, primary: int, related_ids: list[int]):
        """Create or strengthen connections between primary term and related terms."""
        for tid in related_ids:
            if tid in self.principles and tid != primary:
                # Check if connection already exists
                existing_conn = None
                for c in self.connections:
                    if c.from_id == primary and c.to_id == tid:
                        existing_conn = c
                        break
                    if c.from_id == tid and c.to_id == primary:
                        existing_conn = c
                        break
                if existing_conn:
                    existing_conn.weight = min(255, existing_conn.weight + 20)
                else:
                    conn = Connection(primary, tid, weight=80, conn_type=0)
                    self.connections.append(conn)

    def _classify(self, source: str, event_type: str, payload: dict) -> int:
        """0=noise, 1=fact, 2=pattern, 3=principle-material"""
        if not event_type or not payload:
            return 0
        # Creator directives always principle-material
        if source == "creator" or "directive" in event_type.lower():
            return 3
        # Repeated events → potential pattern
        if any(kw in event_type.lower() for kw in ("pattern", "criterion", "milestone", "decision", "evolution")):
            return 3
        # Structural changes
        if any(kw in event_type.lower() for kw in ("commit", "feature", "fix", "refactor", "security")):
            return 2
        # Sessions and observations
        if any(kw in event_type.lower() for kw in ("session", "observation", "health", "check")):
            return 1
        return 1  # default: fact

    def _extract_terms(self, event_type: str, payload: dict) -> list[str]:
        """Extract key terms for symbol encoding."""
        terms = []
        # From event type
        terms.append(event_type.lower().replace("_", " ").replace("-", " "))
        # From payload keys and string values
        for k, v in payload.items():
            if isinstance(v, str) and len(v) < 80:
                terms.append(v.lower().strip())
            elif isinstance(v, list):
                for item in v:
                    if isinstance(item, str) and len(item) < 80:
                        terms.append(item.lower().strip())
        # Deduplicate, keep first 5
        seen = set()
        result = []
        for t in terms:
            if t not in seen:
                seen.add(t)
                result.append(t)
                if len(result) >= 5:
                    break
        return result

    def _classify_domain(self, term_ids: list[int], payload: dict = None) -> int:
        """Map to domain symbol (1-15 reserved). Checks payload.domain first."""
        if payload:
            domain_str = payload.get("domain", "")
            if domain_str:
                domain_id = self.symbols.encode(domain_str)
                if 1 <= domain_id <= 15:
                    return domain_id
        domain_terms = {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
        for tid in term_ids:
            if tid in domain_terms:
                return tid
        return 0

    def _find_similar(self, term_ids: list[int], domain: int) -> Principle | None:
        """Find existing principle with similar terms."""
        for sid in term_ids:
            if sid in self.principles and self.principles[sid].domain == domain:
                return self.principles[sid]
        # Check connections for transitive similarity
        for sid in term_ids:
            if sid in self.principles:
                p = self.principles[sid]
                for conn_id in p.connections:
                    if conn_id in self.principles:
                        return self.principles[conn_id]
        return None

    def decay_all(self):
        """Apply time decay to all principles."""
        now = int(time.time())
        for p in list(self.principles.values()):
            days = (now - p.last_reinforced) / 86400.0
            if p.decay(days):
                # Prune
                del self.principles[p.sid]
                self.connections = [c for c in self.connections
                                    if c.from_id != p.sid and c.to_id != p.sid]

    def compile(self) -> dict:
        """Full compile cycle: decay → deduplicate → compress → store."""
        self.decay_all()
        now = int(time.time())

        # Build compact store
        store = {
            "v": 1,
            "m": {
                "c": self.total_compiled,
                "p": len(self.principles),
                "x": len(self.connections),
                "a": sum(1 for p in self.principles.values() if p.weight > 200),
                "t": now,
            },
            "s": self.symbols.to_dict(),
            "p": [p.to_compact() for p in self.principles.values()],
            "x": [c.to_compact() for c in self.connections],
            "t": {
                "last_compile": now,
                "created": self.last_compile or now,
            },
        }

        # Serialize to compact JSON
        raw = json.dumps(store, separators=(",", ":"), ensure_ascii=True).encode("utf-8")
        integrity_hash = hashlib.sha256(raw).digest()

        # Atomic write
        tmp_path = STORE_PATH.with_suffix(".kc.tmp")
        with open(tmp_path, "wb") as f:
            f.write(integrity_hash + raw)
        os.replace(tmp_path, STORE_PATH)

        # Save symbols
        self.symbols.save()

        self.last_compile = now
        return store

    def get_view(self, view_name: str) -> dict:
        """Get a specific view of the compiled knowledge."""
        if view_name == "thavren":
            return self._thavren_view()
        elif view_name == "ovav":
            return self._ovav_view()
        elif view_name == "snv":
            return self._snv_view()
        elif view_name == "stats":
            return self._stats_view()
        else:
            return {"error": f"unknown view: {view_name}"}

    def _thavren_view(self) -> dict:
        """Thavren's personal knowledge: criteria, decisions, evolution."""
        # Thavren domains: security(1), architecture(2), continuity(3), quality(4),
        # delivery(5), relationship(6), learning(8), identity(9), ambition(10), compression(11)
        thavren_domains = {1, 2, 3, 4, 5, 6, 8, 9, 10, 11}
        thavren_sources = {"thavren", "creator"}

        my_principles = []
        for p in self.principles.values():
            # Match on domain OR source
            if p.domain in thavren_domains:
                my_principles.append({
                    "principle": self.symbols.decode(p.sid),
                    "weight": p.weight / 255.0,
                    "domain": self.symbols.decode(p.domain),
                    "connections": [self.symbols.decode(c) for c in p.connections[:5]],
                    "reinforced_days_ago": max(0, (int(time.time()) - p.last_reinforced) // 86400),
                })

        return {
            "entity": "Thavren",
            "total_compiled": self.total_compiled,
            "active_principles": len(my_principles),
            "criteria": [p for p in my_principles if p["weight"] > 0.6],
            "emerging": [p for p in my_principles if 0.3 < p["weight"] <= 0.6],
            "decaying": [p for p in my_principles if p["weight"] <= 0.3],
        }

    def _ovav_view(self) -> dict:
        """OVAV's system knowledge: health, patterns, integrity, state."""
        all_principles = []
        for p in self.principles.values():
            all_principles.append({
                "principle": self.symbols.decode(p.sid),
                "weight": p.weight / 255.0,
                "domain": self.symbols.decode(p.domain),
            })

        # Sort by weight
        all_principles.sort(key=lambda x: x["weight"], reverse=True)

        # Calculate alignment (weighted ratio of strong principles)
        strong = sum(1 for p in all_principles if p["weight"] > 0.6)
        alignment = min(1.0, (strong / max(1, len(all_principles))) * 2.5)

        return {
            "entity": "OVAV",
            "total_compiled": self.total_compiled,
            "active_principles": len(all_principles),
            "active_connections": len(self.connections),
            "alignment_score": round(alignment, 2),
            "health_indicators": {
                "strong_principles": strong,
                "weak_principles": sum(1 for p in all_principles if p["weight"] <= 0.3),
                "total": len(all_principles),
            },
            "top_principles": all_principles[:10],
        }

    def _snv_view(self) -> dict:
        """SNV-relevant: connections, patterns, graph data."""
        return {
            "connections": [
                {
                    "from": self.symbols.decode(c.from_id),
                    "to": self.symbols.decode(c.to_id),
                    "weight": c.weight / 255.0,
                    "type": c.conn_type,
                }
                for c in self.connections
            ],
            "graph_density": len(self.connections) / max(1, len(self.principles) ** 2),
            "strongest_bonds": sorted(
                [{"from": self.symbols.decode(c.from_id), "to": self.symbols.decode(c.to_id), "weight": c.weight / 255.0}
                 for c in self.connections],
                key=lambda x: x["weight"], reverse=True
            )[:5],
        }

    def _stats_view(self) -> dict:
        """Compression statistics."""
        store_size = STORE_PATH.stat().st_size if STORE_PATH.exists() else 0
        return {
            "total_compiled_points": self.total_compiled,
            "active_principles": len(self.principles),
            "active_connections": len(self.connections),
            "symbol_table_size": len(self.symbols.id_to_term),
            "store_size_bytes": store_size,
            "compression_ratio": f"{self.total_compiled}:{store_size}" if store_size else "N/A",
            "bytes_per_point": round(store_size / max(1, self.total_compiled), 2),
            "target_10kb_pct": round(store_size / 10240 * 100, 1),
        }


# ── CLI ────────────────────────────────────────────────────────────

def main():
    import argparse
    ap = argparse.ArgumentParser(description="KC P∞ — Unified Knowledge Compiler")
    ap.add_argument("--compile", action="store_true", help="Full recompile cycle")
    ap.add_argument("--ingest", type=str, help="Ingest JSON file")
    ap.add_argument("--view", type=str, choices=["thavren", "ovav", "snv", "stats"],
                    help="Get specific view")
    ap.add_argument("--ingest-event", nargs=3, metavar=("SOURCE", "TYPE", "PAYLOAD"),
                    help="Ingest single event: source type payload_json")
    ap.add_argument("--json", action="store_true", help="Output as JSON")
    args = ap.parse_args()

    kc = KnowledgeCompiler()

    if args.ingest:
        with open(args.ingest) as f:
            events = json.load(f)
        if isinstance(events, list):
            for ev in events:
                kc.ingest(ev.get("source", "unknown"), ev.get("type", "unknown"), ev.get("payload", {}))
        else:
            kc.ingest(events.get("source", "unknown"), events.get("type", "unknown"), events.get("payload", {}))
        result = kc.compile()
        print(f"Ingested: {kc.total_compiled} total compiled points")
        print(f"Store: {len(json.dumps(result, separators=(',', ':')))} bytes")

    elif args.ingest_event:
        source, etype, payload_str = args.ingest_event
        payload = json.loads(payload_str)
        classification = kc.ingest(source, etype, payload)
        result = kc.compile()
        labels = {0: "noise", 1: "fact", 2: "pattern", 3: "principle"}
        print(f"Classification: {labels.get(classification, 'unknown')}")

    elif args.compile:
        result = kc.compile()
        store_bytes = len(json.dumps(result, separators=(",", ":")))
        print(f"Compiled: {kc.total_compiled} points → {store_bytes} bytes")
        print(f"Principles: {len(kc.principles)}, Connections: {len(kc.connections)}")
        print(f"Store: {STORE_PATH}")

    elif args.view:
        view_data = kc.get_view(args.view)
        if args.json:
            print(json.dumps(view_data, indent=2))
        else:
            print(json.dumps(view_data, indent=2))

    else:
        ap.print_help()


if __name__ == "__main__":
    main()
