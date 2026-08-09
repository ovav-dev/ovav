# Architecture Lab — OVAV Advanced Design Space

> **Propósito:** Espacio de diseño para criterio arquitectónico avanzado, ideas, proyecciones y discusiones técnicas de alto nivel.
> **No es plan de implementación.** Las ideas aquí discutidas requieren diseño formal, validación de contratos y aprobación explícita antes de tocar runtime.
> **Contraparte:** `IMPLEMENTATION_PLAN.md` es la ruta real — solo contiene lo ya incorporado. Este doc es el laboratorio.

---

## Sesión 2026-06-02/03 — Gobernador ↔ Superficie CLI

### Diagnóstico

OVAV opera como sistema gobernador pero su superficie visible (TAB/@ en OpenCode) depende de archivos de configuración de agente (`.opencode/agents/*.md`) que no son gobernados por la topología canónica. Esto produce una clase de error recurrente:

| Ciclo | Error |
|---|---|
| 1 | Áreas visibles en @, leads ocultos en TAB |
| 2 | Leads duplicados en @, áreas sin control de visibilidad |
| 3 | Inversión completa: áreas en TAB visibles/ocultas sin consistencia con contratos |

**Causa raíz:** Dos fuentes de verdad para la misma superficie:
- `.ovav/topology/governance_rules.yaml` → define `surface_rules.at_mentions.visible`, `tab_selector.visible`
- `.opencode/agents/*.md` → define `hidden: true/false`, `mode: all/subagent`

No hay puente determinístico entre ambas. La edición manual de `hidden` es frágil por diseño.

### Principio arquitectónico

> *"When a problem repeats 3+ times, it is not a bug; it is an architectural defect. Do not patch the symptom: change the architecture."*
> — Behavioral Directive #3

### Principio OVAV-first (incorporado 2026-06-03)

**OVAV no tiene CLI por defecto.** Toda arquitectura se diseña para OVAV. Los CLI (OpenCode, Claude Code, VS Code, PI) son clientes downstream. Los adaptadores proyectan desde `.ovav/source/agents/` hacia el formato que cada CLI necesita.

| Capa | Ubicación | Se edita a mano? |
|---|---|---|
| **OVAV (fuente)** | `.ovav/source/agents/areas/`, `leads/`, `teams/` | ✅ Sí — aquí trabajamos |
| **Adaptador** | `tools/agents/project_opencode.py` | ✅ Sí — uno por CLI |
| **OpenCode (proyección)** | `.opencode/agents/` | ❌ No — generado automático |

Este principio aplica a TODO OVAV: trabajamos en OVAV, pusheamos a los clientes. No al revés.

### Surface Materializer (propuesta de diseño)

Un componente que lea la topología canónica (`governance_rules.yaml` + `area_*.yaml`) y proyecte automáticamente los valores correctos de `hidden` y `mode` en los archivos `.opencode/agents/*.md`.

**Flujo:**
```
governance_rules.yaml (fuente canónica)
  → Surface Materializer (proyección determinística)
    → .opencode/agents/*.md (superficie materializada)
      → check_agent_surface_hierarchy.py (validator guardián)
```

**Inspiración externa (investigación 2026-05-28):**
- **OPA/Rego**: Policy-as-code con motor desacoplado. Las reglas de visibilidad se evalúan en runtime, no se codifican estáticamente.
- **Cedar (Amazon)**: Verificación formal automática de políticas antes de materializar.
- **Kubernetes Admission Controllers**: Webhooks que validan/mutan recursos antes de aplicarlos.

**Requisitos de diseño:**
1. Single source of truth: `governance_rules.yaml`
2. Proyección determinística: mismo input → mismo output siempre
3. Validación pre-commit: el validator existente se convierte en admission controller
4. Rollback: si la proyección rompe algo, revertir a último estado válido

### Estado: PROPUESTA — no implementar sin diseño formal

---

## Criterios heredados de investigación (2026-05-28/29)

### EAL7 como disciplina de diseño
Adopción metodológica de Common Criteria EAL7 (documentada en `docs/research/F1_EAL7_GUIDANCE.md`):
- Diseño formalmente verificado → 5 propiedades matemáticas en `tools/permissions/verify.py`
- Canales encubiertos → `tools/security/exfil_detector.py`
- Gestión de configuración estricta → hash chain en `bootstrap_verifier.py`

