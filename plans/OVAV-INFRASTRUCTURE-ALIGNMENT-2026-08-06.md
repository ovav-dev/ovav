# OVAV Infrastructure Clean Architecture — 2026-08-06

## Estado Actual (antes de cambios)

```
/home/braka/Systems/OVAV/  ← GitHub: ovav-dev/ovav-systems

SECCIONES PRINCIPALES (lo que existe físicamente):
├── go-runtime/           Sistema Go (cPanel, validators, memory, vault, cmd/, internal/)
├── ovav-systems/          NPM packages (pi-memory, pi-extension)
├── docs/                 Documentación interna (arquitectura, guides, implementación)
├── docs-site/            Starlight/Astro docs para docs.ovav.dev (OVAV-DOCS repo)
├── ovav-web/             Landing page + backend (OVAV-WEB repo, SEPARADO)
├── ovav/                 Agent topology YAMLs (leads, areas, teams)
├── tools/                Python tools (cpanel, mcp, validators, etc.)
├── clients/              OpenCode y Claude Code connectors
├── config/               SSH, shell, Wezterm, workstation configs
├── schemas/              JSON schemas (15+ archivos)
├── runtimes/             Harness configs: claude-code, cursor, mimocode, opencode
├── reports/              Reports varios
├── research/             Investigación (agent-memory-2026, harness-absorption, etc.)
├── bin/                  Compiled binaries + incidents
├── bab/                  EMPRESA SEPARADA "BAB" (NO pertenece a OVAV)
├── .ovav/                Gobernanza OVAV para sí mismo
├── .mimocode/            MiMoCode workflows
├── .opencode/            Skills y agentes OVAV
├── plans/                Planes activos
└── [archivos root]       README, VERSION, CHANGELOG, fly.toml, etc.
```

---

## Arquitectura Objetivo (DESPUÉS de la limpieza)

```
/home/braka/Systems/OVAV/  ← GitHub: ovav-dev/ovav-systems

UNICA VERDAD:
├── go-runtime/                  Sistema Go
│   ├── cmd/                    20+ binaries (ovav, cpanel, cockpit, memory-mcp, etc.)
│   ├── internal/
│   │   ├── agents/             ← Agent topology (LEADS, AREAS, TEAMS) (migrado de ovav/agents/)
│   │   ├── schemas/            ← JSON schemas (migrado de schemas/)
│   │   ├── validators/         80+ validators
│   │   ├── memory/             Agent memory system
│   │   ├── vault/              Vault subsystem
│   │   ├── permissions/        Permissions governor
│   │   ├── governor/           Autonomous governance cycle
│   │   ├── runtimes/           Harness configs (migrado de runtimes/)
│   │   └── [demas paquetes]
│   ├── bin/                    Compiled binaries
│   └── Makefile
│
├── ovav-systems/                NPM packages
│   ├── pi-memory/              @ovav/pi-memory (publicado en npm)
│   ├── pi-extension/           @ovav/pi-extension
│   └── dist/                   Built artifacts
│
├── landing/                     Landing page pública
│   ├── frontend/               ← Next.js (migrado de ovav-web/frontend/)
│   └── backend/               ← Python/FastAPI (migrado de ovav-web/backend/)
│
├── docs/                       Documentación unificada
│   ├── public/                ← Starlight/Astro (migrado de docs-site/)
│   │   └── src/content/       docs.ovav.dev content
│   └── internal/              ← Documentación existente (arquitectura, guías, etc.)
│
├── clients/                     Agent harness connectors
│   ├── opencode/              OpenCode connector + themes
│   └── claude-code/           Claude Code connector
│
├── config/                      Configuraciones del sistema
│   ├── wezterm/               Wezterm config (canonical)
│   ├── ssh/                   SSH profiles
│   ├── shell/                 Fish/bash configs
│   └── workstation/           Workstation setup docs
│
├── tools/                      Herramientas auxiliares (Python→Go migration pending)
│   ├── cpanel/                cPanel web UI assets
│   ├── mcp/                   MCP tools
│   ├── validators/            Python validator scripts
│   └── [demas]
│
├── bin/                         Compiled binaries e incidentes
│
├── reports/                     Reportes generados
│
├── research/                    Investigación activa
│
├── .ovav/                      Gobernanza OVAV (el sistema se gobierna a sí mismo)
├── .mimocode/                  MiMoCode workflows
├── .opencode/                  Skills y agentes
│
├── plans/                      Planes activos
├── VERSION                     Versión actual
├── README.md                  
├── CHANGELOG.md
├── fly.toml                    Fly.io production config
├── fly.staging.toml            Fly.io staging config
└── wrangler.toml               Cloudflare Pages config (landing + docs)
```

