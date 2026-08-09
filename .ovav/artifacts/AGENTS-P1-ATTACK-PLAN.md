# OVAV AGENTS — PRIORIDAD 1 ATTACK PLAN
## 1er Ataque Completo: AGENTS + AGENTS/SKILLS

**Fecha:** 2026-07-29
**Estado:** EN PLANIFICACIÓN
**Worktree:** OVAV-fixes-general4

---

## Contexto Ejecutivo (CEO Braka)

### Qué es OVAV AGENTS

OVAV AGENTS es un subsistema de OVAV SYSTEM compuesto por:
- **Áreas profesionales** con leads asignados y equipos con personalidad propia por área
- **Sync.go + Convert.go** — empaquetan la lógica del subsistema para cada CLI destino (opencode, mimocode, cursor, claude code, codex, gemini, etc.)
- Cada CLI lee y requiere formatos diferentes de subagentes — OVAV AGENTS se adapta a cada una sin perder potencia

### Gap detectado: falta modo PLAN obligatorio

El flujo PLAN → BUILD → SCRIPT es el default de todas las CLI de AI desde 2020:
- **PLAN**: preguntas repotenciadoras antes de implementar
- **BUILD**: implementación del plan
- **SCRIPT**: implementación puntual

El problema: OVAV AGENTS con sus áreas, leads, squads funciona en modo implementación directa sin el paso de PLAN. Esto causa:
- Gasto enorme de tokens sin base
- Avance por partes no guiadas a criterio personal
- Sin detención cuando una idea se desvía del proyecto

### Solución propuesta

OVAV AGENTS debe cubrir las necesidades básicas de organización con un modo PLAN obligatorio que:
1. Al iniciar un proyecto o detectar una idea nueva, pregunte: "esta es una idea nueva, deseas armar el plan?"
2. Haga preguntas repotenciadoras según el área (no a criterio libre del CLI)
3. Cubra UX/UI, LÓGICA, FRONT, BACK, BASE DE DATOS de forma guiada
4. Diseñar un plan avanzado con stacks modernos y estables = 70% de estabilidad

---

## Items del Ataque

### PRIORIDAD 1 — AGENTS

| ID | Item | Descripción | Deps |
|----|------|-------------|------|
| A1 | Bug B: GENERAL TASK delegation | Root fix — Bug B en MEMORY-bugs-abc-d | ninguna |
| A2 | T12: Squad delegation | Implementar `workflow + agent(team-*)` jamás usado | ninguna |
| A3 | OVAV AGENTS metacognition failure | M2.7 superó al agente en coherencia — governance drift | ninguna |
| A4 | Elena/Elena-frontend name collision | Colisión de nombres con otra superficie Elena | ninguna |
| A5 | session_greeting OwnerLoggedIn field | Campo missing en GreetingOutput | ninguna |
| A6 | 11 subsystem pending updates | session_greeting, security-gates, skill-resolver, etc. | A5 |
| A7 | T16: Metacognition rule → AGENTS.md | Dureza硬化 de reglas metacognitivas | A3 |
| A8 | CAPA 5 E2E smoke with real agents | Test E2E con agentes reales | A1, A2 |

### PRIORIDAD 2 — AGENTS + SKILLS

| ID | Item | Descripción | Deps |
|----|------|-------------|------|
| SK1 | 4 skills UNTRACKED | ovav-skill-resolver, memory-bridge, squad-delegation, worktree-system | A2 |
| SK2 | Skill ovav-sdd-init: script inexistente | convert_agents GAP-3 | ninguna |
| SK3 | A2A mesh runtime no implementado | Gateway 7-capas existe, A2A no | ninguna |
| SK4 | 19 governor tools Python con 0 imports | "cerebro dormido" — tools sin uso | ninguna |

---

## Plan de Desarrollo por Item

### A1: Bug B — GENERAL TASK delegation

**Problema:** Bug B en MEMORY-bugs-abc-d.md — la delegacion GENERAL TASK no funciona correctamente.

**Análisis requerido:**
1. Leer `.ovav/issues/` para ver si hay ISSUE dedicado a Bug B
2. Leer MEMORY-bugs-abc-d.md para entender el bug específico
3. Buscar en go-runtime el código de delegation (cmd/resolve-subagent, handlers de agent)
4. Identificar por qué la delegacion falla

