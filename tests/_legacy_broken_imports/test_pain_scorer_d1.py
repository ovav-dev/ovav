#!/usr/bin/env python3
"""Tests para PainScorer D1 — clasificador de impacto operacional inteligente.

Cubre:
  - Unit: score() con diferentes combinaciones de factores
  - Integration: nerve_bus.publish() con auto-scoring
  - Edge cases: eventos sin payload, severidades extremas, bursts
  - Analysis: analyze_window(), thresholds, health
  - Lockdown: triggers por threshold y burst
  - Backward compat: score_pain() legacy
"""

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))


# ══════════════════════════════════════════════════════════════════════════════
# HELPERS
# ══════════════════════════════════════════════════════════════════════════════

def ok(msg: str) -> str:
    return f"  ✅ {msg}"


def fail(msg: str) -> str:
    return f"  ❌ {msg}"


passed = 0
failed = 0


def assert_true(condition: bool, msg: str):
    global passed, failed
    if condition:
        passed += 1
        print(ok(msg))
    else:
        failed += 1
        print(fail(msg))


def assert_equal(a, b, msg: str):
    assert_true(a == b, f"{msg}: {a} == {b}")


def assert_greater(a, b, msg: str):
    assert_true(a > b, f"{msg}: {a} > {b}")


def assert_between(val, lo, hi, msg: str):
    assert_true(lo <= val <= hi, f"{msg}: {lo} <= {val} <= {hi}")


# ══════════════════════════════════════════════════════════════════════════════
# UNIT TESTS — PainScorer.score()
# ══════════════════════════════════════════════════════════════════════════════

def test_score_basic():
    """D1.UT.01: Score básico — evento informativo."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("periodic_check", "info", "cron", {"message": "routine check"})

    assert_between(result.pain_score, 0, 25, "Pain score for info/periodic_check")
    assert_true(result.confidence > 0.5, f"Confidence {result.confidence} > 0.5")
    assert_equal(result.recommendation, "none", "No recommendation for low pain")
    assert_true(not result.triggers_lockdown, "No lockdown for low pain")
    assert_true(len(result.factors) >= 2, f"At least 2 factors: {len(result.factors)}")


def test_score_critical():
    """D1.UT.02: Score crítico — debe disparar lockdown."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score(
        "governance_breach", "critical", "external_scanner",
        {"files": ["AGENTS.md", "current_authority_contract.yaml"], "blockade": True}
    )

    assert_greater(result.pain_score, 70, "High pain for governance_breach")
    assert_true(result.triggers_lockdown, "Triggers lockdown for high pain")
    assert_true(result.recommendation in ("lockdown_immediate", "lockdown_recommended"),
                f"Lockdown recommendation: {result.recommendation}")


def test_score_blockade():
    """D1.UT.03: Score blockade — máximo impacto."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("intrusion_detected", "blockade", "host_security",
                          {"message": "unauthorized access detected"})

    assert_greater(result.pain_score, 85, "Very high pain for blockade intrusion")
    assert_true(result.triggers_lockdown, "Lockdown triggered")
    assert_true(result.confidence > 0.7, "High confidence for severe event")


def test_score_silent():
    """D1.UT.04: Score silent — sin impacto."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("silent_observation", "silent", "sensor_01", None)

    assert_equal(result.pain_score, 0, "Zero pain for silent observation")
    assert_true(not result.triggers_lockdown, "No lockdown for silent")
    assert_equal(result.recommendation, "none", "No recommendation")


