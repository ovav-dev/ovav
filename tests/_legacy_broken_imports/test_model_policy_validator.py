"""Eval for source-local model policy validation."""

from __future__ import annotations

import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from tools.validators.validate_model_policy import validate


def test_source_local_model_policy_passes_current_runtime_surfaces():
    assert validate(REPO_ROOT) == []


if __name__ == "__main__":
    test_source_local_model_policy_passes_current_runtime_surfaces()
    print("PASS model policy validator eval")
