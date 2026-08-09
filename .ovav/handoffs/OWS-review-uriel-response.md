# OWS Infrastructure Review — Uriel (DevOps & Infrastructure Lead)

**Revisado:** 2026-06-18 · **Spec:** OWS_SPEC.md v1.1 · **Código:** Fase 1 (`go-runtime/internal/ows/`)  
**Veredicto general:** ✅ Arquitectura viable. 3 riesgos bloqueantes, 5 recomendaciones pre-producción.

---

## Q1: `git maintenance run` en `ows` — ¿Seguro en CI/CD?

**🔴 Veredicto: PELIGROSO en CI/CD si no se configura. Debe ser opt-in con flag explícito.**

### Análisis técnico

`ows` ejecuta 3 operaciones de `git maintenance`:

| Subcomando | Riesgo en CI | Mecanismo de falla |
|---|---|---|
| `gc` | 🔴 ALTO | Lock exclusivo sobre `.git/`. Si dos jobs concurrentes comparten repo → corrupción. Duplica temporalmente el uso de disco (repack). |
| `commit-graph` | 🟡 MEDIO | Escribe `.git/objects/info/commit-graphs/`. Generalmente seguro, pero falla si `.git/objects/info/` es read-only en containers. |
| `prefetch` | 🔴 ALTO | Requiere `remote.origin.fetch` configurado. En CI con shallow clones (`depth: 1`), prefetch intenta negociar objetos que no existen → falla con error confuso. |

### Escenarios de falla concretos

1. **GitHub Actions con `actions/checkout@v4`**: el default es `fetch-depth: 1`. `git maintenance run` no tiene utilidad y `prefetch` falla porque no hay historia para negociar.

2. **GitLab CI con `GIT_STRATEGY: fetch`**: dos jobs en paralelo sobre el mismo repo → `git gc` compite por el lock de `.git/objects/pack/` → uno de los dos jobs falla con `fatal: unable to write new index file`.

3. **Docker con imagen `golang:alpine`**: Alpine no incluye `git` por defecto. Si `ows` se ejecuta dentro del contenedor de build y `git` no está instalado, el comando falla completamente.

### Recomendación

```
🔧 REQUERIDO: Modificar CommandRegistry para `ows`:
   - Agregar flag --no-maintenance (default: true en CI)
   - Agregar flag --ci (skips gc + prefetch, solo prune local)
   - Detectar CI automáticamente: os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") != ""

   Flujo propuesto:
   ows --ci          → fetch --all --prune + worktree prune (sin maintenance)
   ows               → full sync para dev local
   ows --aggressive  → incluye gc (solo dev local, explícito)
```

**Acción:** Agregar campo `CISafe bool` a `Command` struct. `ows` con `CISafe: false` por default. El handler debe detectar CI environment y degradar gracefulmente.

---

## Q2: SQLite embebido (`modernc.org/sqlite`) — ¿Requisitos de filesystem?

**🟢 Veredicto: Compatible con containers y CI. 2 edge cases que requieren manejo explícito.**

### Lo que hace bien el código actual

```go
// audit.go:28 — Configuración correcta
db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
```

| Parámetro | Correcto | Explicación |
|---|---|---|
| `_journal_mode=WAL` | ✅ | Mejor concurrencia, readers + 1 writer simultáneo |
| `_busy_timeout=5000` | ✅ | 5s timeout para contención. Adecuado para 5-10 worktrees concurrentes |
| `os.MkdirAll(dbDir, 0755)` | ✅ | Crea `.ovav/ows/` con permisos estándar |

### Edge cases que requieren manejo

#### 2.1 WAL en filesystems de red/overlay

Los archivos auxiliares de WAL (`audit.db-wal`, `audit.db-shm`) se crean junto a `audit.db`. En estos entornos hay problemas:

