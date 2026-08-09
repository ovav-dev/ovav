# OVAV — Plan de Separación Arquitectónica: Core vs Product

**Versión:** 2.0 — Con flujo de instalación, auth, y sync diseñados
**Fecha:** 2026-06-19 03:30 UTC-5
**Autor:** thavren (Platform Engineering)
**Decisión CEO:** Split real. Core = desarrollo completo + cPanel sync. Product = terminal del usuario. Auth vía OAuth web con detección automática de tier.
**Referencia:** OpenCode workspace panel + GitHub CLI device flow

---

## 1. VISIÓN GENERAL

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│   OVAV-CORE (Mother Root)                                        │
│   Desarrollo interno · cPanel sync server · Gobernanza           │
│                                                                  │
│   ┌──────────────┐    ┌──────────────────┐                      │
│   │ cPanel :5858 │    │ Sync Module      │                      │
│   │ HTTP Server  │    │ ─────────────────│                      │
│   │ JWT + OAuth  │    │ • Version push   │──── actualizaciones ──│──┐
│   │ API REST     │    │ • Product update │                      │  │
│   └──────────────┘    └──────────────────┘                      │  │
│                                                                  │  │
│   .ovav/ · validators/ · convert/ · doctor/ · install/          │  │
│   sbom/ · economy/ · chronos/ · tailor/ · profile/              │  │
│   tools/ (Python) · docs/ · cmd/ovav-dev/                       │  │
│                                                                  │  │
└──────────────────────────────────────────────────────────────────┘  │
                                                                      │
                                                                      │
    ┌─────────────────────────── SYNC ────────────────────────────────┘
    │  cPanel → POST /api/v1/product/version
    │  Product detecta → muestra banner → ovav update
    │
    ▼
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│   OVAV-PRODUCT (lo que el usuario instala)                       │
│                                                                  │
│   ┌────────────────────────────────────────────────────────┐     │
│   │  TERMINAL DEL USUARIO                                   │     │
│   │                                                        │     │
│   │  $ ovav                         ← Abre Cockpit TUI     │     │
│   │  $ ovav connect                 ← OAuth web → login    │     │
│   │  $ ovav worktree create [name]  ← CLI directo          │     │
│   │  $ ovav update                  ← Auto-update          │     │
│   │  $ ovav --install               ← Instalación fresca   │     │
│   └────────────────────────────────────────────────────────┘     │
│                                                                  │
│   Cockpit TUI · OWS · GitFlow · License Client · Vault          │
│   Plugin System · Auto-update Client                            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. FLUJO COMPLETO DEL USUARIO

### 2.1 Instalación inicial

