# OVAV v2.0.0 Release Notes

**Version:** v2.0.0
**Date:** 2026-06-15
**Codename:** Stabilization Release
**Baseline:** B23 + L0-L7 Full Stack

---

## Resumen

OVAV v2.0.0 es la primera release mayor desde v1.0.0 (2026-06-07). Cubre los segmentos
SEG-0 a SEG-5 del plan de estabilización: seguridad crítica, integración frontend-backend,
hardening de infraestructura, calidad del runtime Go, reducción de deuda Python, y hardening
de testing.

**Commits incluidos:** v1.0.0..v2.0.0
**Plataformas soportadas:** linux/darwin/windows × amd64/arm64

---

## Cambios por segmento

### SEG-0: Critical Security & Governance Fixes

| ID | Descripción | Severidad |
|---|---|---|
| C0-C1 | Fix inyección de comandos git en Go cPanel API (`strconv.Atoi` sanitization) | CRITICAL |
| C0-C3 | Inicializar OVAV Vault con AES-256-GCM (eliminar cipher XOR débil) | CRITICAL |
| C0-C4 | Corregir violación `gofmt` en `cmd/ovav/main.go` | HIGH |
| C0-C5 | Eliminar `bin/__pycache__/` del repositorio | MEDIUM |

### SEG-1: Frontend-Backend Integration

| ID | Descripción |
|---|---|
| F1-MEMORY | Fix rutas de fetch en MemorySection (API_BASE relativo → absoluto) |
| F1-PROFILES | Reescribir ProfilesSection para parsear estructura JSON de Go backend |
| F1-HEALTH | Alinear SystemSection con respuesta de health de Go (`checks` vs `returncode`) |
| F1-API-BASE | Hacer API_BASE configurable por entorno (Vite env vars) |
| F1-SSE | Re-habilitar Server-Sent Events con goroutine en backend Go |
| F1-LEGACY | Eliminar componentes legacy (Dashboard, Security, Validators, Profiles, Logs) |

### SEG-2: Infrastructure Hardening

| ID | Descripción |
|---|---|
| I2-DOCKER | Alinear Dockerfile Go version con go.mod (1.22 → 1.24) |
| I2-DOCKER-STATIC | Copiar `static/css/` a imagen runtime de Docker |
| I2-REPO-ROOT | Fix detección RepoRoot en contenedor (inyección ldflags) |
| I2-STAGING | Crear entorno staging (staging.api.ovav.dev en Fly.io) |
| I2-CI-VERIFY | Verificar y corregir workflows de GitHub Actions CI |
| I2-CI-DEPLOY | CD: auto-deploy a staging en merge a `develop` |

### SEG-3: Go Runtime Quality

| ID | Descripción |
|---|---|
| G3-CONFIG-TESTS | Tests para `internal/config` (0% → ≥70% coverage) |
| G3-DEAD-CODE | Eliminar función `contains()` no usada |
| G3-TOOL-CATALOG | Actualizar `cockpit-tui` a StatusActive |
| G3-CROSS-COMPILE | Añadir targets cross-compile para cPanel y tailor |
| G3-STRING-REPEAT | Reemplazar `stringsRepeat` custom con `strings.Repeat` stdlib |

### SEG-4: Python Debt Reduction

| ID | Descripción |
|---|---|
| P4-STALE-REFS | Eliminar referencias a archivos Python borrados en artefactos |
| P4-DEPRECATED-DIR | Eliminar directorio `tools/_deprecated/` |
| P4-SYNC-OPENCODE | Sincronizar `.opencode/` con `clients/opencode/` |
| P4-AUDIT-MIGRATION | Auditoría Python → Go: clasificar directorios (CRITICAL/MIGRATABLE/DEPRECATABLE) |

### SEG-5: Testing Hardening

| ID | Descripción |
|---|---|
| T5-FRONTEND-TESTS | Smoke tests React con vitest + @testing-library/react (≥10 tests) |
| T5-BACKEND-TESTS | Tests para Go cPanel backend con `httptest.Server` (≥40% coverage) |
| T5-EDGE-CASES | Edge case testing agresivo para install gateway (≥15 escenarios) |
| T5-E2E | Test end-to-end: CLI install → verify → rollback |

---

