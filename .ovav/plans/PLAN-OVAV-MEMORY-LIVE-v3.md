# OVAV MEMORY v3 — Pi.dev como Ambiente de Desarrollo Primary

**Versión:** 3.0-pi-native
**Fecha:** 2026-08-06
**Estado:** Plan de diseño
**Autor:** thavren (Platform Engineering)

---

## Resumen ejecutivo

OVAV MEMORY v3 es el siguiente paso después de v2. La diferencia arquitectural clave:

- **v2:** Investigación con enfoque en "cuál harness es mejor" — conclusion correcta pero incompleta
- **v3:** Migración inteligente desde lo ya construido en MiMoCode/OpenCode hacia pi.dev como ambiente de desarrollo primary

**La revelación estratégica:**
> No se construye OVAV SYSTEMS de cero en pi.dev. Se migra lo que ya existe — con el conocimiento aprendido de los conflicts y limitaciones de MiMoCode/OpenCode.

**Lo que ya existe y se migra:**
- OVAV MEMORY MCP (`bin/ovav-mcp-memory`) — reimplementar como TypeScript extension
- `internal/memory/` (Card-based, 3195 LOC) — adaptar a Cell-based
- F0-F5 validators — portar a TypeScript como tools de la extensión
- `ovav-governance.js` plugin de MiMoCode — migrar como modelo
- OWS como concepto — el workflow de gobernanza se preserva

**Por qué pi.dev como base:**
- Runtime minimal — sin conflictos de runtime con OVAV
- Arquitectura limpia — todo es extensión TypeScript
- El overhead más bajo — ~1,000 tokens vs ~6.3K de OpenCode
- Modelo de extensiones correcto — lo que se construye ahí es el núcleo de futuro OVAV CLI

---

## 1. Lo que ya existe — inventario para migración

### 1.1 OVAV MEMORY MCP existente

```
bin/ovav-mcp-memory          ← Binario Go, MCP stdio server
internal/memory/            ← 3195 LOC, Card-based
├── types.go               ← Card, Ledger, ContextPack
├── writer.go              ← SessionWriter (auto-propose/flush)
├── classifier.go           ← Privacy gate
├── recall.go             ← Query engine
├── governor.go            ← Orchestration pipeline
├── agent.go              ← AgentMemory core
└── mcp_server.go        ← MCP stdio server (625 LOC)
```

**Qué se migra:**
- Card → Cell ( lightweight, event-signature lookup )
- Privacy classifier → se preserva tal cual
- SessionWriter → se migra como `before_agent_start` hook
- Query engine → se migra como `context` event handler

### 1.2 OVAV Governance Plugin de MiMoCode

```
.ovav/source/plugins/mimocode/ovav-governance.js
```

Herramientas nativas que el agente MiMoCode puede usar:
- `ovav_validate` → F0-F5 validators
- `ovav_daily` → state summary
- `ovav_next_work` → plan resolution
- `ovav_check_integrity` → system integrity score

**Qué se migra:**
- El patrón de "herramientas como tools del agente" → mismo patrón en pi.dev
- F0-F5 validators se portan a TypeScript
- Security: env allowlist, worktree validation se preservan

### 1.3 Lecciones aprendidas de MiMoCode/OpenCode

Estas lecciones NO se migran — se usan para diseñar mejor:

```
LECCIONES APRENDIDAS (no se copia, se diseña mejor):
├── MiMoCode: JSONL es útil pero no hay hooks → diseñamos con hooks
├── OpenCode: permissions son útiles pero conflictúan con OVAV → diseñamos permisos OVAV-native
├── OpenCode: eventStore queryable es útil → lo preservamos como eventStream
├── MiMoCode: memory store sesional no es suficiente → CellStore con weight
└── General: polling gasta tokens → diseñamos event-driven desde el inicio
```

---

## 2. Arquitectura de migración a pi.dev

### 2.1 Modelo: Extensión TypeScript como OVAV Runtime

