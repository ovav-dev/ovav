# Skill Rule Pack Contract

**Version:** 1.0.0 | **Owner:** Thavren | **Area:** Platform Engineering & DX | **Review:** 30 days

## Purpose & Scope

Skills are loadable instruction packs that extend agent capabilities at runtime. This contract
defines how skills are defined, loaded, verified, and executed—ensuring no skill can compromise
the agent or bypass governance.

Applies to all skills under `.opencode/skills/` and any Cloudflare-managed skills loaded
from `~/.config/opencode/skills/` that OVAV agents may invoke.

## Skill Definition Schema

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique skill identifier (e.g., `ovav-context-pack`). |
| `owner` | string | Service profile that owns this skill. |
| `lane` | string | Service lane within the owner's area. |
| `trigger` | string | When the skill is auto-loaded (regex or event name). |
| `inputs` | list of strings | Required inputs: file paths, env vars, context pack fields. |
| `tools` | list of strings | Allowed tools (empty = all, `[]` = none). |
| `denied_tools` | list of strings | Explicitly blocked tools (e.g., `bash`, `write` for read-only skills). |
| `required_artifacts` | list of strings | Upstream artifacts that must exist before skill runs. |
| `memory_behavior` | `persist \| discard \| redact` | What happens to memory after skill execution. |
| `risk_level` | `low \| medium \| high \| critical` | Risk classification. |
| `evals` | list of strings | Evaluation test IDs that validate this skill. |
| `rollback_strategy` | string | How to undo if skill execution fails. |
| `output_contract` | string | Reference to the contract this skill's output must satisfy. |
| `examples` | list of objects | `{scenario, expected_behavior}`. |
| `anti_patterns` | list of strings | Common misuse patterns that trigger warnings. |

### Forbidden
- A `critical` skill with no `rollback_strategy`.
- A skill that lists `bash` in `tools` but also claims `risk_level: low`.
- A skill owned by one area that references `required_artifacts` from another area without
  a handoff annotation.

## Loading & Verification Pipeline

1. **Load**: Skill YAML parsed. Schema validated against required fields.
2. **Permission Gate**: Skill `tools` list checked against `permission_authority.json` for
   the invoking agent.
3. **Artifact Gate**: All `required_artifacts` verified to exist and be `status: done`.
4. **Risk Gate**: `critical` skills require CEO waiver or pre-approved bypass.
5. **Inject**: Skill instructions injected into agent context.
6. **Post-execution**: Output validated against `output_contract`. Memory handled per
   `memory_behavior`.

## Enforcement Mechanism

| Validator | File | Trigger |
|-----------|------|---------|
| Skill Validator | `tools/validators/validate_skills.py` | Skill load + post-execution |
| Permission Gate | `tools/harnesses/impl/h_permission_gate.py` | Skill load |
| Phase DAG | `tools/validators/validate_phase_dag.py` | Artifact dependency check |

## Breach Consequences

| Severity | Trigger | Consequence |
|----------|---------|-------------|
| **LOW** | Anti-pattern detected | Warning logged. Execution continues. |
| **MEDIUM** | Required artifact missing | Skill blocked. Agent must resolve dependency first. |
| **HIGH** | Skill used denied tool | Skill quarantined for 24h. Audit entry. |
| **CRITICAL** | Critical skill executed without waiver | Agent session terminated. CEO notified. Skill permanently disabled. |

## Review Cycle

Every 30 days:
1. Audit skill execution logs for denied-tool violations.
2. Re-evaluate `risk_level` based on 30-day incident data.
3. Update `anti_patterns` with newly discovered misuse patterns.
4. Verify all `evals` still pass against current codebase.
