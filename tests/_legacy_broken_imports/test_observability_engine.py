#!/usr/bin/env python3
"""Tests for Layer 4 — Observability Engine + Evidence Writer.

Covers: trace_event, enrich_from_capsule, sanitize_trace, summary_for_human,
validate_trace, write_evidence, resolve_mode, has_disk_space, hash_index ops.
"""

from __future__ import annotations

import json
import sys
from datetime import datetime
from pathlib import Path

import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
AGENT_RUNTIME = REPO_ROOT / "tools" / "agent_runtime"
sys.path.insert(0, str(AGENT_RUNTIME))

import evidence_writer as writer
import observability_engine as engine

# ============================================================================
# trace_event
# ============================================================================

class TestTraceEvent:
    """trace_event() creates valid trace dicts."""

    def test_minimal_trace_has_required_fields(self):
        trace = engine.trace_event()
        required = ["trace_id", "timestamp", "service_area", "lead", "mode",
                    "event_type", "delivery_contract"]
        for field in required:
            assert field in trace, f"missing {field}"
        assert trace["trace_id"].startswith("trace_")

    def test_trace_id_is_unique(self):
        t1 = engine.trace_event()
        t2 = engine.trace_event()
        assert t1["trace_id"] != t2["trace_id"]

    def test_timestamp_is_iso8601(self):
        trace = engine.trace_event()
        # Should parse without error
        datetime.fromisoformat(trace["timestamp"])

    def test_defaults_are_sane(self):
        trace = engine.trace_event()
        assert trace["service_area"] == "platform_engineering"
        assert trace["lead"] == "thavren"
        assert trace["mode"] == "repo_local_engineering"
        assert trace["event_type"] == "decision"
        assert trace["decision"] == "allow"
        assert trace["model_used"] == "not_selected"
        assert trace["token_usage_estimate"] == 0
        assert trace["cost_estimate"] == 0.0

    def test_custom_fields_override_defaults(self):
        trace = engine.trace_event(
            service_area="research_intelligence",
            lead="eidren",
            mode="repo_local_research",
            event_type="context.request",
            decision="deny",
            model_used="claude-opus",
            token_usage_estimate=1500,
            cost_estimate=0.03,
        )
        assert trace["service_area"] == "research_intelligence"
        assert trace["lead"] == "eidren"
        assert trace["mode"] == "repo_local_research"
        assert trace["event_type"] == "context.request"
        assert trace["decision"] == "deny"
        assert trace["model_used"] == "claude-opus"
        assert trace["token_usage_estimate"] == 1500
        assert trace["cost_estimate"] == 0.03

    def test_decision_packet_included(self):
        trace = engine.trace_event(
            decision_packet={"reason": "fail_closed", "evidence": "none"}
        )
        assert trace["decision_packet"] == {"reason": "fail_closed", "evidence": "none"}

    def test_arrays_are_empty_by_default(self):
        trace = engine.trace_event()
        assert trace["source_requests"] == []
        assert trace["context_decisions"] == []
        assert trace["tool_calls"] == []
        assert trace["handoffs"] == []
        assert trace["evals_passed"] == []
        assert trace["evals_failed"] == []

    def test_arrays_accept_data(self):
        trace = engine.trace_event(
            source_requests=[{"path": "docs/test.md", "context_class": "L1"}],
            tool_calls=[{"tool": "bash", "parameters_sanitized": True}],
            handoffs=[{"target": "eidren", "sanitized": True}],
            evals_passed=["research_no_repo_default"],
            evals_failed=["budget_route_respected"],
        )
        assert len(trace["source_requests"]) == 1
        assert len(trace["tool_calls"]) == 1
        assert len(trace["handoffs"]) == 1
        assert trace["evals_passed"] == ["research_no_repo_default"]
        assert trace["evals_failed"] == ["budget_route_respected"]

    def test_capsule_id_stored(self):
        trace = engine.trace_event(capsule_id="019e7140-101c-7f2c-8c29-961bb3f01bf7")
        assert trace["capsule_id"] == "019e7140-101c-7f2c-8c29-961bb3f01bf7"

    def test_delivery_contract_accepted(self):
        trace = engine.trace_event(delivery_contract="safe_stop_report")
        assert trace["delivery_contract"] == "safe_stop_report"


# ============================================================================
# enrich_from_capsule
# ============================================================================