```
~/.pi/agent/extensions/ovav-systems/
├── ovav-memory.ts          ← OVAV MEMORY v3 (CellStore + Detector)
├── ovav-governance.ts     ← F0-F5 validators como tools
├── ovav-permissions.ts    ← Permission gates
├── ovav-audit.ts         ← Audit trail
└── ovav-agent.ts         ← Agent behavior integration
```

Cada archivo es una TypeScript extension module que pi.dev carga.

### 2.2 Qué se migra vs qué se rediseña

| Subsistema | Existente en | Migrar a pi.dev | Rediseñar |
|-----------|-------------|-----------------|-----------|
| Cell type | NUEVO (v2) | ✅ Migrar | No |
| CellStore | NUEVO (v2) | ✅ Migrar | No |
| Detector + LiveProfiler | NUEVO (v2) | ✅ Migrar | No |
| HarnessInjector interface | NUEVO (v2) | ✅ Adaptar a hooks pi.dev | Sí |
| F0-F5 validators | Go → ovav-governance.js | ✅ Portar a TypeScript | No |
| Privacy classifier | internal/memory | ✅ Migrar | No |
| Audit trail | hooks/audit.go | ✅ Rediseñar como eventStream | Sí |
| MCP server | bin/ovav-mcp-memory | ❌ MCP es pull-based, wrong for live | NUEVO |

### 2.3 Flujo en pi.dev

```
┌─────────────────────────────────────────────────────────────┐
│                    pi.dev runtime                           │
│                                                             │
│  Extension: ovav-systems/ovav-memory.ts                   │
│  ├── before_agent_start hook                              │
│  │   └── Inyecta Cell relevant al contexto                │
│  │                                                        │
│  ├── tool_call hook                                       │
│  │   └── Detector analiza → ¿struggle signal?            │
│  │       └── Si sí → CellStore.Lookup(sig)               │
│  │           └── Si Weight > 0.6 → Inject               │
│  │                                                        │
│  └── turn_end hook                                        │
│      └── graba outcome → Weight.update()                  │
│                                                             │
│  Extension: ovav-systems/ovav-governance.ts               │
│  └── tools: ovav_validate, ovav_daily, ovav_check_integrity│
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Cell — diseño migrado de v2

```typescript
// Cell.ts — OVAV MEMORY v3 Cell type (TypeScript)
export interface Cell {
    id: string;              // uuid
    eventSignature: string;  // "agends.md:error_loop:ECONF"
    summary: string;         // ≤500 chars — injected into context
    detailRef: string;       // path to full cell .md (on-demand)
    weight: number;          // 0.0-1.0
    tags: string[];          // ["error-recovery", "loop"]
    harnessScope: string[];  // ["pi", "opencode", "claude", "cursor"]
    createdAt: string;       // ISO 8601
    lastHelpedAt: string;
    injectionCount: number;
    lastInjectedAt: string;
}

// Privacy tags
export const PrivacyTag = {
    PUBLIC: "public",
    PROJECT: "project",
    SENSITIVE: "sensitive",
    SECRET: "secret"  // NEVER inject
} as const;

// CellStore — filesystem storage en .ovav/runtime/livemem/
export class CellStore {
    private cells: Map<string, Cell> = new Map();
    private index: Map<string, string[]> = new Map();  // sig → cellIds

    async load(): Promise<void> {
        // Load all cells from .ovav/runtime/livemem/cells/
        // Build index from eventSignature → cellIds
    }

    lookup(sig: string): Cell[] {
        const ids = this.index.get(sig) ?? [];
        return ids.map(id => this.cells.get(id)).filter(Boolean) as Cell[];
    }

    async save(cell: Cell): Promise<void> {
        // Write to .ovav/runtime/livemem/cells/{cell.id}.json
        // Update index
    }

    recordOutcome(cellId: string, helped: boolean): void {
        const cell = this.cells.get(cellId);
        if (!cell) return;
        cell.injectionCount++;
        if (helped) {
            cell.weight = Math.min(1.0, cell.weight + 0.1);
            cell.lastHelpedAt = new Date().toISOString();
        } else {
            cell.weight = Math.max(0.0, cell.weight - 0.05);
        }
    }
}
```

---

## 4. Detector — migrate de v2

```typescript
// Struggle Signals — migrados de v2
export enum SignalType {
    ERROR_LOOP = "error_loop",
    EDIT_BURST = "edit_burst",
    EMPTY_RESULT = "empty_result",
    CONTEXT_PRESS = "context_pressure",
    RETRY_LOOP = "retry_loop",
    STUCK = "stuck_in_loop"
}

