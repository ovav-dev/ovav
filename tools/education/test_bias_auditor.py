#!/usr/bin/env python3
"""
Tests for bias_auditor.py — Education SEG-6 Phase 3
=====================================================
Acceptance criteria:
  - Demographic fairness audit detects disparate impact
  - Prerequisite bias audit flags skill category concentration
  - Content representation audit identifies content deserts
  - Full audit runs all three dimensions
  - Bias-free path adjustments generated correctly
  - Edge cases: empty profiles, single student, no findings
  - Integration with knowledge_tracer export format
  - Severity classification correct
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.education.bias_auditor import (
    BiasAuditor,
    BiasFinding,
    DemographicProfile,
    AuditResult,
    generate_bias_free_path_adjustments,
    DISPARATE_IMPACT_THRESHOLD,
    SEVERITY_CRITICAL,
    SEVERITY_HIGH,
    SEVERITY_MEDIUM,
    SEVERITY_LOW,
    SEVERITY_INFO,
    PROTECTED_ATTRIBUTES,
    CAREER_GOAL_SKILLS,
    SKILL_CATEGORIES,
)


# ═══════════════════════════════════════════════════════════════════════════
# Helpers
# ═══════════════════════════════════════════════════════════════════════════

def make_tracer_export(
    student_ids: list[str],
    masteries_per_student: list[dict[str, float]],
) -> dict:
    """Build a multi-student tracer export."""
    students = [
        {"student_id": sid, "masteries": m}
        for sid, m in zip(student_ids, masteries_per_student)
    ]
    return {"students": students}


def make_profiles(
    ids_and_attrs: list[tuple[str, str, str, str, str, str]],
) -> list[dict]:
    """
    Build student profiles.
    Each tuple: (student_id, gender, age_group, education, language, region)
    """
    profiles = []
    for sid, gender, age, edu, lang, region in ids_and_attrs:
        profiles.append({
            "student_id": sid,
            "gender": gender,
            "age_group": age,
            "education_level": edu,
            "primary_language": lang,
            "geographic_region": region,
        })
    return profiles


# ═══════════════════════════════════════════════════════════════════════════
# 1. Initialization & Edge Cases
# ═══════════════════════════════════════════════════════════════════════════

class TestInitialization:
    """Auditor initialization and edge cases."""

    def test_create_empty(self):
        auditor = BiasAuditor()
        assert auditor.tracer_data == {}
        assert len(auditor.profiles) == 0

    def test_create_with_tracer_data(self):
        data = make_tracer_export(
            ["s1", "s2"],
            [{"python": 0.7, "sql": 0.5}, {"python": 0.3, "sql": 0.8}],
        )
        auditor = BiasAuditor(tracer_data=data)
        assert "students" in auditor.tracer_data

    def test_create_with_profiles(self):
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "male", "25-34", "bachelors", "en", "NA"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        assert len(auditor.profiles) == 2
        assert auditor.profiles["s1"].gender == "female"
        assert auditor.profiles["s2"].geographic_region == "NA"

    def test_empty_mastery_data_returns_no_findings(self):
        """No findings when no mastery data is available."""
        auditor = BiasAuditor()
        findings = auditor.audit_demographic_fairness(mastery_data={})
        assert len(findings) == 0


# ═══════════════════════════════════════════════════════════════════════════
# 2. Demographic Fairness Audit
# ═══════════════════════════════════════════════════════════════════════════

class TestDemographicFairness:
    """Detect disparate impact across demographic groups."""

    def test_no_bias_when_equal_mastery(self):
        """Equal mastery across groups → no findings."""
        mastery = {
            "s1": {"python": 0.80, "sql": 0.80},
            "s2": {"python": 0.80, "sql": 0.80},
            "s3": {"python": 0.82, "sql": 0.79},
            "s4": {"python": 0.81, "sql": 0.80},
            "s5": {"python": 0.79, "sql": 0.81},
            "s6": {"python": 0.80, "sql": 0.79},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "male", "25-34", "bachelors", "en", "NA"),
            ("s3", "female", "25-34", "masters", "es", "LATAM"),
            ("s4", "male", "25-34", "masters", "en", "NA"),
            ("s5", "female", "25-34", "bachelors", "en", "NA"),
            ("s6", "male", "25-34", "bachelors", "es", "LATAM"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_demographic_fairness(mastery_data=mastery)
        assert len(findings) == 0

    def test_detects_disparate_impact(self):
        """One group significantly below reference → flagged."""
        mastery = {
            "s1": {"python": 0.85, "sql": 0.88},  # NA group (high)
            "s2": {"python": 0.90, "sql": 0.92},
            "s3": {"python": 0.88, "sql": 0.87},
            "s4": {"python": 0.30, "sql": 0.25},  # LATAM group (low)
            "s5": {"python": 0.35, "sql": 0.28},
            "s6": {"python": 0.32, "sql": 0.26},
        }
        profiles = make_profiles([
            ("s1", "male", "25-34", "bachelors", "en", "NA"),
            ("s2", "male", "25-34", "masters", "en", "NA"),
            ("s3", "male", "25-34", "bachelors", "en", "NA"),
            ("s4", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s5", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s6", "female", "25-34", "bachelors", "es", "LATAM"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_demographic_fairness(mastery_data=mastery)

        # Should have at least some findings from gender or region
        assert len(findings) > 0

        # Check that low-performing group is flagged
        low_flags = [f for f in findings if f.impact_ratio is not None and f.impact_ratio < 0.80]
        assert len(low_flags) > 0

    def test_finding_has_required_fields(self):
        mastery = {
            "s1": {"python": 0.85, "sql": 0.88},
            "s2": {"python": 0.90, "sql": 0.92},
            "s3": {"python": 0.87, "sql": 0.86},
            "s4": {"python": 0.30, "sql": 0.25},
            "s5": {"python": 0.35, "sql": 0.28},
            "s6": {"python": 0.32, "sql": 0.26},
        }
        profiles = make_profiles([
            ("s1", "male", "25-34", "bachelors", "en", "NA"),
            ("s2", "male", "25-34", "masters", "en", "NA"),
            ("s3", "male", "25-34", "bachelors", "en", "NA"),
            ("s4", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s5", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s6", "female", "25-34", "bachelors", "es", "LATAM"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_demographic_fairness(mastery_data=mastery)

        for f in findings:
            assert f.dimension == "demographic"
            assert f.severity in (SEVERITY_CRITICAL, SEVERITY_HIGH, SEVERITY_MEDIUM, SEVERITY_LOW, SEVERITY_INFO)
            assert f.attribute in PROTECTED_ATTRIBUTES
            assert len(f.description) > 0
            assert isinstance(f.recommendation, str)

    def test_small_groups_not_flagged(self):
        """Groups below MIN_GROUP_SIZE should not trigger false positives."""
        mastery = {
            "s1": {"python": 0.85, "sql": 0.88},
            "s2": {"python": 0.90, "sql": 0.92},
            "s3": {"python": 0.15, "sql": 0.10},  # single outlier
            "s4": {"python": 0.88, "sql": 0.85},
        }
        profiles = make_profiles([
            ("s1", "male", "25-34", "bachelors", "en", "NA"),
            ("s2", "male", "25-34", "masters", "en", "NA"),
            ("s3", "female", "45+", "phd", "pt", "LATAM"),  # unique group
            ("s4", "male", "25-34", "bachelors", "en", "NA"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_demographic_fairness(mastery_data=mastery)

        # s3 is the only "female", "45+", "phd", "pt" — each group size 1 → ratio = 1.0
        # Should not have critical findings on these attributes
        critical = [f for f in findings if f.severity == SEVERITY_CRITICAL]
        # With group size 1, ratio is set to 1.0, so no critical flags
        assert len(critical) == 0


# ═══════════════════════════════════════════════════════════════════════════
# 3. Prerequisite Bias Audit
# ═══════════════════════════════════════════════════════════════════════════

class TestPrerequisiteBias:
    """Detect skill prerequisite and category concentration bias."""

    def test_no_bias_with_balanced_skills(self):
        """Balanced mastery across categories and groups → no high-severity findings."""
        mastery = {
            "s1": {"python": 0.80, "statistics": 0.75, "sql": 0.70},
            "s2": {"python": 0.82, "statistics": 0.78, "sql": 0.72},
            "s3": {"python": 0.79, "statistics": 0.76, "sql": 0.71},
            "s4": {"python": 0.81, "statistics": 0.77, "sql": 0.69},
            "s5": {"python": 0.80, "statistics": 0.74, "sql": 0.73},
            "s6": {"python": 0.78, "statistics": 0.79, "sql": 0.70},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "male", "25-34", "bachelors", "en", "NA"),
            ("s3", "female", "25-34", "masters", "es", "LATAM"),
            ("s4", "male", "25-34", "masters", "en", "NA"),
            ("s5", "female", "25-34", "bachelors", "en", "LATAM"),
            ("s6", "male", "25-34", "bachelors", "es", "NA"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_prerequisite_bias(mastery_data=mastery)
        # With balanced data, no high-severity findings expected
        high_critical = [f for f in findings if f.severity in (SEVERITY_HIGH, SEVERITY_CRITICAL)]
        assert len(high_critical) == 0

    def test_detects_category_concentration(self):
        """One group significantly behind in a skill category → flagged."""
        mastery = {
            "s1": {"python": 0.10, "javascript": 0.12, "sql": 0.15},  # LATAM, low
            "s2": {"python": 0.08, "javascript": 0.10, "sql": 0.12},
            "s3": {"python": 0.12, "javascript": 0.11, "sql": 0.14},
            "s4": {"python": 0.85, "javascript": 0.80, "sql": 0.88},  # NA, high
            "s5": {"python": 0.90, "javascript": 0.88, "sql": 0.85},
            "s6": {"python": 0.88, "javascript": 0.82, "sql": 0.90},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s3", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s4", "male", "25-34", "bachelors", "en", "NA"),
            ("s5", "male", "25-34", "masters", "en", "NA"),
            ("s6", "male", "25-34", "bachelors", "en", "NA"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_prerequisite_bias(mastery_data=mastery)
        # Should detect systematic difference
        assert len(findings) > 0

    def test_path_length_disparity_detected(self):
        """Different group path lengths → flagged."""
        mastery = {
            "s1": {"python": 0.90, "sql": 0.85},
            "s2": {"python": 0.88, "sql": 0.87},
            "s3": {"python": 0.92, "sql": 0.89},
            "s4": {"python": 0.25, "sql": 0.20},
            "s5": {"python": 0.30, "sql": 0.22},
            "s6": {"python": 0.28, "sql": 0.24},
        }
        profiles = make_profiles([
            ("s1", "male", "25-34", "bachelors", "en", "NA"),
            ("s2", "male", "25-34", "masters", "en", "NA"),
            ("s3", "male", "25-34", "bachelors", "en", "NA"),
            ("s4", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s5", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s6", "female", "25-34", "bachelors", "es", "LATAM"),
        ])
        # NA group: short path (high mastery), LATAM group: long path (low mastery)
        recommended_paths = [
            {"student_id": "s1", "modules": [{"id": "m1"}, {"id": "m2"}]},
            {"student_id": "s2", "modules": [{"id": "m1"}, {"id": "m2"}, {"id": "m3"}]},
            {"student_id": "s3", "modules": [{"id": "m1"}, {"id": "m2"}]},
            {"student_id": "s4", "modules": [{"id": "m1"}, {"id": "m2"}, {"id": "m3"}, {"id": "m4"}, {"id": "m5"}]},
            {"student_id": "s5", "modules": [{"id": "m1"}, {"id": "m2"}, {"id": "m3"}, {"id": "m4"}, {"id": "m5"}, {"id": "m6"}]},
            {"student_id": "s6", "modules": [{"id": "m1"}, {"id": "m2"}, {"id": "m3"}, {"id": "m4"}, {"id": "m5"}]},
        ]
        auditor = BiasAuditor(student_profiles=profiles)
        findings = auditor.audit_prerequisite_bias(
            mastery_data=mastery, recommended_paths=recommended_paths
        )
        # Should detect path length disparity
        path_findings = [f for f in findings if "path length" in f.description.lower()]
        assert len(path_findings) > 0


# ═══════════════════════════════════════════════════════════════════════════
# 4. Content Representation Audit
# ═══════════════════════════════════════════════════════════════════════════

class TestContentRepresentation:
    """Detect content deserts and under-represented career goals."""

    def test_all_goals_covered_no_findings(self):
        """Full coverage across all career goals → only low/info findings at most."""
        modules = []
        for goal, skills in CAREER_GOAL_SKILLS.items():
            for skill in skills:
                modules.append({
                    "id": f"MOD-{skill}-01",
                    "skill": skill,
                    "career_goal": goal,
                    "target_level": "intermediate",
                    "hours": 40,
                })

        auditor = BiasAuditor()
        findings = auditor.audit_content_representation(curriculum_modules=modules)
        # With full coverage, no critical or high findings
        critical_high = [f for f in findings if f.severity in (SEVERITY_CRITICAL, SEVERITY_HIGH)]
        assert len(critical_high) == 0

    def test_detects_content_desert(self):
        """A career goal with 0% coverage → critical finding."""
        # Only cover Data Scientist skills, leave ML Engineer completely uncovered
        modules = []
        for skill in CAREER_GOAL_SKILLS["Data Scientist"]:
            modules.append({
                "id": f"MOD-{skill}-01",
                "skill": skill,
                "career_goal": "Data Scientist",
                "target_level": "intermediate",
                "hours": 40,
            })

        auditor = BiasAuditor()
        findings = auditor.audit_content_representation(curriculum_modules=modules)
        # Should flag ML Engineer (0% coverage)
        critical = [f for f in findings if f.severity == SEVERITY_CRITICAL]
        assert len(critical) >= 1

        ml_flags = [f for f in critical if "ML Engineer" in f.description]
        assert len(ml_flags) >= 1

    def test_partial_coverage_high_severity(self):
        """Career goal with <50% coverage → critical or high."""
        modules = []
        # Only give 2/6 skills for Data Scientist (<50%)
        ds_skills = CAREER_GOAL_SKILLS["Data Scientist"]
        for skill in ds_skills[:2]:  # Only 2 out of 6
            modules.append({
                "id": f"MOD-{skill}-01",
                "skill": skill,
                "career_goal": "Data Scientist",
                "target_level": "intermediate",
                "hours": 40,
            })

        auditor = BiasAuditor()
        findings = auditor.audit_content_representation(curriculum_modules=modules)
        ds_findings = [f for f in findings if "Data Scientist" in f.description]
        assert len(ds_findings) >= 1
        assert ds_findings[0].severity in (SEVERITY_CRITICAL, SEVERITY_HIGH)

    def test_representation_finding_fields(self):
        modules = [
            {"id": "m1", "skill": "python", "career_goal": "Data Scientist",
             "target_level": "intermediate", "hours": 40},
        ]
        auditor = BiasAuditor()
        findings = auditor.audit_content_representation(curriculum_modules=modules)

        for f in findings:
            assert f.dimension == "representation"
            assert f.attribute in ("career_goal", "skill_category")
            assert len(f.description) > 0
            assert len(f.recommendation) > 0


# ═══════════════════════════════════════════════════════════════════════════
# 5. Full Audit Integration
# ═══════════════════════════════════════════════════════════════════════════

class TestFullAudit:
    """Full audit pipeline — all three dimensions."""

    def test_full_audit_runs_all_dimensions(self):
        mastery = {
            "s1": {"python": 0.85, "sql": 0.88, "statistics": 0.80},
            "s2": {"python": 0.90, "sql": 0.92, "statistics": 0.85},
            "s3": {"python": 0.87, "sql": 0.86, "statistics": 0.83},
            "s4": {"python": 0.30, "sql": 0.25, "statistics": 0.20},
            "s5": {"python": 0.35, "sql": 0.28, "statistics": 0.22},
            "s6": {"python": 0.32, "sql": 0.26, "statistics": 0.24},
        }
        profiles = make_profiles([
            ("s1", "male", "25-34", "bachelors", "en", "NA"),
            ("s2", "male", "25-34", "masters", "en", "NA"),
            ("s3", "male", "25-34", "bachelors", "en", "NA"),
            ("s4", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s5", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s6", "female", "25-34", "bachelors", "es", "LATAM"),
        ])
        modules = [
            {"id": "m1", "skill": "python", "career_goal": "Data Scientist",
             "target_level": "intermediate", "hours": 40},
        ]

        auditor = BiasAuditor(
            tracer_data=make_tracer_export(
                ["s1", "s2", "s3", "s4", "s5", "s6"],
                [
                    {"python": 0.85, "sql": 0.88, "statistics": 0.80},
                    {"python": 0.90, "sql": 0.92, "statistics": 0.85},
                    {"python": 0.87, "sql": 0.86, "statistics": 0.83},
                    {"python": 0.30, "sql": 0.25, "statistics": 0.20},
                    {"python": 0.35, "sql": 0.28, "statistics": 0.22},
                    {"python": 0.32, "sql": 0.26, "statistics": 0.24},
                ],
            ),
            student_profiles=profiles,
        )
        result = auditor.full_audit(
            mastery_data=mastery,
            curriculum_modules=modules,
        )

        assert isinstance(result, AuditResult)
        assert result.student_count > 0
        assert result.overall_rating in ("fair", "needs_review", "biased")
        assert result.total_flags == len(result.findings)
        assert len(result.fairness_scores) == 3
        assert "demographic" in result.fairness_scores
        assert "prerequisite" in result.fairness_scores
        assert "representation" in result.fairness_scores
        assert len(result.demographic_breakdown) > 0

    def test_fair_system_gets_fair_rating(self):
        """Balanced data → 'fair' rating."""
        mastery = {
            "s1": {"python": 0.85, "sql": 0.85},
            "s2": {"python": 0.87, "sql": 0.83},
            "s3": {"python": 0.86, "sql": 0.84},
            "s4": {"python": 0.84, "sql": 0.86},
            "s5": {"python": 0.88, "sql": 0.82},
            "s6": {"python": 0.83, "sql": 0.87},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "male", "25-34", "bachelors", "en", "NA"),
            ("s3", "female", "25-34", "masters", "es", "LATAM"),
            ("s4", "male", "25-34", "masters", "en", "NA"),
            ("s5", "female", "25-34", "bachelors", "en", "LATAM"),
            ("s6", "male", "25-34", "bachelors", "es", "NA"),
        ])
        # Generate full curriculum coverage
        modules = []
        for goal, skills in CAREER_GOAL_SKILLS.items():
            for skill in skills:
                modules.append({
                    "id": f"MOD-{skill}-{goal[:2]}",
                    "skill": skill,
                    "career_goal": goal,
                    "target_level": "intermediate",
                    "hours": 40,
                })
        # Add professional skills modules to balance category representation
        for prof_skill in ["technical_writing", "problem_solving", "project_management"]:
            modules.append({
                "id": f"MOD-{prof_skill}-GEN",
                "skill": prof_skill,
                "career_goal": "general",
                "target_level": "intermediate",
                "hours": 30,
            })

        auditor = BiasAuditor(student_profiles=profiles)
        result = auditor.full_audit(mastery_data=mastery, curriculum_modules=modules)
        assert result.overall_rating == "fair"

    def test_biased_system_gets_biased_rating(self):
        """Severely biased data → 'biased' rating."""
        mastery = {
            "s1": {"python": 0.10, "sql": 0.12},
            "s2": {"python": 0.08, "sql": 0.10},
            "s3": {"python": 0.12, "sql": 0.11},
            "s4": {"python": 0.90, "sql": 0.88},
            "s5": {"python": 0.92, "sql": 0.90},
            "s6": {"python": 0.89, "sql": 0.91},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s3", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s4", "male", "25-34", "bachelors", "en", "NA"),
            ("s5", "male", "25-34", "masters", "en", "NA"),
            ("s6", "male", "25-34", "bachelors", "en", "NA"),
        ])
        modules = [
            {"id": "m1", "skill": "python", "career_goal": "Data Scientist",
             "target_level": "intermediate", "hours": 40},
        ]

        auditor = BiasAuditor(student_profiles=profiles)
        result = auditor.full_audit(mastery_data=mastery, curriculum_modules=modules)
        # Should have critical findings
        critical = [f for f in result.findings if f.severity == SEVERITY_CRITICAL]
        assert len(critical) >= 1
        assert result.overall_rating == "biased"


# ═══════════════════════════════════════════════════════════════════════════
# 6. Integration: Bias-Free Path Adjustments
# ═══════════════════════════════════════════════════════════════════════════

class TestPathAdjustments:
    """Generate bias-free path adjustments for curriculum_engine."""

    def test_generates_adjustments_from_findings(self):
        result = AuditResult(
            audited_at="2026-06-16T00:00:00+0000",
            student_count=4,
            findings=[
                BiasFinding(
                    dimension="demographic",
                    severity=SEVERITY_HIGH,
                    attribute="gender",
                    description="Gender bias detected",
                    affected_skills=["python", "sql"],
                    recommendation="Review python assessment",
                ),
                BiasFinding(
                    dimension="representation",
                    severity=SEVERITY_CRITICAL,
                    attribute="career_goal",
                    description="ML Engineer missing",
                    affected_skills=["machine_learning", "deep_learning"],
                    recommendation="Add ML modules",
                ),
            ],
        )
        adjustments = generate_bias_free_path_adjustments(result)
        assert "review_skills" in adjustments
        assert "recommendations" in adjustments
        # Both findings are critical/high → both should produce review_skills
        assert "python" in adjustments["review_skills"]
        assert "machine_learning" in adjustments["review_skills"]

    def test_ignores_low_severity_for_adjustments(self):
        result = AuditResult(
            audited_at="2026-06-16T00:00:00+0000",
            student_count=4,
            findings=[
                BiasFinding(
                    dimension="demographic",
                    severity=SEVERITY_LOW,
                    attribute="gender",
                    description="Minor gender difference",
                    affected_skills=["git"],
                    recommendation="Monitor git assessment",
                ),
                BiasFinding(
                    dimension="prerequisite",
                    severity=SEVERITY_INFO,
                    attribute="age_group",
                    description="Informational only",
                    affected_skills=["python"],
                    recommendation="No action needed",
                ),
            ],
        )
        adjustments = generate_bias_free_path_adjustments(result)
        # Low and info severity should NOT trigger review_skills
        assert len(adjustments["review_skills"]) == 0
        assert len(adjustments["recommendations"]) == 0

    def test_adjustments_deduplicates_skills(self):
        result = AuditResult(
            audited_at="2026-06-16T00:00:00+0000",
            student_count=4,
            findings=[
                BiasFinding(
                    dimension="demographic",
                    severity=SEVERITY_HIGH,
                    attribute="gender",
                    description="Gender bias in python",
                    affected_skills=["python", "sql"],
                    recommendation="Review python",
                ),
                BiasFinding(
                    dimension="prerequisite",
                    severity=SEVERITY_CRITICAL,
                    attribute="region",
                    description="Region bias in python",
                    affected_skills=["python", "statistics"],
                    recommendation="Review python again",
                ),
            ],
        )
        adjustments = generate_bias_free_path_adjustments(result)
        # "python" appears in both, should be deduplicated
        assert adjustments["review_skills"].count("python") == 1
        assert sorted(adjustments["review_skills"]) == ["python", "sql", "statistics"]


# ═══════════════════════════════════════════════════════════════════════════
# 7. Severity Classification
# ═══════════════════════════════════════════════════════════════════════════

class TestSeverityClassification:
    """Verify severity classification in audit results."""

    def test_finding_severity_values(self):
        """All findings have valid severity levels."""
        mastery = {
            "s1": {"python": 0.10, "sql": 0.10},
            "s2": {"python": 0.90, "sql": 0.90},
            "s3": {"python": 0.12, "sql": 0.12},
            "s4": {"python": 0.88, "sql": 0.88},
            "s5": {"python": 0.11, "sql": 0.11},
            "s6": {"python": 0.89, "sql": 0.89},
        }
        profiles = make_profiles([
            ("s1", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s2", "male", "25-34", "bachelors", "en", "NA"),
            ("s3", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s4", "male", "25-34", "bachelors", "en", "NA"),
            ("s5", "female", "25-34", "bachelors", "es", "LATAM"),
            ("s6", "male", "25-34", "bachelors", "en", "NA"),
        ])
        auditor = BiasAuditor(student_profiles=profiles)
        result = auditor.full_audit(mastery_data=mastery)

        valid_severities = {
            SEVERITY_CRITICAL, SEVERITY_HIGH, SEVERITY_MEDIUM,
            SEVERITY_LOW, SEVERITY_INFO,
        }
        for f in result.findings:
            assert f.severity in valid_severities, (
                f"Invalid severity '{f.severity}' in {f.dimension}: {f.description[:50]}"
            )

    def test_audit_result_serializable(self):
        """AuditResult can be converted to dict (for JSON output)."""
        from tools.education.bias_auditor import _audit_result_to_dict

        result = AuditResult(
            audited_at="2026-06-16T00:00:00+0000",
            student_count=2,
            findings=[
                BiasFinding(
                    dimension="demographic",
                    severity=SEVERITY_HIGH,
                    attribute="gender",
                    description="Test finding",
                    impact_ratio=0.65,
                    affected_groups=["female", "male"],
                    affected_skills=["python"],
                    recommendation="Review",
                ),
            ],
            fairness_scores={"demographic": 75.0, "prerequisite": 90.0, "representation": 100.0},
            demographic_breakdown={"gender": {"female": 1, "male": 1}},
            overall_rating="needs_review",
            total_flags=1,
        )
        d = _audit_result_to_dict(result)
        assert d["overall_rating"] == "needs_review"
        assert d["total_flags"] == 1
        assert len(d["findings"]) == 1
        assert d["findings"][0]["impact_ratio"] == 0.65
