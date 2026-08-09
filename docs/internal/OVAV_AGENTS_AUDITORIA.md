# OVAV AGENTS — Auditoría de Funcionalidad Real

## Qué es OVAV AGENTS

OVAV AGENTS es el subsistema de agentes autonomous de OVAV que:
- Tiene **10 áreas profesionales** con leads expertos mundiales en cada área
- Cada lead tiene **equipo propio** con personalidad, nombre, skills específicos
- Se empaqueta via `sync.go` + `convert.go` para cualquier CLI (mimocode, opencode, cursor, claude code, etc)
- Tiene gobernanza de delegación, harnesses, y flujo PL-0 automático

---

## Estado Real — Lo QUE SÍ Funciona

### 1. Arquitectura de Agentes ✅

```
Áreas: 10 areas completas en ovav/agents/areas/
Leads: 10 leads con perfiles de experto mundial
Teams: 60 equipos con personalidad propia
Total: ~5153 líneas de perfiles YAML
```

### 2. CRITERIA Injection ✅

```
opencode converter → Area.CRITERIA + Lead.CRITERIA inyectados ✅
mimocode converter → Area.CRITERIA + Lead.CRITERIA inyectados ✅
cursor converter → Area.CRITERIA + Lead.CRITERIA inyectados ✅
claude converter → Area.CRITERIA + Lead.CRITERIA inyectados ✅
```

Commit: `6f7e0db0` — GAP-1 converter fix

### 3. PL-0 — Autonomous Plan Mode ✅

Implementado en AGENTS.md con 4 hard gates:
1. NEW IDEA detector →强制 plan antes de código
2. PLAN EXISTS check obligatorio
3. MULTI-AREA detection obligatoria
4. SEPARATE DESIGN.md por área en proyectos EPIC

### 4. ovav-brainstorm — Preguntas por Área ✅

```
ovav-brainstorm/SKILL.md → HARD GATE protocol
references/area-questions-platform_engineering.md
references/area-questions-ux_design.md
references/area-questions-research_intelligence.md
... (10 áreas)
PPQ1-PPQ10 pre-project questionnaire
```

### 5. Skills BUILD Suite ✅

```
ovav-build — modo implementación
ovav-tdd — test-driven development
ovav-verify — verificación post-build
ovav-review — revisión de código
```

### 6. A2A Delegation Mesh (Go runtime) ✅

```
go-runtime/internal/runtime/delegation.go:
  h_delegation_trigger — evalúa complejidad (score >= 40 = delegate)
  h_do_not_delegate_guard — bloquea triviales (explanation, typos)
  delegation_router — detecta área + resuelve lead
  BuildDelegationPayload — carga perfil OVAV (6132 chars) + git context
```

CLI de test: `go run -C go-runtime ./cmd/delegation/ --route --payload --agent=eidren -- "task"`

### 7. Agent Governance — 9 Harnesses Registrados ✅

```
h_delegation_trigger       → go-runtime/internal/runtime/delegation.go ✅
h_do_not_delegate_guard    → go-runtime/internal/runtime/delegation.go ✅
h_subagent_context_pack    → go-runtime/internal/runtime/delegation.go ✅
h_subagent_result_absorber → go-runtime/internal/runtime/delegation.go ✅
h_skill_score_gate         → go-runtime/internal/runtime/delegation.go ✅
h_result_contract          → go-runtime/internal/runtime/delegation.go ✅
h_work_report_builder      → go-runtime/internal/runtime/delegation.go ✅
h_human_response_translator → go-runtime/internal/runtime/delegation.go ✅
h_verify_evidence          → ya existía ✅
```

### 8. Subagent Catalog ✅

```
.ovav/registry/subagent_catalog.yaml — 10 leads + 60 teams
resolve_subagent CLI → funciona correctamente
```

---

## Estado Real — Lo QUE FALTA

### Gap 1: MImoCode NO consume los agentes OVAV ⚠️ CRÍTICO

**El problema:**
```
actor tool de MiMoCode → whitelist = {explore, general}
lead-eidren, team-clara, etc → caen a "general"
El perfil OVAV NO se inyecta
```

**Lo que SÍ funciona:**
- `convert_agents --levels all` → genera 10 leads + 60 teams en `runtimes/mimocode/agents/`
- Archivos SON correctos (502 líneas, CRITERIA + permission + identity guard)
- El `ovav delegate` command prepara el payload correctamente

**Lo que NO funciona:**
- MiMoCode no está leyendo los archivos de `runtimes/mimocode/agents/`
- No tenemos acceso para verificar cómo MiMoCode spawnea agentes
- La configuración de agentes de MiMoCode está fuera de nuestro alcance