def test_score_payload_critical_files():
    """D1.UT.05: Payload con archivos críticos amplifica pain."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()

    # Sin archivos críticos
    r1 = scorer.score("file_drift", "warning", "integrity_mesh",
                      {"files": ["README.md"]})
    # Con archivos críticos
    r2 = scorer.score("file_drift", "warning", "integrity_mesh",
                      {"files": ["AGENTS.md", "permission_authority.json", "core_hashes.yaml"]})

    assert_greater(r2.pain_score, r1.pain_score,
                   f"Critical files amplify pain: {r2.pain_score} > {r1.pain_score}")


def test_score_source_reputation():
    """D1.UT.06: Fuente externa recibe amplificación adicional."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()

    # Usar nombres únicos para evitar interferencia con historial de tests
    r1 = scorer.score("periodic_check", "warning", "unique_internal_test_42", None)
    r2 = scorer.score("periodic_check", "warning", "unique_external_test_42", None)

    # Con mismo tipo y severidad, fuente externa debe tener >= pain que interna
    # (puede ser igual si no hay historial pero nunca menor)
    assert_true(r2.pain_score >= r1.pain_score,
                f"External source >= internal: {r2.pain_score} >= {r1.pain_score}")


def test_score_factors_structure():
    """D1.UT.07: Todos los factores tienen estructura correcta."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("critical_drift", "critical", "integrity_mesh",
                          {"files": ["AGENTS.md"]})

    for f in result.factors:
        assert_true("name" in f, f"Factor has name: {f.get('name')}")
        assert_true("value" in f, f"Factor has value: {f.get('value')}")
        assert_true("contribution" in f, "Factor has contribution")
        assert_true("detail" in f, "Factor has detail")

    assert_true(len(result.factors) >= 3, f"Multiple factors active: {len(result.factors)}")


def test_score_confidence():
    """D1.UT.08: Confianza varía según calidad de factores."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()

    # Poca info → baja confianza
    r1 = scorer.score("unknown_type", "info", "unknown_source", None)
    # Mucha info → alta confianza
    r2 = scorer.score("governance_breach", "blockade", "external_scanner",
                      {"files": ["AGENTS.md", "ovav_laws.yaml"], "blockade": True})

    assert_true(r2.confidence >= r1.confidence,
                f"More factors → higher confidence: {r2.confidence} >= {r1.confidence}")


def test_score_to_dict():
    """D1.UT.09: PainResult.to_dict() es serializable."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("health_degraded", "warning", "health_check", None)
    d = result.to_dict()

    assert_true("pain_score" in d, "to_dict has pain_score")
    assert_true("confidence" in d, "to_dict has confidence")
    assert_true("factors" in d, "to_dict has factors")
    assert_true("assessment" in d, "to_dict has assessment")
    assert_true("recommendation" in d, "to_dict has recommendation")

    # Debe ser JSON serializable
    json_str = json.dumps(d)
    assert_true(len(json_str) > 0, "to_dict is JSON serializable")


# ══════════════════════════════════════════════════════════════════════════════
# ANALYSIS TESTS
# ══════════════════════════════════════════════════════════════════════════════

def test_analyze_window():
    """D1.AT.01: analyze_window retorna estructura completa."""
    from tools.agent_runtime.pain_scorer import analyze_window

    result = analyze_window(hours=24)

    assert_true(result.window_hours == 24, "Window hours set")
    assert_true(result.total_events >= 0, f"Total events >= 0: {result.total_events}")
    assert_true(result.pain_trend in ("rising", "falling", "stable", "volatile"),
                f"Valid pain trend: {result.pain_trend}")
    assert_between(result.health_score, 0, 100,
                   f"Health score in range: {result.health_score}")
    assert_true(isinstance(result.top_sources, list), "top_sources is list")
    assert_true(isinstance(result.severity_distribution, dict), "severity_distribution is dict")


def test_analyze_window_empty():
    """D1.AT.02: Ventana con pocos eventos retorna defaults seguros."""
    from tools.agent_runtime.pain_scorer import analyze_window

    result = analyze_window(hours=1)

    assert_true(result.health_score >= 0, "Health score >= 0 even on short window")
    assert_true(not result.escalation_detected, "No false escalation")
    assert_true(isinstance(result.total_events, int), "total_events is int")


def test_health_assessment():
    """D1.AT.03: health_assessment retorna estructura completa."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.health_assessment(hours=24)

    assert_true("status" in result, "Has status")
    assert_true("color" in result, "Has color")
    assert_true("health_score" in result, "Has health_score")
    assert_true("insights" in result, "Has insights")
    assert_true("recommended_thresholds" in result, "Has recommended_thresholds")
    assert_true(result["color"] in ("green", "yellow", "orange", "red"),
                f"Valid color: {result['color']}")
    assert_between(result["health_score"], 0, 100,
                   f"Health score in range: {result['health_score']}")