**Criterio de desarrollo:**
- Root cause, no surface patch
- Test que reproduzca el bug antes de fix
- Fix verificado con test passing

---

### A2: T12 — Squad delegation (`workflow + agent(team-*)`)

**Problema:** Nunca se usó `workflow + agent(team-*)`. El skill ovav-squad-delegation existe pero no está integrado en el flujo real.

**Análisis requerido:**
1. Leer skill `ovav-squad-delegation/SKILL.md` — qué hace exactamente
2. Leer cómo se invoca desde el skill resolver
3. Buscar ejemplos de uso en el codebase
4. Diseñar el flujo completo: cómo un lead o el CEO dispara squad delegation

**Criterio de desarrollo:**
- El skill debe activarse cuando el tipo de tarea sea "squad" o "team"
- Integración con permission injector (AUTH must be LOGIN for squad ops)
- Test del flujo completo

---

### A3: OVAV AGENTS metacognition failure

**Problema:** M2.7 superó al agente en coherencia (mixed English/Spanish). El reasoning loop del agente falla en autocontrol de idioma y lógica.

**Análisis requerido:**
1. Leer el AGENTS.md actual — qué reglas metacognitivas existen
2. Buscar en go-runtime el output_guard y cómo procesa mixed-script
3. Entender el reasoning loop del agente — dónde se detecta el failure
4. Diseñar regla metacognitiva: self-check de coherencia antes de output

**Criterio de desarrollo:**
- Regla en AGENTS.md con enforcement verificable
- El agente debe hacer self-audit antes de cada output
- Output 100% español — zero English words

---

### A4: Elena/Elena-frontend name collision

**Problema:** Hay dos superficies "Elena" — una en UX Design y otra posiblemente en frontend.

**Análisis requerido:**
1. Buscar todas las referencias a "Elena" en el codebase
2. Identificar qué es Elena-frontend vs Elena (UX lead)
3. Determinar si es una colisión real o si son contextos distintos

**Criterio de desarrollo:**
- Si colisión real: renombrar Elena-frontend a Elena-ux-frontend o similar
- Si contextos distintos: documentar la diferencia en AGENTS.md

---

### A5: session_greeting OwnerLoggedIn field

**Problema:** El campo OwnerLoggedIn falta en GreetingOutput.SessionMark.

**Análisis requerido:**
1. Leer cmd/session_greeting/main.go — estructura actual de GreetingOutput
2. Buscar OwnerLoggedIn en la arquitectura — qué debería contener
3. Agregar el campo y asegurarlo en el flujo de login

**Criterio de desarrollo:**
- Campo existe en algún lugar o es campo nuevo a crear
- Se llena correctamente desde ceo.IsActive() o session state

---

### A6: 11 subsystem pending updates

**Problema:** 11 subsistemas necesitan actualización post-login-system-v2: session_greeting, security-gates, skill-resolver, etc.

**Análisis requerido:**
1. Listar los 11 subsistemas desde MEMORY.md
2. Leer cada uno para ver qué necesita actualización
3. Priorizarlos por impacto en el flujo de login

**Criterio de desarrollo:**
- Cada subsystem debe ser actualizado para usar el nuevo AuthState
- Verificar con `go build` que no hay errores de compilación

---

### A7: T16 — Metacognition rule → AGENTS.md

**Problema:** Las reglas metacognitivas no están hardenadas en AGENTS.md.

**Análisis requerido:**
1. Leer AGENTS.md actual — qué reglas hay
2. Comparar con lo que dice MEMORY.md que debe tener
3. Escribir las reglas faltantes con enforcement claro

**Criterio de desarrollo:**
- AGENTS.md con reglas duras de metacognición
- Output language enforcement: 100% español, zero English
- Self-audit loop antes de output

---

### A8: CAPA 5 E2E smoke with real agents

**Problema:** No hay test E2E que use agentes reales de principio a fin.

**Análisis requerido:**
1. Leer cómo están los tests actuales en go-runtime
2. Diseñar smoke test que: inicie sesión, ejecute un comando simple via agente, verifique output
3. Integrar con el sistema de CI si existe

