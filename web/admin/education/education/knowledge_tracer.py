#!/usr/bin/env python3
"""
OVAV Knowledge Tracer — knowledge_tracer.py
==============================================
Bayesian Knowledge Tracing (BKT) engine for tracking student skill mastery
probabilities over time based on observed correct/incorrect answers.

Model: P(L₀), P(T), P(G), P(S) per skill.
- P(L₀): initial probability student knows the skill
- P(T):  probability of transition (learning) between opportunities
- P(G):  probability of guessing correctly when skill is unknown
- P(S):  probability of slipping (incorrect when known)

On each observation (correct/incorrect), BKT updates P(L) using Bayes:
    P(L|correct)   = P(L)*(1-P(S)) / [P(L)*(1-P(S)) + (1-P(L))*P(G)]
    P(L|incorrect)  = P(L)*P(S)     / [P(L)*P(S)     + (1-P(L))*(1-P(G))]
    P(L_next)       = P(L|obs) + (1-P(L|obs)) * P(T)

Output: mastery estimates per skill for curriculum engine integration.

Spec canónica: OVAV Phase 2 — Bayesian Knowledge Tracing
Dependencias: gap_detector.py (skill matrix), curriculum_engine.py (module structure)
Autor: Valeria + Sandra + Beatriz (Education squad)
Implementación: Valeria (Lead, Education & Career Development)
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
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

# Default BKT parameters per skill (from education literature, Corbett & Anderson 1995)
DEFAULT_P_L0: float = 0.15   # Low initial knowledge (conservative)
DEFAULT_P_T: float = 0.10    # 10% chance of learning per opportunity
DEFAULT_P_G: float = 0.20    # 20% chance of guessing correctly
DEFAULT_P_S: float = 0.10    # 10% chance of slipping

# Mastery threshold: P(L) >= MASTERY_THRESHOLD means skill is mastered
MASTERY_THRESHOLD: float = 0.85

# Minimum observations before mastery can be declared
MIN_OBSERVATIONS: int = 3


# ═══════════════════════════════════════════════════════════════════════════
# Data Classes
# ═══════════════════════════════════════════════════════════════════════════

@dataclass
class SkillState:
    """BKT state for a single skill."""
    name: str
    p_l0: float = DEFAULT_P_L0
    p_t: float = DEFAULT_P_T
    p_g: float = DEFAULT_P_G
    p_s: float = DEFAULT_P_S
    # Runtime state
    p_l: float = DEFAULT_P_L0       # Current P(L) — mastery probability
    observations: int = 0
    correct_count: int = 0
    trace: list[dict[str, Any]] = field(default_factory=list)
    mastered: bool = False
    mastery_observation: int = -1   # Observation index where mastery reached

    def reset(self) -> None:
        """Reset runtime state for a new session."""
        self.p_l = self.p_l0
        self.observations = 0
        self.correct_count = 0
        self.trace = []
        self.mastered = False
        self.mastery_observation = -1


@dataclass
class Interaction:
    """One student interaction with a skill exercise."""
    skill: str
    correct: bool
    context: str = ""           # module_id or exercise_id
    timestamp: str = ""
    response_time_ms: float = 0.0
    confidence_self: float = 0.5  # Student's self-reported confidence


# ═══════════════════════════════════════════════════════════════════════════
# Core Engine
# ═══════════════════════════════════════════════════════════════════════════

class KnowledgeTracer:
    """
    Bayesian Knowledge Tracing engine.

    Tracks student mastery probabilities for a set of skills over time,
    updating P(L) based on observed correct/incorrect answers.
    """

    def __init__(
        self,
        skills: list[str],
        p_l0_map: dict[str, float] | None = None,
        p_t_map: dict[str, float] | None = None,
        p_g_map: dict[str, float] | None = None,
        p_s_map: dict[str, float] | None = None,
        initial_mastery: dict[str, float] | None = None,
    ):
        """
        Initialize tracer with skill names and optional BKT parameters.

        Args:
            skills: List of skill canonical names to track.
            p_l0_map: Per-skill initial mastery probabilities.
            p_t_map: Per-skill learning transition rates.
            p_g_map: Per-skill guess probabilities.
            p_s_map: Per-skill slip probabilities.
            initial_mastery: Pre-seeded mastery estimates (from gap_detector).
        """
        if not skills:
            raise ValueError("At least one skill is required.")

        self.skill_names = list(skills)
        self.states: dict[str, SkillState] = {}

        for name in skills:
            p_l0 = (p_l0_map or {}).get(name, DEFAULT_P_L0)
            p_t = (p_t_map or {}).get(name, DEFAULT_P_T)
            p_g = (p_g_map or {}).get(name, DEFAULT_P_G)
            p_s = (p_s_map or {}).get(name, DEFAULT_P_S)

            # Validate BKT parameters
            for param_name, param_val in [("P_L0", p_l0), ("P_T", p_t),
                                           ("P_G", p_g), ("P_S", p_s)]:
                if not (0.0 <= param_val <= 1.0):
                    raise ValueError(
                        f"Skill '{name}': {param_name}={param_val:.3f} fuera de [0, 1]."
                    )

            state = SkillState(
                name=name, p_l0=p_l0, p_t=p_t, p_g=p_g, p_s=p_s, p_l=p_l0
            )

            # Seed with initial mastery if provided (overrides p_l0)
            if initial_mastery and name in initial_mastery:
                im = initial_mastery[name]
                if 0.0 <= im <= 1.0:
                    state.p_l = im
                    state.p_l0 = im

            self.states[name] = state

        self.interactions: list[Interaction] = []
        self.student_id: str = ""

    # ── Core BKT Update ───────────────────────────────────────────────────

    def observe(self, interaction: Interaction) -> dict[str, Any]:
        """
        Record an interaction and update P(L) for the skill using BKT.

        Returns the updated state for the affected skill.
        """
        if interaction.skill not in self.states:
            raise ValueError(
                f"Skill '{interaction.skill}' no registrada. "
                f"Skills disponibles: {sorted(self.states.keys())}"
            )

        state = self.states[interaction.skill]
        prev_p_l = state.p_l

        # ── Bayesian update ──
        if interaction.correct:
            # P(L|correct)
            numerator = state.p_l * (1.0 - state.p_s)
            denominator = numerator + (1.0 - state.p_l) * state.p_g
            if denominator > 1e-10:
                p_l_given_obs = numerator / denominator
            else:
                p_l_given_obs = state.p_l  # degenerate, keep prior
        else:
            # P(L|incorrect)
            numerator = state.p_l * state.p_s
            denominator = numerator + (1.0 - state.p_l) * (1.0 - state.p_g)
            if denominator > 1e-10:
                p_l_given_obs = numerator / denominator
            else:
                p_l_given_obs = state.p_l  # degenerate, keep prior

        # ── Learning transition ──
        p_l_next = p_l_given_obs + (1.0 - p_l_given_obs) * state.p_t
        p_l_next = min(1.0, max(0.0, p_l_next))  # clamp

        # ── Update state ──
        state.p_l = p_l_next
        state.observations += 1
        if interaction.correct:
            state.correct_count += 1

        # ── Mastery check ──
        if not state.mastered and state.p_l >= MASTERY_THRESHOLD and state.observations >= MIN_OBSERVATIONS:
            state.mastered = True
            state.mastery_observation = state.observations

        # ── Record trace ──
        trace_entry = {
            "observation": state.observations,
            "correct": interaction.correct,
            "context": interaction.context,
            "p_l_before": round(prev_p_l, 6),
            "p_l_after": round(state.p_l, 6),
            "delta": round(state.p_l - prev_p_l, 6),
            "mastered": state.mastered,
        }
        state.trace.append(trace_entry)
        self.interactions.append(interaction)

        return {
            "skill": interaction.skill,
            "p_l_before": round(prev_p_l, 6),
            "p_l_after": round(state.p_l, 6),
            "observations": state.observations,
            "mastered": state.mastered,
        }

    # ── Batch Operations ─────────────────────────────────────────────────

    def observe_batch(self, interactions: list[Interaction]) -> list[dict[str, Any]]:
        """Record multiple interactions and return results for each."""
        return [self.observe(i) for i in interactions]

    def get_mastery(self, skill: str) -> float:
        """Get current P(L) mastery probability for a skill."""
        if skill not in self.states:
            raise ValueError(f"Skill '{skill}' no registrada.")
        return self.states[skill].p_l

    def get_all_masteries(self) -> dict[str, float]:
        """Get current mastery probabilities for all skills."""
        return {name: state.p_l for name, state in self.states.items()}

    def is_mastered(self, skill: str) -> bool:
        """Check if a skill has reached mastery threshold."""
        if skill not in self.states:
            raise ValueError(f"Skill '{skill}' no registrada.")
        return self.states[skill].mastered

    def get_mastered_skills(self) -> list[str]:
        """Return list of skills that have reached mastery."""
        return [name for name, state in self.states.items() if state.mastered]

    def get_unmastered_skills(self) -> list[str]:
        """Return list of skills not yet mastered, sorted by P(L) ascending."""
        unmastered = [(name, state.p_l) for name, state in self.states.items()
                       if not state.mastered]
        unmastered.sort(key=lambda x: x[1])
        return [name for name, _ in unmastered]

    def get_skill_trace(self, skill: str) -> list[dict[str, Any]]:
        """Return the full trace for a skill."""
        if skill not in self.states:
            raise ValueError(f"Skill '{skill}' no registrada.")
        return self.states[skill].trace

    # ── Level Estimation ─────────────────────────────────────────────────

    def estimate_level(self, skill: str) -> str:
        """
        Map P(L) mastery probability to a discrete level.

        Thresholds:
            P(L) < 0.20 → beginner
            P(L) < 0.40 → novice
            P(L) < 0.60 → intermediate
            P(L) < 0.80 → advanced
            P(L) >= 0.80 → expert
        """
        p = self.get_mastery(skill)
        if p < 0.20:
            return "beginner"
        elif p < 0.40:
            return "novice"
        elif p < 0.60:
            return "intermediate"
        elif p < 0.80:
            return "advanced"
        else:
            return "expert"

    def estimate_all_levels(self) -> dict[str, str]:
        """Estimate levels for all skills."""
        return {name: self.estimate_level(name) for name in self.skill_names}

    # ── Next-Item Mastery Prediction ─────────────────────────────────────

    def predict_next_correct(self, skill: str, n_items: int = 1) -> float:
        """
        Predict probability of answering correctly on the next n_items.
        Uses current P(L) with BKT parameters.

        P(correct) = P(L)*(1-P(S)) + (1-P(L))*P(G)
        """
        if skill not in self.states:
            raise ValueError(f"Skill '{skill}' no registrada.")
        state = self.states[skill]
        single_prob = state.p_l * (1.0 - state.p_s) + (1.0 - state.p_l) * state.p_g
        # For n_items, assume independent and apply learning transition
        p = state.p_l
        prob_correct = 0.0
        for _ in range(n_items):
            prob_correct += p * (1.0 - state.p_s) + (1.0 - p) * state.p_g
            # Apply expected learning between items
            p = p + (1.0 - p) * state.p_t
        return prob_correct / n_items

    # ── Persistence ──────────────────────────────────────────────────────

    def to_dict(self) -> dict[str, Any]:
        """Serialize complete tracer state."""
        skills_data = {}
        for name, state in self.states.items():
            skills_data[name] = {
                "p_l0": state.p_l0,
                "p_t": state.p_t,
                "p_g": state.p_g,
                "p_s": state.p_s,
                "p_l_current": state.p_l,
                "observations": state.observations,
                "correct_count": state.correct_count,
                "mastered": state.mastered,
                "mastery_observation": state.mastery_observation,
                "trace": state.trace,
            }

        return {
            "student_id": self.student_id,
            "skills": skills_data,
            "total_interactions": len(self.interactions),
            "mastered_count": len(self.get_mastered_skills()),
            "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S%z"),
        }

    def export_for_transfer(self) -> dict[str, Any]:
        """
        Export data for transfer_validator.py.
        Returns mastery estimates and BKT parameters.
        """
        masteries = self.get_all_masteries()
        levels = self.estimate_all_levels()
        params = {}
        for name, state in self.states.items():
            params[name] = {
                "p_l": state.p_l,
                "p_t": state.p_t,
                "p_s": state.p_s,
                "p_g": state.p_g,
                "observations": state.observations,
                "mastered": state.mastered,
            }
        return {
            "masteries": masteries,
            "levels": levels,
            "params": params,
            "mastered_skills": self.get_mastered_skills(),
            "unmastered_skills": self.get_unmastered_skills(),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> KnowledgeTracer:
        """Deserialize tracer from a dict."""
        skills = list(data["skills"].keys())
        tracer = cls(skills)
        tracer.student_id = data.get("student_id", "")
        for name, sd in data["skills"].items():
            if name in tracer.states:
                state = tracer.states[name]
                state.p_l0 = sd.get("p_l0", DEFAULT_P_L0)
                state.p_t = sd.get("p_t", DEFAULT_P_T)
                state.p_g = sd.get("p_g", DEFAULT_P_G)
                state.p_s = sd.get("p_s", DEFAULT_P_S)
                state.p_l = sd.get("p_l_current", state.p_l0)
                state.observations = sd.get("observations", 0)
                state.correct_count = sd.get("correct_count", 0)
                state.mastered = sd.get("mastered", False)
                state.mastery_observation = sd.get("mastery_observation", -1)
                state.trace = sd.get("trace", [])
        return tracer

    # ── Integration: Seed from Gap Detector ──────────────────────────────

    def seed_from_gap_detector(self, gaps_result: dict[str, Any]) -> None:
        """
        Initialize mastery priors from gap_detector output.
        Converts estimated levels to P(L) seeds.
        """
        level_to_p_l = {
            "beginner": 0.10,
            "novice": 0.25,
            "intermediate": 0.50,
            "advanced": 0.75,
            "expert": 0.92,
        }
        gap_list = gaps_result.get("gaps", [])
        for gap in gap_list:
            skill = gap["skill"]
            estimated = gap.get("estimated_level", "beginner")
            if skill in self.states:
                seed = level_to_p_l.get(estimated, 0.15)
                self.states[skill].p_l0 = seed
                self.states[skill].p_l = seed

    # ── Summary ──────────────────────────────────────────────────────────

    def summary(self) -> dict[str, Any]:
        """Return a compact summary for integration with other tools."""
        return {
            "student_id": self.student_id,
            "total_skills": len(self.skill_names),
            "mastered": self.get_mastered_skills(),
            "unmastered": self.get_unmastered_skills(),
            "masteries": self.get_all_masteries(),
            "levels": self.estimate_all_levels(),
            "readiness": self._compute_readiness(),
        }

    def _compute_readiness(self) -> float:
        """Overall student readiness (fraction of skills mastered or near-mastery)."""
        if not self.states:
            return 0.0
        scores = []
        for state in self.states.values():
            if state.mastered:
                scores.append(1.0)
            else:
                scores.append(state.p_l / MASTERY_THRESHOLD)
        return _stats.mean(scores) if scores else 0.0


# ═══════════════════════════════════════════════════════════════════════════
# Factory: Load from gap_detector output
# ═══════════════════════════════════════════════════════════════════════════

def create_tracer_from_gaps(
    gaps_result: dict[str, Any],
    p_t_map: dict[str, float] | None = None,
    p_g_map: dict[str, float] | None = None,
    p_s_map: dict[str, float] | None = None,
) -> KnowledgeTracer:
    """
    Create a KnowledgeTracer seeded from gap_detector output.

    Args:
        gaps_result: Output from gap_detector.py
        p_t_map, p_g_map, p_s_map: Optional BKT parameter overrides per skill.
    """
    gap_list = gaps_result.get("gaps", [])
    skills = [g["skill"] for g in gap_list]

    level_to_p_l = {
        "beginner": 0.10,
        "novice": 0.25,
        "intermediate": 0.50,
        "advanced": 0.75,
        "expert": 0.92,
    }
    p_l0_map: dict[str, float] = {}
    for gap in gap_list:
        estimated = gap.get("estimated_level", "beginner")
        p_l0_map[gap["skill"]] = level_to_p_l.get(estimated, 0.15)

    tracer = KnowledgeTracer(
        skills=skills,
        p_l0_map=p_l0_map,
        p_t_map=p_t_map,
        p_g_map=p_g_map,
        p_s_map=p_s_map,
    )
    tracer.student_id = gaps_result.get("student_id", "")
    return tracer


# ═══════════════════════════════════════════════════════════════════════════
# CLI
# ═══════════════════════════════════════════════════════════════════════════

def main() -> None:
    parser = argparse.ArgumentParser(
        description="OVAV Knowledge Tracer — BKT mastery tracking"
    )
    parser.add_argument(
        "input_file",
        nargs="?",
        help="JSON file con skills y opcionalmente interacciones",
    )
    parser.add_argument(
        "--interactions",
        help="JSON file con interacciones (skill, correct, context)",
    )
    parser.add_argument(
        "--export", action="store_true",
        help="Exportar para transfer_validator.py",
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
            data = json.load(sys.stdin) if not sys.stdin.isatty() else {}

        # Determine if input is gap_detector output or serialized tracer
        if "skills" in data and isinstance(data["skills"], dict):
            tracer = KnowledgeTracer.from_dict(data)
        elif "gaps" in data:
            tracer = create_tracer_from_gaps(data)
        elif "skill_names" in data or "skills" in data:
            skill_list = data.get("skill_names") or list(data.get("skills", {}).keys()) or []
            if not skill_list:
                raise ValueError("Input debe contener 'skill_names', 'skills', o 'gaps'.")
            tracer = KnowledgeTracer(skills=skill_list)
        else:
            raise ValueError("Formato de input no reconocido. Use gap_detector output o skill list.")

        # Load interactions if provided
        if args.interactions:
            with open(args.interactions) as f:
                interactions_data = json.load(f)
            interactions = [
                Interaction(
                    skill=i["skill"],
                    correct=i["correct"],
                    context=i.get("context", ""),
                    timestamp=i.get("timestamp", ""),
                    response_time_ms=i.get("response_time_ms", 0.0),
                    confidence_self=i.get("confidence_self", 0.5),
                )
                for i in interactions_data
            ]
            tracer.observe_batch(interactions)

        if args.export:
            print(json.dumps(tracer.export_for_transfer(), indent=2, ensure_ascii=False))
        elif args.summary:
            print(json.dumps(tracer.summary(), indent=2, ensure_ascii=False))
        else:
            print(json.dumps(tracer.to_dict(), indent=2, ensure_ascii=False))

    except ValueError as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)
    except (json.JSONDecodeError, FileNotFoundError) as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(2)


if __name__ == "__main__":
    main()
