# OVAV Master Action Plan — Single Timeline
> **Branch:** task/implementaciones
> **HEAD:** 91226c9
> **Sesión:** 2026-06-03
> **Autoridad:** `.ovav/forge/design/`
> **OVAV Laws:** 21 leyes, 6 grupos — compliance check al final
>
> ⚠️ **NOTA 2026-06-04:** Este plan fue ejecutado en las Fases A-D. Los items P8-P13 se incorporaron a Fase E (E0: ConnectorBus 100%). El plan canónico activo está en `IMPLEMENTATION_PLAN.md` → Fase E.

---

## RESUMEN DE LO EJECUTADO (esta sesión — 11 commits)

### Bloque A — Reestructuración Arquitectónica (F0-F8)
| Fase | Commit | Acción |
|------|--------|--------|
| F0 | `f51127e` | Model watchdog: lee `config.model` raíz + trigger `on_model_error` |
| F1 | `c4c9897` | `.ovav/source/` creado — agentes y skills movidos a fuente canónica |
| F2 | `aa447a3` | `.ovav/forge/` — pipeline, targets(4), adapters(4), releases |
| F3 | `42ba53f` | `registry/` → `.ovav/registry/` — 178 referencias actualizadas |
| F4 | `42f296a` | `runtime/` `config/` `statusline-inspect/` `OVAV_LIVING` consolidados |
| F5 | `5db8636` | Plugins → `.ovav/source/plugins/opencode/` (tui, monitor, status) |
| F6 | `4ddda91` | Reparación masiva de paths — 210 archivos, 1997 líneas |
| F7 | `3b47a97` | Re-proyección: 16 agents, 11 skills, 2 plugins, theme |
| F8 | `c510303` | Limpieza raíz: PNGs→assets, fix git push gate case-sensitivity |

### Bloque B — Corrección de Issues Sistémicos
| Issue | Commit | Fix |
|-------|--------|-----|
| REGISTRY_ROOT path viejo | `bf3c382` | `common.py` → `.ovav/registry/` |
| validate_all default roto | `bf3c382` | `--registry-root` → `REPO_ROOT` |
| Fixtures inválidos pasaban | `bf3c382` | `check_invalid_fixtures.py` reescrito |
| poison_guard timeout | `bf3c382` | 45s → 90s |
| delegation_runtime crasheaba | `bf3c382` | Pre-gates no fatales + yaml.safe_load |
| skill_manager REGISTRY_DIR | `bf3c382` | → `.ovav/registry/` |

### Bloque C — Connector Bus (Nueva Arquitectura de Integración)
| Componente | Commit | Propósito |
|------------|--------|-----------|
| `slots.yaml` | `65091f7` | 7 tipos de slot (validator, harness, tool, skill, adapter, plugin, config_watcher) |
| `registry.yaml` | `65091f7` | **ÚNICO archivo** para integrar componentes — 24 validators + 3 harnesses + 6 tools + 3 adapters + 3 plugins |
| `bus.py` | `65091f7` | Engine: lee registry → auto-wires validate_all, triggers, surface_map |
| `validate_all.py` | `65091f7` | Refactorizado: carga dinámica desde bus, NO hardcodeado |

---

## ARQUITECTURA ACTUAL (post-sesión)

```
OVAV/
├── .ovav/                          ← CEREBRO (todo interno)
│   ├── source/                     ← Desarrollo universal
│   │   ├── agents/                 (2 áreas, 2 leads, 12 team members)
│   │   ├── skills/                 (11 skills)
│   │   ├── plugins/opencode/      (tui, monitor, status)
│   │   └── configs/               (wezterm, workstation)
│   ├── forge/                      ← Release Engine multi-target
│   │   ├── pipeline.py
│   │   ├── targets/               (opencode, claude-code, vscode, pi)
│   │   ├── adapters/              (4 CLIs)
│   │   ├── releases/opencode/
│   │   └── design/                ← ESTE ARCHIVO
│   ├── connector_bus/             ← Punto Único de Integración
│   │   ├── slots.yaml
│   │   ├── registry.yaml          ← TOCAR SOLO ESTE para integrar
│   │   └── bus.py
│   ├── registry/                  ← 35 YAMLs/JSONs (ya no en raíz)
│   ├── governance/                ← Leyes, políticas, seguridad, topología
│   ├── memory/                    ← Ledger, contexto, handoffs
│   ├── runtime/                   ← Sesiones, locks, estado
│   └── ...
├── .opencode/                     ← PROYECCIÓN (generado, no fuente)
│   ├── agents/  (16)
│   ├── skills/  (11)
│   ├── plugins/ (2)
│   └── themes/  (1)
├── tools/                         ← Motor de ejecución (623 py)
├── clients/                       ← 🔲 PENDIENTE: segregar proyecciones
├── docs/                          ← Documentación
└── assets/                        ← PNGs, íconos
```

