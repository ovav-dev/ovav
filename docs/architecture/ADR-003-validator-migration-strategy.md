# ADR-003: OVAV Monitoring & Auto-Remediation System (OMARS)

**Date:** 2026-08-09
**Status:** Active
**Decider:** Thavren

## Context

El sistema de validators (81 validators) tenía problemas fundamentales:

1. **Duplicación**: Muchos validators medían lo mismo (e.g., `merge_readiness` + `release_gate`)
2. **Fragilidad**: Comparaban archivos uno a uno, cualquier espacio = FAIL
3. **Bloqueo**: Fallas bloqueaban trabajo aunque no eran críticas
4. **OWS overlap**: OWS ya hacía hygiene checks pero validators los重复
5. **Sin auto-fix**: El humano tenía que手动 fixear todo

## Decision

Crear OMARS: OVAV Monitoring & Auto-Remediation System

### Arquitectura

```
Monitor → Alert → Dispatcher → [AutoFix | Human | Archive]
```

**Monitores implementados:**
- `HygieneMonitor` — Wraps OWS hygiene scan (10 checks)
- `AgentProjectionMonitor` — Count + timestamp checks para agent sync

**Sistema de alertas:**
- Cola en `.ovav/runtime/alerts/{queue,auto_fixed,acknowledged,archived}.jsonl`
- 4 niveles: CRIT (bloquea) / ERROR (auto-fix) / WARN / INFO

**Auto-fix runbooks:**
- `fix_generated_drift` — `ovav project sync`
- `fix_stale_locks` — Expira locks >24h
- `fix_agent_projection` — `ovav convert --agents`
- `fix_sbom_baseline` — `sbom_regen` + baseline
- `fix_runtime_integrity` — Crea/actualiza baseline

### Validator Deprecation

Reducido de 81 → 71 validators en DefaultRegistry.

**Eliminados:**
- `ContextFirewallV2` (duplicado)
- `MergeReadiness` → OMARS HygieneMonitor
- `ReleaseGate` → OMARS HygieneMonitor
- `HandoffSync` → OMARS HygieneMonitor
- `HeadIntegrity` → Eliminado (hash drift normal)
- `ArchitectureGuardian` → WARN only
- `CapsChronosAlignment` → WARN only (stale no es error)
- `CrossTargetConsistency` → OMARS AgentProjectionMonitor

## Consequences

**Positivo:**
- Alertas no bloquean trabajo
- Auto-fix ejecuta en 60s
- count-based checks no se rompen por espacios
- OWS es source de verdad para hygiene

**Negativo:**
- Validator count reducido
- Algunos checks ahora son WARN en lugar de FAIL

## Migration Path

**Phase 1** (Actual): Dual operation
- OMARS corriendo con monitors básicos
- Validators aún registran (71 ahora vs 81 antes)

**Phase 2** (Post-merge): Soft migration
- Más validators migrados a monitors
- Validators emiten WARN en lugar de FAIL

**Phase 3** (Futuro): Monitor-only
- OMARS es el único source de verdad
- `ovav validate` → `ovav monitor`

## Commands

```bash
# Run monitors manually
ovav monitor run

# Check status
ovav monitor status

# View history
ovav monitor history

# Start background loop
ovav monitor start
```
