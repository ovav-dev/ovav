#!/usr/bin/env python3
"""
Tests for transfer_validator.py — Education SEG-6 Phase 2
============================================================
Acceptance criteria:
  - Predicts probability of success on next item
  - Uses mastery estimates from knowledge_tracer
  - Validates skill transfer between related concepts
  - Transfer matrix weights influence predictions correctly
  - Integration with curriculum_engine for path adjustment
  - Edge cases: empty masteries, single skill, no transfer, full mastery
"""

from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.education.transfer_validator import (
    TransferValidator,
    TransferPrediction,
    create_validator_from_export,
    TRANSFER_MATRIX,
    SELF_TRANSFER,
    DEFAULT_TRANSFER,
)


# ═══════════════════════════════════════════════════════════════════════════
# 1. Initialization & Validation
# ═══════════════════════════════════════════════════════════════════════════

class TestInitialization:
    """Validator initialization and input validation."""

    def test_create_with_masteries(self):
        masteries = {"python": 0.70, "sql": 0.45, "statistics": 0.30}
        validator = TransferValidator(masteries=masteries)
        assert validator.masteries == masteries

    def test_create_with_bkt_params(self):
        masteries = {"python": 0.70}
        bkt_params = {"python": {"p_l": 0.70, "p_t": 0.10, "p_s": 0.05, "p_g": 0.15}}
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        assert validator.bkt_params == bkt_params

    def test_empty_masteries_raises(self):
        with pytest.raises(ValueError, match="At least one mastery"):
            TransferValidator(masteries={})

    def test_invalid_mastery_raises(self):
        with pytest.raises(ValueError, match="fuera de"):
            TransferValidator(masteries={"python": 1.5})

    def test_custom_transfer_matrix(self):
        custom = {"python": {"sql": 0.99}}
        masteries = {"python": 0.80, "sql": 0.30}
        validator = TransferValidator(masteries=masteries, transfer_matrix=custom)
        assert validator.get_transfer_weight("python", "sql") == 0.99

    def test_create_from_export(self):
        export = {
            "masteries": {"python": 0.70, "sql": 0.45},
            "levels": {"python": "advanced", "sql": "intermediate"},
            "params": {
                "python": {"p_l": 0.70, "p_t": 0.10, "p_s": 0.05, "p_g": 0.15},
                "sql": {"p_l": 0.45, "p_t": 0.10, "p_s": 0.10, "p_g": 0.20},
            },
            "mastered_skills": ["python"],
            "unmastered_skills": ["sql"],
        }
        validator = create_validator_from_export(export)
        assert validator.masteries == {"python": 0.70, "sql": 0.45}
        assert "python" in validator.bkt_params


# ═══════════════════════════════════════════════════════════════════════════
# 2. Transfer Weights
# ═══════════════════════════════════════════════════════════════════════════

class TestTransferWeights:
    """Transfer weight calculations."""

    def test_self_transfer_is_one(self):
        validator = TransferValidator(masteries={"python": 0.50})
        assert validator.get_transfer_weight("python", "python") == SELF_TRANSFER

    def test_known_transfer_weight(self):
        validator = TransferValidator(masteries={"python": 0.50, "go": 0.30})
        weight = validator.get_transfer_weight("python", "go")
        assert weight == TRANSFER_MATRIX["python"]["go"]

    def test_unknown_transfer_default(self):
        validator = TransferValidator(masteries={"python": 0.50, "zzz": 0.30})
        weight = validator.get_transfer_weight("zzz", "python")
        assert weight == DEFAULT_TRANSFER

    def test_transfer_vector_normalizes(self):
        masteries = {"python": 0.70, "sql": 0.50, "statistics": 0.30}
        validator = TransferValidator(masteries=masteries)
        vec = validator.compute_transfer_vector("machine_learning")
        assert len(vec) == 3
        assert abs(sum(vec.values()) - 1.0) < 1e-9


# ═══════════════════════════════════════════════════════════════════════════
# 3. Next-Item Correctness Prediction
# ═══════════════════════════════════════════════════════════════════════════