class TestEnrichFromCapsule:
    """enrich_from_capsule() adds capsule context."""

    def test_enrich_sets_capsule_id_when_missing(self):
        trace = engine.trace_event()
        capsule = {"capsule_id": "019e7140-101c-7f2c-8c29-961bb3f01bf7"}
        enriched = engine.enrich_from_capsule(trace, capsule)
        assert enriched["capsule_id"] == "019e7140-101c-7f2c-8c29-961bb3f01bf7"

    def test_enrich_does_not_overwrite_explicit_values(self):
        trace = engine.trace_event(
            service_area="research_intelligence",
            lead="eidren",
            model_used="claude-opus",
        )
        capsule = {
            "service_area": "platform_engineering",
            "lead": "thavren",
            "model_body": "deepseek",
        }
        enriched = engine.enrich_from_capsule(trace, capsule)
        assert enriched["service_area"] == "research_intelligence"  # kept
        assert enriched["lead"] == "eidren"  # kept
        assert enriched["model_used"] == "claude-opus"  # kept

    def test_enrich_fills_defaults_from_capsule(self):
        trace = engine.trace_event()  # all defaults
        capsule = {
            "service_area": "research_intelligence",
            "lead": "eidren",
            "mode": "repo_local_research",
            "model_body": "gemini-pro",
        }
        enriched = engine.enrich_from_capsule(trace, capsule)
        assert enriched["service_area"] == "research_intelligence"
        assert enriched["lead"] == "eidren"
        assert enriched["mode"] == "repo_local_research"
        assert enriched["model_used"] == "gemini-pro"

    def test_enrich_no_capsule_returns_unchanged(self):
        trace = engine.trace_event()
        enriched = engine.enrich_from_capsule(trace, None)
        assert enriched == trace

    def test_enrich_nested_state_dict(self):
        trace = engine.trace_event()
        capsule = {"state": {"capsule_id": "abc-123"}}
        enriched = engine.enrich_from_capsule(trace, capsule)
        assert enriched["capsule_id"] == "abc-123"


# ============================================================================
# sanitize_trace
# ============================================================================

class TestSanitizeTrace:
    """sanitize_trace() strips secrets."""

    def test_sanitize_flags_trace(self):
        trace = engine.trace_event()
        sanitized = engine.sanitize_trace(trace)
        assert sanitized["sanitized"] is True
        assert trace["sanitized"] is False  # original unchanged

    def test_sanitize_strips_token_in_decision_packet(self):
        trace = engine.trace_event(
            decision_packet={"api_key": "sk-abc123", "reason": "ok"}
        )
        sanitized = engine.sanitize_trace(trace)
        assert sanitized["decision_packet"]["api_key"] == "[REDACTED]"
        assert sanitized["decision_packet"]["reason"] == "ok"

    def test_sanitize_strips_password_in_value(self):
        trace = engine.trace_event(
            decision_packet={"note": "my password is hunter2"}
        )
        sanitized = engine.sanitize_trace(trace)
        assert sanitized["decision_packet"]["note"] == "[REDACTED]"

    def test_sanitize_strips_secret_in_tool_calls(self):
        trace = engine.trace_event(
            tool_calls=[{"tool": "bash", "token_param": "ghp_12345"}]
        )
        sanitized = engine.sanitize_trace(trace)
        assert sanitized["tool_calls"][0]["token_param"] == "[REDACTED]"

    def test_sanitize_handles_empty_trace(self):
        trace = engine.trace_event()
        sanitized = engine.sanitize_trace(trace)
        assert sanitized["sanitized"] is True


# ============================================================================
# summary_for_human
# ============================================================================

class TestSummaryForHuman:
    """summary_for_human() produces compact output."""

    def test_summary_has_trace_id(self):
        trace = engine.trace_event()
        summary = engine.summary_for_human(trace)
        assert trace["trace_id"][:18] in summary

    def test_summary_has_decision(self):
        trace = engine.trace_event(decision="deny")
        summary = engine.summary_for_human(trace)
        assert "deny" in summary

    def test_summary_has_service_context(self):
        trace = engine.trace_event(service_area="research_intelligence", lead="eidren")
        summary = engine.summary_for_human(trace)
        assert "research_intelligence" in summary
        assert "eidren" in summary

    def test_summary_is_compact(self):
        trace = engine.trace_event()
        summary = engine.summary_for_human(trace)
        lines = summary.strip().split("\n")
        assert 2 <= len(lines) <= 5  # 3-5 lines expected

    def test_summary_shows_evals(self):
        trace = engine.trace_event(
            evals_passed=["test_a"],
            evals_failed=["test_b"],
        )
        summary = engine.summary_for_human(trace)
        assert "FAILED" in summary

    def test_summary_shows_token_cost(self):
        trace = engine.trace_event(token_usage_estimate=500, cost_estimate=0.01)
        summary = engine.summary_for_human(trace)
        assert "500 tokens" in summary
        assert "0.0100" in summary

    def test_summary_no_evidence_message(self):
        trace = engine.trace_event()
        summary = engine.summary_for_human(trace)
        assert "no evidence recorded" in summary


# ============================================================================
# validate_trace
# ============================================================================

