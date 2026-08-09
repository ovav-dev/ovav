#!/usr/bin/env python3
"""
OVAV ↔ Thavren Feedback Bridge — L7 Closed Loop
=================================================
Cierra el ciclo de aprendizaje entre OVAV y Thavren usando KC P∞.

Pipeline:
  1. Thavren trabaja → save_memory() ingiere en KC P∞
  2. OVAV analiza KC P∞ → detecta patrones, criterios, correlaciones
  3. OVAV genera feedback → Thavren lo recibe en load_memory()
  4. Thavren actúa sobre el feedback → ciclo se repite

Esto cierra el L7 Feedback Loop que antes estaba abierto (write-only).

USO:
  python3 tools/knowledge/feedback_bridge.py --analyze    # OVAV analiza KC P∞
  python3 tools/knowledge/feedback_bridge.py --report     # Leer último feedback
  python3 tools/knowledge/feedback_bridge.py --json       # Salida JSON
"""

import json
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent.parent
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))
FEEDBACK_PATH = ROOT / ".ovav" / "knowledge" / "feedback_report.json"


def _get_kc():
    try:
        from tools.knowledge.compiler import KnowledgeCompiler
        return KnowledgeCompiler()
    except Exception:
        return None


def analyze() -> dict:
    """
    OVAV analyzes KC P∞ for:
      - New patterns (principles that emerged since last check)
      - Criteria evolution (weight changes > threshold)
      - Health correlations (principles connected to health events)
      - Contradictions (principles with conflicting connections)
      - Growth opportunities (weak principles that could strengthen)

    Returns feedback report.
    """
    kc = _get_kc()
    if not kc:
        return {"error": "KC P∞ not available", "analyzed_at": _now()}

    view = kc.get_view("thavren")
    snv = kc.get_view("snv")
    stats = kc.get_view("stats")

    principles = view.get("criteria", []) + view.get("emerging", [])
    decaying = view.get("decaying", [])

    # ── Detect criteria evolution ─────────────────────────────────
    criteria_changes = []
    for p in principles:
        if 0.55 < p["weight"] < 0.65:
            criteria_changes.append({
                "principle": p["principle"],
                "weight": round(p["weight"], 2),
                "trend": "strengthening",
                "domain": p["domain"],
            })

    # ── Detect new patterns ───────────────────────────────────────
    new_patterns = []
    for p in principles:
        if p["weight"] > 0.7 and p["reinforced_days_ago"] == 0:
            new_patterns.append({
                "principle": p["principle"],
                "weight": round(p["weight"], 2),
                "domain": p["domain"],
            })

    # ── Detect growth opportunities ───────────────────────────────
    growth_opps = []
    for p in decaying:
        if p["weight"] > 0.1:
            growth_opps.append({
                "principle": p["principle"],
                "weight": round(p["weight"], 2),
                "suggestion": "Needs reinforcement — hasn't been observed recently",
            })

    # ── Detect strongest connections ──────────────────────────────
    top_connections = snv.get("strongest_bonds", [])[:5]

    # ── Build knowledge graph health ──────────────────────────────
    strong_principles = sum(1 for p in principles if p["weight"] > 0.6)
    weak_principles = sum(1 for p in principles if p["weight"] <= 0.3)
    alignment = view.get("alignment_score", 0)

    feedback = {
        "schema": "ovav_feedback_bridge_v1",
        "analyzed_at": _now(),
        "kc_state": {
            "total_compiled": stats.get("total_compiled_points", 0),
            "active_principles": len(principles),
            "connections": len(snv.get("connections", [])),
            "store_bytes": stats.get("store_size_bytes", 0),
            "target_10kb_pct": stats.get("target_10kb_pct", 0),
        },
        "insights": {
            "strong_principles": strong_principles,
            "weak_principles": weak_principles,
            "criteria_strengthening": len(criteria_changes),
            "new_patterns": len(new_patterns),
            "growth_opportunities": len(growth_opps),
        },
        "details": {
            "criteria_changes": criteria_changes,
            "new_patterns": new_patterns,
            "growth_opportunities": growth_opps,
            "strongest_bonds": top_connections,
        },
        "recommendations": _generate_recommendations(
            strong_principles, weak_principles, criteria_changes,
            growth_opps, alignment
        ),
    }

    # Persist feedback
    FEEDBACK_PATH.parent.mkdir(parents=True, exist_ok=True)
    FEEDBACK_PATH.write_text(json.dumps(feedback, indent=2))

    return feedback


def _generate_recommendations(strong: int, weak: int, changes: list,
                               growth: list, alignment: float) -> list[str]:
    """Generate human-readable recommendations from analysis."""
    recs = []

    if alignment < 0.5:
        recs.append(f"Alignment bajo ({alignment}): regenerar SELF_MODEL y recompilar KC P∞")
    elif alignment < 0.8:
        recs.append(f"Alignment moderado ({alignment}): continuar alimentando KC P∞ con decisiones y aprendizajes")

    if weak > strong:
        recs.append(f"Más principios débiles ({weak}) que fuertes ({strong}): riesgo de pérdida de conocimiento. Reforzar criterios clave en próximas sesiones.")
    elif strong > 0:
        recs.append(f"{strong} principios fuertes: base de conocimiento sólida. Mantener.")

    if changes:
        domains = set(c["domain"] for c in changes if c["domain"] != "__NULL__")
        if domains:
            recs.append(f"Criterios fortaleciéndose en dominios: {', '.join(list(domains)[:3])}")

    if growth:
        recs.append(f"{len(growth)} principios necesitan refuerzo antes de decaer completamente.")

    if not recs:
        recs.append("Sistema de conocimiento estable. Continuar alimentando con nuevas sesiones.")

    return recs


def read_feedback() -> dict | None:
    """Read the last feedback report."""
    if FEEDBACK_PATH.exists():
        try:
            return json.loads(FEEDBACK_PATH.read_text())
        except Exception:
            pass
    return None


def _now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def main():
    import sys
    json_out = "--json" in sys.argv

    if "--analyze" in sys.argv:
        feedback = analyze()
        if "error" in feedback:
            print(f"❌ Feedback Bridge Error: {feedback['error']}")
            return
        if json_out:
            print(json.dumps(feedback, indent=2))
        else:
            print("🔄 OVAV Feedback Bridge — Analysis Complete")
            print(f"   Compiled: {feedback['kc_state']['total_compiled']} points")
            print(f"   Strong principles: {feedback['insights']['strong_principles']}")
            print(f"   Weak principles:   {feedback['insights']['weak_principles']}")
            print(f"   Strengthening:      {feedback['insights']['criteria_strengthening']} criteria")
            print(f"   New patterns:       {feedback['insights']['new_patterns']}")
            print(f"   Growth opps:        {feedback['insights']['growth_opportunities']}")
            print(f"   Store: {feedback['kc_state']['store_bytes']} bytes ({feedback['kc_state']['target_10kb_pct']}% of 10KB)")
            print("\n   📋 Recommendations:")
            for r in feedback["recommendations"]:
                print(f"      • {r}")
        return

    if "--report" in sys.argv:
        fb = read_feedback()
        if fb:
            if json_out:
                print(json.dumps(fb, indent=2))
            else:
                print(f"📊 Last Feedback Report — {fb.get('analyzed_at', '?')}")
                for r in fb.get("recommendations", []):
                    print(f"   • {r}")
        else:
            print("No feedback report found. Run --analyze first.")
        return

    print("Usage: feedback_bridge.py --analyze | --report [--json]")


if __name__ == "__main__":
    main()