**Criterio de desarrollo:**
- Test rápido (< 30s) que cubra el flujo completo
- No requiere mock — usa los agentes reales

---

### SK1: 4 skills UNTRACKED

**Problema:** ovav-skill-resolver, ovav-memory-bridge, ovav-squad-delegation, ovav-worktree-system están en symlink chain pero status UNTRACKED.

**Análisis requerido:**
1. Leer caps.yaml new_skills_v80 — estado actual
2. Verificar que los symlinks están correctos
3. Marcar como TRACKED y commitear

**Criterio de desarrollo:**
- Los 4 skills deben responder a skill_search
- Verify: cargar cada skill y confirmar que funciona

---

### SK2: Skill ovav-sdd-init — script inexistente

**Problema:** El skill ovav-sdd-init referencia un script que no existe.

**Análisis requerido:**
1. Leer .ovav/source/skills/ovav-sdd-init/SKILL.md
2. Encontrar qué script referencia y por qué no existe
3. Crear el script o corregir la referencia

**Criterio de desarrollo:**
- El skill debe invocarse sin errores
- Si el script es obsoleto: eliminar la referencia

---

### SK3: A2A mesh runtime no implementado

**Problema:** Gateway de 7 capas existe pero A2A mesh no.

**Análisis requerido:**
1. Leer la arquitectura de gateway en go-runtime/internal/
2. Entender qué es A2A mesh y cómo debería funcionar
3. Diseñar e implementar o marcar como deferred si es demasiado complejo

**Criterio de desarrollo:**
- Si es scope de Phase 2: documentar como deferred
- Si es implementable ahora: implementar con tests

---

### SK4: 19 governor tools Python con 0 imports

**Problema:** 19 tools en governor/ Python no se usan — "cerebro dormido".

**Análisis requerido:**
1. Listar las 19 tools en tools/governor/
2. Verificar cuáles tienen 0 imports reales
3. Determinar si son: (a) deprecated, (b) para migrar a Go, o (c) para eliminar

**Criterio de desarrollo:**
- Migrar a Go las que tengan lógica vigente
- Eliminar las que sean obsoletas
- Documentar las decisiones en el commit

---

## ANÁLISIS COMPLETO: Sistema PLAN de 6 CLI (2026-07-29)

### Benchmark — PLAN MODE en cada CLI

| CLI | PLAN MODE | Read-Only | Approval Gate | Subagent | MCP | Init | Differentiator |
|---|---|---|---|---|---|---|---|
| **MiMoCode** | compose:brainstorm+plan+build | ✅ brainstorm | ✅ antes de plan | ✅ subagent/task | ✅ | /init | 3-layer pipeline |
| **Kimi Code** | `--plan` + `plan` subagent | ✅ Plan + subagent | ✅ antes de edits | ✅ coder/explore/plan | ✅ full stdio/HTTP/SSE | /init | Read-only subagent sin shell |
| **Claude Code** | `plan` permission mode | ✅ Shift+Tab | ✅ antes de edits | ✅ Plan subagent + /batch | ✅ | /init → CLAUDE.md | /batch + git worktrees |
| **Cursor AI** | Plan Mode (Shift+Tab) | ✅ | ✅ | ❌ composer model | ✅ | /init | Checkpoints + Composer model |
| **OpenCode** | Plan agent (Tab key) | ✅ deny all edits | ✅ approval | ✅ General/Explore/Scout | ✅ | /init → AGENTS.md | Permission-as-code granular |
| **Codex** | ❌ No plan mode | ❌ | ❌ review only | ✅ subagents + Micro | ✅ | /init → AGENTS.md | Review workflow |

### Lo MEJOR de cada CLI → synthesize into OVAV

