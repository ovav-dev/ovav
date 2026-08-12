"""Source Verifier — BUILD 11 Research Pipeline.

Validates source credibility, freshness, bias indicators, and methodological rigor.
Output: SourceQualityReport with component scores and classification.

Source-local: operates on source metadata provided by the research session.
No web fetches, no API calls. All verification is signal-based on structured input.
"""

from __future__ import annotations

import re
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime
from enum import Enum

# ── Enums ──────────────────────────────────────────────────────────────────────

class SourceClassification(str, Enum):
    TRUSTED = "trusted"           # Authoritative, peer-reviewed, recent
    VERIFIED = "verified"         # Cross-referenced, credible signals
    UNCERTAIN = "uncertain"       # Mixed signals, needs corroboration
    UNRELIABLE = "unreliable"     # Red flags: bias, stale, anonymous
    UNKNOWN = "unknown"           # Insufficient data to classify


class CredibilityTier(str, Enum):
    TIER_1 = "tier_1"   # Academic peer-reviewed, government, standards body
    TIER_2 = "tier_2"   # Industry publication, established vendor docs
    TIER_3 = "tier_3"   # Blog, community wiki, tutorial
    TIER_4 = "tier_4"   # Personal opinion, unverified social, anonymous


class BiasIndicator(str, Enum):
    NONE = "none"
    COMMERCIAL = "commercial"
    PROMOTIONAL = "promotional"
    IDEOLOGICAL = "ideological"
    CONFIRMATION = "confirmation"
    SELECTIVE = "selective"
    SPONSORED = "sponsored"


# ── Source metadata input ──────────────────────────────────────────────────────

@dataclass
class SourceMetadata:
    """Structured input for source verification. All fields optional."""

    # Identity
    title: str = ""
    author: str | None = None
    organization: str | None = None
    publisher: str | None = None

    # Timeliness
    published_date: str | None = None           # ISO 8601 or YYYY-MM-DD
    last_updated: str | None = None
    max_age_days: int = 365                        # How stale is too stale?

    # Credibility signals
    has_citations: bool = False
    citation_count: int = 0
    peer_reviewed: bool = False
    has_methodology: bool = False
    has_data_availability: bool = False
    author_credentials_verified: bool = False
    domain_authority: str | None = None         # edu, gov, org, com, io, etc.
    venue_reputation: str | None = None         # "nature", "arxiv", "medium", etc.

    # Bias signals
    funding_source: str | None = None
    commercial_affiliation: bool = False
    sponsored: bool = False
    language_tone: str | None = None            # "neutral", "promotional", "critical", "sensational"
    conflicts_of_interest_disclosed: bool = False
    retraction_history: bool = False

    # Cross-reference
    corroborated_by: list[str] = field(default_factory=list)
    contradicted_by: list[str] = field(default_factory=list)


# ── Output report ──────────────────────────────────────────────────────────────

@dataclass
class SourceQualityReport:
    """Result of source verification."""

    source_title: str
    classification: SourceClassification
    overall_score: float                          # 0.0 – 1.0

    # Component scores
    freshness_score: float
    credibility_score: float
    methodology_score: float
    bias_penalty: float                           # 0 = no bias, higher = more bias
    corroboration_bonus: float

    # Details
    credibility_tier: CredibilityTier
    bias_indicators: list[BiasIndicator] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    recommendations: list[str] = field(default_factory=list)

    # Metadata
    verified_at: str = field(default_factory=lambda: datetime.now(UTC).isoformat())


# ── Bias detection patterns ────────────────────────────────────────────────────