// Live Profiler — sliding window de 5 minutos
export class LiveProfiler {
    private events: Map<string, { ts: Date; data: any }[]> = new Map();
    private readonly window = 5 * 60 * 1000;  // 5 minutes

    feed(file: string, operation: string, data: any): void {
        const key = `${file}:${operation}`;
        const now = Date.now();

        // Add new event
        const existing = this.events.get(key) ?? [];
        existing.push({ ts: new Date(now), data });
        this.events.set(key, existing);

        // Prune old events
        const cutoff = now - this.window;
        this.events.set(key, existing.filter(e => e.ts.getTime() > cutoff));
    }

    classify(): SignalType | null {
        for (const [key, events] of this.events) {
            if (events.length >= 3) {
                const [file, op] = key.split(":");
                if (op === "error") return SignalType.ERROR_LOOP;
                if (op === "write") return SignalType.EDIT_BURST;
                return SignalType.STUCK;
            }
        }
        return null;
    }
}

// Detector — usa tool_call hook de pi.dev
export class Detector {
    private profiler: LiveProfiler;
    private cellStore: CellStore;

    constructor(cellStore: CellStore) {
        this.profiler = new LiveProfiler();
        this.cellStore = cellStore;
    }

    onToolCall(tool: string, file: string, result: any): void {
        if (result.error) {
            this.profiler.feed(file, "error", { tool, error: result.error });
        } else if (tool === "Write" && result.empty) {
            this.profiler.feed(file, "write", { empty: true });
        }

        const signal = this.profiler.classify();
        if (signal) {
            const sig = `${file}:${signal}:${result.error ?? "noop"}`;
            const cells = this.cellStore.lookup(sig);
            const best = cells.filter(c => c.weight > 0.6).sort((a, b) => b.weight - a.weight)[0];
            if (best) {
                return { inject: best, signal };
            }
        }
        return null;
    }
}
```

---

## 5. Migración de F0-F5 validators

Los validators existentes en Go se portan a TypeScript como tools de la extensión:

```typescript
// ovav-governance.ts — F0-F5 validators como tools de pi.dev

interface ValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
}

// F0: Safety check (deprecated — usar F5)
const f0Safety: Tool = {
    name: "ovav_f0_safety",
    description: "Deprecated: use ovav_f5_integration instead",
    execute: async () => ({ valid: true, errors: [], warnings: ["F0 deprecated"] })
};

// F1: Architecture gate
const f1Architecture: Tool = {
    name: "ovav_f1_architecture",
    description: "Verify architecture decisions are documented",
    execute: async (ctx) => {
        const errors: string[] = [];
        const checks = [
            { path: ".ovav/plans/", pattern: /\.md$/, required: true },
            { path: "internal/", pattern: /\.go$/, required: true }
        ];
        for (const check of checks) {
            const exists = await fileExists(check.path);
            if (check.required && !exists) {
                errors.push(`Missing required path: ${check.path}`);
            }
        }
        return { valid: errors.length === 0, errors, warnings: [] };
    }
};

// F2: Infrastructure gate
const f2Infrastructure: Tool = {
    name: "ovav_f2_infrastructure",
    description: "Verify infrastructure files are present",
    execute: async () => {
        // Check: .gitignore, go.mod, VERSION, caps.yaml
        return { valid: true, errors: [], warnings: [] };
    }
};

// F3: Roles gate
const f3Roles: Tool = {
    name: "ovav_f3_roles",
    description: "Verify service area roles are defined",
    execute: async () => {
        // Check: service areas defined in caps.yaml
        return { valid: true, errors: [], warnings: [] };
    }
};