```
┌─ Terminal ─────────────────────────────────────────────────────┐
│                                                                 │
│  $ git clone https://github.com/ovav/product.git               │
│  $ cd product                                                   │
│  $ make install                                                 │
│                                                                 │
│  ════════════════════════════════════════════════════════════   │
│  OVAV Product v1.0.0 instalado                                  │
│                                                                 │
│  Binario: /usr/local/bin/ovav                                   │
│  Config:  ~/.config/ovav/                                       │
│                                                                 │
│  Para empezar: ovav                                             │
│  ════════════════════════════════════════════════════════════   │
│                                                                 │
│  $ ovav                                                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  🛩️ OVAV — Free Tier                                    │  │
│  │                                                          │  │
│  │  Conecta tu cuenta para desbloquear más:                 │  │
│  │  ovav connect                                            │  │
│  │                                                          │  │
│  │  [Worktrees] [Plan] [Ayuda]           Free · Sin cuenta  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Conexión de cuenta (OAuth)

```
┌─ Terminal ───────────────────┐    ┌─ Navegador ─────────────────────┐
│                              │    │                                  │
│ $ ovav connect              │    │ https://ovav.dev/connect        │
│                              │    │         ?code=XK9M-2PLQ          │
│ ┌──────────────────────────┐ │    │                                  │
│ │ Conectando con OVAV...   │ │    │ ┌────────────────────────────┐  │
│ │                          │ │    │ │ Inicia sesión en OVAV      │  │
│ │ Abre tu navegador en:    │ │    │ │                            │  │
│ │ ovav.dev/connect         │ │    │ │ Email: [________]          │  │
│ │                          │ │    │ │ Pass:  [________]          │  │
│ │ Código: XK9M-2PLQ        │ │    │ │                            │  │
│ │                          │ │    │ │ [Iniciar sesión]           │  │
│ │ ⠋ Esperando...           │ │    │ │                            │  │
│ │                          │ │    │ │ o: Google · GitHub         │  │
│ │ ✅ Conectado              │ │    │ └────────────────────────────┘  │
│ │ user@email.com · Pro     │ │    │         ↓                        │
│ │                          │ │    │ ┌────────────────────────────┐  │
│ │ [Cerrar]                 │ │    │ │ ✅ Sesión iniciada          │  │
│ └──────────────────────────┘ │    │ │                            │  │
│                              │    │ │ Cuenta: user@email.com     │  │
│ Cockpit se actualiza:        │    │ │ Membresía: Pro             │  │
│ Free → Pro ✅                │    │ │                            │  │
│                              │    │ │ ¿Conectar con OVAV en      │  │
│ Ahora con acceso a:          │    │ │ tu terminal?               │  │
│ • Worktrees ilimitados       │    │ │                            │  │
│ • Modo Dev                   │    │ │ [Conectar]  [Cancelar]     │  │
│ • Diff horizontal            │    │ └────────────────────────────┘  │
│ • AI Resolve (300/mes)       │    │         ↓                        │
│                              │    │ ┌────────────────────────────┐  │
└──────────────────────────────┘    │ │ ✅ Conectado                │  │
                                    │ │                            │  │
                                    │ │ Puedes cerrar esta página. │  │
                                    │ └────────────────────────────┘  │
                                    └──────────────────────────────────┘

FLUJO TÉCNICO:
  1. ovav connect → genera código 8 chars (XK9M-2PLQ)
  2. Abre navegador en ovav.dev/connect?code=XK9M-2PLQ
  3. Usuario inicia sesión (email/password o Google/GitHub OAuth)
  4. ovav.dev detecta tier del usuario (Free/Pro/Pro+/Business/Enterprise)
  5. Usuario confirma "Conectar"
  6. Server asocia code → user session → tier + features
  7. Terminal (polling cada 2s): GET /api/v1/connect/status?code=XK9M-2PLQ
  8. Respuesta: { status: "connected", email, tier, features, quotas }
  9. Terminal escribe state.json → Cockpit desbloquea tier
```

### 2.3 Uso diario

```
$ ovav
┌──────────────────────────────────────────────────────────────────┐
│  🛩️ OVAV Cockpit · Pro · user@email.com                         │
│  Plan: v40.0 — Python-Free                                       │
│                                                                  │
│  ┌─ Worktrees ──────────────────────────────────────────────┐   │
│  │ ⬤ feature/continue-plan  🟢 clean  👤 thavren   🕐 6.7h  │   │
│  │ ⬤ hotfix/login-error     🟡 dirty  👤 soren    🕐 1.2h  │   │
│  │ ⬤ feature/docs-update    🔴 stale  👤 —        🕐 3d    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  owc:New  owd:Done  owv:Verify  owx:Route  owlk:Lock            │
│                                                                  │
│  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░  ai_resolve: 45/300 · 15% usado      │
│                                                                  │
│  🔄 Update disponible: v1.0.1 → v1.1.0   [ovav update]          │
└──────────────────────────────────────────────────────────────────┘
```

### 2.4 Actualización automática del producto

```
┌─ cPanel (CORE) ──────────────────────────────────────────────────┐
│                                                                  │
│  POST /api/v1/product/version                                    │
│  {                                                               │
│    "version": "1.1.0",                                           │
│    "channel": "stable",                                          │
│    "released_at": "2026-07-01T00:00:00Z",                        │
│    "changelog": "F6 Cockpit: Worktrees view. License activation.",│
│    "min_core_version": "2.0.0",                                  │
│    "download": {                                                 │
│      "linux_amd64": "https://cdn.ovav.dev/releases/v1.1.0/...",  │
│      "darwin_arm64": "https://cdn.ovav.dev/releases/v1.1.0/...", │
│      "windows_amd64": "https://cdn.ovav.dev/releases/v1.1.0/..." │
│    },                                                            │
│    "checksums": { "linux_amd64": "sha256:abc123...", ... }       │
│  }                                                               │
│                                                                  │
│  WebSocket: /api/v1/product/watch                                │
│  → Notificación instantánea: "new version available"             │
│                                                                  │
└──────────────────────────┬───────────────────────────────────────┘
                           │
    ┌──────────────────────┘
    │  Product (cliente) en startup:
    │  1. GET /api/v1/product/version → compara con versión local
    │  2. Si hay nueva → banner en Cockpit: "🔄 v1.1.0 disponible"
    │  3. WebSocket /watch → notificación instantánea
    │
    ▼
