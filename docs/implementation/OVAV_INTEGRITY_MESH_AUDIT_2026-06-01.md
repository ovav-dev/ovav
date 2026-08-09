# OVAV Integrity Mesh — Auditoría y Plan de Implementación

> **Documento de trabajo conjunto Thavren + Braka**
> Fecha: 2026-06-01 | Branch: `task/implementaciones`
> Propósito: Registrar hallazgos, guiar implementación y permitir mejoras incrementales sin perder el rumbo.
> **Regla**: Toda mejora, adición o cambio de rumbo sobre este plan se registra aquí. Este MD es la fuente de verdad operativa de esta iniciativa.

---

## 1. Diagnóstico Inicial

### 1.1 El fallo que nos trajo aquí

En la sesión anterior (L7 Memory Governor Integration), `validate_all` pasaba en verde pero el sistema tenía:

- **10 archivos duplicados** (mismo hash, distinto path) — sin detector
- **20+ imports rotos** apuntando a paths no canónicos — sin auditor
- **Handoff fantasma** desincronizado del runtime — sin sync check
- **Dos write paths al ledger** (`governor.py` + `feedback_loop.py` + `belief_manager.py`) — sin detector de double-writers

**Conclusión**: Los validadores actuales verifican compliance de features pero no integridad arquitectónica. Necesitamos una capa nueva.

### 1.2 Filosofía de la solución

No construir *un validador para cada problema*. Construir un **sistema extensible de integridad arquitectónica** (Integrity Mesh) que:

1. Se gatille automáticamente
2. Crezca con cada nuevo tipo de archivo, fase o conector
3. Quite carga mental a Thavren
4. Detecte fallos antes de que detengan el trabajo

---

## 2. Hallazgos de Auditoría Completa

### 2.1 🔴 Grado 1 — Roturas o Riesgo Inminente

| # | Hallazgo | Ubicación | Impacto |
|---|---|---|---|
| **H1** | Error de sintaxis YAML | `.ovav/registry/surface_validator_map.yaml:144` | La sección `.ovav/evaluation/` tiene 3 espacios de indentación (necesita 2). Rompe parsers YAML. |
| **H3** | `validate_all` cobertura insuficiente | `tools/validators/validate_all.py` | Solo 12 de 47 validadores están incluidos. El sistema dice "OK" mientras tiene deuda arquitectónica invisible. |

### 2.2 🟡 Grado 2 — Frágil o No Automático

| # | Hallazgo | Evidencia |
|---|---|---|
| **H4** | 86 harnesses no auto-triggereados | De 119 `check_*.py` en `tools/harnesses/`, solo ~33 están en `auto_triggers.yaml`. El resto existe pero nunca se ejecuta automáticamente. |
| **H5** | Sin verificación de frescura de contratos | `current_authority_contract.yaml`, `permission_authority.json` y demás contratos no tienen auditor de mtime/checksum. |
| **H6** | `handoff_protocol.yaml` es un stub | 1 sola línea. El protocolo de handoff no tiene especificación real. |
| **H7** | 16 TODOs en código crítico | `check_todo_progress_runtime.py`, `harness_index.py`, y varios builds referencian paths de artefactos hardcodeados con prefijos `S41_`. |
| **H8** | Validadores existen pero no en `validate_all` | 35 de 47 `check_*.py` no están en la suite principal. Son invisibles a menos que se ejecuten manualmente. |

### 2.3 ⚫ Grado 3 — No Existe

| # | Qué falta | Por qué es necesario |
|---|---|---|
| **H9** | Validador de integridad canónica | Detectar archivos duplicados (mismo hash, distinto path) e imports a paths no canónicos. |
| **H10** | Sync check runtime ↔ handoff | Verificar que el handoff refleje el estado real del runtime y viceversa. |
| **H12** | Detector de drift registry ↔ filesystem | Los registries YAML declaran cosas que el filesystem puede no reflejar. Sin reconciliación automática. |
| **H13** | Validador de cobertura | ¿Están todos los validadores existentes corriendo en algún pipeline? ¿O hay checkers huérfanos? |
| **H14** | Auditor de frescura de contratos | Checksum + mtime de cada contrato. Alerta si un contrato requerido se editó sin registro. |
| **H15** | Validador de YAML/JSON sintaxis | `surface_validator_map.yaml` tiene error y nadie lo detectó. Todos los YAML/JSON del repo deberían validarse. |

