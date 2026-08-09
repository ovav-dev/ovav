# OVAV Go Runtime — Architecture Canonical Document

> **CAPA 9 · Go Runtime v5.0**
> Documento canónico permanente. Todo cambio en go-runtime/ debe reflejarse aquí.
> Este documento NO se pierde. Es la fuente de verdad de la arquitectura Go.

---

## Stack

```
Lenguaje:     Go 1.22+
Dependencias: Cero — stdlib only
Módulo:       github.com/ovav/ovav
Build:        CGO_ENABLED=0 (static binaries)
Targets:      linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
Vet:          Zero issues
Fmt:          go fmt enforced
```

---

## Árbol de directorios

```
go-runtime/
├── cmd/
│   ├── ovav/                    # CLI pública (409 loc)
│   │   └── main.go              # 7 comandos: status, profile, config, tools, update, wv, version
│   └── cpanel/                  # Servidor control panel (14 .go)
│       ├── main.go              # Entry point
│       ├── server.go            # HTTP server + middleware
│       ├── routes.go            # API route definitions
│       ├── auth.go              # Authentication
│       ├── oauth.go             # OAuth 2.0 (Google, GitHub)
│       ├── agents.go            # Agent management
│       ├── profiles.go          # Service profile management
│       ├── system.go            # System health
│       ├── status.go            # Status endpoints
│       ├── security.go          # Security headers + CORS
│       ├── validators.go        # Native integrity + health checks (Go, zero exec)
│       ├── git.go               # Git info endpoints
│       ├── memory.go            # Memory/log access
│       ├── shared.go            # Shared utilities
│       └── static.go            # SPA static file serving
├── internal/
│   ├── cli/
│   │   ├── format.go            # Output formatting, git helpers, YAML parser (264 loc)
│   │   └── yaml_fuzz_test.go    # 38 tests — edge cases + security fuzzing
│   ├── config/
│   │   └── deploy.go            # Config display (61 loc)
│   ├── doctor/
│   │   ├── diagnostic.go        # System diagnostic — 13 checks (200+ loc)
│   │   └── diagnostic_test.go   # 11 tests — PASS
│   ├── license/
│   │   ├── bind.go              # License key binding + PBKDF2 derivation (300 loc)
│   │   └── bind_test.go         # 7 tests — PASS
│   ├── profile/
│   │   └── compiler.go          # Profile list/apply/remove (439 loc)
│   ├── tools/
│   │   ├── catalog.go           # Tool catalog — 43 tools registered (350+ loc)
│   │   └── catalog_test.go      # 19 tests — PASS
│   ├── vault/
│   │   ├── encrypt.go           # AES-256-GCM encrypt/decrypt (112 loc)
│   │   └── encrypt_test.go      # 7 tests — PASS
│   └── install/
│       ├── install.go           # Types, constants, mode resolution (318 loc)
│       ├── plan.go              # Plan builder from install packs (152 loc)
│       ├── manifest.go          # Manifest builder (73 loc)
│       ├── safety.go            # Safety evaluation (73 loc)
│       ├── boundaries.go        # Boundary validation (90 loc)
│       ├── backup.go            # Backup engine — SHA-256 (188 loc)
│       ├── apply.go             # Pipeline orchestrator (239 loc)
│       ├── rollback.go          # Rollback engine (151 loc)
│       ├── verify.go            # Post-apply verification (81 loc)
│       ├── report.go            # Evidence report writer (108 loc)
│       ├── ux.go                # Human-readable previews (90 loc)
│       ├── config.go            # Config deploy + governed deploy (250 loc)
│       └── install_test.go      # 38 tests — PASS (parity with Python)
├── go.mod                       # yaml.v3 + bubbletea deps
├── Makefile                     # Cross-compile, test, vet, fmt, release
└── ARCHITECTURE.md              # Este documento
```

---

## Binarios

| Binario | Path | Descripción | Estado |
|---|---|---|---|
| `ovav` | `cmd/ovav/main.go` | CLI pública OVAV — 6 comandos | ✅ Activo |
| `cpanel` | `cmd/cpanel/main.go` | Servidor HTTP control panel (Fly.io) | ✅ Activo |
| `cockpit` | `cmd/cockpit/main.go` | TUI Bubble Tea v1.0 — 8 vistas, 17 archivos, 5 targets | 🟢 Activo |

---

## Paquetes internos