| Entorno | Problema |
|---|---|
| **Docker overlay2** | `fsync()` en el WAL puede degradar 10x en I/O. Mitigación: `_synchronous=NORMAL` en lugar de `FULL`. |
| **NFS / CIFS** | SQLite no soporta locking sobre NFS. Si `.ovav/` está montado en red → corrupción silenciosa. |
| **Read-only rootfs** | Típico en Kubernetes con `readOnlyRootFilesystem: true`. `.ovav/ows/` debe ser un volume mount writable. |
| **tmpfs (`/tmp`)** | Si `ovavRoot` apunta a `/tmp`, el DB se pierde al reiniciar. No es crítico (es audit, no estado de negocio), pero debe documentarse. |

#### 2.2 Inicialización en `driver.go`

```go
// driver.go:14 — init() abre :memory: para verificar driver
func init() {
    db, err := sql.Open(DriverName, ":memory:")
    if err == nil { db.Close() }
}
```

**Problema:** Este `init()` se ejecuta en todo binary que importe `package ows`, incluso si nunca usa SQLite. Si el driver no está enlazado correctamente, el error se traga silenciosamente (`if err == nil`). La verificación real ocurre en `OpenAudit()` con `_journal_mode=WAL`.

**Recomendación:** Eliminar el `init()` en `driver.go` y hacer la verificación en `OpenAudit()` con un ping explícito:

```go
func OpenAudit(ovavRoot string) (*AuditDB, error) {
    // ... existing setup ...
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("ows: sqlite driver not available: %w", err)
    }
    // ...
}
```

### Checklist pre-producción para SQLite

```
☐ Agregar _synchronous=NORMAL para entornos containerizados (opcional, vía DSN param)
☐ Documentar que .ovav/ows/ requiere filesystem writable (no read-only rootfs)
☐ Eliminar init() driver check → mover a OpenAudit con Ping()
☐ Agregar test de integración: abrir DB en /tmp, verificar que -wal y -shm se crean correctamente
☐ Considerar opción de memoria para CI: sql.Open("sqlite", ":memory:?_journal_mode=WAL")
```

---

## Q3: Event bus con `fsnotify` + JSON — ¿Escala a 5+ worktrees?

**🟡 Veredicto: Escala para 5-10 worktrees. 3 edge cases que deben resolverse antes de Fase 6.**

### Análisis de capacidad

| Componente | Límite | OWS estimado | ¿Ok? |
|---|---|---|---|
| inotify watches (Linux) | 8192 por usuario (default) | ~10-20 watches | ✅ |
| kqueue (macOS) | Ilimitado (kernel) | ~10-20 watches | ✅ |
| Tasa de eventos | ~1000/seg (fsnotify overhead) | ~1-5 eventos/minuto | ✅ |
| Race (escritura concurrente) | Sin protección en spec actual | Riesgo real | 🔴 |

### Edge cases identificados

#### 3.1 Atomicidad de escritura — CRÍTICO

El spec dice "archivos JSON en `.ovav/events/`" pero no especifica el protocolo de escritura. Si el escritor hace:
```
write(".ovav/events/event-001.json", json)  // ← NO atómico
```
El watcher puede leer un archivo incompleto → JSON mal formado → el agente crashea.

**Solución requerida (patrón estándar):**
```
1. Escribir a archivo temporal: .ovav/events/.tmp-event-001.json
2. fsync() el archivo temporal
3. rename(".ovav/events/.tmp-event-001.json", ".ovav/events/event-001.json")
   ↑ atómico en POSIX. fsnotify dispara IN_MOVED_TO → archivo completo garantizado.
```

#### 3.2 Cleanup de archivos — necesario

El spec no menciona quién borra los archivos de eventos. Sin cleanup, `.ovav/events/` crece indefinidamente (aunque lento, ~KB/día).

**Solución:** El watcher borra el archivo después de procesarlo exitosamente. Si falla, el archivo queda como "dead letter" para diagnóstico.

#### 3.3 Event ordering en multi-worktree