## Breaking Changes

### 1. Vault: XOR cipher eliminado, solo AES-256-GCM

**Impacto:** Secrets encriptados con XOR en v1.x no son legibles.

**Migración:**
```bash
# 1. Decrypt con v1.x (si es necesario):
python3 -c "from tools.security.secrets_vault import SecretsVault; v = SecretsVault(); v.export_plaintext('/tmp/secrets.json')"

# 2. Upgrade a v2.0

# 3. Re-encrypt con AES-256-GCM:
ovav vault init
ovav vault scan
ovav vault encrypt --key .ovav/vault/master.key
```

### 3. API_BASE ahora requiere variable de entorno

**Impacto:** Frontend ya no asume `https://api.ovav.dev` hardcoded.

**Migración:** Crear `.env` en `tools/cpanel/`:
```bash
VITE_API_BASE=https://api.ovav.dev   # producción
# VITE_API_BASE=http://localhost:5858  # desarrollo
```

---

## Upgrade desde v1.x

### Paso 1: Backup

```bash
# Backup del vault actual
bash tools/security/vault_backup.sh

# Backup del estado actual
git stash
```

### Paso 2: Update

```bash
git fetch origin
git checkout v2.0.0
```

### Paso 3: Migrar vault (si usabas secrets)

```bash
# Exportar secrets de v1.x (XOR)
python3 tools/security/secrets_vault.py export --output /tmp/v1_secrets.json

# Reinicializar vault con AES-256-GCM
ovav vault init --force
ovav vault import /tmp/v1_secrets.json
ovav vault encrypt --key .ovav/vault/master.key

# Limpiar plaintext
shred -u /tmp/v1_secrets.json
```

### Paso 4: Verificar

```bash
# Validar runtime
python3 tools/ovav_runtime.py validate

# Verificar vault
bash tools/security/vault_backup.sh --check

# Verificar Go runtime
cd go-runtime && go test ./... && go vet ./...
```

### Paso 5: Configurar API_BASE (frontend)

```bash
cd tools/cpanel
cp .env.example .env
# Editar VITE_API_BASE según entorno
pnpm install && pnpm build
```

---

## Binaries disponibles

| Plataforma | amd64 | arm64 |
|---|---|---|
| **linux** | `ovav-linux-amd64` | `ovav-linux-arm64` |
| **darwin** | `ovav-darwin-amd64` | `ovav-darwin-arm64` |
| **windows** | `ovav-windows-amd64.exe` | `ovav-windows-arm64.exe` |

Binarios adicionales:
- `cockpit-tui` — TUI dashboard (5 plataformas)
- `cpanel` — Control panel backend (5 plataformas)
- `tailor` — Artifact composer (5 plataformas)

Descarga:
```bash
# Linux/macOS
curl -fsSL https://releases.ovav.dev/v2.0.0/install.sh | bash

# Windows (PowerShell)
irm https://releases.ovav.dev/v2.0.0/install.ps1 | iex

# Docker
docker pull ghcr.io/ovav/cpanel:v2.0.0
```

---

## Known Issues

| ID | Severity | Descripción | Workaround |
|---|---|---|---|
| KI-001 | LOW | SSE puede desconectar tras 30min de inactividad | Reconexión automática implementada en frontend |
| KI-002 | LOW | Docker build requiere Go 1.24+ (no 1.22) | Actualizar Dockerfile si se customiza |
| KI-003 | MEDIUM | `ovav vault import` no valida formato JSON v1.x | Exportar con v1.x antes de upgrade |
| KI-004 | LOW | Frontend tests requieren Node 20+ | Verificar con `node --version` |

---

## Verificación post-install

```bash
# 1. Health check
ovav health

# 2. Go runtime
cd go-runtime && go test ./...

# 3. Vault integrity
bash tools/security/vault_backup.sh --check

# 4. Frontend build
cd tools/cpanel && pnpm build

# 5. Docker (opcional)
docker build -f Dockerfile.cpanel -t ovav-cpanel:v2.0.0 .
```

---

## Créditos

Release gestionada por Thavren (Release Manager).
Segmentos SEG-0 a SEG-5 ejecutados por Platform Engineering.

**Full changelog:** `git log v1.0.0..v2.0.0 --oneline`
