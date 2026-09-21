"""Benchmark Engine — BUILD 11 Research Pipeline.

Compares multiple candidates across configurable dimensions.
Produces ranked comparison matrices with confidence scores.

Source-local: all candidate data provided by the research session.
No external benchmarks or API calls.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from enum import Enum


class BenchmarkDecision(str, Enum):
    ADOPT = "ADOPT"               # Clear winner across key dimensions
    ADAPT = "ADAPT"               # Promising but needs modification
    REJECT = "REJECT"             # Clear loser or critical gaps
    MONITOR = "MONITOR"           # Not ready now, watch for maturation
    TIE = "TIE"                   # Multiple viable candidates with trade-offs


@dataclass
class BenchmarkDimension:
    """A single dimension for comparison."""
    name: str
    weight: float = 1.0            # Relative importance (1.0 = baseline)
    description: str = ""
    higher_is_better: bool = True
    threshold_good: float = 0.7    # Above this = good
    threshold_bad: float = 0.3     # Below this = bad


@dataclass
class CandidateScore:
    """Score for one candidate on one dimension."""
    candidate_name: str
    dimension_name: str
    score: float                    # 0.0 – 1.0
    confidence: float = 0.8         # How confident we are in this score
    evidence: str = ""             # Brief justification
    source_references: list[str] = field(default_factory=list)


@dataclass
class Candidate:
    """A research candidate for benchmarking."""
    name: str
    description: str = ""
    url: str = ""
    version: str = ""
    license: str = ""
    category: str = ""             # e.g., "agentic_framework", "llm_provider", "memory_system"
    metadata: dict = field(default_factory=dict)


@dataclass
class BenchmarkMatrix:
    """Full comparison matrix result."""
    candidates: list[Candidate]
    dimensions: list[BenchmarkDimension]
    scores: list[CandidateScore]    # Flattened list of all candidate×dimension scores
    generated_at: str = field(default_factory=lambda: datetime.now(UTC).isoformat())

    # Computed by engine
    rankings: list[tuple[str, float, str]] = field(default_factory=list)  # (name, weighted_score, decision)
    dimension_tradeoffs: dict[str, list[str]] = field(default_factory=dict)
    summary: str = ""


class BenchmarkEngine:
    """Compares candidates across dimensions and produces ranked matrices.

    Usage:
        engine = BenchmarkEngine()
        engine.add_dimension("performance", weight=1.5, description="Speed and throughput")
        engine.add_dimension("ecosystem", weight=1.0, description="Library support and community")
        engine.add_candidate("LangChain", description="Agentic framework", category="framework")
        engine.add_candidate("CrewAI", description="Multi-agent orchestration", category="framework")

        engine.score("LangChain", "performance", 0.8, confidence=0.9, evidence="...")
        engine.score("CrewAI", "performance", 0.6, confidence=0.85, evidence="...")

        matrix = engine.compare()
        for name, score, decision in matrix.rankings:
            print(f"{name}: {score:.2f} → {decision}")
    """

    def __init__(self):
        self._dimensions: dict[str, BenchmarkDimension] = {}
        self._candidates: dict[str, Candidate] = {}
        self._scores: list[CandidateScore] = []


    def add_dimension(self, name: str, weight: float = 1.0, **kwargs) -> BenchmarkDimension:
        dim = BenchmarkDimension(name=name, weight=weight, **kwargs)
        self._dimensions[name] = dim
        return dim


    def add_candidate(self, name: str, **kwargs) -> Candidate:
        cand = Candidate(name=name, **kwargs)
        self._candidates[name] = cand
        return cand


    def score(
        self,
        candidate_name: str,
        dimension_name: str,
        score: float,
        confidence: float = 0.8,
        evidence: str = "",
        source_references: list[str] | None = None,
    ) -> None:
        """Record a score for one candidate on one dimension."""
        if candidate_name not in self._candidates:
            raise KeyError(f"Unknown candidate: {candidate_name}")
        if dimension_name not in self._dimensions:
            raise KeyError(f"Unknown dimension: {dimension_name}")
        if not 0.0 <= score <= 1.0:
            raise ValueError(f"Score must be 0.0–1.0, got {score}")

        # Remove any previous score for same candidate+dimension (upsert)
        self._scores = [
            s for s in self._scores
            if not (s.candidate_name == candidate_name and s.dimension_name == dimension_name)
        ]
        self._scores.append(CandidateScore(
            candidate_name=candidate_name,
            dimension_name=dimension_name,
            score=score,
            confidence=confidence,
            evidence=evidence,
            source_references=source_references or [],
        ))


    def compare(self) -> BenchmarkMatrix:
        """Run comparison: compute weighted scores, rank, classify, and detect trade-offs.

        Returns BenchmarkMatrix with rankings, trade-offs, and summary.
        """
        if not self._candidates or not self._dimensions:
            return BenchmarkMatrix(
                candidates=list(self._candidates.values()),
                dimensions=list(self._dimensions.values()),
                scores=list(self._scores),
                summary="Insufficient data: add candidates and dimensions before comparing.",
            )

        # Build score lookup: (candidate, dim) -> CandidateScore
        score_map: dict[tuple[str, str], CandidateScore] = {}
        for s in self._scores:
            score_map[(s.candidate_name, s.dimension_name)] = s

        # Compute weighted aggregate per candidate
        raw = {}        # candidate_name -> (weighted_score, total_weight, confidence_avg)
        dim_ranks: dict[str, list[tuple[str, float]]] = {d.name: [] for d in self._dimensions.values()}

        for cname in self._candidates:
            total_weighted = 0.0
            total_weight = 0.0
            confidences = []
            for dname, dim in self._dimensions.items():
                key = (cname, dname)
                if key in score_map:
                    s = score_map[key]
                    total_weighted += s.score * dim.weight
                    total_weight += dim.weight
                    confidences.append(s.confidence)
                    dim_ranks[dname].append((cname, s.score))
            if total_weight > 0:
                raw[cname] = (
                    round(total_weighted / total_weight, 3),
                    total_weight,
                    round(sum(confidences) / len(confidences), 3) if confidences else 0.0,
                )
            else:
                raw[cname] = (0.0, 0.0, 0.0)

        # Rank by weighted score descending
        ranked = sorted(raw.items(), key=lambda x: x[1][0], reverse=True)

        # Classify each candidate
        decisions: dict[str, BenchmarkDecision] = {}
        for cname, (wscore, _, conf) in ranked:
            decisions[cname] = self._decide(cname, wscore, conf, score_map)

        # Build rankings list
        rankings = [(name, score, decisions[name].value) for name, (score, _, _) in ranked]

        # Detect dimension trade-offs
        tradeoffs: dict[str, list[str]] = {}
        for dname, cand_scores in dim_ranks.items():
            cand_scores.sort(key=lambda x: x[1], reverse=True)
            # A trade-off exists if the top scorer on this dimension is NOT the overall winner
            if ranked and cand_scores:
                top_on_dim = cand_scores[0][0]
                overall_winner = ranked[0][0]
                if top_on_dim != overall_winner:
                    tradeoffs[dname] = [
                        f"{name} ({score:.2f})" for name, score in cand_scores[:3]
                    ]

        # Generate summary — flatten ranked for the summary generator
        flat_ranked: list[tuple[str, float, float]] = [
            (name, score, conf) for name, (score, _, conf) in ranked
        ]
        summary = self._generate_summary(flat_ranked, decisions, tradeoffs)

        return BenchmarkMatrix(
            candidates=list(self._candidates.values()),
            dimensions=list(self._dimensions.values()),
            scores=list(self._scores),
            rankings=rankings,
            dimension_tradeoffs=tradeoffs,
            summary=summary,
        )


    def _decide(
        self,
        cname: str,
        weighted_score: float,
        confidence: float,
        score_map: dict[tuple[str, str], CandidateScore],
    ) -> BenchmarkDecision:
        """Apply decision logic to one candidate."""
        # Check for any score below threshold_bad
        for dim in self._dimensions.values():
            key = (cname, dim.name)
            if key in score_map:
                if score_map[key].score < dim.threshold_bad:
                    return BenchmarkDecision.REJECT

        # Check critical dimensions for minimum viability
        critical_dims = [d for d in self._dimensions.values() if d.weight >= 1.5]
        for dim in critical_dims:
            key = (cname, dim.name)
            if key not in score_map:
                return BenchmarkDecision.REJECT      # missing critical dimension

        # Main classification
        if weighted_score >= 0.80 and confidence >= 0.75:
            return BenchmarkDecision.ADOPT
        if weighted_score >= 0.65:
            return BenchmarkDecision.ADAPT
        if weighted_score >= 0.45 and confidence >= 0.5:
            return BenchmarkDecision.MONITOR
        return BenchmarkDecision.REJECT


    def _generate_summary(
        self,
        ranked: list[tuple[str, float, float]],    # (name, score, confidence)
        decisions: dict[str, BenchmarkDecision],
        tradeoffs: dict[str, list[str]],
    ) -> str:
        """Generate human-readable summary."""
        if not ranked:
            return "No candidates to compare."

        winner_name, winner_score, winner_conf = ranked[0]
        winner_decision = decisions.get(winner_name, BenchmarkDecision.TIE)

        lines = ["## Benchmark Summary\n"]
        lines.append(f"**Compared:** {len(self._candidates)} candidates across {len(self._dimensions)} dimensions\n")

        # Winner
        if winner_decision == BenchmarkDecision.ADOPT:
            lines.append(f"**Winner:** {winner_name} (score: {winner_score:.2f}, confidence: {winner_conf:.2f}) — RECOMMEND ADOPT")
        elif winner_decision == BenchmarkDecision.ADAPT:
            lines.append(f"**Front-runner:** {winner_name} (score: {winner_score:.2f}) — RECOMMEND ADAPT (needs modifications)")
        elif winner_decision == BenchmarkDecision.MONITOR:
            lines.append(f"**Best available:** {winner_name} (score: {winner_score:.2f}) — RECOMMEND MONITOR (wait for maturation)")
        else:
            lines.append(f"**No clear winner.** Top: {winner_name} (score: {winner_score:.2f}) — REJECT or seek alternatives")

        # Rankings table
        lines.append("\n### Rankings\n")
        lines.append("| # | Candidate | Score | Confidence | Decision |")
        lines.append("|---|---|---|---|---|")
        for i, (name, score, conf) in enumerate(ranked, 1):
            d = decisions.get(name, BenchmarkDecision.TIE)
            lines.append(f"| {i} | {name} | {score:.3f} | {conf:.2f} | **{d.value}** |")

        # Trade-offs
        if tradeoffs:
            lines.append(f"\n### Dimension Trade-offs ({len(tradeoffs)} dimensions)\n")
            for dim, leaders in tradeoffs.items():
                lines.append(f"- **{dim}** is dominated by: {', '.join(leaders)} (not the overall winner)")

        return "\n".join(lines)


    # ── Utilities ───────────────────────────────────────────────────────────

    def to_dict(self, matrix: BenchmarkMatrix) -> dict:
        """Serialize matrix to dict for JSON evidence."""
        return {
            "candidates": [asdict(c) for c in matrix.candidates],
            "dimensions": [asdict(d) for d in matrix.dimensions],
            "scores": [asdict(s) for s in matrix.scores],
            "rankings": [
                {"name": n, "score": s, "decision": d}
                for n, s, d in matrix.rankings
            ],
            "tradeoffs": matrix.dimension_tradeoffs,
            "summary": matrix.summary,
            "generated_at": matrix.generated_at,
        }


    def dimension_gap_analysis(self, matrix: BenchmarkMatrix) -> dict[str, dict]:
        """Identify gaps: dimensions where all candidates score below threshold_good."""
        gaps = {}
        for dim in self._dimensions.values():
            scores_on_dim = [s.score for s in matrix.scores if s.dimension_name == dim.name]
            if scores_on_dim:
                avg = sum(scores_on_dim) / len(scores_on_dim)
                if avg < dim.threshold_good:
                    gaps[dim.name] = {
                        "avg_score": round(avg, 3),
                        "threshold_good": dim.threshold_good,
                        "all_scores": scores_on_dim,
                        "candidates": [s.candidate_name for s in matrix.scores if s.dimension_name == dim.name],
                        "recommendation": f"No candidate meets the {dim.name} threshold. Consider this dimension a risk factor.",
                    }
        return gaps
