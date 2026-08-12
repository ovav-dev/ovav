"""Evidence Scorer — BUILD 11 Research Pipeline.

Calibrates evidence strength from multiple sources with confidence intervals.
Produces EvidenceReport with calibrated score, corroboration map, and risk flags.

Source-local: operates on evidence items with source quality already verified.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from enum import Enum


class EvidenceStrength(str, Enum):
    STRONG = "strong"           # Multiple TRUSTED/VERIFIED sources, direct evidence
    MODERATE = "moderate"       # Mixed sources, some indirect evidence
    WEAK = "weak"               # Single source, indirect, or low credibility
    INSUFFICIENT = "insufficient"  # Too few sources or contradictory
    CONTRADICTORY = "contradictory"  # Sources disagree fundamentally


class EvidenceType(str, Enum):
    DIRECT = "direct"               # Primary measurement, reproduction, benchmark
    INDIRECT = "indirect"           # Inference, analogy, related study
    ANECDOTAL = "anecdotal"         # Personal experience, case study, testimonial
    THEORETICAL = "theoretical"     # Mathematical proof, logical deduction
    EXPERT_OPINION = "expert_opinion"  # Authority statement without data
    ABSENT = "absent"               # Claimed but no evidence provided


@dataclass
class EvidenceItem:
    """One piece of evidence from a source."""
    claim: str                      # The claim being supported/refuted
    source_title: str               # Link to SourceVerifier output
    source_quality_score: float     # From SourceQualityReport.overall_score
    source_classification: str      # trusted/verified/uncertain/unreliable
    evidence_type: EvidenceType
    supports_claim: bool = True     # True = supports, False = contradicts
    directness: float = 0.5         # 0 = indirect/hand-wavy, 1 = direct measurement
    recency_days: int = 0           # Days since evidence was published
    notes: str = ""


@dataclass
class EvidenceReport:
    """Calibrated evidence assessment for a claim or comparison."""
    claim: str
    evidence_items: list[EvidenceItem] = field(default_factory=list)
    strength: EvidenceStrength = EvidenceStrength.INSUFFICIENT

    # Calibrated scores
    calibrated_score: float = 0.0           # 0.0 – 1.0
    confidence_low: float = 0.0             # Lower bound of confidence interval
    confidence_high: float = 0.0            # Upper bound of confidence interval
    corroboration_count: int = 0
    contradiction_count: int = 0
    source_count: int = 0

    # Risk flags
    risk_flags: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    recommendations: list[str] = field(default_factory=list)

    generated_at: str = field(default_factory=lambda: datetime.now(UTC).isoformat())


class EvidenceScorer:
    """Calibrates evidence from multiple sources into a scored assessment.

    Weights evidence by source quality, directness, recency, and corroboration.
    Produces confidence intervals using a simple resampling-inspired approach.

    Usage:
        scorer = EvidenceScorer()
        report = scorer.assess("Tool X is faster than Tool Y", [
            EvidenceItem(
                claim="Tool X is faster than Tool Y",
                source_title="Benchmark Study 2024",
                source_quality_score=0.85,
                source_classification="trusted",
                evidence_type=EvidenceType.DIRECT,
                directness=0.9,
            ),
            EvidenceItem(
                claim="Tool X is faster than Tool Y",
                source_title="Community Report",
                source_quality_score=0.55,
                source_classification="uncertain",
                evidence_type=EvidenceType.INDIRECT,
                directness=0.3,
            ),
        ])
        print(report.calibrated_score)   # e.g., 0.72
    """

    # Weights for composite score
    _WEIGHT_SOURCE_QUALITY = 0.35
    _WEIGHT_DIRECTNESS = 0.30
    _WEIGHT_RECENCY = 0.15
    _WEIGHT_CORROBORATION = 0.20

    # Minimum items for reliability
    _MIN_ITEMS_MODERATE = 2
    _MIN_ITEMS_STRONG = 3

    # Decay
    _RECENCY_HALF_LIFE_DAYS = 365


    def assess(self, claim: str, evidence_items: list[EvidenceItem]) -> EvidenceReport:
        """Assess evidence strength for a claim.

        Args:
            claim: The statement being evaluated.
            evidence_items: List of evidence items from verified sources.

        Returns:
            EvidenceReport with calibrated score, confidence intervals, and risk flags.
        """
        if not evidence_items:
            return EvidenceReport(
                claim=claim,
                strength=EvidenceStrength.INSUFFICIENT,
                warnings=["No evidence provided for this claim."],
                recommendations=["Gather at least 2 independent sources before making claims."],
            )

        report = EvidenceReport(claim=claim, evidence_items=evidence_items)

        # Separate support vs contradiction
        supporting = [e for e in evidence_items if e.supports_claim]
        contradicting = [e for e in evidence_items if not e.supports_claim]

        report.corroboration_count = len(supporting)
        report.contradiction_count = len(contradicting)
        report.source_count = len(evidence_items)

        # 1. Per-item scores
        item_scores = []
        for item in evidence_items:
            s = self._item_score(item)
            item_scores.append(s)

        # 2. Aggregate
        if supporting:
            sup_scores = [self._item_score(e) for e in supporting]
            avg_support = sum(sup_scores) / len(sup_scores)
        else:
            avg_support = 0.0

        if contradicting:
            con_scores = [self._item_score(e) for e in contradicting]
            avg_contra = sum(con_scores) / len(con_scores)
            report.warnings.append(
                f"Claim is contradicted by {len(contradicting)} source(s). "
                f"Contradiction reduces confidence significantly."
            )
        else:
            avg_contra = 0.0

        # Net score: support weighted by count, minus contradiction penalty
        net_score = (
            avg_support * (len(supporting) / max(1, len(evidence_items)))
            - avg_contra * (len(contradicting) / max(1, len(evidence_items))) * 1.5  # contradiction penalty
        )
        net_score = max(0.0, min(1.0, net_score))

        report.calibrated_score = round(net_score, 3)

        # 3. Confidence interval
        report.confidence_low, report.confidence_high = self._confidence_interval(
            item_scores, net_score
        )

        # 4. Strength classification
        report.strength = self._classify_strength(
            net_score,
            report.corroboration_count,
            report.contradiction_count,
            supporting,
        )

        # 5. Risk flags
        report.risk_flags = self._detect_risks(report, supporting, contradicting)

        # 6. Recommendations
        report.recommendations = self._generate_recommendations(report)

        return report


    def _item_score(self, item: EvidenceItem) -> float:
        """Score a single evidence item (0.0–1.0)."""
        import math
        score = (
            self._WEIGHT_SOURCE_QUALITY * item.source_quality_score +
            self._WEIGHT_DIRECTNESS * item.directness +
            self._WEIGHT_RECENCY * math.exp(
                -math.log(2) * item.recency_days / self._RECENCY_HALF_LIFE_DAYS
            )
        )
        # Type bonus/penalty
        type_modifiers = {
            EvidenceType.DIRECT: 0.10,
            EvidenceType.INDIRECT: 0.00,
            EvidenceType.THEORETICAL: 0.00,
            EvidenceType.EXPERT_OPINION: -0.05,
            EvidenceType.ANECDOTAL: -0.10,
            EvidenceType.ABSENT: -0.20,
        }
        score += type_modifiers.get(item.evidence_type, 0.0)
        return max(0.0, min(1.0, score))


    def _confidence_interval(
        self, scores: list[float], net_score: float
    ) -> tuple[float, float]:
        """Estimate confidence interval using score variance."""
        if not scores:
            return 0.0, 0.0
        if len(scores) == 1:
            # Single source: wide interval
            return max(0.0, net_score - 0.25), min(1.0, net_score + 0.25)
        mean = sum(scores) / len(scores)
        variance = sum((s - mean) ** 2 for s in scores) / len(scores)
        import math
        std = math.sqrt(variance)
        # Minimum uncertainty floor: even identical sources have some uncertainty
        min_margin = 0.02 * (1.0 / math.sqrt(len(scores)))
        # Wider interval for fewer items
        expansion = 1.5 if len(scores) < 3 else 1.0
        margin = max(min_margin, std * expansion * 1.96 / math.sqrt(len(scores)))
        return (
            round(max(0.0, net_score - margin), 3),
            round(min(1.0, net_score + margin), 3),
        )


    def _classify_strength(
        self,
        net_score: float,
        corroboration: int,
        contradiction: int,
        supporting: list[EvidenceItem],
    ) -> EvidenceStrength:
        """Classify overall evidence strength."""
        if contradiction > corroboration:
            return EvidenceStrength.CONTRADICTORY
        if corroboration >= self._MIN_ITEMS_STRONG and net_score >= 0.70:
            # Check that at least 2 sources are trusted/verified
            trusted_count = sum(
                1 for e in supporting if e.source_classification in ("trusted", "verified")
            )
            if trusted_count >= 2:
                return EvidenceStrength.STRONG
        if corroboration >= self._MIN_ITEMS_MODERATE and net_score >= 0.50:
            return EvidenceStrength.MODERATE
        if net_score >= 0.30:
            return EvidenceStrength.WEAK
        return EvidenceStrength.INSUFFICIENT


    def _detect_risks(
        self,
        report: EvidenceReport,
        supporting: list[EvidenceItem],
        contradicting: list[EvidenceItem],
    ) -> list[str]:
        """Detect evidence quality risks."""
        risks = []
        if report.contradiction_count > 0:
            risks.append(f"CONTRADICTION: {report.contradiction_count} sources contradict. Evidence is contested.")
        if report.source_count < self._MIN_ITEMS_MODERATE:
            risks.append(f"SINGLE_SOURCE: Only {report.source_count} source(s). Evidence is fragile.")
        unreliable_count = sum(
            1 for e in report.evidence_items if e.source_classification == "unreliable"
        )
        if unreliable_count > 0:
            risks.append(f"UNRELIABLE_SOURCES: {unreliable_count} source(s) classified as unreliable.")
        anecdotal_count = sum(
            1 for e in report.evidence_items if e.evidence_type == EvidenceType.ANECDOTAL
        )
        if anecdotal_count > 0 and anecdotal_count == report.source_count:
            risks.append("ALL_ANECDOTAL: All evidence is anecdotal. Claims are unsubstantiated.")
        # Recency risk
        old_items = [e for e in report.evidence_items if e.recency_days > 730]
        if old_items:
            risks.append(f"STALE_EVIDENCE: {len(old_items)} item(s) older than 2 years.")
        if report.confidence_low < 0.30:
            risks.append("LOW_CONFIDENCE: Confidence interval lower bound is below 0.30.")
        return risks


    def _generate_recommendations(self, report: EvidenceReport) -> list[str]:
        """Generate actionable recommendations."""
        recs = []
        if report.strength == EvidenceStrength.INSUFFICIENT:
            recs.append("Gather at least 2 independent TRUSTED or VERIFIED sources.")
        elif report.strength == EvidenceStrength.WEAK:
            recs.append("Seek additional corroboration from a TIER_1 or TIER_2 source.")
        elif report.strength == EvidenceStrength.CONTRADICTORY:
            recs.append("Resolve contradictions before making claims. Identify methodological differences.")
        if report.contradiction_count > 0:
            recs.append("Analyze why sources disagree. Check methodology, recency, and bias.")
        if report.confidence_low < 0.40:
            recs.append("Confidence is low. Add more direct evidence to narrow the interval.")
        return recs


    # ── Utilities ───────────────────────────────────────────────────────────

    def to_dict(self, report: EvidenceReport) -> dict:
        """Serialize report for JSON evidence."""
        d = asdict(report)
        d["strength"] = report.strength.value
        d["evidence_items"] = [
            {**asdict(e), "evidence_type": e.evidence_type.value}
            for e in report.evidence_items
        ]
        return d


    def merge_reports(self, reports: list[EvidenceReport]) -> EvidenceReport:
        """Merge multiple EvidenceReports into a meta-assessment."""
        if not reports:
            return EvidenceReport(claim="(empty)")
        if len(reports) == 1:
            return reports[0]
        all_items = []
        for r in reports:
            all_items.extend(r.evidence_items)
        # Use the first claim as the primary
        merged = self.assess(reports[0].claim, all_items)
        merged.claim = " | ".join(set(r.claim for r in reports))
        return merged
