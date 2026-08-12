"""Decision Brief Builder — BUILD 11 Research Pipeline.

Produces structured decision briefs: ADOPT / ADAPT / REJECT / MONITOR.
Integrates source verification, benchmark results, and evidence scoring.

Output: DecisionBrief with structured sections ready for artifact writing.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from enum import Enum


class Decision(str, Enum):
    ADOPT = "ADOPT"
    ADAPT = "ADAPT"
    REJECT = "REJECT"
    MONITOR = "MONITOR"


class ConfidenceLevel(str, Enum):
    HIGH = "high"           # Multiple TRUSTED sources, strong evidence
    MEDIUM = "medium"       # VERIFIED sources, moderate evidence
    LOW = "low"             # UNCERTAIN sources or weak evidence
    INSUFFICIENT = "insufficient"


@dataclass
class DecisionBrief:
    """Structured decision brief ready for artifact writing."""
    title: str
    decision: Decision = Decision.REJECT
    confidence: ConfidenceLevel = ConfidenceLevel.INSUFFICIENT
    summary: str = ""

    # Context
    research_question: str = ""
    candidates: list[str] = field(default_factory=list)
    date: str = field(default_factory=lambda: datetime.now(UTC).strftime("%Y-%m-%d"))

    # Sections
    evidence_summary: str = ""
    benchmark_matrix: str = ""             # Markdown table
    tradeoff_analysis: str = ""
    risks_and_mitigations: list[str] = field(default_factory=list)
    recommendation_detail: str = ""
    implementation_notes: str = ""
    alternatives: list[str] = field(default_factory=list)
    next_steps: list[str] = field(default_factory=list)
    references: list[str] = field(default_factory=list)

    # Metadata
    generated_by: str = "OVAV Research Intelligence (Eidren)"
    pipeline_version: str = "BUILD 11"


class DecisionBriefBuilder:
    """Builds structured decision briefs from research pipeline outputs.

    Integrates:
    - SourceQualityReport from SourceVerifier
    - BenchmarkMatrix from BenchmarkEngine
    - EvidenceReport from EvidenceScorer

    Produces a DecisionBrief with ADOPT/ADAPT/REJECT/MONITOR recommendation.

    Usage:
        builder = DecisionBriefBuilder()
        brief = builder.build(
            research_question="Which agentic framework should OVAV adopt?",
            candidates=["LangChain", "CrewAI", "AutoGen"],
            benchmark_matrix=matrix,
            evidence_reports={"performance": perf_report, "ecosystem": eco_report},
            source_reports=source_reports,
        )
        print(brief.decision)   # ADOPT, ADAPT, REJECT, or MONITOR
    """

    # Decision thresholds
    _ADOPT_SCORE = 0.80
    _ADAPT_SCORE = 0.65
    _MONITOR_SCORE = 0.45

    def build(
        self,
        research_question: str,
        candidates: list[str],
        benchmark_matrix=None,              # BenchmarkMatrix from BenchmarkEngine
        evidence_reports: dict[str, EvidenceReport] | None = None,
        source_reports: list | None = None, # list of SourceQualityReport
        tradeoff_notes: str = "",
    ) -> DecisionBrief:
        """Build a complete decision brief.

        Args:
            research_question: The question being answered.
            candidates: List of candidate names being compared.
            benchmark_matrix: Ranked comparison matrix.
            evidence_reports: Dict of dimension_name -> EvidenceReport.
            source_reports: Source quality assessments.
            tradeoff_notes: Additional trade-off analysis.

        Returns:
            DecisionBrief with ADOPT/ADAPT/REJECT/MONITOR.
        """
        brief = DecisionBrief(
            title=f"Decision Brief: {research_question[:80]}",
            research_question=research_question,
            candidates=candidates,
        )

        # ── 1. Determine decision ─────────────────────────────────────────────

        weighted_score, confidence = self._compute_decision(
            benchmark_matrix, evidence_reports or {}, source_reports or []
        )
        brief.decision = self._classify_decision(weighted_score, confidence)
        brief.confidence = self._classify_confidence(confidence, evidence_reports or {})

        # ── 2. Build evidence summary ─────────────────────────────────────────

        brief.evidence_summary = self._build_evidence_summary(
            evidence_reports or {}, source_reports or []
        )

        # ── 3. Benchmark matrix ───────────────────────────────────────────────

        if benchmark_matrix:
            brief.benchmark_matrix = self._format_benchmark_matrix(benchmark_matrix)

        # ── 4. Trade-off analysis ──────────────────────────────────────────────

        brief.tradeoff_analysis = tradeoff_notes or self._build_tradeoff_analysis(
            benchmark_matrix
        )

        # ── 5. Risks & mitigations ─────────────────────────────────────────────

        brief.risks_and_mitigations = self._build_risks(
            evidence_reports or {}, source_reports or [], brief.decision
        )

        # ── 6. Recommendation detail ───────────────────────────────────────────

        brief.recommendation_detail = self._build_recommendation(
            brief.decision, weighted_score, confidence, candidates
        )

        # ── 7. Implementation notes ────────────────────────────────────────────

        brief.implementation_notes = self._build_implementation_notes(
            brief.decision, candidates
        )

        # ── 8. Next steps ──────────────────────────────────────────────────────

        brief.next_steps = self._build_next_steps(brief.decision, evidence_reports or {})

        # ── 9. Summary ─────────────────────────────────────────────────────────

        brief.summary = self._build_summary(brief)

        # ── 10. Alternatives ───────────────────────────────────────────────────

        if len(candidates) > 1 and benchmark_matrix:
            if benchmark_matrix.rankings:
                winner = benchmark_matrix.rankings[0][0]
                brief.alternatives = [c for c in candidates if c != winner]

        return brief


    def _compute_decision(
        self,
        benchmark_matrix,
        evidence_reports: dict,
        source_reports: list,
    ) -> tuple[float, float]:
        """Compute decision score and confidence from all inputs."""
        score = 0.0
        total_weight = 0.0
        confidences = []

        # Benchmark weight: 40%
        if benchmark_matrix and benchmark_matrix.rankings:
            top_score = benchmark_matrix.rankings[0][1]
            score += top_score * 0.40
            total_weight += 0.40

        # Evidence weight: 40%
        if evidence_reports:
            ev_scores = [r.calibrated_score for r in evidence_reports.values()]
            ev_confidences = [
                (r.confidence_high + r.confidence_low) / 2
                for r in evidence_reports.values()
            ]
            if ev_scores:
                score += (sum(ev_scores) / len(ev_scores)) * 0.40
                total_weight += 0.40
                confidences.extend(ev_confidences)

        # Source quality weight: 20%
        if source_reports:
            # Use SourceQualityReport objects
            try:
                src_scores = [r.overall_score for r in source_reports]
                score += (sum(src_scores) / len(src_scores)) * 0.20
                total_weight += 0.20
            except AttributeError:
                # Fallback: treat as dicts
                src_scores = [r.get("overall_score", 0.5) if isinstance(r, dict) else 0.5 for r in source_reports]
                if src_scores:
                    score += (sum(src_scores) / len(src_scores)) * 0.20
                    total_weight += 0.20

        # Normalize
        if total_weight > 0:
            score = score / total_weight
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0.5

        return round(score, 3), round(avg_confidence, 3)


    def _classify_decision(self, score: float, confidence: float) -> Decision:
        """Apply decision thresholds."""
        if score >= self._ADOPT_SCORE and confidence >= 0.70:
            return Decision.ADOPT
        if score >= self._ADAPT_SCORE:
            return Decision.ADAPT
        if score >= self._MONITOR_SCORE:
            return Decision.MONITOR
        return Decision.REJECT


    def _classify_confidence(
        self, confidence: float, evidence_reports: dict
    ) -> ConfidenceLevel:
        """Classify overall confidence level."""
        if confidence >= 0.75:
            return ConfidenceLevel.HIGH
        if confidence >= 0.50:
            return ConfidenceLevel.MEDIUM
        if confidence >= 0.30:
            return ConfidenceLevel.LOW
        return ConfidenceLevel.INSUFFICIENT


    def _build_evidence_summary(
        self,
        evidence_reports: dict,
        source_reports: list,
    ) -> str:
        """Build a compact evidence summary paragraph."""
        lines = []

        if evidence_reports:
            strengths = [r.strength.value for r in evidence_reports.values()]
            strong_count = strengths.count("strong")
            moderate_count = strengths.count("moderate")
            weak_count = strengths.count("weak")
            lines.append(
                f"Evidence assessment across {len(evidence_reports)} dimensions: "
                f"{strong_count} strong, {moderate_count} moderate, {weak_count} weak."
            )

            # Detail per dimension
            for dim, report in evidence_reports.items():
                lines.append(
                    f"- **{dim}**: {report.strength.value} "
                    f"(score: {report.calibrated_score:.2f}, "
                    f"CI: [{report.confidence_low:.2f}–{report.confidence_high:.2f}], "
                    f"{report.corroboration_count} sources)"
                )

        if source_reports:
            try:
                classifications = [r.classification.value for r in source_reports]
            except AttributeError:
                classifications = [
                    r.get("classification", "unknown") if isinstance(r, dict) else "unknown"
                    for r in source_reports
                ]
            trusted = classifications.count("trusted")
            verified = classifications.count("verified")
            uncertain = classifications.count("uncertain")
            unreliable = classifications.count("unreliable")
            lines.append(
                f"Source quality: {len(source_reports)} sources — "
                f"{trusted} trusted, {verified} verified, {uncertain} uncertain, {unreliable} unreliable."
            )

        return "\n".join(lines) if lines else "No evidence provided."


    def _format_benchmark_matrix(self, matrix) -> str:
        """Format BenchmarkMatrix as markdown."""
        if not matrix or not matrix.rankings:
            return "No benchmark data available."
        return matrix.summary if hasattr(matrix, "summary") else str(matrix.rankings)


    def _build_tradeoff_analysis(self, benchmark_matrix) -> str:
        """Extract trade-off analysis from benchmark matrix."""
        if not benchmark_matrix or not hasattr(benchmark_matrix, "dimension_tradeoffs"):
            return "No trade-off analysis available."
        tradeoffs = getattr(benchmark_matrix, "dimension_tradeoffs", {})
        if not tradeoffs:
            return "No significant trade-offs detected. Winner leads across all dimensions."
        lines = ["Key trade-offs identified:"]
        for dim, leaders in tradeoffs.items():
            lines.append(f"- **{dim}**: dominated by {', '.join(leaders)}")
        return "\n".join(lines)


    def _build_risks(
        self,
        evidence_reports: dict,
        source_reports: list,
        decision: Decision,
    ) -> list[str]:
        """Aggregate risks from all evidence reports."""
        risks = []
        for dim, report in evidence_reports.items():
            for flag in getattr(report, "risk_flags", []):
                if flag not in risks:
                    risks.append(f"[{dim}] {flag}")
        # Decision-specific risks
        if decision == Decision.ADAPT:
            risks.append("ADAPT decision requires modification effort — estimate scope before committing.")
        elif decision == Decision.MONITOR:
            risks.append("MONITOR decision means waiting — set a review date and watch-list criteria.")
        elif decision == Decision.REJECT:
            risks.append("REJECT decision — ensure rejected candidates are documented to avoid revisiting.")
        return risks if risks else ["No significant risks identified."]


    def _build_recommendation(
        self,
        decision: Decision,
        score: float,
        confidence: float,
        candidates: list[str],
    ) -> str:
        """Build detailed recommendation text."""
        primary = candidates[0] if candidates else "the candidate"
        lines = [
            f"**Decision:** {decision.value}",
            f"**Score:** {score:.3f} | **Confidence:** {confidence:.3f}",
        ]
        if decision == Decision.ADOPT:
            lines.append(f"{primary} is recommended for adoption. Evidence is strong and sources are reliable.")
        elif decision == Decision.ADAPT:
            lines.append(f"{primary} shows promise but requires adaptation. Identify gaps and plan modifications before full adoption.")
        elif decision == Decision.MONITOR:
            lines.append(f"{primary} is not ready for adoption. Set a 3-6 month review window and define trigger criteria.")
        else:
            lines.append("No candidate meets the minimum threshold. Consider expanding the search or building in-house.")
        return "\n".join(lines)


    def _build_implementation_notes(
        self, decision: Decision, candidates: list[str]
    ) -> str:
        """Generate implementation guidance."""
        notes = []
        if decision == Decision.ADOPT:
            notes.append("Integration path: start with a proof-of-concept, validate against OVAV gates, then graduate.")
            notes.append("Monitor for breaking changes in the adopted tool's release cycle.")
        elif decision == Decision.ADAPT:
            notes.append("Identify minimum viable modifications before integration.")
            notes.append("Document adaptation requirements for future maintainers.")
        elif decision == Decision.MONITOR:
            notes.append("Set calendar reminder for re-evaluation.")
            notes.append("Define concrete adoption triggers: version milestone, community growth, security audit.")
        else:
            notes.append("Explore alternatives outside the current comparison set.")
            notes.append("Consider whether the problem is better solved with a custom solution.")
        return "\n".join(notes)


    def _build_next_steps(
        self, decision: Decision, evidence_reports: dict
    ) -> list[str]:
        """Generate actionable next steps."""
        steps = []
        if decision in (Decision.ADOPT, Decision.ADAPT):
            steps.append("1. Create implementation SPEC following OVAV artifact-first SDD.")
            steps.append("2. Run a controlled sandbox test before source-local apply.")
            steps.append("3. Document integration boundaries and blocked surfaces.")
        if decision == Decision.MONITOR:
            steps.append("1. Set a review date (recommended: 3 months).")
            steps.append("2. Add to OVAV absorption backlog for periodic re-evaluation.")
        if decision == Decision.REJECT:
            steps.append("1. Archive this decision brief as rationale for future reference.")
            steps.append("2. Expand research scope to additional candidates.")
        steps.append("3. Register decision in OVAV work ledger.")
        return steps


    def _build_summary(self, brief: DecisionBrief) -> str:
        """Generate compact summary."""
        return (
            f"**{brief.decision.value}** — {brief.research_question[:100]}. "
            f"Confidence: {brief.confidence.value}. "
            f"Evaluated {len(brief.candidates)} candidate(s). "
            f"Brief generated by {brief.generated_by}, {brief.pipeline_version}."
        )


    def to_dict(self, brief: DecisionBrief) -> dict:
        """Serialize brief for JSON evidence."""
        d = asdict(brief)
        d["decision"] = brief.decision.value
        d["confidence"] = brief.confidence.value
        return d
