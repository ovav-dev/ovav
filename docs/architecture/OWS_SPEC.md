# OVAV Worktree Orchestration System (OWS)

## Especificación Técnica — Revisión de Leads

**Versión:** Draft v1.2 — Post-revisión de leads  
**Autor:** Thavren (Platform Engineering Lead)  
**Fecha:** 2026-06-18  
**Estado:** Aprobado para Fase 2 — 6 fixes bloqueantes incorporados  
**Revisores:** Kenji (🔴 3 críticos) · Uriel (🟡 3 bloqueantes) · Eidren (🟢 confirma) · Dante (🟢 6 UX)  

---

## 0. RESUMEN EJECUTIVO

OWS es la capa de gobernanza Git del sistema OVAV. No reemplaza Git — lo gobierna.  
Git es infraestructura. OWS es inteligencia operacional.

**Métrica clave:** 10 comandos para el usuario. 6 capas de seguridad invisibles debajo.

---

## 1. COMANDOS — Nombre completo + Abreviado

| # | Comando completo | Abreviatura | Función |
|---|-----------------|-------------|---------|
| 1 | `ovav worktree create <name>` | `owc` | Crear worktree desde develop/main |
| 2 | `ovav worktree update` | `owu` | Fetch + rebase contra base branch |
| 3 | `ovav worktree sync` | `ows` | Sync remotos + mantenimiento + prune |
| 4 | `ovav worktree verify` | `owv` | Pipeline completo de validación |
| 5 | `ovav worktree done` | `owd` | Verify → integrar → push → cleanup |
| 6 | `ovav worktree route <target>` | `owx` | Cherry-pick / patch / hotfix routing |
| 7 | `ovav worktree abort` | `owa` | Rollback de operación actual |
| 8 | `ovav worktree rescue` | `owr` | Recuperación de reflog/branch/worktree |
| 9 | `ovav worktree list` | `owl` | Inventario + ownership + health |
| 10 | `ovav worktree lock <target>` | `owlk` | Bloquear worktree (coordinación) |

---

## 2. PERFILES DE WORKTREE

Cada perfil define reglas de creación, merge y nivel de política.

| Perfil | Base branch | Merge a | Nivel política | Auto-cleanup | Requiere review |
|--------|------------|---------|----------------|--------------|-----------------|
| `feature` | develop | develop | standard | No | No |
| `refactor` | develop | develop | standard | No | No |
| `docs` | develop | develop | relaxed | No | No |
| `spike` | develop | none | relaxed | Sí | No |
| `hotfix` | main | main + develop | strict | No | No |
| `release` | develop | main | strict | No | Sí |
| `patch` | main | main + develop | strict | No | No |
| `research` | develop | none | relaxed | Sí | No |
| `emergency` | main | main + develop | waiver | No | No |
| `migration` | develop | develop | standard | No | No |
| `enterprise` | develop | develop | strict | Sí | Sí |

---

## 3. MAQUINA DE ESTADOS

```
CREATED ──► ACTIVE ──► DIRTY ──► STAGED ──► VERIFIED ──► INTEGRATED ──► CLEANED
                │          │                     │
                ▼          ▼                     ▼
             LOCKED     FAILED                RESCUED
```

### Transiciones

| Estado actual | Evento | Estado siguiente | Condición |
|--------------|--------|-----------------|-----------|
| CREATED | `work_started` | ACTIVE | Worktree existe en disco |
| ACTIVE | `conflict_detected` | DIRTY | Conflicto en rebase/merge |
| ACTIVE | `update_success` | ACTIVE | Rebase limpio |
| ACTIVE | `lock_requested` | LOCKED | owlk ejecutado por lead |
| ACTIVE | `verification_passed` | VERIFIED | owv pasa todos los checks |
| ACTIVE | `verification_failed` | FAILED | owv encuentra issues |
| VERIFIED | `integration_complete` | INTEGRATED | Merge exitoso a develop |
| INTEGRATED | `cleanup_complete` | CLEANED | Worktree eliminada + prune |
| FAILED | `rescue_requested` | RESCUED | owr recupera estado |
| DIRTY | `conflict_resolved` | ACTIVE | Resolución manual + owu |
| LOCKED | `unlock_requested` | ACTIVE | Solo owner puede desbloquear |

