"""
Eval: Protocol audit logs all access attempts (success and failure).

Tests that the audit logger records every access attempt regardless of outcome.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from tools.harnesses.h_protocol_audit_log import log_protocol_access, read_audit_trail


def test_protocol_audit_logs_all_access():
    """Audit log must record both allowed and denied accesses."""

    # Log a denied access
    r1 = log_protocol_access(
        protocol_id="mcp_filesystem",
        caller_profile="ovav_systems_architect",
        operation="read_file",
        result_status="denied_whitelist",
        protocol_type="mcp"
    )
    assert r1["audit_entry_written"] == True, "Denied access should be logged"

    # Log a denied permission access
    r2 = log_protocol_access(
        protocol_id="mcp_web_fetch",
        caller_profile="ovav_research_analyst",
        operation="fetch_url",
        result_status="denied_permission",
        protocol_type="mcp"
    )
    assert r2["audit_entry_written"] == True, "Permission denied should be logged"

    # Log a denied budget access
    r3 = log_protocol_access(
        protocol_id="mcp_git",
        caller_profile="ovav_systems_architect",
        operation="git_status",
        result_status="denied_budget",
        protocol_type="mcp"
    )
    assert r3["audit_entry_written"] == True, "Budget denied should be logged"

    # Log an allowed access (for future BUILD 13+)
    r4 = log_protocol_access(
        protocol_id="mcp_git",
        caller_profile="ovav_systems_architect",
        operation="git_status",
        result_status="allowed",
        protocol_type="mcp"
    )
    assert r4["audit_entry_written"] == True, "Allowed access should be logged"

    # Verify all 4 entries are readable
    trail = read_audit_trail(limit=10)
    assert trail["count"] >= 4, f"Should have at least 4 entries, got {trail['count']}"

    # Verify entries have required fields
    for entry in trail["entries"][:4]:
        for field in ["timestamp", "protocol_id", "caller_profile", "operation", "result_status"]:
            assert field in entry, f"Field '{field}' missing from audit entry"

    print(f"PASS: Audit log correctly records {trail['count']} entries (both allowed and denied)")
    return True


if __name__ == "__main__":
    try:
        test_protocol_audit_logs_all_access()
        sys.exit(0)
    except AssertionError as e:
        print(f"FAIL: {e}")
        sys.exit(1)
