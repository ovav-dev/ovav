"""S83 Model Body Router — Layer 3 test suite.

Tests for provider_abstraction, identity_guard, and model_body_router.
"""

from __future__ import annotations

import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))


# ═══ provider_abstraction tests ═══════════════════════════════════════════

def test_detect_opencode_model():
    """Model in opencode known_models maps to opencode family."""
    from tools.agent_runtime.provider_abstraction import detect_current_provider
    r = detect_current_provider(model_id="deepseek-v4-pro")
    assert r["provider_family"] == "opencode"
    assert r["model_tier"] == "lead"
    assert r["confidence"] == "high"
    return True


def test_detect_opencode_with_prefix():
    """Prefixed model ID is correctly stripped."""
    from tools.agent_runtime.provider_abstraction import detect_current_provider
    r = detect_current_provider(model_id="opencode-go/deepseek-v4-pro")
    assert r["provider_family"] == "opencode"
    assert r["model_tier"] == "lead"
    assert r["confidence"] == "high"
    return True


def test_detect_unknown_legacy_model():
    """Legacy providers (openai, anthropic, google) are unknown."""
    from tools.agent_runtime.provider_abstraction import detect_current_provider
    for legacy_id in ("gpt-5.5", "claude-sonnet-4-20250514", "gemini-2.5-pro"):
        r = detect_current_provider(model_id=legacy_id)
        assert r["provider_family"] == "unknown", f"{legacy_id} should be unknown"
        assert r["model_tier"] == "unknown"
    return True


def test_detect_unknown():
    from tools.agent_runtime.provider_abstraction import detect_current_provider
    r = detect_current_provider(model_id="some-fake-model-xyz")
    assert r["provider_family"] == "unknown"
    assert r["confidence"] == "low"
    return True


def test_detect_none_defaults_opencode():
    """No model_id defaults to opencode (sole authorized provider)."""
    from tools.agent_runtime.provider_abstraction import detect_current_provider
    r = detect_current_provider(model_id=None)
    assert r["provider_family"] == "opencode"
    return True


def test_build_ladder_opencode():
    from tools.agent_runtime.provider_abstraction import build_provider_ladder
    l = build_provider_ladder(preferred_family="opencode", required_tier="lead")
    assert len(l) > 0
    assert l[0]["provider_family"] == "opencode"
    return True


def test_build_ladder_exclude():
    from tools.agent_runtime.provider_abstraction import build_provider_ladder
    l = build_provider_ladder(
        preferred_family="opencode",
        required_tier="lead",
        exclude_families=["opencode"],
    )
    families = {m["provider_family"] for m in l}
    assert "opencode" not in families
    return True


def test_match_capability_pass():
    from tools.agent_runtime.provider_abstraction import match_capability_to_task
    r = match_capability_to_task(provider_family="opencode", task_requires_tools=True)
    assert r["capability_match"]
    return True


def test_match_capability_fail():
    from tools.agent_runtime.provider_abstraction import match_capability_to_task
    r = match_capability_to_task(provider_family="unknown", task_requires_tools=True)
    assert not r["capability_match"]
    return True


def test_resolve_capabilities():
    from tools.agent_runtime.provider_abstraction import resolve_provider_capabilities
    r = resolve_provider_capabilities("opencode")
    assert "tool_use" in r["capabilities"]
    assert len(r["known_models"]) == 14
    return True


def test_provider_family_for_legacy_unknown():
    """Legacy provider models map to unknown."""
    from tools.agent_runtime.provider_abstraction import provider_family_for
    assert provider_family_for("gpt-5.5") == "unknown"
    assert provider_family_for("claude-sonnet-4-20250514") == "unknown"
    return True


def test_provider_family_for_opencode_models():
    """OpenCode models map to opencode family."""
    from tools.agent_runtime.provider_abstraction import provider_family_for
    assert provider_family_for("deepseek-v4-pro") == "opencode"
    assert provider_family_for("opencode-go/deepseek-v4-pro") == "opencode"
    return True


# ═══ identity_guard tests ══════════════════════════════════════════════════

def test_capture_snapshot():
    from tools.agent_runtime.identity_guard import capture_identity_snapshot
    r = capture_identity_snapshot(profile="thavren")
    assert r["status"] == "ok"
    assert len(r["snapshot"]["packet_hash"]) == 64
    return True


def test_verify_identity_match():
    from tools.agent_runtime.identity_guard import (
        capture_identity_snapshot,
        verify_identity_post_switch,
    )
    pre = capture_identity_snapshot(profile="thavren")
    post = verify_identity_post_switch(pre_snapshot=pre, profile="thavren")
    assert post["identity_preserved"]
    assert post["hash_match"]
    assert post["status"] == "pass"
    return True


def test_verify_nonexistent_profile():
    from tools.agent_runtime.identity_guard import capture_identity_snapshot
    r = capture_identity_snapshot(profile="nonexistent")
    assert r["status"] == "error"
    return True


def test_detect_identity_drift_none():
    from tools.agent_runtime.identity_guard import capture_identity_snapshot, detect_identity_drift
    pre = capture_identity_snapshot(profile="thavren")
    # Same snapshot → no drift
    drift = detect_identity_drift(pre_snapshot=pre, post_snapshot=pre)
    assert len(drift) == 0
    return True


def test_forbidden_mutations_defined():
    from tools.agent_runtime.identity_guard import FORBIDDEN_MUTATIONS
    assert len(FORBIDDEN_MUTATIONS) >= 5
    critical = [m for m in FORBIDDEN_MUTATIONS if m["severity"] == "critical"]
    assert len(critical) >= 3
    return True


# ═══ model_body_router tests ═══════════════════════════════════════════════

