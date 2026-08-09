"""M1 evals for the source-local Global Control Bridge."""

from __future__ import annotations

import json
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from tools.harnesses.global_control_bridge_m1 import (
    GlobalControlRequest,
    build_m1_report,
    evaluate_request,
)


def test_global_opencode_config_write_is_blocked_even_with_consent():
    decision = evaluate_request(GlobalControlRequest("opencode_global_config", "write", "~/.config/opencode/opencode.json", consent=True))
    assert decision["decision"] == "blocked"
    assert decision["global_writes_performed"] is False
    assert "consent_does_not_override_blocked_surface" in decision["issues"]


def test_install_plan_is_plan_only_without_global_apply():
    decision = evaluate_request(GlobalControlRequest("install_apply_deploy", "plan", "source_to_global", consent=True))
    assert decision["decision"] == "plan_only"
    assert decision["global_writes_performed"] is False
    assert decision["home_writes_performed"] is False


def test_install_apply_remains_blocked():
    decision = evaluate_request(GlobalControlRequest("install_apply_deploy", "apply", "source_to_global", consent=True, dry_run=False))
    assert decision["decision"] == "blocked"
    assert "mutating_global_action_blocked:apply" in decision["issues"]


def test_source_local_artifact_validation_is_allowed():
    decision = evaluate_request(GlobalControlRequest("source_local_artifact", "validate", ".ovav/artifacts/M1"))
    assert decision["decision"] == "source_local_allowed"
    assert decision["status"] == "pass"


def test_aggregate_report_keeps_blocked_surface_invariants():
    report = build_m1_report()
    serialized = json.dumps(report).lower()
    assert report["status"] == "pass"
    assert report["safety_invariants"]["global_writes_performed"] is False
    assert report["safety_invariants"]["home_writes_performed"] is False
    assert "source_local_control_bridge" in serialized
    assert "production_global_ready_claim" in serialized


if __name__ == "__main__":
    test_global_opencode_config_write_is_blocked_even_with_consent()
    test_install_plan_is_plan_only_without_global_apply()
    test_install_apply_remains_blocked()
    test_source_local_artifact_validation_is_allowed()
    test_aggregate_report_keeps_blocked_surface_invariants()
    print("PASS M1 Global Control Bridge evals")
