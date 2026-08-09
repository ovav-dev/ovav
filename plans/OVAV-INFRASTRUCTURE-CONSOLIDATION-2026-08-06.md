# OVAV Infrastructure Consolidation Plan — 2026-08-06

## Executive Summary

OVAV tiene 3GitHub repos + estructura local dispersa + configs de hosting desalineados.
Este plan centraliza TODO en `ovav-dev/ovav-system` como único repo, con directorios
claros para cada superficie.

---

## Estado Actual — Mapeo Completo

### GitHub (ovav-dev org)
```
ovav-system/     ← ¿? No visible en la lista pública (¿privado? ¿archivado?)
ovav-web/        ← Landing page pública (TypeScript/Next.js)
ovav-docs/       ← Docs site pública (JavaScript/Starlight)
```

### Sistema de archivos local (`/home/braka/Systems/OVAV/`)

| Directorio | Contenido | Estado |
|---|---|---|
| `go-runtime/` | Runtime Go (cPanel, validators, memory, vault) | ✅ Sistema principal |
| `ovav-systems/` | npm packages: pi-memory, pi-extension, dist | ✅ Sub-sistema npm |
| `docs/` | Documentación Go SDK / internal | ⚠️ A integrar en `docs/` |
| `docs-site/` | Starlight/Astro docs (corresponde a `ovav-docs`) | ⚠️ Migrar a `docs/` |
| `ovav-web/` | Landing page Next.js + backend Python | ⚠️ Migrar a `landing/` |
| `landing/` | ¿? — ¿existe? | ❓ Verificar |
| `ovav/` | agents/ | ⚠️ ¿Contenido duplicado? |

### Infrastructure

| Servicio | Config | Dominios |
|---|---|---|
| **Fly.io** | `fly.toml` → `ovav-systems` app | `ovav-systems.fly.dev`, `ovav-cpanel.fly.dev`, `d678beea.ovav.dev` |
| **Fly.io Staging** | `fly.staging.toml` → `ovav-systems-staging` | staging |
| **Cloudflare Pages** | `wrangler.toml` → `ovav-systems` project | `get.ovav.dev` (landing), `docs.ovav.dev` (docs) |

### Dominios en producción (desde context_firewall_v2.go)
```
get.ovav.dev         → Landing page (Cloudflare Pages)
docs.ovav.dev        → Docs site (Cloudflare Pages)
ovav-cpanel.fly.dev  → cPanel backend (Fly.io tunnel)
d678beea.ovav.dev    → cPanel backend (Fly.io tunnel, app-specific subdomain)
api.ovav.dev         → API
cdn.ovav.dev         → CDN
mcp.ovav.dev         → MCP server
status.ovav.dev      → Status page
cpanel.ovav.dev      → ❌ ELIMINADO (comentado en wrangler.toml: "URL ofuscada anti-path-guessing")
```

### Archivos dirty (git status)
```
 M clients/opencode/themes/ovav.json     (132 líneas removidas)
 M go-runtime/internal/project/sync.go   (120 líneas cambiadas)
 M opencode.json                         (2 líneas)
 M tui.json                              (2 líneas)
```

---

## Arquitectura Objetivo

```
ovav-dev/ovav-system/           ← ÚNICO REPO
│
├── go-runtime/                 ← Sistema Go (cPanel, validators, memory, vault)
├── ovav-systems/               ← NPM packages (pi-memory, pi-extension)
├── landing/                    ← Landing page (Next.js, migrado de ovav-web/)
│   ├── frontend/               ← Next.js app
│   └── backend/                ← Python/FastAPI backend
├── docs/                       ← Documentación unificada
│   ├── site/                   ← Starlight/Astro (migrado de docs-site/)
│   └── internal/               ← Go SDK docs (migrado de docs/)
├── clients/                    ← OpenCode themes y connectors
├── tools/                      ← Herramientas auxiliares
├── .ovav/                      ← Gobernanza OVAV
├── .opencode/                  ← Skills y agentes
├── bin/                        ← Binarios compilados
├── Dockerfile.cpanel
├── fly.toml                    ← Fly.io: ovav-systems (production)
├── fly.staging.toml            ← Fly.io: ovav-systems-staging
├── wrangler.toml               ← Cloudflare Pages: landing + docs
└── VERSION
```

---

## Tareas de Ejecución

### FASE 0: Auditoría y Verificación (antes de tocar nada)

