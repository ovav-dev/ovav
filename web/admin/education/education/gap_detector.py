#!/usr/bin/env python3
"""
OVAV Gap Detector — gap_detector.py
=====================================
Self-assessment analysis engine. Recibe un YAML del estudiante y produce
un JSON estructurado con gaps de conocimiento identificados.

Pipeline: parse → normalize → estimate_real_level (RPKT) → compute_gap → prioritize

Modelo RPKT (Rasch-based Proficiency Knowledge Tracing):
    Combina self_rated_level, years_experience, confidence y evidence en una
    estimación del nivel real. Aplica Dunning-Kruger correction: si confidence
    es alta pero la evidencia es escasa, degrada el nivel estimado.

Spec canónica: .ovav/plan/education_roadmap.yaml (líneas 22-200)
Taxonomía canónica: .ovav/plan/education_roadmap.yaml (líneas 604-895)
Autor spec: Valeria + Carmen + Sandra
Implementación: Thavren (Platform Engineering)
"""

from __future__ import annotations

import argparse
import json
import math
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from difflib import SequenceMatcher
from pathlib import Path
from typing import Any

import yaml

# ═══════════════════════════════════════════════════════════════════════════
# Constants — Taxonomy & Level Mapping
# ═══════════════════════════════════════════════════════════════════════════

LEVEL_ORDER: dict[str, int] = {
    "beginner": 0,
    "novice": 1,
    "intermediate": 2,
    "advanced": 3,
    "expert": 4,
}

LEVEL_NAMES: list[str] = ["beginner", "novice", "intermediate", "advanced", "expert"]

# Mapeo canónico ID → nombre (Valeria adjustment: incluir mapping explícito)
LEVEL_ID_MAP: dict[str, str] = {
    "L0": "beginner",
    "L1": "novice",
    "L2": "intermediate",
    "L3": "advanced",
    "L4": "expert",
}

# ── Career Goal → Required Skills Matrix ──────────────────────────────────
# Define el target_level para cada skill según el career_goal.
# Fuente: skills_taxonomy.target_level_by_role en education_roadmap.yaml

CAREER_REQUIREMENTS: dict[str, dict[str, str]] = {
    "Data Scientist": {
        "python": "advanced",
        "sql": "intermediate",
        "statistics": "advanced",
        "machine_learning": "intermediate",
        "data_visualization": "intermediate",
        "git": "intermediate",
        "technical_writing": "intermediate",
        "problem_solving": "advanced",
    },
    "Backend Developer": {
        "python": "advanced",
        "go": "advanced",
        "sql": "advanced",
        "git": "intermediate",
        "backend_development": "expert",
        "devops_and_cloud": "intermediate",
        "cybersecurity": "intermediate",
        "shell_scripting": "intermediate",
        "system_design": "advanced",
        "technical_writing": "intermediate",
        "code_review": "intermediate",
        "problem_solving": "advanced",
    },
    "Full-Stack Developer": {
        "python": "intermediate",
        "javascript": "advanced",
        "sql": "intermediate",
        "git": "intermediate",
        "frontend_development": "advanced",
        "backend_development": "advanced",
        "system_design": "intermediate",
        "devops_and_cloud": "intermediate",
        "technical_writing": "intermediate",
        "code_review": "intermediate",
        "problem_solving": "advanced",
    },
    "ML Engineer": {
        "python": "advanced",
        "sql": "intermediate",
        "statistics": "advanced",
        "machine_learning": "expert",
        "deep_learning": "advanced",
        "data_engineering": "intermediate",
        "git": "intermediate",
        "technical_writing": "intermediate",
        "problem_solving": "advanced",
    },
    "DevOps Engineer": {
        "python": "intermediate",
        "go": "advanced",
        "shell_scripting": "advanced",
        "sql": "intermediate",
        "git": "intermediate",
        "devops_and_cloud": "expert",
        "cybersecurity": "intermediate",
        "system_design": "intermediate",
        "technical_writing": "intermediate",
        "problem_solving": "advanced",
    },
    "Data Engineer": {
        "python": "advanced",
        "sql": "expert",
        "data_engineering": "expert",
        "statistics": "intermediate",
        "shell_scripting": "intermediate",
        "devops_and_cloud": "intermediate",
        "git": "intermediate",
        "technical_writing": "intermediate",
        "problem_solving": "advanced",
    },
    "Frontend Developer": {
        "javascript": "expert",
        "frontend_development": "expert",
        "sql": "novice",
        "git": "intermediate",
        "technical_writing": "intermediate",
        "code_review": "intermediate",
        "problem_solving": "advanced",
    },
}