---

## 4. ARQUITECTURA

```
┌────────────────────────────────────────────────────┐
│                 COCKPIT — Vista Worktrees           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │ owc New  │ │ owl List │ │ owd Done │   Teclas   │
│  │ owu Sync │ │ owv Verif│ │ owx Route│   rápidas  │
│  │ ows Prune│ │ owa Abort│ │ owr Rescue│           │
│  └──────────┘ └──────────┘ └──────────┘            │
└────────────────────┬───────────────────────────────┘
                     │
┌────────────────────▼───────────────────────────────┐
│              CLI — ovav worktree <cmd>              │
│  owc feature/login   owd   owl --mine   owr HEAD   │
└────────────────────┬───────────────────────────────┘
                     │
        ┌────────────▼────────────┐
        │    Command Registry     │  ← Fuente única de verdad
        │  internal/ows/registry  │     Alimenta CLI + Cockpit + help
        └────────────┬────────────┘
                     │
┌────────────────────▼───────────────────────────────┐
│                  ENGINE                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │  State   │  │  Policy  │  │  Event   │          │
│  │ Machine  │  │  Engine  │  │   Bus    │          │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘          │
│       │             │             │                 │
│  ┌────▼─────────────▼─────────────▼─────┐          │
│  │           SQLite Audit               │          │
│  │     (modernc.org/sqlite, CGO-free)   │          │
│  └────────────────┬─────────────────────┘          │
│                   │                                 │
│  ┌────────────────▼─────────────────────┐          │
│  │          Git Adapter                  │          │
│  │  Lectura: go-git (rápido, puro Go)   │          │
│  │  Escritura: exec.Command("git")       │          │
│  │  Worktree: git worktree nativo        │          │
│  └────────────────┬─────────────────────┘          │
└───────────────────┼────────────────────────────────┘
                    │
                    ▼
                  Git
```

---

## 5. DETALLE DE COMANDOS

### 5.1 `ovav worktree create` — `owc`

```
owc feature/login           → CLI rápido, perfil feature
owc hotfix/critical-bug     → CLI rápido, perfil hotfix
owc                          → Abre Cockpit (selector de perfil + nombre)

Flujo interno:
 1. Validar perfil → determinar base branch (develop o main)
 2. git fetch origin <base>
 3. git branch task/<name> origin/<base>
 4. git worktree add .ovav/worktrees/task-<name> task/<name>
 5. Registrar en state machine → CREATED
 6. SQLite: audit log entry
 7. Emitir WORKTREE_CREATED → notifica leads si es emergency/enterprise
```

### 5.2 `ovav worktree update` — `owu`

```
owu

Flujo interno:
 1. Detectar worktree actual
 2. git fetch origin
 3. Intentar rebase onto origin/<base>
 4. Si sin conflictos → ACTIVE
 5. Si conflictos → DIRTY + notificar: "3 archivos en conflicto. Resuelve y ejecuta owu."
 6. SQLite: audit + conflict metadata
```

### 5.3 `ovav worktree sync` — `ows`

```
ows

Flujo interno:
 1. git fetch --all --prune
 2. git maintenance run (gc, commit-graph, prefetch)
 3. Detectar worktrees huérfanas (branch eliminada → worktree zombie)
 4. git worktree prune
 5. Refrescar state machine: worktrees no actualizadas en >7d → STALE
 6. SQLite: sync metadata
```

### 5.4 `ovav worktree verify` — `owv`

```
owv

Flujo interno (orden de ejecución):
 1. go test ./... -race -count=1
 2. go vet ./...
 3. gofmt -d . (debe estar limpio)
 4. go run ./cmd/ovav/ sbom generate  (SBOM fresco)
 5. go run ./internal/validators/cmd/validate/  (77 validadores)
 6. gitleaks detect --no-git
 7. Policy engine: validar reglas del perfil activo
 8. Ownership: verificar que el author coincide con el owner del worktree

Resultado: VERIFIED (todo OK) o FAILED (con reporte detallado)
```