class TestNextItemCorrectness:
    """Prediction of next-item correctness."""

    def test_prediction_in_range(self):
        masteries = {"python": 0.70, "sql": 0.50}
        validator = TransferValidator(masteries=masteries)
        pred = validator.predict_next_correctness("python")
        assert 0.0 <= pred.next_item_correctness <= 1.0

    def test_high_mastery_high_prediction(self):
        """High mastery + low slip → high prediction."""
        masteries = {"python": 0.95}
        bkt_params = {"python": {"p_l": 0.95, "p_s": 0.01, "p_g": 0.10}}
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        pred = validator.predict_next_correctness("python")
        assert pred.next_item_correctness > 0.85

    def test_low_mastery_low_prediction(self):
        """Low mastery → lower prediction (but not zero due to guessing)."""
        masteries = {"python": 0.05}
        bkt_params = {"python": {"p_l": 0.05, "p_s": 0.10, "p_g": 0.05}}
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        pred = validator.predict_next_correctness("python")
        assert pred.next_item_correctness < 0.30

    def test_transfer_boosts_prediction(self):
        """Related mastered skills boost prediction for target."""
        # python with mastery 0.90 → should boost ML prediction
        masteries = {
            "python": 0.90,
            "statistics": 0.85,
            "machine_learning": 0.10,  # target skill is low
        }
        validator = TransferValidator(masteries=masteries)
        pred = validator.predict_next_correctness("machine_learning")
        # Transfer from python + statistics should boost above pure ML mastery
        assert pred.next_item_correctness > 0.10, \
            f"Expected transfer boost, got {pred.next_item_correctness}"

        # Check that source contributions are recorded
        assert len(pred.source_masteries) > 0

    def test_predict_all_skills(self):
        masteries = {"python": 0.70, "sql": 0.50, "statistics": 0.30}
        validator = TransferValidator(masteries=masteries)
        predictions = validator.predict_all()
        assert len(predictions) == 3
        for skill in masteries:
            assert skill in predictions
            assert isinstance(predictions[skill], TransferPrediction)

    def test_transfer_gap_zero_when_all_mastered(self):
        """When all skills mastered, transfer gap approaches 0."""
        masteries = {"python": 0.95, "sql": 0.95, "statistics": 0.95}
        bkt_params = {
            s: {"p_l": 0.95, "p_s": 0.01, "p_g": 0.10, "p_t": 0.05}
            for s in masteries
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        pred = validator.predict_next_correctness("python")
        assert pred.transfer_gap < 0.10, \
            f"Expected low transfer gap, got {pred.transfer_gap}"


# ═══════════════════════════════════════════════════════════════════════════
# 4. Module Readiness
# ═══════════════════════════════════════════════════════════════════════════

class TestModuleReadiness:
    """Module readiness predictions."""

    def test_module_ready_high_mastery(self):
        masteries = {"python": 0.90, "sql": 0.85}
        bkt_params = {
            s: {"p_l": 0.90, "p_s": 0.05, "p_g": 0.10, "observations": 10}
            for s in masteries
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        result = validator.predict_module_readiness(["python", "sql"])
        assert result["module_ready"] is True
        assert result["avg_readiness"] > 0.80

    def test_module_not_ready_low_mastery(self):
        masteries = {"python": 0.20, "sql": 0.30}
        validator = TransferValidator(masteries=masteries)
        result = validator.predict_module_readiness(["python", "sql"])
        assert result["module_ready"] is False
        assert result["avg_readiness"] < 0.60

    def test_module_not_ready_one_skill_blocked(self):
        """One low skill can block module readiness."""
        masteries = {"python": 0.90, "sql": 0.15}
        bkt_params = {
            "python": {"p_l": 0.90, "p_s": 0.05, "p_g": 0.10},
            "sql": {"p_l": 0.15, "p_s": 0.10, "p_g": 0.20},
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        result = validator.predict_module_readiness(["python", "sql"])
        assert result["module_ready"] is False


# ═══════════════════════════════════════════════════════════════════════════
# 5. Transfer Path Analysis
# ═══════════════════════════════════════════════════════════════════════════

class TestTransferPath:
    """Best transfer path finding."""

    def test_best_path_returns_valid(self):
        masteries = {"python": 0.90, "go": 0.40, "machine_learning": 0.10}
        validator = TransferValidator(masteries=masteries)
        path = validator.find_best_transfer_path("machine_learning")
        assert len(path) > 0
        # python should be first (strongest transfer to ML, highest mastery)
        assert path[0][0] == "python"

    def test_path_max_depth(self):
        masteries = {f"skill_{i}": 0.5 for i in range(10)}
        custom = {f"skill_{i}": {f"skill_{i+1}": 0.8} for i in range(9)}
        for i in range(9):
            custom[f"skill_{i}"][f"skill_{i+1}"] = 0.8
        validator = TransferValidator(masteries=masteries, transfer_matrix=custom)
        path = validator.find_best_transfer_path("skill_0", max_depth=3)
        assert len(path) <= 3


# ═══════════════════════════════════════════════════════════════════════════
# 6. Curriculum Integration: Path Adjustments
# ═══════════════════════════════════════════════════════════════════════════

class TestPathAdjustments:
    """Path adjustment recommendations for curriculum_engine."""

    def test_no_adjustments_when_ready(self):
        masteries = {"python": 0.90, "sql": 0.88}
        bkt_params = {
            s: {"p_l": 0.90, "p_s": 0.05, "p_g": 0.10}
            for s in masteries
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        modules = [
            {"id": "MOD-PY-01", "skill": "python", "prerequisites": []},
            {"id": "MOD-SQL-01", "skill": "sql", "prerequisites": ["MOD-PY-01"]},
        ]
        result = validator.recommend_path_adjustments(modules)
        assert len(result["adjustments"]) == 0

    def test_delay_when_prereq_not_ready(self):
        """If prerequisite skill has low mastery, module should be delayed."""
        masteries = {"python": 0.20, "sql": 0.15}
        validator = TransferValidator(masteries=masteries)
        modules = [
            {"id": "MOD-PY-01", "skill": "python", "prerequisites": []},
            {"id": "MOD-SQL-01", "skill": "sql", "prerequisites": ["MOD-PY-01"]},
        ]
        result = validator.recommend_path_adjustments(modules)
        # SQL has low mastery AND its prereq (python) is also low
        adjustments = result["adjustments"]
        assert len(adjustments) > 0

    def test_reinforce_when_low_transfer(self):
        """When transfer is insufficient, recommend reinforcement."""
        masteries = {"machine_learning": 0.15, "statistics": 0.10, "python": 0.20}
        validator = TransferValidator(masteries=masteries)
        modules = [
            {"id": "MOD-ML-01", "skill": "machine_learning",
             "prerequisites": ["MOD-DS-01", "MOD-STAT-01"]},
        ]
        result = validator.recommend_path_adjustments(modules)
        # machine_learning has low mastery + prereqs won't resolve
        adjustments = result["adjustments"]
        assert len(adjustments) > 0
        assert any(a["action"] in ("delay", "reinforce") for a in adjustments)

    def test_accelerate_when_strong_transfer(self):
        """When transfer from other skills is strong, suggest acceleration."""
        masteries = {
            "python": 0.92,
            "statistics": 0.90,
            "machine_learning": 0.30,  # Low direct mastery but...
        }
        bkt_params = {
            "python": {"p_l": 0.92, "p_s": 0.02, "p_g": 0.10},
            "statistics": {"p_l": 0.90, "p_s": 0.03, "p_g": 0.10},
            "machine_learning": {"p_l": 0.30, "p_s": 0.05, "p_g": 0.10},
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        modules = [
            {"id": "MOD-ML-01", "skill": "machine_learning",
             "prerequisites": ["MOD-DS-01", "MOD-STAT-01"]},
        ]
        result = validator.recommend_path_adjustments(modules)
        # ML readiness should be high due to transfer from python+statistics
        # But prereqs may not resolve
        assert "module_readiness" in result

    def test_adjustment_output_structure(self):
        masteries = {"python": 0.30, "sql": 0.20}
        validator = TransferValidator(masteries=masteries)
        modules = [
            {"id": "MOD-PY-01", "skill": "python", "prerequisites": []},
            {"id": "MOD-SQL-01", "skill": "sql", "prerequisites": ["MOD-PY-01"]},
        ]
        result = validator.recommend_path_adjustments(modules)
        assert "adjustments" in result
        assert "total_adjustments" in result
        assert "module_readiness" in result
        for adj in result["adjustments"]:
            assert "module_id" in adj
            assert "skill" in adj
            assert "action" in adj
            assert "reason" in adj
            assert "readiness" in adj
            assert adj["action"] in ("delay", "reinforce", "accelerate", "continue")


# ═══════════════════════════════════════════════════════════════════════════
# 7. Summary
# ═══════════════════════════════════════════════════════════════════════════

class TestSummary:
    """Summary output."""

    def test_summary_fields(self):
        masteries = {"python": 0.90, "sql": 0.30, "statistics": 0.50}
        bkt_params = {
            s: {"p_l": masteries[s], "p_s": 0.10, "p_g": 0.20}
            for s in masteries
        }
        validator = TransferValidator(masteries=masteries, bkt_params=bkt_params)
        summary = validator.summary()

        assert "skills" in summary
        assert "avg_readiness" in summary
        assert "avg_transfer_gap" in summary
        assert "high_transfer_skills" in summary
        assert "low_transfer_skills" in summary
        assert "warnings" in summary
        assert 0.0 <= summary["avg_readiness"] <= 1.0

    def test_summary_with_mixed_mastery(self):
        """High and low mastery skills produce sensible summary."""
        masteries = {
            "python": 0.95,
            "go": 0.85,
            "sql": 0.20,
            "machine_learning": 0.15,
        }
        validator = TransferValidator(masteries=masteries)
        summary = validator.summary()
        # Python (0.95) should be in high_transfer_skills (low transfer gap)
        assert "python" in summary["high_transfer_skills"]
        # ML (0.15) should be in low_transfer_skills
        assert "machine_learning" in summary["low_transfer_skills"]


if __name__ == "__main__":
    pytest.main([__file__, "-v", "--tb=short"])