- [ ] **0.1** Verificar si `ovav-system` repo existe en GitHub (¿privado? ¿archivado?)
- [ ] **0.2** Verificar contenido de `ovav/` (agents/) — ¿es duplicado de algo?
- [ ] **0.3** Verificar si existe `landing/` en la raíz
- [ ] **0.4** Check Cloudflare Pages para ver qué projects existen (`wrangler pages project list`)
- [ ] **0.5** Check Fly.io apps (`fly apps list`)
- [ ] **0.6** Mapear TODOS los subdomains en Cloudflare DNS

---

### FASE 1: Limpieza de Git (archivos dirty)

- [ ] **1.1** Revisar diff de `clients/opencode/themes/ovav.json` — ¿cambio legítimo o error?
- [ ] **1.2** Revisar diff de `go-runtime/internal/project/sync.go` — ¿cambio legítimo?
- [ ] **1.3** Commitear cambios legítimos o revertirlos
- [ ] **1.4** Remover `plans/OVAV-PI-MEMORY-TOOLS-2026-08-06.md` (ya fue ejecutado inline)

---

### FASE 2: Consolidar Landing Page (`ovav-web/` → `landing/`)

- [ ] **2.1** Mover `ovav-web/` → `landing/`
  ```bash
  mv ovav-web/frontend landing/
  mv ovav-web/backend landing/backend
  ```
- [ ] **2.2** Actualizar `wrangler.toml` para que apunte a `landing/frontend/.next`
  (Cloudflare Pages necesita saber dónde está el build output)
- [ ] **2.3** Verificar que `docker-compose.yml` en landing sigue funcionando
- [ ] **2.4** Actualizar CI/CD de GitHub Actions si existe pipeline para landing
- [ ] **2.5** Commit: `refactor: migrate ovav-web/ → landing/`

---

### FASE 3: Consolidar Documentación (`docs-site/` + `docs/` → `docs/`)

- [ ] **3.1** Mover `docs-site/src/content/` → `docs/site/content/`
- [ ] **3.2** Mover contenido de `docs/` → `docs/internal/` (si no está ya en docs/)
- [ ] **3.3** Unificar configs de Starlight en `docs/site/astro.config.mjs`
- [ ] **3.4** Actualizar `wrangler.toml` para docs en Cloudflare Pages
  - `docs.ovav.dev` → `docs/site/` build output
- [ ] **3.5** Commit: `refactor: migrate docs-site/ + docs/ → docs/`

---

### FASE 4: Verificar y limpiar configs de hosting

#### 4.1 Wrangler.toml (Cloudflare Pages)

**Estado actual:**
```toml
name = "ovav-systems"
pages_build_output_dir = "tools/cpanel/static/dist"  # ← ❌ DESACTUALIZADO
[vars]
  API_URL = "https://ovav-systems.fly.dev"
```

**Esto hay que corregirlo:**
- `ovav-systems` project en Cloudflare Pages sirve `get.ovav.dev` (landing) y `docs.ovav.dev` (docs)
- `pages_build_output_dir` debería ser `landing/frontend/.next` para landing
- Para docs: un proyecto separado de Cloudflare Pages para `docs.ovav.dev`

**Acción CF Pages:**
- Proyecto 1: `ovav-landing` → `landing/frontend/` → `get.ovav.dev`
- Proyecto 2: `ovav-docs` → `docs/site/` → `docs.ovav.dev`

#### 4.2 Fly.toml (cPanel en Fly.io)

**Estado actual:**
```toml
app = "ovav-systems"   # App: ovav-systems en Fly.io
primary_region = "dfw"
```

**Problemas detectados:**
- `fly.toml` conecta a `d678beea.ovav.dev` vía Cloudflare Tunnel
- El binary de cPanel se build desde `Dockerfile.cpanel`
- Puerto interno: 5858

**Acciones:**
- [ ] Verificar que `Dockerfile.cpanel` existe y está correcto
- [ ] Verificar que `go-runtime/cmd/cpanel/main.go` tiene el código correcto (v5.2)
- [ ] Verificar que `memory_mcp.go` en cPanel es el relay correcto
- [ ] Test: `cd go-runtime && go build -o cpanel_test ./cmd/cpanel/`
- [ ] Commit de cualquier fix necesario

#### 4.3 Limpiar `memory_mcp.go` contaminado

Según `OVAV_CLEANUP_PLAN.md`, `go-runtime/cmd/cpanel/memory_mcp.go` (494 líneas)
contiene código contaminado del relay MCP. El código correcto está en
`go-runtime/internal/memory/mcp_server.go` (625 líneas).