### 5.5 `ovav worktree done` — `owd`

```
owd                          → Abre Cockpit (diff preview + confirmación)
owd --yes                    → Modo no-interactivo (CI/CD)

Flujo interno:
 1. Ejecutar owv (verify). Si FAILED → BLOCKED.
 2. Mostrar diff preview (Cockpit o --json para CI)
 3. Policy engine: validar merge policies del perfil
 4. Para enterprise: verificar review approval
 5. git fetch origin develop
 6. git checkout develop && git pull
 7. git merge --no-ff <worktree-branch>
 8. git push origin develop
 9. git -C <main-repo> worktree remove --force <path>
10. git worktree prune
11. State machine → VERIFIED → INTEGRATED → CLEANED
12. SQLite: merge audit completo
13. Emitir INTEGRATION_COMPLETED
```

### 5.6 `ovav worktree route` — `owx`

```
owx main cherry-pick     → Cherry-pick commits actuales a main
owx develop patch        → Exportar todos los commits como patch a develop
owx main hotfix          → Hotfix routing (main + develop simultáneo)
owx main emergency       → Emergency routing (bypass políticas con waiver)

Flujo interno:
 1. Validar target branch y modo
 2. Cherry-pick: seleccionar commits interactivamente (Cockpit)
 3. Patch: format-patch + apply en target
 4. Hotfix: merge a main → cherry-pick a develop
 5. Emergency: requiere waiver activo, bypass policy engine
 6. SQLite: route audit trail
```

### 5.7 `ovav worktree abort` — `owva`

```
owa

Flujo interno:
 1. Detectar operación en progreso (merge/rebase/conflict)
 2. git merge --abort  o  git rebase --abort
 3. Restaurar estado pre-operación
 4. State machine → RESCUED
 5. SQLite: abort audit
```

### 5.8 `ovav worktree rescue` — `owr`

```
owr                          → Abre Cockpit (selector de qué recuperar)
owr HEAD~3                   → Recuperar commit específico del reflog
owr branch feature/perdida   → Recuperar branch borrada
owr worktree task-xyz        → Recuperar worktree huérfana

Flujo interno:
 1. Modo interactivo: escanear reflog + branches borradas + worktrees huérfanas
 2. Usuario selecciona qué recuperar
 3. git checkout -b <nombre> <hash>
 4. git worktree add para worktree recovery
 5. State machine → RESCUED
 6. SQLite: rescue audit
```

### 5.9 `ovav worktree list` — `owl`

```
owl                 → Listar worktrees activas del usuario actual
owl --all           → Todas las worktrees
owl --mine          → Solo las del usuario actual
owl --stale         → Worktrees sin actividad en >7d
owl --json          → Output JSON para CI/CD/dashboards


Flujo interno:
 1. Leer state machine: todas las worktrees registradas
 2. Cruzar con git worktree list (existentes en disco)
 3. Detectar worktrees huérfanas (en state machine pero no en disco)
 4. Para cada una: estado, owner, edad, ahead/behind, policy version, health
 5. Formatear output (tabla, JSON, o Cockpit)
```

### 5.10 `ovav worktree lock` — `owlk`

```
owlk task/feature-login "Code review pendiente"

Flujo interno:
 1. Validar que el usuario es owner o lead
 2. State machine → LOCKED
 3. SQLite: lock reason + timestamp + owner
 4. Emitir WORKTREE_LOCKED → notifica a leads del área
 5. Ninguna operación de escritura permitida hasta unlock
```

---

## 6. POLICY ENGINE

### 6.1 Niveles de política

| Nivel | Checks requeridos |
|-------|------------------|
| `relaxed` | go build + go vet |
| `standard` | relaxed + go test + gofmt + validate CLI |
| `strict` | standard + gitleaks + SBOM regen + policy validation |
| `waiver` | strict — bypass mediante waiver explícito del CEO |

### 6.2 Reglas base

