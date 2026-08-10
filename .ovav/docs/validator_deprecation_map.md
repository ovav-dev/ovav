# Validator Deprecation Map

## Overview

This file documents the mapping from legacy validators to the new OMARS (OVAV Monitoring & Auto-Remediation System).

**Principle**: Validators are NOT deleted — they are marked DEPRECATED and delegate to monitors internally. This ensures backward compatibility while the new system takes over.

---

## Validator → Monitor Mapping

| Legacy Validator | Replaced By | Reason |
|----------------|-------------|--------|
| `cross_target_consistency` | `AgentProjectionMonitor` | Count-based check, not file diff |
| `merge_readiness` | `HygieneMonitor` | OWS hygiene already checks this |
| `workspace_safety` | `HygieneMonitor` | OWS hygiene already checks this |
| `runtime_integrity` | `RuntimeIntegrityMonitor` + auto-fix | Creates baseline if missing |
| `caps_chronos_temporal_alignment` | REMOVED (→ WARN only) | Stale caps.yaml is not an error |
| `head_integrity_verifier` | REMOVED | Hash drift is normal between sessions |
| `architecture_guardian` | `ArchitectureMonitor` (→ WARN only) | Directory structure not critical |
| `context_firewall_v2` | REMOVED | Identical to context_firewall |
| `release_gate` | `HygieneMonitor` | Same check as merge_readiness |
| `handoff_sync` | REMOVED | Redundant with hygiene |

---

## Validators That Remain BLOCKING (Tier 1)

These validators remain active as they check security-critical items:

| Validator | Reason |
|-----------|--------|
| `credential_governance` | Vault and credential policy |
| `agent_permission_invariants` | Lead↔area permission alignment |
| `security_hardening` | F4 bash commands dangerous |
| `secrets_hygiene` | Secrets in code detection |

---

## Validators That Remain WARNING (Tier 2)

These are useful checks but don't block work:

| Validator | Reason |
|-----------|--------|
| `contract_freshness` | Contract hash vs baseline |
| `single_authority` | caps.yaml + git HEAD integrity |
| `agent_runtime_enforcement` | Skills loaded vs surface |
| `tool_readiness` | Skill matrix validity |

---

## Validators That Are DEPRECATED (Tier 3 - Info Only)

These will run but emit DEPRECATED warning:

| Validator | Replacement |
|-----------|-------------|
| `context_firewall` | No replacement (check absorbed into hygiene) |
| `supply_chain` | SBOM monitor (info only) |
| `agent_surface_hierarchy` | No replacement (count check only) |
| `caps_schema` | No replacement (YAML syntax check only) |
| `registry_validator` | No replacement (file existence check only) |

---

## How Deprecation Works

Each deprecated validator:

1. Still runs when called explicitly
2. Emits a `DEPRECATED` status with message pointing to new system
3. Returns PASS if the underlying monitor would have passed
4. Returns WARN (not FAIL) if issues found

```
ovav validate --validator cross_target_consistency
DEPRECATED: This validator is deprecated. Use AgentProjectionMonitor instead.
```

---

## Migration Path

**Phase 1** (Current): Dual operation
- Monitors run in background
- Validators still fail (blocking)
- Mapping file documents transition

**Phase 2** (After 30 days): Soft migration
- Validators emit WARN instead of FAIL
- Monitors become primary
- Validators still exist for compatibility

**Phase 3** (After 60 days): Monitor-only
- Blocking validators removed
- OMARS is the single source of truth
- `ovav validate` runs monitors and reports

---

## Running Monitors Directly

```bash
# Run all monitors once
ovav monitor run

# Run specific monitor
ovav monitor run hygiene
ovav monitor run agent_projection

# Run monitor loop (background)
ovav monitor start

# View pending alerts
ovav monitor status

# View auto-fixed history
ovav monitor history
```

---

## Auto-Fix Configuration

```yaml
# .ovav/config/auto_fix.yaml
auto_fix:
  enabled: true
  runbook_timeout: 60s
  max_retries: 3
  notify_on_success: false  # Don't spam if auto-fixed
  notify_on_failure: true    # Always notify failures
```