---

## 3. Inventario Actual

### 3.1 Superficies con cobertura (surface_validator_map)

| Superficie | Validadores asignados | Estado |
|---|---|---|
| `.opencode/` | 4 | ✅ Bien |
| `tools/` | 4 | ✅ Bien |
| `.ovav/policy/` | 3 | ✅ Bien |
| `.ovav/service_areas/` | 3 | ✅ Bien |
| `docs/` | 2 | ✅ Bien |
| `config/` | 2 | ✅ Bien |
| `registry/` | 3 | ✅ Bien |
| `schemas/` | 2 | ✅ Bien |
| `tests/` | 2 | ✅ Bien |
| `.ovav/laws/` | 1 | ✅ Bien |
| `.ovav/evaluation/` | 4 | ❌ YAML roto |

### 3.2 Contadores rápidos

```
Validadores check_*.py:          47
Validadores en validate_all:     ~12
Harnesses check_*.py:            119
Harnesses auto-triggereados:     ~33
Contratos en contract_manifest:  12
Superficies mapeadas:            11
Errores de sintaxis YAML:        1
TODOs en código crítico:         16
```

---

## 4. Plan de Ataque — Fases

### Fase 0: Parche de emergencia (YA) ✅ COMPLETA
- [x] **F0.1** — Reparar sintaxis YAML en `surface_validator_map.yaml` (2 errores: línea 144 `.ovav/evaluation/` + línea 207 `evaluation_layer`)
- [x] **F0.2** — Correr validate_all post-fix y verificar que no haya nuevas roturas

### Fase 1: Integridad canónica (núcleo del Integrity Mesh) ✅ COMPLETA
- [x] **F1.1** — Crear `tools/validators/check_canonical_integrity.py`
  - Detector de archivos duplicados (SHA256) — funcional, sin falsos positivos
  - Auditor de imports canónicos — funcional, filtrado para repo-local solamente (stdlib y 3rd-party ignorados)
  - Detector de paths no canónicos — funcional (harness→memory cross-refs)
- [x] **F1.2** — Agregado a `validate_all.py`
- [x] **F1.3** — Agregado a `auto_triggers.yaml` → `before_implementation` + entrada propia con fallback
- [x] **F1.4** — Agregado a `surface_validator_map.yaml` para superficies `tools/` y `registry/`
- [x] **Verificación**: validate_all ahora detecta 3 imports rotos reales (antes invisibles)

**Hallazgos detectados por el nuevo validador:**
1. `tools.install` — importado por `check_s88_deploy_config_governance.py` — directorio sin `__init__.py`

### Fase 3: Wiring — Automatizar lo que ya existe ✅ COMPLETA
- [x] **F3.1** — Auditoría de 86 harnesses no cableados → clasificación completa:
  - **18 ACTIVE_CRITICAL** — wire inmediato (protocol, context, evidence, git, handoff, memory, install, rollback)
  - **32 ACTIVE_USEFUL** — wire selectivo (activate/deactivate, harness reuse, hardening, session, probes)
  - **31 HISTORICAL** — build-specific (build1-build16), marcar historical_only
  - **32 UNCLEAR** — necesita revisión humana (opencode surfaces, dual profile, e2e, signals)
- [x] **F3.2** — Cableados 11 harnesses críticos a `auto_triggers.yaml`:
  - `before_implementation`: check_context_budget_task_router, check_delegation_runtime, check_memory_firewall
  - `before_apply`: check_rollback_protocol, check_protocol_circuit_breaker
  - `before_close`: check_evidence_stability, check_handoff_runtime_ux, check_context_compaction_runtime, check_session_continuity
  - `after_git_stage`: check_git_stage_intelligence, check_git_change_manifest