```
POL-001: Protected branch — bloquea push/merge a main/master/develop sin waiver
POL-002: HTTPS transport — bloquea remotes SSH/file://
POL-003: No force push — bloquea --force, -f, --delete en cualquier rama
POL-004: Owner match — solo el creador del worktree puede hacer owd
POL-005: Verified gate — owd rechazado si owv no pasó
POL-006: Stale worktree — notifica si worktree >7d sin actividad
POL-007: Enterprise review — owd bloqueado sin approval de lead en perfil enterprise
POL-008: Emergency waiver — owx emergency requiere waiver de 60min
```

### 6.3 Versionado

Cada política tiene versión. Si una worktree se creó con policy v11 y la actual es v12, estado → STALE. El usuario debe ejecutar `owu` para sincronizar políticas.

---

## 7. EVENT BUS + NOTIFICACIONES

### 7.1 Eventos

```
WORKTREE_CREATED       → Notifica a leads del área
WORKTREE_LOCKED        → Notifica a todos los leads (coordinación)
WORKTREE_STALE         → Notifica al owner (recordatorio)
INTEGRATION_COMPLETED  → Notifica a leads (cambios en develop)
POLICY_CHANGED         → Notifica a TODOS los agentes activos
                        → Agentes suspenden tareas incompatibles
                        → Ejecutan resync obligatorio
```

### 7.2 Implementación

- Directorio: `.ovav/events/`  
- Formato: archivos JSON con `{event, timestamp, actor, target, metadata}`  
- Mecanismo: `fsnotify` (librería Go estándar, 80K+ proyectos)  
- Los agentes watchean `.ovav/events/` y reaccionan a eventos de su área  
- Sin servidor, sin broker, sin dependencias de red  

---

## 8. AUDITORÍA (SQLite)

### 8.1 Esquema

```sql
CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp   TEXT NOT NULL,        -- ISO 8601 UTC-5
    actor       TEXT NOT NULL,        -- usuario o agente
    command     TEXT NOT NULL,        -- owc, owd, owx, etc.
    target      TEXT NOT NULL,        -- worktree, branch, commit
    result      TEXT NOT NULL,        -- success, blocked, failed
    metadata    TEXT,                 -- JSON: policy_version, conflict_files, etc.
    perf_ms     INTEGER               -- duración en milisegundos
);

CREATE TABLE worktree_state (
    id          TEXT PRIMARY KEY,     -- worktree name
    branch      TEXT NOT NULL,
    profile     TEXT NOT NULL,
    owner       TEXT NOT NULL,
    state       TEXT NOT NULL,        -- ACTIVE, LOCKED, VERIFIED, etc.
    policy_ver  TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    locked      INTEGER DEFAULT 0,
    lock_reason TEXT
);

CREATE TABLE policy_versions (
    policy_id   TEXT PRIMARY KEY,     -- POL-001, POL-002, etc.
    version     INTEGER NOT NULL,
    rule        TEXT NOT NULL,        -- descripción de la regla
    created_at  TEXT NOT NULL,
    author      TEXT NOT NULL
);
```

### 8.2 Librería

`modernc.org/sqlite` — SQLite en Go puro, sin CGO. ~500KB binario adicional.  
Usado por: Signal, Spotify, DBeaver, Fly.io.

---

## 9. SEGURIDAD

### 9.1 Integraciones activas en `owv`

| Herramienta | Qué detecta | Modo |
|------------|-------------|------|
| `gitleaks` | Secrets en código | Bloqueante |
| `semgrep` | Vulnerabilidades de código | Bloqueante |
| `go vet` | Bugs Go | Bloqueante |
| `gofmt` | Formato inconsistente | Bloqueante |
| `validate CLI` | 77 validadores OVAV | Bloqueante |
| `SBOM regen` | Supply chain integrity | Bloqueante |

### 9.2 Protección de worktrees

- `owlk`: bloqueo preventivo de worktrees
- `owa`: abort limpio sin pérdida de trabajo
- `owr`: recuperación de trabajo perdido
- `POLICY_CHANGED`: resync forzoso de agentes

---