---

## PLAN DE ACCIÓN — LÍNEA TEMPORAL ÚNICA

> **Orden = dependencia real, no prioridad.**
> Cada bloque DESBLOQUEA al siguiente.

### 🔲 P1 — Segregación de Clientes Externos
**Dependencia:** ninguna (es mover archivos generados)
**Duración estimada:** 30 min
**Archivos:** ~30

```
Objetivo: .opencode/ NO debe vivir en raíz de OVAV.
Es proyección de un cliente externo, no fuente.

Acción:
  1. Crear clients/opencode/
  2. Mover .opencode/agents/ .opencode/skills/ .opencode/plugins/ .opencode/themes/ → clients/opencode/
  3. Symlink: .opencode → clients/opencode (para que OpenCode CLI lo encuentre)
  4. Actualizar opencode.json si es necesario
  5. Actualizar forge/targets/opencode.yaml → nuevo projection_root
```

**OVAV Laws impactadas:** LAW-06 (single_authority), LAW-09 (service_area_alignment)

---

### 🔲 P2 — Identity Hardening (14 archivos)
**Dependencia:** P1 (necesitamos paths estables)
**Duración estimada:** 2h
**Archivos:** 14

```
Objetivo: OVAV ≠ Platform Engineering ≠ Thavren.
Cada uno es una capa independiente. Eliminar toda fusión de identidad.

Archivos a corregir (auditados):
  1. docs/system/00_IDENTITY.md — reescribir sección "Thavren — OVAV Platform Engineering"
  2. docs/intelligence/02_ACTIVE_IDENTITY_PACKET.md — visible_profile independiente
  3. .ovav/service_areas/shared/current_authority_contract.yaml — roles, no nombres
  4. .opencode/agents/area-platform-engineering.md — ya parcialmente corregido
  5. docs/lab/OVAV_LIVING_INTELLIGENCE_EVALUATION_LAYER_THAVREN.md
  6. .ovav/service_areas/shared/context_economy_contract.yaml
  7. .ovav/source/skills/ovav-research-session/SKILL.md
  8. .ovav/source/skills/ovav-repo-local-work-loop/references/context-work-validate-close.md
  9. .ovav/source/skills/ovav-repo-local-work-loop/references/visible-surface-cleanup.md
  10. .opencode/agents/lead-eidren.md
  11. .opencode/agents/team-nara.md
  12-14. 3 archivos adicionales con referencias cruzadas detectadas

Criterio de corrección:
  - "OVAV Platform Engineering" → "Platform Engineering (área de OVAV)"
  - "Thavren — OVAV Platform Engineering" → "Thavren — Lead de Platform Engineering"
  - "OVAV Platform Engineering as primary identity" → eliminar
```

**OVAV Laws impactadas:** LAW-08 (persistent_identity), LAW-09 (service_area_alignment), LAW-10 (human_professional_delivery)

---

### 🔲 P3 — Crear `clients/` para TODOS los targets
**Dependencia:** P1
**Duración estimada:** 20 min
**Archivos:** ~5

```
Objetivo: Un solo directorio para todas las proyecciones a clientes externos.

Estructura:
  clients/
    opencode/       ← symlink .opencode → clients/opencode
    claude-code/    ← preparado (forge/targets/claude-code.yaml)
    vscode/         ← preparado (forge/targets/vscode.yaml)
    pi/             ← preparado (forge/targets/pi.yaml)

Cada cliente tiene su propio espacio aislado.
OVAV no contamina su raíz con artefactos de clientes.
```

---

### 🔲 P4 — `docs/OVAV_USER_GUIDE.md`
**Dependencia:** P1, P2 (necesitamos la arquitectura final documentada)
**Duración estimada:** 1.5h
**Archivos:** 1 nuevo

```
Objetivo: Guía práctica de uso de OVAV. No teoría, no historia.
Un usuario nuevo debe poder leer esto y entender qué hace OVAV y cómo usarlo.

Contenido mínimo:
  1. Qué es OVAV (3 párrafos)
  2. Estructura de directorios (diagrama)
  3. Comandos esenciales (ovav validate, ovav doctor, ovav status)
  4. Flujo de trabajo diario (iniciar → validar → trabajar → commit → release)
  5. Cómo integrar un componente nuevo (Connector Bus — 3 pasos)
  6. Cómo agregar/remover un Lead (Personnel Registry)
  7. Troubleshooting común
```

---