Si 2 worktrees emiten eventos simultáneamente (ej. dos `owd` en paralelo), los watchers ven 2 archivos creados. El orden de procesamiento depende del orden de `readdir()` del kernel, NO del orden cronológico real.

**Para OWS esto es aceptable** porque los eventos son notificaciones (no operaciones transaccionales). Pero debe documentarse: los agentes NO deben asumir orden causal entre eventos.

#### 3.4 Alternativa evaluada: SQLite como event bus

Ventaja: transacciones ACID, orden garantizado, sin race de archivos. Desventaja: los agentes necesitan polling o un mecanismo de notificación. `fsnotify` es más simple para el caso de uso actual (1-5 worktress).

**Recomendación:** Mantener `fsnotify` para Fase 6. Si en producción se detecta >20 worktrees concurrentes, migrar a SQLite-based events con NOTIFY.

### Checklist pre-producción para Event Bus

```
☐ Implementar write-then-rename atómico para eventos JSON
☐ Naming convention: .ovav/events/{timestamp_ns}-{uuid}.json para unicidad
☐ Watcher debe filtrar solo archivos .json (ignorar .tmp-*)
☐ Watcher borra archivo después de procesar (con opción --keep-events para debug)
☐ Documentar: no garantía de orden causal entre eventos de diferentes worktrees
```

---

## Q4: Cola offline — ¿Git commit o estado local?

**🔴 Veredicto: Estado local (SQLite). Sincronización vía git crea más problemas de los que resuelve.**

### Comparación de estrategias

| Criterio | Cola local (SQLite) | Cola vía git (commit pending.json) |
|---|---|---|
| Funciona offline | ✅ Siempre | ❌ Necesita conectividad para commit |
| Merge de colas | ⚠️ Requiere lógica de merge al reconectar | ✅ Git mergea el JSON |
| Consistencia | ✅ Transaccional (SQLite) | ⚠️ Merge conflicts en JSON |
| Simplicidad | ✅ 1 archivo, 1 writer | ❌ N+1 writers, N posibles conflictos |
| Seguridad | ✅ Validación al ejecutar | ❌ Cola viaja entre worktrees → surface de ataque ampliada |

### Análisis del escenario crítico: 2 worktrees offline → ambas quieren mergear

**Problema planteado:** Worktree A (task/add-cache) y Worktree B (task/refactor-db) están offline. Ambas ejecutan `owd` → sus merges se encolan. Al reconectar, ¿quién mergea primero?

**Análisis de resolución:**

```
Worktree A se conecta primero:
  1. Ejecuta su cola: owd task/add-cache ✅ (merge exitoso a develop)
  2. Sincroniza develop actualizado
  3. Detecta cola de Worktree B (vía fetch de remote)
  4. Worktree B se conecta después:
     a. Intenta owd task/refactor-db
     b. Rebase contra develop (ahora incluye cambios de A)
     c. Si conflicto → DIRTY (el mismo flujo que si estuviera online)
     d. Si sin conflicto → merge exitoso ✅
```

Este es el comportamiento CORRECTO: el segundo merge siempre se rebasea contra el develop actualizado, igual que si ambas worktrees estuvieran online.

La clave es que **la cola se valida al momento de ejecutar, no al momento de encolar**. El spec actual no menciona esto explícitamente.

### Recomendación de implementación