- [ ] **4.3.1** Comparar ambos archivos byte-by-byte (¿son idénticos?)
- [ ] **4.3.2** Si son diferentes, entender la diferencia
- [ ] **4.3.3** Mantener el de `internal/memory/mcp_server.go` como canonical
- [ ] **4.3.4** Limpiar el `memory_mcp.go` de cPanel si es redundante

---

### FASE 5: GitHub — Consolidar repos

**Meta final:** Un solo repo `ovav-dev/ovav-system` con esta estructura.

#### Opción A (Recomendada): Si `ovav-system` ya existe y está actualizado
- Archivar `ovav-web` y `ovav-docs` en GitHub
- Hacer push de los cambios de landing/docs al repo unificado
- Agregar como submodule o subtree si es necesario

#### Opción B: Si `ovav-system` está desactualizado o no existe
- Crear `ovav-system` fresh desde el estado local actual
- Push todo
- Archivar los otros repos

**Pasos:**
- [ ] **5.1** Verificar estado de `ovav-system` en GitHub
- [ ] **5.2** Sincronizar contenido si es necesario
- [ ] **5.3** Archivar `ovav-web` (GitHub Settings → Danger Zone → Archive)
- [ ] **5.4** Archivar `ovav-docs`
- [ ] **5.5** Actualizar README principal con la estructura unificada

---

### FASE 6: Verificación final

- [ ] **6.1** `git status` → debe estar limpio
- [ ] **6.2** `ovav validate` → todos los gates verdes
- [ ] **6.3** Verificar que `get.ovav.dev` sirve la landing page
- [ ] **6.4** Verificar que `docs.ovav.dev` sirve la documentación
- [ ] **6.5** Verificar que `ovav-cpanel.fly.dev` responde en `:5858`
- [ ] **6.6** Revisar que no quedan referencias huérfanas a `cpanel.ovav.dev`
- [ ] **6.7** Verificar que `context_firewall_v2.go` tiene los dominios correctos

---

## Resumen de Cambios por Archivo

### Archivos a Mover
```
ovav-web/frontend/  → landing/frontend/
ovav-web/backend/   → landing/backend/
docs-site/src/      → docs/site/src/
```

### Archivos a Modificar
```
wrangler.toml                          # Actualizar pages_build_output_dir + domains
fly.toml                               # Verificar app name y config
go-runtime/cmd/cpanel/memory_mcp.go   # Auditar y limpiar
go-runtime/internal/project/sync.go    # ¿cambio legítimo? commit o revert
clients/opencode/themes/ovav.json      # ¿cambio legítimo? commit o revert
```

### Archivos a Eliminar (si están duplicados)
```
ovav-web/           # Ya migrado a landing/
docs-site/          # Ya migrado a docs/
OVAV_CLEANUP_PLAN.md  # Obsoleto, ya ejecutado
```

### Dominios a Verificar en Cloudflare
```
get.ovav.dev         → Cloudflare Pages (landing)
docs.ovav.dev        → Cloudflare Pages (docs)
ovav-cpanel.fly.dev  → Fly.io (cPanel tunnel)
d678beea.ovav.dev    → Fly.io (cPanel app)
api.ovav.dev         → ¿?
cdn.ovav.dev         → ¿?
mcp.ovav.dev         → ¿?
status.ovav.dev      → ¿?
cpanel.ovav.dev      → ❌ ELIMINADO (no debería existir en DNS)
```

---

## Orden de Ejecución Sugerido

```
1. Fase 0 (auditoría)          → 30 min
2. Fase 1 (git cleanup)         → 20 min
3. Fase 2 (landing)             → 45 min
4. Fase 3 (docs)               → 45 min
5. Fase 4 (hosting configs)     → 60 min
6. Fase 5 (GitHub repos)        → 30 min
7. Fase 6 (verificación)        → 30 min
─────────────────────────────────────
Total estimado                   → 4-5 horas
```

---

## Comandos de Verificación Rápida

```bash
# Estado git
git status --short

# ¿Está limpio el go runtime?
cd go-runtime && go build ./... && go vet ./...

# ¿Ovav validate pasa?
cd go-runtime && go run ./cmd/ovav/ validate

# ¿El cPanel compila?
cd go-runtime && go build -o /tmp/cpanel_test ./cmd/cpanel/

# ¿La estructura de landing es correcta?
ls landing/frontend/ landing/backend/

# ¿La estructura de docs es correcta?
ls docs/site/ docs/internal/
```
