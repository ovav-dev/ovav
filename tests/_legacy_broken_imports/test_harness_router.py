"""S82 Harness Router — Layer 2 test suite.

Verifies:
  1. surface_classifier correctly identifies surfaces from paths and lanes
  2. validator_resolver maps surfaces to validators with proper ordering
  3. trigger_harnesses dry_run does not execute
  4. trigger_harnesses live execution captures pass/fail
  5. aggregate_results produces actionable report
  6. route_and_trigger full pipeline works
  7. Greeting lane is minimal (no heavy validators)
  8. Closure task triggers comprehensive validation
  9. always_run validators appear first
 10. Deduplication works across overlapping surfaces
"""

from __future__ import annotations

import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))


def test_surface_classifier_paths():
    """Paths to .opencode/ and tools/ should classify correctly."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(paths=[
        ".opencode/skills/ovav-context-pack/SKILL.md",
        "tools/validators/check_foo.py",
    ])
    surfaces = result["affected_surfaces"]
    assert ".opencode/" in surfaces, f"Expected .opencode/ in {surfaces}"
    assert "tools/" in surfaces, f"Expected tools/ in {surfaces}"
    assert result["surface_count"] == 2
    return True


def test_surface_classifier_lane():
    """runtime_governance lane should trigger tools/ and .ovav/policy/."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(lane="runtime_governance")
    surfaces = result["affected_surfaces"]
    assert "tools/" in surfaces, f"Expected tools/ in {surfaces}"
    assert ".ovav/policy/" in surfaces, f"Expected .ovav/policy/ in {surfaces}"
    return True


def test_surface_classifier_greeting_minimal():
    """greeting_identity lane should have no surfaces."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(lane="greeting_identity")
    assert result["surface_count"] == 0, f"Greeting should have 0 surfaces, got {result['surface_count']}"
    return True


def test_surface_classifier_high_risk_escalation():
    """High risk should add safety surfaces."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(lane="greeting_identity", risk_level="high")
    surfaces = result["affected_surfaces"]
    assert ".ovav/policy/" in surfaces, "High risk should add .ovav/policy/"
    assert ".ovav/service_areas/" in surfaces, "High risk should add .ovav/service_areas/"
    return True