| Feature | CLI | Implementación en OVAV |
|---|---|---|
| HARD GATE (sin plan = sin código) | MiMoCode (brainstorm) | ✅ ovav-brainstorm — HARD GATE |
| Read-only + approval antes de edits | Kimi, Claude, Cursor | ✅ En ovav-brainstorm + ovav-plan |
| Subagent especializado en plan | Kimi (plan subagent) | ✅ Skill `ovav-plan-agent` |
| TDD: test falla primero | MiMoCode (compose:plan) | ✅ En plan template |
| Bite-sized tasks (2-5 min) | MiMoCode (compose:plan) | ✅ En plan template |
| Arquitectura archivos primero | MiMoCode (compose:plan) | ✅ Step 0 del plan |
| **Preguntas POR ÁREA (DIFFERENTIATOR)** | Ninguna — todas genéricas | ✅ **OVAV exclusivo** |
| Self-review checklist | MiMoCode (compose:plan) | ✅ En plan template |
| Batch decomposition + git worktrees | Claude Code (/batch) | ✅ Integrar con OWS (owc/owd) |
| Checkpoints/auto-snapshot | Cursor AI | ⚠️ OWD checkpoint integration |
| Permission-as-code granular | OpenCode | ✅ Ya existe (permission_authority.json) |
| Effort calibration | Cursor (Composer) | ⚠️ Phase 2 |
| Cloud planning (ultraplan) | Claude Code | ❌ No priority ahora |
| Model per agent | OpenCode | ✅ En skill architecture |

### ovav-brainstorm skill — IMPLEMENTATION DESIGN

**Ubicación:** `.ovav/source/skills/ovav-brainstorm/`

```
ovav-brainstorm/
├── SKILL.md                    — HARD GATE skill (se activa en idea nueva)
├── BRAINSTORM.md               — Cuestionario guiado (preguntas por área)
├── design-template.md           — Template de diseño generado
└── areas/
    ├── ux-design.md             — Elena: wireframes, flujos, paleta, tipografía
    ├── platform.md              — Thavren: arquitectura, stack, runtime, CI/CD
    ├── research.md              — Eidren: fuentes, evidencia, benchmarks
    ├── health.md                — Renata: métricas, performance, seguridad
    ├── commercial.md            — Sofía: monetización, usuarios, mercado
    ├── product.md               — Dante: frontend, componentes, API
    ├── devops.md                — Uriel: infraestructura, cloud, containers
    ├── education.md             — Valeria: currículo, cursos, tutorías
    ├── legal.md                 — Camila: contratos, compliance, GDPR
    └── adversarial.md           — Kenji: red team, penetración, hardening
```

### Flujo OVAV AGENTS Native (3 fases)

```
IDEA (usuario)
    ↓
┌──────────────────────────────────────────────────────────────┐
│ ovav-brainstorm (HARD GATE) — 1era fase                      │
│ • Detecta idea nueva o desviación del proyecto               │
│ • Cuestionario guiado POR ÁREA (Elena/Thavren/etc.)         │
│ • Genera diseño en 5 secciones                              │
│ • CEO aprueba → pasa a ovav-plan                            │
│ • CEO rechaza → redefine diseño                              │
└──────────────────────────────────────────────────────────────┘
    ↓ (CEO approval)
┌──────────────────────────────────────────────────────────────┐
│ ovav-plan — 2da fase                                        │
│ • Arquitectura de archivos (Step 0)                          │
│ • Tareas bite-sized (2-5 min) con TDD                        │
│ • Self-review checklist                                      │
│ • CEO aprueba → pasa a ovav-build                           │
└──────────────────────────────────────────────────────────────┘
    ↓ (CEO approval)
┌──────────────────────────────────────────────────────────────┐
│ ovav-build — 3era fase                                      │
│ • Subagent por tarea (integrar con OWS worktree)            │
│ • Verificación en cada checkpoint                            │
│ • Test passing antes de siguiente tarea                      │
└──────────────────────────────────────────────────────────────┘
```

### Preguntas POR ÁREA — DETALLE COMPLETO

