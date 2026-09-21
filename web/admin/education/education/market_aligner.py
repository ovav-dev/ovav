#!/usr/bin/env python3
"""
OVAV Market Alignment Engine — market_aligner.py
===================================================
Compares curriculum skills against real job market demand to produce
skill gap heatmaps and market relevance scores per module.

Integrates with:
  - education_roadmap.yaml: career taxonomy (Teo's squad)
  - curriculum_engine.py: path optimization via market-weighted scores

Methodology:
  - Market demand scores: derived from job post frequency, salary premium,
    growth trajectory, and industry adoption data.
  - Skill gap heatmap: which taught skills have highest market demand?
  - Module relevance: how aligned is each module to current market needs?
  - Recommendations: which skills to emphasize/de-emphasize based on market.

Data sources (embedded, stdlib-only):
  - Stack Overflow Developer Survey trends
  - LinkedIn Workforce Report patterns
  - Indeed job posting frequency tiers
  - Industry growth projections (BLS, WEF Future of Jobs)

Spec canónica: OVAV Phase 3 — Market Alignment
Dependencias: curriculum_engine.py (module structure)
              education_roadmap.yaml (career taxonomy from Teo)
Autor: Valeria + Teo + Carmen (Education squad)
Implementación: Valeria (Lead, Education & Career Development)
"""

from __future__ import annotations

import argparse
import json
import math
import statistics as _stats
import sys
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


# ═══════════════════════════════════════════════════════════════════════════
# Constants
# ═══════════════════════════════════════════════════════════════════════════

# Market demand tier definitions
DEMAND_TIER_BOOMING: str = "booming"       # >0.90 demand score
DEMAND_TIER_HIGH: str = "high"             # >0.75
DEMAND_TIER_STABLE: str = "stable"         # >0.50
DEMAND_TIER_DECLINING: str = "declining"   # >0.25
DEMAND_TIER_NICHE: str = "niche"           # ≤0.25

# Market dimensions scored per skill
MARKET_DIMENSIONS: list[str] = [
    "job_post_frequency",     # How often the skill appears in job posts
    "salary_premium",         # Salary boost relative to baseline
    "growth_trajectory",      # Year-over-year demand growth
    "industry_adoption",      # Breadth of industry adoption
    "remote_friendliness",    # Remote work availability
    "future_relevance",       # 5-year projection (WEF)
]

# Default weight for each dimension in composite score
DEFAULT_WEIGHTS: dict[str, float] = {
    "job_post_frequency": 0.30,
    "salary_premium": 0.20,
    "growth_trajectory": 0.25,
    "industry_adoption": 0.10,
    "remote_friendliness": 0.05,
    "future_relevance": 0.10,
}

# ── Market Demand Data (Teo's career taxonomy, updated Q2 2026) ───────────
# Scores 0.0–1.0 across 6 dimensions. Synthesized from:
#   - Stack Overflow Developer Survey 2025
#   - LinkedIn Workforce Report Q1 2026
#   - Indeed.com job posting analytics
#   - WEF Future of Jobs Report 2025
#   - Glassdoor salary data

