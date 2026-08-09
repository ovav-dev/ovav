"""
Eval: Protocol circuit breaker resets after success.

Tests that recording a success resets the failure counter and closes the circuit.
"""

import sys
import time
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_circuit_breaker import check_circuit, record_failure, record_success


def test_protocol_circuit_breaker_resets():
    """Circuit breaker must reset to CLOSED after recording success."""

    test_id = f"test_cb_reset_{int(time.time())}"

    # Set up: open the circuit with 5 failures
    for i in range(5):
        record_failure(test_id)

    state = check_circuit(test_id)
    assert state["state"] == "OPEN", "After 5 failures, circuit should be OPEN"

    # Record success → should reset to CLOSED with 0 failures
    record_success(test_id)
    state = check_circuit(test_id)
    assert state["state"] == "CLOSED", f"After success, circuit should reset to CLOSED, got {state['state']}"
    assert state["failure_count"] == 0, f"Failure count should reset to 0, got {state['failure_count']}"

    # Verify we can accumulate failures again from 0
    record_failure(test_id)
    state = check_circuit(test_id)
    assert state["failure_count"] == 1, f"Failure count should be 1 after one new failure, got {state['failure_count']}"

    print("PASS: Circuit breaker correctly resets after success (CLOSED, failure_count=0)")
    return True


if __name__ == "__main__":
    try:
        test_protocol_circuit_breaker_resets()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