# Canonical skill names — usados para fuzzy matching
CANONICAL_SKILLS: set[str] = {
    "python", "javascript", "go", "sql", "git", "shell_scripting",
    "statistics", "machine_learning", "data_visualization", "deep_learning",
    "data_engineering",
    "frontend_development", "backend_development", "devops_and_cloud",
    "system_design", "cybersecurity",
    "technical_writing", "code_review", "problem_solving", "project_management",
}

# Fuzzy match threshold
FUZZY_THRESHOLD: float = 0.75

# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════


@dataclass
class SkillAssessment:
    """Una skill evaluada por el estudiante."""
    name: str
    self_rated_level: str
    years_experience: float
    last_used: str
    confidence: float
    evidence: list[str] = field(default_factory=list)
    canonical_name: str | None = None
    fuzzy_match_score: float = 1.0


@dataclass
class GapResult:
    """Un gap de conocimiento identificado."""
    skill: str
    estimated_level: str
    target_level: str
    gap_severity: int
    priority: int
    dunning_kruger_flag: bool
    rationale: str


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════


class GapDetector:
    """Motor de detección de gaps de conocimiento."""

    def __init__(self, input_yaml: dict[str, Any]):
        self.raw = input_yaml
        self.student_id: str = ""
        self.career_goal: str = ""
        self.assessments: list[SkillAssessment] = []
        self.gaps: list[GapResult] = []
        self.unmapped_skills: list[str] = []
        self._parse_input()

    # ── Step 1: Parse ─────────────────────────────────────────────────────

    def _parse_input(self) -> None:
        """Validar schema YAML. Rechazar con error descriptivo si falta campo."""
        required = ["student_id", "career_goal", "self_assessment"]
        for field in required:
            if field not in self.raw:
                raise ValueError(
                    f"Campo obligatorio '{field}' faltante en el input YAML."
                )

        self.student_id = str(self.raw["student_id"])
        self.career_goal = str(self.raw["career_goal"])

        if self.career_goal not in CAREER_REQUIREMENTS:
            known = ", ".join(sorted(CAREER_REQUIREMENTS.keys()))
            raise ValueError(
                f"Career goal '{self.career_goal}' no reconocido. "
                f"Opciones válidas: {known}"
            )

        raw_assessments = self.raw["self_assessment"]
        if not isinstance(raw_assessments, list):
            raise ValueError("self_assessment debe ser una lista de skills.")

        for item in raw_assessments:
            if not isinstance(item, dict):
                raise ValueError(f"Cada skill debe ser un objeto. Recibido: {type(item)}")
            # El spec usa 'skill' como key para el nombre (ver example_input en education_roadmap.yaml)
            skill_name = item.get("skill") or item.get("name")
            if not skill_name:
                raise ValueError("Cada skill requiere un campo 'skill' (o 'name').")
            if "self_rated_level" not in item:
                raise ValueError(f"Skill '{skill_name}' requiere 'self_rated_level'.")

            level = item.get("self_rated_level", "").lower()
            if level not in LEVEL_ORDER:
                raise ValueError(
                    f"Nivel '{item['self_rated_level']}' no válido para skill "
                    f"'{item['name']}'. Opciones: {list(LEVEL_ORDER.keys())}"
                )

            confidence = float(item.get("confidence", 0.5))
            if not (0.0 <= confidence <= 1.0):
                raise ValueError(
                    f"Confidence debe estar entre 0.0 y 1.0. "
                    f"Recibido: {confidence} para skill '{item['name']}'"
                )

            self.assessments.append(SkillAssessment(
                name=str(skill_name),
                self_rated_level=level,
                years_experience=float(item.get("years_experience", 0.0)),
                last_used=str(item.get("last_used", "never")),
                confidence=confidence,
                evidence=list(item.get("evidence", [])),
            ))

    # ── Step 2: Normalize ─────────────────────────────────────────────────

    def normalize_skills(self) -> None:
        """Mapear nombres a taxonomía canónica con fuzzy matching."""
        for sa in self.assessments:
            name_lower = sa.name.lower().strip()

            # Exact match
            if name_lower in CANONICAL_SKILLS:
                sa.canonical_name = name_lower
                sa.fuzzy_match_score = 1.0
                continue

            # ID mapping (L0→beginner, etc.)
            if name_lower in LEVEL_ID_MAP:
                # Es un level ID, no una skill — error
                self.unmapped_skills.append(sa.name)
                continue

            # Fuzzy match
            best_score = 0.0
            best_match = None
            for canonical in CANONICAL_SKILLS:
                score = SequenceMatcher(None, name_lower, canonical).ratio()
                # Boost: si el nombre contiene el canonical
                if canonical in name_lower or name_lower in canonical:
                    score = max(score, 0.85)
                if score > best_score:
                    best_score = score
                    best_match = canonical

            if best_score >= FUZZY_THRESHOLD and best_match is not None:
                sa.canonical_name = best_match
                sa.fuzzy_match_score = best_score
            else:
                self.unmapped_skills.append(sa.name)
                # Intentar mapear de todas formas si score > 0.5
                if best_score >= 0.5 and best_match is not None:
                    sa.canonical_name = best_match
                    sa.fuzzy_match_score = best_score

    # ── Step 3: Estimate Real Level (RPKT + Dunning-Kruger) ────────────────

    def estimate_real_level(self, sa: SkillAssessment) -> tuple[str, bool]:
        """
        RPKT: combinar self_rated_level, years_experience, confidence y evidence
        en una estimación del nivel real.

        Dunning-Kruger correction: si confidence es alta pero evidence es escasa,
        degradar nivel estimado 1 categoría.
        """
        self_rated_idx = LEVEL_ORDER.get(sa.self_rated_level, 0)

        # ── Evidence score (0.0–1.0) ──
        evidence_count = len(sa.evidence)
        evidence_score: float
        if evidence_count == 0:
            evidence_score = 0.0
        elif evidence_count == 1:
            evidence_score = 0.3
        elif evidence_count == 2:
            evidence_score = 0.6
        elif evidence_count <= 4:
            evidence_score = 0.8
        else:
            evidence_score = 1.0

        # ── Experience score (0.0–1.0) ──
        # 0 years → 0.0, 1 year → 0.3, 3 years → 0.6, 5+ years → 0.8, 10+ → 1.0
        exp = sa.years_experience
        if exp <= 0:
            experience_score = 0.0
        elif exp < 1:
            experience_score = 0.2
        elif exp < 2:
            experience_score = 0.4
        elif exp < 4:
            experience_score = 0.6
        elif exp < 8:
            experience_score = 0.8
        else:
            experience_score = 1.0

        # ── Recency factor ──
        if sa.last_used == "never":
            recency_score = 0.0
        else:
            try:
                last_date = datetime.fromisoformat(sa.last_used.replace("Z", "+00:00"))
                months_ago = (datetime.now(timezone.utc) - last_date).days / 30.0
                if months_ago < 1:
                    recency_score = 1.0
                elif months_ago < 3:
                    recency_score = 0.8
                elif months_ago < 6:
                    recency_score = 0.6
                elif months_ago < 12:
                    recency_score = 0.3
                else:
                    recency_score = 0.1
            except (ValueError, TypeError):
                recency_score = 0.5  # unknown date

        # ── Bayesian combination ──
        # Peso: evidence 40%, experience 30%, recency 30%
        combined_score = (
            evidence_score * 0.40
            + experience_score * 0.30
            + recency_score * 0.30
        )

        # ── Map combined score to estimated level ──
        # self_rated es el prior bayesiano; el combined score ajusta
        # Si combined_score es alto (>0.6), subir 1 nivel respecto al self_rated
        # Si combined_score es bajo (<0.3), bajar 1 nivel
        # Si está en medio, mantener self_rated

        estimated_idx = self_rated_idx
        if combined_score > 0.65:
            estimated_idx = min(self_rated_idx + 1, 4)
        elif combined_score < 0.25:
            estimated_idx = max(self_rated_idx - 1, 0)

        # ── Dunning-Kruger correction ──
        dk_flag = False
        if sa.confidence > 0.6 and evidence_score < 0.4:
            # Alta confianza, poca evidencia → sobreestimación
            dk_flag = True
            estimated_idx = max(estimated_idx - 1, 0)
        elif sa.confidence < 0.3 and evidence_score > 0.5:
            # Baja confianza, buena evidencia → subestimación (impostor syndrome)
            # No degradamos, pero podríamos subir
            pass

        estimated_level = LEVEL_NAMES[estimated_idx]
        return estimated_level, dk_flag

    # ── Step 4: Compute Gaps ──────────────────────────────────────────────

    def compute_gaps(self) -> None:
        """Comparar nivel estimado con target_level para cada skill requerida."""
        requirements = CAREER_REQUIREMENTS.get(self.career_goal, {})

        # Mapa: canonical_name → SkillAssessment
        sa_map: dict[str, SkillAssessment] = {}
        for sa in self.assessments:
            if sa.canonical_name:
                sa_map[sa.canonical_name] = sa

        self.gaps = []
        for skill, target_level in requirements.items():
            target_idx = LEVEL_ORDER.get(target_level, 2)

            if skill in sa_map:
                sa = sa_map[skill]
                estimated_level, dk_flag = self.estimate_real_level(sa)
                estimated_idx = LEVEL_ORDER[estimated_level]

                if estimated_idx >= target_idx:
                    # No gap — nivel suficiente
                    continue

                gap_severity = target_idx - estimated_idx
                rationale = self._build_rationale(
                    sa, estimated_level, target_level, dk_flag, gap_severity
                )

                self.gaps.append(GapResult(
                    skill=skill,
                    estimated_level=estimated_level,
                    target_level=target_level,
                    gap_severity=gap_severity,
                    priority=0,  # se calcula en prioritize
                    dunning_kruger_flag=dk_flag,
                    rationale=rationale,
                ))
            else:
                # Skill no evaluada por el estudiante → gap máximo
                estimated_idx = 0
                gap_severity = target_idx - estimated_idx
                self.gaps.append(GapResult(
                    skill=skill,
                    estimated_level="beginner",
                    target_level=target_level,
                    gap_severity=gap_severity,
                    priority=0,
                    dunning_kruger_flag=False,
                    rationale=(
                        f"No se proporcionó evaluación para '{skill}'. "
                        f"Asumiendo nivel beginner. Requerido {target_level} "
                        f"para {self.career_goal}."
                    ),
                ))

    def _build_rationale(
        self,
        sa: SkillAssessment,
        estimated_level: str,
        target_level: str,
        dk_flag: bool,
        gap_severity: int,
    ) -> str:
        """Construir rationale legible para el gap."""
        parts = [
            f"Autoevaluado como {sa.self_rated_level} "
            f"(confianza: {sa.confidence:.1f}, "
            f"{sa.years_experience:.1f} años, "
            f"{len(sa.evidence)} evidencias)."
        ]
        if dk_flag:
            parts.append(
                "Dunning-Kruger correction aplicada: confianza alta "
                "con evidencia débil → nivel estimado degradado."
            )
        parts.append(
            f"Nivel estimado: {estimated_level}. "
            f"Target para {self.career_goal}: {target_level}. "
            f"Gap: {gap_severity} nivel(es)."
        )
        return " ".join(parts)

    # ── Step 5: Prioritize ────────────────────────────────────────────────

    def prioritize(self) -> None:
        """
        Ordenar gaps por:
        1. Prerrequisitos bloqueantes (skills con dependencias hard)
        2. Severidad del gap (descendente)
        3. Criticidad para el career_goal
        """
        # Hard prerequisites: skills que bloquean otras
        # Fuente: grafo de prerrequisitos del curriculum_engine
        blocking_skills = {
            "python": 1,       # Bloquea machine_learning, data_engineering, etc.
            "statistics": 1,   # Bloquea machine_learning
            "sql": 2,          # Bloquea data_engineering parcialmente
            "javascript": 1,   # Bloquea frontend_development
            "git": 3,          # Bloquea todo parcialmente (soft prereq)
        }

        for gap in self.gaps:
            # Prioridad base: severidad (4=crítico → prioridad alta)
            base = 10 - gap.gap_severity * 2  # gap 4 → priority 2, gap 1 → priority 8

            # Boost para skills bloqueantes
            if gap.skill in blocking_skills:
                base -= blocking_skills[gap.skill]  # más negativo = más prioritario

            # Boost para DK flag (riesgo de autoevaluación incorrecta)
            if gap.dunning_kruger_flag:
                base -= 1

            gap.priority = max(1, base)

        # Sort por priority (ascendente → más urgente primero), luego gap_severity desc
        self.gaps.sort(key=lambda g: (g.priority, -g.gap_severity))

    # ── Run Pipeline ──────────────────────────────────────────────────────

    def run(self) -> dict[str, Any]:
        """Ejecutar el pipeline completo."""
        t0 = time.monotonic()

        self.normalize_skills()
        self.compute_gaps()
        self.prioritize()

        elapsed_ms = (time.monotonic() - t0) * 1000
        return self._build_output(elapsed_ms)

    def _build_output(self, elapsed_ms: float) -> dict[str, Any]:
        """Construir JSON de salida."""
        now_iso = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z")
        # Fix timezone format: +0000 → +00:00
        now_iso = now_iso[:-2] + ":" + now_iso[-2:]

        output: dict[str, Any] = {
            "student_id": self.student_id,
            "career_goal": self.career_goal,
            "generated_at": now_iso,
            "gaps": [
                {
                    "skill": g.skill,
                    "estimated_level": g.estimated_level,
                    "target_level": g.target_level,
                    "gap_severity": g.gap_severity,
                    "priority": g.priority,
                    "dunning_kruger_flag": g.dunning_kruger_flag,
                    "rationale": g.rationale,
                }
                for g in self.gaps
            ],
            "metadata": {
                "total_gaps": len(self.gaps),
                "skills_assessed": len(self.assessments),
                "unmapped_skills": self.unmapped_skills,
                "pipeline_ms": round(elapsed_ms, 2),
                "detector_version": "1.0.0",
            },
        }
        return output


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════


