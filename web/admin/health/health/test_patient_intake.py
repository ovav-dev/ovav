#!/usr/bin/env python3
"""
Tests for patient_intake.py — Health & Performance Science, Fase 1
===================================================================
Acceptance criteria (from health_audit.yaml GAP-H2):
  - Datos obligatorios faltantes → error descriptivo, no crash
  - Campos fuera de rango → ValueError con mensaje claro
  - Red flags médicas detectadas automáticamente → CRIT-R08 trigger
  - Datos completos válidos → output JSON con BMI, clasificación, flags
  - Privacidad: patient_id hasheado, nunca en plaintext en output
  - 10+ tests PASS
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.health.patient_intake import (
    PatientIntake,
    IntakeResult,
    result_to_dict,
    _hash_patient_id,
    _classify_bmi,
    RANGES,
    RED_FLAG_CONDITIONS,
    RED_FLAG_SYMPTOMS,
    VALID_GOALS,
    VALID_ACTIVITY_LEVELS,
)

# ═══════════════════════════════════════════════════════════════════════════
# Fixtures
# ═══════════════════════════════════════════════════════════════════════════

@pytest.fixture
def valid_input() -> dict:
    """Paciente sano — sin red flags."""
    return {
        "patient_id": "paciente-001",
        "demographics": {
            "age": 30,
            "sex": "M",
            "height_cm": 175,
            "weight_kg": 80,
        },
        "health_history": {
            "conditions": [],
            "medications": [],
            "injuries": [],
            "allergies": ["pollen"],
            "surgeries": [],
        },
        "goals": {
            "primary": "muscle_gain",
            "target_weight_kg": 85,
            "timeframe_weeks": 12,
        },
        "lifestyle": {
            "activity_level": "moderately_active",
            "sleep_hours": 7.5,
            "stress_level": "low",
            "occupation_type": "sedentary",
            "meals_per_day": 4,
        },
        "biometrics": {
            "body_fat_pct": 18.0,
            "resting_heart_rate_bpm": 62,
        },
    }


# ═══════════════════════════════════════════════════════════════════════════
# 1. Happy Path — intake completo y válido
# ═══════════════════════════════════════════════════════════════════════════

class TestHappyPath:
    """AC: Datos completos válidos → output JSON con BMI, clasificación, sin red flags."""

    def test_full_valid_intake_produces_correct_output(self, valid_input):
        engine = PatientIntake(valid_input)
        result = engine.process()
        output = result_to_dict(result)

        assert result.patient_id_hashed == _hash_patient_id("paciente-001")
        assert result.demographics.age == 30
        assert result.demographics.sex == "M"
        assert result.demographics.bmi == pytest.approx(26.1, abs=0.2)
        assert result.bmi_category == "overweight"
        assert result.goals.primary == "muscle_gain"
        assert result.goals.target_weight_kg == pytest.approx(85)
        assert output["safety"]["cleared_for_plan"] is True
        assert output["safety"]["red_flags_count"] == 0

    def test_minimal_valid_intake(self):
        """Mínimo requerido para procesar: patient_id + demographics."""
        minimal = {
            "patient_id": "min-001",
            "demographics": {
                "age": 25,
                "sex": "F",
                "height_cm": 165,
                "weight_kg": 60,
            },
        }
        engine = PatientIntake(minimal)
        result = engine.process()

        assert result.demographics.bmi == pytest.approx(22.0, abs=0.2)
        assert result.bmi_category == "normal"
        assert result.goals.primary == "general_health"  # default
        assert result.lifestyle.activity_level == "sedentary"  # default


# ═══════════════════════════════════════════════════════════════════════════
# 2. Schema Validation — campos faltantes y malformados
# ═══════════════════════════════════════════════════════════════════════════

class TestSchemaValidation:
    """AC: Campos obligatorios faltantes → error descriptivo, no crash."""

    def test_missing_patient_id(self):
        with pytest.raises(ValueError, match="patient_id"):
            PatientIntake({"demographics": {"age": 30, "sex": "M", "height_cm": 175, "weight_kg": 80}}).process()

    def test_empty_patient_id(self):
        with pytest.raises(ValueError, match="patient_id"):
            PatientIntake({"patient_id": "   ", "demographics": {"age": 30, "sex": "M", "height_cm": 175, "weight_kg": 80}}).process()

    def test_missing_demographics_section(self):
        with pytest.raises(ValueError, match="demographics"):
            PatientIntake({"patient_id": "x"}).process()

    def test_missing_age(self):
        with pytest.raises(ValueError, match="age"):
            PatientIntake({"patient_id": "x", "demographics": {"sex": "M", "height_cm": 175, "weight_kg": 80}}).process()

    def test_missing_height(self):
        with pytest.raises(ValueError, match="height_cm"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 30, "sex": "M", "weight_kg": 80}}).process()

    def test_missing_weight(self):
        with pytest.raises(ValueError, match="weight_kg"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 30, "sex": "M", "height_cm": 175}}).process()

    def test_input_not_dict_rejected(self):
        with pytest.raises(ValueError, match="diccionario"):
            PatientIntake("string_invalido")  # type: ignore[arg-type]

    def test_demographics_not_dict_rejected(self):
        with pytest.raises(ValueError, match="diccionario"):
            PatientIntake({"patient_id": "x", "demographics": "no_soy_dict"}).process()


# ═══════════════════════════════════════════════════════════════════════════
# 3. Range Validation — campos fuera de rango
# ═══════════════════════════════════════════════════════════════════════════

class TestRangeValidation:
    """AC: Campos fuera de rango → ValueError con mensaje claro que incluye el rango aceptado."""

    def test_age_below_range(self):
        with pytest.raises(ValueError, match="fuera de rango"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 10, "sex": "M", "height_cm": 175, "weight_kg": 80}}).process()

    def test_age_above_range(self):
        with pytest.raises(ValueError, match="fuera de rango"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 130, "sex": "M", "height_cm": 175, "weight_kg": 80}}).process()

    def test_weight_below_range(self):
        with pytest.raises(ValueError, match="fuera de rango"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 30, "sex": "M", "height_cm": 175, "weight_kg": 20}}).process()

    def test_height_above_range(self):
        with pytest.raises(ValueError, match="fuera de rango"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 30, "sex": "M", "height_cm": 300, "weight_kg": 80}}).process()

    def test_invalid_sex_rejected(self):
        with pytest.raises(ValueError, match="sex"):
            PatientIntake({"patient_id": "x", "demographics": {"age": 30, "sex": "X", "height_cm": 175, "weight_kg": 80}}).process()

    def test_non_numeric_age_rejected(self):
        with pytest.raises(ValueError, match="no es numérico"):
            PatientIntake({"patient_id": "x", "demographics": {"age": "treinta", "sex": "M", "height_cm": 175, "weight_kg": 80}}).process()


# ═══════════════════════════════════════════════════════════════════════════
# 4. Red Flag Detection — CRIT-R08
# ═══════════════════════════════════════════════════════════════════════════

class TestRedFlagDetection:
    """AC: Red flags médicas detectadas automáticamente → CRIT-R08 trigger."""

    def test_cardiovascular_disease_triggers_red_flag(self):
        inp = {
            "patient_id": "cardio-001",
            "demographics": {"age": 55, "sex": "M", "height_cm": 175, "weight_kg": 90},
            "health_history": {"conditions": ["cardiovascular_disease"]},
        }
        engine = PatientIntake(inp)
        result = engine.process()
        output = result_to_dict(result)

        assert output["safety"]["cleared_for_plan"] is False
        assert output["safety"]["red_flags_count"] >= 1
        assert any("cardiovascular" in rf["detail"].lower() for rf in output["safety"]["red_flags"])

    def test_multiple_conditions_triggers_multiple_flags(self):
        inp = {
            "patient_id": "multi-001",
            "demographics": {"age": 45, "sex": "F", "height_cm": 165, "weight_kg": 70},
            "health_history": {"conditions": ["diabetes_type1", "kidney_disease", "osteoporosis"]},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert len(result.red_flags) >= 3

    def test_symptom_triggers_red_flag(self):
        inp = {
            "patient_id": "symp-001",
            "demographics": {"age": 35, "sex": "M", "height_cm": 180, "weight_kg": 85},
            "health_history": {"symptoms": ["chest_pain_exercise"]},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert len(result.red_flags) >= 1
        assert result.red_flags[0]["type"] == "symptom"

    def test_critical_bmi_triggers_red_flag(self):
        inp = {
            "patient_id": "bmi-low-001",
            "demographics": {"age": 22, "sex": "F", "height_cm": 165, "weight_kg": 40},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert result.demographics.bmi == pytest.approx(14.7, abs=0.2)
        assert any("bmi_critical" in rf["type"] for rf in result.red_flags)

    def test_high_bp_triggers_critical_flag(self):
        inp = {
            "patient_id": "bp-001",
            "demographics": {"age": 50, "sex": "M", "height_cm": 178, "weight_kg": 95},
            "biometrics": {"blood_pressure_systolic": 190, "blood_pressure_diastolic": 100},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert any("blood_pressure" in rf["type"] for rf in result.red_flags)

    def test_elderly_with_conditions_triggers_red_flag(self):
        inp = {
            "patient_id": "elder-001",
            "demographics": {"age": 70, "sex": "M", "height_cm": 170, "weight_kg": 78},
            "health_history": {"conditions": ["hypertension_uncontrolled"]},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert any("age_risk" in rf["type"] for rf in result.red_flags)


# ═══════════════════════════════════════════════════════════════════════════
# 5. Privacy — PII never exposed in output
# ═══════════════════════════════════════════════════════════════════════════

class TestPrivacy:
    """AC: patient_id hasheado, nunca en plaintext en output."""

    def test_patient_id_is_hashed_in_output(self, valid_input):
        engine = PatientIntake(valid_input)
        result = engine.process()
        output = result_to_dict(result)

        raw_id = valid_input["patient_id"]
        assert raw_id not in json.dumps(output)
        assert output["patient_id_hashed"] == _hash_patient_id(raw_id)
        assert output["patient_id_hashed"] != raw_id

    def test_different_patients_have_different_hashes(self):
        a = PatientIntake({"patient_id": "alice", "demographics": {"age": 30, "sex": "F", "height_cm": 160, "weight_kg": 55}})
        b = PatientIntake({"patient_id": "bob", "demographics": {"age": 30, "sex": "M", "height_cm": 180, "weight_kg": 80}})

        assert a.process().patient_id_hashed != b.process().patient_id_hashed

    def test_same_patient_id_produces_same_hash(self):
        base = {"patient_id": "same-patient", "demographics": {"age": 30, "sex": "F", "height_cm": 160, "weight_kg": 55}}
        h1 = PatientIntake(base).process().patient_id_hashed
        h2 = PatientIntake(base).process().patient_id_hashed

        assert h1 == h2


# ═══════════════════════════════════════════════════════════════════════════
# 6. Edge Cases — goals inválidos, activity levels, warnings
# ═══════════════════════════════════════════════════════════════════════════

class TestEdgeCases:
    """AC: Datos parciales/borde → warnings no bloqueantes, defaults sensatos."""

    def test_invalid_goal_warns_and_defaults(self):
        inp = {
            "patient_id": "goal-001",
            "demographics": {"age": 28, "sex": "F", "height_cm": 160, "weight_kg": 58},
            "goals": {"primary": "become_superhero"},
        }
        engine = PatientIntake(inp)
        result = engine.process()
        output = result_to_dict(result)

        assert result.goals.primary == "general_health"
        assert output["safety"]["warnings_count"] >= 1
        assert any("goal" in w.lower() for w in output["safety"]["warnings"])

    def test_invalid_activity_level_warns_and_defaults(self):
        inp = {
            "patient_id": "act-001",
            "demographics": {"age": 30, "sex": "M", "height_cm": 175, "weight_kg": 80},
            "lifestyle": {"activity_level": "supernova"},
        }
        engine = PatientIntake(inp)
        result = engine.process()

        assert result.lifestyle.activity_level == "sedentary"

    def test_health_history_not_dict_handled_gracefully(self):
        inp = {
            "patient_id": "hist-001",
            "demographics": {"age": 30, "sex": "M", "height_cm": 175, "weight_kg": 80},
            "health_history": "not_a_dict",
        }
        with pytest.raises(ValueError, match="diccionario"):
            PatientIntake(inp).process()


# ═══════════════════════════════════════════════════════════════════════════
# 7. Constants integrity
# ═══════════════════════════════════════════════════════════════════════════

class TestConstants:
    """Las constantes clínicas deben ser coherentes y completas."""

    def test_all_goals_have_descriptive_names(self):
        for goal in VALID_GOALS:
            assert len(goal) > 2
            assert "_" in goal or goal.isalpha()

    def test_all_activity_levels_are_valid(self):
        for level in VALID_ACTIVITY_LEVELS:
            assert level in (
                "sedentary", "lightly_active", "moderately_active",
                "very_active", "athlete",
            )

    def test_bmi_classification_covers_extremes(self):
        assert _classify_bmi(14.0) == "severe_thinness"
        assert _classify_bmi(18.5) == "normal"
        assert _classify_bmi(22.0) == "normal"
        assert _classify_bmi(27.0) == "overweight"
        assert _classify_bmi(32.0) == "obese_class_i"
        assert _classify_bmi(42.0) == "obese_class_iii"

    def test_red_flag_conditions_are_informative(self):
        for cond in RED_FLAG_CONDITIONS:
            assert len(cond) > 2, f"Condition too short: {cond}"
            # Most are snake_case, single-words like 'pregnancy' are exceptions
            assert cond.islower(), f"Condition should be lowercase: {cond}"

    def test_red_flag_symptoms_have_descriptions(self):
        for key, desc in RED_FLAG_SYMPTOMS.items():
            assert len(key) > 3
            assert len(desc) > 10

    def test_all_ranges_have_min_max_unit(self):
        for field, r in RANGES.items():
            assert "min" in r
            assert "max" in r
            assert "unit" in r
            assert r["min"] < r["max"]


# ═══════════════════════════════════════════════════════════════════════════
# 8. Output structure
# ═══════════════════════════════════════════════════════════════════════════

class TestOutputStructure:
    """El output JSON debe tener la estructura esperada."""

    def test_output_has_required_sections(self, valid_input):
        engine = PatientIntake(valid_input)
        result = engine.process()
        output = result_to_dict(result)

        required_keys = [
            "patient_id_hashed", "intake_timestamp",
            "demographics", "health_history", "goals",
            "lifestyle", "biometrics", "safety",
        ]
        for key in required_keys:
            assert key in output, f"Missing key: {key}"

    def test_demographics_include_bmi(self, valid_input):
        engine = PatientIntake(valid_input)
        result = engine.process()
        output = result_to_dict(result)

        assert "bmi" in output["demographics"]
        assert "bmi_category" in output["demographics"]

    def test_safety_section_has_all_counts(self, valid_input):
        engine = PatientIntake(valid_input)
        result = engine.process()
        output = result_to_dict(result)

        safety = output["safety"]
        assert "red_flags_count" in safety
        assert "warnings_count" in safety
        assert "missing_fields_count" in safety
        assert "cleared_for_plan" in safety


# ═══════════════════════════════════════════════════════════════════════════
# 9. CLI — smoke test (validate-only mode)
# ═══════════════════════════════════════════════════════════════════════════

class TestCLI:
    """Smoke test del CLI. No usa subprocess — prueba la lógica main internamente."""

    def test_main_with_validate_only(self, tmp_path, valid_input):
        """CLI con --validate-only sobre archivo válido."""
        import subprocess

        inp_file = tmp_path / "intake.json"
        inp_file.write_text(json.dumps(valid_input), encoding="utf-8")

        result = subprocess.run(
            [sys.executable, "-m", "tools.health.patient_intake",
             str(inp_file), "--validate-only"],
            capture_output=True,
            text=True,
            cwd=str(Path(__file__).resolve().parents[2]),
        )
        assert result.returncode == 0
        assert "OK" in result.stderr

    def test_main_with_invalid_json(self, tmp_path):
        """CLI con JSON inválido debe fallar con código 1."""
        inp_file = tmp_path / "bad.json"
        inp_file.write_text("{bad json", encoding="utf-8")

        import subprocess
        result = subprocess.run(
            [sys.executable, "-m", "tools.health.patient_intake",
             str(inp_file)],
            capture_output=True,
            text=True,
            cwd=str(Path(__file__).resolve().parents[2]),
        )
        assert result.returncode == 1
        assert "JSON" in result.stderr

    def test_main_with_validation_error(self, tmp_path):
        """CLI con datos inválidos (age fuera de rango) debe fallar con código 2."""
        bad = {"patient_id": "x", "demographics": {"age": 10, "sex": "M", "height_cm": 175, "weight_kg": 80}}
        inp_file = tmp_path / "invalid.json"
        inp_file.write_text(json.dumps(bad), encoding="utf-8")

        import subprocess
        result = subprocess.run(
            [sys.executable, "-m", "tools.health.patient_intake",
             str(inp_file)],
            capture_output=True,
            text=True,
            cwd=str(Path(__file__).resolve().parents[2]),
        )
        assert result.returncode == 2

    def test_main_with_output_file(self, tmp_path, valid_input):
        """CLI con --output guarda JSON en archivo."""
        inp_file = tmp_path / "intake.json"
        inp_file.write_text(json.dumps(valid_input), encoding="utf-8")
        out_file = tmp_path / "result.json"

        import subprocess
        result = subprocess.run(
            [sys.executable, "-m", "tools.health.patient_intake",
             str(inp_file), "-o", str(out_file)],
            capture_output=True,
            text=True,
            cwd=str(Path(__file__).resolve().parents[2]),
        )
        assert result.returncode == 0
        assert out_file.exists()
        output = json.loads(out_file.read_text(encoding="utf-8"))
        assert "patient_id_hashed" in output