// F4: Security gate
const f4Security: Tool = {
    name: "ovav_f4_security",
    description: "Security validators (secrets, exfil, supply chain)",
    execute: async (ctx) => {
        // Check: no plaintext secrets, no exfil patterns
        return { valid: true, errors: [], warnings: [] };
    }
};

// F5: Integration gate
const f5Integration: Tool = {
    name: "ovav_f5_integration",
    description: "Full system integration check",
    execute: async (ctx) => {
        // Run F0-F4 in sequence
        return { valid: true, errors: [], warnings: [] };
    }
};

// Registry
export const ovavGovernanceTools: Tool[] = [
    f1Architecture,
    f2Infrastructure,
    f3Roles,
    f4Security,
    f5Integration
];
```

---

## 6. Migración del plugin MiMoCode → pi.dev

El plugin MiMoCode existente:
```
.ovav/source/plugins/mimocode/ovav-governance.js
```

Se migra como referencia para crear la versión pi.dev:

```
MIMOCODE (existente):
├── tools: ovav_validate, ovav_daily, ovav_next_work, ovav_check_integrity
├── Security: env allowlist, worktree validation
└── Integration: OVAV validates before agent acts

PI.DEV (migración):
├── Mismo patrón de tools
├── Misma security model
├── Pero usando hooks de pi.dev en vez de MiMoCode hooks
└── Y usando CellStore de OVAV MEMORY v3
```

---

## 7. Lo que NO se migra — se descarta

```
NO SE MIGRAAF (razones):
├── bin/ovav-mcp-memory          ← MCP es pull-based, wrong for live memory
├── internal/memory/mcp_server.go ← MCP server no es el patrón de pi.dev
├── Go runtime bindings            ← pi.dev usa TypeScript, no Go FFI
├── MiMoCode config.json schema  ← específico de MiMoCode
├── OpenCode permissions config    ← conflictúan con OVAV permissions
└── eventStore polling code       ← pi.dev es event-driven, no polling
```

---

## 8. Estructura de archivos en OVAV repo

```
ovav-systems/                              ← NUEVA carpetá en repo
├── src/
│   ├── ovav-memory/                     ← TypeScript source
│   │   ├── cell.ts
│   │   ├── cellstore.ts
│   │   ├── detector.ts
│   │   ├── profiler.ts
│   │   ├── injector.ts
│   │   └── privacy.ts
│   ├── ovav-governance/                 ← F0-F5 validators
│   │   ├── f1_architecture.ts
│   │   ├── f2_infrastructure.ts
│   │   ├── f3_roles.ts
│   │   ├── f4_security.ts
│   │   └── f5_integration.ts
│   ├── ovav-permissions/                ← Permission gates
│   │   └── gate.ts
│   └── ovav-audit/                     ← Audit trail
│       └── logger.ts
├── pi-extension/                         ← pi.dev extension config
│   ├── extension.json                    ← pi.dev manifest
│   ├── tsconfig.json
│   └── build.sh                         ← Compila TypeScript
└── tests/
    ├── cell.test.ts
    ├── detector.test.ts
    └── integration.test.ts

go-runtime/internal/livemem/              ← SE PRESERVA
                                               ← El core Go existe para
                                               ← cuando OVAV CLI se construya