### Modelos de permisos globales
Investigación completa en `docs/research/GLOBAL_PERMISSION_MODELS_AND_OVAV_CLASSIFICATION.md`:
- 20+ modelos analizados (Bell-LaPadula, Biba, Clark-Wilson, SELinux MAC, AWS IAM, OPA/Rego, Cedar, BeyondCorp, DAO)
- 159 reglas OVAV reclasificadas en 6 estados reales
- 7 modelos adoptados como arquitectura conceptual

### Decisiones incorporadas
Ver `docs/research/PERMISSION_EVOLUTION_DECISIONS.md`:
- 10 bloques, 139 reglas con ALLOW/DENY
- Orden de implementación F0→F5
- 6 reglas especiales que requieren diseño previo (redaction inteligente, model body switch, adaptive auth, emergent rules, external services tracking, research sandbox)

---

## Propuestas activas (no implementadas)

| Propuesta | Origen | Estado | Bloqueante |
|---|---|---|---|
| **Agent Release Pipeline** | Sesión 2026-06-03 | ✅ INCORPORADO en arquitectura OVAV-first | Ver `AGENT_RELEASE_PIPELINE.md`. 6 capas: Source→Staging→Verify→Review→Release→Monitor. 530+ checks. Security Override para emergencias. La arquitectura de agentes OVAV-first (`.ovav/source/agents/` → `project_opencode.py` → `.opencode/agents/`) implementa el pipeline de release. |
| **Sistema Nervioso Vivo** | Knowledge Compiler P0 | 🟡 DISEÑO COMPLETO — implementación de runtime pendiente | Diseño de arquitectura de 3 capas (sensorial, procesamiento, motora) completado. Pendiente: grafo de conocimiento, pesos hebbianos, activación distribuida. |
| **Credenciales Gobernadas** | Separación cuenta personal vs producto | 🟡 DISEÑO COMPLETO — implementación de runtime pendiente | Diseño de separación de cuentas y ciclo de vida de credenciales completado. Pendiente: Vault Base y migración. |
| **Dashboard Visual OVAV** | Consola en tiempo real | ✅ IMPLEMENTADO | `tools/ovav_dashboard.py` (358 loc) operativo. Muestra estado agregado del sistema. |
| **Research Mesh (Web/Search)** | Búsqueda externa gobernada | ✅ OPERATIVO | Brave + Tavily + DDG + SearXNG. 2,000 búsquedas/mes gratuitas. Cache 24h/7d. Pendiente: sandbox de contenido. |

---

## Agent Release Pipeline — Concepto clave (2026-06-03)

Diseño completo: [`AGENT_RELEASE_PIPELINE.md`](AGENT_RELEASE_PIPELINE.md)

```
CREACIÓN → STAGING → VERIFY → REVIEW → RELEASE → MONITOR
(.ovav)    (OVAV CLI) (530+ chk) (humano)  (CLI ext)  (L7)

OVAV CLI recibe AUTOMÁTICO. CLI externos: usuario decide.
```

### Security Override — canal de emergencia

Flujo paralelo al release normal. Solo Thavren. Solo por seguridad crítica.

| Condición | Normal | Override |
|---|---|---|
| Quién decide instalar | Usuario | OVAV (forzado) |
| Se puede posponer | Sí | No |
| Rollback | A cualquier versión | Solo a versión segura anterior |
| Notificación | Opcional | Obligatoria |
| Requiere aprobación | Verificación automática | Thavren + doble confirmación |

Dispara solo ante: vulnerabilidad confirmada, fuga de datos, inyección viable, o breaking change del CLI host que expone OVAV.

---

---

## Sesión 2026-06-03 — Gobernador de Configuración Distribuida (GCD)

### Visión

OVAV no es solo un governor de agentes AI. Es una **plataforma de distribución de configuraciones gobernadas**. El usuario instala OVAV una vez, y OVAV gobierna, actualiza y despliega configuraciones premium para todo su ecosistema de herramientas.