```
🔧 REQUERIDO para §13.3 Offline-first:

Estructura de cola en SQLite (extensión de audit.db):

CREATE TABLE pending_ops (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    command     TEXT NOT NULL,        -- "owd", "owx"
    worktree    TEXT NOT NULL,
    args        TEXT,                 -- JSON con argumentos
    queued_at   TEXT NOT NULL,
    status      TEXT DEFAULT 'pending', -- pending|executing|done|failed
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_msg   TEXT
);

Protocolo de reconexión:

1. Detectar conectividad: probe TCP a remote URL (git remote get-url origin)
2. Al reconectar:
   a. Cargar pending_ops WHERE status='pending' ORDER BY queued_at ASC
   b. Para cada op:
      - Validar precondiciones: ¿el worktree existe? ¿su branch está actualizada?
      - Si precondiciones fallan → status='failed', notificar al owner
      - Si precondiciones ok → ejecutar comando normalmente
      - El comando pasa por el mismo policy engine y state machine que online
      - Si falla → retry_count++, si < max_retries → re-queue
   c. Emitir evento RECONNECT_COMPLETED con resumen de operaciones
3. La cola es ESTRICTAMENTE LOCAL. No se comparte entre worktrees.
4. El orden de ejecución es FIFO por timestamp de encolado.
```

### Implicación de seguridad (respondiendo a pregunta de Kenji)

**Riesgo:** Un atacante con acceso al filesystem podría modificar `.ovav/ows/audit.db` para inyectar operaciones maliciosas en la cola.

**Mitigaciones:**
- Cada operación encolada guarda el `actor` que la creó
- Al ejecutar, se verifica que el `actor` coincida con el owner del worktree
- Las operaciones pasan por el policy engine completo al ejecutarse (no bypass)
- La cola nunca ejecuta `owx emergency` sin waiver activo (waiver se valida al reconectar, no al encolar)

---

## Q5: Dependencia SQLite — ¿~500KB aceptable?

**🟢 Veredicto: Totalmente aceptable. La alternativa sin SQLite costaría más código y más bugs.**

### Análisis de tamaño

| Componente | Tamaño (aprox) | Porcentaje |
|---|---|---|
| Binario OVAV actual | ~22-25MB (estimado con go-git + bubbletea) | 100% |
| `modernc.org/sqlite` (v1.52.0) | ~500KB | ~2% |
| `modernc.org/libc` (dependencia) | ~400KB | ~1.6% |
| **Total SQLite stack** | **~900KB** | **~3.6%** |

### Qué obtienes por ese costo

Sin SQLite, necesitarías implementar manualmente:
- **Log transaccional**: Append-only JSON no es atómico. Implementar write-ahead log desde cero → ~500-800 LOC + bugs.
- **Queries**: Filtrar logs por actor, comando, fecha → cada query sería un scan lineal de archivos JSON.
- **Concurrencia**: SQLite maneja readers+writer con WAL. Con JSON necesitarías file locking (`flock`) → race conditions.
- **Integridad**: Crash durante escritura → JSON corrupto. SQLite con WAL se recupera automáticamente.

### Alternativas evaluadas

| Alternativa | Tamaño | CGO-free | ¿Viable? |
|---|---|---|---|
| `zombiezen.com/go-sqlite` | ~600KB | ✅ | API más moderna pero ecosistema menor. `modernc.org/sqlite` es más battle-tested (Signal, Fly.io). |
| `github.com/mattn/go-sqlite3` | ~200KB | ❌ Requiere CGO | Descarta — pierdes cross-compilación simple. |
| BoltDB (`go.etcd.io/bbolt`) | ~200KB | ✅ | Solo key-value. Sin queries. Migrar esquema sería manual. |
| JSON files + `flock` | 0KB externo | ✅ | ~800 LOC adicionales. Más bugs. Sin queries eficientes. |

**Conclusión:** `modernc.org/sqlite` es la decisión correcta. Los 900KB son insignificantes frente a la capacidad de auditoría que habilita.

---

## Hallazgos adicionales de infraestructura

### A1: `driver.go init()` — riesgo de startup silencioso

```go
func init() {
    db, err := sql.Open(DriverName, ":memory:")
    if err == nil { db.Close() }
}
```

Este `init()` no reporta error si el driver falla al cargarse. En un binary compilado con `-trimpath` o en un entorno donde `modernc.org/sqlite` no está correctamente linked, el error pasa desapercibido y el fallo real ocurre en `OpenAudit()`.

**Fix:** Eliminar `init()` y validar en `OpenAudit()` con `db.Ping()`.

