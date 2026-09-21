"""Eval: Evidence Scoring — BUILD 11.

Tests EvidenceScorer across strong, moderate, weak, contradictory evidence.
"""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parents[2]
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

from tools.research.evidence_scorer import (
    EvidenceItem,
    EvidenceScorer,
    EvidenceStrength,
    EvidenceType,
)


def make_item(**kwargs) -> EvidenceItem:
    """Create a default EvidenceItem with sensible defaults."""
    defaults = {
        "claim": "Test claim",
        "source_title": "Test Source",
        "source_quality_score": 0.7,
        "source_classification": "verified",
        "evidence_type": EvidenceType.DIRECT,
        "directness": 0.7,
        "supports_claim": True,
    }
    defaults.update(kwargs)
    return EvidenceItem(**defaults)


class TestEvidenceScoring:
    """Evidence scorer eval suite."""

    def setup_method(self):
        self.s = EvidenceScorer()

    def test_no_evidence(self):
        """No evidence → INSUFFICIENT."""
        report = self.s.assess("Claim", [])
        assert report.strength == EvidenceStrength.INSUFFICIENT
        assert report.calibrated_score == 0.0

    def test_single_trusted_source(self):
        """Single trusted source → MODERATE or WEAK."""
        items = [
            make_item(
                source_quality_score=0.90,
                source_classification="trusted",
                evidence_type=EvidenceType.DIRECT,
                directness=0.95,
            ),
        ]
        report = self.s.assess("Claim", items)
        assert report.strength in (EvidenceStrength.MODERATE, EvidenceStrength.WEAK)
        assert report.calibrated_score > 0.30

    def test_multiple_strong_sources(self):
        """Multiple trusted sources → STRONG."""
        items = [
            make_item(source_title="S1", source_quality_score=0.90, source_classification="trusted"),
            make_item(source_title="S2", source_quality_score=0.85, source_classification="trusted"),
            make_item(source_title="S3", source_quality_score=0.88, source_classification="verified"),
        ]
        report = self.s.assess("Claim", items)
        assert report.strength == EvidenceStrength.STRONG
        assert report.calibrated_score > 0.65

    def test_contradiction_reduces_score(self):
        """Contradicting sources should reduce score."""
        items = [
            make_item(source_title="For", source_quality_score=0.80, supports_claim=True),
            make_item(source_title="Against", source_quality_score=0.80, supports_claim=False),
        ]
        report = self.s.assess("Claim", items)
        assert report.contradiction_count == 1
        assert report.calibrated_score < 0.50

    def test_anecdotal_penalty(self):
        """Anecdotal evidence should score lower than direct."""
        direct = self.s.assess("Claim", [
            make_item(evidence_type=EvidenceType.DIRECT, source_quality_score=0.8),
        ])
        anecdotal = self.s.assess("Claim", [
            make_item(evidence_type=EvidenceType.ANECDOTAL, source_quality_score=0.8),
        ])
        assert direct.calibrated_score > anecdotal.calibrated_score

    def test_confidence_interval(self):
        """Confidence interval should be meaningful."""
        items = [
            make_item(source_title="S1"),
            make_item(source_title="S2"),
            make_item(source_title="S3"),
        ]
        report = self.s.assess("Claim", items)
        assert 0.0 <= report.confidence_low <= report.confidence_high <= 1.0
        assert report.confidence_high - report.confidence_low > 0.0

    def test_stale_evidence_flag(self):
        """Old evidence should generate risk flag."""
        items = [make_item(recency_days=800)]
        report = self.s.assess("Claim", items)
        has_stale = any("STALE" in r for r in report.risk_flags)
        assert has_stale

    def test_risk_flags_on_single_source(self):
        """Single source should flag fragility."""
        items = [make_item()]
        report = self.s.assess("Claim", items)
        has_fragile = any("SINGLE_SOURCE" in r or "fragile" in r.lower() for r in report.risk_flags)
        assert has_fragile

    def test_recommendations_for_insufficient(self):
        """INSUFFICIENT evidence should have recommendations."""
        report = self.s.assess("Claim", [])
        assert len(report.recommendations) > 0

    def test_contradiction_warning(self):
        """Contradiction should produce warning."""
        items = [
            make_item(supports_claim=True),
            make_item(supports_claim=False),
        ]
        report = self.s.assess("Claim", items)
        assert len(report.warnings) > 0

    def test_merge_reports(self):
        """Merging reports should combine evidence."""
        r1 = self.s.assess("Claim A", [make_item(source_title="S1")])
        r2 = self.s.assess("Claim A", [make_item(source_title="S2")])
        merged = self.s.merge_reports([r1, r2])
        assert merged.source_count == 2

    def test_to_dict_serializable(self):
        """Report converts to dict."""
        report = self.s.assess("Claim", [make_item()])
        d = self.s.to_dict(report)
        assert isinstance(d, dict)
        assert "calibrated_score" in d


if __name__ == "__main__":
    t = TestEvidenceScoring()
    for name in sorted(dir(t)):
        if name.startswith("test_"):
            t.setup_method()
            method = getattr(t, name)
            try:
                method()
                print(f"  ✅ {name}")
            except Exception as e:
                print(f"  ❌ {name}: {e}")
