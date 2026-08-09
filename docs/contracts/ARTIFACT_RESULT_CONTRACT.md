# Artifact Result Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

Every artifact produced by any OVAV agent—whether a code change, a research brief, a deployment
manifest, or a generated config—must conform to a strict output schema. This contract guarantees
that downstream consumers (other agents, validators, harnesses, the CEO dashboard) can parse and
act on results without guessing or manual inspection.

Applies to all artifacts under `.ovav/artifacts/`, `docs/`, `go-runtime/`, and any file
produced as the result of a work-order execution.

## Input / Output Schema

### Input
A work order containing: `area`, `layer`, `capsule_id`, `request`, `context_pack`.

### Required Output Fields

| Field | Type | Description |
|-------|------|-------------|
| `status` | `ok \| degraded \| blocked` | Overall outcome. |
| `executive_summary` | string (≤200 chars) | One-paragraph summary for CEO dashboard. |
| `artifacts_produced` | list of paths | Files created or modified (absolute repo-relative). |
| `evidence` | list of objects | Each with `type` (test/validator/benchmark) and `result` (pass/fail). |
| `risks` | list of strings | Known risks introduced or remaining. |
| `next_recommended_action` | string or null | What to do next, or `null` if complete. |
| `skill_resolution` | string | Skill or harness that produced the primary result. |
| `memory_action` | `persist \| discard \| redact` | What to do with memory from this execution. |

### Forbidden in Output
- Raw stack traces (redact to user-safe message).
- Internal agent reasoning (only `executive_summary` goes to the user).

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Result Contract Validator | `tools/validators/validate_result_contracts.py` | Post-artifact-creation |
| Phase DAG | `tools/validators/validate_phase_dag.py` | Phase-gate transition |
| Harness Alignment | `tools/validators/check_harness_contract_alignment.py` | Harness-run completion |

The Result Contract Validator checks every `.ovav/artifacts/` entry for the 8 required fields.
Missing fields or `status: blocked` artifacts halt the phase DAG—no downstream work can start.

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Optional field missing (`next_recommended_action`) | Warning. Artifact accepted. |
| **MEDIUM** | Required field missing | Artifact rejected. Phase gate blocked. Lead notified. |
| **HIGH** | `status: ok` but evidence shows failures | Artifact quarantined. Audit log entry created. Agent output distrusted for 24h. |
| **CRITICAL** | Fabricated evidence | Agent permanently suspended. CEO notified. Commit reverted. |

## Review Cycle

Every 30 days the Platform Engineering lead reviews breach logs and adjusts field requirements
if the schema has drifted from operational reality. Schema changes require a new contract
version and a 7-day migration window for all active agents.
