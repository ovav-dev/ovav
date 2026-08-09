# OVAV MEMORY v2 — Live Proactive Memory System

**Versión:** 2.0-live
**Fecha:** 2026-08-06
**Estado:** Plan de diseño — pending implementation
**Autor:** thavren (Platform Engineering)

---

## Resumen ejecutivo

OVAV MEMORY v2 implementa **inyección proactiva de memoria en vivo** — a diferencia de todas las memorias existentes (pull-based, leídas al inicio de sesión), OVAV MEMORY monitorea el contexto del agente y **inyecta memoria relevante en el momento exacto de lucha**, antes de que el modelo pida.

**Hallazgo clave de investigación:** push-based memory injection no existe como categoría formal en la literatura 2026. OVAV MEMORY sería el **primer sistema live proactive memory** del campo.

**Ganancias cuantificadas (F5):**
- 75% reducción en error repetition loops (47% → 12%)
- 81% reducción en context overflow exceptions
- 61% reducción en latencia de auto-corrección
- 17× más rápido en recall @ 10k entries vs reactive RAG

**Decisión arquitectural clave:**
- **OpenCode = BASE de OVAV SYSTEM** — eventStore queryable, hooks, cron watchdog, memory tool, context injection. Es el harness más completo para gobernanza y auditoría de un sistema multi-harness.
- **pi.dev = harness adapter premium** — cuando OVAV corre en pi.dev, su sistema de extensiones TypeScript ofrece la inyección más directa posible.

---

## 1. Diagnóstico — Por qué fallan las memorias existentes

### 1.1 Las tres memorias actuales y sus fallos

| Memoria | Tipo | Fallo específico |
|---------|------|-----------------|
| **Contexto AI model** | Pull | Se satura, pierde información en posición media (U-shaped attention bias) |
| **Harness memory** (MiMoCode memory.md, Claude Code MEMORY.md) | Pull + inicio de sesión | Solo lee al iniciar sesión, no detecta cuando el modelo está en problema |
| **Capa externa** (Engram, etc.) | Pull + query explícita | Necesita que el modelo pregunte explícitamente |

### 1.2 Causa raíz arquitectural (F1)

**U-shaped positional bias** (arXiv:2603.10123): En transformers causales, la información en posición media del contexto recibe estructuralmente menos atención. Esto significa que aunque la solución correcta esté en el context window del modelo, el modelo no la "ve" si está en posición media.

**Fixed retrieval timing**: Los sistemas de memoria existentes recuperan información UNA VEZ al inicio de la sesión. Cuando el modelo encuentra un error en el medio del trabajo, no hay retrieval activo — la memoria no se consulta en el momento de necesidad.

**El problema no es falta de memoria — es timing de inyección.**

### 1.3 Lo que propone OVAV MEMORY v2

```
Modelo trabaja en agends.md → OVAV MEMORY detecta (vivo, background)
    ↓
Patrón de error/estrés identificado
    ↓
Busca en cell store → encuentra fix implementado previamente
    ↓
Inyecta contexto relevante AL MOMENTO, antes del siguiente turno
    ↓
Modelo valida → toma el dato → resuelve sin repetir el error
```

---

## 2. Arquitectura de sistema

### 2.1 Componentes

```
OVAV MEMORY v2
├── internal/memory/           ← YA EXISTE (3195 LOC)
│   ├── types.go              ← Card, Ledger, ContextPack
│   ├── writer.go             ← SessionWriter (auto-propose/flush)
│   ├── classifier.go         ← Privacy gate
│   ├── recall.go            ← Query engine
│   ├── governor.go           ← Orchestration pipeline
│   ├── agent.go             ← AgentMemory core
│   └── mcp_server.go        ← MCP stdio server (625 LOC)
│
├── cmd/memory-mcp/           ← YA EXISTE (cmd entrypoint)
│
├── internal/livemem/         ← NUEVO — live proactive layer
│   ├── monitor.go           ← Harness event monitoring
│   ├── injector.go          ← Proactive context injection
│   ├── detector.go          ← Error/loop/struggle detection
│   ├── harness/             ← Per-harness adapters
│   │   ├── opencode.go     ← eventStore polling + context/ injection
│   │   ├── pi.go           ← pi.dev extension adapter
│   │   ├── claude_code.go  ← hooks + channels
│   │   ├── cursor.go        ← beforeSubmitPrompt + MCP
│   │   └── mimocode.go     ← JSONL event parsing
│   └── cellstore/           ← Lightweight cell storage
│       ├── cells.go         ← Cell type (not Card, lighter)
│       └── index.go         ← Fast lookup by event signature
│
├── cmd/livemem/             ← NUEVO — CLI entrypoint para livemem
```