---

## Plan de Ejecución — Paso a Paso

### PASO 0: Backup completo (safety)

```bash
# Crear snapshot antes de cualquier movimiento
cd /home/braka/Systems/OVAV
cp -r . /tmp/ovav-backup-$(date +%Y%m%d%H%M%S)
```

---

### PASO 1: Mover `bab/` FUERA de la repo OVAV

`bab/` es la empresa "BAB — Building A Better" — completamente separada de OVAV.

```bash
# Crear directorio padre si no existe
mkdir -p /home/braka/BAB

# Mover bab/ completo
mv /home/braka/Systems/OVAV/bab/ /home/braka/BAB/bab-archive-$(date +%Y%m%d)

# Verificar que no queda nada de bab en OVAV
ls /home/braka/Systems/OVAV/bab/ 2>/dev/null && echo "ERROR: bab still exists" || echo "OK: bab removed"
```

---

### PASO 2: Mover `ovav/agents/` → `go-runtime/internal/agents/`

```bash
cd /home/braka/Systems/OVAV

# Crear directorio destino
mkdir -p go-runtime/internal/agents

# Mover el contenido de ovav/agents/ (NO el directorio ovav/ en sí)
mv ovav/agents/areas/ go-runtime/internal/agents/
mv ovav/agents/leads/ go-runtime/internal/agents/
mv ovav/agents/teams/ go-runtime/internal/agents/

# Verificar
ls go-runtime/internal/agents/
```

---

### PASO 3: Mover `schemas/` → `go-runtime/internal/schemas/`

```bash
cd /home/braka/Systems/OVAV

mkdir -p go-runtime/internal/schemas
mv schemas/*.schema.json go-runtime/internal/schemas/
mv schemas/*.yaml go-runtime/internal/schemas/ 2>/dev/null || true

# Verificar
ls go-runtime/internal/schemas/ | head -10
ls schemas/ 2>/dev/null | head -5 || echo "schemas/ vacío o eliminado"
```

---

### PASO 4: Mover `runtimes/` → `go-runtime/internal/runtimes/`

```bash
cd /home/braka/Systems/OVAV

mkdir -p go-runtime/internal/runtimes
mv runtimes/claude-code/ go-runtime/internal/runtimes/
mv runtimes/cursor/ go-runtime/internal/runtimes/
mv runtimes/mimocode/ go-runtime/internal/runtimes/
mv runtimes/opencode/ go-runtime/internal/runtimes/

# Verificar
ls go-runtime/internal/runtimes/
```

---

### PASO 5: Consolidar `docs/` + `docs-site/` → `docs/`