```
UX DESIGN (Elena)
  1. ¿Wireframes o mockups en Figma?
  2. ¿Paleta de colores + tipografía definida?
  3. ¿Flujo de usuario principal mapeado?
  4. ¿Responsive breakpoints definidos?
  5. ¿Accesibilidad WCAG 2.1 AA requerida?
  6. ¿Componentes UI reutilizables?

PLATFORM ENGINEERING (Thavren)
  1. ¿Monorepo o polyrepo?
  2. ¿Runtime: Go / Node.js / Python / Rust?
  3. ¿API: REST / GraphQL / gRPC / tRPC?
  4. ¿Database: PostgreSQL / SQLite / DynamoDB / etc?
  5. ¿CI/CD: GitHub Actions / CircleCI / ArgoCD?
  6. ¿Autenticación: OAuth / JWT / Session / Passkeys?
  7. ¿Monitoreo: Prometheus / OpenTelemetry / Datadog?
  8. ¿Secrets: Vault / AWS SM / ENV?

RESEARCH INTELLIGENCE (Eidren)
  1. ¿Fuentes académicas o de industria?
  2. ¿Benchmarks existentes para comparar?
  3. ¿Evidencia de usuario o solo hipótesis?
  4. ¿Competitive analysis requerida?
  5. ¿Data Sources: APIs públicas o scrap?

COMMERCIAL & GROWTH (Sofía)
  1. ¿Modelo: SaaS / Freemium / One-time / Marketplace?
  2. ¿Pricing: por usuario / por consumo / flat?
  3. ¿Integración de pagos: Stripe / PayPal / local?
  4. ¿Onboarding flow definido?
  5. ¿Metrics: LTV / CAC / Churn?

PRODUCT (Dante)
  1. ¿Frontend: React / Vue / Svelte / Next.js / Astro?
  2. ¿Mobile: PWA / nativo / RN / Flutter?
  3. ¿API design: OpenAPI / AsyncAPI?
  4. ¿State management: Zustand / Redux / Jotai?
  5. ¿Testing: unit / e2e / visual regression?

DEVOPS (Uriel)
  1. ¿Infra: AWS / GCP / Azure / self-hosted?
  2. ¿Containers: Docker / Podman / Kubernetes?
  3. ¿IaC: Terraform / Pulumi / Ansible?
  4. ¿Secrets rotation strategy?
  5. ¿Backup y disaster recovery plan?

HEALTH (Renata)
  1. ¿Métricas: tokens / latency / errors?
  2. ¿SLA requerido: 99.9% / 99.99%?
  3. ¿Load testing baseline?
  4. ¿Seguridad: OWASP Top 10 considerada?
  5. ¿Data retention policy?
```

### Criterio de éxito — PL MODE en OVAV AGENTS

| Métrica | Sin plan | Con ovav-brainstorm+plan |
|---|---|---|
| Estabilidad del proyecto | ~10% | ~70% |
| Gasto de tokens | Sin control | Optimizado por fase |
| Tiempo hasta producción | Meses | Semanas |
| Base técnica | Ad-hoc | Stacks modernos validados |
| Detección de riesgos | Tardía | Temprana (fase diseño) |
| Desviación del proyecto | Frecuente | Bloqueada por HARD GATE |

---

## Orden de Ataque — ACTUALIZADO

```
PL.  ovav-brainstorm (ITEM 0 — FUNDACIONAL)
   → Crea ovav-brainstorm skill (HARD GATE)
   → Skill ovav-plan (read-only + approval)
   → Skill ovav-build (subagent + OWS)
   → Template de diseño + 10 archivos por área

1.  A5 (OwnerLoggedIn) — rápido, clarifica el estado
2.  A4 (Elena collision) — rápido, evita confusión
3.  A1 (Bug B delegation) — crítico, afecta a todos los agentes
4.  A2 (Squad delegation) — desbloquea trabajo paralelo
5.  A3 (Metacognition) — governance crítico
6.  SK2 (sdd-init script) — quick win
7.  SK4 (19 tools Python) — limpieza, rápido
8.  A7 (AGENTS.md hardening) —跟着 A3
9.  SK1 (4 skills UNTRACKED) —跟着 A2
10. SK3 (A2A mesh) — deferred o simple
11. A6 (11 subsystems) —跟着 A5
12. A8 (E2E smoke) — final, validación
```

**PL. ovav-brainstorm es el item más crítico — sin él, todo lo demás se construye sin base.**

---

**Regla del plan:** ANTES de cada item, crear sub-plan de desarrollo confirmando criterio con CEO.
**Regla de commit:** Cada item = 1 commit mínimo, con test verificable.
**Regla de memoria:** Todo descubrimiento va a notes.md + MEMORY.md.