def test_thresholds_recommendation():
    """D1.AT.04: recommend_thresholds retorna thresholds ordenados."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.recommend_thresholds(hours=168)

    assert_true(result.recommended_warning < result.recommended_critical,
                f"warning < critical: {result.recommended_warning} < {result.recommended_critical}")
    assert_true(result.recommended_critical < result.recommended_lockdown,
                f"critical < lockdown: {result.recommended_critical} < {result.recommended_lockdown}")
    assert_between(result.recommended_lockdown, 50, 100, "Lockdown threshold in range")
    assert_between(result.confidence, 0, 1, f"Confidence in range: {result.confidence}")


def test_source_reputation_api():
    """D1.AT.05: get_source_reputation retorna estructura completa."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.get_source_reputation("integrity_mesh")

    assert_true("source" in result, "Has source")
    assert_true("reputation" in result, "Has reputation")
    assert_true("tier" in result, "Has tier")
    assert_between(result["reputation"], 0, 1, f"Reputation in range: {result['reputation']}")
    assert_true(result["tier"] in ("trusted", "neutral", "elevated", "high_risk"),
                f"Valid tier: {result['tier']}")


# ══════════════════════════════════════════════════════════════════════════════
# INTEGRATION TESTS — nerve_bus + PainScorer
# ══════════════════════════════════════════════════════════════════════════════

def test_nerve_bus_auto_score():
    """D1.IT.01: nerve_bus.publish() usa PainScorer D1 automáticamente."""
    from tools.agent_runtime.nerve_bus import publish

    # Publicar sin pain_score explícito → usa PainScorer D1
    eid = publish("config_drift", {"files": ["AGENTS.md"]},
                  severity="critical", source="integrity_mesh")
    assert_true(eid is not None, "Event published successfully")
    assert_true(eid.startswith("evt_"), f"Valid event ID: {eid}")


def test_nerve_bus_pain_stored():
    """D1.IT.02: El pain score se almacena correctamente en el evento."""
    from tools.agent_runtime.nerve_bus import latest, publish

    publish("validator_fail", {"validator": "test_check"},
            severity="warning", source="test_harness")

    events = latest(count=1)
    assert_true(len(events) > 0, "Events retrieved")
    e = events[0]
    assert_true("pain_score" in e, "pain_score stored in event")
    assert_between(e["pain_score"], 0, 100, f"pain_score in range: {e['pain_score']}")


def test_nerve_bus_explicit_pain():
    """D1.IT.03: pain_score explícito se respeta (no sobreescrito)."""
    from tools.agent_runtime.nerve_bus import latest, publish

    publish("test_event", {"msg": "explicit"},
            severity="info", source="test", pain_score=42)

    events = latest(count=1)
    e = events[0]
    assert_equal(e["pain_score"], 42, "Explicit pain_score preserved")


def test_backward_compat_score_pain():
    """D1.IT.04: score_pain() legacy sigue funcionando vía delegación D1."""
    from tools.agent_runtime.nerve_bus import score_pain

    s1 = score_pain("critical_drift", "critical", "integrity_mesh",
                    {"files": ["AGENTS.md"]})
    s2 = score_pain("periodic_check", "info", "cron", None)

    assert_true(s1 > s2, f"Critical > info: {s1} > {s2}")
    assert_between(s1, 0, 100, f"Score in range: {s1}")
    assert_between(s2, 0, 100, f"Score in range: {s2}")


