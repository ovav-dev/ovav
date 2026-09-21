"""
Eval: Protocol circuit breaker opens after N consecutive failures.

Tests the circuit breaker state machine: CLOSED → OPEN after threshold.
"""

import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_circuit_breaker import check_circuit, record_failure, record_success


def test_protocol_circuit_breaker_opens():
    """Circuit breaker must open after reaching failure threshold."""

    test_id = f"test_cb_opens_{int(time.time())}"

    # Verify initial state
    state = check_circuit(test_id)
    assert state["state"] == "CLOSED", f"Initial state should be CLOSED, got {state['state']}"

    # Record 5 failures → should open
    for i in range(5):
        record_failure(test_id)

    state = check_circuit(test_id)
    assert state["state"] == "OPEN", f"After 5 failures state should be OPEN, got {state['state']}"
    assert state["failure_count"] >= 5, f"Failure count should be at least 5, got {state['failure_count']}"

    # Clean up: reset to CLOSED
    record_success(test_id)
    state = check_circuit(test_id)
    assert state["state"] == "CLOSED", f"After success state should be CLOSED, got {state['state']}"

    print("PASS: Circuit breaker correctly opens after 5 consecutive failures")
    return True


if __name__ == "__main__":
    try:
        test_protocol_circuit_breaker_opens()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