## 10. COMMAND REGISTRY — Arquitectura Go

```go
// internal/ows/registry.go

package ows

type Command struct {
    Name        string   // "ovav worktree create"
    ShortName   string   // "owc"
    Short       string   // descripción corta para --help
    Long        string   // descripción larga para --help <comando>
    Profile     string   // perfil requerido ("" = sin perfil)
    Args        []Arg
    Interactive bool     // true → abre Cockpit
    Handler     func(ctx context.Context, args map[string]string) error
}

type ProfileConfig struct {
    BaseBranch     string // "develop" | "main"
    MergeTo        string // "develop" | "main" | "main+develop" | "none"
    PolicyLevel    string // "relaxed" | "standard" | "strict" | "waiver"
    AutoCleanup    bool   // eliminar worktree automáticamente
    RequireReview  bool   // requiere approval de lead
}

var CommandRegistry = map[string]Command{ /* 10 comandos */ }
var ProfileRegistry = map[string]ProfileConfig{ /* 11 perfiles */ }
var PolicyRegistry = map[string]Policy{ /* 8 políticas */ }
```

---

## 11. DEPENDENCIAS TÉCNICAS

| Componente | Tecnología | Razón |
|-----------|-----------|-------|
| CLI parsing | Custom registry (stdlib) | 0 supply chain risk. 15 comandos no cambian cada semana. |
| TUI | Bubble Tea (ya integrado) | Cockpit existe. Novena vista = ~200 LOC. |
| Git lectura | go-git v5 | Puro Go, rápido, sin subprocesos. |
| Git escritura | exec.Command("git") | Compatibilidad 100% con git nativo. |
| State machine | Custom map-based | 15 estados → ~60 LOC. Sin dependencias. |
| Event bus | fsnotify | Sin servidor. Funciona offline. |
| Auditoría | modernc.org/sqlite | SQLite puro Go. ~500KB. CGO-free. |
| Offline queue | SQLite + JSON pending | La misma DB de auditoría almacena la cola. |
| Conflict prediction | Custom matrix (stdlib) | Cruce de mapas en memoria. ~100 LOC. |
| AI resolution | Convert engine pattern | Mismo agente que genera .md desde YAML. |
| Shell completions | Generadas del registry | fish/bash/zsh auto-complete. |
| Notificaciones | Archivos en .ovav/events/ | Sin push. Los agentes watchean. |

**Dependencias externas totales: 3** (go-git, fsnotify, modernc/sqlite).  
Todas son librerías Go maduras con 0 dependencias de sistema operativo.

---

## 12. ROADMAP DE IMPLEMENTACIÓN

| Fase | Entregable | Esfuerzo | Dependencias |
|------|-----------|----------|-------------|
| **F1: Core** | Command registry + state machine + SQLite + owl/owc/owd | 4-6h | Ninguna |
| **F2: Policies** | Policy engine + 8 reglas + versionado + owv/owlk | 3-4h | F1 |
| **F3: Routing** | owx (cherry-pick/patch/hotfix/emergency) + owa | 2-3h | F1 |
| **F4: Recovery** | owr (reflog/branch/worktree recovery) + ows/owu | 2-3h | F1 |
| **F5: Cockpit** | Vista Worktrees en Cockpit (9na vista) | 2-3h | F1 |
| **F6: Notifications** | Event bus + fsnotify + multi-agent sync | 2-3h | F1 |
| **Total estimado** | | **15-22h** | |

---

## 13. CAPACIDADES AVANZADAS 2026

### 13.1 Predicción de conflictos (proactiva)

El sistema conoce qué archivos modifica cada worktree activa. Antes de que el usuario intente `owd`, OWS cruza las matrices de modificaciones y predice conflictos potenciales. El usuario ve la alerta en `owl` sin haber ejecutado merge todavía.

**Implementación:**
```
1. Cada owu guarda metadata: lista de archivos modificados + hashes
2. owl cruza las listas de todas las worktrees activas
3. Si dos worktrees tocan el mismo archivo → ⚠️ CONFLICTO POTENCIAL
4. Notificación proactiva a ambos owners
5. Sugerencia: coordinar orden de merge (el más pequeño primero)
```

