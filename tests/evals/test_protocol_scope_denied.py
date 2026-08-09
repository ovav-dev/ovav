"""
Eval: Protocol scope enforcement.

Tests that access is denied when context scope does not allow protocol access.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.protocols.gateway import ProtocolAccessRequest, ProtocolGateway


def test_protocol_scope_denied():
    """Protocol access must be denied for blocked scopes."""

    gateway = ProtocolGateway()

    # Blocked scopes
    blocked_scopes = ["global", "production"]
    for scope in blocked_scopes:
        request = ProtocolAccessRequest(
            protocol_id="mcp_filesystem",
            caller_profile="ovav_systems_architect",
            operation="read_file",
            context_scope=scope
        )
        scope_result = gateway._check_scope(request)
        assert scope_result.passed == False, f"Scope '{scope}' should be denied"
        assert scope in scope_result.reason.lower() or "blocked" in scope_result.reason.lower(), \
            f"Reason should mention blocked scope, got: {scope_result.reason}"

    # Allowed scopes
    allowed_scopes = ["source_local", "project_local", "session"]
    for scope in allowed_scopes:
        request = ProtocolAccessRequest(
            protocol_id="mcp_filesystem",
            caller_profile="ovav_systems_architect",
            operation="read_file",
            context_scope=scope
        )
        scope_result = gateway._check_scope(request)
        assert scope_result.passed == True, f"Scope '{scope}' should be allowed"

    print(f"PASS: Scope enforcement verified ({len(blocked_scopes)} blocked, {len(allowed_scopes)} allowed)")
    return True


if __name__ == "__main__":
    try:
        test_protocol_scope_denied()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
