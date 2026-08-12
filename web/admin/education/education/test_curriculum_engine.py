#!/usr/bin/env python3
"""
Tests for curriculum_engine.py — Education SEG-6
==================================================
Acceptance criteria (from education_roadmap.yaml):
  - Gaps sin prerrequisitos completados → no aparecen en el path
  - Orden topológico respetado — ningún módulo antes que sus prerequisitos
  - Camino crítico calculado correctamente
  - Módulo con target_level ya alcanzado → omitido automáticamente
  - YAML output válido, legible y parseable
  - Estimación de horas total no excede time_constraint_days o emite warning
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest
import yaml

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.education.curriculum_engine import CurriculumEngine, MODULES


FIXTURES = Path(__file__).resolve().parent / "fixtures"


# ═══════════════════════════════════════════════════════════════════════════
# Acceptance: Prerequisite Respect
# ═══════════════════════════════════════════════════════════════════════════


class TestPrerequisiteRespect:
    """AC: Orden topológico respetado — ningún módulo antes que sus prerequisitos."""

    def _make_gaps(self, skills_missing: list[str]) -> dict:
        gaps_list = []
        for skill in skills_missing:
            gaps_list.append({
                "skill": skill,
                "estimated_level": "beginner",
                "target_level": "intermediate",
                "gap_severity": 2,
                "priority": 5,
                "dunning_kruger_flag": False,
                "rationale": f"Gap en {skill}",
            })
        return {
            "student_id": "test",
            "career_goal": "Data Scientist",
            "gaps": gaps_list,
        }

    def _make_profile(self, hours=20, constraint=0, completed=None):
        return {
            "profile": {
                "student_id": "test",
                "career_goal": "Data Scientist",
                "available_hours_per_week": hours,
                "learning_style": "mixed",
                "preferred_language": "es",
                "time_constraint_days": constraint,
                "prior_modules_completed": completed or [],
            }
        }

    def test_topological_order_python_path(self):
        """MOD-PY-01 debe aparecer antes que MOD-PY-02, etc."""
        gaps = self._make_gaps(["python", "statistics", "machine_learning"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        modules = result["modules"]
        module_ids = [m["id"] for m in modules]

        # PY-01 before PY-02 before DS-01
        if "MOD-PY-01" in module_ids and "MOD-PY-02" in module_ids:
            assert module_ids.index("MOD-PY-01") < module_ids.index("MOD-PY-02")
        if "MOD-PY-02" in module_ids and "MOD-DS-01" in module_ids:
            assert module_ids.index("MOD-PY-02") < module_ids.index("MOD-DS-01")

    def test_prerequisite_chain_ml(self):
        """ML module should come after its hard prerequisites."""
        gaps = self._make_gaps(["python", "statistics", "machine_learning"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        modules = result["modules"]
        module_ids = [m["id"] for m in modules]

        if "MOD-ML-01" in module_ids:
            ml_idx = module_ids.index("MOD-ML-01")
            if "MOD-DS-01" in module_ids:
                assert module_ids.index("MOD-DS-01") < ml_idx, \
                    f"DS-01 should be before ML-01. Order: {module_ids}"
            if "MOD-STAT-01" in module_ids:
                assert module_ids.index("MOD-STAT-01") < ml_idx, \
                    f"STAT-01 should be before ML-01. Order: {module_ids}"

    def test_no_module_before_its_prerequisites(self):
        """Verificar cada módulo: sus prerequisitos aparecen antes."""
        gaps = self._make_gaps([
            "python", "sql", "statistics", "machine_learning",
            "data_visualization", "git"
        ])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        modules = result["modules"]
        module_ids = [m["id"] for m in modules]
        id_to_idx = {mid: i for i, mid in enumerate(module_ids)}

        for mod in modules:
            for prereq in mod.get("prerequisites", []):
                if prereq in id_to_idx:
                    assert id_to_idx[prereq] < id_to_idx[mod["id"]], \
                        f"Prereq {prereq} should be before {mod['id']}"


# ═══════════════════════════════════════════════════════════════════════════
# Acceptance: Completed Modules Filtered
# ═══════════════════════════════════════════════════════════════════════════


class TestFilterCompleted:
    """AC: Módulo con target_level ya alcanzado → omitido automáticamente."""

    def test_completed_modules_omitted(self):
        """Si el usuario ya completó MOD-PY-01, no debe aparecer."""
        gaps = self._make_gaps(["python", "sql"])
        profile = {
            "profile": {
                "student_id": "test",
                "career_goal": "Data Scientist",
                "available_hours_per_week": 20,
                "learning_style": "mixed",
                "preferred_language": "es",
                "time_constraint_days": 0,
                "prior_modules_completed": ["MOD-PY-01"],
            }
        }
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        module_ids = [m["id"] for m in result["modules"]]
        assert "MOD-PY-01" not in module_ids, \
            f"MOD-PY-01 should be omitted. Found: {module_ids}"

    def test_skill_already_at_target_omitted(self):
        """Si la skill ya está en target_level, módulos para ese nivel se omiten."""
        gaps_list = [{
            "skill": "python",
            "estimated_level": "advanced",  # Ya tiene nivel avanzado
            "target_level": "advanced",
            "gap_severity": 0,
            "priority": 10,
            "dunning_kruger_flag": False,
            "rationale": "No gap — nivel ya alcanzado.",
        }]
        gaps = {
            "student_id": "test",
            "career_goal": "Data Scientist",
            "gaps": gaps_list,
        }
        profile = {
            "profile": {
                "student_id": "test",
                "career_goal": "Data Scientist",
                "available_hours_per_week": 20,
                "learning_style": "mixed",
                "preferred_language": "es",
                "time_constraint_days": 0,
                "prior_modules_completed": [],
            }
        }
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        # No python modules should be selected since python is already at target
        python_modules = [m for m in result["modules"] if m["skill"] == "python"]
        assert len(python_modules) == 0, \
            f"No Python modules expected. Found: {[m['id'] for m in python_modules]}"


# ═══════════════════════════════════════════════════════════════════════════
# Acceptance: Critical Path
# ═══════════════════════════════════════════════════════════════════════════


class TestCriticalPath:
    """AC: Camino crítico calculado correctamente."""

    def test_critical_path_exists(self):
        gaps = self._make_gaps(["python", "sql"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        assert "critical_path_hours" in result
        assert result["critical_path_hours"] > 0
        # Critical path must be <= total hours
        assert result["critical_path_hours"] <= result["total_estimated_hours"]

    def test_critical_path_not_exceed_total(self):
        """Critical path hours never exceeds total hours."""
        gaps = self._make_gaps([
            "python", "sql", "statistics", "machine_learning"
        ])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        assert result["critical_path_hours"] <= result["total_estimated_hours"]


# ═══════════════════════════════════════════════════════════════════════════
# Acceptance: Output Schema
# ═══════════════════════════════════════════════════════════════════════════


class TestOutputSchema:
    """AC: YAML output válido, legible y parseable."""

    def test_output_fields(self):
        gaps = self._make_gaps(["python"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        assert "student_id" in result
        assert "career_goal" in result
        assert "generated_at" in result
        assert "total_estimated_hours" in result
        assert "total_modules" in result
        assert "critical_path_hours" in result
        assert "modules" in result
        assert isinstance(result["modules"], list)

    def test_module_fields(self):
        gaps = self._make_gaps(["python"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        for mod in result["modules"]:
            assert "id" in mod
            assert "name" in mod
            assert "skill" in mod
            assert "target_level" in mod
            assert "hours" in mod
            assert "prerequisites" in mod
            assert "order" in mod
            assert "topics" in mod
            assert "resources" in mod
            assert "mastery_criteria" in mod

    def test_yaml_parseable(self):
        gaps = self._make_gaps(["python"])
        profile = self._make_profile()
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        yaml_str = yaml.dump(result, allow_unicode=True, sort_keys=False)
        parsed = yaml.safe_load(yaml_str)
        assert parsed is not None
        assert parsed["student_id"] == result["student_id"]


# ═══════════════════════════════════════════════════════════════════════════
# Acceptance: Time Constraint Warning
# ═══════════════════════════════════════════════════════════════════════════


class TestTimeConstraint:
    """AC: Estimación no excede time_constraint_days o emite warning."""

    def test_warning_on_exceeding_constraint(self):
        """Con 2h/semana, Data Science completo → excede 30 días → warning."""
        gaps = self._make_gaps([
            "python", "sql", "statistics", "machine_learning", "data_visualization"
        ])
        profile = {
            "profile": {
                "student_id": "test",
                "career_goal": "Data Scientist",
                "available_hours_per_week": 2,  # Muy poco tiempo
                "learning_style": "mixed",
                "preferred_language": "es",
                "time_constraint_days": 30,  # Muy poco tiempo
                "prior_modules_completed": [],
            }
        }
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        assert len(result["warnings"]) > 0
        assert any("excede" in w.lower() or "time constraint" in w.lower()
                   for w in result["warnings"])

    def test_no_warning_without_constraint(self):
        """Sin time constraint definido (0) → no warning."""
        gaps = self._make_gaps(["python", "statistics"])
        profile = self._make_profile(hours=20, constraint=0)
        engine = CurriculumEngine(gaps, profile)
        result = engine.run()

        # Should have no warnings about time
        time_warnings = [w for w in result["warnings"] if "time" in w.lower() or "días" in w.lower()]
        assert len(time_warnings) == 0


# ═══════════════════════════════════════════════════════════════════════════
# Test Helpers (shared with TestPrerequisiteRespect)
# ═══════════════════════════════════════════════════════════════════════════

# Mixin para compartir helpers entre clases
class TestHelpersMixin:
    def _make_gaps(self, skills_missing: list[str]) -> dict:
        gaps_list = []
        for skill in skills_missing:
            gaps_list.append({
                "skill": skill,
                "estimated_level": "beginner",
                "target_level": "intermediate",
                "gap_severity": 2,
                "priority": 5,
                "dunning_kruger_flag": False,
                "rationale": f"Gap en {skill}",
            })
        return {
            "student_id": "test",
            "career_goal": "Data Scientist",
            "gaps": gaps_list,
        }

    def _make_profile(self, hours=20, constraint=0, completed=None):
        return {
            "profile": {
                "student_id": "test",
                "career_goal": "Data Scientist",
                "available_hours_per_week": hours,
                "learning_style": "mixed",
                "preferred_language": "es",
                "time_constraint_days": constraint,
                "prior_modules_completed": completed or [],
            }
        }


# Inject helpers into existing test classes (pytest-safe)
TestFilterCompleted._make_gaps = TestHelpersMixin._make_gaps  # type: ignore
TestFilterCompleted._make_profile = TestHelpersMixin._make_profile  # type: ignore
TestCriticalPath._make_gaps = TestHelpersMixin._make_gaps  # type: ignore
TestCriticalPath._make_profile = TestHelpersMixin._make_profile  # type: ignore
TestOutputSchema._make_gaps = TestHelpersMixin._make_gaps  # type: ignore
TestOutputSchema._make_profile = TestHelpersMixin._make_profile  # type: ignore
TestTimeConstraint._make_gaps = TestHelpersMixin._make_gaps  # type: ignore
TestTimeConstraint._make_profile = TestHelpersMixin._make_profile  # type: ignore


# ═══════════════════════════════════════════════════════════════════════════
# Integration: End-to-End Pipeline
# ═══════════════════════════════════════════════════════════════════════════


class TestEndToEnd:
    """Pipeline completo: gap_detector → curriculum_engine."""

    def test_full_pipeline_data_scientist(self):
        """Load stu-0042 fixture, detect gaps, generate curriculum."""
        from tools.education.gap_detector import GapDetector

        # Step 1: Gap Detection
        fixture = FIXTURES / "stu0042_data_scientist.yaml"
        with open(fixture) as f:
            data = yaml.safe_load(f)

        detector = GapDetector(data)
        gaps_result = detector.run()

        assert len(gaps_result["gaps"]) > 0, "Should detect gaps for Data Scientist"

        # Step 2: Curriculum Generation
        profile_fixture = FIXTURES / "stu0042_profile.yaml"
        with open(profile_fixture) as f:
            profile = yaml.safe_load(f)

        engine = CurriculumEngine(gaps_result, profile)
        curriculum = engine.run()

        assert curriculum["student_id"] == "stu-0042"
        assert curriculum["career_goal"] == "Data Scientist"
        assert curriculum["total_modules"] > 0
        assert curriculum["total_estimated_hours"] > 0

        # Verify modules are ordered by prerequisites
        modules = curriculum["modules"]
        module_ids = [m["id"] for m in modules]
        id_to_idx = {mid: i for i, mid in enumerate(module_ids)}

        for mod in modules:
            for prereq in mod.get("prerequisites", []):
                if prereq in id_to_idx:
                    assert id_to_idx[prereq] < id_to_idx[mod["id"]], \
                        f"Prereq {prereq} should come before {mod['id']}"

    def test_full_pipeline_backend_developer(self):
        """Load stu-0099 fixture, detect gaps, generate curriculum."""
        from tools.education.gap_detector import GapDetector

        fixture = FIXTURES / "stu0099_backend_dev.yaml"
        with open(fixture) as f:
            data = yaml.safe_load(f)

        detector = GapDetector(data)
        gaps_result = detector.run()

        # Step 2
        profile_fixture = FIXTURES / "stu0099_profile.yaml"
        with open(profile_fixture) as f:
            profile = yaml.safe_load(f)

        engine = CurriculumEngine(gaps_result, profile)
        curriculum = engine.run()

        assert curriculum["total_modules"] > 0
        # Go should be a module (novice → advanced)
        go_modules = [m for m in curriculum["modules"] if m["skill"] == "go"]
        assert len(go_modules) > 0

    def test_modules_are_resolvable(self):
        """Cada módulo en MODULES referencia prerrequisitos que existen."""
        module_ids = {m["id"] for m in MODULES}
        for mod in MODULES:
            for prereq in mod.get("prerequisites", []):
                assert prereq in module_ids, \
                    f"Module {mod['id']} references nonexistent prereq {prereq}"


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
