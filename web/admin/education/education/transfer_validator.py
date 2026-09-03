#!/usr/bin/env python3
"""
OVAV Transfer Validator — transfer_validator.py
==================================================
Predicts probability of success on the next learning item using mastery
estimates from knowledge_tracer. Validates skill transfer between related
concepts and integrates with curriculum_engine for adaptive path adjustment.

Model: Transfer-Weighted Next-Item Correctness (TWNIC)
    P(correct_next | skill_i) = Σ w_ij * P(L_j) * (1-P(S_j))
    where w_ij is the transfer weight from skill j to skill i,
    normalized: Σ w_ij = 1

Also computes:
    - Transfer gap: how much a student's unmastered skills block transfer
    - Readiness score: weighted probability of mastering next module
    - Path adjustment recommendations for curriculum_engine

Spec canónica: OVAV Phase 2 — Transfer/Next-Item Correctness
Dependencias: knowledge_tracer.py (mastery estimates)
              curriculum_engine.py (module DAG for path adjustment)
Autor: Valeria + Beatriz + Alicia (Education squad)
Implementación: Valeria (Lead, Education & Career Development)
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

import statistics as _stats

# ═══════════════════════════════════════════════════════════════════════════
# Constants
# ═══════════════════════════════════════════════════════════════════════════

LEVEL_ORDER: dict[str, int] = {
    "beginner": 0,
    "novice": 1,
    "intermediate": 2,
    "advanced": 3,
    "expert": 4,
}

LEVEL_NAMES: list[str] = ["beginner", "novice", "intermediate", "advanced", "expert"]

# Transfer matrix: how much skill A transfers to skill B (0.0–1.0)
# Built from learning science literature and domain expertise
# Key: source_skill → {target_skill: transfer_weight}
TRANSFER_MATRIX: dict[str, dict[str, float]] = {
    "python": {
        "go": 0.60,
        "javascript": 0.50,
        "sql": 0.20,
        "machine_learning": 0.40,
        "data_visualization": 0.30,
        "data_engineering": 0.35,
        "backend_development": 0.45,
        "devops_and_cloud": 0.20,
        "shell_scripting": 0.30,
        "statistics": 0.15,
    },
    "go": {
        "python": 0.50,
        "backend_development": 0.40,
        "devops_and_cloud": 0.45,
        "shell_scripting": 0.35,
        "system_design": 0.30,
    },
    "javascript": {
        "python": 0.45,
        "frontend_development": 0.70,
        "backend_development": 0.35,
    },
    "sql": {
        "data_engineering": 0.50,
        "backend_development": 0.35,
        "machine_learning": 0.15,
    },
    "statistics": {
        "machine_learning": 0.65,
        "data_visualization": 0.30,
        "data_engineering": 0.25,
        "problem_solving": 0.20,
    },
    "machine_learning": {
        "statistics": 0.35,
        "deep_learning": 0.60,
        "data_engineering": 0.40,
        "data_visualization": 0.25,
    },
    "backend_development": {
        "system_design": 0.50,
        "devops_and_cloud": 0.35,
        "cybersecurity": 0.25,
    },
    "frontend_development": {
        "backend_development": 0.20,
        "data_visualization": 0.30,
    },
    "system_design": {
        "backend_development": 0.45,
        "devops_and_cloud": 0.40,
    },
    "git": {
        "code_review": 0.40,
        "devops_and_cloud": 0.30,
        "shell_scripting": 0.10,
    },
    "problem_solving": {
        "machine_learning": 0.30,
        "system_design": 0.35,
        "code_review": 0.25,
        "technical_writing": 0.20,
        "project_management": 0.30,
    },
    "technical_writing": {
        "code_review": 0.30,
        "project_management": 0.25,
        "problem_solving": 0.15,
    },
    "code_review": {
        "technical_writing": 0.20,
        "backend_development": 0.15,
    },
    "shell_scripting": {
        "devops_and_cloud": 0.40,
        "python": 0.15,
        "go": 0.15,
    },
    "devops_and_cloud": {
        "backend_development": 0.30,
        "system_design": 0.35,
        "cybersecurity": 0.25,
        "shell_scripting": 0.30,
    },
    "cybersecurity": {
        "backend_development": 0.25,
        "devops_and_cloud": 0.30,
        "system_design": 0.20,
    },
    "data_engineering": {
        "sql": 0.40,
        "python": 0.30,
        "machine_learning": 0.35,
        "devops_and_cloud": 0.25,
    },
    "data_visualization": {
        "machine_learning": 0.20,
        "statistics": 0.25,
    },
    "deep_learning": {
        "machine_learning": 0.60,
        "statistics": 0.35,
    },
    "project_management": {
        "technical_writing": 0.20,
        "code_review": 0.15,
        "problem_solving": 0.15,
    },
}

# Default self-transfer (a skill always transfers 1.0 to itself)
SELF_TRANSFER: float = 1.0

# Default transfer from skills not in matrix
DEFAULT_TRANSFER: float = 0.10

# Low transfer warning threshold
LOW_TRANSFER_THRESHOLD: float = 0.30


# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════

@dataclass
class TransferPrediction:
    """Prediction result for a single skill."""
    skill: str
    next_item_correctness: float      # P(correct on next item for this skill)
    confidence: float                  # Model confidence in prediction
    source_masteries: dict[str, float] # Which skills contributed and how much
    transfer_gap: float                # Gap: 0 = full transfer, 1 = no transfer
    readiness: float                   # 0–1 readiness for next module
    warning: str = ""                  # Optional warning (low transfer,etc.)


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════

class TransferValidator:
    """
    Validates skill transfer and predicts next-item correctness.

    Uses mastery estimates from knowledge_tracer and a transfer matrix
    to predict how likely a student is to succeed on the next item for
    each skill, accounting for transfer from related mastered skills.
    """

    def __init__(
        self,
        masteries: dict[str, float],
        bkt_params: dict[str, dict[str, float]] | None = None,
        transfer_matrix: dict[str, dict[str, float]] | None = None,
    ):
        """
        Args:
            masteries: {skill_name: P(L) mastery probability}
            bkt_params: {skill_name: {p_l, p_t, p_s, p_g, ...}}
            transfer_matrix: Override default transfer matrix.
        """
        if not masteries:
            raise ValueError("At least one mastery estimate is required.")

        self.masteries = dict(masteries)
        self.bkt_params = bkt_params or {}
        self.transfer_matrix = transfer_matrix or TRANSFER_MATRIX

        # Validate masteries
        for skill, p_l in self.masteries.items():
            if not (0.0 <= p_l <= 1.0):
                raise ValueError(
                    f"Mastery for '{skill}'={p_l:.3f} fuera de [0, 1]."
                )

    # ── Transfer Weight Calculation ───────────────────────────────────────

    def get_transfer_weight(self, source_skill: str, target_skill: str) -> float:
        """Get transfer weight from source_skill to target_skill."""
        if source_skill == target_skill:
            return SELF_TRANSFER
        if source_skill in self.transfer_matrix:
            return self.transfer_matrix[source_skill].get(target_skill, DEFAULT_TRANSFER)
        return DEFAULT_TRANSFER

    def compute_transfer_vector(self, target_skill: str) -> dict[str, float]:
        """
        Compute normalized transfer weights from all skills to target_skill.

        Returns dict of {source_skill: normalized_weight}
        """
        raw_weights: dict[str, float] = {}
        for source in self.masteries:
            raw_weights[source] = self.get_transfer_weight(source, target_skill)

        # Normalize
        total = sum(raw_weights.values())
        if total > 0:
            return {k: v / total for k, v in raw_weights.items()}
        # Fallback: uniform
        n = len(raw_weights)
        return {k: 1.0 / n for k in raw_weights}

    # ── Next-Item Correctness Prediction ─────────────────────────────────

    def predict_next_correctness(self, target_skill: str) -> TransferPrediction:
        """
        Predict P(correct on next item) for target_skill.

        Uses transfer-weighted mastery: Σ w_i * P(L_i) * (1-P(S_i))
        where w_i is normalized transfer weight from skill i to target.
        """
        transfer_vec = self.compute_transfer_vector(target_skill)

        weighted_mastery = 0.0
        total_contrib = 0.0
        source_contribs: dict[str, float] = {}

        for source, weight in transfer_vec.items():
            p_l = self.masteries.get(source, 0.0)
            # Get slip probability from BKT params or use default
            p_s = 0.10  # default
            if self.bkt_params and source in self.bkt_params:
                p_s = self.bkt_params[source].get("p_s", 0.10)

            contrib = weight * p_l * (1.0 - p_s)
            weighted_mastery += contrib
            total_contrib += weight
            source_contribs[source] = round(contrib, 6)

        # Add guessing baseline
        guessing_baseline = 0.0
        for source, weight in transfer_vec.items():
            p_l = self.masteries.get(source, 0.0)
            p_g = 0.20  # default
            if self.bkt_params and source in self.bkt_params:
                p_g = self.bkt_params[source].get("p_g", 0.20)
            guessing_baseline += weight * (1.0 - p_l) * p_g

        next_correct = weighted_mastery + guessing_baseline
        next_correct = min(1.0, max(0.0, next_correct))

        # Transfer gap: 1 - weighted average of related masteries
        if total_contrib > 0:
            related_mastery_avg = weighted_mastery / total_contrib if total_contrib > 0 else 0
        else:
            related_mastery_avg = 0.0
        transfer_gap = 1.0 - related_mastery_avg

        # Readiness: if mastery is high via any transfer, student is ready
        self_mastery = self.masteries.get(target_skill, 0.0)
        readiness = 0.5 * self_mastery + 0.5 * next_correct

        # Confidence: higher when more observations available
        obs_count = 0
        if self.bkt_params and target_skill in self.bkt_params:
            obs_count = self.bkt_params[target_skill].get("observations", 0)
        confidence = min(1.0, obs_count / 10.0)  # max confidence after 10 obs

        # Warning
        warning = ""
        if transfer_gap > 0.5:
            warning = f"Transfer gap alto ({transfer_gap:.2f}): baja transferencia de habilidades relacionadas."
        elif self_mastery < 0.3 and next_correct < 0.4:
            warning = f"Bajo dominio ({self_mastery:.2f}) y baja predicción ({next_correct:.2f}): considerar refuerzo."

        return TransferPrediction(
            skill=target_skill,
            next_item_correctness=round(next_correct, 4),
            confidence=round(confidence, 4),
            source_masteries=source_contribs,
            transfer_gap=round(transfer_gap, 4),
            readiness=round(readiness, 4),
            warning=warning,
        )

    def predict_all(self) -> dict[str, TransferPrediction]:
        """Predict next-item correctness for all known skills."""
        return {skill: self.predict_next_correctness(skill)
                for skill in self.masteries}

    # ── Batch Prediction for Modules ─────────────────────────────────────

    def predict_module_readiness(
        self, module_skills: list[str]
    ) -> dict[str, Any]:
        """
        Predict readiness for a module composed of multiple skills.

        Returns readiness score and per-skill predictions.
        """
        predictions = {
            skill: self.predict_next_correctness(skill)
            for skill in module_skills
        }

        readiness_scores = [p.readiness for p in predictions.values()]
        avg_readiness = _stats.mean(readiness_scores) if readiness_scores else 0.0
        min_readiness = min(readiness_scores) if readiness_scores else 0.0

        # Module is ready if avg_readiness >= 0.7 and no skill < 0.4
        module_ready = avg_readiness >= 0.70 and min_readiness >= 0.40

        warnings = [p.warning for p in predictions.values() if p.warning]

        return {
            "module_ready": module_ready,
            "avg_readiness": round(avg_readiness, 4),
            "min_readiness": round(min_readiness, 4),
            "skill_predictions": {
                skill: {
                    "next_correct": p.next_item_correctness,
                    "readiness": p.readiness,
                    "transfer_gap": p.transfer_gap,
                }
                for skill, p in predictions.items()
            },
            "warnings": warnings,
        }

    # ── Transfer Path Analysis ───────────────────────────────────────────

    def find_best_transfer_path(
        self, target_skill: str, max_depth: int = 3
    ) -> list[tuple[str, float]]:
        """
        Find the strongest transfer path to a target skill.

        Uses greedy search: at each step, pick the unmastered source skill
        with highest transfer weight * mastery product.
        """
        if target_skill not in self.masteries:
            return []

        path: list[tuple[str, float]] = []
        visited = {target_skill}
        current = target_skill

        for _ in range(max_depth):
            best_source = None
            best_score = 0.0
            for source, p_l in self.masteries.items():
                if source in visited:
                    continue
                weight = self.get_transfer_weight(source, current)
                score = weight * p_l
                if score > best_score:
                    best_score = score
                    best_source = source
            if best_source is None or best_score < 0.01:
                break
            path.append((best_source, round(best_score, 4)))
            visited.add(best_source)
            current = best_source

        return path

    # ── Curriculum Integration ───────────────────────────────────────────

    def recommend_path_adjustments(
        self,
        module_sequence: list[dict[str, Any]],
    ) -> dict[str, Any]:
        """
        Generate path adjustment recommendations for curriculum_engine.

        For each upcoming module, check if prerequisites are truly mastered
        via transfer validation. If not, recommend:
          - delay: move module later in sequence
          - reinforce: add prerequisite reinforcement
          - accelerate: skip if transfer from other skills covers it

        Args:
            module_sequence: List of modules from curriculum_engine,
                             each with 'id', 'skill', 'prerequisites'.
        """
        adjustments = []
        module_readiness_scores: dict[str, float] = {}

        for mod in module_sequence:
            mod_id = mod.get("id", "?")
            skill = mod.get("skill", "")
            prereqs = mod.get("prerequisites", [])

            # Get mastery of this skill
            self_mastery = self.masteries.get(skill, 0.0)

            # Check prerequisite readiness via transfer
            prereq_ready = True
            prereq_scores: dict[str, float] = {}
            for prereq_id in prereqs:
                # Find the skill for this prereq module
                prereq_skill = self._resolve_prereq_skill(prereq_id, module_sequence)
                if prereq_skill:
                    pred = self.predict_next_correctness(prereq_skill)
                    prereq_scores[prereq_id] = pred.readiness
                    if pred.readiness < 0.5:
                        prereq_ready = False

            # Predict this module's readiness
            pred = self.predict_next_correctness(skill)
            readiness = pred.readiness
            module_readiness_scores[mod_id] = readiness

            action = "continue"  # default
            reason = ""

            if not prereq_ready:
                action = "delay"
                reason = f"Prerrequisitos no transferidos: {[k for k, v in prereq_scores.items() if v < 0.5]}"
            elif readiness < 0.4:
                action = "reinforce"
                reason = f"Transfer insuficiente: readiness={readiness:.2f}, transfer_gap={pred.transfer_gap:.2f}"
            elif readiness >= 0.85 and self_mastery < 0.6:
                # Strong transfer from other skills → may accelerate
                action = "accelerate"
                reason = f"Alta transferencia desde otras skills: readiness={readiness:.2f} > dominio directo={self_mastery:.2f}"

            if action != "continue":
                adjustments.append({
                    "module_id": mod_id,
                    "skill": skill,
                    "action": action,
                    "reason": reason,
                    "readiness": round(readiness, 4),
                    "self_mastery": round(self_mastery, 4),
                })

        return {
            "adjustments": adjustments,
            "total_adjustments": len(adjustments),
            "module_readiness": module_readiness_scores,
            "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z"),
        }

    def _resolve_prereq_skill(
        self, module_id: str, module_sequence: list[dict[str, Any]]
    ) -> str | None:
        """Find the skill for a prereq module ID."""
        for mod in module_sequence:
            if mod.get("id") == module_id:
                return mod.get("skill")
        return None

    # ── Summary ──────────────────────────────────────────────────────────

    def summary(self) -> dict[str, Any]:
        """Compact summary for integration."""
        predictions = self.predict_all()
        skills_data = {}
        for skill, pred in predictions.items():
            skills_data[skill] = {
                "next_correctness": pred.next_item_correctness,
                "transfer_gap": pred.transfer_gap,
                "readiness": pred.readiness,
                "warning": pred.warning,
            }

        avg_readiness = _stats.mean([p.readiness for p in predictions.values()]) if predictions else 0.0
        avg_transfer_gap = _stats.mean([p.transfer_gap for p in predictions.values()]) if predictions else 0.0

        return {
            "skills": skills_data,
            "avg_readiness": round(avg_readiness, 4),
            "avg_transfer_gap": round(avg_transfer_gap, 4),
            "high_transfer_skills": [
                s for s, p in predictions.items() if p.transfer_gap < 0.3
            ],
            "low_transfer_skills": [
                s for s, p in predictions.items() if p.transfer_gap > 0.5
            ],
            "warnings": [p.warning for p in predictions.values() if p.warning],
        }


# ═══════════════════════════════════════════════════════════════════════════
# Factory: Load from knowledge_tracer export
# ═══════════════════════════════════════════════════════════════════════════

def create_validator_from_export(export_data: dict[str, Any]) -> TransferValidator:
    """
    Create a TransferValidator from knowledge_tracer.export_for_transfer() output.
    """
    masteries = export_data.get("masteries", {})
    params = export_data.get("params", {})
    return TransferValidator(masteries=masteries, bkt_params=params)


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Transfer Validator — Next-item correctness & transfer analysis"
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="JSON file con mastery data (export de knowledge_tracer o custom)",
    )
    parser.add_argument(
        "--module-sequence",
        help="JSON file con secuencia de módulos de curriculum_engine",
    )
    parser.add_argument(
        "--target-skill",
        help="Skill específica para predicción individual",
    )
    parser.add_argument(
        "--path-adjust", action="store_true",
        help="Generar recomendaciones de ajuste de path para curriculum_engine",
    )
    parser.add_argument(
        "--summary", action="store_true",
        help="Emitir resumen compacto",
    )
    args = parser.parse_args()

    try:
        if args.input_file:
            with open(args.input_file) as f:
                data = json.load(f)
        else:
            data = json.load(sys.stdin)

        # Determine input format
        if "masteries" in data:
            validator = create_validator_from_export(data)
        elif isinstance(data, dict) and all(isinstance(v, (int, float)) for v in data.values()):
            # Raw mastery dict
            validator = TransferValidator(masteries=data)
        else:
            raise ValueError(
                "Formato no reconocido. Use export de knowledge_tracer "
                "o dict de masteries {skill: P(L)}."
            )

        if args.target_skill:
            pred = validator.predict_next_correctness(args.target_skill)
            result = {
                "skill": pred.skill,
                "next_item_correctness": pred.next_item_correctness,
                "transfer_gap": pred.transfer_gap,
                "readiness": pred.readiness,
                "warning": pred.warning,
            }
            print(json.dumps(result, indent=2, ensure_ascii=False))
            return

        if args.path_adjust and args.module_sequence:
            with open(args.module_sequence) as f:
                module_seq = json.load(f) if args.module_sequence.endswith(".json") else yaml_load(args.module_sequence)
            if isinstance(module_seq, dict) and "modules" in module_seq:
                module_seq = module_seq["modules"]
            result = validator.recommend_path_adjustments(module_seq)
        elif args.summary:
            result = validator.summary()
        else:
            predictions = validator.predict_all()
            result = {
                "predictions": {s: {
                    "next_correctness": p.next_item_correctness,
                    "transfer_gap": p.transfer_gap,
                    "readiness": p.readiness,
                    "warning": p.warning,
                } for s, p in predictions.items()},
                "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z"),
            }

        print(json.dumps(result, indent=2, ensure_ascii=False))

    except ValueError as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)
    except (json.JSONDecodeError, FileNotFoundError) as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)


def yaml_load(path: str) -> Any:
    """Load YAML file (deferred import)."""
    try:
        import yaml
        with open(path) as f:
            return yaml.safe_load(f)
    except ImportError:
        with open(path) as f:
            return json.load(f)


if __name__ == "__main__":
    main()
