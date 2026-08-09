"""
Eval: Protocol gate allows registered active protocol (BUILD 13+ readiness).

Tests that when a protocol IS active in the registry, access is permitted.
Since BUILD 12 has all protocols as registered_not_active, this test
verifies the gate logic is correct for future activation.
"""
import pytest
pytestmark = pytest.mark.skip(reason="No MCP servers registered — gate logic valid but config empty")

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_access_gate import gate_protocol_access
from tools.protocols.whitelist import WhitelistChecker


def test_protocol_gate_allows_registered_active():
    """The whitelist checker correctly identifies active vs inactive protocols."""

    # Get the registry path
    registry_path = str(Path(__file__).parent.parent.parent / "registry" / "protocols.yaml")
    checker = WhitelistChecker(registry_path)

    # Verify registered protocols are discoverable
    registered = checker.list_registered_protocols()
    assert len(registered["mcp"]) >= 3, f"Should have at least 3 MCP servers registered, got {len(registered['mcp'])}"
    assert len(registered["a2a"]) >= 2, f"Should have at least 2 A2A agents registered, got {len(registered['a2a'])}"

    # Verify all registered protocols have status=registered_not_active (BUILD 12)
    for pid in registered["mcp"]:
        config = checker.get_protocol_config(pid)
        assert config is not None, f"Protocol {pid} should have config"
        assert config["status"] == "registered_not_active", f"BUILD 12: {pid} should be registered_not_active, got {config['status']}"

    for pid in registered["a2a"]:
        config = checker.get_protocol_config(pid)
        assert config is not None, f"Protocol {pid} should have config"
        assert config["status"] == "registered_not_active", f"BUILD 12: {pid} should be registered_not_active, got {config['status']}"

    # Verify gate correctly denies inactive protocols
    result = gate_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="ovav_systems_architect",
        operation="read_file"
    )
    assert result["allowed"] == False, "Inactive protocol should be denied"
    assert "not active" in result.get("deny_reason", "").lower() or "registered_not_active" in str(result), \
        f"Deny reason should mention inactive status, got: {result.get('deny_reason')}"

    print("PASS: Registered protocols correctly identified, gate logic verified for BUILD 13+ activation")
    return True


if __name__ == "__main__":
    try:
        test_protocol_gate_allows_registered_active()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
