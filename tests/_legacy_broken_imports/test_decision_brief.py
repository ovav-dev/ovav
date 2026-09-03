"""Eval: Decision Brief — BUILD 11.

Tests DecisionBriefBuilder across ADOPT, ADAPT, REJECT, MONITOR scenarios.
"""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parents[2]
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

from tools.research.benchmark_engine import BenchmarkEngine
from tools.research.decision_brief_builder import ConfidenceLevel, Decision, DecisionBriefBuilder
from tools.research.evidence_scorer import EvidenceItem, EvidenceScorer, EvidenceType


def make_evidence_item(**kwargs) -> EvidenceItem:
    defaults = {
        "claim": "Test",
        "source_title": "Test Source",
        "source_quality_score": 0.8,
        "source_classification": "verified",
        "evidence_type": EvidenceType.DIRECT,
        "directness": 0.8,
        "supports_claim": True,
    }
    defaults.update(kwargs)
    return EvidenceItem(**defaults)


class TestDecisionBrief:
    """Decision brief eval suite."""

    def setup_method(self):
        self.builder = DecisionBriefBuilder()
        self.engine = BenchmarkEngine()
        self.scorer = EvidenceScorer()

    def _make_matrix(self, scores: dict[str, float]) -> tuple:
        """Helper: build engine with scores dict."""
        e = BenchmarkEngine()
        e.add_dimension("quality", weight=1.5)
        e.add_candidate("Candidate")
        for name, s in scores.items():
            e.add_candidate(name)
            e.score(name, "quality", s, confidence=0.9)
        return e.compare()

    def test_adopt_decision(self):
        """High-scoring candidate with strong evidence → ADOPT."""
        matrix = self._make_matrix({"Tool": 0.90})
        report = self.scorer.assess("Is Tool good?", [
            make_evidence_item(source_quality_score=0.90, source_classification="trusted"),
            make_evidence_item(source_quality_score=0.85, source_classification="trusted"),
        ])
        brief = self.builder.build(
            research_question="Is Tool good?",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={"quality": report},
        )
        assert brief.decision == Decision.ADOPT
        assert brief.confidence in (ConfidenceLevel.HIGH, ConfidenceLevel.MEDIUM)

    def test_adapt_decision(self):
        """Moderate score → ADAPT."""
        matrix = self._make_matrix({"Tool": 0.70})
        report = self.scorer.assess("Test", [
            make_evidence_item(source_quality_score=0.65),
        ])
        brief = self.builder.build(
            research_question="Test",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={"quality": report},
        )
        assert brief.decision in (Decision.ADAPT, Decision.MONITOR)

    def test_reject_decision(self):
        """Very low score → REJECT."""
        matrix = self._make_matrix({"Tool": 0.20})
        brief = self.builder.build(
            research_question="Test",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={},
        )
        assert brief.decision == Decision.REJECT

    def test_monitor_decision(self):
        """Borderline score → MONITOR."""
        matrix = self._make_matrix({"Tool": 0.55})
        brief = self.builder.build(
            research_question="Test",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={},
        )
        assert brief.decision == Decision.MONITOR

    def test_brief_has_all_sections(self):
        """Decision brief should have all required sections."""
        matrix = self._make_matrix({"Tool": 0.85})
        report = self.scorer.assess("Test", [
            make_evidence_item(source_classification="trusted"),
        ])
        brief = self.builder.build(
            research_question="Test question?",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={"quality": report},
        )
        assert brief.research_question
        assert brief.decision
        assert brief.confidence
        assert brief.summary
        assert brief.evidence_summary
        assert brief.recommendation_detail
        assert len(brief.risks_and_mitigations) > 0
        assert len(brief.next_steps) > 0

    def test_alternatives_listed(self):
        """When multiple candidates exist, alternatives should be listed."""
        e = BenchmarkEngine()
        e.add_dimension("x", weight=1.0)
        e.add_candidate("Winner")
        e.add_candidate("Loser")
        e.score("Winner", "x", 0.9)
        e.score("Loser", "x", 0.3)
        matrix = e.compare()

        brief = self.builder.build(
            research_question="Test",
            candidates=["Winner", "Loser"],
            benchmark_matrix=matrix,
        )
        assert "Loser" in brief.alternatives

    def test_to_dict_serializable(self):
        """Brief serializes to dict."""
        brief = self.builder.build(
            research_question="Test",
            candidates=["Tool"],
        )
        d = self.builder.to_dict(brief)
        assert isinstance(d, dict)
        assert "decision" in d

    def test_with_source_reports(self):
        """Integration with SourceVerifier reports."""
        from tools.research.source_verifier import SourceMetadata, SourceVerifier
        v = SourceVerifier()
        meta = SourceMetadata(
            title="Good Paper",
            author="Author",
            peer_reviewed=True,
            venue_reputation="nature",
        )
        src_report = v.verify(meta)

        matrix = self._make_matrix({"Tool": 0.85})
        ev_report = self.scorer.assess("Test", [
            make_evidence_item(source_classification="trusted"),
        ])
        brief = self.builder.build(
            research_question="Test",
            candidates=["Tool"],
            benchmark_matrix=matrix,
            evidence_reports={"quality": ev_report},
            source_reports=[src_report],
        )
        assert brief.decision is not None


if __name__ == "__main__":
    t = TestDecisionBrief()
    for name in sorted(dir(t)):
        if name.startswith("test_"):
            t.setup_method()
            method = getattr(t, name)
            try:
                method()
                print(f"  ✅ {name}")
            except Exception as e:
                print(f"  ❌ {name}: {e}")