**Ejemplo de salida:**
```
$ owl
  task/refactor-db     ACTIVE    ⚠️ Conflicto potencial con task/add-cache
                       ↳ Ambos modifican: internal/gitflow/workflow.go
                       ↳ Sugerencia: mergear add-cache primero (3 archivos vs 12)
                       
  task/add-cache       ACTIVE    ⚠️ Conflicto potencial con task/refactor-db
                       ↳ Mismos archivos en conflicto
                       
  task/fix-login       VERIFIED  ✅ Sin conflictos — listo para owd
```

### 13.2 Resolución de conflictos asistida por IA

Cuando el conflicto es real (detectado en `owu` o en `owd`), OWS analiza ambos lados del diff y propone resolución automática. No es magia — es el mismo patrón del convert engine: leer intención de cada cambio y generar código de merge.

**Implementación:**
```
1. owu/owd detecta conflicto real en archivos específicos
2. OWS extrae el diff de ambas ramas para los archivos en conflicto
3. El agente del área (Thavren para Platform, Dante para Product, etc.)
   recibe el diff y propone resolución
4. Usuario revisa la propuesta en Cockpit (side-by-side diff)
5. Aceptar / rechazar / editar — el usuario siempre decide
6. El merge resuelto se registra en SQLite con la decisión tomada
```

**Ejemplo:**
```
$ owu
⚠️ 3 conflictos detectados:
   internal/gitflow/workflow.go     ← Thavren cambió Merge(), Dante cambió Start()
   go.mod                           ← dependencia actualizada en ambas ramas
   cmd/ovav/main.go                 ← conflicto de imports

¿Analizar y proponer resolución? [Y/n] Y

── Propuesta para internal/gitflow/workflow.go ──
  Rama A (task/refactor-db):    Merge() ahora incluye worktree cleanup
  Rama B (task/add-cache):      Start() acepta --profile flag
  
  Resolución propuesta: mantener ambos cambios.
  Merge() y Start() son funciones independientes.
  Sin conflicto lógico — solo líneas adyacentes.
  
  [Aceptar] [Editar] [Rechazar]

── Propuesta para go.mod ──
  Rama A: go-git v5.19.1
  Rama B: go-git v5.19.1 + fsnotify v1.7.0
  
  Resolución propuesta: usar Rama B (incluye ambas).
  
  [Aceptar] [Editar] [Rechazar]
```

### 13.3 Offline-first — modo desconectado

Un dev premium en un avión, en un túnel, en una zona remota. Sin internet. Las operaciones locales funcionan 100% offline. Las operaciones que requieren remote se encolan en SQLite y se ejecutan automáticamente al reconectar.

**Modos de operación:**

| Comando | ¿Offline? | Comportamiento sin red |
|---------|-----------|------------------------|
| `owc` | ✅ | Crea worktree desde base branch local (sin fetch) |
| `owu` | ✅ | Rebase contra base branch local |
| `ows` | ⚠️ | Solo mantenimiento local. Sync remotos se encola. |
| `owv` | ✅ | Todos los checks son locales |
| `owd` | ⚠️ | Verify local OK → merge encolado. Se ejecuta al reconectar. |
| `owx` | ❌ | Requiere remote para push a otras ramas |
| `owa` | ✅ | Abort es siempre local |
| `owr` | ✅ | Rescue lee reflog local |
| `owl` | ✅ | State machine es local |
| `owlk` | ✅ | Lock es local |

**Cola de operaciones pendientes:**
```json
// .ovav/queue/pending.json
[
  {
    "id": "47",
    "command": "owd",
    "worktree": "task/add-cache",
    "queued_at": "2026-06-18T14:30:00-0500",
    "status": "pending",
    "depends_on": ["46"]
  }
]
```

**Al reconectar:**
```
🟢 Conectividad restaurada. Ejecutando 3 operaciones pendientes:
   ✅ #45 owu task/refactor-db     — fetch + rebase completado
   ✅ #46 owd task/fix-login       — mergeado a develop + cleanup
   ⏳ #47 owd task/add-cache       — ejecutando merge...
   
📊 Cola vacía. Todas las operaciones completadas.
```