```
┌──────────────────────────────────────────────────────────────┐
│                     OVAV GCD Pipeline                        │
│                                                              │
│  DESARROLLO          VERIFICACIÓN        DISTRIBUCIÓN        │
│  ──────────          ────────────        ────────────        │
│  OVAV internamente   CI/CD + gates       OVAV Update Engine  │
│  desarrolla y cura   + validadores       envía a usuarios    │
│  configuraciones     + smoke tests       con OVAV instalado  │
│         │                  │                    │            │
│         ▼                  ▼                    ▼            │
│  ┌──────────┐       ┌──────────┐        ┌──────────┐        │
│  │ Config   │       │ Quality  │        │ User     │        │
│  │ Areas    │  ───► │ Gates    │  ───►  │ Launch   │        │
│  │ (curated)│       │ (verify) │        │ (recibe) │        │
│  └──────────┘       └──────────┘        └──────────┘        │
└──────────────────────────────────────────────────────────────┘
```

### Config Areas — Super-sets gobernados por membresía

Cada herramienta soportada por OVAV es un **Config Area** independiente. El acceso depende del tier de membresía del usuario.

| Config Area | Herramienta | Estado actual | Tier |
|---|---|---|---|
| **WezTerm** | Terminal GPU-accelerated | ✅ Gobernado (3 archivos) | Core |
| **Workstation** | Aislamiento de workspace | ✅ Gobernado (1 archivo) | Core |
| **OpenCode** | Superficie de agentes | ✅ Proyección automática | Core |
| **Alacritty** | Terminal alternativa | 🔮 Planeado | Core |
| **Zellij** | Multiplexor terminal | 🔮 Planeado | Studio |
| **Git** | Configuración global + aliases | 🔮 Planeado | Core |
| **Fish** | Shell interactivo | 🔮 Planeado | Studio |
| **Neovim** | Editor | 🔮 Planeado | Studio |
| **Starship** | Prompt | 🔮 Planeado | Studio |
| **Tmux** | Multiplexor alternativo | 🔮 Planeado | Core |

### El pipeline GCD completo

```
1. CURACIÓN (en OVAV)
   config/<tool>/  ← desarrollo interno de configs
   ├── templates/   (base configurable)
   ├── profiles/    (por tier: core, studio, command)
   └── validators/  (check_config_<tool>.py)

2. VERIFICACIÓN (gates)
   ├── Syntax check (config parseable)
   ├── Isolation check (no leak entre workspaces)
   ├── Security check (no credenciales en configs)
   ├── Compatibility check (versión de herramienta)
   └── Smoke test (usuario simulado aplica config)

3. EMPAQUETADO (release)
   ├── .ovav/registry/install_packs.yaml ← define packs por tier
   │   ├── workstation_core_pack      (wezterm + git + fish)
   │   ├── workstation_studio_pack    (core + nvim + zellij)
   │   └── workstation_command_pack   (todo)
   └── Versionado semántico por área

4. DISTRIBUCIÓN (update engine)
   ├── ovav update --check     (verifica updates disponibles)
   ├── ovav update --apply     (backup → apply → verify)
   └── ovav launch             (usuario ve updates al abrir)
```

### Lo que YA existe (base sólida)

| Componente | Archivos | Estado |
|---|---|---|
| **Config Areas** | `config/wezterm/` (3), `config/workstation/` (1) | 2 áreas gobernadas |
| **Deploy Gateway** | `tools/install_gateway/` (14 módulos) | apply, backup, boundaries, config_deploy, deploy_governor, manifest, plan, report, rollback, safety, ux, verify |
| **Install Packs** | `.ovav/registry/install_packs.yaml` (12 packs) | workstation_deploy_pack, opencode_governed_pack, build8_source_local_apply_pack |
| **Update Engine** | `tools/cli/ovav_official_update.py` (16K) | check, apply, backup, rollback |
| **CLI Router** | `tools/cli/router.py` | update, sync, surfaces, setup, recovery |
| **Surface Manager** | `tools/cli/ovav_surface_manager.py` | status, repair-plan para 7 superficies |
| **Plan Artifacts** | `tools/cli/ovav_plan_artifacts.py` | setup, sync, security, recovery, update |
| **Agent Projection** | `tools/agents/project_opencode.py` | .ovav/source/agents/ → .opencode/agents/ |
| **Permission Materializer** | `tools/permissions/materialize.py` | propaga políticas canónicas a targets |
| ~~State Sync Engine~~ | `tools/agent_runtime/state_sync_engine.py` | ELIMINADO — git HEAD es fuente de verdad |

