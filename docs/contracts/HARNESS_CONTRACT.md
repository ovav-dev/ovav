# Harness Execution Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

A harness is OVAV's unit of automated verification. Every harness must declare its identity,
pre/post conditions, allowed side effects, and blocking behavior before it can be executed.
This contract ensures harnesses are predictable, auditable, and safe to run in production.

Applies to all harnesses under `tools/harnesses/impl/`, `tools/harnesses/checks/`, and any
Go-native harnesses under `go-runtime/internal/validators/`.

## Input / Output Schema

### Harness Definition (minimum required fields)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique harness identifier (e.g., `h_git_push_review`). |
| `tier` | `critical \| standard \| optional` | Yes | Critical harnesses block production; optional ones warn. |
| `trigger` | `auto \| manual \| scheduled` | Yes | How the harness is invoked. |
| `inputs` | list of paths | Yes | Files or artifacts the harness reads. |
| `outputs` | list of objects | Yes | Expected output format: `{name, type, required}`. |
| `allowed_side_effects` | list of strings | Yes | What the harness may modify (e.g., `status_file`, `audit_log`). |
| `blocking` | boolean | Yes | If `true`, failure halts the current work phase. |
| `evidence` | list of strings | Yes | Types of evidence produced: `test_result`, `file_hash`, `diff`, `metric`. |
| `evals` | list of strings | Yes | Evaluation criteria: `pass_on`, `fail_on`, `warn_on`. |
| `owner_profile` | string | Yes | Service profile that owns this harness. |

### Runtime Pre-Conditions (checked before execution)
1. All `inputs` exist and are readable.
2. No other harness with the same `id` is currently running (no concurrent duplicate).
3. The caller has permission in `permission_authority.json` for this harness's `owner_profile`.

### Runtime Post-Conditions (checked after execution)
1. All required `outputs` were produced.
2. `evidence` artifacts are present and non-empty.
3. Side effects are within `allowed_side_effects`—any unexpected write is a violation.

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Harness Validator | `tools/validators/validate_harnesses.py` | Pre-execution + post-execution |
| Harness Contract Alignment | `tools/validators/check_harness_contract_alignment.py` | Every commit |
| Phase DAG | `tools/validators/validate_phase_dag.py` | Phase-gate transition |

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Optional output missing | Warning. Harness result accepted. |
| **MEDIUM** | Pre-condition not met (missing input) | Harness skipped. Phase not advanced. |
| **HIGH** | Side effect outside `allowed_side_effects` | Harness quarantined. Audit entry. Lead operator review required. |
| **CRITICAL** | Critical-tier harness fails in production | System-wide hard stop. CEO notified. Rollback initiated. |

## Review Cycle

Every 30 days, the Platform Engineering lead:
1. Audits harness execution logs for unexpected side effects.
2. Reviews `tier` assignments—promote/demote based on 30-day reliability data.
3. Retires harnesses with zero invocations in the review window.

Retired harnesses are moved to `_deprecated/` and removed from `auto_triggers.yaml`.