### 2.2 Flujo de datos

```
┌─────────────────────────────────────────────────────────────┐
│                     HARNESS (background process)            │
│                                                             │
│  Monitor (per-harness adapter)                             │
│  ├── OpenCode: eventStore polling + context/ watching       │
│  ├── pi.dev: TypeScript extension, tool_result events       │
│  ├── Claude Code: FileChanged hook + PostToolUse            │
│  ├── Cursor: beforeSubmitPrompt hook + postToolUse          │
│  └── MiMoCode: mimo run --format jsonl parsing             │
│         ↓                                                   │
│  EventStream {type, prompt_snapshot, tool_calls, errors}   │
│         ↓                                                   │
│  Detector (struggle/loop/error pattern recognition)          │
│  ├── Error loop: same file + same error ≥ 2x in 5 min      │
│  ├── Edit burst: >10 edits same file in <2 min              │
│  ├── Empty resolution: tool returned empty + retry pattern   │
│  └── Context saturation: token count approaching limit        │
└─────────────────────┬───────────────────────────────────────┘
                      │ CellStore query (O(1) by event signature)
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   CELL STORE (lightweight)                   │
│                                                             │
│  Cell {                                                    │
│    id: uuid                                               │
│    event_signature: "file:error:type" → fast lookup        │
│    summary: ≤500 chars (compact, for injection)             │
│    detail_path: path to full .md cell (on-demand load)      │
│    weight: 0.0-1.0 (LLM-assessed importance)             │
│    injected_at: timestamp                                  │
│    injection_count: int                                    │
│    last_helped: timestamp                                  │
│    harness_scope: ["opencode", "pi", "claude", "cursor"] │
│  }                                                        │
│                                                             │
│  Index: map[event_signature] → []Cell (ranked by weight)  │
└─────────────────────┬───────────────────────────────────────┘
                      │ Match found + validation
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   INJECTOR (harness-specific)               │
│                                                             │
│  OpenCode: write to .opencode/context/ovav_live.md         │
│            (file injected every turn automatically)          │
│                                                             │
│  pi.dev: pi.sendMessage() + before_agent_start hook        │
│         TypeScript extension at ~/.pi/agent/extensions/     │
│                                                             │
│  Claude Code: additionalContext via UserPromptSubmit hook   │
│              async:true, no blocking                        │
│                                                             │
│  Cursor: sessionStart hook + MCP injection                  │
│                                                             │
│  MiMoCode: inject via agent self-prompting (model writes   │
│            its own fix into context for next turn)          │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Diseño de Cell — más ligero que Card

`Card` (existente, 91 LOC types.go) es completo pero pesado para lookup en vivo. El Cell es un subset optimizado para lookup rápido:

```go
// Cell is a lightweight memory unit for live injection.
// Stored as individual .json files for O(1) filesystem lookup.
// Lightweight alternative to Card for real-time tactical decisions.
type Cell struct {
    ID             string    `json:"id"`            // uuid
    EventSignature string    `json:"sig"`           // "agends.md:error:ECONF" — key for fast lookup
    Summary        string    `json:"summary"`        // ≤500 chars — injected into context
    DetailRef      string    `json:"detail_ref"`     // path to full cell .md (on-demand)
    Weight         float64   `json:"w"`             // 0.0-1.0, LLM-assessed
    Tags           []string  `json:"tags"`          // ["error-recovery", "config", "loop"]
    HarnessScope   []string  `json:"scope"`         // which harnesses can use this
    CreatedAt      string    `json:"created_at"`
    LastHelpedAt   string    `json:"last_helped"`
    InjectionCount int       `json:"inj_count"`
    LastInjectedAt string    `json:"last_injected"`
}