- [x] **F3.3** — Creado `check_validator_coverage.py`:
  - Mide % de validadores/harnesses corriendo en validate_all, auto_triggers y surface_map
  - Umbrales actuales: validate_all≥10%, auto_triggers≥15%, surface_map≥10%
  - Previene regresión: alerta si la cobertura baja
  - Cableado a validate_all + auto_triggers + surface_map

**Resultado**: auto_triggers pasó de 6 eventos a 10 eventos con cobertura ampliada.

### Fase 4: Sincronización y Frescura ✅ COMPLETA
- [x] **F4.1** — Creado `check_registry_drift.py`
  - Verifica contract_manifest: todos los paths existen
  - Verifica auto_triggers: fallback scripts existen
  - Verifica surface_validator_map: validadores referenciados existen
  - Sampleo de artifact registry
- [x] **F4.2** — Creado `check_contract_freshness.py`
  - Checksum + mtime de 9 contratos requeridos
  - Alerta staleness (>30 días sin modificación)
  - Detecta stubs (<10 bytes)
  - Validación estructural de permission_authority.json
- [x] **F4.3** — Creado `check_handoff_sync.py`
  - Extrae next_work de CURRENT_HANDOFF.md y runtime report
  - Detecta drift entre ambos
  - Detecta handoff ausente o stub
- [x] **Cableado**: validate_all, auto_triggers (before_implementation + before_close), surface_validator_map

**Hallazgos**:
- Registry drift: PASS — sin drift detectado
- Contract freshness: WARN — permission_authority.json usa estructura no estándar
- Handoff sync: DRIFT — handoff dice `l7_memory_governor_integration`, runtime dice `S33` (por cambio a Integrity Mesh). Requiere actualización manual.

### Fase 5: YAML/JSON Syntax Gate ✅ COMPLETA
- [x] **F5.1** — Creado `check_config_syntax.py`
  - Valida 250 archivos YAML/JSON en `.ovav/`, `registry/`, `.opencode/`
  - Excluye backups históricos y artifacts
- [x] **F5.2** — Cableado a validate_all + auto_triggers + surface_map
- [x] **Reparaciones**: 7 errores de sintaxis corregidos (3 contratos vivos con YAML roto)

### Fase 6: TODO Debt Tracker ✅ COMPLETA
- [x] **F6.1** — Creado `check_todo_debt.py`
  - Cuenta 20 TODO/FIXME en 7 archivos (bajo umbral de 50)
  - Alerta si el conteo crece (regresión)
- [x] **F6.2** — Cableado a validate_all

---

## 5. Criterio de Orden

El orden de fases no es arbitrario:

1. **F0** primero porque rompe activamente el sistema
2. **F1–F2** son el núcleo — atacan los problemas que ya vivimos
3. **F3** maximiza el retorno cableando lo que ya construimos
4. **F4** previene los próximos fallos de drift
5. **F5** es defensivo — evita regresiones de sintaxis
6. **F6** es higiene — deuda técnica visible

---

## 6. Registro de Cambios

| Fecha | Cambio | Autor |
|---|---|---|
| 2026-06-01 | Documento inicial con auditoría completa y plan de 6 fases | Thavren |
| 2026-06-01 | Fase 0 completa: YAML syntax fix en surface_validator_map.yaml (2 errores) + validate_all OK | Thavren |
| 2026-06-01 | Fase 1 completa: check_canonical_integrity.py creado y cableado (validate_all, auto_triggers, surface_map). Detecta 3 imports rotos reales. | Thavren |
| 2026-06-01 | Fase 3 completa: 113 harnesses clasificados, 11 críticos cableados a auto_triggers, check_validator_coverage.py creado y cableado. | Thavren |
| 2026-06-01 | Fase 4 completa: check_registry_drift.py, check_contract_freshness.py, check_handoff_sync.py creados y cableados. Detectado drift handoff↔runtime. | Thavren |

---

## 7. Notas de Trabajo

> *Esta sección se llena durante la implementación. Cada avance, bloqueo o cambio de dirección se registra aquí con fecha.*

---