┌─ Terminal del usuario ───────────────────────────────────────────┐
│                                                                  │
│  $ ovav update                                                   │
│  ════════════════════════════════════════════════════════════   │
│  Actualizando OVAV Product v1.0.1 → v1.1.0                       │
│                                                                  │
│  [████████████████████████████] 100%  12.3 MB / 12.3 MB          │
│                                                                  │
│  ✅ v1.1.0 instalado. Reinicia OVAV para aplicar.                │
│  ════════════════════════════════════════════════════════════   │
│                                                                  │
│  Changelog:                                                      │
│  • F6 Cockpit: Worktrees view with status cards                  │
│  • License activation via ovav connect                           │
│  • Windows support (experimental)                                │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. INVENTARIO DETALLADO

### 3.1 OVAV-CORE (Mother Root)

| Componente | LOC | Propósito |
|-----------|-----|-----------|
| `.ovav/` | ~200 archivos | Gobernanza canónica: leyes, contratos, plan, registry, políticas |
| `.ovav/source/` | ~50 archivos | Fuente canónica de agentes, perfiles, configs |
| `cmd/cpanel/` | ~3,000 | HTTP server :5858 · JWT · OAuth · Sync Module |
| `cmd/ovav-dev/` | ~200 | CLI para desarrolladores OVAV |
| `internal/validators/` | ~8,000 | 77 validadores F0-F5 |
| `internal/convert/` | ~1,500 | Engine de sync: genera exports para product |
| `internal/doctor/` | ~800 | Diagnóstico del sistema |
| `internal/install/` | ~1,200 | Pipeline de instalación (dev) |
| `internal/sbom/` | ~600 | SBOM generator |
| `internal/economy/` | ~500 | Budget tracking |
| `internal/chronos/` | ~400 | Git history engine |
| `internal/tailor/` | ~900 | Profile composer |
| `internal/profile/` | ~400 | Profile management |
| `internal/config/` | ~300 | Config management |
| `internal/project/` | ~500 | Project sync |
| `internal/status/` | ~400 | Status engine |
| `internal/tools/` | ~300 | Tool management |
| `tools/` | ~173,000 | Python legacy en migración |
| `docs/` | ~50 archivos | Documentación técnica |
| **Total core** | **~188K LOC** | Mayormente Python legacy + validadores |

### 3.2 OVAV-PRODUCT (Sellable)

| Componente | LOC | Propósito |
|-----------|-----|-----------|
| `cmd/cockpit/` | ~2,800 | TUI Bubble Tea · 8 vistas |
| `cmd/ovav/` | ~500 | CLI usuario: `ovav`, `ovav connect`, `ovav update` |
| `internal/ows/` | ~6,000 | Workflow System F1-F9 |
| `internal/gitflow/` | ~2,500 | Git operations engine |
| `internal/license/` | ~500 | License client: DecodeLicenseKey, Verify, state.json |
| `internal/vault/` | ~200 | AES-256-GCM client-side |
| `internal/cli/` | ~300 | CLI helpers compartidos |
| `internal/update/` | ~300 | 🆕 Auto-update client |
| `internal/connect/` | ~300 | 🆕 OAuth connect flow (device code + polling) |
| `internal/plugins/` | ~500 | 🆕 Plugin system (carga lazy) |
| **Total product** | **~13,900 LOC** | Liviano, instalable, testeable en Windows |