def test_detect_switch_token_exhaustion():
    from tools.agent_runtime.model_body_router import detect_switch_needed
    r = detect_switch_needed(
        current_model_id="deepseek-v4-pro",
        token_budget_remaining=500,
        token_budget_allocated=10000,
    )
    assert r["switch_recommended"]
    assert r["urgency"] in ("medium", "high")
    return True


def test_detect_switch_no_need():
    from tools.agent_runtime.model_body_router import detect_switch_needed
    r = detect_switch_needed(
        current_model_id="deepseek-v4-pro",
        token_budget_remaining=9000,
        token_budget_allocated=10000,
        consecutive_errors=0,
    )
    assert not r["switch_recommended"]
    return True


def test_detect_switch_consecutive_errors():
    from tools.agent_runtime.model_body_router import detect_switch_needed
    r = detect_switch_needed(
        consecutive_errors=3,
        token_budget_remaining=9000,
        token_budget_allocated=10000,
    )
    assert r["switch_recommended"]
    return True


def test_detect_switch_credit_constrained():
    from tools.agent_runtime.model_body_router import detect_switch_needed
    r = detect_switch_needed(credit_mode="constrained")
    assert r["switch_recommended"]
    return True


def test_build_fallback_ladder():
    from tools.agent_runtime.model_body_router import build_fallback_ladder
    r = build_fallback_ladder(current_model_id="deepseek-v4-pro", required_tier="lead")
    assert len(r["fallback_ladder"]) > 0
    assert r["candidate_count"] > 0
    return True


def test_build_fallback_ladder_constrained():
    from tools.agent_runtime.model_body_router import build_fallback_ladder
    r = build_fallback_ladder(
        current_model_id="deepseek-v4-pro",
        required_tier="lead",
        credit_mode="constrained",
    )
    assert len(r["fallback_ladder"]) > 0
    assert r["effective_tier"] != "lead"  # should step down
    return True


def test_full_switch_cycle_no_switch_needed():
    from tools.agent_runtime.model_body_router import full_switch_cycle
    r = full_switch_cycle(
        current_model_id="deepseek-v4-pro",
        token_budget_remaining=9000,
        token_budget_allocated=10000,
        dry_run=True,
    )
    assert r["status"] == "ok"
    assert not r["switch_executed"]
    return True


def test_full_switch_cycle_dry_run():
    from tools.agent_runtime.model_body_router import full_switch_cycle
    r = full_switch_cycle(
        current_model_id="deepseek-v4-pro",
        token_budget_remaining=500,
        token_budget_allocated=10000,
        dry_run=True,
    )
    assert r["status"] in ("pass", "safe_stop")
    return True


def test_safe_stop_if_exhausted():
    from tools.agent_runtime.model_body_router import safe_stop_if_exhausted
    r = safe_stop_if_exhausted(
        tried_models=["m1", "m2", "m3"],
        tried_count=3,
        reason="all_bodies_exhausted",
    )
    assert r["status"] == "safe_stop"
    assert r["exhausted"]
    return True


def test_safe_stop_reasons_defined():
    from tools.agent_runtime.model_body_router import SAFE_STOP_REASONS
    assert len(SAFE_STOP_REASONS) >= 4
    return True


def test_full_switch_cycle_no_ladder():
    """When no suitable models exist, should safe stop."""
    from tools.agent_runtime.model_body_router import full_switch_cycle
    # Force token exhaustion on an unknown model
    r = full_switch_cycle(
        current_model_id="some-fake-model-xyz",
        token_budget_remaining=100,
        token_budget_allocated=10000,
        dry_run=True,
        max_attempts=1,
    )
    # Should either pass (via fallback ladder) or safe_stop
    assert r["status"] in ("pass", "safe_stop")
    return True


def run_all() -> int:
    tests = [
        ("detect_opencode_model", test_detect_opencode_model),
        ("detect_opencode_with_prefix", test_detect_opencode_with_prefix),
        ("detect_unknown_legacy_model", test_detect_unknown_legacy_model),
        ("detect_unknown", test_detect_unknown),
        ("detect_none_defaults_opencode", test_detect_none_defaults_opencode),
        ("build_ladder_opencode", test_build_ladder_opencode),
        ("build_ladder_exclude", test_build_ladder_exclude),
        ("match_capability_pass", test_match_capability_pass),
        ("match_capability_fail", test_match_capability_fail),
        ("resolve_capabilities", test_resolve_capabilities),
        ("provider_family_for_legacy_unknown", test_provider_family_for_legacy_unknown),
        ("provider_family_for_opencode_models", test_provider_family_for_opencode_models),
        ("capture_snapshot", test_capture_snapshot),
        ("verify_identity_match", test_verify_identity_match),
        ("verify_nonexistent_profile", test_verify_nonexistent_profile),
        ("detect_identity_drift_none", test_detect_identity_drift_none),
        ("forbidden_mutations_defined", test_forbidden_mutations_defined),
        ("detect_switch_token_exhaustion", test_detect_switch_token_exhaustion),
        ("detect_switch_no_need", test_detect_switch_no_need),
        ("detect_switch_consecutive_errors", test_detect_switch_consecutive_errors),
        ("detect_switch_credit_constrained", test_detect_switch_credit_constrained),
        ("build_fallback_ladder", test_build_fallback_ladder),
        ("build_fallback_ladder_constrained", test_build_fallback_ladder_constrained),
        ("full_switch_cycle_no_switch_needed", test_full_switch_cycle_no_switch_needed),
        ("full_switch_cycle_dry_run", test_full_switch_cycle_dry_run),
        ("safe_stop_if_exhausted", test_safe_stop_if_exhausted),
        ("safe_stop_reasons_defined", test_safe_stop_reasons_defined),
        ("full_switch_cycle_no_ladder", test_full_switch_cycle_no_ladder),
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
