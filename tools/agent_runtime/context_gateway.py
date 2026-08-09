"""OVAV Context Gateway — Agent Runtime L7"""

def request_context(agent_id, context_type):
    """Request context for an agent."""
    return {}

def research_no_repo_default(agent_id):
    """Return research agents' default repo access."""
    return False

def decision(context_request):
    """Decide whether to grant or deny a context request."""
    return "granted"