### 3.3 Plugins (carga lazy, dentro de product)

| Plugin | LOC | Cuándo carga |
|--------|-----|-------------|
| `cpanel/` | ~3,000 | Solo Negocio/Empresas que necesiten servidor local |
| `install/` | ~1,200 | Solo durante `ovav --install` |
| `doctor/` | ~800 | Solo durante diagnóstico |
| `tailor/` | ~900 | Solo durante composición de perfiles |

---

## 4. SEGURIDAD — LÍMITE CORE ↔ PRODUCT

```
┌──────────────────────────────────────────────────────────────────┐
│                   LÍMITE DE SEGURIDAD                             │
│                                                                  │
│  CORE (trusted)                    PRODUCT (untrusted)            │
│  ─────────────                     ──────────────────             │
│  • HMAC signing key                • Solo HMAC verify             │
│  • Vault master secrets            • Solo encrypt/decrypt local   │
│  • License generation (server)     • Solo DecodeLicenseKey()      │
│  • caps.yaml canónico              • Solo exports firmados        │
│  • Validator internals             • NUNCA acceso                 │
│  • cPanel auth server              • Solo cliente HTTP            │
│                                                                  │
│  CONVERT ENGINE:                                                  │
│  core genera ──► .ovav/export/product/ ──► product embebe         │
│  (solo datos públicos: perfiles, políticas, configs)              │
│                                                                  │
│  SYNC MODULE (cPanel):                                            │
│  core publica ──► /api/v1/product/version ──► product consulta    │
│  (solo metadata pública: versión, changelog, URL descarga)       │
│                                                                  │
│  OAUTH (cPanel):                                                  │
│  product solicita ──► /api/v1/connect/status?code=XXX             │
│  core responde ──► { email, tier, features } (NO secretos)       │
└──────────────────────────────────────────────────────────────────┘
```

---

## 5. cPanel SYNC MODULE — Especificación

### 5.1 Endpoints

| Endpoint | Método | Propósito |
|----------|--------|-----------|
| `/api/v1/product/version` | GET | Versión actual, changelog, URLs de descarga |
| `/api/v1/product/watch` | WebSocket | Notificación instantánea de nueva versión |
| `/api/v1/connect/initiate` | POST | Inicia OAuth: recibe device_code → responde { code, expires_in } |
| `/api/v1/connect/status` | GET | Polling: `?code=XK9M-2PLQ` → { status, email?, tier?, features? } |
| `/api/v1/account/sync` | POST | Sync periódico: device_token → { tier, features, quotas } |
| `/api/v1/account/quotas` | GET | Estado de cuotas: { ai_resolve: { used, max, reset } } |

### 5.2 Flujo de versión

```
cPanel detecta nueva versión en core:
  → Actualiza /api/v1/product/version
  → Emite evento por WebSocket /watch
  → Product recibe notificación instantánea
  → Cockpit muestra banner: "🔄 Update v1.1.0"
```

### 5.3 Seguridad de endpoints

| Endpoint | Auth requerida | Rate limit |
|----------|:---:|:---:|
| `/api/v1/product/version` | ❌ Público | 60/min |
| `/api/v1/product/watch` | ❌ Público | 10 conexiones/IP |
| `/api/v1/connect/initiate` | ❌ Público | 5/min |
| `/api/v1/connect/status` | ❌ Público | 30/min |
| `/api/v1/account/sync` | ✅ device_token | 30/min |
| `/api/v1/account/quotas` | ✅ device_token | 30/min |

---

## 6. ovav.dev WEB — Panel del Usuario