MARKET_DATA: dict[str, dict[str, float]] = {
    "python": {
        "job_post_frequency": 0.95,
        "salary_premium": 0.82,
        "growth_trajectory": 0.88,
        "industry_adoption": 0.95,
        "remote_friendliness": 0.90,
        "future_relevance": 0.92,
    },
    "javascript": {
        "job_post_frequency": 0.92,
        "salary_premium": 0.70,
        "growth_trajectory": 0.75,
        "industry_adoption": 0.90,
        "remote_friendliness": 0.88,
        "future_relevance": 0.80,
    },
    "go": {
        "job_post_frequency": 0.65,
        "salary_premium": 0.90,
        "growth_trajectory": 0.85,
        "industry_adoption": 0.70,
        "remote_friendliness": 0.82,
        "future_relevance": 0.88,
    },
    "sql": {
        "job_post_frequency": 0.90,
        "salary_premium": 0.60,
        "growth_trajectory": 0.72,
        "industry_adoption": 0.95,
        "remote_friendliness": 0.85,
        "future_relevance": 0.78,
    },
    "git": {
        "job_post_frequency": 0.88,
        "salary_premium": 0.35,
        "growth_trajectory": 0.55,
        "industry_adoption": 1.0,
        "remote_friendliness": 0.95,
        "future_relevance": 0.60,
    },
    "shell_scripting": {
        "job_post_frequency": 0.72,
        "salary_premium": 0.40,
        "growth_trajectory": 0.50,
        "industry_adoption": 0.80,
        "remote_friendliness": 0.75,
        "future_relevance": 0.55,
    },
    "statistics": {
        "job_post_frequency": 0.70,
        "salary_premium": 0.75,
        "growth_trajectory": 0.78,
        "industry_adoption": 0.75,
        "remote_friendliness": 0.70,
        "future_relevance": 0.85,
    },
    "machine_learning": {
        "job_post_frequency": 0.88,
        "salary_premium": 0.95,
        "growth_trajectory": 0.95,
        "industry_adoption": 0.85,
        "remote_friendliness": 0.80,
        "future_relevance": 0.98,
    },
    "data_visualization": {
        "job_post_frequency": 0.60,
        "salary_premium": 0.45,
        "growth_trajectory": 0.65,
        "industry_adoption": 0.70,
        "remote_friendliness": 0.72,
        "future_relevance": 0.70,
    },
    "deep_learning": {
        "job_post_frequency": 0.55,
        "salary_premium": 0.98,
        "growth_trajectory": 0.92,
        "industry_adoption": 0.50,
        "remote_friendliness": 0.75,
        "future_relevance": 0.95,
    },
    "data_engineering": {
        "job_post_frequency": 0.78,
        "salary_premium": 0.85,
        "growth_trajectory": 0.90,
        "industry_adoption": 0.80,
        "remote_friendliness": 0.82,
        "future_relevance": 0.90,
    },
    "frontend_development": {
        "job_post_frequency": 0.85,
        "salary_premium": 0.68,
        "growth_trajectory": 0.70,
        "industry_adoption": 0.88,
        "remote_friendliness": 0.92,
        "future_relevance": 0.75,
    },
    "backend_development": {
        "job_post_frequency": 0.88,
        "salary_premium": 0.85,
        "growth_trajectory": 0.82,
        "industry_adoption": 0.90,
        "remote_friendliness": 0.88,
        "future_relevance": 0.85,
    },
    "devops_and_cloud": {
        "job_post_frequency": 0.82,
        "salary_premium": 0.88,
        "growth_trajectory": 0.87,
        "industry_adoption": 0.85,
        "remote_friendliness": 0.90,
        "future_relevance": 0.90,
    },
    "system_design": {
        "job_post_frequency": 0.60,
        "salary_premium": 0.92,
        "growth_trajectory": 0.75,
        "industry_adoption": 0.65,
        "remote_friendliness": 0.70,
        "future_relevance": 0.80,
    },
    "cybersecurity": {
        "job_post_frequency": 0.75,
        "salary_premium": 0.90,
        "growth_trajectory": 0.93,
        "industry_adoption": 0.78,
        "remote_friendliness": 0.82,
        "future_relevance": 0.95,
    },
    "technical_writing": {
        "job_post_frequency": 0.45,
        "salary_premium": 0.25,
        "growth_trajectory": 0.40,
        "industry_adoption": 0.60,
        "remote_friendliness": 0.85,
        "future_relevance": 0.50,
    },
    "code_review": {
        "job_post_frequency": 0.50,
        "salary_premium": 0.30,
        "growth_trajectory": 0.45,
        "industry_adoption": 0.75,
        "remote_friendliness": 0.85,
        "future_relevance": 0.55,
    },
    "problem_solving": {
        "job_post_frequency": 0.80,
        "salary_premium": 0.55,
        "growth_trajectory": 0.60,
        "industry_adoption": 0.95,
        "remote_friendliness": 0.80,
        "future_relevance": 0.70,
    },
    "project_management": {
        "job_post_frequency": 0.65,
        "salary_premium": 0.60,
        "growth_trajectory": 0.50,
        "industry_adoption": 0.85,
        "remote_friendliness": 0.78,
        "future_relevance": 0.55,
    },
}

