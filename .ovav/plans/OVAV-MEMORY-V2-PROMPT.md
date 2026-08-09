# PROMPT — OVAV MEMORY v2 Implementation Advisor

## Contexto

OVAV es un sistema governor comercial para workstations de AI agents. OVAV MEMORY v2 es un nuevo subsystem que implementa **inyección proactiva de memoria en vivo** — monitorea el contexto del agente y **inyecta memoria relevante en el momento exacto de lucha**, antes de que el modelo pida.

## Investigación existente

Toda la investigación está en:
- **Plan completo:** `/home/braka/Systems/OVAV/.ovav/plans/PLAN-OVAV-MEMORY-LIVE-v3.md` (latest)
- **Reporte síntesis:** `/home/braka/Systems/OVAV/research/live-memory-validation/REPORT.md`
- **Findings F1-F8:** `/home/braka/Systems/OVAV/research/live-memory-validation/findings/`
- **Research previo:** `/home/braka/Systems/OVAV/research/agent-memory-2026/` (5 findings)

## Hallazgos clave para leer primero

### Lo que propone OVAV MEMORY v2

```
Modelo trabaja en agends.md → OVAV MEMORY detecta (vivo, background)
    ↓
Patrón de error/estrés identificado (Struggle Signals)
    ↓
Busca en cell store → encuentra fix implementado previamente
    ↓
Inyecta contexto relevante AL MOMENTO, antes del siguiente turno
    ↓
Modelo valida → toma el dato → resuelve sin repetir el error
```

### Arquitectura

```
internal/livemem/              ← NUEVO — live proactive layer
├── cellstore/cells.go        ← Cell type (JSON, ≤500 chars, O(1) lookup)
├── detector.go                ← 6 Struggle Signals (error_loop, edit_burst...)
├── monitor.go                 ← EventStream interface
├── injector.go               ← HarnessInjector interface
└── harness/
    ├── opencode.go           ← BASE: eventStore + .opencode/context/
    ├── pi.go                 ← PREMIUM: before_agent_start + pi.sendMessage()
    ├── claude_code.go        ← hooks + Channels
    ├── cursor.go             ← sessionStart + beforeSubmitPrompt
    └── mimocode.go           ← JSONL parsing (detector only)
```

### Cell type (diseño)

```go
type Cell struct {
    ID             string    `json:"id"`            // uuid
    EventSignature string    `json:"sig"`           // "agends.md:error:ECONF"
    Summary        string    `json:"summary"`        // ≤500 chars para inyección
    DetailRef      string    `json:"detail_ref"`     // path a full cell (on-demand)
    Weight         float64   `json:"w"`            // 0.0-1.0, LLM-assessed
    Tags           []string  `json:"tags"`          // ["error-recovery", "loop"]
    HarnessScope   []string  `json:"scope"`         // ["opencode", "pi", "claude"]
    CreatedAt      string    `json:"created_at"`
    LastHelpedAt   string    `json:"last_helped"`
    InjectionCount int       `json:"inj_count"`
    LastInjectedAt string    `json:"last_injected"`
}
```

### Decisión arquitectural — LEER ESTO

**OpenCode = BASE de OVAV SYSTEM** (NO pi.dev).

Razones:
1. **eventStore queryable** — log estructurado por source/type/sessionId — fundamental para auditoría de gobernanza
2. **`.opencode/context/`** — inyección directa en cada turno sin hooks
3. **memory tool** con BM25 search
4. **cron watchdog** para monitoreo pasivo
5. **hooks pre-*/post-*** — todos los eventos de ciclo de vida observables
6. **Multi-harness** — funciona igual donde OVAV corre

**pi.dev = harness adapter premium** para cuando la inyección directa es prioridad:
- `before_agent_start` inyecta message + system prompt cada turno
- `tool_result` detecta errores en tiempo real
- Sin límite de chars documentado
- Pero solo funciona en pi.agent — no es base multi-harness

**MiMoCode = detector/recopilador solamente** — no tiene hooks públicos, no tiene eventStore, no tiene context injection. Es un fork mejorado para UX Chinese market, pero NO para gobernanza OVAV.

### Ganancias cuantificadas (reales, no ficticias)

| Métrica | Reactivo | Proactivo | Fuente |
|---------|----------|-----------|--------|
| Error repetition | 47% | 12% (-75%) | arXiv:2503.14442 |
| Latencia corrección | 3.1s | 1.2s (-61%) | arXiv:2507.00085 |
| Context overflow | 8/1000 | 1.5/1000 (-81%) | Anthropic 2025 |
| Recall @ 10k entries | 800ms | 45ms (17×) | arXiv:2502.07381 |

### 6 Struggle Signals

| Signal | Definición | Detección |
|--------|-----------|-----------|
| **Error loop** | Mismo error en mismo archivo ≥2x en 5 min | tool_use error + same file |
| **Edit burst** | >10 edits en mismo archivo en <2 min | Write tool frequency |
| **Empty resolution** | Tool empty + retry inmediato | tool result empty + same tool |
| **Context pressure** | Token count >80% context window | prompt_snapshot token estimate |
| **Repeated failure** | Mismo tool falló ≥3x mismo objetivo | PostToolUse failure pattern |
| **Stuck in loop** | Mismo file + operation ≥3x | Operation signature match |

## Qué preguntar

Dado el plan en `.ovav/plans/PLAN-OVAV-MEMORY-LIVE-v2.md`:

1. **Revisar Fase 1 del roadmap** — ¿el diseño de Cell + CellStore es correcto? ¿faltan casos edge?
2. **OpenCode adapter** — dado que OpenCode es la BASE, ¿cómo debería estructurarse el adapter para maximizar auditoría de gobernanza?
3. **Cell format** — ¿el diseño de Cell es óptimo para lookup O(1)? ¿debería ser Protobuf en vez de JSON?
4. **Weight evolution** — ¿el algoritmo de weight (+0.1 helped, -0.05 not helped) es robusto o puede gaming?
5. **Testing** — ¿cómo se prueba un sistema de inyección de memoria? ¿fuzzing? ¿simulation?
6. **Consolidation** — ¿el threshold de 500 cells para consolidación es correcto?
7. **Privacy** — ¿los privacy tags (public/project/sensitive/secret) son completos?
8. **Integración con existing** — ¿cómo debería Cell → Card promotion cuando Weight > 0.8?

## Comandos de referencia

```bash
# Ir al repo
cd /home/braka/Systems/OVAV

# Ver el plan completo
cat .ovav/plans/PLAN-OVAV-MEMORY-LIVE-v2.md

# Ver reporte de investigación
cat research/live-memory-validation/REPORT.md

# Ver findings individuales
cat research/live-memory-validation/findings/F1.md
cat research/live-memory-validation/findings/F2.md
# ... F3-F8

# Construir para verificar que todo compila
cd go-runtime && go build ./...

# Ver memory existente (v1)
ls internal/memory/
```

## Contexto de implementación

- El repo OVAV usa OWS (Worktree System) para todo feature work
- No usar `git add .` — solo `git add path/to/file`
- OVAV usa Conventional Commits para branch naming
- `internal/memory` existente (v1, 3195 LOC) = Card-based, governance estratégico
- `internal/livemem` (nuevo, v2) = Cell-based, tactical live debugging
- v1 y v2 coexisten

## Para usar este prompt

Copialo y pegalo en una nueva sesión. El asistente leerá el plan y los findings y te ayudará a implementar cualquiera de las 7 fases.