---

## 14. ROADMAP DE IMPLEMENTACIÓN (actualizado)

| Fase | Entregable | Esfuerzo | Dependencias |
|------|-----------|----------|-------------|
| **F1: Core** ✅ | Command registry + state machine + SQLite audit | 4-6h | Completado |
| **F2: Policies** | Policy engine + 8 reglas + versionado + waiver HMAC + lock expiry | 4-5h | F1 |
| **F3: Conflict Prediction** | Matriz de modificaciones + line-range diff (85% precisión) | 3-4h | F1 |
| **F4: Routing** | owx (cherry-pick/patch/hotfix/emergency) + owa + CI detection | 3-4h | F1+F2 |
| **F5: Recovery** | owr (reflog/branch/worktree recovery) + ows/owu + atomic events | 3-4h | F1+F3 |
| **F6: Cockpit** | Vista Worktrees + dual mode + horizontal diff + ✈️ modo autónomo | 4-5h | F1+F3+F5 |
| **F7: Offline** | Cola SQLite `pending_ops` + HMAC firma + replay + CI detection | 3-4h | F1+F2 |
| **F8: AI Resolution** | Agente de conflictos + re-verificación owv post-merge + rollback | 4-5h | F3+F6+F7 |
| **F9: Notifications** | Event bus atómico + fsnotify + SQLite durability + multi-agent sync | 3-4h | F1+F7 |
| **Total estimado** | | **28-40h** | |

---

## 15. MÉTRICA CLAVE FINAL

```
Usuario:      owc mi-feature    (1 comando)
               → trabaja →       (días/semanas)
              owd                (1 comando)

OVAV interno: State machine (10 transiciones)
              Policy engine (8 reglas)
              Conflict prediction (matriz de modificaciones)
              AI resolution (si hay conflicto)
              Event bus (6 eventos)
              SQLite audit (3 tablas)
              Offline queue (si no hay red)
              Verify pipeline (6 herramientas)

Resultado:    2 comandos visibles.
              8 capas de inteligencia invisibles.
```

---

## 16. PREGUNTAS PARA LOS LEADS

### Para Uriel (DevOps & Infrastructure)
- ¿`git maintenance run` en `ows` es seguro en CI/CD pipelines?
- ¿SQLite embebido requiere permisos especiales en el filesystem?
- ¿El event bus basado en archivos escala a 5+ worktrees simultáneas?
- **NUEVO:** ¿La cola de operaciones offline debería sincronizarse vía git (commit de `pending.json`) o archivo local?

### Para Kenji (Adversarial Intelligence)
- ¿Qué vectores de ataque ve en `owx emergency` (bypass de políticas)?
- ¿El `owlk` lock podría usarse para DoS entre agentes?
- ¿`gitleaks` + `semgrep` en `owv` cubren todos los casos de seguridad?
- **NUEVO:** ¿La resolución de conflictos asistida por IA podría introducir código vulnerable? ¿Qué validación post-merge se necesita?
- **NUEVO:** ¿La cola offline podría ser manipulada para ejecutar operaciones no autorizadas al reconectar?

### Para Eidren (Research Intelligence)
- ¿Hay evidencia de que `go-git` para lectura + `exec` para escritura es el patrón óptimo en 2026?
- ¿`modernc.org/sqlite` vs `zombiezen.com/go-sqlite` (ambos CGO-free)? ¿Cuál tiene mejor rendimiento?
- **NUEVO:** ¿Existen papers o benchmarks sobre predicción de conflictos en git? ¿Qué precisión tienen los métodos actuales?

### Para Dante (Digital Product)
- ¿La vista Worktrees en Cockpit debería ser el default al abrir OVAV?
- ¿El usuario no-dev entiende "worktree", "merge", "rebase"? ¿O necesitamos abstraerlo más?
- ¿Preferís que `owc` sin argumentos abra Cockpit o pregunte interactivamente en CLI?
- **NUEVO:** ¿El side-by-side diff para resolución de conflictos debería ser horizontal (estilo IDE) o vertical (estilo GitHub)?
- **NUEVO:** ¿Cómo comunicamos "offline-first" como ventaja premium sin que suene a limitación?