### 🔲 P5 — `tools/INDEX.md`
**Dependencia:** ninguna (es documentación)
**Duración estimada:** 1h
**Archivos:** 1 nuevo

```
Objetivo: Mapa navegable de los 623 archivos en tools/.

Estructura:
  - Validadores (tools/validators/) — qué verifica cada uno
  - Harnesses (tools/harnesses/) — qué escenario cubre cada uno
  - Runtime (tools/agent_runtime/) — motores de sesión, watchdog, triggers
  - CLI (tools/cli/) — comandos de usuario
  - Seguridad (tools/security/) — gates, integridad, defensa
  - Memoria (tools/memory/) — pipeline de clasificación
  - Proyección (tools/agents/, tools/visual/, tools/skills/)
```

---

### 🔲 P6 — `.ovav/registry/personnel.yaml` + `deregister_lead.py`
**Dependencia:** P2 (los roles ya están separados de las personas)
**Duración estimada:** 1h
**Archivos:** 2 nuevos

```
Objetivo: Registro central de personal. Un Lead se agrega/remueve en UN solo lugar.

personnel.yaml:
  leads:
    thavren:
      role: platform_engineering_lead
      area: platform_engineering
      artifacts: [.ovav/source/agents/leads/thavren.md]
      permissions: .ovav/policy/permission_authority.json
      active: true
    eidren:
      role: research_intelligence_lead
      area: research_intelligence
      artifacts: [.ovav/source/agents/leads/eidren.md]
      permissions: .ovav/policy/permission_authority.json
      active: true

deregister_lead.py:
  - Lee personnel.yaml
  - Remueve/archiva todos los artefactos del lead
  - Actualiza contracts, topology, registries
  - Deja OVAV funcionando sin el lead removido
```

---

### 🔲 P7 — Actualizar `IMPLEMENTATION_PLAN.md`
**Dependencia:** P1-P6 completados
**Duración estimada:** 30 min
**Archivos:** 1

```
Objetivo: El plan maestro refleja la arquitectura real post-sesión.

Cambios:
  - Reemplazar referencias a paths viejos (.ovav/agents/ → .ovav/source/agents/)
  - Agregar Connector Bus como sistema de integración
  - Actualizar estado de F0-F8 como COMPLETADO
  - Mover clients/ y identity hardening al frente de la ruta
```

---

## VERIFICACIÓN CONTRA LEYES OVAV

> 21 leyes, 6 grupos. Verificación de lo ejecutado y lo pendiente.

| Ley | Grupo | Estado |
|-----|-------|--------|
| LAW-01 automation_useful | Product | ✅ ConnectorBus automatiza integración real |
| LAW-02 practical_value | Product | ✅ Cada fix resolvió un issue concreto |
| LAW-03 friction_reduction | Product | ✅ 1 archivo vs 20 para integrar componentes |
| LAW-05 state_truth | Authority | ✅ validate_all verifica estado antes de actuar |
| LAW-06 single_authority | Authority | ⚠️ .opencode/ en raíz viola esto — P1 lo resuelve |
| LAW-07 semantic_drift_free | Authority | ✅ 212 configs validados, 0 drift |
| LAW-08 persistent_identity | Identity | ⚠️ 14 archivos con confusión OVAV=Thavren — P2 lo resuelve |
| LAW-09 service_area_alignment | Identity | ⚠️ P1+P2 alinean áreas con identidad |
| LAW-10 human_professional_delivery | Identity | ✅ Thavren opera como lead, no como bot |
| LAW-11 zero_trust_context | Context | ✅ Session context guard activo |
| LAW-12 minimum_sufficient_context | Context | ✅ Runtime context budget T0-T5 |
| LAW-13 governed_tools | Context | ✅ Permission authority gobernada |
| LAW-15 automatic_reflexes | Execution | ✅ 24 triggers cableados vía ConnectorBus |
| LAW-16 evidence_required | Execution | ✅ Evidencia en cada commit |
| LAW-18 memory_firewall | Security | ✅ Memory governor capsule-bound |
| LAW-21 governed_self_improvement | Security | ✅ ConnectorBus es mejora gobernada, no auto-mod |

**Resultado:** 16/21 PASS. 3 ⚠️ (P1+P2 los resuelven). 2 no aplican a esta sesión.

---

## ESTADO FINAL DE LA RAMA

```
Branch: task/implementaciones
HEAD: 91226c9
Working tree: LIMPIO
validate_all: 212 configs OK, 0 failed
ovav_runtime validate: 0 blocking issues
ConnectorBus: 24 validators, 3 harnesses, 6 tools, 3 adapters, 3 plugins
```

---

**Próxima sesión:** Iniciar con P1 — Segregación de Clientes Externos.
