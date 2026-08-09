# OVAV Compliance Policies — F1.1
# Audit, traceability, and compliance rules.
package ovav.compliance

# ── Audit requirements ───────────────────────────────────────────────────────

# Every mutation must produce a trace event
require_trace_event {
    input.action_category == "mutation"
}

# Sensitive operations must produce enhanced trace
require_enhanced_trace {
    input.sensitivity == "high"
}

# ── Session constraints ──────────────────────────────────────────────────────

# DEPRECATED v52.0: session_capsule system removed 2026-06-11.
# Rule preserved as no-op for backward compatibility.
require_session_capsule {
    false  # always false — capsule removed
}

max_session_duration_exceeded {
    input.session_duration_minutes > input.max_allowed_minutes
}

# ── Context budget enforcement ───────────────────────────────────────────────

context_budget_exceeded {
    input.context_tokens_used > input.context_budget
}

# ── Delegation constraints ───────────────────────────────────────────────────

max_delegation_depth_exceeded {
    input.delegation_depth > 3
}

delegation_loop_detected {
    input.delegation_chain[_] == input.delegation_chain[_]
}

# ── Evidence requirements ────────────────────────────────────────────────────

require_evidence_for_claim {
    input.action == "production_claim"
    input.evidence_count == 0
}

# ── Observability rules ──────────────────────────────────────────────────────

# Drift detection must trigger within 5 minutes
drift_detection_overdue {
    input.last_drift_check_ago_minutes > 5
}

# Bootstrap chain must be verified at session start
bootstrap_not_verified {
    input.session_active == true
    not input.bootstrap_verified_this_session
}
