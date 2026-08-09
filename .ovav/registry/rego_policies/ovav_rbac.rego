# OVAV Operator Policies — F1.1
# Role-based access control (RBAC) with IAM-style conditions.
package ovav.rbac

# ── Operator definitions ─────────────────────────────────────────────────────

operators := {
    "thavren": {
        "roles": ["systems_architect", "platform_lead"],
        "scopes": ["repo_local", "global_diagnostic", "install_sandbox"],
        "max_session_minutes": 480,
        "step_up_required_for": ["global_write", "plugin_install", "production_claim"],
    },
    "eidren": {
        "roles": ["research_analyst", "research_lead"],
        "scopes": ["repo_local", "research_external"],
        "max_session_minutes": 240,
        "step_up_required_for": ["git_write", "external_service_behavior"],
    },
}

default allow_operator_scope = false

allow_operator_scope {
    op := operators[input.operator]
    input.scope == op.scopes[_]
}

require_step_up {
    op := operators[input.operator]
    input.action == op.step_up_required_for[_]
}

# ── Resource policies ────────────────────────────────────────────────────────

resources := {
    "permission_authority.json": {
        "write_operators": ["thavren"],
        "read_operators": ["thavren", "eidren"],
        "integrity_required": true,
    },
    "bootstrap_chain": {
        "verify_operators": ["thavren", "eidren"],
        "modify_operators": ["thavren"],
        "startup_required": true,
    },
    "secrets_vault": {
        "access_operators": ["thavren"],
        "read_operators": ["thavren"],
        "require_unlock": true,
    },
    "network_guard": {
        "configure_operators": ["thavren"],
        "bypass_operators": [],
    },
    "integrity_monitor": {
        "baseline_operators": ["thavren"],
        "heal_operators": ["thavren"],
    },
}

allow_resource_action {
    res := resources[input.resource]
    input.operator == res[input.access_type][_]
}