_BIAS_PATTERNS: dict[BiasIndicator, list[str]] = {
    BiasIndicator.COMMERCIAL: [
        r"\bbuy now\b", r"\bexclusive offer\b", r"\bpricing\b.*\bonly\b",
        r"\b competitors?\b", r"\bmarket(?:ing)?\b.*\bsolution\b",
    ],
    BiasIndicator.PROMOTIONAL: [
        r"\brevolutionary\b", r"\bgame.?changing\b", r"\bunmatched\b",
        r"\bbest.in.class\b", r"\bworld.?class\b", r"\bindustry.leading\b",
    ],
    BiasIndicator.IDEOLOGICAL: [
        r"\bwoke\b", r"\bagenda\b", r"\bconspiracy\b", r"\bdeep state\b",
        r"\bfake news\b", r"\bgrooming\b",
    ],
    BiasIndicator.SELECTIVE: [
        r"\bcherry.?pick\b", r"\bselective\b.*\bevidence\b",
        r"\bconveniently\b.*\bignore\b",
    ],
    BiasIndicator.SPONSORED: [
        r"\bsponsored\b", r"\bpaid\b.*\bpromotion\b", r"\baffiliate\b",
        r"\badvertorial\b", r"\bpartner content\b",
    ],
}

_VENUE_TIERS: dict[str, CredibilityTier] = {
    "nature": CredibilityTier.TIER_1, "science": CredibilityTier.TIER_1,
    "cell": CredibilityTier.TIER_1, "lancet": CredibilityTier.TIER_1,
    "nejm": CredibilityTier.TIER_1, "pnas": CredibilityTier.TIER_1,
    "arxiv": CredibilityTier.TIER_2, "acm": CredibilityTier.TIER_1,
    "ieee": CredibilityTier.TIER_1, "springer": CredibilityTier.TIER_1,
    "elsevier": CredibilityTier.TIER_1, "wiley": CredibilityTier.TIER_1,
    "oreilly": CredibilityTier.TIER_2, "manning": CredibilityTier.TIER_2,
    "pragprog": CredibilityTier.TIER_2,
    "medium": CredibilityTier.TIER_3, "dev.to": CredibilityTier.TIER_3,
    "hashnode": CredibilityTier.TIER_3, "substack": CredibilityTier.TIER_3,
    "reddit": CredibilityTier.TIER_4, "twitter": CredibilityTier.TIER_4,
    "youtube": CredibilityTier.TIER_4,
}