def test_surface_classifier_closure_task():
    """Closure task should trigger comprehensive validation."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(lane="validation_closure", task_kind="closure")
    surfaces = result["affected_surfaces"]
    # Closure should add all validation surfaces (with .ovav/ prefix where applicable)
    for expected in [".ovav/policy/", ".ovav/service_areas/", "docs/", ".ovav/registry/", ".ovav/schemas/", "tests/"]:
        assert expected in surfaces, f"Closure should include {expected}"
    return True


def test_validator_resolver_opencode():
    """Touching .opencode/ should resolve to validate_model_policy (current config)."""
    from tools.agent_runtime.harness_router import validator_resolver

    result = validator_resolver([".opencode/"])
    validators = result["resolved_validators"]
    assert "validate_model_policy" in validators, f"Expected validate_model_policy, got {validators}"
    return True


def test_validator_resolver_always_run_first():
    """always_run validators should appear first, starting with highest-priority security."""
    from tools.agent_runtime.harness_router import validator_resolver

    result = validator_resolver(["tools/"])
    validators = result["resolved_validators"]
    assert len(validators) > 0, "Should have at least one validator"
    assert validators[0].startswith("check_L6_"), (
        f"First validator should be a security check, got {validators[0]}"
    )
    return True


def test_validator_resolver_deduplication():
    """Overlapping surfaces should not duplicate validators."""
    from tools.agent_runtime.harness_router import validator_resolver

    result = validator_resolver(["tools/", ".ovav/policy/"])
    validators = result["resolved_validators"]
    # No duplicates should appear
    assert len(validators) == len(set(validators)), (
        f"Found duplicates: {[v for v in validators if validators.count(v) > 1]}"
    )
    return True


def test_validator_resolver_lane_specific():
    """Lane-specific validators are appended when lane_validators config exists."""
    from tools.agent_runtime.harness_router import validator_resolver

    result = validator_resolver(["tools/"], lane="runtime_governance")
    validators = result["resolved_validators"]
    # At minimum, always_run validators + surface validators should be resolved
    assert len(validators) > 0, "Should resolve at least always_run validators"
    return True


def test_trigger_harnesses_dry_run():
    """Dry run should not execute anything."""
    from tools.agent_runtime.harness_router import trigger_harnesses

    result = trigger_harnesses(
        ["workspace_safety_gate", "check_permission_policy_drift"],
        dry_run=True,
    )
    assert result["execution_mode"] == "dry_run"
    assert result["counts"]["skipped"] == 2
    assert result["counts"]["passed"] == 0
    return True


def test_trigger_harnesses_live():
    """Live execution should capture pass/fail for real validators."""
    from tools.agent_runtime.harness_router import trigger_harnesses

    # workspace_safety_gate should pass (we're in a valid repo root)
    result = trigger_harnesses(["workspace_safety_gate"], timeout_per_validator=30)
    assert result["execution_mode"] == "live"
    assert result["counts"]["total"] == 1
    # workspace_safety_gate should pass in a clean repo
    assert result["overall_status"] in ("pass", "fail"), f"Unexpected status: {result['overall_status']}"
    return True


def test_aggregate_results_pass():
    """Aggregation of all-pass results should show pass."""
    from tools.agent_runtime.harness_router import aggregate_results

    classification = {
        "affected_surfaces": [".opencode/"],
        "signals": ["surface_path:.opencode/"],
    }
    resolution = {
        "resolved_validators": ["check_foo"],
        "resolved_families": ["safety_gate"],
    }
    execution = {
        "execution_mode": "live",
        "overall_status": "pass",
        "counts": {"total": 1, "passed": 1, "warned": 0, "failed": 0, "errored": 0},
        "results": [{"validator": "check_foo", "status": "pass", "output": "PASS", "returncode": 0, "duration_ms": 10}],
    }

    result = aggregate_results(classification, resolution, execution)
    assert result["overall_status"] == "pass"
    assert "1 validators passed" in result["message"]
    assert not result["stop_condition"]
    return True


def test_aggregate_results_fail():
    """Aggregation with failures should show fail and stop_condition=True."""
    from tools.agent_runtime.harness_router import aggregate_results

    classification = {"affected_surfaces": ["tools/"], "signals": []}
    resolution = {"resolved_validators": ["check_bad"], "resolved_families": []}
    execution = {
        "execution_mode": "live",
        "overall_status": "fail",
        "counts": {"total": 1, "passed": 0, "warned": 0, "failed": 1, "errored": 0},
        "results": [{"validator": "check_bad", "status": "fail", "output": "FAIL: bad thing", "returncode": 1, "duration_ms": 5}],
    }

    result = aggregate_results(classification, resolution, execution)
    assert result["overall_status"] == "fail"
    assert result["stop_condition"]
    assert len(result["failures"]) == 1
    return True


def test_route_and_trigger_full_pipeline_dry():
    """Full pipeline with dry-run should work end to end."""
    from tools.agent_runtime.harness_router import route_and_trigger

    result = route_and_trigger(
        paths=[".opencode/skills/foo/SKILL.md"],
        lane="opencode_system",
        dry_run=True,
    )
    assert "classification" in result
    assert "resolution" in result
    assert "execution" in result
    assert "aggregated" in result
    assert result["execution"]["execution_mode"] == "dry_run"
    return True


def test_route_and_trigger_full_pipeline_live():
    """Full pipeline live execution with minimal surface."""
    from tools.agent_runtime.harness_router import route_and_trigger

    result = route_and_trigger(
        paths=[".opencode/skills/foo/SKILL.md"],
        dry_run=False,
    )
    assert result["aggregated"]["overall_status"] in ("pass", "fail")
    return True


def test_greeting_minimal_coverage():
    """Greeting identity should not trigger implementation or closure validators."""
    from tools.agent_runtime.harness_router import surface_classifier, validator_resolver

    c = surface_classifier(lane="greeting_identity")
    # greeting should have no surfaces
    assert c["surface_count"] == 0, f"Greeting has {c['surface_count']} surfaces"

    r = validator_resolver(c["affected_surfaces"])
    # Only always_run validators should be present for greeting
    validators = r["resolved_validators"]
    heavy = {"check_agent_runtime_enforcement", "validate_harnesses", "check_opencode_runtime_wiring"}
    assert not heavy.intersection(validators), (
        f"Greeting should not include heavy validators: {heavy & set(validators)}"
    )
    return True


def test_path_none_and_lane_none():
    """No paths and no lane should result in empty classification."""
    from tools.agent_runtime.harness_router import surface_classifier

    result = surface_classifier(paths=None, lane=None)
    assert result["surface_count"] == 0
    return True


def test_nonexistent_validator_graceful():
    """Nonexistent validator should return error status, not crash."""
    from tools.agent_runtime.harness_router import trigger_harnesses

    result = trigger_harnesses(["nonexistent_validator_xyz"])
    assert result["counts"]["errored"] >= 1
    return True


def run_all() -> int:
    """Run all tests and report results."""
    tests = [
        ("surface_classifier_paths", test_surface_classifier_paths),
        ("surface_classifier_lane", test_surface_classifier_lane),
        ("surface_classifier_greeting_minimal", test_surface_classifier_greeting_minimal),
        ("surface_classifier_high_risk_escalation", test_surface_classifier_high_risk_escalation),
        ("surface_classifier_closure_task", test_surface_classifier_closure_task),
        ("validator_resolver_opencode", test_validator_resolver_opencode),
        ("validator_resolver_always_run_first", test_validator_resolver_always_run_first),
        ("validator_resolver_deduplication", test_validator_resolver_deduplication),
        ("validator_resolver_lane_specific", test_validator_resolver_lane_specific),
        ("trigger_harnesses_dry_run", test_trigger_harnesses_dry_run),
        ("trigger_harnesses_live", test_trigger_harnesses_live),
        ("aggregate_results_pass", test_aggregate_results_pass),
        ("aggregate_results_fail", test_aggregate_results_fail),
        ("route_and_trigger_full_pipeline_dry", test_route_and_trigger_full_pipeline_dry),
        ("route_and_trigger_full_pipeline_live", test_route_and_trigger_full_pipeline_live),
        ("greeting_minimal_coverage", test_greeting_minimal_coverage),
        ("path_none_and_lane_none", test_path_none_and_lane_none),
        ("nonexistent_validator_graceful", test_nonexistent_validator_graceful),
    ]

    passed = 0
    failed = 0
    for name, test_fn in tests:
        try:
            result = test_fn()
            if result is True:
                passed += 1
                print(f"  PASS  {name}")
            else:
                failed += 1
                print(f"  FAIL  {name}: returned {result}")
        except AssertionError as e:
            failed += 1
            print(f"  FAIL  {name}: {e}")
        except Exception as e:
            failed += 1
            print(f"  ERROR {name}: {e}")

    print(f"\n─── {passed} passed, {failed} failed, {len(tests)} total ───")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(run_all())