```
https://ovav.dev/workspace/ws_8a7b3c2d...
┌────────────────────────────────────────────────────┐
│  OVAV Workspace                                    │
│                                                    │
│  👤 user@email.com                                 │
│  🏷️ Pro · Desde 2026-06-18                         │
│                                                    │
│  ┌─ Resumen ────────────────────────────────────┐ │
│  │ Dispositivos conectados: 1/2                 │ │
│  │ ai_resolve usado: 45/300 (15%)               │ │
│  │ Próximo cobro: 2026-07-18 · $29              │ │
│  │ [Cambiar plan]  [Facturación]                │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  ┌─ Conectar dispositivo ───────────────────────┐ │
│  │ En tu terminal ejecuta: ovav connect         │ │
│  │ Código de un solo uso: XK9M-2PLQ             │ │
│  │ [Copiar código]                              │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  ┌─ Dispositivos ───────────────────────────────┐ │
│  │ 💻 ThinkPad-X1 · Linux · Última: 2026-06-19  │ │
│  │ 🖥️ (disponible 1 slot)                       │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  ┌─ Historial ──────────────────────────────────┐ │
│  │ 2026-06-19  Activo · Pro                     │ │
│  │ 2026-06-18  Registro · Trial                 │ │
│  └──────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────┘
```

---

## 7. PLAN DE IMPLEMENTACIÓN DEL SPLIT

| Fase | Qué | Esfuerzo | Resultado |
|------|-----|----------|-----------|
| **S1: Auditoría** | Mapa de dependencias. Clasificación core/product. Script de extracción. | 3h | `split_audit.yaml` |
| **S2: Bootstrap product** | Crear repo `ovav-product`. Extraer paquetes product. `go build` limpio. | 4h | `ovav` binario compila solo |
| **S3: Bootstrap core** | Crear repo `ovav-core`. cPanel + sync module. Convert engine. | 4h | `ovav-dev` compila. cPanel corre. |
| **S4: Connect flow** | `internal/connect/` en product. Endpoints en cPanel. | 3h | `ovav connect` funcional |
| **S5: Update client** | `internal/update/`. WebSocket watch. Banner en Cockpit. | 3h | `ovav update` funcional |
| **S6: Convert exports** | Expandir convert engine. `//go:embed` en product. | 3h | Product consume exports firmados |
| **S7: Plugin system** | `internal/plugins/`. Carga lazy. Migrar cPanel/install/doctor. | 3h | Plugins independientes |
| **S8: Windows build** | CI/CD Windows. Binario portable. Smoke test. | 3h | `ovav.exe` funcional |
| **Total split** | | **~26h** | Dos repos funcionales + Windows |

---

## 8. ORDEN COMPLETO (SPLIT + F6)

```
S1-S3: Bootstrap (11h)
  → ovav-product compila solo
  → ovav-core compila con cPanel + sync
  → Ambos repos existen
      ↓
S4-S5: Connect + Update (6h)
  → ovav connect funcional con OAuth web
  → ovav update funcional con cPanel sync
      ↓
F6-LICENSE (8h) — en ovav-product
  → state.json, feature gating, device fingerprint
  → Ya tiene connect flow, ahora gatea features
      ↓
F6-CORE + F6-PLAN + F6-ACTIVATE + F6-DIFF + F6-TIER (11h) — en ovav-product
  → Cockpit con vista Worktrees, plan enforcement, diff
      ↓
S6-S7: Exports + Plugins (6h)
  → Convert engine genera exports para product
  → Plugins migrados
      ↓
S8: Windows (3h)
  → Primer build Windows → smoke test → reporte
      ↓
F6-TEAM+F6-ENT (3h) — en ovav-product
  → Vistas de equipo y enterprise
```

**Total acumulado: ~48h**

---

## 9. DECISIONES PENDIENTES

1. **Repositorios:** ¿`github.com/ovav/core` y `github.com/ovav/product`? ¿O `ovav-core` y `ovav` (product como repo principal)?
2. **Core visibilidad:** ¿Privado u open source? Product ES comercial cerrado.
3. **ovav.dev:** ¿Se construye junto con el split o después? Es necesario para el connect flow.
4. **Stripe/pagos:** ¿La integración de pagos en ovav.dev ya existe o se construye?
5. **Windows installer:** ¿MSI, portable .exe, o Chocolatey/Winget para MVP?
6. **¿Arrancamos S1 (auditoría de dependencias) AHORA?**

---

*Documento generado por Thavren (Platform Engineering).*
*Sesión: feature/continue-plan · HEAD `348aea47`*
*Discusión CEO: flujo de instalación, OAuth connect, auto-update vía cPanel sync.*