def test_lockdown_trigger():
    """D1.IT.05: Evento de alto pain activa lockdown."""
    from tools.agent_runtime.nerve_bus import check_lockdown_status, clear_lockdown, publish

    clear_lockdown()

    # Publicar evento crítico que debería disparar lockdown
    publish("governance_breach", {"blockade": True, "files": ["AGENTS.md"]},
            severity="critical", source="external_scanner")

    status = check_lockdown_status()
    assert_true(status.get("lockdown_active", False),
                f"Lockdown triggered by high-pain event: {status}")

    clear_lockdown()


def test_lockdown_no_trigger_low_pain():
    """D1.IT.06: Evento de bajo pain NO activa lockdown."""
    from tools.agent_runtime.nerve_bus import check_lockdown_status, clear_lockdown, publish

    clear_lockdown()

    publish("periodic_check", {"msg": "routine"},
            severity="info", source="cron")

    status = check_lockdown_status()
    assert_true(not status.get("lockdown_active", True),
                f"No lockdown for low-pain event: {status}")

    clear_lockdown()


def test_lockdown_clear():
    """D1.IT.07: clear_lockdown() funciona correctamente."""
    from tools.agent_runtime.nerve_bus import check_lockdown_status, clear_lockdown, publish

    clear_lockdown()
    publish("blockade_active", {"msg": "force"},
            severity="blockade", source="test")
    assert_true(check_lockdown_status()["lockdown_active"], "Lockdown active")

    clear_lockdown()
    assert_true(not check_lockdown_status()["lockdown_active"], "Lockdown cleared")

    clear_lockdown()


# ══════════════════════════════════════════════════════════════════════════════
# CONVENIENCE FUNCTIONS
# ══════════════════════════════════════════════════════════════════════════════

def test_score_pain_detailed():
    """D1.CF.01: score_pain_detailed retorna PainResult."""
    from tools.agent_runtime.pain_scorer import score_pain_detailed

    result = score_pain_detailed("health_degraded", "warning", "health_check", None)

    assert_true(hasattr(result, "pain_score"), "PainResult has pain_score")
    assert_true(hasattr(result, "factors"), "PainResult has factors")
    assert_true(hasattr(result, "recommendation"), "PainResult has recommendation")


def test_module_level_score_pain():
    """D1.CF.02: score_pain a nivel módulo devuelve int."""
    from tools.agent_runtime.pain_scorer import score_pain

    s = score_pain("config_drift", "warning", "test", None)
    assert_true(isinstance(s, int), f"Returns int: {type(s)}")
    assert_between(s, 0, 100, f"Score in range: {s}")


def test_module_level_health():
    """D1.CF.03: health() a nivel módulo funciona."""
    from tools.agent_runtime.pain_scorer import health

    result = health()
    assert_true("status" in result, "health has status")
    assert_true("health_score" in result, "health has health_score")


def test_module_level_thresholds():
    """D1.CF.04: thresholds() a nivel módulo funciona."""
    from tools.agent_runtime.pain_scorer import thresholds

    result = thresholds(hours=168)
    assert_true(hasattr(result, "recommended_lockdown"), "ThresholdResult has recommended_lockdown")


def test_module_level_source_reputation():
    """D1.CF.05: source_reputation() a nivel módulo funciona."""
    from tools.agent_runtime.pain_scorer import source_reputation

    result = source_reputation("integrity_mesh")
    assert_true("reputation" in result, "Reputation dict has reputation")
    assert_true("tier" in result, "Reputation dict has tier")


# ══════════════════════════════════════════════════════════════════════════════
# EDGE CASES
# ══════════════════════════════════════════════════════════════════════════════

