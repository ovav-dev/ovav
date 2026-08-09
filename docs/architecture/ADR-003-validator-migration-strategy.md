# ADR-003: Validator Migration Python → Go

**Date:** 2026-06-17  
**Status:** In Progress  
**Decider:** Thavren

## Context

79 total validators. Migration from Python to Go for performance and OS-native execution.

## Progress

| Batch | Validators | Status |
|-------|-----------|--------|
| Batch 1 | secrets_hygiene, exfil_patterns, supply_chain, protected_branch, workspace_safety | ✅ Done |
| Batch 2 | git_push, permission_drift, contract_freshness, runtime_integrity | ✅ Done |
| Batch 3 | config_syntax, config_integrity, agent_governance, plugin_security, merge_readiness, release_gate, living_integrity, registry_drift, runtime_wiring, handoff_sync, zero_trust, single_authority | ✅ Done |
| Batch 4 | architecture_compliance (F1), contract_enforcement (F2), architecture_governance (F3) | ✅ Done |
| Batch 5 | context_firewall, credential_governance, network_hardening, security_hardening, advanced_hardening, install_verification | ✅ Merged |

**Current:** 30/79 (38%) migrated.

## Consequences

- Python validators remain as operational governance layer
- Go validators run as native binaries (no Python runtime needed for product)
- Performance improvement: sub-ms validation vs 50-200ms Python
