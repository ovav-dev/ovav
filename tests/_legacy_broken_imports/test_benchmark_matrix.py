"""Eval: Benchmark Matrix — BUILD 11.

Tests BenchmarkEngine comparison, ranking, trade-off detection, gap analysis.
"""

import sys
from pathlib import Path

_src = Path(__file__).resolve().parents[2]
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))

from tools.research.benchmark_engine import BenchmarkDecision, BenchmarkEngine


class TestBenchmarkMatrix:
    """Benchmark engine eval suite."""

    def setup_method(self):
        self.e = BenchmarkEngine()

    def test_basic_comparison_two_candidates(self):
        """Two candidates with clear winner."""
        self.e.add_dimension("speed", weight=1.0)
        self.e.add_candidate("Fast")
        self.e.add_candidate("Slow")
        self.e.score("Fast", "speed", 0.9, confidence=0.9)
        self.e.score("Slow", "speed", 0.4, confidence=0.8)
        matrix = self.e.compare()
        assert len(matrix.rankings) == 2
        assert matrix.rankings[0][0] == "Fast"

    def test_weighted_dimensions(self):
        """Weighted dimensions should influence ranking."""
        self.e.add_dimension("critical", weight=2.0)
        self.e.add_dimension("nice_to_have", weight=0.5)
        self.e.add_candidate("A")
        self.e.add_candidate("B")
        self.e.score("A", "critical", 0.3)
        self.e.score("A", "nice_to_have", 0.9)
        self.e.score("B", "critical", 0.9)
        self.e.score("B", "nice_to_have", 0.3)
        matrix = self.e.compare()
        # B should win because critical is weighted heavily
        assert matrix.rankings[0][0] == "B"

    def test_tradeoff_detection(self):
        """When winner doesn't lead all dimensions, trade-offs should be detected."""
        self.e.add_dimension("security", weight=1.0)
        self.e.add_dimension("usability", weight=1.0)
        self.e.add_candidate("SecureTool")
        self.e.add_candidate("EasyTool")
        self.e.score("SecureTool", "security", 0.95)
        self.e.score("SecureTool", "usability", 0.30)
        self.e.score("EasyTool", "security", 0.40)
        self.e.score("EasyTool", "usability", 0.95)
        matrix = self.e.compare()
        # There should be trade-offs since each leads on one dimension
        assert len(matrix.dimension_tradeoffs) >= 1

    def test_reject_on_bad_score(self):
        """Candidate below threshold_bad should be REJECT."""
        self.e.add_dimension("quality", weight=1.0, threshold_bad=0.3)
        self.e.add_candidate("BadTool")
        self.e.score("BadTool", "quality", 0.2)
        matrix = self.e.compare()
        assert matrix.rankings[0][2] == BenchmarkDecision.REJECT.value

    def test_adopt_on_high_score(self):
        """Candidate above ADOPT threshold should get ADOPT."""
        self.e.add_dimension("quality", weight=1.5)
        self.e.add_candidate("GreatTool")
        self.e.score("GreatTool", "quality", 0.95, confidence=0.9)
        matrix = self.e.compare()
        assert matrix.rankings[0][2] == BenchmarkDecision.ADOPT.value

    def test_missing_dimension_in_score(self):
        """Scores for dimensions not registered should raise KeyError."""
        self.e.add_candidate("Tool")
        try:
            self.e.score("Tool", "nonexistent", 0.5)
            assert False, "Should have raised KeyError"
        except KeyError:
            pass

    def test_empty_comparison(self):
        """Empty comparison should not crash."""
        matrix = self.e.compare()
        assert "Insufficient" in matrix.summary or "Insufficient" in str(matrix.rankings)

    def test_gap_analysis(self):
        """Gap analysis should detect dimensions where all candidates score low."""
        self.e.add_dimension("innovation", weight=1.0, threshold_good=0.8, threshold_bad=0.3)
        self.e.add_candidate("A")
        self.e.add_candidate("B")
        self.e.score("A", "innovation", 0.4)
        self.e.score("B", "innovation", 0.5)
        matrix = self.e.compare()
        gaps = self.e.dimension_gap_analysis(matrix)
        assert "innovation" in gaps
        assert gaps["innovation"]["avg_score"] < 0.8

    def test_to_dict_serializable(self):
        """Matrix serializes to dict."""
        self.e.add_dimension("x")
        self.e.add_candidate("C")
        self.e.score("C", "x", 0.5)
        matrix = self.e.compare()
        d = self.e.to_dict(matrix)
        assert "candidates" in d
        assert "rankings" in d

    def test_score_upsert(self):
        """Scoring same candidate+dimension twice should update, not duplicate."""
        self.e.add_dimension("x")
        self.e.add_candidate("C")
        self.e.score("C", "x", 0.3)
        self.e.score("C", "x", 0.7)  # update
        matrix = self.e.compare()
        # Should only have one score for C on x
        scores_for_c_x = [s for s in matrix.scores if s.candidate_name == "C" and s.dimension_name == "x"]
        assert len(scores_for_c_x) == 1
        assert scores_for_c_x[0].score == 0.7


if __name__ == "__main__":
    t = TestBenchmarkMatrix()
    for name in sorted(dir(t)):
        if name.startswith("test_"):
            t.setup_method()  # Reset engine before each test
            method = getattr(t, name)
            try:
                method()
                print(f"  ✅ {name}")
            except Exception as e:
                print(f"  ❌ {name}: {e}")
