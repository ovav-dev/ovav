#!/usr/bin/env python3
"""
Tests for knowledge_tracer.py — Education SEG-6 Phase 2
=========================================================
Acceptance criteria:
  - BKT model: P(L₀), P(T), P(G), P(S) per skill
  - Correct answer increases P(L)
  - Incorrect answer decreases P(L)
  - P(L) converges to mastery after multiple correct answers
  - P(L) degrades after incorrect answers (slipping)
  - Multiple skills tracked independently
  - Mastery threshold reached at appropriate observation count
  - Edge cases: empty skill, invalid params, single observation
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.education.knowledge_tracer import (
    KnowledgeTracer,
    Interaction,
    SkillState,
    create_tracer_from_gaps,
    MASTERY_THRESHOLD,
    DEFAULT_P_L0,
    DEFAULT_P_T,
    DEFAULT_P_G,
    DEFAULT_P_S,
)


# ═══════════════════════════════════════════════════════════════════════════
# 1. Initialization & Validation
# ═══════════════════════════════════════════════════════════════════════════

class TestInitialization:
    """Initialize tracer with various configurations."""

    def test_create_with_skill_list(self):
        tracer = KnowledgeTracer(skills=["python", "sql", "statistics"])
        assert len(tracer.skill_names) == 3
        assert all(s in tracer.states for s in ["python", "sql", "statistics"])

    def test_create_with_custom_bkt_params(self):
        tracer = KnowledgeTracer(
            skills=["python"],
            p_l0_map={"python": 0.30},
            p_t_map={"python": 0.15},
            p_g_map={"python": 0.10},
            p_s_map={"python": 0.05},
        )
        state = tracer.states["python"]
        assert state.p_l0 == 0.30
        assert state.p_t == 0.15
        assert state.p_g == 0.10
        assert state.p_s == 0.05

    def test_create_with_initial_mastery(self):
        tracer = KnowledgeTracer(
            skills=["python", "sql"],
            initial_mastery={"python": 0.70, "sql": 0.45},
        )
        assert tracer.states["python"].p_l == 0.70
        assert tracer.states["sql"].p_l == 0.45

    def test_empty_skills_raises(self):
        with pytest.raises(ValueError, match="At least one skill"):
            KnowledgeTracer(skills=[])

    def test_invalid_bkt_param_raises(self):
        with pytest.raises(ValueError, match="fuera de"):
            KnowledgeTracer(skills=["python"], p_l0_map={"python": 1.5})

    def test_default_values_sane(self):
        """Default BKT parameters must be in [0,1]."""
        tracer = KnowledgeTracer(skills=["test"])
        state = tracer.states["test"]
        assert 0.0 <= state.p_l0 <= 1.0
        assert 0.0 <= state.p_t <= 1.0
        assert 0.0 <= state.p_g <= 1.0
        assert 0.0 <= state.p_s <= 1.0


# ═══════════════════════════════════════════════════════════════════════════
# 2. Core BKT Update Mechanism
# ═══════════════════════════════════════════════════════════════════════════

class TestBKTUpdates:
    """BKT updates correctly modify P(L)."""

    def test_correct_increases_mastery(self):
        """A correct answer should increase P(L)."""
        tracer = KnowledgeTracer(skills=["python"])
        initial = tracer.get_mastery("python")
        tracer.observe(Interaction(skill="python", correct=True))
        after = tracer.get_mastery("python")
        assert after > initial, f"Expected {after} > {initial}"

    def test_incorrect_decreases_mastery(self):
        """An incorrect answer should decrease P(L)."""
        tracer = KnowledgeTracer(skills=["python"],
                                  initial_mastery={"python": 0.70})
        initial = tracer.get_mastery("python")
        tracer.observe(Interaction(skill="python", correct=False))
        after = tracer.get_mastery("python")
        assert after < initial, f"Expected {after} < {initial}"

    def test_converges_to_mastery_with_repeated_correct(self):
        """Multiple correct answers should push P(L) above MASTERY_THRESHOLD."""
        tracer = KnowledgeTracer(skills=["python"])
        for i in range(10):
            tracer.observe(Interaction(skill="python", correct=True))
        assert tracer.get_mastery("python") >= MASTERY_THRESHOLD
        assert tracer.is_mastered("python")

    def test_mastery_declared_at_appropriate_time(self):
        """Mastery should be declared when threshold AND min observations met."""
        tracer = KnowledgeTracer(skills=["python"])
        # 2 correct answers: not enough observations
        for _ in range(2):
            tracer.observe(Interaction(skill="python", correct=True))
        # With default params, 2 observations rarely hits 0.85
        # Force higher initial so we test the MIN_OBSERVATIONS gate
        tracer2 = KnowledgeTracer(skills=["python"],
                                   initial_mastery={"python": 0.80},
                                   p_t_map={"python": 0.50})
        tracer2.observe(Interaction(skill="python", correct=True))
        # 1 observation < 3 → not mastered
        assert not tracer2.is_mastered("python")
        tracer2.observe(Interaction(skill="python", correct=True))
        tracer2.observe(Interaction(skill="python", correct=True))
        # 3 observations, high mastery → mastered
        assert tracer2.is_mastered("python")

    def test_incorrect_pulls_below_mastery(self):
        """Slipping: incorrect answer on mastered skill reduces P(L)."""
        tracer = KnowledgeTracer(skills=["python"],
                                  initial_mastery={"python": 0.88})
        # Need 3 observations for mastery to be declared
        for _ in range(3):
            tracer.observe(Interaction(skill="python", correct=True))
        assert tracer.is_mastered("python")
        prev = tracer.get_mastery("python")
        tracer.observe(Interaction(skill="python", correct=False))
        after = tracer.get_mastery("python")
        assert after < prev
        # May or may not lose mastery depending on params

    def test_independent_skill_tracking(self):
        """Skills are tracked independently."""
        tracer = KnowledgeTracer(skills=["python", "sql"])
        tracer.observe(Interaction(skill="python", correct=True))
        tracer.observe(Interaction(skill="python", correct=True))
        tracer.observe(Interaction(skill="sql", correct=False))

        python_p = tracer.get_mastery("python")
        sql_p = tracer.get_mastery("sql")
        # Python should be higher (2 correct vs 1 incorrect starting from same prior)
        assert python_p > sql_p, f"Python={python_p:.4f}, SQL={sql_p:.4f}"


# ═══════════════════════════════════════════════════════════════════════════
# 3. Guessing & Slipping Parameters
# ═══════════════════════════════════════════════════════════════════════════

class TestGuessSlip:
    """P(G) and P(S) influence update magnitude correctly."""

    def test_high_guess_reduces_correct_impact(self):
        """High P(G) means correct answers are less informative."""
        tracer_low_g = KnowledgeTracer(skills=["s"], p_g_map={"s": 0.05})
        tracer_high_g = KnowledgeTracer(skills=["s"], p_g_map={"s": 0.50})

        # Same initial mastery
        assert abs(tracer_low_g.get_mastery("s") - tracer_high_g.get_mastery("s")) < 1e-10

        tracer_low_g.observe(Interaction(skill="s", correct=True))
        tracer_high_g.observe(Interaction(skill="s", correct=True))

        delta_low = tracer_low_g.get_mastery("s") - DEFAULT_P_L0
        delta_high = tracer_high_g.get_mastery("s") - DEFAULT_P_L0
        assert delta_low > delta_high, \
            f"Low-G delta={delta_low:.4f}, High-G delta={delta_high:.4f}"

    def test_high_slip_reduces_correct_impact(self):
        """High P(S): slipping reduces confidence even from correct answers."""
        tracer_low_s = KnowledgeTracer(skills=["s"], p_s_map={"s": 0.01})
        tracer_high_s = KnowledgeTracer(skills=["s"], p_s_map={"s": 0.50})

        tracer_low_s.observe(Interaction(skill="s", correct=True))
        tracer_high_s.observe(Interaction(skill="s", correct=True))

        delta_low = tracer_low_s.get_mastery("s") - DEFAULT_P_L0
        delta_high = tracer_high_s.get_mastery("s") - DEFAULT_P_L0
        assert delta_low > delta_high, \
            f"Low-S delta={delta_low:.4f}, High-S delta={delta_high:.4f}"

    def test_guess_slip_extreme_values(self):
        """Extreme P(G)=0, P(S)=0: correct always increases, incorrect always decreases."""
        tracer = KnowledgeTracer(
            skills=["s"],
            p_g_map={"s": 0.0},
            p_s_map={"s": 0.0},
            initial_mastery={"s": 0.50},
        )
        # Correct with P(G)=0, P(S)=0: P(L|correct) = 1.0 → P(L_next) = 1.0
        tracer.observe(Interaction(skill="s", correct=True))
        assert tracer.get_mastery("s") > 0.95

        # Incorrect with P(G)=0, P(S)=0: P(L|incorrect) = 0 → P(L_next) = P(T)
        tracer2 = KnowledgeTracer(
            skills=["s"],
            p_g_map={"s": 0.0},
            p_s_map={"s": 0.0},
            initial_mastery={"s": 0.50},
            p_t_map={"s": 0.10},
        )
        tracer2.observe(Interaction(skill="s", correct=False))
        assert tracer2.get_mastery("s") < 0.15  # Near P(T) after incorrect


# ═══════════════════════════════════════════════════════════════════════════
# 4. Observation Tracking
# ═══════════════════════════════════════════════════════════════════════════

class TestObservations:
    """Trace and observation counts are maintained correctly."""

    def test_trace_recorded(self):
        tracer = KnowledgeTracer(skills=["python"])
        tracer.observe(Interaction(skill="python", correct=True, context="MOD-PY-01"))
        tracer.observe(Interaction(skill="python", correct=False, context="MOD-PY-01"))

        trace = tracer.get_skill_trace("python")
        assert len(trace) == 2
        assert trace[0]["correct"] is True
        assert trace[0]["context"] == "MOD-PY-01"
        assert trace[1]["correct"] is False
        assert "p_l_before" in trace[0]
        assert "p_l_after" in trace[0]
        assert "delta" in trace[0]

    def test_observation_counts_accurate(self):
        tracer = KnowledgeTracer(skills=["python", "sql"])
        tracer.observe(Interaction(skill="python", correct=True))
        tracer.observe(Interaction(skill="python", correct=False))
        tracer.observe(Interaction(skill="sql", correct=True))

        assert tracer.states["python"].observations == 2
        assert tracer.states["sql"].observations == 1
        assert tracer.states["python"].correct_count == 1
        assert tracer.states["sql"].correct_count == 1

    def test_unobserved_skill_raises(self):
        tracer = KnowledgeTracer(skills=["python"])
        with pytest.raises(ValueError, match="no registrada"):
            tracer.observe(Interaction(skill="sql", correct=True))


# ═══════════════════════════════════════════════════════════════════════════
# 5. Level Estimation
# ═══════════════════════════════════════════════════════════════════════════

class TestLevelEstimation:
    """P(L) → discrete level mapping."""

    def test_level_thresholds(self):
        tracer = KnowledgeTracer(skills=["s"])
        # Test each threshold
        thresholds = [
            (0.05, "beginner"),
            (0.25, "novice"),
            (0.45, "intermediate"),
            (0.65, "advanced"),
            (0.85, "expert"),
        ]
        for p_l, expected_level in thresholds:
            tracer.states["s"].p_l = p_l
            assert tracer.estimate_level("s") == expected_level, \
                f"P(L)={p_l} → expected {expected_level}, got {tracer.estimate_level('s')}"

    def test_estimate_all_levels(self):
        tracer = KnowledgeTracer(
            skills=["python", "sql"],
            initial_mastery={"python": 0.75, "sql": 0.35},
        )
        levels = tracer.estimate_all_levels()
        assert levels["python"] == "advanced"
        assert levels["sql"] == "novice"


# ═══════════════════════════════════════════════════════════════════════════
# 6. Serialization
# ═══════════════════════════════════════════════════════════════════════════

class TestSerialization:
    """Round-trip serialization preserves state."""

    def test_roundtrip(self):
        tracer = KnowledgeTracer(
            skills=["python", "sql"],
            initial_mastery={"python": 0.60},
        )
        tracer.observe(Interaction(skill="python", correct=True, context="test"))
        tracer.student_id = "stu-0001"

        data = tracer.to_dict()
        tracer2 = KnowledgeTracer.from_dict(data)

        assert tracer2.student_id == "stu-0001"
        assert abs(tracer2.get_mastery("python") - tracer.get_mastery("python")) < 1e-6
        assert abs(tracer2.get_mastery("sql") - tracer.get_mastery("sql")) < 1e-6
        assert len(tracer2.get_skill_trace("python")) == 1


# ═══════════════════════════════════════════════════════════════════════════
# 7. Next-Item Prediction
# ═══════════════════════════════════════════════════════════════════════════

class TestNextItemPrediction:
    """Predict next correct probability."""

    def test_prediction_in_range(self):
        tracer = KnowledgeTracer(skills=["python"],
                                  initial_mastery={"python": 0.60})
        prob = tracer.predict_next_correct("python")
        assert 0.0 <= prob <= 1.0

    def test_high_mastery_high_prediction(self):
        tracer = KnowledgeTracer(skills=["python"],
                                  initial_mastery={"python": 0.95},
                                  p_s_map={"python": 0.01})
        prob = tracer.predict_next_correct("python")
        assert prob > 0.85, f"Expected high prediction, got {prob:.4f}"

    def test_low_mastery_still_some_chance(self):
        """Even with low mastery, P(G) gives non-zero chance."""
        tracer = KnowledgeTracer(skills=["python"],
                                  initial_mastery={"python": 0.01},
                                  p_g_map={"python": 0.30})
        prob = tracer.predict_next_correct("python")
        assert prob > 0.15, f"Expected >0.15 due to guessing, got {prob:.4f}"


# ═══════════════════════════════════════════════════════════════════════════
# 8. Integration: Gap Detector Seeding
# ═══════════════════════════════════════════════════════════════════════════

class TestGapDetectorIntegration:
    """Seed tracer from gap_detector output."""

    def test_create_from_gaps(self):
        gaps_result = {
            "student_id": "stu-0042",
            "career_goal": "Data Scientist",
            "gaps": [
                {"skill": "python", "estimated_level": "intermediate", "target_level": "advanced"},
                {"skill": "sql", "estimated_level": "beginner", "target_level": "intermediate"},
                {"skill": "statistics", "estimated_level": "novice", "target_level": "advanced"},
            ],
        }
        tracer = create_tracer_from_gaps(gaps_result)
        assert tracer.student_id == "stu-0042"
        # intermediate → 0.50 seed
        assert 0.40 <= tracer.get_mastery("python") <= 0.60
        # beginner → 0.10 seed
        assert tracer.get_mastery("sql") <= 0.20
        # novice → 0.25 seed
        assert 0.20 <= tracer.get_mastery("statistics") <= 0.35

    def test_seed_from_gap_detector_method(self):
        gaps_result = {
            "student_id": "test",
            "gaps": [
                {"skill": "python", "estimated_level": "advanced"},
                {"skill": "sql", "estimated_level": "novice"},
            ],
        }
        tracer = KnowledgeTracer(skills=["python", "sql"])
        tracer.seed_from_gap_detector(gaps_result)
        assert tracer.get_mastery("python") > tracer.get_mastery("sql")


# ═══════════════════════════════════════════════════════════════════════════
# 9. Summary
# ═══════════════════════════════════════════════════════════════════════════

class TestSummary:
    """Summary output is correct."""

    def test_summary_fields(self):
        tracer = KnowledgeTracer(
            skills=["python", "sql", "statistics"],
            initial_mastery={"python": 0.90, "sql": 0.30, "statistics": 0.50},
        )
        # Mark python as mastered (need 3 observations)
        for _ in range(3):
            tracer.observe(Interaction(skill="python", correct=True))
        tracer.observe(Interaction(skill="sql", correct=False))

        summary = tracer.summary()
        assert summary["total_skills"] == 3
        assert "python" in summary["mastered"]
        assert "sql" in summary["unmastered"]
        assert "masteries" in summary
        assert "levels" in summary
        assert "readiness" in summary
        assert 0.0 <= summary["readiness"] <= 1.0

    def test_export_for_transfer(self):
        tracer = KnowledgeTracer(
            skills=["python", "sql"],
            initial_mastery={"python": 0.70},
        )
        tracer.observe(Interaction(skill="python", correct=True))
        export = tracer.export_for_transfer()

        assert "masteries" in export
        assert "levels" in export
        assert "params" in export
        assert "mastered_skills" in export
        assert "unmastered_skills" in export
        assert export["masteries"]["python"] > 0.0
        assert "p_l" in export["params"]["python"]


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
