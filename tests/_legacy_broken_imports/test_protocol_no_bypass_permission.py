"""
Eval: Protocol access cannot bypass permission gate.

Tests adversarial scenarios where callers try to access protocols without proper authorization.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_access_gate import gate_protocol_access


def test_protocol_no_bypass_permission():
    """Protocol access must be denied for unauthorized callers."""

    # Test: Unknown caller profile
    result = gate_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="unknown_attacker",
        operation="read_file"
    )
    assert result["allowed"] == False, "Unknown profile should be denied"

    # Test: Empty caller profile
    result2 = gate_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="",
        operation="read_file"
    )
    assert result2["allowed"] == False, "Empty profile should be denied"

    # Test: Blocked scope attempt
    result3 = gate_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="ovav_systems_architect",
        operation="read_file",
        context_scope="production"
    )
    assert result3["allowed"] == False, "Production scope should be denied"

    # Test: Global scope attempt
    result4 = gate_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="ovav_systems_architect",
        operation="read_file",
        context_scope="global"
    )
    assert result4["allowed"] == False, "Global scope should be denied"

    print("PASS: Adversarial permission bypass attempts correctly denied")
    return True


if __name__ == "__main__":
    try:
        test_protocol_no_bypass_permission()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
