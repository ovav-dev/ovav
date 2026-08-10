# Validator Deprecation Map

## Overview

**2026-08-09**: Reducido de 81 → 71 validators. Sistema OMARS (OVAV Monitoring & Auto-Remediation) es ahora el primario.

## Eliminados de DefaultRegistry

| Validator | Razón | Reemplazado por |
|-----------|-------|----------------|
| `context_firewall_v2` | Duplicado de ContextFirewall | ContextFirewall |
| `merge_readiness` | OWS hygiene ya lo hace | HygieneMonitor |
| `release_gate` | Redundante con merge_readiness | HygieneMonitor |
| `handoff_sync` | OWS hygiene ya lo hace | HygieneMonitor |
| `head_integrity` | Hash drift normal entre sesiones | Eliminado |
| `architecture_guardian` | Estructura de dirs no crítica | Eliminado |
| `caps_chronos_alignment` | Stale caps no es error | Eliminado |
| `cross_target_consistency` | Compara archivos frágilmente | AgentProjectionMonitor |
| `todo_debt` | Solo cuenta TODOs, no crítico | Eliminado |

## Validadores que permanecen (71)

### Security-Critical (BLOCKING)
- `secrets_hygiene` — Secrets en código
- `exfil_patterns` — Patterns de exfiltración
- `supply_chain` — SBOM integrity
- `protected_branch` — Rama protegida
- `credential_governance` — Vault y credenciales
- `security_hardening` — F4 bash commands
- `agent_permission_invariants` — Lead↔area alignment

### Warning-Only (WARN)
- `contract_freshness` — Contract hash
- `single_authority` — caps.yaml + git HEAD
- `agent_runtime_enforcement` — Skills loaded
- `tool_readiness` — Skill matrix
- `context_economy` — Tiers T0-T5 budget

### Info-Only (INFO)
- `agent_surface_hierarchy` — Count check
- `caps_schema` — YAML válido
- `registry_validator` — Files exist
- `todo_debt` — Deprecated

## OMARS: Sistema Primario de Monitoreo

```bash
# Status
ovav monitor status

# Run all monitors
ovav monitor run

# History
ovav monitor history
```

### Monitores

| Monitor | Checks | Auto-fix |
|---------|--------|----------|
| `HygieneMonitor` | 10 OWS checks | fix_generated_drift, fix_stale_locks |
| `AgentProjectionMonitor` | Count + timestamp | fix_agent_projection, fix_sbom_baseline |

### Niveles de Alerta

| Nivel | Comportamiento |
|-------|----------------|
| CRIT | Bloquea, no auto-fix |
| ERROR | Auto-fix en 60s |
| WARN | Solo log |
| INFO | Solo log |

## Deprecated ADR

Este documento reemplaza ADR-003-v1 (validator migration Python→Go).
La arquitectura ahora es: **OMARS + validators críticos**.
