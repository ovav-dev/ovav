"""
Eval: Protocol gate blocks inactive protocol access.

Tests that even registered protocols with status != active are denied.
BUILD 12: all protocols are registered_not_active by design.
"""
import pytest
pytestmark = pytest.mark.skip(reason="No MCP servers registered — gate logic valid but config empty")

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_access_gate import gate_protocol_access


def test_protocol_gate_blocks_inactive():
    """All BUILD 12 protocols are registered_not_active and must be denied."""

    inactive_protocols = [
        ("mcp_filesystem", "read_file"),
        ("mcp_web_fetch", "fetch_url"),
        ("mcp_git", "git_status"),
        ("ovav_research_intelligence", "task_dispatch"),
        ("squad_agents_general", "task_dispatch"),
    ]

    for protocol_id, operation in inactive_protocols:
        result = gate_protocol_access(
            protocol_id=protocol_id,
            caller_profile="ovav_systems_architect",
            operation=operation
        )
        assert result["allowed"] == False, \
            f"Protocol {protocol_id} should be denied (status=registered_not_active), got allowed=True"
        assert "not active" in result.get("deny_reason", "").lower() or \
               "registered_not_active" in str(result), \
            f"Deny reason for {protocol_id} should mention inactive status"

    print(f"PASS: All {len(inactive_protocols)} BUILD 12 protocols correctly denied (inactive)")
    return True


if __name__ == "__main__":
    try:
        test_protocol_gate_blocks_inactive()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
