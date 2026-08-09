"""OVAV Research Pipeline — BUILD 11.

Source verification, benchmark engine, evidence scoring, decision briefs.
P0 operator: Eidren (OVAV Research Intelligence).

Source-local only. No external services, web fetches, or API calls.
All modules operate on structured inputs provided by the research session.
"""

__version__ = "11.0.0"
__build__ = "BUILD 11 — Research Pipeline Maturation"

from tools.research.benchmark_engine import (
    BenchmarkDecision,
    BenchmarkDimension,
    BenchmarkEngine,
    BenchmarkMatrix,
    Candidate,
    CandidateScore,
)
from tools.research.decision_brief_builder import (
    ConfidenceLevel,
    Decision,
    DecisionBrief,
    DecisionBriefBuilder,
)
from tools.research.evidence_scorer import (
    EvidenceItem,
    EvidenceReport,
    EvidenceScorer,
    EvidenceStrength,
    EvidenceType,
)
from tools.research.research_artifact_writer import ResearchArtifactWriter
from tools.research.source_verifier import (
    BiasIndicator,
    CredibilityTier,
    SourceClassification,
    SourceMetadata,
    SourceQualityReport,
    SourceVerifier,
)

__all__ = [
    "BenchmarkDecision",
    "BenchmarkDimension",
    "BenchmarkEngine",
    "BenchmarkMatrix",
    "BiasIndicator",
    "Candidate",
    "CandidateScore",
    "ConfidenceLevel",
    "CredibilityTier",
    "Decision",
    "DecisionBrief",
    "DecisionBriefBuilder",
    "EvidenceItem",
    "EvidenceReport",
    "EvidenceScorer",
    "EvidenceStrength",
    "EvidenceType",
    "ResearchArtifactWriter",
    "SourceClassification",
    "SourceMetadata",
    "SourceQualityReport",
    "SourceVerifier",
]
