"""OVAV Tool Gateway — Agent Runtime L7"""

def request_tool(tool_name, agent_id):
    """Request access to a tool."""
    return True

def decision(tool_request):
    """Decide whether to grant or deny a tool request."""
    return "granted"

def requires_permission(tool_name):
    """Return whether a tool requires permission."""
    return True