class SourceVerifier:
    """Analyzes source metadata and produces a SourceQualityReport.

    Pure signal analysis. No external calls. All inputs must be provided
    as SourceMetadata by the research session.

    Usage:
        verifier = SourceVerifier()
        report = verifier.verify(SourceMetadata(
            title="Attention Is All You Need",
            author="Vaswani et al.",
            published_date="2017-06-12",
            peer_reviewed=True,
            citation_count=120000,
            domain_authority="org",
            venue_reputation="arxiv",
        ))
        print(report.classification)  # TRUSTED
    """

    # ── Freshness scoring ───────────────────────────────────────────────────

    def _score_freshness(self, meta: SourceMetadata) -> tuple[float, list[str]]:
        """Score how current the source is. 1.0 = published today. Decays over max_age_days."""
        warnings: list[str] = []
        if not meta.published_date:
            warnings.append("No publication date — freshness cannot be verified.")
            return 0.3, warnings          # penalty for unknown date

        try:
            pub_date = datetime.fromisoformat(meta.published_date.replace("Z", "+00:00"))
            if pub_date.tzinfo is None:
                pub_date = pub_date.replace(tzinfo=UTC)
        except (ValueError, AttributeError):
            try:
                pub_date = datetime.strptime(meta.published_date, "%Y-%m-%d").replace(tzinfo=UTC)
            except ValueError:
                warnings.append("Unparseable publication date format.")
                return 0.3, warnings

        now = datetime.now(UTC)
        age_days = (now - pub_date).days

        if age_days < 0:
            warnings.append("Publication date is in the future.")
            return 0.5, warnings

        # Exponential decay: score = e^(-age / max_age_days * ln(2))
        # At max_age_days, score = 0.5. At 2*max, score = 0.25.
        import math
        decay_factor = math.log(2) / meta.max_age_days
        score = math.exp(-decay_factor * age_days)

        # Adjust for updates
        if meta.last_updated:
            try:
                upd_date = datetime.fromisoformat(meta.last_updated.replace("Z", "+00:00"))
                if upd_date.tzinfo is None:
                    upd_date = upd_date.replace(tzinfo=UTC)
                upd_age = (now - upd_date).days
                # Blend: 70% original date, 30% update date
                score = 0.7 * score + 0.3 * math.exp(-decay_factor * upd_age)
            except (ValueError, AttributeError):
                pass

        # Floor
        score = max(0.1, min(1.0, score))

        if age_days > meta.max_age_days:
            warnings.append(f"Source is {age_days} days old (max allowed: {meta.max_age_days}).")
        if age_days > meta.max_age_days * 2:
            warnings.append("Source is significantly outdated. Consider finding a more recent reference.")

        return round(score, 3), warnings


    # ── Credibility scoring ─────────────────────────────────────────────────

    def _score_credibility(self, meta: SourceMetadata) -> tuple[float, CredibilityTier, list[str]]:
        """Composite credibility score from signals."""
        score = 0.5         # neutral baseline
        warnings: list[str] = []

        # Authorship (+0.15 if named, +0.1 if credentials verified)
        if meta.author and meta.author.strip():
            score += 0.15
            if meta.author_credentials_verified:
                score += 0.10
        else:
            warnings.append("Anonymous or missing author.")
            score -= 0.10

        # Organization (+0.10 if named)
        if meta.organization:
            score += 0.10

        # Domain authority
        if meta.domain_authority:
            da = meta.domain_authority.lower()
            if da in ("edu", "gov"):
                score += 0.10
            elif da == "org":
                score += 0.05
            elif da in ("com", "io"):
                score += 0.0
            else:
                score -= 0.05

        # Venue reputation
        tier = CredibilityTier.TIER_3          # default
        if meta.venue_reputation:
            vr = meta.venue_reputation.lower().strip()
            tier = _VENUE_TIERS.get(vr, CredibilityTier.TIER_3)

        tier_bonus = {
            CredibilityTier.TIER_1: 0.20,
            CredibilityTier.TIER_2: 0.10,
            CredibilityTier.TIER_3: 0.0,
            CredibilityTier.TIER_4: -0.10,
        }
        score += tier_bonus[tier]

        # Citations (+0.05 per order of magnitude, capped)
        if meta.citation_count > 0:
            import math
            cit_bonus = min(0.20, 0.05 * math.log10(max(1, meta.citation_count)))
            score += cit_bonus

        # Corroboration (each corroborating source adds 0.02, max 0.10)
        corr_bonus = min(0.10, len(meta.corroborated_by) * 0.02)
        score += corr_bonus

        # Contradiction penalty
        contr_penalty = min(0.15, len(meta.contradicted_by) * 0.05)
        score -= contr_penalty
        if meta.contradicted_by:
            warnings.append(f"Contradicted by {len(meta.contradicted_by)} sources. Verify claims independently.")

        return round(max(0.0, min(1.0, score)), 3), tier, warnings


    # ── Methodology scoring ─────────────────────────────────────────────────

    def _score_methodology(self, meta: SourceMetadata) -> tuple[float, list[str]]:
        """Score methodological rigor."""
        score = 0.3          # baseline — most sources lack explicit methodology
        warnings: list[str] = []

        if meta.peer_reviewed:
            score += 0.35
        if meta.has_methodology:
            score += 0.25
        if meta.has_data_availability:
            score += 0.10
        if meta.has_citations:
            score += min(0.10, 0.005 * meta.citation_count)

        if not meta.peer_reviewed and not meta.has_methodology:
            warnings.append("No peer review or methodology section. Treat claims as unverified.")
            score = min(score, 0.40)

        return round(max(0.0, min(1.0, score)), 3), warnings


    # ── Bias detection ──────────────────────────────────────────────────────

    def _detect_bias(self, meta: SourceMetadata) -> tuple[float, list[BiasIndicator], list[str]]:
        """Detect bias indicators from metadata and language patterns."""
        indicators: list[BiasIndicator] = []
        warnings: list[str] = []
        penalty = 0.0

        # Metadata-based indicators
        if meta.sponsored:
            indicators.append(BiasIndicator.SPONSORED)
            penalty += 0.15
            warnings.append("Source is sponsored — disclosure of commercial interest required.")

        if meta.commercial_affiliation:
            indicators.append(BiasIndicator.COMMERCIAL)
            penalty += 0.10
            if not meta.conflicts_of_interest_disclosed:
                warnings.append("Commercial affiliation without disclosed conflicts of interest.")

        if meta.retraction_history:
            penalty += 0.20
            warnings.append("Source has retraction history. Claims require independent verification.")

        if meta.funding_source:
            # Check for industry funding patterns
            industry_keywords = ["inc", "corp", "ltd", "llc", "pharma", "biotech", "labs"]
            if any(kw in (meta.funding_source or "").lower() for kw in industry_keywords):
                indicators.append(BiasIndicator.COMMERCIAL)
                penalty += 0.05

        # Language tone analysis (simple keyword-based)
        if meta.language_tone:
            tone = meta.language_tone.lower()
            if tone in ("promotional", "sensational"):
                indicators.append(BiasIndicator.COMMERCIAL)
                penalty += 0.08
                warnings.append(f"Language tone is {tone}. Apply skepticism to claims.")
            elif tone in ("critical", "negative"):
                if not indicators:
                    pass          # Critical tone alone is not necessarily bias
            elif tone == "neutral":
                pass               # No penalty

        # Text content analysis (if provided)
        # Check content_text via pattern matching in the full verify flow

        penalty = min(0.35, penalty)       # cap bias penalty
        return round(penalty, 3), indicators, warnings


    def _scan_content_bias(self, content: str) -> tuple[float, list[BiasIndicator]]:
        """Scan text content for bias language patterns. Returns additional penalty."""
        if not content:
            return 0.0, []
        content_lower = content.lower()
        extra_penalty = 0.0
        extra_indicators: list[BiasIndicator] = []
        for indicator, patterns in _BIAS_PATTERNS.items():
            matches = sum(1 for p in patterns if re.search(p, content_lower))
            if matches > 0:
                extra_indicators.append(indicator)
                extra_penalty += min(0.05 * matches, 0.10)
        return min(0.15, extra_penalty), extra_indicators


    # ── Main verification ───────────────────────────────────────────────────

    def verify(self, meta: SourceMetadata, content_text: str = "") -> SourceQualityReport:
        """Run full verification pipeline on source metadata.

        Args:
            meta: Structured source metadata.
            content_text: Optional full text for language bias scanning.

        Returns:
            SourceQualityReport with component scores and classification.
        """
        warnings: list[str] = []
        recommendations: list[str] = []

        # 1. Freshness
        freshness, fw = self._score_freshness(meta)
        warnings.extend(fw)

        # 2. Credibility
        credibility, tier, cw = self._score_credibility(meta)
        warnings.extend(cw)

        # 3. Methodology
        methodology, mw = self._score_methodology(meta)
        warnings.extend(mw)

        # 4. Bias
        bias_penalty, bias_indicators, bw = self._detect_bias(meta)
        warnings.extend(bw)

        # Content bias scan
        if content_text:
            content_bias_penalty, content_indicators = self._scan_content_bias(content_text)
            bias_penalty += content_bias_penalty
            bias_indicators.extend(content_indicators)

        # 5. Corroboration bonus (separate from credibility composite)
        corroboration_bonus = min(0.10, len(meta.corroborated_by) * 0.025)

        # 6. Composite score
        # Weights: freshness 15%, credibility 40%, methodology 25%, corroboration 10%, minus bias
        overall = (
            0.15 * freshness +
            0.40 * credibility +
            0.25 * methodology +
            0.10 * corroboration_bonus -
            bias_penalty
        )
        # Also give base score a 0.10 floor from the remaining weight
        overall = max(0.0, min(1.0, overall))

        # 7. Classification
        classification = self._classify(overall, bias_penalty, tier, freshness, meta.citation_count)

        # 8. Recommendations
        if classification == SourceClassification.UNRELIABLE:
            recommendations.append("Do not cite this source. Seek alternatives.")
        elif classification == SourceClassification.UNCERTAIN:
            recommendations.append("Corroborate with at least one TRUSTED or VERIFIED source.")
        if freshness < 0.3:
            recommendations.append("Source is stale — seek a more recent reference.")
        if bias_penalty > 0.15:
            recommendations.append("Significant bias detected. Cross-reference with neutral sources.")
        if tier == CredibilityTier.TIER_4:
            recommendations.append("Low-credibility venue. Prefer TIER_1 or TIER_2 sources.")

        return SourceQualityReport(
            source_title=meta.title or "Untitled Source",
            classification=classification,
            overall_score=round(overall, 3),
            freshness_score=freshness,
            credibility_score=credibility,
            methodology_score=methodology,
            bias_penalty=round(bias_penalty, 3),
            corroboration_bonus=round(corroboration_bonus, 3),
            credibility_tier=tier,
            bias_indicators=bias_indicators,
            warnings=warnings,
            recommendations=recommendations,
        )


    def _classify(
        self,
        overall: float,
        bias_penalty: float,
        tier: CredibilityTier,
        freshness: float,
        citation_count: int = 0,
    ) -> SourceClassification:
        """Classify source based on composite signals.

        Foundational papers (highly cited, tier 1/2) get a freshness exemption
        because their impact transcends recency.
        """
        is_foundational = citation_count >= 1000 and tier in (CredibilityTier.TIER_1, CredibilityTier.TIER_2)

        if overall >= 0.80 and bias_penalty < 0.10 and tier in (CredibilityTier.TIER_1, CredibilityTier.TIER_2):
            return SourceClassification.TRUSTED
        # Foundational papers can be TRUSTED even with lower overall due to age
        if is_foundational and overall >= 0.55 and bias_penalty < 0.10:
            return SourceClassification.TRUSTED
        if overall >= 0.65 and bias_penalty < 0.15:
            return SourceClassification.VERIFIED
        if overall >= 0.40 or (overall >= 0.30 and freshness > 0.5):
            return SourceClassification.UNCERTAIN
        return SourceClassification.UNRELIABLE


    def verify_batch(self, sources: list[tuple[SourceMetadata, str]]) -> list[SourceQualityReport]:
        """Verify multiple sources and return ranked by overall_score."""
        reports = [self.verify(meta, content) for meta, content in sources]
        reports.sort(key=lambda r: r.overall_score, reverse=True)
        return reports


    # ── Report utilities ────────────────────────────────────────────────────

    def report_to_dict(self, report: SourceQualityReport) -> dict:
        """Convert report to serializable dict."""
        d = asdict(report)
        d["classification"] = report.classification.value
        d["credibility_tier"] = report.credibility_tier.value
        d["bias_indicators"] = [b.value for b in report.bias_indicators]
        return d


    def summary(self, reports: list[SourceQualityReport]) -> dict:
        """Generate summary statistics for a batch of reports."""
        if not reports:
            return {"count": 0, "by_classification": {}}
        by_class = {}
        for r in reports:
            c = r.classification.value
            by_class[c] = by_class.get(c, 0) + 1
        return {
            "count": len(reports),
            "by_classification": by_class,
            "avg_score": round(sum(r.overall_score for r in reports) / len(reports), 3),
            "top_source": reports[0].source_title if reports else None,
        }
