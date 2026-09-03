"""HISTORICAL — S144 evals for retired model task routing."""

from __future__ import annotations

import json
import sys
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from tools.harnesses.model_task_router import TaskSignals, build_model_route

GOLDEN_CASES_PATH = REPO_ROOT / "tests/fixtures/golden/model_task_router_cases.json"


def _load_golden_cases() -> dict:
    return json.loads(GOLDEN_CASES_PATH.read_text(encoding="utf-8"))


def _signals_from_case(case: dict) -> TaskSignals:
    return TaskSignals(
        task_text=case["task"],
        file_count=case.get("files", 0),
        source_count=case.get("sources", 0),
        risk_flags=tuple(case.get("risk_flags", ())),
    )


def _get_nested(payload: dict, dotted_key: str):
    current = payload
    for part in dotted_key.split("."):
        current = current[part]
    return current


def test_low_risk_small_task_uses_fast_tier():
    route = build_model_route(TaskSignals(task_text="docs typo", file_count=1))
    assert route["recommended_model_tier"] == "fast"
    assert route["requires_high"] is False


def test_medium_cli_task_uses_medium_tier():
    route = build_model_route(TaskSignals(task_text="add CLI test coverage metrics", file_count=3))
    assert route["recommended_model_tier"] == "coding-light"
    assert route["variant"] in ("opencode-go/qwen3.7-plus", "opencode-go/glm-5.1")


def test_security_or_global_surface_requires_high():
    route = build_model_route(TaskSignals(task_text="change global deploy permission token handling", file_count=2))
    assert route["recommended_model_tier"] == "high"
    assert route["requires_high"] is True
    assert route["fallback"].startswith("opencode-go/")
    assert route["variant"].startswith("opencode-go/")


def test_four_files_triggers_delegation():
    route = build_model_route(TaskSignals(task_text="runtime refactor", file_count=4))
    assert route["delegation"]["required"] is True
    assert route["delegation"]["suggested_squad"] == "squad-sys-architect"


def test_golden_routing_cases_match_expected_decisions():
    fixture = _load_golden_cases()
    for case in fixture["cases"]:
        route = build_model_route(_signals_from_case(case))
        for key, expected in case["expected"].items():
            assert _get_nested(route, key) == expected, f"{case['id']} expected {key}={expected}"


def test_routes_use_only_authorized_active_models():
    routes = [
        build_model_route(TaskSignals(task_text="docs typo", file_count=1)),
        build_model_route(TaskSignals(task_text="add CLI test coverage metrics", file_count=3)),
        build_model_route(TaskSignals(task_text="research benchmark compare sources with evidence", source_count=4)),
        build_model_route(TaskSignals(task_text="change global deploy permission token handling", file_count=2)),
    ]
    for route in routes:
        serialized = json.dumps(route).lower()
        for forbidden in (
            "google",
            "gemini",
            "openrouter",
            "openai/",
            "anthropic/",
            "gpt-4",
            "gpt-5",
            "claude",
        ):
            assert forbidden not in serialized, f"forbidden '{forbidden}' found in route"
        for field in ("variant", "fallback"):
            if "/" in route[field]:
                assert route[field].startswith("opencode-go/")


def test_routes_expose_multi_model_execution_plan():
    route = build_model_route(TaskSignals(task_text="implement runtime validator patch", file_count=3))
    planned_models = {step["model"] for step in route["execution_plan"]}
    assert len(planned_models) >= 2, f"Expected at least 2 models in execution plan, got {len(planned_models)}"
    for model in planned_models:
        assert model.startswith("opencode-go/"), f"Model {model} should be opencode-go/*"


def test_router_latency_metric_stays_under_interactive_threshold():
    fixture = _load_golden_cases()
    cases = fixture["cases"]
    iterations = fixture["performance"]["iterations"]
    threshold_ms = fixture["performance"]["max_average_ms"]

    started = time.perf_counter()
    for _ in range(iterations):
        for case in cases:
            build_model_route(_signals_from_case(case))
    elapsed_ms = (time.perf_counter() - started) * 1000
    average_ms = elapsed_ms / (iterations * len(cases))

    assert average_ms < threshold_ms


if __name__ == "__main__":
    test_low_risk_small_task_uses_fast_tier()
    test_medium_cli_task_uses_medium_tier()
    test_security_or_global_surface_requires_high()
    test_four_files_triggers_delegation()
    test_golden_routing_cases_match_expected_decisions()
    test_routes_use_only_authorized_active_models()
    test_routes_expose_multi_model_execution_plan()
    test_router_latency_metric_stays_under_interactive_threshold()
    print("PASS S144 model task router evals")