**Impacto:** El subsistema OVAV AGENTS está 100% listo pero MiMoCode no lo usa. Estamos trabajando en el sistema fuente sin que el consumer lo reciba.

### Gap 2: NEW IDEA Detector NO es automático en runtime ⚠️ MEDIO

**El problema:**
- El NEW IDEA detector está en AGENTS.md (como texto/instrucciones)
- NO está implementado como componente runtime que se ejecute automáticamente
- Depende del agente leer y seguir AGENTS.md cada vez

**Lo que funciona:**
- Si el agente sigue AGENTS.md → el gate de new idea se activa
- Si el agente NO lo sigue → no hay gate automático

**Impacto:** Sesiones que empiezan directo a código sin plan. El gasto de tokens puede ser 5x mayor en proyectos sin plan.

### Gap 3: Síncronización convert/sync NO está wired ⚠️ MEDIO

**Lo que existe:**
- `convert.go` — 4 converters (opencode, mimocode, cursor, claude)
- `sync.go` — mecanismo de sync para diferentes CLIs

**Lo que falta:**
- No hay evidencia de que `sync.go` se ejecute automáticamente
- No hay trigger que regener los agentes cuando los profiles OVAV cambian
- El workflow de sync para mantener los agentes actualizados no está automatizado

### Gap 4: Personalidad de equipos NO implementada ⚠️ BAJO

**Lo que existe:**
- Archivos YAML con `name`, `description`, `role`
- Squad members tienen `name` y `role`

**Lo que falta:**
- Los perfiles de equipo NO tienen "personalidad, nombre, vida, cerebro" más allá de los campos YAML
- No hay "backstory" o "thinking patterns" implementados
- Los nombres son funcionales, no tienen identidad narrativa

---

## Benchmark — PLAN/BUILD/SCRIPT Absorption

| CLI Original | Función | Absorbido en OVAV? |
|---|---|---|
| PLAN mode | Preguntas de diseño antes de código | ✅ ovav-brainstorm (PPQ1-PPQ10) |
| BUILD mode | Implementación siguiendo plan | ✅ ovav-build skill |
| SCRIPT mode | Ejecución puntual | ⚠️ No hay skill equivalente |

**Lo que falta del benchmark:**
- SCRIPT/BUILD/RUN no están diferenciados como skills separados
- No hay "modo Debug" o "modo Test" como skills dedicados
- El pipeline PLAN → BUILD → REVIEW no está wired como flujo automático

---

## Recomendaciones — Prór So

### Críticos (resolver ahora):

1. **MiMoCode agent consumption** — Necesitamos que alguien con acceso a la config de MiMoCode configure que los agentes se lean de `runtimes/mimocode/agents/`. Alternativa: implementar invocación directa via API de modelos sin pasar por actor tool.

2. **NEW IDEA gate automático** — Implementar el detector como componente runtime (no solo como instrucción en AGENTS.md) que se ejecute en cada input sin depender de que el agente lo recuerde.

### Medios (resolver en siguientes sprints):

3. **Sync automático** — Trigger que regener agents cuando profiles OVAV cambian. Hook en git post-commit.

4. **Pipeline PLAN → BUILD → REVIEW wired** — Crear skill `ovav-execute` que maneje el flujo completo con checkpoints.

### Menores (para más adelante):

5. **Personalidad narrativa de equipos** — Expandir los perfiles YAML con backstory, thinking patterns, communication style.

6. **Skill diferenciado SCRIPT/BUILD/RUN** — Separar los modos de ejecución en skills específicos.

---

## Veredicto

**Lo que SÍ tenemos:**
- Arquitectura de agentes completa (10 áreas, 10 leads, 60 teams)
- CRITERIA injection funcionando en los 4 converters
- PL-0 con 4 hard gates (pero no automáticos en runtime)
- ovav-brainstorm con preguntas por área
- A2A delegation mesh en Go funcionando
- 9 harnesses de gobernanza registrados y funcionales
- Skills BUILD suite (build, tdd, verify, review)

**Lo que NO tenemos:**
- MiMoCode consume los agentes (Gap crítico — el sistema fuente está listo pero el consumer no recibe)
- NEW IDEA gate automático en runtime (solo existe como instrucción, no como componente)
- Sync automático wired (existe el código pero no el trigger)
- Pipeline PLAN → BUILD → REVIEW como flujo automático

**Lo que el CEO debe decidir:**
1. ¿Quién tiene acceso a la config de MiMoCode para hacer que use `runtimes/mimocode/agents/`?
2. ¿Prioridad de implementar NEW IDEA detector como componente runtime automático?
3. ¿Automatizar sync trigger en post-commit?

---

*Auditoría OVAV AGENTS — 2026-07-30 — Thavren*