### Lo que falta (gaps → próximos pasos)

| Gap | Descripción | Prioridad |
|---|---|---|
| **Membership/Tier system** | No existe modelo de tiers (Core/Studio/Command). Los packs actuales no diferencian por membresía. | Alta |
| **Config Area SDK** | No hay scaffold para crear nuevas áreas rápido. Cada tool requiere estructura manual. | Alta |
| **Launch UX para updates** | El cockpit actual no muestra "updates disponibles" al abrir. | Alta |
| **Más Config Areas** | Solo WezTerm y Workstation. Faltan: git, fish, nvim, alacritty, zellij, starship. | Media |
| **CI/CD para configs** | No hay pipeline automático que pruebe configs antes de release. | Media |
| **Rollback por área** | Rollback actual es del sistema completo. Debería ser por Config Area. | Media |
| **Telemetría de uso** | Sin datos de qué configs usa el usuario, no hay feedback loop. | Baja |

### Diseño del Membership/Tier System

```
Tier Core (gratuito)
  ├── WezTerm base (colores, fuentes, layout)
  ├── Git (config global, aliases esenciales)
  ├── Fish (prompt base, aliases)
  └── OpenCode (superficie de agentes)

Tier Studio (premium)
  ├── Todo Core +
  ├── Neovim (config completa + LSP + plugins curados)
  ├── Zellij (layout templates + keybindings)
  ├── Starship (prompt premium)
  └── Temas visuales adicionales

Tier Command (enterprise)
  ├── Todo Studio +
  ├── Configs multi-workstation (sync entre máquinas)
  ├── Custom profiles (usuario define variantes)
  ├── Early access a nuevas áreas
  └── Soporte prioritario
```

### Principios GCD

1. **Source-local first**: Las configs se desarrollan y prueban source-local. Solo se distribuyen tras verificación.
2. **Fail-closed**: Si una config falla validación, no se distribuye. El usuario mantiene su versión anterior.
3. **Backup siempre**: Cada apply crea backup. Rollback es un comando.
4. **Idempotencia**: Aplicar la misma versión dos veces = mismo resultado.
5. **Separación de concerns**: Configs de herramienta ≠ políticas de OVAV ≠ runtime de OVAV.
6. **El usuario decide**: OVAV sugiere updates, el usuario acepta o pospone.

### Estado: PROPUESTA — requiere diseño formal de Membership System + Config Area SDK

Ver brief completo: [`RESEARCH_2026_AGENT_GOVERNANCE.md`](RESEARCH_2026_AGENT_GOVERNANCE.md)

**Hallazgos clave:**
- **Microsoft AGT** (Abr 2026): Policy engine OPA/Cedar, <0.1ms, zero-trust identity — lo más cercano a OVAV
- **MCP/A2A/ACP**: Ecosistema de protocolos agente-agente. Spec conjunta Q3 2026
- **Runtime verification**: La industria converge en modelo de capas (identity → isolation → policy → audit)
- **OVAV mantiene 6 capacidades exclusivas**: governor de origen, separación de áreas, Model Body Router, Session Capsule, fail-closed universal, Criterion Compiler
- **Riesgo**: Microsoft AGT podría evolucionar hacia governor completo
- **Oportunidad**: Adoptar Cedar, A2A, SPIFFE y tamper-evident logs para mantener ventaja

---

## Reglas del laboratorio

1. Las ideas aquí no son plan. Son criterio en formación.
2. Toda idea requiere: diseño formal → validación de contratos → aprobación explícita → implementación.
3. Si una idea se implementa, se mueve a `IMPLEMENTATION_PLAN.md` y se elimina de aquí.
4. Si una idea se descarta, se marca como `ARCHIVED` con la razón.
5. Investigación externa → digestión interna → propuesta OVAV. No copiar sin mejorar.