| Paquete | Propósito | Tests | Estado |
|---|---|---|---|
| `cli` | Output, git helpers, YAML parse, timestamps | ✅ 38/38 | ✅ Completo |
| `config` | Configuración OVAV display | ❌ 0 | 🟡 Sin tests |
| `doctor` | System diagnostic — 13 checks | ✅ 11/11 | ✅ Completo |
| `license` | PBKDF2 key derivation + bind + verify | ✅ 7/7 | ✅ Completo |
| `profile` | Profile list/apply/remove compiler | ✅ 18/18 | ✅ Completo |
| `tools` | Tool catalog — discoverable named registry | ✅ 19/19 | ✅ Completo |
| `vault` | AES-256-GCM crypto | ✅ 7/7 | ✅ Completo |
| `install` | Install gateway — plan, manifest, backup, apply, rollback, verify, report | ✅ 38/38 | 🟢 Nuevo — paridad con Python |

**Total: 138 tests Go + 16 Python security = 154 tests. 7/8 packages with full coverage. go vet: zero issues.**

---

## Comandos CLI (`ovav`)

| Comando | Implementado | Descripción |
|---|---|---|
| `status` | ✅ | System posture + git info |
| `profile list` | ✅ | Lista perfiles profesionales |
| `profile apply` | ✅ | Aplica perfil a directorio |
| `profile remove` | ✅ | Remueve perfil |
| `config` | ✅ | Muestra configuración |
| `update` | ⚠️ Stub | Check updates (no implementado aún) |
| `waiver <motivo>` | ✅ | Crea waiver firmado, ligado a identidad y sesión (máximo 60 min) |
| `waiver status` | ✅ | Verifica firma, identidad, rama y vigencia |
| `waiver revoke` | ✅ | Revoca y audita el waiver activo |
| `version` | ✅ | Build info + SHA |
| `doctor` | ✅ | Diagnóstico completo — 13 checks (go, git, branch, config, registry) |
| `doctor --quick` | ✅ | Fast check — 5 core checks |
| `tools list` | ✅ | Catálogo de herramientas — 43 tools registradas |
| `tools search` | ✅ | Búsqueda por keyword, ID, categoría |
| `tools show` | ✅ | Detalle de una herramienta |
| `tools go` | ✅ | Solo herramientas Go-native |
| `tools categories` | ✅ | Categorías con conteo |

---

## Convenciones

1. **Stdlib only** — cero dependencias externas. `crypto/*`, `net/http`, `encoding/json`.
2. **Naming** — paquetes descriptivos. Una responsabilidad por paquete.
3. **Tests** — todo paquete nuevo debe tener tests. Mínimo 80% coverage.
4. **Errors** — siempre envueltos con `fmt.Errorf("pkg: context: %w", err)`.
5. **Build** — `CGO_ENABLED=0`, `-ldflags="-s -w"`, `-trimpath`.
6. **Cross-compile** — linux/darwin/windows, amd64/arm64.
7. **ARCHITECTURE.md** — actualizar con cada cambio estructural.

---

## Mapa de migración Python → Go

| Componente Python | Equivalente Go | Estado |
|---|---|---|
| `bin/ovav` (1642 loc) | `cmd/ovav/main.go` (409 loc) | ✅ Migrado |
| `tools/cpanel/*.py` (backend) | `cmd/cpanel/*.go` (14 archivos) | ✅ Migrado |
| `tools/cli/ovav_profile.py` | `internal/profile/compiler.go` | ✅ Migrado |
| `tools/security/credential_vault.py` | `internal/vault/encrypt.go` | ✅ Migrado |
| `tools/install_gateway/` (13 archivos, 2391 loc) | `internal/install/` (12 archivos, ~1800 loc) | 🟢 Migrado v1.0 — 38 tests PASS |
| `tools/cli/ovav_first_run_cockpit.py` | `cmd/cockpit/` (17 archivos, 5 targets) | 🟢 Migrado v1.0 — Python congelado |
| `tools/cli/ovav_tailor_composer.py` | `internal/tailor/` | ⬜ Pendiente — GO-TAILOR |

**Regla**: Python congelado = solo bugfixes. Nuevo desarrollo → Go.

---

## Historial de cambios

| Fecha | Cambio | Commit |
|---|---|---|
| 2026-06-16 | `internal/install/` — Go Install Gateway v1.0 (12 archivos, ~1800 loc, 38 tests PASS, paridad Python) | — |
| 2026-06-15 | `ovav doctor` implementado — 13 checks, 11 tests | — |
| 2026-06-15 | `ovav tools` implementado — 43 tools, 6 comandos, 19 tests | — |
| 2026-06-15 | YAML parser hardening — 38 fuzz/security tests | — |
| 2026-06-15 | Documento canónico creado | — |
| 2026-06-15 | Go cPanel 100% nativo (zero exec Python) | `2a1fccd` |
| 2026-06-11 | Go runtime inicial (CAPA 9) | `a10f5a9` |