```

---

## 9. Roadmap de migración

### Fase 1: OVAV MEMORY core (5-7 días)

**Objetivo:** Tener OVAV MEMORY funcionando en pi.dev

- [ ] `ovav-systems/src/ovav-memory/cell.ts` — Cell type
- [ ] `ovav-systems/src/ovav-memory/cellstore.ts` — CellStore con filesystem
- [ ] `ovav-systems/src/ovav-memory/detector.ts` — Detector + LiveProfiler
- [ ] `ovav-systems/src/ovav-memory/profiler.ts` — Sliding window
- [ ] `ovav-systems/src/ovav-memory/injector.ts` — HarnessInjector interface
- [ ] `ovav-systems/src/ovav-memory/privacy.ts` — Privacy tags
- [ ] `pi-extension/` — Setup de extensión pi.dev
- [ ] Unit tests para cada module
- [ ] Build passing

### Fase 2: OVAV GOVERNANCE tools (3-5 días)

**Objetivo:** F0-F5 validators como TypeScript tools

- [ ] `ovav-systems/src/ovav-governance/f1_architecture.ts`
- [ ] `ovav-systems/src/ovav-governance/f2_infrastructure.ts`
- [ ] `ovav-systems/src/ovav-governance/f3_roles.ts`
- [ ] `ovav-systems/src/ovav-governance/f4_security.ts`
- [ ] `ovav-systems/src/ovav-governance/f5_integration.ts`
- [ ] Integration con tool_call hook de pi.dev
- [ ] Unit tests

### Fase 3: OVAV PERMISSIONS (2-3 días)

- [ ] Permission gates usando `project_trust` hook
- [ ] Env allowlist validation
- [ ] Worktree validation at extension load

### Fase 4: OVAV AUDIT (1-2 días)

- [ ] Audit trail usando event stream de pi.dev
- [ ] Session events logging
- [ ] Integration con CellStore para audit cells

### Fase 5: Integration + Testing (3-5 días)

- [ ] Full integration test en pi.dev
- [ ] End-to-end test: error loop → Cell creation → Injection → Weight update
- [ ] Performance test: token overhead measurement
- [ ] Migration test: cells from Go CellStore import correctly

---

## 10. Por qué esto es mejor que empezar de cero

### Lo que ya sabemos (y no repetimos)

```
DE MI/MIMOCODE:
├── JSONL es útil para debugging pero no para inyección
├── Hooks de MiMoCode son limitados → diseñamos mejor en pi.dev
└── Memory sesional no funciona → CellStore con weight

DE OPENSEODE:
├── Permissions de OpenCode conflictúan con OVAV → diseñamos OVAV-native
├── eventStore queryable es útil → preservamos como eventStream
└── Policies de OpenCode son enterprise-only → no dependemos
```

### Lo que迁移amos (no construimos de cero)

```
CELLS:           Ya diseñados en v2, solo se portan a TypeScript
DETECTOR:        Ya diseñado en v2, solo se portan a TypeScript
F0-F5 VALIDATORS: Ya existen en Go, se portan a TypeScript
PRIVACY TAGS:    Ya diseñados, se preservan
WEIGHT SYSTEM:   Ya diseñado, se portan a TypeScript
```

### Lo que es realmente nuevo

```
SÍ HAY QUE DISEÑAR:
├── TypeScript module structure para pi.dev extensions
├── Integration con pi.dev hooks (API diferente a MiMoCode)
├── TypeScript build system (tsconfig, bundling)
├── Testing framework para TypeScript en contexto pi.dev
└── Migration path desde Go CellStore a TS CellStore
```

---

## 11. Métricas de éxito

| Métrica | Target |
|---------|--------|
| Token overhead de OVAV MEMORY en pi.dev | <500 tokens por inyección |
| Cell lookup latency | <50ms |
| Weight evolution accuracy | >70% helped rate |
| F0-F5 validation coverage | 100% de lo que validan en Go |
| Migration completeness | >90% del existing CellStore migrable |

---

## 12. Open questions

1. ¿Cómo se migran los Cells existentes del Go CellStore al TypeScript CellStore?
2. ¿pi.dev extensions pueden hacer file I/O arbitrario o está sandboxed?
3. ¿Hay limit de tamaño de la extensión TypeScript?
4. ¿Cómo se hace hot-reload de la extensión durante desarrollo?
5. ¿El extension API de pi.dev permite tool descriptions dinámicas?

---

## 13. Dependencies con investigación previa

Este plan se basa en:

- `research/harness-absorption/OPENCODE.md` — governance capabilities
- `research/harness-absorption/PI-DEV.md` — pi.dev extension API
- `research/harness-absorption/MIMOCODE.md` — MiMoCode plugin pattern
- `.ovav/plans/PLAN-OVAV-MEMORY-LIVE-v2.md` — diseño v2 (Cell, Detector, CellStore)
- `.ovav/source/plugins/mimocode/ovav-governance.js` — plugin a migrar