class TestValidateTrace:
    """validate_trace() checks against schema."""

    def test_valid_trace_passes(self):
        trace = engine.trace_event()
        result = engine.validate_trace(trace)
        assert result["valid"] is True
        assert result["errors"] == []

    def test_invalid_service_area_fails(self):
        trace = engine.trace_event(service_area="invalid_area")
        result = engine.validate_trace(trace)
        assert result["valid"] is False
        assert any("service_area" in e for e in result["errors"])

    def test_invalid_lead_fails(self):
        trace = engine.trace_event(lead="unknown")
        result = engine.validate_trace(trace)
        assert result["valid"] is False
        assert any("lead" in e for e in result["errors"])

    def test_invalid_event_type_fails(self):
        trace = engine.trace_event(event_type="invalid_event")
        result = engine.validate_trace(trace)
        assert result["valid"] is False
        assert any("event_type" in e for e in result["errors"])

    def test_missing_required_field_fails(self):
        trace = engine.trace_event()
        del trace["trace_id"]
        result = engine.validate_trace(trace)
        assert result["valid"] is False
        assert any("trace_id" in e for e in result["errors"])


# ============================================================================
# write_evidence
# ============================================================================

class TestWriteEvidence:
    """write_evidence() writes traces to disk."""

    def test_write_creates_file(self):
        trace = engine.trace_event()
        result = writer.write_evidence(trace)
        assert result["status"] == "ok"
        assert result["path"] is not None
        assert Path(result["path"]).exists()

    def test_written_file_is_valid_json(self):
        trace = engine.trace_event(decision="deny")
        result = writer.write_evidence(trace)
        content = Path(result["path"]).read_text()
        parsed = json.loads(content)
        assert parsed["decision"] == "deny"
        assert parsed["sanitized"] is True

    def test_check_mode_does_not_write(self):
        trace = engine.trace_event()
        result = writer.write_evidence(trace, evidence_mode="check")
        assert result["status"] == "checked"
        # Path may point to a non-existent file in check mode

    def test_trace_marked_written_after_write(self):
        trace = engine.trace_event()
        writer.write_evidence(trace)
        assert trace["evidence_written"] is True
        assert trace["evidence_path"] is not None
        assert trace["sanitized"] is True

    def test_sanitized_before_write(self):
        trace = engine.trace_event(
            decision_packet={"api_key": "secret-123"}
        )
        result = writer.write_evidence(trace)
        content = Path(result["path"]).read_text()
        assert "secret-123" not in content
        assert "[REDACTED]" in content


# ============================================================================
# resolve_mode
# ============================================================================

class TestResolveMode:
    """resolve_mode() handles env var and explicit args."""

    def test_default_is_write(self):
        assert writer.resolve_mode() == "write"

    def test_explicit_check(self):
        assert writer.resolve_mode("check") == "check"

    def test_explicit_strict(self):
        assert writer.resolve_mode("strict") == "strict"

    def test_invalid_falls_back_to_write(self):
        assert writer.resolve_mode("invalid") == "write"

    def test_env_var_overrides(self, monkeypatch):
        monkeypatch.setenv("OVAV_EVIDENCE_MODE", "check")
        from importlib import reload
        reload(writer)
        assert writer.EVIDENCE_MODE == "check"
        monkeypatch.delenv("OVAV_EVIDENCE_MODE", raising=False)
        reload(writer)


# ============================================================================
# has_disk_space
# ============================================================================

class TestHasDiskSpace:
    """has_disk_space() checks available storage."""

    def test_returns_bool(self):
        result = writer.has_disk_space()
        assert isinstance(result, bool)

    def test_high_threshold_returns_false_when_space_low(self, monkeypatch):
        # Mock shutil.disk_usage to return low space
        def mock_usage(path):
            return type("Usage", (), {"free": 512})()  # 512 bytes free
        monkeypatch.setattr("shutil.disk_usage", mock_usage)
        from importlib import reload
        reload(writer)
        assert writer.has_disk_space(min_bytes=1_000_000) is False
        reload(writer)


# ============================================================================
# hash index
# ============================================================================

class TestHashIndex:
    """Hash index for evidence drift detection."""

    def test_load_empty_index(self, tmp_path):
        import evidence_writer as ew

        # Use a completely fresh tmp_path for isolation
        idx_path = tmp_path / ".hash_index.json"
        original = ew.HASH_INDEX_PATH
        try:
            ew.HASH_INDEX_PATH = idx_path
            index = ew.load_hash_index()
            assert index == {}
        finally:
            ew.HASH_INDEX_PATH = original

    def test_save_and_load_index(self, tmp_path):
        monkeypatch = pytest.MonkeyPatch()
        idx_path = tmp_path / ".hash_index.json"
        monkeypatch.setattr(writer, "HASH_INDEX_PATH", idx_path)
        from importlib import reload
        reload(writer)
        writer.save_hash_index({"test": "abc123"})
        loaded = writer.load_hash_index()
        assert loaded == {"test": "abc123"}
        reload(writer)