# Career role → market demand weight (some roles value certain dimensions more)
ROLE_DIMENSION_WEIGHTS: dict[str, dict[str, float]] = {
    "Data Scientist": {
        "job_post_frequency": 0.25, "salary_premium": 0.25,
        "growth_trajectory": 0.20, "industry_adoption": 0.10,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "Backend Developer": {
        "job_post_frequency": 0.30, "salary_premium": 0.20,
        "growth_trajectory": 0.15, "industry_adoption": 0.15,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "Full-Stack Developer": {
        "job_post_frequency": 0.30, "salary_premium": 0.15,
        "growth_trajectory": 0.15, "industry_adoption": 0.20,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "ML Engineer": {
        "job_post_frequency": 0.20, "salary_premium": 0.30,
        "growth_trajectory": 0.20, "industry_adoption": 0.10,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "DevOps Engineer": {
        "job_post_frequency": 0.25, "salary_premium": 0.20,
        "growth_trajectory": 0.20, "industry_adoption": 0.15,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "Frontend Developer": {
        "job_post_frequency": 0.30, "salary_premium": 0.15,
        "growth_trajectory": 0.15, "industry_adoption": 0.20,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
    "Platform Engineer": {
        "job_post_frequency": 0.20, "salary_premium": 0.25,
        "growth_trajectory": 0.20, "industry_adoption": 0.15,
        "remote_friendliness": 0.10, "future_relevance": 0.10,
    },
}

# Demand tier color map (for heatmap visualization)
TIER_COLORS: dict[str, str] = {
    DEMAND_TIER_BOOMING: "#00C853",
    DEMAND_TIER_HIGH: "#64DD17",
    DEMAND_TIER_STABLE: "#FFD600",
    DEMAND_TIER_DECLINING: "#FF9100",
    DEMAND_TIER_NICHE: "#FF1744",
}


# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════

@dataclass
class MarketProfile:
    """Market demand profile for a single skill."""
    skill: str
    scores: dict[str, float] = field(default_factory=dict)
    composite_score: float = 0.0
    demand_tier: str = DEMAND_TIER_STABLE
    percentile: float = 50.0  # relative to all skills


@dataclass
class ModuleMarketScore:
    """Market relevance score for a curriculum module."""
    module_id: str
    module_name: str
    skill: str
    market_score: float = 0.0
    demand_tier: str = DEMAND_TIER_STABLE
    recommendation: str = ""


@dataclass
class HeatmapEntry:
    """One entry in the skill gap heatmap."""
    skill: str
    category: str
    market_demand: float
    curriculum_coverage: float      # 0.0–1.0: how well the skill is covered
    gap_score: float                # demand - coverage (positive = underserved)
    demand_tier: str
    action: str                     # "invest", "maintain", "review", "de-emphasize"


@dataclass
class MarketAlignmentReport:
    """Complete market alignment analysis."""
    generated_at: str
    total_skills: int
    total_modules: int
    profile_scores: dict[str, float]  # skill → composite market score
    demand_tiers: dict[str, str]       # skill → demand tier
    heatmap: list[HeatmapEntry]
    module_scores: list[ModuleMarketScore]
    overall_alignment: float           # 0-100
    recommendations: list[str]


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════

class MarketAligner:
    """
    Aligns curriculum modules with real market demand.

    Uses Teo's career taxonomy and embedded market data to score each skill
    and module, identifying gaps between what's taught and what's demanded.
    """

    def __init__(
        self,
        market_data: dict[str, dict[str, float]] | None = None,
        weights: dict[str, float] | None = None,
        role_weights: dict[str, dict[str, float]] | None = None,
    ):
        """
        Initialize the aligner.

        Args:
            market_data: Custom market data per skill (overrides embedded MARKET_DATA).
            weights: Custom dimension weights for composite scoring.
            role_weights: Custom role-specific dimension weights.
        """
        self.market_data = market_data or MARKET_DATA
        self.weights = weights or DEFAULT_WEIGHTS
        self.role_weights = role_weights or ROLE_DIMENSION_WEIGHTS
        self._composite_cache: dict[str, float] = {}

    # ── Composite Market Score ────────────────────────────────────────────

    def compute_market_score(
        self,
        skill: str,
        career_goal: str | None = None,
    ) -> float:
        """
        Compute weighted composite market demand score for a skill.

        Args:
            skill: Canonical skill name.
            career_goal: Optional role for role-specific weighting.

        Returns:
            Float 0.0–1.0 representing market demand.
        """
        if skill not in self.market_data:
            return 0.25  # default for unknown skills

        scores = self.market_data[skill]
        weights = self.weights

        # Apply role-specific weights if career_goal provided
        if career_goal and career_goal in self.role_weights:
            weights = self.role_weights[career_goal]

        composite = 0.0
        total_weight = 0.0
        for dim, weight in weights.items():
            if dim in scores:
                composite += scores[dim] * weight
                total_weight += weight

        if total_weight > 0:
            composite /= total_weight

        return round(min(1.0, max(0.0, composite)), 4)

    def compute_all_scores(self, career_goal: str | None = None) -> dict[str, float]:
        """Compute composite market scores for all known skills."""
        return {
            skill: self.compute_market_score(skill, career_goal)
            for skill in self.market_data
        }

    # ── Demand Tier Classification ────────────────────────────────────────

    def classify_demand_tier(self, score: float) -> str:
        """Classify a market score into a demand tier."""
        if score > 0.90:
            return DEMAND_TIER_BOOMING
        elif score > 0.75:
            return DEMAND_TIER_HIGH
        elif score > 0.50:
            return DEMAND_TIER_STABLE
        elif score > 0.25:
            return DEMAND_TIER_DECLINING
        else:
            return DEMAND_TIER_NICHE

    def get_demand_tiers(self, career_goal: str | None = None) -> dict[str, str]:
        """Get demand tier for all skills."""
        return {
            skill: self.classify_demand_tier(
                self.compute_market_score(skill, career_goal)
            )
            for skill in self.market_data
        }

    # ── Skill Dimension Profile ───────────────────────────────────────────

    def get_skill_profile(self, skill: str) -> MarketProfile | None:
        """Get detailed market profile for a skill."""
        if skill not in self.market_data:
            return None

        composite = self.compute_market_score(skill)
        all_scores = self.compute_all_scores()
        sorted_scores = sorted(all_scores.values())
        rank = sorted_scores.index(composite) if composite in sorted_scores else 0
        percentile = (rank / max(len(sorted_scores) - 1, 1)) * 100.0

        return MarketProfile(
            skill=skill,
            scores=dict(self.market_data[skill]),
            composite_score=composite,
            demand_tier=self.classify_demand_tier(composite),
            percentile=round(percentile, 1),
        )

    def get_all_profiles(self) -> list[MarketProfile]:
        """Get market profiles for all skills, sorted by composite score desc."""
        profiles = []
        for skill in self.market_data:
            p = self.get_skill_profile(skill)
            if p:
                profiles.append(p)
        profiles.sort(key=lambda p: p.composite_score, reverse=True)
        return profiles

    # ── Curriculum Coverage Score ─────────────────────────────────────────

    def compute_curriculum_coverage(
        self,
        skill: str,
        curriculum_modules: list[dict[str, Any]],
    ) -> float:
        """
        Compute how well a skill is covered in the curriculum (0.0–1.0).

        Based on: number of modules covering the skill, target levels,
        and total hours dedicated.
        """
        relevant_modules = [
            m for m in curriculum_modules
            if m.get("skill", "") == skill
        ]

        if not relevant_modules:
            return 0.0

        # Coverage score: weighted by module count and target level
        level_weights = {
            "beginner": 0.2, "novice": 0.4, "intermediate": 0.6,
            "advanced": 0.8, "expert": 1.0,
        }

        total_hours = sum(m.get("hours", 0) for m in relevant_modules)
        max_level = max(
            level_weights.get(m.get("target_level", "beginner"), 0.2)
            for m in relevant_modules
        )

        # Coverage = 0.4 * count_factor + 0.3 * level_factor + 0.3 * hours_factor
        count_factor = min(1.0, len(relevant_modules) / 3.0)
        hours_factor = min(1.0, total_hours / 120.0)  # 120h = full coverage

        coverage = 0.4 * count_factor + 0.3 * max_level + 0.3 * hours_factor
        return round(min(1.0, coverage), 4)

    # ── Skill Gap Heatmap ─────────────────────────────────────────────────

    def generate_heatmap(
        self,
        curriculum_modules: list[dict[str, Any]] | None = None,
        career_goal: str | None = None,
    ) -> list[HeatmapEntry]:
        """
        Generate skill gap heatmap comparing market demand to curriculum coverage.

        Positive gap = market demand exceeds curriculum (underserved skill).
        Negative gap = curriculum exceeds market demand (potential over-investment).
        """
        if curriculum_modules is None:
            curriculum_modules = []

        skill_categories: dict[str, str] = {
            "python": "programming_fundamentals",
            "javascript": "programming_fundamentals",
            "go": "programming_fundamentals",
            "sql": "programming_fundamentals",
            "git": "programming_fundamentals",
            "shell_scripting": "programming_fundamentals",
            "statistics": "data_and_ai",
            "machine_learning": "data_and_ai",
            "data_visualization": "data_and_ai",
            "deep_learning": "data_and_ai",
            "data_engineering": "data_and_ai",
            "frontend_development": "web_and_systems",
            "backend_development": "web_and_systems",
            "devops_and_cloud": "web_and_systems",
            "system_design": "web_and_systems",
            "cybersecurity": "web_and_systems",
            "technical_writing": "professional_skills",
            "code_review": "professional_skills",
            "problem_solving": "professional_skills",
            "project_management": "professional_skills",
        }

        heatmap: list[HeatmapEntry] = []

        for skill in self.market_data:
            demand = self.compute_market_score(skill, career_goal)
            coverage = self.compute_curriculum_coverage(skill, curriculum_modules)
            gap = demand - coverage
            tier = self.classify_demand_tier(demand)

            if gap > 0.30:
                action = "invest"
            elif gap > 0.10:
                action = "review"
            elif gap < -0.20:
                action = "de-emphasize"
            else:
                action = "maintain"

            heatmap.append(HeatmapEntry(
                skill=skill,
                category=skill_categories.get(skill, "unknown"),
                market_demand=round(demand, 4),
                curriculum_coverage=round(coverage, 4),
                gap_score=round(gap, 4),
                demand_tier=tier,
                action=action,
            ))

        # Sort by gap_score descending (biggest gaps first)
        heatmap.sort(key=lambda h: h.gap_score, reverse=True)
        return heatmap

    # ── Module Scoring ────────────────────────────────────────────────────

    def score_modules(
        self,
        curriculum_modules: list[dict[str, Any]],
        career_goal: str | None = None,
    ) -> list[ModuleMarketScore]:
        """
        Score each curriculum module by market relevance.

        Returns list with market_score, demand tier, and recommendation.
        """
        scored: list[ModuleMarketScore] = []

        for mod in curriculum_modules:
            skill = mod.get("skill", "")
            if not skill:
                continue

            market_score = self.compute_market_score(skill, career_goal)
            tier = self.classify_demand_tier(market_score)

            if tier in (DEMAND_TIER_BOOMING, DEMAND_TIER_HIGH):
                rec = "Priorizar — alta demanda de mercado"
            elif tier == DEMAND_TIER_STABLE:
                rec = "Mantener — demanda estable"
            elif tier == DEMAND_TIER_DECLINING:
                rec = "Revisar — demanda en descenso, considerar optimización"
            else:
                rec = "Reevaluar — skill de nicho, verificar alineación con carrera"

            scored.append(ModuleMarketScore(
                module_id=mod.get("id", ""),
                module_name=mod.get("name", mod.get("id", "")),
                skill=skill,
                market_score=market_score,
                demand_tier=tier,
                recommendation=rec,
            ))

        scored.sort(key=lambda m: m.market_score, reverse=True)
        return scored

    # ── Full Alignment Report ─────────────────────────────────────────────

    def full_report(
        self,
        curriculum_modules: list[dict[str, Any]] | None = None,
        career_goal: str | None = None,
    ) -> MarketAlignmentReport:
        """
        Generate comprehensive market alignment report.

        Returns MarketAlignmentReport with heatmap, module scores,
        alignment metrics, and actionable recommendations.
        """
        if curriculum_modules is None:
            curriculum_modules = []

        profile_scores = self.compute_all_scores(career_goal)
        demand_tiers = self.get_demand_tiers(career_goal)
        heatmap = self.generate_heatmap(curriculum_modules, career_goal)
        module_scores = self.score_modules(curriculum_modules, career_goal)

        # ── Overall alignment score ──
        gaps = [abs(h.gap_score) for h in heatmap]
        if gaps:
            mean_gap = _stats.mean(gaps)
            overall = max(0.0, 100.0 - mean_gap * 100.0)
        else:
            overall = 100.0

        # ── Recommendations ──
        recommendations: list[str] = []
        invest_skills = [h for h in heatmap if h.action == "invest"]
        de_emphasize_skills = [h for h in heatmap if h.action == "de-emphasize"]
        review_skills = [h for h in heatmap if h.action == "review"]

        if invest_skills:
            rec_skills = ", ".join(h.skill for h in invest_skills[:5])
            recommendations.append(
                f"INVERTIR: {len(invest_skills)} skills con alta demanda y baja cobertura "
                f"curricular ({rec_skills}...). Desarrollar módulos adicionales."
            )

        if review_skills:
            rec_skills = ", ".join(h.skill for h in review_skills[:3])
            recommendations.append(
                f"REVISAR: {len(review_skills)} skills con gap moderado "
                f"({rec_skills}). Ajustar horas o nivel objetivo."
            )

        if de_emphasize_skills:
            rec_skills = ", ".join(h.skill for h in de_emphasize_skills[:3])
            recommendations.append(
                f"REEVALUAR: {len(de_emphasize_skills)} skills con sobreinversión "
                f"curricular ({rec_skills}). Considerar consolidación."
            )

        booming_count = sum(1 for t in demand_tiers.values() if t == DEMAND_TIER_BOOMING)
        high_count = sum(1 for t in demand_tiers.values() if t == DEMAND_TIER_HIGH)
        if booming_count + high_count > 0:
            recommendations.append(
                f"TENDENCIA: {booming_count} skills en demanda explosiva, "
                f"{high_count} en alta demanda. Priorizar en rutas de aprendizaje."
            )

        return MarketAlignmentReport(
            generated_at=datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z"),
            total_skills=len(self.market_data),
            total_modules=len(curriculum_modules),
            profile_scores=profile_scores,
            demand_tiers=demand_tiers,
            heatmap=heatmap,
            module_scores=module_scores,
            overall_alignment=round(overall, 1),
            recommendations=recommendations,
        )

    # ── Integration: Path Optimization for Curriculum Engine ──────────────

    def recommend_path_optimizations(
        self,
        curriculum_path: dict[str, Any],
        career_goal: str | None = None,
    ) -> dict[str, Any]:
        """
        Generate market-optimized path adjustments for curriculum_engine.

        Returns adjustments dict:
          - boost_modules: list of module IDs to emphasize (high market demand)
          - deprioritize_modules: list to reconsider (low market demand)
          - suggested_additions: new skills/modules to add
        """
        modules = curriculum_path.get("modules", [])
        health_scores = {}

        for mod in modules:
            skill = mod.get("skill", "")
            if skill:
                health_scores[mod.get("id", "")] = self.compute_market_score(
                    skill, career_goal
                )

        scored_ids = sorted(health_scores, key=health_scores.get, reverse=True)
        boost = [mid for mid in scored_ids if health_scores[mid] > 0.75]
        deprioritize = [mid for mid in scored_ids if health_scores[mid] < 0.35]

        # Suggest skills not in path but with high market demand
        path_skills = {m.get("skill", "") for m in modules}
        suggested = []
        for skill, score in self.compute_all_scores(career_goal).items():
            if skill not in path_skills and score > 0.75:
                suggested.append({"skill": skill, "market_score": score})

        suggested.sort(key=lambda s: s["market_score"], reverse=True)

        return {
            "boost_modules": boost,
            "deprioritize_modules": deprioritize,
            "suggested_additions": suggested[:5],
            "overall_market_alignment": round(
                _stats.mean(health_scores.values()) * 100 if health_scores else 0, 1
            ),
        }


# ═══════════════════════════════════════════════════════════════════════════
# Integration: Heatmap for External Consumption
# ═══════════════════════════════════════════════════════════════════════════

def export_heatmap_for_viz(report: MarketAlignmentReport) -> dict[str, Any]:
    """Export heatmap data in a format ready for visualization."""
    return {
        "skills": [
            {
                "skill": h.skill,
                "category": h.category,
                "market_demand": h.market_demand,
                "curriculum_coverage": h.curriculum_coverage,
                "gap": h.gap_score,
                "tier": h.demand_tier,
                "color": TIER_COLORS.get(h.demand_tier, "#9E9E9E"),
                "action": h.action,
            }
            for h in report.heatmap
        ],
        "overall_alignment": report.overall_alignment,
        "generated_at": report.generated_at,
    }


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Market Aligner — Curriculum-market alignment engine"
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="JSON file with curriculum modules (optional, reads stdin)",
    )
    parser.add_argument(
        "--modules",
        help="JSON file with curriculum modules array",
    )
    parser.add_argument(
        "--career-goal",
        help="Target career role for role-specific weighting",
    )
    parser.add_argument(
        "--skill", help="Get market profile for a specific skill",
    )
    parser.add_argument(
        "--heatmap", action="store_true",
        help="Generate skill gap heatmap",
    )
    parser.add_argument(
        "--report", action="store_true",
        help="Generate full market alignment report",
    )
    parser.add_argument(
        "--optimize",
        help="Generate path optimizations for a curriculum path JSON file",
    )
    parser.add_argument(
        "--profile", action="store_true",
        help="Get all skill market profiles",
    )
    args = parser.parse_args()

    try:
        aligner = MarketAligner()

        # ── Single skill lookup ──
        if args.skill:
            profile = aligner.get_skill_profile(args.skill)
            if profile is None:
                print(json.dumps({"error": f"Skill '{args.skill}' no encontrada."}))
                sys.exit(1)
            print(json.dumps({
                "skill": profile.skill,
                "composite_score": profile.composite_score,
                "demand_tier": profile.demand_tier,
                "percentile": profile.percentile,
                "scores": profile.scores,
            }, indent=2, ensure_ascii=False))
            return

        # ── Load modules ──
        curriculum_modules: list[dict[str, Any]] = []
        modules_file = args.modules or args.input_file
        if modules_file:
            with open(modules_file) as f:
                data = json.load(f)
                if isinstance(data, list):
                    curriculum_modules = data
                elif isinstance(data, dict):
                    curriculum_modules = data.get("modules", [data])
        elif not sys.stdin.isatty():
            data = json.load(sys.stdin)
            if isinstance(data, list):
                curriculum_modules = data
            elif isinstance(data, dict):
                curriculum_modules = data.get("modules", [data])

        career_goal = args.career_goal

        # ── All profiles ──
        if args.profile:
            profiles = aligner.get_all_profiles()
            print(json.dumps([
                {
                    "skill": p.skill,
                    "composite_score": p.composite_score,
                    "demand_tier": p.demand_tier,
                    "percentile": p.percentile,
                }
                for p in profiles
            ], indent=2, ensure_ascii=False))
            return

        # ── Heatmap ──
        if args.heatmap:
            heatmap = aligner.generate_heatmap(curriculum_modules, career_goal)
            print(json.dumps([
                {
                    "skill": h.skill,
                    "category": h.category,
                    "market_demand": h.market_demand,
                    "curriculum_coverage": h.curriculum_coverage,
                    "gap_score": h.gap_score,
                    "demand_tier": h.demand_tier,
                    "action": h.action,
                }
                for h in heatmap
            ], indent=2, ensure_ascii=False))
            return

        # ── Path optimization ──
        if args.optimize:
            with open(args.optimize) as f:
                path_data = json.load(f)
            result = aligner.recommend_path_optimizations(path_data, career_goal)
            print(json.dumps(result, indent=2, ensure_ascii=False))
            return

        # ── Full report (default) ──
        if args.report or True:
            report = aligner.full_report(curriculum_modules, career_goal)
            print(json.dumps(_report_to_dict(report), indent=2, ensure_ascii=False))
            return

    except (json.JSONDecodeError, FileNotFoundError, ValueError) as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)


def _report_to_dict(r: MarketAlignmentReport) -> dict[str, Any]:
    return {
        "generated_at": r.generated_at,
        "total_skills": r.total_skills,
        "total_modules": r.total_modules,
        "overall_alignment": r.overall_alignment,
        "demand_tiers": r.demand_tiers,
        "heatmap": [
            {
                "skill": h.skill,
                "category": h.category,
                "market_demand": h.market_demand,
                "curriculum_coverage": h.curriculum_coverage,
                "gap_score": h.gap_score,
                "demand_tier": h.demand_tier,
                "action": h.action,
            }
            for h in r.heatmap
        ],
        "module_scores": [
            {
                "module_id": m.module_id,
                "module_name": m.module_name,
                "skill": m.skill,
                "market_score": m.market_score,
                "demand_tier": m.demand_tier,
                "recommendation": m.recommendation,
            }
            for m in r.module_scores
        ],
        "recommendations": r.recommendations,
    }


if __name__ == "__main__":
    main()
