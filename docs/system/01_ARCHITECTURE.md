# OVAV Architecture — Capas del Sistema Vivo

## Visión General

OVAV opera como un **Professional AI Workstation Governor** que evoluciona hacia un **Collective Intelligence System** gobernado, con capas de inteligencia, gobernanza, ejecución, evidencia y memoria viva.

```text
                    USUARIO
                       │
                       ▼
┌──────────────────────────────────────────────┐
│            CLI COCKPIT (curses TUI)           │
│     Launch · Tailor · Preview · Recovery      │
│            Update · Control                   │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│           SERVICE AREA ROUTER                 │
│   ┌─────────────────┐  ┌──────────────────┐  │
│   │ PLATFORM ENG.   │  │ RESEARCH INTEL.  │  │
│   │ (Thavren)       │  │ (Eidren)         │  │
│   └────────┬────────┘  └────────┬─────────┘  │
└────────────┼────────────────────┼────────────┘
             ▼                    ▼ (aislado)
┌──────────────────────────────────────────────┐
│          SESSION CAPSULE                      │
│   · Isolation total (sin herencia de chat)    │
│   · Active Identity Packet cargado            │
│   · Token budget asignado                     │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│           POLICY ENGINE                       │
│   permission_authority.json (canónico)        │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│  ┌────────────┐ ┌────────────┐ ┌───────────┐ │
│  │ CONTEXT    │ │ TOOL       │ │ DELEGATION│ │
│  │ GATEWAY    │ │ GATEWAY    │ │ ROUTER    │ │
│  │            │ │            │ │           │ │
│  │ L0-L4      │ │ allow/deny │ │ lead_only │ │
│  │ T0-T5      │ │ risk score │ │ → squad   │ │
│  │ fail_closed│ │ fail_closed│ │ → critical│ │
│  └────────────┘ └────────────┘ └─────┬─────┘ │
└──────────────────────────────────────┼───────┘
                                       ▼
┌──────────────────────────────────────────────┐
│            8 SQUADS ESPECIALIZADOS            │
│  codex · spark · code-reviewer · security    │
│  sys-architect · benchmark · explorer ·      │
│  summarizer · install-engineer               │
└──────────────────────┬───────────────────────┘
                       ▼
┌──────────────────────────────────────────────┐
│  HANDOFF → OBSERVABILITY → DELIVERY          │
│  → KNOWLEDGE COMPILER → ALIGNMENT            │
└──────────────────────────────────────────────┘
```

---

## Capas del Sistema

### Capa 0 — Identidad
- Active Identity Packet: compilado de contratos, portable entre modelos
- Service Area Contracts: definen quién es cada perfil profesional
- Delivery Contracts: definen cómo se comunica cada perfil

### Capa 1 — Aislamiento
- Session Capsule: estado mental de sesión sin herencia de chat
- Context Firewall: L0-L4 classification con deny-before-allow
- Token Budget: T0-T5 enforcement por turno

### Capa 2 — Gobernanza
- Context Gateway: atención selectiva, solo carga lo necesario
- Tool Gateway: permisos granulares por perfil, fail-closed
- Delegation Router: decide lead_only → squad según task_size + risk

### Capa 3 — Ejecución
- Harness Router: reflejos condicionales (surface tocada → validator requerido)
- Squads: 8 agentes especializados gobernados
- Handoff Protocol: transferencia entre áreas sin contaminación

### Capa 4 — Evidencia
- Observability Trace: registro completo de decisiones
- Work Ledger: continuidad entre sesiones
- Evidence Engine: evidencia source-local auditable

### Capa 5 — Proyección
- M1 Global Control Bridge: triage de intenciones globales
- M2 Global Projection Payload: OVAV desde cualquier directorio
- Advanced Readiness Matrix: OFF/READY/READ-ONLY/GATED/ACTIVE

### Capa 6 — Inteligencia Colectiva
- Knowledge Compiler: compila autoridad, handoff, ledger y git en conocimiento operativo
- Pattern Detector: detecta recursiones, contradicciones y oportunidades
- Alignment Engine + Transition Detector: alerta drift real sin confundir cambios legítimos de fase
- Criterion Compiler: preserva criterio de ingeniería entre chats y marca ideas nuevas como propuestas hasta validarlas

---

## Runtime Path (cadena operativa completa)

```text
User intent
  → Service Area Router (selecciona PROFILE + LEAD)
  → Session Capsule (aísla, carga identity packet)
  → Policy Engine (aplica permission_authority.json)
  → Context Gateway (clasifica y aprueba/niega contexto)
  → Delegation Router (decide lead_only | skill | squad)
  → Tool Gateway (concede o niega herramientas)
  → Harness Router (dispara validadores según superficie)
  → Model Budget Router (selecciona profundidad de modelo)
  → Delivery Contract (moldea output visible)
  → Observability Trace (registra decisión)
  → Work Ledger (captura continuidad)
  → Knowledge Compiler (compila conocimiento y patrones)
  → Alignment Engine (detecta drift de rumbo)
```

---

## Componentes Implementados y ruta viva

| Componente | Estado | Implementación |
|---|---|---|
| Service Area Router | ✅ Activo | `tools/agent_runtime/service_area_router.py` |
| Session Capsule | ✅ Activo | auto-activate, isolation, inherited knowledge, self-diagnosis |
| Policy Engine | ✅ Activo | `.ovav/policy/permission_authority.json` |
| Context Firewall / Gateway | ✅ Activo | L5 Context Firewall v2 + budget/injection/overreach checks |
| Tool / Permission Governance | ✅ Activo | canonical permission authority + F0-F5 gates |
| Delegation Router | ✅ Gobernado | lead/squad routing por tamaño y riesgo |
| Harness Router | ✅ Activo | surface→validator mapping + auto triggers |
| Observability Trace | ✅ Activo | evidence source-local + summaries humanos |
| CLI Cockpit | ✅ Activo | `bin/ovav` (curses TUI) |
| Install Gateway | ✅ Activo | `tools/install/` (dry-run) |
| Knowledge Compiler | ✅ P0.2 activo | compiler + pattern + alignment + transition + criterion |

### Gap Crítico Re-encuadrado

El gap principal ya no es “contratos avanzados vs Python placeholder”. Tras L0-L7, F0-F5, Integrity Mesh, Runtime Safety Governor y Knowledge Compiler P0.2, el gap se movió al **diseño con base del Sistema Nervioso Vivo**: grafo de conocimiento gobernado, pesos, activación y consolidación continua sin duplicar rutas ni activar superficies bloqueadas.

---

## Principios de Diseño

1. **Fail-closed por defecto.** Fuente desconocida = denegado. Tool desconocida = denegado.
2. **Deny-before-allow.** El contexto se niega primero, se concede solo si cumple criterios.
3. **Source-local first.** Todo opera dentro del repo. Nada escribe fuera sin gate explícito.
4. **Evidence always.** Cada decisión deja traza. Cada acción deja artefacto.
5. **Identity over model.** La identidad persiste aunque el cuerpo cambie.
6. **Contracts govern, code enforces.** Los YAML definen la ley. El Python la ejecuta.
