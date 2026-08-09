# Automatic Trigger Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

Critical validators and harnesses in OVAV must fire automatically based on runtime conditions—
never relying on human memory or manual invocation. This contract defines what triggers exist,
when they fire, and what happens if they fail to activate.

Applies to all triggers registered in `.ovav/registry/auto_triggers.yaml` and all harnesses
with `trigger: auto`.

## Input / Output Schema

### Trigger Definition (in `auto_triggers.yaml`)

```yaml
triggers:
  - id: pre_commit_gate
    event: git.pre_commit
    harness: h_git_stage_review
    blocking: true
  - id: push_review
    event: git.pre_push
    harness: h_git_push_review
    blocking: true
```

### Required Auto-Triggers

| Trigger ID | Event | Harness | Blocking |
|------------|-------|---------|----------|
| `sdd_init` | `session.start` or `project.unknown` | `h_sdd_init` | No |
| `skill_refresh` | `skills.modified` | `h_skill_registry_refresh` | No |
| `permission_gate` | `work_order.received` | `h_permission_gate` | Yes |
| `phase_dag` | `artifact.created` | `h_phase_dag` | Yes |
| `evidence_verify` | `evidence.attached` | `h_verify_evidence` | Yes |
| `stage_review` | `git.stage` | `h_git_stage_review` | Yes |
| `commit_review` | `git.commit` | `h_git_commit_review` | Yes |
| `push_review` | `git.push` | `h_git_push_review` | Yes |

Manual invocation (`ovav harness run <name>`) is allowed only for debugging and must be
logged with reason `manual_debug`.

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Harness Validator | `tools/validators/validate_harnesses.py` | Every harness invocation |
| Registry Drift | `tools/validators/check_registry_drift.py` | Every 6 hours |
| Auto-Trigger Integrity | `tools/validators/check_registry_drift.py` (trigger section) | Post-merge |

The system checks that every trigger in `auto_triggers.yaml` has a corresponding harness
definition and that no critical trigger has been disabled without an active CEO waiver.

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Non-blocking trigger fails | Logged. Retry on next event. |
| **MEDIUM** | Blocking trigger skipped manually | Work blocked until trigger re-run. Audit entry. |
| **HIGH** | Critical trigger disabled without waiver | Integrity mesh `DEGRADED`. All writes suspended. |
| **CRITICAL** | Trigger list tampered (auto_triggers.yaml modified) | `CRITICAL` alert. System-wide hard stop. CEO notified. |

## Review Cycle

Every 30 days, the Platform Engineering lead reviews the trigger registry for:
- Stale triggers referencing deleted harnesses.
- Missing triggers for new critical paths.
- Trigger performance (execution time, failure rate).

Triggers with >5% failure rate in the review window are marked `degraded` and must be
repaired or replaced within 7 days.
