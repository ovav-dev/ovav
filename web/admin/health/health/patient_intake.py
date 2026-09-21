#!/usr/bin/env python3
"""
OVAV Patient Intake Engine — patient_intake.py
================================================
Ficha de admisión estructurada para pacientes y deportistas.
Recolecta demográficos, historial médico, objetivos, estilo de vida y
biométricos. Aplica validación de rangos, detección de red flags médicas
y salida JSON estructurada con privacidad del paciente (PII hasheado).

Pipeline: parse → validate → red_flag_scan → anonymize → output

CRIT-R03: Sin datos completos, no hay plan.
CRIT-R08: Red flags médicas → derivar a profesional licenciado.
CRIT-R10: Privacidad del paciente — PII nunca en plaintext.

Spec canónica: .ovav/plan/health_audit.yaml (GAP-H2, Fase 1)
Autor: Renata (Health & Performance Science Lead)
Squad: Fátima (Progress Tracker)
Revisión: Marina (Medical Researcher)
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

# ═══════════════════════════════════════════════════════════════════════════
# Constants — Valid Ranges & Enums
# ═══════════════════════════════════════════════════════════════════════════

VALID_SEX: tuple[str, ...] = ("M", "F")

VALID_ACTIVITY_LEVELS: tuple[str, ...] = (
    "sedentary",
    "lightly_active",
    "moderately_active",
    "very_active",
    "athlete",
)

VALID_STRESS_LEVELS: tuple[str, ...] = ("low", "moderate", "high")

VALID_OCCUPATION_TYPES: tuple[str, ...] = ("sedentary", "standing", "manual_labor")

VALID_GOALS: tuple[str, ...] = (
    "weight_loss",
    "muscle_gain",
    "endurance",
    "general_health",
    "performance",
    "recomposition",
)

# ── Biometric Ranges ──────────────────────────────────────────────────────
# Todos los rangos validados contra literatura clínica (WHO, ACSM, NIH).

RANGES: dict[str, dict[str, float]] = {
    "age":            {"min": 18,  "max": 120, "unit": "years"},
    "height_cm":      {"min": 100, "max": 250,  "unit": "cm"},
    "weight_kg":      {"min": 30,  "max": 300,  "unit": "kg"},
    "body_fat_pct":   {"min": 3,   "max": 60,   "unit": "%"},
    "waist_circumference_cm": {"min": 50,  "max": 200, "unit": "cm"},
    "resting_heart_rate_bpm": {"min": 30,  "max": 120, "unit": "bpm"},
    "blood_pressure_systolic":  {"min": 70,  "max": 250, "unit": "mmHg"},
    "blood_pressure_diastolic": {"min": 40,  "max": 150, "unit": "mmHg"},
    "sleep_hours":    {"min": 0,   "max": 24,   "unit": "hours"},
    "meals_per_day":  {"min": 1,   "max": 8,    "unit": "meals"},
    "target_weight_kg": {"min": 30, "max": 300, "unit": "kg"},
    "timeframe_weeks":  {"min": 1,  "max": 104, "unit": "weeks"},
}

# ── Red Flag Conditions ────────────────────────────────────────────────────
# Condiciones que requieren autorización médica antes de cualquier plan.
# CRIT-R08: detectar → advertir → derivar. No diagnosticar.

RED_FLAG_CONDITIONS: tuple[str, ...] = (
    "cardiovascular_disease",
    "hypertension_uncontrolled",
    "diabetes_type1",
    "diabetes_type2_uncontrolled",
    "kidney_disease",
    "liver_disease",
    "cancer_active",
    "eating_disorder_active",
    "pregnancy",
    "recent_surgery_6months",
    "osteoporosis",
    "thyroid_disorder_uncontrolled",
    "seizure_disorder",
    "respiratory_disease_severe",
)

RED_FLAG_SYMPTOMS: dict[str, str] = {
    "chest_pain_exercise":      "Dolor torácico durante el ejercicio",
    "unexplained_weight_loss":  "Pérdida de peso inexplicada (>5% en 1 mes)",
    "syncope_exercise":         "Desmayo o mareo severo durante ejercicio",
    "severe_joint_pain":        "Dolor articular severo sin diagnóstico",
    "shortness_of_breath_rest": "Dificultad respiratoria en reposo",
    "palpitations_irregular":   "Palpitaciones irregulares frecuentes",
    "edema_unexplained":        "Edema o hinchazón sin causa conocida",
    "fatigue_extreme":          "Fatiga extrema que interfiere con vida diaria",
}

# ── Privacy — Hashing salt ─────────────────────────────────────────────────
# En producción, usar una salt aleatoria por paciente almacenada en vault.
# Para el intake engine, usamos una salt fija que permite verificación
# de unicidad sin exponer el identificador real.

_INTAKE_SALT: str = "ovav_health_intake_v1"


# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════


@dataclass
class Demographics:
    age: int
    sex: str
    height_cm: float
    weight_kg: float
    bmi: float = 0.0

    def __post_init__(self) -> None:
        self.bmi = round(self.weight_kg / ((self.height_cm / 100) ** 2), 1)


@dataclass
class HealthHistory:
    conditions: list[str] = field(default_factory=list)
    medications: list[str] = field(default_factory=list)
    injuries: list[str] = field(default_factory=list)
    allergies: list[str] = field(default_factory=list)
    surgeries: list[str] = field(default_factory=list)


@dataclass
class Goals:
    primary: str = "general_health"
    target_weight_kg: float | None = None
    timeframe_weeks: int | None = None
    secondary: list[str] = field(default_factory=list)


@dataclass
class Lifestyle:
    activity_level: str = "sedentary"
    sleep_hours: float = 7.0
    stress_level: str = "moderate"
    occupation_type: str = "sedentary"
    meals_per_day: int = 3


@dataclass
class Biometrics:
    body_fat_pct: float | None = None
    waist_circumference_cm: float | None = None
    resting_heart_rate_bpm: int | None = None
    blood_pressure_systolic: int | None = None
    blood_pressure_diastolic: int | None = None


@dataclass
class IntakeResult:
    """Estructura de salida completa con todos los datos validados y flags."""
    patient_id_hashed: str
    demographics: Demographics
    health_history: HealthHistory
    goals: Goals
    lifestyle: Lifestyle
    biometrics: Biometrics
    red_flags: list[dict[str, str]] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    missing_fields: list[str] = field(default_factory=list)
    bmi_category: str = ""
    intake_timestamp: str = ""

    def __post_init__(self) -> None:
        self.bmi_category = _classify_bmi(self.demographics.bmi)
        if not self.intake_timestamp:
            self.intake_timestamp = datetime.now(timezone.utc).isoformat()


# ═══════════════════════════════════════════════════════════════════════════
# Validation Helpers
# ═══════════════════════════════════════════════════════════════════════════


def _validate_range(value: Any, field_name: str) -> float | None:
    """Valida que un valor numérico esté dentro del rango permitido.
    Retorna el valor como float si es válido, o None si está fuera de rango.
    Lanza ValueError si el valor no es numérico."""
    if value is None:
        return None
    try:
        v = float(value)
    except (TypeError, ValueError):
        raise ValueError(
            f"Campo '{field_name}': valor '{value}' no es numérico. "
            f"Rango esperado: {RANGES[field_name]['min']}–{RANGES[field_name]['max']} "
            f"{RANGES[field_name]['unit']}."
        )
    r = RANGES[field_name]
    if v < r["min"] or v > r["max"]:
        raise ValueError(
            f"Campo '{field_name}': {v} {r['unit']} fuera de rango. "
            f"Rango aceptado: {r['min']}–{r['max']} {r['unit']}."
        )
    return v


def _hash_patient_id(raw_id: str) -> str:
    """Hashea el identificador del paciente con SHA-256 + salt.
    El hash es determinístico — permite verificar unicidad sin exponer PII."""
    return hashlib.sha256(f"{raw_id}:{_INTAKE_SALT}".encode()).hexdigest()[:16]


def _classify_bmi(bmi: float) -> str:
    """Clasificación OMS del IMC."""
    if bmi < 16.0:
        return "severe_thinness"
    if bmi < 17.0:
        return "moderate_thinness"
    if bmi < 18.5:
        return "mild_thinness"
    if bmi < 25.0:
        return "normal"
    if bmi < 30.0:
        return "overweight"
    if bmi < 35.0:
        return "obese_class_i"
    if bmi < 40.0:
        return "obese_class_ii"
    return "obese_class_iii"


# ═══════════════════════════════════════════════════════════════════════════
# PatientIntake — Main Engine
# ═══════════════════════════════════════════════════════════════════════════


class PatientIntake:
    """Motor de admisión de pacientes.
    Uso:
        engine = PatientIntake(input_dict)
        result = engine.process()
        print(result.to_json())
    """

    def __init__(self, raw: dict[str, Any]) -> None:
        if not isinstance(raw, dict):
            raise ValueError("Input debe ser un diccionario (YAML/JSON parseado).")
        self._raw: dict[str, Any] = raw
        self._warnings: list[str] = []
        self._red_flags: list[dict[str, str]] = []
        self._missing: list[str] = []

    # ── Main Pipeline ─────────────────────────────────────────────────────

    def process(self) -> IntakeResult:
        """Ejecuta el pipeline completo: parse → validate → scan → output."""
        patient_id = self._extract_patient_id()
        demographics = self._parse_demographics()
        health_history = self._parse_health_history()
        goals = self._parse_goals()
        lifestyle = self._parse_lifestyle()
        biometrics = self._parse_biometrics()

        self._scan_red_flags(demographics, health_history, biometrics)

        return IntakeResult(
            patient_id_hashed=_hash_patient_id(patient_id),
            demographics=demographics,
            health_history=health_history,
            goals=goals,
            lifestyle=lifestyle,
            biometrics=biometrics,
            red_flags=self._red_flags,
            warnings=self._warnings,
            missing_fields=self._missing,
        )

    # ── Field Extractors ──────────────────────────────────────────────────

    def _extract_patient_id(self) -> str:
        pid = self._raw.get("patient_id")
        if not pid or not isinstance(pid, str) or not pid.strip():
            raise ValueError(
                "Campo requerido 'patient_id' faltante o vacío. "
                "Debe ser un string identificador único (nombre, email, o código)."
            )
        return pid.strip()

    def _parse_demographics(self) -> Demographics:
        d = self._raw.get("demographics", {})
        if not isinstance(d, dict):
            raise ValueError("Sección 'demographics' debe ser un diccionario.")

        age_raw = d.get("age")
        if age_raw is None:
            self._missing.append("demographics.age")
            raise ValueError(
                "Campo requerido 'demographics.age' faltante. "
                "Edad del paciente (18–120 años)."
            )
        age = int(_validate_range(age_raw, "age"))  # type: ignore[arg-type]

        sex = d.get("sex", "").upper().strip()
        if sex not in VALID_SEX:
            raise ValueError(
                f"Campo 'demographics.sex': '{sex}' no válido. "
                f"Valores aceptados: {', '.join(VALID_SEX)}."
            )

        height_raw = d.get("height_cm")
        if height_raw is None:
            self._missing.append("demographics.height_cm")
            raise ValueError(
                "Campo requerido 'demographics.height_cm' faltante. "
                "Altura en centímetros (100–250)."
            )
        height_cm = _validate_range(height_raw, "height_cm")
        if height_cm is None:
            raise ValueError("demographics.height_cm no puede ser nulo.")

        weight_raw = d.get("weight_kg")
        if weight_raw is None:
            self._missing.append("demographics.weight_kg")
            raise ValueError(
                "Campo requerido 'demographics.weight_kg' faltante. "
                "Peso en kilogramos (30–300)."
            )
        weight_kg = _validate_range(weight_raw, "weight_kg")
        if weight_kg is None:
            raise ValueError("demographics.weight_kg no puede ser nulo.")

        return Demographics(age=age, sex=sex, height_cm=height_cm, weight_kg=weight_kg)

    def _parse_health_history(self) -> HealthHistory:
        h = self._raw.get("health_history", {})
        if not isinstance(h, dict):
            raise ValueError("Sección 'health_history' debe ser un diccionario.")

        conditions = self._extract_string_list(h, "conditions")
        medications = self._extract_string_list(h, "medications")
        injuries = self._extract_string_list(h, "injuries")
        allergies = self._extract_string_list(h, "allergies")
        surgeries = self._extract_string_list(h, "surgeries")

        return HealthHistory(
            conditions=conditions,
            medications=medications,
            injuries=injuries,
            allergies=allergies,
            surgeries=surgeries,
        )

    def _parse_goals(self) -> Goals:
        g = self._raw.get("goals", {})
        if not isinstance(g, dict):
            self._missing.append("goals")
            return Goals()

        primary = g.get("primary", "general_health")
        if primary not in VALID_GOALS:
            self._warnings.append(
                f"Goal '{primary}' no está en la lista canónica. "
                f"Válidos: {', '.join(VALID_GOALS)}. Usando 'general_health'."
            )
            primary = "general_health"

        target_weight = None
        if g.get("target_weight_kg") is not None:
            try:
                target_weight = _validate_range(g["target_weight_kg"], "target_weight_kg")
            except ValueError:
                self._warnings.append("target_weight_kg fuera de rango — ignorado.")

        timeframe = None
        if g.get("timeframe_weeks") is not None:
            try:
                timeframe = int(_validate_range(g["timeframe_weeks"], "timeframe_weeks"))  # type: ignore[arg-type]
            except ValueError:
                self._warnings.append("timeframe_weeks fuera de rango — ignorado.")

        secondary = self._extract_string_list(g, "secondary")

        return Goals(
            primary=primary,
            target_weight_kg=target_weight,
            timeframe_weeks=timeframe,
            secondary=secondary,
        )

    def _parse_lifestyle(self) -> Lifestyle:
        ls = self._raw.get("lifestyle", {})
        if not isinstance(ls, dict):
            self._missing.append("lifestyle")
            return Lifestyle()

        activity = ls.get("activity_level", "sedentary")
        if activity not in VALID_ACTIVITY_LEVELS:
            self._warnings.append(
                f"activity_level '{activity}' no válido. Usando 'sedentary'."
            )
            activity = "sedentary"

        sleep = 7.0
        if ls.get("sleep_hours") is not None:
            try:
                sleep = float(_validate_range(ls["sleep_hours"], "sleep_hours"))  # type: ignore[arg-type]
            except ValueError:
                self._warnings.append("sleep_hours fuera de rango — usando 7.0.")

        stress = ls.get("stress_level", "moderate")
        if stress not in VALID_STRESS_LEVELS:
            self._warnings.append(
                f"stress_level '{stress}' no válido. Usando 'moderate'."
            )
            stress = "moderate"

        occupation = ls.get("occupation_type", "sedentary")
        if occupation not in VALID_OCCUPATION_TYPES:
            self._warnings.append(
                f"occupation_type '{occupation}' no válido. Usando 'sedentary'."
            )
            occupation = "sedentary"

        meals = 3
        if ls.get("meals_per_day") is not None:
            try:
                meals = int(_validate_range(ls["meals_per_day"], "meals_per_day"))  # type: ignore[arg-type]
            except ValueError:
                self._warnings.append("meals_per_day fuera de rango — usando 3.")

        return Lifestyle(
            activity_level=activity,
            sleep_hours=sleep,
            stress_level=stress,
            occupation_type=occupation,
            meals_per_day=meals,
        )

    def _parse_biometrics(self) -> Biometrics:
        b = self._raw.get("biometrics", {})
        if not isinstance(b, dict):
            return Biometrics()

        def _safe_optional_int(key: str) -> int | None:
            val = b.get(key)
            if val is None:
                return None
            try:
                return int(_validate_range(val, key))  # type: ignore[arg-type]
            except ValueError:
                self._warnings.append(f"{key} fuera de rango — ignorado.")
                return None

        def _safe_optional_float(key: str) -> float | None:
            val = b.get(key)
            if val is None:
                return None
            try:
                return _validate_range(val, key)
            except ValueError:
                self._warnings.append(f"{key} fuera de rango — ignorado.")
                return None

        return Biometrics(
            body_fat_pct=_safe_optional_float("body_fat_pct"),
            waist_circumference_cm=_safe_optional_float("waist_circumference_cm"),
            resting_heart_rate_bpm=_safe_optional_int("resting_heart_rate_bpm"),
            blood_pressure_systolic=_safe_optional_int("blood_pressure_systolic"),
            blood_pressure_diastolic=_safe_optional_int("blood_pressure_diastolic"),
        )

    # ── Red Flag Scanner ──────────────────────────────────────────────────

    def _scan_red_flags(
        self,
        demographics: Demographics,
        history: HealthHistory,
        bio: Biometrics,
    ) -> None:
        """CRIT-R08: Detecta condiciones que requieren autorización médica."""
        # Condiciones conocidas que son red flags
        for condition in history.conditions:
            cond_lower = condition.lower().replace(" ", "_")
            if cond_lower in RED_FLAG_CONDITIONS:
                self._red_flags.append({
                    "type": "medical_condition",
                    "detail": condition,
                    "action": "Requiere autorización médica antes de iniciar cualquier plan.",
                    "severity": "high",
                })

        # Síntomas reportados
        symptoms_raw = self._raw.get("health_history", {}).get("symptoms", [])
        if isinstance(symptoms_raw, list):
            for symptom in symptoms_raw:
                if isinstance(symptom, str):
                    sym_key = symptom.lower().replace(" ", "_")
                    if sym_key in RED_FLAG_SYMPTOMS:
                        self._red_flags.append({
                            "type": "symptom",
                            "detail": RED_FLAG_SYMPTOMS[sym_key],
                            "action": "Evaluación médica requerida antes de continuar.",
                            "severity": "high",
                        })

        # BMI extremos
        if demographics.bmi < 16.0:
            self._red_flags.append({
                "type": "bmi_critical",
                "detail": f"IMC severamente bajo: {demographics.bmi}",
                "action": "Requiere evaluación médica y plan supervisado.",
                "severity": "high",
            })
        elif demographics.bmi > 45.0:
            self._red_flags.append({
                "type": "bmi_critical",
                "detail": f"IMC muy elevado: {demographics.bmi}",
                "action": "Requiere autorización médica antes de iniciar ejercicio.",
                "severity": "high",
            })

        # Presión arterial elevada
        if bio.blood_pressure_systolic and bio.blood_pressure_systolic > 180:
            self._red_flags.append({
                "type": "blood_pressure_critical",
                "detail": f"Presión sistólica elevada: {bio.blood_pressure_systolic} mmHg",
                "action": "Requiere evaluación médica urgente.",
                "severity": "critical",
            })
        if bio.blood_pressure_diastolic and bio.blood_pressure_diastolic > 120:
            self._red_flags.append({
                "type": "blood_pressure_critical",
                "detail": f"Presión diastólica elevada: {bio.blood_pressure_diastolic} mmHg",
                "action": "Requiere evaluación médica urgente.",
                "severity": "critical",
            })

        # RHR extremo
        if bio.resting_heart_rate_bpm is not None:
            if bio.resting_heart_rate_bpm > 100:
                self._red_flags.append({
                    "type": "rhr_elevated",
                    "detail": f"Frecuencia cardíaca en reposo elevada: {bio.resting_heart_rate_bpm} bpm",
                    "action": "Sugerir evaluación médica antes de ejercicio intenso.",
                    "severity": "medium",
                })
            elif bio.resting_heart_rate_bpm < 40:
                self._red_flags.append({
                    "type": "rhr_low",
                    "detail": f"Frecuencia cardíaca en reposo muy baja: {bio.resting_heart_rate_bpm} bpm",
                    "action": "Verificar si es atleta entrenado o requiere evaluación.",
                    "severity": "medium",
                })

        # Edad > 65 con condiciones
        if demographics.age > 65 and len(history.conditions) > 0:
            self._red_flags.append({
                "type": "age_risk",
                "detail": f"Paciente >65 años ({demographics.age}) con condiciones preexistentes.",
                "action": "Requiere autorización médica antes de cualquier plan.",
                "severity": "high",
            })

    # ── Utility ───────────────────────────────────────────────────────────

    @staticmethod
    def _extract_string_list(source: dict[str, Any], key: str) -> list[str]:
        """Extrae una lista de strings de un dict, con tolerancia a errores."""
        val = source.get(key, [])
        if not isinstance(val, list):
            return []
        return [str(item).strip() for item in val if isinstance(item, str) and item.strip()]


# ═══════════════════════════════════════════════════════════════════════════
# Output Formatter
# ═══════════════════════════════════════════════════════════════════════════


def result_to_dict(result: IntakeResult) -> dict[str, Any]:
    """Convierte IntakeResult a un diccionario serializable a JSON.
    No incluye PII en texto plano — el patient_id está hasheado."""
    rf = result.red_flags
    return {
        "patient_id_hashed": result.patient_id_hashed,
        "intake_timestamp": result.intake_timestamp,
        "demographics": {
            "age": result.demographics.age,
            "sex": result.demographics.sex,
            "height_cm": result.demographics.height_cm,
            "weight_kg": result.demographics.weight_kg,
            "bmi": result.demographics.bmi,
            "bmi_category": result.bmi_category,
        },
        "health_history": {
            "conditions": result.health_history.conditions,
            "medications": result.health_history.medications,
            "injuries": result.health_history.injuries,
            "allergies": result.health_history.allergies,
            "surgeries": result.health_history.surgeries,
        },
        "goals": {
            "primary": result.goals.primary,
            "target_weight_kg": result.goals.target_weight_kg,
            "timeframe_weeks": result.goals.timeframe_weeks,
            "secondary": result.goals.secondary,
        },
        "lifestyle": {
            "activity_level": result.lifestyle.activity_level,
            "sleep_hours": result.lifestyle.sleep_hours,
            "stress_level": result.lifestyle.stress_level,
            "occupation_type": result.lifestyle.occupation_type,
            "meals_per_day": result.lifestyle.meals_per_day,
        },
        "biometrics": {
            "body_fat_pct": result.biometrics.body_fat_pct,
            "waist_circumference_cm": result.biometrics.waist_circumference_cm,
            "resting_heart_rate_bpm": result.biometrics.resting_heart_rate_bpm,
            "blood_pressure_systolic": result.biometrics.blood_pressure_systolic,
            "blood_pressure_diastolic": result.biometrics.blood_pressure_diastolic,
        },
        "safety": {
            "red_flags_count": len(rf),
            "red_flags": rf,
            "warnings_count": len(result.warnings),
            "warnings": result.warnings,
            "missing_fields_count": len(result.missing_fields),
            "missing_fields": result.missing_fields,
            "cleared_for_plan": len(rf) == 0,
        },
    }


# ═══════════════════════════════════════════════════════════════════════════
# CLI Entry Point
# ═══════════════════════════════════════════════════════════════════════════


def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Patient Intake — Admisión de pacientes y deportistas.",
        epilog="CRIT-R08: Red flags médicas detectadas → derivar a profesional.",
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="Archivo JSON de admisión del paciente. Si no se especifica, lee stdin.",
    )
    parser.add_argument(
        "--output", "-o",
        help="Archivo de salida JSON. Si no se especifica, escribe a stdout.",
    )
    parser.add_argument(
        "--validate-only",
        action="store_true",
        help="Solo validar, no generar output completo.",
    )
    parser.add_argument(
        "--version",
        action="version",
        version="patient_intake.py v1.0.0 — OVAV Health & Performance Science",
    )

    args = parser.parse_args()

    # Read input
    if args.input_file:
        raw_text = Path(args.input_file).read_text(encoding="utf-8")
    else:
        raw_text = sys.stdin.read()

    try:
        raw = json.loads(raw_text)
    except json.JSONDecodeError as e:
        print(f"ERROR: JSON inválido: {e}", file=sys.stderr)
        sys.exit(1)

    # Process
    try:
        engine = PatientIntake(raw)
        result = engine.process()
    except ValueError as e:
        print(f"ERROR de validación: {e}", file=sys.stderr)
        sys.exit(2)

    output = result_to_dict(result)

    if args.validate_only:
        issues = output["safety"]["red_flags_count"] + output["safety"]["warnings_count"]
        if issues > 0:
            print(f"VALIDATION: {issues} issue(s) encontrados.", file=sys.stderr)
        else:
            print("VALIDATION: OK — sin red flags ni warnings.", file=sys.stderr)
        sys.exit(0 if output["safety"]["cleared_for_plan"] else 3)

    json_out = json.dumps(output, indent=2, ensure_ascii=False)

    if args.output:
        Path(args.output).write_text(json_out, encoding="utf-8")
        print(f"Intake guardado en {args.output}")
    else:
        print(json_out)


if __name__ == "__main__":
    main()