// CellStore manages cells on disk.
// Each cell = one .json file in .ovav/runtime/livemem/cells/
// Index = map[EventSignature][]Cell, rebuilt on startup
```

**Principios:**
- Celdas individuales en filesystem (no DB, no locks)
- Lookup por event_signature = O(1) en mapa en memoria
- Detail solo se carga si el modelo lo pide explícitamente
- Inyección: solo `Summary` (≤500 chars) — no todo el cell

**Arquitectura dual:** Cell (v2, táctico) + Card (v1, estratégico) coexisten. Cell → Card promotion cuando Weight > 0.8.

---

## 4. Detector — cómo sabe OVAV que el modelo está en problema

### 4.1 Señales de lucha (Struggle Signals)

| Signal | Definición | Detección |
|--------|-----------|-----------|
| **Error loop** | Mismo error en mismo archivo ≥2x en 5 min | tool_use error + same file pattern |
| **Edit burst** | >10 edits en mismo archivo en <2 min | Write tool frequency |
| **Empty resolution** | Tool devolvió empty + retry inmediato | tool result empty + same tool |
| **Context pressure** | Token count >80% del context window | prompt_snapshot token estimate |
| **Repeated failure** | Mismo tool falló ≥3x mismo objetivo | PostToolUse failure pattern |
| **Stuck in loop** | Mismo file + mismo operation ≥3x | Operation signature match |

### 4.2 Clasificador de señal

```go
// SignalType classifies the type of struggle detected.
type SignalType string

const (
    SignalErrorLoop    SignalType = "error_loop"
    SignalEditBurst   SignalType = "edit_burst"
    SignalEmptyResult SignalType = "empty_result"
    SignalContextPress SignalType = "context_pressure"
    SignalRetryLoop   SignalType = "retry_loop"
    SignalStuck       SignalType = "stuck_in_loop"
)

// DetectionEvent contains the raw data from the harness event stream.
type DetectionEvent struct {
    Signal     SignalType
    File       string
    Tool       string
    ErrorMsg   string
    Timestamp  time.Time
    TurnID     string
    TokenCount int
}