def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Gap Detector — Self-assessment → gap analysis"
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="YAML file con self-assessment (si no se especifica, lee stdin)",
    )
    parser.add_argument(
        "--json-output", action="store_true", help="Emitir JSON en lugar de YAML"
    )
    parser.add_argument(
        "--validate-only", action="store_true",
        help="Solo validar schema, no ejecutar pipeline"
    )
    args = parser.parse_args()

    try:
        if args.input_file:
            with open(args.input_file) as f:
                raw = yaml.safe_load(f)
        else:
            raw = yaml.safe_load(sys.stdin)

        if raw is None:
            print(json.dumps({"error": "Input vacío o YAML inválido"}))
            sys.exit(1)

        detector = GapDetector(raw)

        if args.validate_only:
            print(json.dumps({
                "valid": True,
                "student_id": detector.student_id,
                "career_goal": detector.career_goal,
                "skills_count": len(detector.assessments),
            }))
            return

        result = detector.run()

        if args.json_output:
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print(json.dumps(result, indent=2, ensure_ascii=False))

        # Exit code: 1 si hay gaps críticos
        critical = any(g.gap_severity >= 3 for g in detector.gaps)
        if critical:
            sys.exit(1)

    except ValueError as e:
        print(json.dumps({"error": str(e), "valid": False}))
        sys.exit(2)
    except yaml.YAMLError as e:
        print(json.dumps({"error": f"YAML inválido: {e}", "valid": False}))
        sys.exit(2)


if __name__ == "__main__":
    main()