**docs/** = documentación interna (arquitectura, implementation, etc.)
**docs-site/** = Starlight/Astro para docs.ovav.dev (público)

```bash
cd /home/braka/Systems/OVAV

# docs-site/src/content → docs/public/src/content
mkdir -p docs/public
mv docs-site/src/content/ docs/public/

# El astro.config.mjs y package.json de docs-site son para el build de Starlight
# Mover el proyecto Starlight completo
mv docs-site/ docs/public/starlight-astro

# docs/ existing content → docs/internal/
# (ya está en docs/ con subdirs: architecture, implementation, etc.)
# Renombrar para claridad
# docs/ → docs/internal/  (los docs internos ya existentes)
mv docs/ docs/internal-docs-temp
mkdir docs
mv internal-docs-temp/ docs/internal

# Verificar estructura
ls docs/
ls docs/internal/
ls docs/public/
```

---

### PASO 6: Traer landing page de `ovav-web/` → `landing/`

```bash
cd /home/braka/Systems/OVAV

# ovav-web/frontend → landing/frontend
mkdir -p landing
mv ovav-web/frontend/ landing/

# ovav-web/backend → landing/backend
mv ovav-web/backend/ landing/

# ovav-web/nginx.conf → landing/
mv ovav-web/nginx.conf landing/

# ovav-web/docker-compose.yml → landing/
mv ovav-web/docker-compose.yml landing/

# ovav-web/README.md → landing/README.md
mv ovav-web/README.md landing/

# Guardar referencia al repo original (ovav-web GitHub)
echo "Source: https://github.com/ovav-dev/ovav-web (archived)" > landing/FROM_OVAV_WEB_REPO.txt

# Verificar
ls landing/
```

---

### PASO 7: Limpiar `ovav/` directorio residual

Después de mover `ovav/agents/`, el directorio `ovav/` queda vacío o solo con archivos residuales.

```bash
cd /home/braka/Systems/OVAV

# Ver qué queda en ovav/
ls -la ovav/

# Si solo tiene agents/ (ya movido), eliminar el directorio vacío
rm -rf ovav/agents/
rmdir ovav/ 2>/dev/null || echo "ovav/ no está vacío, revisar manualmente"
ls ovav/ 2>/dev/null || echo "OK: ovav/ eliminado"
```

---

### PASO 8: Actualizar wrangler.toml para nueva estructura

```bash
# Ver estado actual de wrangler.toml
cat wrangler.toml

# Actualizar paths de landing y docs
# landing: get.ovav.dev → landing/frontend/.next
# docs: docs.ovav.dev → docs/public/starlight-astro/dist
```

---

### PASO 9: Actualizar referencias en el código

Buscar y actualizar TODAS las referencias a las rutas antiguas:

```bash
# Buscar referencias a ovav/agents/
grep -r "ovav/agents" --include="*.go" --include="*.yaml" --include="*.md" -l

# Buscar referencias a schemas/
grep -r "schemas/" --include="*.go" -l

# Buscar referencias a runtimes/
grep -r "runtimes/" --include="*.go" -l

# Buscar referencias a docs-site/
grep -r "docs-site" --include="*.go" --include="*.yaml" --include="*.toml" -l

# Buscar referencias a ovav-web/
grep -r "ovav-web" --include="*.go" --include="*.yaml" --include="*.toml" --include="*.md" -l
```

---

### PASO 10: Git commit y push

```bash
cd /home/braka/Systems/OVAV

git status --short
# Esperado: muchos archivos movidos/modificados, ningún archivo perdido

git add -A
git commit -m "refactor: consolidate OVAV infrastructure — clean architecture

- Remove bab/ (separate company archive)
- Move ovav/agents/ → go-runtime/internal/agents/
- Move schemas/ → go-runtime/internal/schemas/
- Move runtimes/ → go-runtime/internal/runtimes/
- Consolidate docs/ + docs-site/ → docs/internal/ + docs/public/
- Bring landing page from ovav-web/ → landing/
- Clean up residual directories"

git push
```

---

## Verificación Post-Ejecución

```bash
# 1. Estructura correcta
ls go-runtime/internal/agents/
ls go-runtime/internal/schemas/
ls go-runtime/internal/runtimes/
ls landing/
ls docs/internal/
ls docs/public/

# 2. Ningún directorio huérfano
ls ovav/ 2>/dev/null && echo "ERROR: ovav/ still exists" || echo "OK: ovav/ gone"
ls docs-site/ 2>/dev/null && echo "ERROR: docs-site/ still exists" || echo "OK: docs-site/ gone"
ls schemas/ 2>/dev/null && echo "ERROR: schemas/ still exists" || echo "OK: schemas/ gone"
ls runtimes/ 2>/dev/null && echo "ERROR: runtimes/ still exists" || echo "OK: runtimes/ gone"

# 3. Git status limpio
git status --short

# 4. Go build sigue funcionando
cd go-runtime && go build ./... && echo "OK: go build passes"

# 5. Ovav validate
cd go-runtime && go run ./cmd/ovav/ validate
```

---

## Resumen de Movimientos

| Antes | Después |
|---|---|
| `ovav/agents/` | `go-runtime/internal/agents/` |
| `schemas/` | `go-runtime/internal/schemas/` |
| `runtimes/` | `go-runtime/internal/runtimes/` |
| `docs-site/` | `docs/public/starlight-astro/` |
| `docs/` (contenido) | `docs/internal/` |
| `ovav-web/frontend/` | `landing/frontend/` |
| `ovav-web/backend/` | `landing/backend/` |
| `bab/` | `/home/braka/BAB/bab-archive-YYYYMMDD/` |
| `ovav/` (directorio) | Eliminado (vacío tras mover agents) |

**Resultado: Un solo repo `ovav-dev/ovav-systems` con estructura limpia y coherente.**
