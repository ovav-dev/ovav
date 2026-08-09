"""
Eval: Protocol gate blocks unregistered protocol access.

Tests that the protocol access gate denies any protocol not registered in protocols.yaml.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_access_gate import gate_protocol_access


def test_protocol_gate_blocks_unregistered():
    """Unregistered protocols must be denied."""
    result = gate_protocol_access(
        protocol_id="mcp_completely_fake_nonexistent_xyz",
        caller_profile="ovav_systems_architect",
        operation="test_op"
    )

    assert result["allowed"] == False, f"Unregistered protocol should be denied, got: {result}"
    assert "deny_reason" in result, "Deny reason should be present"
    assert "whitelist" in result["gate_checks"], "Whitelist check should be present"
    assert result["gate_checks"]["whitelist"]["passed"] == False, "Whitelist check should fail"

    print("PASS: Unregistered protocol correctly denied")
    return True


if __name__ == "__main__":
    try:
        test_protocol_gate_blocks_unregistered()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
