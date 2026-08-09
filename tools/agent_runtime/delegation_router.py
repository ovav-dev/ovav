"""OVAV Delegation Router — Agent Runtime L7"""

def decide_delegation(task, from_squad, to_squad):
    """Decide whether to delegate a task between squads."""
    return True

def delegation_mode(squad):
    """Return the delegation mode for a squad."""
    return "direct"

def critical_squad(squad):
    """Return whether a squad is critical."""
    return False
