# OVAV MEMORY v3 — Prompt de Implementación Completa

## Contexto del Proyecto

OVAV es un sistema governor comercial para workstations de AI agents. OVAV MEMORY v3 implementa **inyección proactiva de memoria en vivo** en pi.dev como ambiente de desarrollo primary.

## Decisión Arquitectural Clave

**pi.dev como base, NO OpenCode ni MiMoCode.**

Razones:
- Runtime minimal — sin conflictos de runtime con OVAV
- Arquitectura TypeScript-native — lo que construyas ES el futuro OVAV CLI
- Token overhead mínimo — ~1,000 tokens vs ~6.3K de OpenCode
- Event-driven por arquitectura — no polling

## Lo que ya existe (migración inteligente, NO construcción de cero)

```
EXISTE Y SE MIGRA:
├── OVAV MEMORY v2 plan (.ovav/plans/PLAN-OVAV-MEMORY-LIVE-v2.md)
├── F0-F5 validators en Go (internal/)
├── OVAV governance plugin de MiMoCode (.ovav/source/plugins/mimocode/ovav-governance.js)
├── internal/memory/ (Card-based, 3195 LOC)
└── Privacy classifier + weight evolution

SE DESCARTA (by design):
├── bin/ovav-mcp-memory — MCP es pull-based, wrong for live
├── OpenCode permissions config — conflictúan con OVAV
└── Go runtime bindings — pi.dev es TypeScript-native
```

## Plan Principal

**Archivo:** `.ovav/plans/PLAN-OVAV-MEMORY-LIVE-v3.md`

## Estructura a Construír

```
ovav-systems/src/
├── ovav-memory/          ← Cell + CellStore + Detector
│   ├── cell.ts           ← Cell type (migrado de v2)
│   ├── cellstore.ts      ← Filesystem storage
│   ├── detector.ts       ← Struggle signals
│   ├── profiler.ts       ← Sliding window 5min
│   ├── injector.ts       ← HarnessInjector interface
│   └── privacy.ts        ← Privacy tags
├── ovav-governance/      ← F0-F5 validators
│   ├── f1_architecture.ts
│   ├── f2_infrastructure.ts
│   ├── f3_roles.ts
│   ├── f4_security.ts
│   └── f5_integration.ts
├── ovav-permissions/      ← Permission gates
│   └── gate.ts
└── ovav-audit/          ← Audit trail
    └── logger.ts

pi-extension/              ← Config para pi.dev
├── extension.json
├── tsconfig.json
└── build.sh
```

## Cell Type (migrado de v2)

```typescript
export interface Cell {
    id: string;              // uuid
    eventSignature: string;  // "agends.md:error_loop:ECONF"
    summary: string;         // ≤500 chars
    weight: number;          // 0.0-1.0
    tags: string[];
    harnessScope: string[];  // ["pi", "opencode", "claude"]
    createdAt: string;
    lastHelpedAt: string;
    injectionCount: number;
    lastInjectedAt: string;
}
```

## 6 Struggle Signals

| Signal | Detección |
|--------|-----------|
| ERROR_LOOP | Mismo error ≥2x en 5 min |
| EDIT_BURST | >10 edits en mismo archivo <2 min |
| EMPTY_RESULT | Tool empty + retry inmediato |
| CONTEXT_PRESS | Token count >80% context |
| RETRY_LOOP | Mismo tool falló ≥3x |
| STUCK | Mismo file + operation ≥3x |

## Weight Evolution

```typescript
recordOutcome(helped: boolean): void {
    this.injectionCount++;
    if (helped) {
        this.weight = Math.min(1.0, this.weight + 0.1);
        this.lastHelpedAt = new Date().toISOString();
    } else {
        this.weight = Math.max(0.0, this.weight - 0.05);
    }
}
```

## Ganancias Cuantificadas (de investigación)

| Métrica | Reactivo | Proactivo |
|---------|----------|-----------|
| Error repetition | 47% | 12% (-75%) |
| Context overflow | 8/1000 | 1.5/1000 (-81%) |
| Correction latency | 3.1s | 1.2s (-61%) |
| Recall @ 10k | 800ms | 45ms (17×) |

## Roadmap de Implementación

```
Fase 1: OVAV MEMORY core      (5-7 días)
  → cell.ts, cellstore.ts, detector.ts, profiler.ts, injector.ts, privacy.ts

Fase 2: OVAV GOVERNANCE       (3-5 días)
  → f1-f5 validators como TypeScript tools

Fase 3: OVAV PERMISSIONS      (2-3 días)
  → Permission gates con project_trust hook

Fase 4: OVAV AUDIT            (1-2 días)
  → Audit trail con event stream

Fase 5: Integration + Testing (3-5 días)
  → Full E2E test, token overhead measurement
```

## Comandos para Iniciar

```bash
# Ir al repo
cd /home/braka/Systems/OVAV

# Ver plan completo
cat .ovav/plans/PLAN-OVAV-MEMORY-LIVE-v3.md

# Ver investigación de harnesses
ls research/harness-absorption/
cat research/harness-absorption/PI-DEV.md
cat research/harness-absorption/OPENCODE.md
cat research/harness-absorption/MIMOCODE.md

# Ver plugin a migrar
cat .ovav/source/plugins/mimocode/ovav-governance.js

# Ver existing memory system
ls go-runtime/internal/memory/

# Build verificación
cd go-runtime && go build ./...
```

## Integración con pi.dev

```typescript
// En ~/.pi/agent/extensions/ovav-systems/

// before_agent_start — inyectar cells relevantes
export const beforeAgentStart: BeforeAgentStartHook = async (ctx) => {
    const relevant = cellStore.lookup(ctx.currentFile);
    if (relevant.length > 0) {
        return { systemPrompt: buildPrompt(relevant) };
    }
};

// tool_call — detectar struggle signals
export const toolCall: ToolCallHook = async (tool, file, result) => {
    const signal = detector.onToolCall(tool, file, result);
    if (signal?.inject) {
        await injector.inject(signal.inject);
    }
};

// turn_end — grabar outcome
export const turnEnd: TurnEndHook = async (ctx, result) => {
    if (ctx.lastInject) {
        const helped = result.resolved === true;
        cellStore.recordOutcome(ctx.lastInject.id, helped);
    }
};
```

## Criterio de Éxito

Al final de Fase 5:
- OVAV MEMORY v3 funcionando en pi.dev
- CellStore con ≥1 cell de prueba
- F0-F5 validators ejecutándose como tools
- Weight evolution funcionando
- Token overhead <500 por inyección

---

**Para usar:** Copiá este prompt y ejecutalo en una sesión nueva de pi.dev o del editor que prefieras para comenzar la implementación.