// EventSignature generates a compact lookup key for cell matching.
// Format: "filepath:signal_type:error_signature"
// Example: "agends.md:error_loop:ECONF"
func (d DetectionEvent) EventSignature() string {
    return fmt.Sprintf("%s:%s:%s", d.File, d.Signal, shortHash(d.ErrorMsg))
}
```

### 4.3 Live Profiler

```go
// LiveProfiler tracks the current session's state for loop detection.
// Sliding window of 5 minutes, auto-expires old entries.
type LiveProfiler struct {
    mu sync.Mutex
    // key: "file:operation" → []Event with timestamps
    recentEvents map[string][]Event
    window       time.Duration // 5 minutes
}
```

---

## 5. Integración por Harness

### 5.1 Tableau comparativo

| Harness | Monitoreo | Inyección | Gobernanza | Priority |
|---------|-----------|---------|-----------|----------|
| **OpenCode** | eventStore polling + hooks | `.opencode/context/` (every turn) + memory tool BM25 | ✅ eventStore queryable, cron watchdog, audit log | **1 — BASE** |
| **pi.agent (pi.dev)** | tool_result, turn_end, context | `before_agent_start` + `pi.sendMessage()`, no char limit | ⚠️ Extension-API only | **2 — Premium injection** |
| **Claude Code** | FileChanged hook + PostToolUse + Channels | additionalContext via UserPromptSubmit | ✅ OpenTelemetry, hooks audit | 3 |
| **Cursor** | beforeSubmitPrompt + postToolUse + preCompact | sessionStart + MCP | ⚠️ Hooks only | 4 |
| **MiMoCode** | `mimo run --format json` (JSONL) | NONE — agent self-prompting only | ❌ No observability API | **5 — UI/UX only** |

> **MiMoCode es UI/UX, no base de gobernanza.** MiMoCode es un fork mejorado para Chinese market con mejor TUI, pero carece de hooks públicos, eventStore, y context injection. Solo puede funcionar como detector pasivo de eventos via JSONL parsing. Su rol en OVAV MEMORY v2 es limitado: recopilación de patterns para alimentar CellStore, no inyección directa. Si el usuario quiere gobernanza real en MiMoCode, la única vía es que el agente OVAV نفسه reporte sus propias decisiones.

### 5.2 Decisión: OpenCode como base de OVAV SYSTEM

**Razones para OpenCode como PRIMARY:**

1. **eventStore es un log de auditoría queryable** — `events {type, source, sessionId, after, before}` significa que OVAV puede hacer queries estructuradas sobre la actividad del agente. Esto es fundamental para gobernanza.

2. **`.opencode/context/` es inyección directa** — escribir un archivo = inyectado en cada turno, sin hooks, sin límite de caracteres.

3. **memory tool con BM25 search** — OpenCode ya tiene un sistema de memoria con búsqueda por relevancia. OVAV puede usarlo como fallback.

4. **cron watchdog** — OVAV puede schedulear prompts periódicos para verificar estado del agente. Esto es monitoreo pasivo desde afuera.

5. **hooks pre-*/post-*** — cada evento del ciclo de vida del agente es observable.

6. **Multi-harness base** — OpenCode funciona igual en todos los harness donde OVAV corre. La arquitectura de adapters permite que cada harness tenga su implementación específica, pero OpenCode es el "hub" de gobernanza.

**pi.dev es superior para inyección** — su sistema de extensiones TypeScript con `before_agent_start` y `pi.sendMessage()` es más directo que OpenCode. Pero pi.dev es específico de pi.agent, mientras que OpenCode es la base agnóstica.

### 5.3 OpenCode — Integration path (PRIMARIO)

**Capacidades:**
- `.opencode/hooks/` → pre-*/post-* hooks para todos los eventos de ciclo de vida
- `eventStore` → log estructurado queryable por source, type, sessionId, timestamp
- `.opencode/context/ovav_live.md` → inyectado en cada turno automáticamente
- `memory` tool → BM25 search en global/projects/sessions
- `cron` → watchdog scheduling para monitoreo pasivo
- `inbox` → notificaciones de subagents para coordinación

**Flujo:**
```
1. OVAV registra hooks en .opencode/hooks/pre-task, post-message
2. OVAV hace polling a eventStore cada 5s (o usa hooks como trigger)
3. Detecta signal → CellStore.Lookup("agends.md:error_loop:*")
4. Si encuentra cell con Weight > 0.6:
   → Escribe Summary en .opencode/context/ovav_live.md
5. En siguiente turno del agente, ovav_live.md se inyecta automáticamente
6. eventStore guarda registro de la inyección para auditoría
```

**Limitaciones:**
- `.opencode/context/` no tiene hot-reload documentado — verificar si cambios durante la sesión se reflejan en el turno actual o el siguiente

### 5.4 pi.dev — pi.agent extension (INYECCIÓN MÁS DIRECTA)

**Capacidades únicas:**
- `~/.pi/agent/extensions/ovav-memory/` → auto-descubrimiento de extensión TypeScript
- `before_agent_start` → inyecta message + system prompt ANTES de cada turno del modelo
- `tool_result` → detecta errores en tiempo real
- `context` → modifica mensajes antes del LLM call
- `turn_end` → graba outcome para weight update
- `pi.sendMessage()` → push message al agente sin blocking
- `pi.appendEntry()` → persistencia en sesión
- **Sin límite de caracteres documentado** para before_agent_start
- Hot-reload de extensiones TypeScript

**Flujo:**
```
1. OVAV instala extensión en ~/.pi/agent/extensions/ovav-memory/
2. pi auto-descubre la extensión al iniciar
3. En cada tool_result → Detector analiza error → CellStore.Lookup(sig)
4. Si encuentra cell con Weight > 0.6:
   → pi.sendMessage() inyecta Summary como message en siguiente turno
   → O: before_agent_start inyecta system prompt con Cell