### A2: `_busy_timeout=5000` — insuficiente para CI con paralelismo extremo

5 segundos es suficiente para 5-10 worktrees concurrentes. Si OVAV escala a 50+ agentes corriendo tests en CI simultáneamente, el timeout debería ser configurable:

```go
dbPath + fmt.Sprintf("?_journal_mode=WAL&_busy_timeout=%d", timeoutMs)
```

### A3: Backup del audit.db — no mencionado en spec

`audit.db` contiene el historial completo de operaciones. Si se corrompe, se pierde trazabilidad. Recomiendo:

```
☐ Agregar comando owv backup → VACUUM INTO 'audit-{date}.db'
☐ Antes de cada owd, hacer backup automático si audit.db > 10MB
☐ Documentar procedimiento de restauración en docs/operations/
```

### A4: Inconsistencia spec vs código — cola offline

- **Spec §13.3:** `"La cola se almacena en SQLite"` + `"pending.json"` en `.ovav/queue/`
- **Spec §11:** `"Offline queue: SQLite + JSON pending — la misma DB de auditoría almacena la cola"`

Debe resolverse a UNA ubicación canónica. Recomiendo SQLite (extensión de `audit.db` con tabla `pending_ops`) y eliminar `pending.json`.

### A5: `gitleaks` y `semgrep` en `owv` — dependencias externas no declaradas

El spec §5.4 menciona `gitleaks detect --no-git` y `semgrep` como parte del pipeline de verificación. Estas NO son dependencias Go — son binarios externos que deben estar instalados en el sistema.

**Recomendación:**
- `owv` debe verificar que `gitleaks` y `semgrep` están en `$PATH` antes de ejecutarlos
- Si no están instalados, degradar gracefulmente con warning (no bloquear)
- En CI, estos binarios deben instalarse explícitamente en el pipeline
- Agregar flag `owv --skip-security` para entornos donde no están disponibles

---

## Resumen de acciones

| # | Acción | Prioridad | Bloquea |
|---|---|---|---|
| 1 | Agregar `--ci` / `--no-maintenance` a `ows`, detección automática de CI | 🔴 ALTA | Fase 2 |
| 2 | Resolver cola offline: solo SQLite (tabla `pending_ops`), eliminar `pending.json` | 🔴 ALTA | Fase 7 |
| 3 | Implementar write-then-rename atómico para eventos JSON | 🔴 ALTA | Fase 9 |
| 4 | Eliminar `init()` en `driver.go`, mover validación a `OpenAudit()` | 🟡 MEDIA | Fase 2 |
| 5 | Documentar requerimientos de filesystem para SQLite (writable, no NFS) | 🟡 MEDIA | Fase 2 |
| 6 | Verificar disponibilidad de `gitleaks`/`semgrep` en `owv`, degradar graceful | 🟡 MEDIA | Fase 2 |
| 7 | Agregar `_synchronous=NORMAL` opcional para entornos containerizados | 🟢 BAJA | Fase 5 |
| 8 | Backup automático de `audit.db` antes de `owd` | 🟢 BAJA | Fase 5 |

---

## Veredicto final

**La arquitectura es sólida.** Las 3 dependencias externas (go-git, fsnotify, modernc/sqlite) son maduras, CGO-free, y funcionan en containers y CI sin modificaciones. Los riesgos están en la implementación, no en el diseño.

**3 cosas que haría diferente como DevOps:**
1. La cola offline iría 100% en SQLite desde el día 1, no JSON híbrido.
2. `git maintenance run` estaría deshabilitado por default en CI, requiriendo flag explícito.
3. El event bus usaría el patrón write-then-rename desde la primera línea de código.

Apruebo para Fase 2 con las 3 correcciones bloqueantes implementadas.

---

*Revisión completada por Uriel — DevOps & Infrastructure Lead, Digital Product Engineering.*  
*Reporta a: dante*
