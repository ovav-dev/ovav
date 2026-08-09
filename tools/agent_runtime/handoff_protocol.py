"""OVAV Handoff Protocol — Agent Runtime L7"""

def create_handoff(from_agent, to_agent, context):
    """Create a handoff between two agents."""
    return {"from": from_agent, "to": to_agent, "context": context}

def decision(handoff):
    """Decide whether to approve or deny a handoff."""
    return "approved"

def denied_context(handoff):
    """Return the denied context for a rejected handoff."""
    return None