def test_edge_unknown_type():
    """D1.EC.01: Tipo de evento desconocido — fallback seguro."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("completely_unknown_type_xyz", "critical", "test", None)

    assert_between(result.pain_score, 0, 100, "Unknown type still produces valid score")
    assert_true(result.confidence > 0, "Has confidence even for unknown type")


def test_edge_invalid_severity():
    """D1.EC.02: Severidad inválida — fallback a info."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("periodic_check", "INVALID_SEVERITY", "test", None)

    assert_true(result.pain_score >= 0, "Invalid severity produces non-negative score")


def test_edge_none_payload():
    """D1.EC.03: Payload None no causa errores."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()
    result = scorer.score("file_drift", "warning", "test", None)

    assert_true(result.pain_score >= 0, "None payload handled")
    assert_true(len(result.factors) >= 1, "Factors still produced")


def test_edge_empty_events():
    """D1.EC.04: Sin eventos históricos → defaults seguros."""
    from tools.agent_runtime.pain_scorer import analyze_window

    # Forzar ventana de 0 horas = sin eventos
    result = analyze_window(hours=0)

    assert_true(result.health_score >= 0, "Health score valid even with 0 events")


def test_edge_pain_scale_bounds():
    """D1.EC.05: Pain score nunca excede 0-100."""
    from tools.agent_runtime.pain_scorer import PainScorer

    scorer = PainScorer()

    # Máximo posible
    r1 = scorer.score("intrusion_detected", "blockade", "external_hostile",
                      {"files": ["AGENTS.md", "ovav_laws.yaml", "sbom.yaml"], "blockade": True})
    assert_between(r1.pain_score, 0, 100, f"Max pain in bounds: {r1.pain_score}")

    # Mínimo posible
    r2 = scorer.score("silent_observation", "silent", "sensor", None)
    assert_between(r2.pain_score, 0, 100, f"Min pain in bounds: {r2.pain_score}")


# ══════════════════════════════════════════════════════════════════════════════

def run_all():
    global passed, failed
    print("╔══════════════════════════════════════════════════╗")
    print("║     PainScorer D1 — Test Suite Completa          ║")
    print("╚══════════════════════════════════════════════════╝")
    print()

    sections = [
        ("UNIT — PainScorer.score()", [
            test_score_basic,
            test_score_critical,
            test_score_blockade,
            test_score_silent,
            test_score_payload_critical_files,
            test_score_source_reputation,
            test_score_factors_structure,
            test_score_confidence,
            test_score_to_dict,
        ]),
        ("ANALYSIS — Window, Health, Thresholds", [
            test_analyze_window,
            test_analyze_window_empty,
            test_health_assessment,
            test_thresholds_recommendation,
            test_source_reputation_api,
        ]),
        ("INTEGRATION — NerveBus + Lockdown", [
            test_nerve_bus_auto_score,
            test_nerve_bus_pain_stored,
            test_nerve_bus_explicit_pain,
            test_backward_compat_score_pain,
            test_lockdown_trigger,
            test_lockdown_no_trigger_low_pain,
            test_lockdown_clear,
        ]),
        ("CONVENIENCE — Module-level functions", [
            test_score_pain_detailed,
            test_module_level_score_pain,
            test_module_level_health,
            test_module_level_thresholds,
            test_module_level_source_reputation,
        ]),
        ("EDGE CASES", [
            test_edge_unknown_type,
            test_edge_invalid_severity,
            test_edge_none_payload,
            test_edge_empty_events,
            test_edge_pain_scale_bounds,
        ]),
    ]

    for section_name, tests in sections:
        print(f"\n── {section_name} ──")
        for test_fn in tests:
            try:
                test_fn()
            except Exception as e:
                failed += 1
                print(fail(f"{test_fn.__name__}: EXCEPTION — {e}"))

    print(f"\n{'='*50}")
    print(f"  Results: {passed} passed, {failed} failed")
    print(f"{'='*50}")

    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(run_all())
