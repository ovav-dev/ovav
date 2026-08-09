"""Research Artifact Writer — BUILD 11 Research Pipeline.

Writes structured research artifacts following OVAV artifact-first SDD doctrine.
Outputs go to .ovav/artifacts/ under a research session directory.

Artifacts:
- RESEARCH_SCOPE.md — Research question, scope, methodology
- SOURCE_MAP.md — Source inventory with quality classifications
- EVIDENCE_REVIEW.md — Calibrated evidence assessment
- BENCHMARK_MATRIX.md — Comparative ranking table
- DECISION_BRIEF.md — ADOPT/ADAPT/REJECT/MONITOR recommendation
- HANDOFF.md — Closure and next steps
"""

from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path


class ResearchArtifactWriter:
    """Writes research pipeline outputs to OVAV artifact format.

    Source-local only. Writes to .ovav/artifacts/ inside the OVAV repo.

    Usage:
        writer = ResearchArtifactWriter()
        session = writer.create_session("agentic_framework_comparison_2026")
        writer.write_scope(session, research_question="...", methodology="...")
        writer.write_source_map(session, source_reports=[...])
        writer.write_evidence(session, evidence_report=report)
        writer.write_benchmark(session, benchmark_matrix=matrix)
        writer.write_decision(session, brief=brief)
        writer.write_handoff(session, summary="...")
    """

    def __init__(self, base_path: str | None = None):
        """Initialize writer with optional custom base path.

        Args:
            base_path: Root for .ovav/artifacts/. Defaults to current working directory.
        """
        self._base = Path(base_path) if base_path else Path.cwd()
        self._artifacts_root = self._base / ".ovav" / "artifacts"


    def create_session(self, session_id: str) -> Path:
        """Create a research session directory and return its path.

        Args:
            session_id: Unique session identifier (e.g., 'agentic_framework_comparison_2026').

        Returns:
            Path to the session artifact directory.
        """
        session_dir = self._artifacts_root / session_id
        session_dir.mkdir(parents=True, exist_ok=True)
        evidence_dir = session_dir / "evidence"
        evidence_dir.mkdir(exist_ok=True)
        return session_dir


    def write_scope(
        self,
        session_dir: Path,
        research_question: str,
        scope: str = "",
        methodology: str = "",
        constraints: str = "",
        expected_outputs: str = "",
    ) -> Path:
        """Write RESEARCH_SCOPE.md artifact."""
        content = self._md_header("Research Scope", session_dir.name)
        content += f"## Research Question\n\n{research_question}\n\n"

        if scope:
            content += f"## Scope\n\n{scope}\n\n"
        if methodology:
            content += f"## Methodology\n\n{methodology}\n\n"
        if constraints:
            content += f"## Constraints\n\n{constraints}\n\n"
        if expected_outputs:
            content += f"## Expected Outputs\n\n{expected_outputs}\n\n"

        content += self._md_footer("RESEARCH_SCOPE")
        return self._write(session_dir, "RESEARCH_SCOPE.md", content)


    def write_source_map(
        self,
        session_dir: Path,
        source_reports: list,
        summary: str = "",
    ) -> Path:
        """Write SOURCE_MAP.md artifact from SourceQualityReport objects."""
        content = self._md_header("Source Map", session_dir.name)
        content += "## Source Inventory\n\n"
        content += f"**Total sources:** {len(source_reports)}\n\n"

        # Summary table
        content += "| # | Source | Classification | Score | Tier | Warnings |\n"
        content += "|---|---|---|---|---|---|\n"

        for i, report in enumerate(source_reports, 1):
            try:
                name = report.source_title
                cls = report.classification.value
                score = f"{report.overall_score:.3f}"
                tier = report.credibility_tier.value
                warnings = "; ".join(report.warnings[:2]) if report.warnings else "—"
            except AttributeError:
                # Dict fallback
                name = report.get("source_title", "Unknown")
                cls = report.get("classification", "unknown")
                score = f"{report.get('overall_score', 0):.3f}"
                tier = report.get("credibility_tier", "unknown")
                warnings = "; ".join(report.get("warnings", [])[:2]) or "—"

            content += f"| {i} | {name[:60]} | {cls} | {score} | {tier} | {warnings[:80]} |\n"

        if summary:
            content += f"\n## Summary\n\n{summary}\n\n"

        content += self._md_footer("SOURCE_MAP")
        return self._write(session_dir, "SOURCE_MAP.md", content)


    def write_evidence(
        self,
        session_dir: Path,
        evidence_report,          # EvidenceReport
        additional_reports: list | None = None,
    ) -> Path:
        """Write EVIDENCE_REVIEW.md artifact."""
        content = self._md_header("Evidence Review", session_dir.name)

        try:
            content += f"## Claim\n\n{evidence_report.claim}\n\n"
            content += f"**Strength:** {evidence_report.strength.value}\n"
            content += f"**Calibrated Score:** {evidence_report.calibrated_score:.3f}\n"
            content += f"**Confidence Interval:** [{evidence_report.confidence_low:.3f}, {evidence_report.confidence_high:.3f}]\n\n"
        except AttributeError:
            content += f"## Claim\n\n{evidence_report.get('claim', 'N/A')}\n\n"

        # Evidence items table
        content += "## Evidence Items\n\n"
        content += "| Source | Type | Quality | Directness | Supports |\n"
        content += "|---|---|---|---|---|\n"

        try:
            items = evidence_report.evidence_items
        except AttributeError:
            items = evidence_report.get("evidence_items", [])

        for item in items:
            try:
                src = item.source_title[:40]
                etype = item.evidence_type.value
                qual = f"{item.source_quality_score:.2f}"
                direct = f"{item.directness:.2f}"
                supports = "✅" if item.supports_claim else "❌"
            except AttributeError:
                src = item.get("source_title", "?")[:40]
                etype = item.get("evidence_type", "?")
                qual = f"{item.get('source_quality_score', 0):.2f}"
                direct = f"{item.get('directness', 0):.2f}"
                supports = "✅" if item.get("supports_claim", True) else "❌"
            content += f"| {src} | {etype} | {qual} | {direct} | {supports} |\n"

        # Risks
        try:
            risks = evidence_report.risk_flags
        except AttributeError:
            risks = evidence_report.get("risk_flags", [])
        if risks:
            content += "\n## Risk Flags\n\n"
            for r in risks:
                content += f"- {r}\n"

        content += self._md_footer("EVIDENCE_REVIEW")
        return self._write(session_dir, "EVIDENCE_REVIEW.md", content)


    def write_benchmark(
        self,
        session_dir: Path,
        benchmark_matrix,        # BenchmarkMatrix
    ) -> Path:
        """Write BENCHMARK_MATRIX.md artifact."""
        content = self._md_header("Benchmark Matrix", session_dir.name)

        try:
            content += benchmark_matrix.summary
        except AttributeError:
            content += str(benchmark_matrix)

        content += f"\n\n{self._md_footer('BENCHMARK_MATRIX')}"
        return self._write(session_dir, "BENCHMARK_MATRIX.md", content)


    def write_decision(
        self,
        session_dir: Path,
        brief,                    # DecisionBrief
    ) -> Path:
        """Write DECISION_BRIEF.md artifact."""
        content = self._md_header("Decision Brief", session_dir.name)

        try:
            content += f"## Decision\n\n**{brief.decision.value}** — Confidence: {brief.confidence.value}\n\n"
            content += f"## Research Question\n\n{brief.research_question}\n\n"
            content += f"## Summary\n\n{brief.summary}\n\n"
            content += f"## Evidence Summary\n\n{brief.evidence_summary}\n\n"

            if brief.benchmark_matrix:
                content += f"## Benchmark Results\n\n{brief.benchmark_matrix}\n\n"

            if brief.tradeoff_analysis:
                content += f"## Trade-off Analysis\n\n{brief.tradeoff_analysis}\n\n"

            if brief.risks_and_mitigations:
                content += "## Risks & Mitigations\n\n"
                for r in brief.risks_and_mitigations:
                    content += f"- {r}\n"
                content += "\n"

            content += f"## Recommendation Detail\n\n{brief.recommendation_detail}\n\n"

            if brief.implementation_notes:
                content += f"## Implementation Notes\n\n{brief.implementation_notes}\n\n"

            if brief.alternatives:
                content += "## Alternatives Considered\n\n"
                for a in brief.alternatives:
                    content += f"- {a}\n"
                content += "\n"

            if brief.next_steps:
                content += "## Next Steps\n\n"
                for s in brief.next_steps:
                    content += f"{s}\n"
                content += "\n"

            if brief.references:
                content += "## References\n\n"
                for r in brief.references:
                    content += f"- {r}\n"
                content += "\n"
        except AttributeError:
            # Dict fallback
            content += f"## Decision\n\n**{brief.get('decision', 'N/A')}**\n\n"
            content += f"## Summary\n\n{brief.get('summary', 'N/A')}\n\n"

        content += self._md_footer("DECISION_BRIEF")
        return self._write(session_dir, "DECISION_BRIEF.md", content)


    def write_handoff(
        self,
        session_dir: Path,
        decision: str,
        summary: str,
        next_steps: str = "",
        session_id: str = "",
    ) -> Path:
        """Write HANDOFF.md artifact."""
        content = self._md_header("Research Handoff", session_dir.name)
        content += f"## Decision\n\n{decision}\n\n"
        content += f"## Summary\n\n{summary}\n\n"
        if next_steps:
            content += f"## Next Steps\n\n{next_steps}\n\n"
        if session_id:
            content += f"## Session\n\n`{session_id}`\n\n"
        content += self._md_footer("HANDOFF")
        return self._write(session_dir, "HANDOFF.md", content)


    def write_evidence_json(
        self,
        session_dir: Path,
        data: dict,
        filename: str,
    ) -> Path:
        """Write a JSON evidence file to session/evidence/."""
        evidence_dir = session_dir / "evidence"
        evidence_dir.mkdir(exist_ok=True)
        path = evidence_dir / filename
        with open(path, "w") as f:
            json.dump(data, f, indent=2, default=str)
        return path


    # ── Internal helpers ────────────────────────────────────────────────────

    def _write(self, session_dir: Path, filename: str, content: str) -> Path:
        """Write content to a file in the session directory."""
        path = session_dir / filename
        with open(path, "w") as f:
            f.write(content)
        return path


    def _md_header(self, title: str, session_id: str) -> str:
        ts = datetime.now(UTC).strftime("%Y-%m-%d %H:%M UTC")
        return (
            f"# {title}\n\n"
            f"**Session:** `{session_id}`\n"
            f"**Generated:** {ts}\n"
            f"**Pipeline:** OVAV Research Pipeline — BUILD 11\n"
            f"**Generated by:** OVAV Research Intelligence (Eidren)\n\n"
            "---\n\n"
        )


    def _md_footer(self, artifact_type: str) -> str:
        return (
            f"\n\n---\n\n"
            f"*Artifact type: {artifact_type}. Generated by OVAV Research Pipeline BUILD 11.*\n"
        )
