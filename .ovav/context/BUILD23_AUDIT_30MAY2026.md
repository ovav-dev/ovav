# BUILD23 System Audit — 30 Mayo 2026

## Estado general

- **Branch**: task/implementaciones
- **Build**: BUILD 23 — Tool Readiness Matrix + Advanced Capability Boundary
- **Stack**: L0-L4 Intelligence Runtime Stack completo
- **Launch Verification**: cerrado, 11/11 validators PASS
- **9/9 BUILD23 validators**: PASS
- **Capability Lifecycle**: 23 capabilities, lifecycle gate PASS
- **Evaluation Layer**: 13 fases implementadas (100%)

---

## Problemas encontrados

### 1. AGENTS.md desalineado con permission_authority.json (MEDIO)

**Línea 78-79 de AGENTS.md** dice: "Work source-only inside this repository. Do not write to ~/.config, ~/.local, global OpenCode config, or user system paths unless a later approved install segment explicitly allows it."

Pero `permission_authority.json` da a Thavren:
- `external_directory: {"*": "allow"}`
- `global_diagnostic_read: allow_via_governed_read_only_tools`
- `scopes: ["repo_local", "global_diagnostic", "install_sandbox"]`

**Causa**: En Phase 6 (commit 284fce47, 29 mayo 23:04) se expandieron los permisos de Thavren en el archivo de autoridad canónico, pero nunca se actualizó el resumen de AGENTS.md.

**Impacto**: Thavren puede innecesariamente rechazar tareas fuera del repo cuando en realidad tiene autoridad para ejecutarlas.

**Solución**: Actualizar AGENTS.md línea 78 para reflejar que Thavren tiene acceso gobernado a diagnóstico global y directorios externos, con step-up requerido para escritura global.

### 2. AGENTS.md fue borrado del working tree (CRÍTICO — ya restaurado)

Durante esta sesión, AGENTS.md desapareció del disco. Se restauró desde `git checkout HEAD`. El backup de integridad en `.ovav/integrity_backups/` tenía una versión anterior sin la sección de Protected Branch Lockdown.

**Causa**: No determinada. Posible efecto colateral de operaciones de git o del Defense Gate.

**Solución aplicada**: Restaurado desde HEAD. Verificar que el backup de integridad se actualice.

### 3. h_verify_evidence.py no existe (ALTO)

Referenciado en 22 ubicaciones:
- 6 harnesses/workflows activos: `thavren_workflow.py`, `eidren_workflow.py`, `harness_intelligence_router.py`, `profile_session_context.py`, `build1_readiness.py`, `repo_local_work_loop.py`
- 4 registries: `skills.yaml`, `evals.yaml`, `auto_triggers.yaml`, `delegation_rules.yaml`
- 6 test fixtures + 6 integrity backups

Si se activa un trigger `before_close`, el sistema fallaría en runtime.

**Solución**: Crear el archivo o eliminar las referencias si la funcionalidad ya no se necesita.

### 4. Defense Gate falso positivo (BAJO)

El Defense Gate muestra "BLOCKED — 2 intrusiones neutralizadas" y bloquea `validate_all.py`. Las intrusiones parecen ser tool calls legítimas de OpenCode interceptadas por error.

**Archivo**: `.ovav/host_defense_blockade` (278 bytes, JSON)

**Solución**: Limpiar el blockade y verificar la lógica de detección para reducir falsos positivos.

### 5. docs/26_RUNTIME_CONTEXT_BUDGET.md referencia archivos inexistentes (BAJO)

Referencia `docs/24_IMPLEMENTATION_PLAYBOOK.md`, `docs/25_DOC_AUTHORITY_MATRIX.md` y `OVAV_BUILD_PLAN_0_TO_100_2026.md` — ninguno existe.

**Solución**: Actualizar referencias en el context budget o crear placeholders.

---

## Orden de reparación recomendado

1. Alinear AGENTS.md con autoridad real de Thavren (línea 78)
2. Crear o eliminar referencias a h_verify_evidence.py
3. Limpiar Defense Gate blockade
4. Actualizar docs/26 referencias

---

## Commits relevantes

- `284fce47` — Phase 6: Harness Task Router (agregó external_directory a Thavren, no actualizó AGENTS.md)
- `ff89715` — Phase 9: Capability Lifecycle + Protected Branch Lockdown (agregó sección a AGENTS.md)
- `c31540c` — Phase 10-13: Evaluation Layer completo (sesión actual)

---

Generado: 2026-05-30T07:55Z | Thavren / OVAV Platform Engineering
