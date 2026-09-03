"""Eval: Source Verification — BUILD 11.

Tests SourceVerifier across trusted, uncertain, unreliable, and edge cases.
"""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parents[2]
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

from tools.research.source_verifier import (
    CredibilityTier,
    SourceClassification,
    SourceMetadata,
    SourceVerifier,
)


class TestSourceVerification:
    """Source verification eval suite."""

    def setup_method(self):
        self.v = SourceVerifier()

    def test_trusted_peer_reviewed(self):
        """Peer-reviewed paper with citations → TRUSTED (foundational exemption for age)."""
        meta = SourceMetadata(
            title="Foundational Paper",
            author="Vaswani et al.",
            published_date="2017-06-12",
            peer_reviewed=True,
            citation_count=100000,
            domain_authority="org",
            venue_reputation="nature",
            has_methodology=True,
        )
        report = self.v.verify(meta)
        assert report.classification == SourceClassification.TRUSTED
        # Foundational papers get TRUSTED even with lower overall due to age
        assert report.overall_score >= 0.55
        assert report.credibility_tier in (CredibilityTier.TIER_1, CredibilityTier.TIER_2)

    def test_verified_recent_publication(self):
        """Recent publication with good signals → VERIFIED."""
        meta = SourceMetadata(
            title="Recent Study",
            author="Dr. Researcher",
            published_date="2025-12-01",
            peer_reviewed=True,
            citation_count=15,
            domain_authority="edu",
            venue_reputation="ieee",
        )
        report = self.v.verify(meta)
        assert report.classification in (SourceClassification.VERIFIED, SourceClassification.TRUSTED)
        assert report.freshness_score > 0.6

    def test_uncertain_blog_post(self):
        """Blog without methodology → UNCERTAIN."""
        meta = SourceMetadata(
            title="My Thoughts on AI",
            author="Blogger",
            published_date="2024-06-01",
            domain_authority="com",
            venue_reputation="medium",
        )
        report = self.v.verify(meta)
        assert report.classification in (SourceClassification.UNCERTAIN, SourceClassification.UNRELIABLE)

    def test_unreliable_anonymous(self):
        """Anonymous, no date, no credentials → UNRELIABLE."""
        meta = SourceMetadata(
            title="Hot Take",
            domain_authority="com",
            venue_reputation="twitter",
        )
        report = self.v.verify(meta)
        assert report.classification == SourceClassification.UNRELIABLE
        assert report.overall_score < 0.50

    def test_stale_source(self):
        """Very old source → low freshness."""
        meta = SourceMetadata(
            title="Old Paper",
            author="Old Author",
            published_date="1995-01-01",
            max_age_days=365,
        )
        report = self.v.verify(meta)
        assert report.freshness_score < 0.30
        assert len(report.warnings) > 0

    def test_bias_detection_commercial(self):
        """Sponsored content → bias penalty."""
        meta = SourceMetadata(
            title="Why Our Product Is the Best",
            author="Marketing Team",
            sponsored=True,
            commercial_affiliation=True,
            language_tone="promotional",
        )
        report = self.v.verify(meta)
        assert report.bias_penalty > 0.10
        assert len(report.bias_indicators) > 0

    def test_bias_detection_content_scan(self):
        """Content scanning detects commercial language."""
        meta = SourceMetadata(
            title="Product Review",
            author="Reviewer",
            published_date="2025-01-01",
        )
        report = self.v.verify(meta, content_text="This revolutionary product is unmatched and game-changing!")
        assert report.bias_penalty > 0.05

    def test_contradicted_source(self):
        """Source contradicted by others → penalty."""
        meta = SourceMetadata(
            title="Controversial Claim",
            author="Author",
            contradicted_by=["Study A", "Study B", "Study C"],
        )
        report = self.v.verify(meta)
        assert len(report.warnings) > 0

    def test_corroborated_source(self):
        """Source corroborated by others → bonus."""
        meta = SourceMetadata(
            title="Well-Supported Claim",
            author="Author",
            corroborated_by=["Study X", "Study Y", "Study Z"],
        )
        report = self.v.verify(meta)
        assert report.corroboration_bonus > 0.0

    def test_batch_verification_ranking(self):
        """Batch verification should rank trusted above unreliable."""
        trusted = SourceMetadata(
            title="Good", author="A", peer_reviewed=True, venue_reputation="nature"
        )
        unreliable = SourceMetadata(
            title="Bad", venue_reputation="reddit"
        )
        reports = self.v.verify_batch([(trusted, ""), (unreliable, "")])
        assert reports[0].overall_score >= reports[1].overall_score

    def test_report_to_dict_serializable(self):
        """Report converts to dict cleanly."""
        meta = SourceMetadata(title="Test")
        report = self.v.verify(meta)
        d = self.v.report_to_dict(report)
        assert isinstance(d, dict)
        assert "classification" in d
        assert "overall_score" in d

    def test_summary_statistics(self):
        """Summary generates correct statistics."""
        reports = self.v.verify_batch([
            (SourceMetadata(title="A", peer_reviewed=True, venue_reputation="nature"), ""),
            (SourceMetadata(title="B", venue_reputation="medium"), ""),
        ])
        summary = self.v.summary(reports)
        assert summary["count"] == 2
        assert "by_classification" in summary


if __name__ == "__main__":
    t = TestSourceVerification()
    for name in sorted(dir(t)):
        if name.startswith("test_"):
            t.setup_method()
            method = getattr(t, name)
            try:
                method()
                print(f"  ✅ {name}")
            except Exception as e:
                print(f"  ❌ {name}: {e}")