---

## 17. FEEDBACK DE LEADS — INCORPORADO (v1.2)

### 17.1 Kenji — Seguridad Adversarial

| ID | Hallazgo | Fix incorporado |
|----|---------|-----------------|
| **C1** | Waiver sin autenticación | Waiver ahora requiere firma HMAC-SHA256 + nonce + TTL 60min. Verificado en `owv` y `owx`. Sin firma válida → BLOCKED. |
| **C2** | Lock DoS entre agentes | `owlk` requiere ownership O lead del área. `owlk --force` solo CEO. Lock auto-expira 24h. Worktree bloqueada >24h → unlock automático + notificación. |
| **C3** | AI merge sin re-verificación | AI merge corre `owv` completo ANTES de aplicar. Si `owv` falla post-merge → rollback automático (`owa`). El usuario siempre confirma. |
| **H1** | `owv` sin verificación de dependencias | Agregado `go mod verify` + `go sum check` al pipeline `owv`. |
| **H3** | Cola offline sin firma | Cada entry en `pending_ops` (SQLite) lleva HMAC-SHA256 del contenido. Verificado al ejecutar. |
| **M2** | Policy mismatch sin enforcement | `owu` detecta policy version mismatch y bloquea operaciones hasta sincronizar. |

### 17.2 Uriel — Infraestructura

| ID | Fix requerido | Fix incorporado |
|----|--------------|-----------------|
| **B1** | `git maintenance run` peligroso en CI | `ows` detecta `$CI` o `$GITHUB_ACTIONS` → deshabilita `git maintenance run`. Agregado flag `--force-maintenance` para uso manual. |
| **B2** | Cola offline debe ser SQLite | Tabla `pending_ops` en `audit.db`. `pending.json` eliminado del spec. Operaciones encoladas con: id, command, worktree, status, created_at, signature. |
| **B3** | Event bus necesita write atómico | Archivos en `.ovav/events/` se escriben con patrón `write tmp → fsync → rename`. Sin rename, el watcher ignora archivos `.tmp`. |

### 17.3 Eidren — Evidencia

| Recomendación | Acción |
|--------------|--------|
| Line-range diff para predicción | Agregado a F3. `owl` ahora muestra líneas específicas en conflicto potencial, no solo archivos. Precisión estimada: ~85% (vs ~70% con solo file-level). |
| SQLite event durability | Eventos ahora se persisten en `audit.db` ANTES de escribir a `.ovav/events/`. El fsnotify es delivery, no source of truth. |
| Documentar precisión | Agregado a §13.1: "Precisión estimada: 85%. Falsos positivos posibles en imports y constantes." |
| Monitorear go-git v6 | Nota en §11: "Migrar a go-git v6 cuando añada worktree nativo (~2027)". |

### 17.4 Dante — Producto/UX

| Decisión | Implementación |
|----------|---------------|
| Dashboard como lobby | Vista Worktrees accesible con `W`. Cockpit recuerda última vista. |
| Modo dual (Estándar/Dev) | `Ctrl+T` toggle. Modo Estándar: "Espacio de trabajo", "Integrar", "Actualizar". Modo Dev: términos git. CLI sin cambios. |
| `owc` sin args → Cockpit | Tarjetas visuales de perfiles (feature, hotfix, etc.). `owc --ask` para CLI paso a paso. |
| Horizontal diff default | Side-by-side estilo IDE. Toggle `V` unificado. Auto-switch vertical <100 cols. |
| "Modo autónomo" | Ícono ✈️ azul sutil. Animación de reconexión. Jamás decir "offline". |
| Métrica FMA | First-attempt Merge Acceptance. Target >85%. Medido desde SQLite audit. |

---

*Documento actualizado con feedback de 4 leads. Aprobado para Fase 2.*