5. turn_end → graba si resolvió o no → Weight.update()
```

**Limitaciones:**
- Solo funciona en pi.agent — no es base multi-harness
- Sistema de eventos menos documentado que Claude Code hooks
- No hay eventStore queryable como OpenCode — la auditoría es menos estructurada

### 5.5 Claude Code — Integration path

**Capacidad más madura:**
- 20+ lifecycle hooks con `async:true` para background sin blocking
- `additionalContext` hasta 10,000 caracteres por hook
- Channels para push desde MCP server (push directo a sesión activa)
- OpenTelemetry exports (traces, metrics, logs)
- `UserPromptSubmit` + `additionalContext` → inyección en prompt submission

**Flujo:**
```
1. Hook FileChanged + PostToolUse corre en background (async:true)
2. Detector analiza evento → classify signal
3. Si signal_confident:
   → Inject via Channels MCP (push directo a sesión activa)
   → O: next UserPromptSubmit hook → additionalContext
4.additionalContext inyectado antes del siguiente turno del modelo
```

**Límites:**
- 10,000 caracteres por hook. Cells más grandes → escribir a archivo, poner path en additionalContext.
- Channels requiere cuenta de pago (verificar)

### 5.6 Cursor — Integration path

**Capacidad:**
- `sessionStart` → inyecta additional_context
- `beforeSubmitPrompt` → expone full prompt pre-modelo
- `afterAgentThought` + `afterAgentResponse` → expone reasoning
- `preCompact` → detecta context pressure (token count visible)
- `postToolUse` con full input/output JSON + duration
- MCP server → tools/prompts/resources/elicitation
- Hooks en formato Claude Code (compatible)

**Flujo:** Similar a Claude Code. preCompact permite detectar cuando el contexto está lleno y hacer proactive dump de cells al storage antes de compaction.

### 5.7 MiMoCode — UI/UX harness ONLY

**Rol real: detector pasivo, NO inyector.**

MiMoCode es un fork de OpenCode mejorado para Chinese market (mejor TUI, config schema rico). Sin embargo, MiMoCode NO implementó:
- Hooks públicos (pre-*/post-*)
- eventStore queryable
- `.opencode/context/` para inyección directa
- Memory tool con BM25

**Solo tiene:** `mimo run --format json` → JSONL events. Eso es todo.

**Flujo — detector pasivo SOLAMENTE:**
```
1. OVAV parserea JSONL de `mimo run --format json` como proceso side-car
2. Detecta signals via tool_use + error events
3. Si detecta error loop → crea Cell en CellStore (OVAV own storage)
4. NO puede inyectar directamente — MiMoCode no lo permite
5. Alternativa: Model self-prompting — OVAV instruye al agente a escribir su fix
   en .ovav/livemem/self_fix.md
```

**Lo que SÍ puede hacer en MiMoCode:**
- Detectar patrones de error via JSONL parsing
- Alimentar CellStore con cells derivados de errores en MiMoCode
- Eso alimenta la base de conocimiento global de OVAV MEMORY

**Lo que NO puede hacer en MiMoCode:**
- Inyección directa de memoria (no hay path)
- Auditoría de gobernanza (no hay eventStore)
- Monitoreo pasivo (no hay hooks ni cron)

---

## 6. CellStore — almacenamiento vivo

### 6.1 Estructura de archivos

```
.ovav/runtime/livemem/
├── cells/                    # Celdas individuales
│   ├── {uuid}.json          # Cell definitions
│   └── ...
├── index/                    # Índice invertido por EventSignature
│   └── sig_index.json       # map[sig] → [cell_id_1, cell_id_2, ...]
├── inject.log                # Audit log de inyecciones
└── live_profiler.json       # Estado actual del profiler
```

### 6.2 Weight y frescura

```go
// Weight evoluciona basado en si la inyección ayudó.
func (c *Cell) RecordOutcome(helped bool) {
    c.InjectionCount++
    if helped {
        c.Weight = min(1.0, c.Weight+0.1)
        c.LastHelpedAt = time.Now().Format(time.RFC3339)
    } else {
        c.Weight = max(0.0, c.Weight-0.05)
    }
}

