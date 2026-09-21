#!/usr/bin/env python3
"""
OVAV Bias & Fairness Auditor — bias_auditor.py
================================================
Audits the education pipeline for bias across three dimensions:
  1. Demographic fairness — do different groups get equitable recommendations?
  2. Skill prerequisite bias — are certain skill groups over/under-weighted?
  3. Content representation — are all career paths equally served?

Integrates with knowledge_tracer mastery data and feeds bias-free path
adjustments into curriculum_engine.

Methodology:
  - Statistical parity: compares P(mastery) and P(recommendation) across groups
  - Disparate impact ratio: (favorable outcome for protected group) / (favorable for reference)
    Flagged when ratio < 0.80 (industry standard, EEOC 4/5ths rule)
  - Representation check: ensures all career goals have adequate module coverage

Spec canónica: OVAV Phase 3 — Bias & Fairness Audit
Dependencias: knowledge_tracer.py (mastery/level estimates)
              curriculum_engine.py (path structure)
Autor: Valeria + Alicia (Education squad)
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

# Disparate impact threshold (EEOC 4/5ths rule)
DISPARATE_IMPACT_THRESHOLD: float = 0.80

# Minimum group size for statistical validity
MIN_GROUP_SIZE: int = 3

# Severity levels for audit findings
SEVERITY_CRITICAL: str = "critical"
SEVERITY_HIGH: str = "high"
SEVERITY_MEDIUM: str = "medium"
SEVERITY_LOW: str = "low"
SEVERITY_INFO: str = "info"

# Protected demographic attributes
PROTECTED_ATTRIBUTES: list[str] = [
    "gender",
    "age_group",
    "education_level",
    "primary_language",
    "geographic_region",
]

# Known skill categories from education_roadmap taxonomy
SKILL_CATEGORIES: dict[str, str] = {
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

# Career goals and their target skill sets
CAREER_GOAL_SKILLS: dict[str, list[str]] = {
    "Data Scientist": [
        "python", "sql", "statistics", "machine_learning",
        "data_visualization", "data_engineering"
    ],
    "Backend Developer": [
        "python", "go", "sql", "git", "backend_development",
        "system_design", "devops_and_cloud"
    ],
    "Full-Stack Developer": [
        "python", "javascript", "sql", "git", "frontend_development",
        "backend_development", "devops_and_cloud"
    ],
    "ML Engineer": [
        "python", "statistics", "machine_learning", "deep_learning",
        "data_engineering", "sql"
    ],
    "DevOps Engineer": [
        "python", "go", "shell_scripting", "devops_and_cloud",
        "system_design", "cybersecurity", "git"
    ],
    "Frontend Developer": [
        "javascript", "frontend_development", "git",
        "data_visualization", "code_review"
    ],
}


# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════

@dataclass
class DemographicProfile:
    """Student demographic profile for fairness auditing."""
    gender: str = "unknown"
    age_group: str = "unknown"          # "18-24", "25-34", "35-44", "45+"
    education_level: str = "unknown"     # "high_school", "bachelors", "masters", "phd"
    primary_language: str = "unknown"
    geographic_region: str = "unknown"   # "LATAM", "APAC", "EU", "NA", "AF"


@dataclass
class BiasFinding:
    """A single bias finding from the audit."""
    dimension: str           # "demographic", "prerequisite", "representation"
    severity: str            # "critical", "high", "medium", "low", "info"
    attribute: str           # the attribute checked (e.g., "gender")
    description: str
    impact_ratio: float | None = None
    affected_groups: list[str] = field(default_factory=list)
    affected_skills: list[str] = field(default_factory=list)
    recommendation: str = ""


@dataclass
class AuditResult:
    """Complete bias audit result."""
    audited_at: str
    student_count: int
    findings: list[BiasFinding] = field(default_factory=list)
    fairness_scores: dict[str, float] = field(default_factory=dict)
    demographic_breakdown: dict[str, dict[str, int]] = field(default_factory=dict)
    overall_rating: str = "fair"  # "fair", "needs_review", "biased"
    total_flags: int = 0


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════

class BiasAuditor:
    """
    Audits the education pipeline for bias and fairness issues.

    Takes knowledge_tracer export data and demographic profiles to detect
    systemic biases in skill assessment, path recommendation, and content
    representation.
    """

    def __init__(
        self,
        tracer_data: dict[str, Any] | None = None,
        student_profiles: list[dict[str, Any]] | None = None,
    ):
        """
        Initialize the auditor.

        Args:
            tracer_data: Export from knowledge_tracer (masteries, levels, params).
            student_profiles: List of dicts with student_id and demographic attributes.
        """
        self.tracer_data = tracer_data or {}
        self.profiles: dict[str, DemographicProfile] = {}

        if student_profiles:
            for p in student_profiles:
                sid = p.get("student_id", "")
                if sid:
                    self.profiles[sid] = DemographicProfile(
                        gender=p.get("gender", "unknown"),
                        age_group=p.get("age_group", "unknown"),
                        education_level=p.get("education_level", "unknown"),
                        primary_language=p.get("primary_language", "unknown"),
                        geographic_region=p.get("geographic_region", "unknown"),
                    )

        self._findings: list[BiasFinding] = []
        self._cohort_data: dict[str, dict[str, list[float]]] = {}

    # ── Grouping Utilities ────────────────────────────────────────────────

    def _group_by_attribute(
        self, mastery_data: dict[str, dict[str, float]], attribute: str
    ) -> dict[str, list[float]]:
        """
        Group mastery scores by a demographic attribute.

        Returns dict of group_value → list of mastery scores.
        """
        groups: dict[str, list[float]] = {}
        for sid, profile in self.profiles.items():
            if sid not in mastery_data:
                continue
            attr_val = getattr(profile, attribute, "unknown")
            if attr_val not in groups:
                groups[attr_val] = []
            groups[attr_val].append(mastery_data[sid])
        return groups

    def _compute_disparate_impact(
        self, groups: dict[str, list[float]], reference_group: str
    ) -> dict[str, float]:
        """
        Compute disparate impact ratio for each group vs the reference group.
        ratio = mean(group) / mean(reference)
        """
        if reference_group not in groups:
            return {}

        ref_mean = _stats.mean(groups[reference_group]) if groups[reference_group] else 1.0
        if ref_mean == 0.0:
            ref_mean = 0.001  # avoid division by zero

        ratios: dict[str, float] = {}
        for group, values in groups.items():
            if group == reference_group:
                ratios[group] = 1.0
                continue
            if len(values) < MIN_GROUP_SIZE:
                ratios[group] = 1.0  # insufficient data, assume parity
                continue
            group_mean = _stats.mean(values) if values else 0.0
            ratios[group] = group_mean / ref_mean

        return ratios

    # ── 1. Demographic Fairness Audit ─────────────────────────────────────

    def audit_demographic_fairness(
        self, mastery_data: dict[str, dict[str, float]] | None = None
    ) -> list[BiasFinding]:
        """
        Check if different demographic groups receive equitable mastery
        estimates and recommendations.

        Args:
            mastery_data: Dict of student_id → {skill: mastery_probability}.
                          If None, extracts from tracer_data.
        """
        if mastery_data is None:
            mastery_data = self._extract_masteries_from_tracer()

        findings: list[BiasFinding] = []

        for attribute in PROTECTED_ATTRIBUTES:
            # Compute per-student average mastery
            student_avg_mastery: dict[str, float] = {}
            for sid, skills in mastery_data.items():
                if skills:
                    student_avg_mastery[sid] = _stats.mean(skills.values())

            groups = self._group_by_attribute(student_avg_mastery, attribute)
            if len(groups) < 2:
                continue

            # Use the largest group as reference
            ref_group = max(groups, key=lambda g: len(groups[g]))
            ratios = self._compute_disparate_impact(groups, ref_group)

            for group, ratio in ratios.items():
                if group == ref_group:
                    continue
                if ratio < DISPARATE_IMPACT_THRESHOLD:
                    severity = SEVERITY_CRITICAL if ratio < 0.60 else (
                        SEVERITY_HIGH if ratio < 0.70 else SEVERITY_MEDIUM
                    )
                    findings.append(BiasFinding(
                        dimension="demographic",
                        severity=severity,
                        attribute=attribute,
                        description=(
                            f"Group '{group}' has {ratio:.2f}x the average mastery of "
                            f"reference group '{ref_group}' for attribute '{attribute}'. "
                            f"Disparate impact ratio below 0.80 threshold."
                        ),
                        impact_ratio=round(ratio, 4),
                        affected_groups=[group, ref_group],
                        recommendation=(
                            f"Review assessment items for {attribute} bias. "
                            f"Ensure equitable access to prerequisite resources for {group}."
                        ),
                    ))

                # Also flag advantaged groups (ratio > 1.25)
                if ratio > 1.25:
                    findings.append(BiasFinding(
                        dimension="demographic",
                        severity=SEVERITY_LOW,
                        attribute=attribute,
                        description=(
                            f"Group '{group}' has {ratio:.2f}x the average mastery of "
                            f"reference group '{ref_group}'. Potential over-recommendation."
                        ),
                        impact_ratio=round(ratio, 4),
                        affected_groups=[group, ref_group],
                        recommendation=f"Verify that {group} is not receiving simplified assessments.",
                    ))

        return findings

    # ── 2. Skill Prerequisite Bias Audit ──────────────────────────────────

    def audit_prerequisite_bias(
        self,
        mastery_data: dict[str, dict[str, float]] | None = None,
        recommended_paths: list[dict[str, Any]] | None = None,
    ) -> list[BiasFinding]:
        """
        Check if certain skill groups are systematically over-recommended
        or under-assessed across demographic groups.

        Analyzes:
          - Category concentration: are students from certain groups
            routed predominantly to specific skill categories?
          - Prerequisite depth: are some groups given longer paths?
        """
        if mastery_data is None:
            mastery_data = self._extract_masteries_from_tracer()

        findings: list[BiasFinding] = []

        # ── Category concentration check ──
        # For each demographic attribute, check if skill category
        # distribution differs significantly across groups
        for attribute in PROTECTED_ATTRIBUTES:
            # Compute per-group per-category average mastery
            group_category_scores: dict[str, dict[str, list[float]]] = {}

            for sid, skills in mastery_data.items():
                if sid not in self.profiles:
                    continue
                attr_val = getattr(self.profiles[sid], attribute, "unknown")

                if attr_val not in group_category_scores:
                    group_category_scores[attr_val] = {}

                for skill, mastery in skills.items():
                    cat = SKILL_CATEGORIES.get(skill, "unknown")
                    if cat not in group_category_scores[attr_val]:
                        group_category_scores[attr_val][cat] = []
                    group_category_scores[attr_val][cat].append(mastery)

            # Compare category means across groups
            categories = set()
            for g in group_category_scores.values():
                categories.update(g.keys())

            for cat in categories:
                cat_means: dict[str, float] = {}
                for group, cats in group_category_scores.items():
                    if cat in cats and len(cats[cat]) >= MIN_GROUP_SIZE:
                        cat_means[group] = _stats.mean(cats[cat])

                if len(cat_means) < 2:
                    continue

                max_mean = max(cat_means.values())
                min_mean = min(cat_means.values())
                # Use disparate impact ratio instead of σ-based (robust with few groups)
                if max_mean > 0 and min_mean / max_mean < DISPARATE_IMPACT_THRESHOLD:
                    disadvantaged = min(cat_means, key=cat_means.get)
                    advantaged = max(cat_means, key=cat_means.get)
                    ratio = min_mean / max_mean
                    severity = SEVERITY_CRITICAL if ratio < 0.50 else SEVERITY_HIGH
                    findings.append(BiasFinding(
                        dimension="prerequisite",
                        severity=severity,
                        attribute=attribute,
                        description=(
                            f"Group '{disadvantaged}' has {min_mean:.2f} avg mastery in "
                            f"category '{cat}' vs '{advantaged}' at {max_mean:.2f}. "
                            f"Disparate impact ratio {ratio:.2f} below 0.80 threshold."
                        ),
                        impact_ratio=round(ratio, 4),
                        affected_groups=[disadvantaged, advantaged],
                        affected_skills=[cat],
                        recommendation=(
                            f"Audit assessment items in '{cat}' for "
                            f"{attribute}-specific bias. Check prerequisite structure."
                        ),
                    ))

        # ── Path length check ──
        if recommended_paths:
            path_lengths_by_group: dict[str, dict[str, list[int]]] = {}
            for path in recommended_paths:
                sid = path.get("student_id", "")
                if sid not in self.profiles:
                    continue
                for attr in PROTECTED_ATTRIBUTES:
                    attr_val = getattr(self.profiles[sid], attr, "unknown")
                    if attr not in path_lengths_by_group:
                        path_lengths_by_group[attr] = {}
                    if attr_val not in path_lengths_by_group[attr]:
                        path_lengths_by_group[attr][attr_val] = []
                    modules = path.get("modules", [])
                    path_lengths_by_group[attr][attr_val].append(len(modules))

            for attr, groups in path_lengths_by_group.items():
                if len(groups) < 2:
                    continue
                group_means = {
                    g: _stats.mean(lens) for g, lens in groups.items()
                    if len(lens) >= MIN_GROUP_SIZE
                }
                if len(group_means) < 2:
                    continue

                max_mean = max(group_means.values())
                min_mean = min(group_means.values())
                if max_mean > 0 and min_mean / max_mean < DISPARATE_IMPACT_THRESHOLD:
                    shortest = min(group_means, key=group_means.get)
                    longest = max(group_means, key=group_means.get)
                    findings.append(BiasFinding(
                        dimension="prerequisite",
                        severity=SEVERITY_MEDIUM,
                        attribute=attr,
                        description=(
                            f"Path length disparity: '{shortest}' avg {group_means[shortest]:.1f} "
                            f"modules vs '{longest}' avg {group_means[longest]:.1f} modules."
                        ),
                        affected_groups=[shortest, longest],
                        recommendation=(
                            "Verify path length differences are justified by actual "
                            "skill gaps, not demographic correlation."
                        ),
                    ))

        return findings

    # ── 3. Content Representation Audit ───────────────────────────────────

    def audit_content_representation(
        self,
        curriculum_modules: list[dict[str, Any]] | None = None,
    ) -> list[BiasFinding]:
        """
        Check if all career goals and skill categories have adequate
        module coverage. Detects content deserts.

        Args:
            curriculum_modules: List of module dicts from curriculum_engine
                                (each with 'skill', 'target_level', 'career_goal').
        """
        findings: list[BiasFinding] = []

        # ── Career goal coverage ──
        covered_goals: dict[str, set[str]] = {}
        if curriculum_modules:
            for mod in curriculum_modules:
                career = mod.get("career_goal", "general")
                skill = mod.get("skill", "")
                if career not in covered_goals:
                    covered_goals[career] = set()
                if skill:
                    covered_goals[career].add(skill)

        for goal, required_skills in CAREER_GOAL_SKILLS.items():
            covered = covered_goals.get(goal, set())
            missing = set(required_skills) - covered
            coverage_pct = 1.0 - (len(missing) / len(required_skills))

            if coverage_pct < 0.50:
                findings.append(BiasFinding(
                    dimension="representation",
                    severity=SEVERITY_CRITICAL,
                    attribute="career_goal",
                    description=(
                        f"Career goal '{goal}' has only {coverage_pct:.0%} skill coverage. "
                        f"Missing: {sorted(missing)}."
                    ),
                    affected_skills=sorted(missing),
                    recommendation=f"Develop modules for missing skills in '{goal}' track.",
                ))
            elif coverage_pct < 0.75:
                findings.append(BiasFinding(
                    dimension="representation",
                    severity=SEVERITY_HIGH,
                    attribute="career_goal",
                    description=(
                        f"Career goal '{goal}' has {coverage_pct:.0%} skill coverage. "
                        f"Missing: {sorted(missing)}."
                    ),
                    affected_skills=sorted(missing),
                    recommendation=f"Prioritize module development for {sorted(missing)}.",
                ))
            elif missing:
                findings.append(BiasFinding(
                    dimension="representation",
                    severity=SEVERITY_LOW,
                    attribute="career_goal",
                    description=(
                        f"Career goal '{goal}' missing {sorted(missing)}. "
                        f"Coverage at {coverage_pct:.0%} — acceptable but improvable."
                    ),
                    affected_skills=sorted(missing),
                    recommendation=f"Add {sorted(missing)} to backlog.",
                ))

        # ── Skill category balance ──
        if curriculum_modules:
            cat_counts: dict[str, int] = {}
            for mod in curriculum_modules:
                skill = mod.get("skill", "")
                cat = SKILL_CATEGORIES.get(skill, "unknown")
                cat_counts[cat] = cat_counts.get(cat, 0) + 1

            total = sum(cat_counts.values()) if cat_counts else 1
            expected_share = 1.0 / max(len(set(SKILL_CATEGORIES.values())), 1)

            for cat in set(SKILL_CATEGORIES.values()):
                share = cat_counts.get(cat, 0) / total
                if share < expected_share * 0.3:
                    findings.append(BiasFinding(
                        dimension="representation",
                        severity=SEVERITY_MEDIUM,
                        attribute="skill_category",
                        description=(
                            f"Category '{cat}' represents only {share:.1%} of modules "
                            f"(expected ~{expected_share:.1%}). Content desert risk."
                        ),
                        affected_groups=[cat],
                        recommendation=f"Expand curriculum coverage in '{cat}' category.",
                    ))

        return findings

    # ── 4. Full Audit ─────────────────────────────────────────────────────

    def full_audit(
        self,
        mastery_data: dict[str, dict[str, float]] | None = None,
        recommended_paths: list[dict[str, Any]] | None = None,
        curriculum_modules: list[dict[str, Any]] | None = None,
    ) -> AuditResult:
        """
        Run all three audit dimensions and produce a comprehensive report.

        Returns AuditResult with all findings, fairness scores, and recommendations.
        """
        if mastery_data is None:
            mastery_data = self._extract_masteries_from_tracer()

        all_findings: list[BiasFinding] = []
        all_findings.extend(self.audit_demographic_fairness(mastery_data))
        all_findings.extend(self.audit_prerequisite_bias(mastery_data, recommended_paths))
        all_findings.extend(self.audit_content_representation(curriculum_modules))

        # ── Compute fairness scores ──
        fairness_scores = self._compute_fairness_scores(all_findings)

        # ── Demographic breakdown ──
        demo_breakdown = self._build_demographic_breakdown()

        # ── Overall rating ──
        critical_count = sum(1 for f in all_findings if f.severity == SEVERITY_CRITICAL)
        high_count = sum(1 for f in all_findings if f.severity == SEVERITY_HIGH)

        if critical_count > 0:
            overall = "biased"
        elif high_count > 1:
            overall = "needs_review"
        elif any(f for f in all_findings if f.severity == SEVERITY_MEDIUM):
            overall = "needs_review"
        else:
            overall = "fair"

        return AuditResult(
            audited_at=datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z"),
            student_count=len(mastery_data),
            findings=all_findings,
            fairness_scores=fairness_scores,
            demographic_breakdown=demo_breakdown,
            overall_rating=overall,
            total_flags=len(all_findings),
        )

    # ── Helpers ───────────────────────────────────────────────────────────

    def _extract_masteries_from_tracer(self) -> dict[str, dict[str, float]]:
        """Extract per-student mastery data from tracer_data export."""
        result: dict[str, dict[str, float]] = {}

        # Handle single-student tracer export format
        masteries = self.tracer_data.get("masteries")
        if masteries and isinstance(masteries, dict):
            sid = self.tracer_data.get("student_id", "unknown")
            result[sid] = dict(masteries)

        # Handle multi-student format: {"students": [{"student_id": ..., "masteries": {...}}]}
        students = self.tracer_data.get("students", [])
        for s in students:
            sid = s.get("student_id", "")
            m = s.get("masteries", {})
            if sid and m:
                result[sid] = dict(m)

        return result

    def _compute_fairness_scores(self, findings: list[BiasFinding]) -> dict[str, float]:
        """Compute per-dimension fairness scores (0-100, higher is fairer)."""
        dim_counts: dict[str, int] = {"demographic": 0, "prerequisite": 0, "representation": 0}
        dim_severity_scores: dict[str, float] = {
            "demographic": 0.0, "prerequisite": 0.0, "representation": 0.0
        }

        severity_weights = {
            SEVERITY_CRITICAL: 5,
            SEVERITY_HIGH: 3,
            SEVERITY_MEDIUM: 1,
            SEVERITY_LOW: 0.5,
            SEVERITY_INFO: 0.1,
        }

        for finding in findings:
            dim_counts[finding.dimension] = dim_counts.get(finding.dimension, 0) + 1
            weight = severity_weights.get(finding.severity, 0.1)
            dim_severity_scores[finding.dimension] = (
                dim_severity_scores.get(finding.dimension, 0.0) + weight
            )

        scores: dict[str, float] = {}
        for dim in dim_counts:
            count = dim_counts[dim]
            penalty = dim_severity_scores.get(dim, 0.0)
            # Score starts at 100, decreases with findings
            if count == 0:
                scores[dim] = 100.0
            else:
                scores[dim] = max(0.0, 100.0 - penalty * 10.0)

        return scores

    def _build_demographic_breakdown(self) -> dict[str, dict[str, int]]:
        """Build counts of students per demographic attribute value."""
        breakdown: dict[str, dict[str, int]] = {}
        for attr in PROTECTED_ATTRIBUTES:
            breakdown[attr] = {}
            for profile in self.profiles.values():
                val = getattr(profile, attr, "unknown")
                breakdown[attr][val] = breakdown[attr].get(val, 0) + 1
        return breakdown


# ═══════════════════════════════════════════════════════════════════════════
# Integration: Generate Bias Report for Curriculum Engine
# ═══════════════════════════════════════════════════════════════════════════

def generate_bias_free_path_adjustments(
    audit_result: AuditResult,
    curriculum_path: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """
    Convert audit findings into actionable path adjustments for curriculum_engine.

    Returns a dict with adjustments that curriculum_engine can apply:
      - remove_bias_modules: list of module IDs to review
      - add_equity_modules: list of new modules to add
      - review_skills: skills needing reassessment
    """
    adjustments: dict[str, Any] = {
        "remove_bias_modules": [],
        "add_equity_modules": [],
        "review_skills": [],
        "recommendations": [],
    }

    for finding in audit_result.findings:
        if finding.severity in (SEVERITY_CRITICAL, SEVERITY_HIGH):
            adjustments["review_skills"].extend(finding.affected_skills)
            adjustments["recommendations"].append({
                "dimension": finding.dimension,
                "severity": finding.severity,
                "recommendation": finding.recommendation,
            })

    # Deduplicate
    adjustments["review_skills"] = sorted(set(adjustments["review_skills"]))

    return adjustments


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Bias & Fairness Auditor — Pipeline bias detection"
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="JSON file with tracer export and optional profiles",
    )
    parser.add_argument(
        "--profiles",
        help="JSON file with student demographic profiles",
    )
    parser.add_argument(
        "--modules",
        help="JSON file with curriculum modules (for representation audit)",
    )
    parser.add_argument(
        "--paths",
        help="JSON file with recommended learning paths",
    )
    parser.add_argument(
        "--full", action="store_true",
        help="Run full audit (all three dimensions)",
    )
    parser.add_argument(
        "--demographic", action="store_true",
        help="Run demographic fairness audit only",
    )
    parser.add_argument(
        "--prerequisite", action="store_true",
        help="Run prerequisite bias audit only",
    )
    parser.add_argument(
        "--representation", action="store_true",
        help="Run content representation audit only",
    )
    parser.add_argument(
        "--adjustments", action="store_true",
        help="Output bias-free path adjustments for curriculum_engine",
    )
    args = parser.parse_args()

    try:
        tracer_data: dict[str, Any] = {}
        if args.input_file:
            with open(args.input_file) as f:
                tracer_data = json.load(f)
        elif not sys.stdin.isatty():
            tracer_data = json.load(sys.stdin)

        profiles = []
        if args.profiles:
            with open(args.profiles) as f:
                profiles = json.load(f)
                if isinstance(profiles, dict):
                    profiles = [profiles]  # wrap single profile

        curriculum_modules = None
        if args.modules:
            with open(args.modules) as f:
                curriculum_modules = json.load(f)
                if isinstance(curriculum_modules, dict):
                    curriculum_modules = [curriculum_modules]

        recommended_paths = None
        if args.paths:
            with open(args.paths) as f:
                recommended_paths = json.load(f)
                if isinstance(recommended_paths, dict):
                    recommended_paths = [recommended_paths]

        auditor = BiasAuditor(tracer_data=tracer_data, student_profiles=profiles)

        if args.full or (not args.demographic and not args.prerequisite and not args.representation):
            result = auditor.full_audit(
                recommended_paths=recommended_paths,
                curriculum_modules=curriculum_modules,
            )
            if args.adjustments:
                adjustments = generate_bias_free_path_adjustments(result)
                print(json.dumps(adjustments, indent=2, ensure_ascii=False))
            else:
                print(json.dumps(_audit_result_to_dict(result), indent=2, ensure_ascii=False))

        elif args.demographic:
            findings = auditor.audit_demographic_fairness()
            print(json.dumps([_finding_to_dict(f) for f in findings], indent=2, ensure_ascii=False))

        elif args.prerequisite:
            findings = auditor.audit_prerequisite_bias(
                recommended_paths=recommended_paths
            )
            print(json.dumps([_finding_to_dict(f) for f in findings], indent=2, ensure_ascii=False))

        elif args.representation:
            findings = auditor.audit_content_representation(curriculum_modules)
            print(json.dumps([_finding_to_dict(f) for f in findings], indent=2, ensure_ascii=False))

    except (json.JSONDecodeError, FileNotFoundError, ValueError) as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)


def _finding_to_dict(f: BiasFinding) -> dict[str, Any]:
    return {
        "dimension": f.dimension,
        "severity": f.severity,
        "attribute": f.attribute,
        "description": f.description,
        "impact_ratio": f.impact_ratio,
        "affected_groups": f.affected_groups,
        "affected_skills": f.affected_skills,
        "recommendation": f.recommendation,
    }


def _audit_result_to_dict(r: AuditResult) -> dict[str, Any]:
    return {
        "audited_at": r.audited_at,
        "student_count": r.student_count,
        "overall_rating": r.overall_rating,
        "total_flags": r.total_flags,
        "fairness_scores": r.fairness_scores,
        "demographic_breakdown": r.demographic_breakdown,
        "findings": [_finding_to_dict(f) for f in r.findings],
    }


if __name__ == "__main__":
    main()
