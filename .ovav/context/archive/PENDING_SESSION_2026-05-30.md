# OVAV — Pendientes al cierre de sesión 2026-05-30

## P1 — Naming profesional avanzado en OpenCode
El documento OVAV_PROFESSIONAL_AREAS define labels como `Thavren — Platform Engineering Lead`.
- opencode.json: agregar description con nombre canónico de área
- .opencode/agents/thavren.md: frontmatter description alineada
- Labels visibles deben ser LEAD + área, no solo el nombre corto

## P2 — Eidren identity (mismo patrón LEAD-first)
- .opencode/agents/ovav-research-intelligence.md: invertir identidad
- .opencode/agents/eidren.md: requiere mismo tratamiento que thavren.md
- Eidren debe ser primario, Research Intelligence es el área

## P3 — Estructura de carpetas por área
- Falta boundaries.yaml, capabilities.yaml, lanes.yaml en platform_engineering/
- team/ vacío — falta formalizar squads

## P4 — LockdownAuthority: módulos pendientes de migración
- session_context_guard.py, head_integrity_verifier.py, gate_self_protection.py
- check_host_config_drift.py aún lee BLOCKADE_FILE directamente

## P5 — Regenerar SBOM (.ovav/registry/core_hashes.yaml)
- ~22 archivos core modificados legítimamente
- check_host_config_drift marca intrusiones que son cambios autorizados

## P6 — Model switching mid-session credit exhaustion
- ovav_launch.sh wrapper existe pero requiere usarlo como entry point
- OpenCode no confirmado si soporta fallback_model nativo

## P7 — Archivos legacy por limpiar
- .ovav/host_defense_blockade, posibles integrity_lockdown, exfil_lockdown