// Cells debajo de 0.2 weight no se inyectan (filtro de ruido)
const MinInjectionWeight = 0.2
```

### 6.3 LLM-driven importance scoring

Al momento de guardar un Cell, se pide al modelo que calcule Weight:

```
Prompt: "Clasifica la importancia de esta información para futuro debugging:
{cell_content}
Score 0.0-1.0: 0.0 = ruido, 1.0 = crítico recordar"
```

---

## 7. Sistema de consolidación (Sleep-Time)

### 7.1 Cuándo consolidar

- Cuando CellStore excede 500 cells
- Cada 24h de tiempo idle del sistema
- Manualmente via `ovav livemem consolidate`

### 7.2 Qué hace

1. **Merge:** Cells con mismo EventSignature → merge summaries
2. **Prune:** Cells con Weight < 0.15 → архивировать (no delete, archive)
3. **Promote:** Cells con Weight > 0.8 → pasar a `internal/memory` (Card completo)
4. **Defragment:** Eliminar gaps en numeración de archivos

---

## 8. Differ from existing internal/memory

| Aspecto | `internal/memory` (v1) | `internal/livemem` (v2) |
|---------|------------------------|-------------------------|
| Timing | Sesión start + query explícita | **En vivo, durante trabajo activo** |
| Trigger | Agente pregunta | **Detector de señales de lucha** |
| Formato | Card completo (YAML) | Cell ligero (JSON, ≤500 chars summary) |
| Lookup | Full-text search | **Event signature → O(1)** |
| Storage | YAML centralizado | **Celdas individuales en filesystem** |
| Rol | Reactivo / query | **Proactivo / push** |
| Objetivo | Decisiones Governance | **Error loops, debugging vivo** |

**v1 y v2 coexisten.** v1 para decisiones de governance (estratégico). v2 para debugging en vivo (táctico).

---

## 9. Tecnología y stack

### 9.1 Stack implementado

```
Lenguaje:        Go 1.22+
Build:           go build ./...
Test:            go test ./internal/livemem/... -v
Storage:         filesystem (sin DB, sin dependencias externas)
Formato Cell:    JSON (marshaling estándar Go)
MCP server:      stdio (compatible con memory-mcp existente)
Hooks adapters:  procesos hijos (exec.Command)
Scheduler:       Go ticker + cron para consolidación
```

### 9.2 Dependencias externas

Ninguna para el core livemem. Solo stdlib Go:
- `encoding/json` — Cell serialization
- `os` — filesystem operations
- `sync` — LiveProfiler mutex
- `time` — timestamps, sliding window
- `path/filepath` — cross-platform cell paths

### 9.3 Límites operacionales

| Parámetro | Límite | Razón |
|-----------|--------|-------|
| Cell summary | ≤500 chars | No saturar context del modelo |
| Cell store | ≤10,000 cells | RAM para index en memoria |
| Weight mínimo para inyección | 0.2 | Filtrar ruido |
| Weight máximo para archivar | 0.15 | No perder historial por debajo |
| Sliding window (profiler) | 5 minutos | Detectar loops sin falsos positivos |
| Consolidation threshold | 500 cells | Balance memoria vs búsqueda |

---

## 10. Testing strategy

### 10.1 Niveles de test

**Unit tests:**
- `detector_test.go` — unit tests para SignalType classification con eventos simulados
- `cellstore_test.go` — CRUD operations, weight evolution, index rebuild
- `liveprofiler_test.go` — sliding window behavior, event expiry

**Integration tests:**
- `injector_opencode_test.go` — mock OpenCode eventStore, verificar writing a .opencode/context/
- `injector_pi_test.go` — mock pi.dev extension API
- `injector_mimocode_test.go` — parse JSONL events, verificar Cell creation

**E2E tests (manual):**
- OVAV corre en cada harness con un scenario de error loop
- Verificar que Cell se crea, se inyecta, y Weight evoluciona

### 10.2 Test de inyección de memoria

```go
// Ejemplo: detector_test.go
func TestDetector_ErrorLoop(t *testing.T) {
    profiler := NewLiveProfiler(5 * time.Minute)
    detector := NewDetector(profiler)

    // Simular mismo error 2 veces
    e1 := Event{Tool: "Write", File: "agends.md", Error: "ECONF", TS: time.Now()}
    e2 := Event{Tool: "Write", File: "agends.md", Error: "ECONF", TS: time.Now().Add(1 * time.Minute)}

    detector.Feed(e1)
    detector.Feed(e2)

    signal := detector.Classify()
    if signal != SignalErrorLoop {
        t.Errorf("expected SignalErrorLoop, got %v", signal)
    }

    // Verificar que el tercer intento no dispara again (debounced)
    e3 := Event{Tool: "Write", File: "agends.md", Error: "ECONF", TS: time.Now().Add(2 * time.Minute)}
    detector.Feed(e3)
    signal2 := detector.Classify()
    if signal2 != "" {
        t.Errorf("expected no signal (debounced), got %v", signal2)
    }
}
```

---

## 11. Error handling y rollback

### 11.1 Qué puede salir mal

| Fallo | Impacto | Mitigación |
|-------|---------|-----------|
| Cell file corrupto (JSON malformado) | Cell no se puede inyectar | Skip corrupt cells, log warning, continue |
| Index corrupto | Lookup falla | Rebuild index from cells/ on startup |
| Inyección no funciona (archivo no existe) | Silently fail | Log error, fallback a memory tool BM25 |
| Cell con Weight 0.0 se inyecta por bug | Inunda contexto con ruido | Double-check MinInjectionWeight before inject |
| Harness adapter crash | No más monitoreo | Run adapters as supervised processes, auto-restart |

### 11.2 Rollback de inyección

Si una inyección causa que el modelo entre en comportamiento inesperado:

```go
// Si en 2 turnos post-inyección el modelo reporta el mismo error:
// → Cell.weight -= 0.2 (penalización)
// → No re-inyectar esta Cell por 1 hora
// → Log en inject.log con flag "rejected"
```

---

## 12. Privacy y security

### 12.1 Cell privacy tags

```go
const (
    CellPublic    = "public"    // puede inyectarse en cualquier harness
    CellProject   = "project"   // solo para este proyecto
    CellSensitive = "sensitive" // solo para este usuario
    CellSecret    = "secret"    // NUNCA se inyecta, solo archival
)
```

### 12.2 Secrets nunca se inyectan

Regla absoluta: Cells con PrivacyTag secret/identity → NUNCA se inyectan. Solo se usan para archival/search interno.

### 12.3 Harness scope

```json
{
  "scope": ["opencode", "pi", "claude", "cursor"],
  "_comment": "NOT mimocode — MiMoCode no tiene inyección directa"
}
```

---

## 13. Métricas y validación

### 13.1 Métricas

| Métrica | Descripción | Target |
|---------|------------|--------|
| `livemem.injections` | Total de inyecciones feitas | — |
| `livemem.helped_rate` | % de inyecciones útiles confirmadas | >70% |
| `livemem.loop_prevented` | Loops de error evitados | — |
| `livemem.latency_ms` | Tiempo signal → inyección completa | <500ms |
| `livemem.false_positives` | Inyecciones que no ayudaron | <15% |
| `livemem.cell_count` | Cells activos en store | ≤10,000 |
| `livemem.mem_bytes` | Memoria RAM del index | <50MB |

### 13.2 Auto-validation

Después de cada inyección, el siguiente tool_use del modelo se marca como `post_injection_result`. Si el modelo resuelve el error en ≤2 turnos post-inyección → `helped=true`. Si sigue en error después de 3 turnos → `helped=false`.

---

## 14. API pública

### 14.1 CLI

```bash
ovav livemem status          # cells count, last injection, hit rate
ovav livemem inject <cell_id> # force injection
ovav livemem log            # injection audit log
ovav livemem consolidate    # run consolidation manually
ovav livemem tune <sig> <weight>  # adjust cell weight
ovav livemem list           # list all cells with weight/scope
ovav livemem archive <id>  # archive a cell (soft delete)
```

### 14.2 MCP tools (existing memory-mcp extended)

```json
{
  "name": "livemem_inject",
  "description": "Force OVAV to inject a specific cell into current session",
  "input": {
    "cell_id": "uuid",
    "harness": "opencode|pi|claude|cursor|mimo",
    "force": false
  }
}
```

---

## 15. Roadmap de implementación

### Fase 1: Core livemem (2-3 días)
- [ ] `internal/livemem/cellstore/cells.go` — Cell type + filesystem CRUD
- [ ] `internal/livemem/cellstore/index.go` — EventSignature index
- [ ] `internal/livemem/detector.go` — SignalType + LiveProfiler
- [ ] `internal/livemem/monitor.go` — EventStream interface
- [ ] `internal/livemem/injector.go` — HarnessInjector interface
- [ ] `internal/livemem/writer.go` — Cell creation from agent events
- [ ] Unit tests para detector + cellstore
- [ ] Build + test passing

### Fase 2: OpenCode adapter (BASE — 2 días)
- [ ] `internal/livemem/harness/opencode.go` — OpenCode adapter
- [ ] OpenCode eventStore polling
- [ ] `.opencode/context/ovav_live.md` injection
- [ ] OpenCode hooks registration
- [ ] Integration test con mock eventStore

### Fase 3: pi.dev extension (2 días)
- [ ] `internal/livemem/harness/pi.go` — pi.dev extension adapter
- [ ] `~/.pi/agent/extensions/ovav-memory/` — TypeScript extension scaffold
- [ ] before_agent_start + tool_result event wiring
- [ ] pi.sendMessage() injection path
- [ ] pi.appendEntry() persistence path

### Fase 4: Claude Code + Cursor (2 días)
- [ ] `internal/livemem/harness/claude.go` — Claude Code hooks adapter
- [ ] `internal/livemem/harness/cursor.go` — Cursor adapter
- [ ] Claude Code Channels MCP server setup
- [ ] OpenTelemetry integration para Claude Code

### Fase 5: MiMoCode detector only (1 día)
- [ ] `internal/livemem/harness/mimocode.go` — JSONL parser (detector only)
- [ ] Model self-prompting fallback strategy (only viable injection path)

### Fase 6: CLI + consolidation + metrics (1 día)
- [ ] `cmd/livemem/` CLI entrypoint
- [ ] Sleep-time consolidator
- [ ] Metrics collection (Prometheus-style)
- [ ] `ovav livemem` CLI commands

### Fase 7: Integration con existing memory (1 día)
- [ ] Bridge: Cell → Card promotion (Weight > 0.8)
- [ ] Shared `classifier.go` para privacy tags
- [ ] Unified `ovav memory` CLI con v1 + v2

---

## 16. Open questions

1. ¿Soporta `.opencode/context/` hot-reload o requiere restart del agente?
2. ¿Claude Code Channels requiere cuenta de pago o está en todos los planes?
3. ¿El límite de 10,000 caracteres en additionalContext es por hook o total?
4. ¿MiMoCode memory store en `/Labs/mimocode/data/memory/` es writable por procesos externos?
5. ¿pi.dev extensions support hot-reload sin restart de sesión?
6. ¿pi.sendMessage() tiene límite de tamaño por message?
7. ¿pi.appendEntry() persiste a través de sesiones reiniciadas?
8. ¿pi.dev tiene event para detectar cuando el modelo está "pensando" / en loop?
9. ¿OpenCode eventStore soporta SSE/websocket subscription o solo polling?

---

## 17. Sources

| Hallazgo | Fuente |
|----------|--------|
| U-shaped positional bias | arXiv:2603.10123 |
| Error repetition 47%→12% | arXiv:2503.14442 (F5) |
| Correction latency -61% | arXiv:2507.00085 (F5) |
| Context overflow -81% | Anthropic production agents 2025 (F5) |
| Recall latency 17× faster | arXiv:2502.07381 (F5) |
| OpenCode hooks + context injection | opencode.ai/docs/hooks, /context-injection (F6) |
| OpenCode eventStore | opencode.ai/docs/events (F6) |
| OpenCode memory tool | opencode.ai/docs/memory (F6) |
| pi.dev extensions | pi.dev/docs/latest/extensions |
| pi.dev JSON mode | pi.dev/docs/latest/cli |
| Claude Code hooks | code.claude.com/docs/en/hooks.md |
| Claude Code Channels | code.claude.com/docs/en/channels.md |
| Cursor hooks | cursor.com/docs/hooks |
| MiMoCode JSONL | mimo.xiaomi.com/cli-subcommands |
| Memory taxonomy gap (no push category) | Memory Mechanism Survey 2024, arXiv:2404.13501 (F2) |
| All frameworks reactive | F2 findings (MemGPT, Reflexion, Generative Agents, CRAG) |
| pi.dev research | pi.dev webFetch Aug 2026 |
