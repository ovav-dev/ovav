# F6 Cockpit — Plan de Acción Completo

**Versión:** 2.0 — Incluye auditoría de cadena de licencia + arquitectura de membresías
**Fecha:** 2026-06-19 00:30 UTC-5
**Autor:** thavren (Platform Engineering) + Eidren (Research Intelligence)
**Referencias:** `caps.yaml` v40.0, `OWS_SPEC.md`, `DECISION_BRIEF.md` (Eidren), Auditoría completa de `internal/license/`
**Estado:** 🟡 EN REVISIÓN — CEO revisando criterios de membresía y plan enforcement
**Sesión:** feature/continue-plan · HEAD `348aea47`

---

## 1. VISIÓN GENERAL

F6 Cockpit es la última fase pendiente del OVAV Workflow System (OWS). Las fases F1-F5 y F7-F9 están completas (5924 LOC Go, 18 archivos, 39 tests, 15/15 paquetes PASS). El Cockpit TUI ya existe con 8 vistas funcionales (1691 LOC, 1092 LOC tests). Lo que falta es tender el puente entre ambos: la vista de Worktrees dentro del Cockpit.

**Meta:** Que cualquier usuario — técnico o no — pueda orquestar worktrees desde una interfaz visual sin tocar la terminal. Y que el sistema obligue a trabajar con un plan.

---

## 2. PRINCIPIOS ARQUITECTÓNICOS (NO NEGOCIABLES)

### 2.1 Plan Mandatorio

> Sin plan, un worktree es solo un branch de git. OVAV es un sistema gobernado.

| Regla | Mecanismo |
|-------|-----------|
| `owc` siempre requiere plan | El sistema no permite crear worktree sin referencia a `caps.yaml` |
| Worktree huérfano bloqueado | Si se crea por fuera de OVAV (`git worktree add`), el Cockpit lo marca ⚠️ "sin plan" y `owd` rechaza integrar |
| Plan sticky | El plan activo persiste entre sesiones. Cambiarlo es una acción explícita |
| Commit↔Plan trazable | Cada worktree registra a qué cap del plan pertenece |

**Filosofía CEO:** Si el usuario dejó un plan seleccionado que no quería, no es un error del sistema — es el sistema funcionando. El plan es ley. Cambiarlo debe ser deliberado, nunca accidental.

### 2.2 Segmentación por Tier

```
TIER 0 → TIER 1 → TIER 2 → TIER 3 → TIER 4
No-dev    Dev      Pro Dev  Negocio   Empresas
(Free)    (Pro)    (Pro+)   (Business)(Enterprise)
```

Cada tier hereda las capacidades del anterior. Ver matriz completa en §6.

### 2.3 Absorción de Herramientas

OVAV Cockpit reemplaza — simplificado y enfocado en productividad inmediata:
- **Notion/Obsidian** → Plan (`caps.yaml` + vista de plan en Cockpit)
- **Linear/Jira** → Worktrees vinculados a tareas del plan
- **GitHub Desktop/GitKraken** → Orquestación visual de worktrees
- **Notas sueltas** → Documentación ligada al plan y worktree

No se busca clonar estas herramientas. Se busca absorber su función esencial y simplificarla al contexto de desarrollo gobernado.

---

## 3. AUDITORÍA — CADENA DE LICENCIA ACTUAL

### 3.1 Diagnóstico: Huérfano arquitectónico

El paquete `internal/license/bind.go` está **técnicamente impecable** — HMAC-SHA256, PBKDF2, machine binding, todo correcto. Pero **ningún componente lo importa.** Es un puente perfectamente construido que no conecta con nada.

```
Lo que EXISTE y FUNCIONA:         Lo que FALTA:
─────────────────────────────     ─────────────────────────────
🔐 License struct (Tier,           ❌ Cero imports del paquete license
   Features, Holder, Email,        ❌ License.Tier nunca se lee en producción
   IssuedAt, ExpiresAt)            ❌ License.Features nunca se lee en producción
🔐 HMAC-SHA256 sign/verify         ❌ Cero feature gating (if tier == "pro")
🔐 PBKDF2 key derivation           ❌ Sin conexión license → vault
   (600K iteraciones)              ❌ Sin conexión license → cPanel
🔐 MachineID() — Linux, Darwin,    ❌ Sin conexión license → Cockpit
   Windows                         ❌ Sin conexión license → CLI
🔐 EncodeLicenseKey()              ❌ Sin api.ovav.dev
🔐 DecodeLicenseKey()              ❌ Sin pantalla de activación en TUI
🔐 Bind() / Verify()               ❌ Sin state.json (single source of truth)
🔐 cPanel HTTP server              ❌ Sin endpoint de activación server-side
   (JWT + OAuth + rate limiting)   ❌ Sin middleware de verificación de tier
🔐 Vault AES-256-GCM               ❌ Sin registro de dispositivos (multi-machine)
   (encrypt/decrypt independiente)
```

**La criptografía está lista. El cableado no.**

### 3.2 Archivos auditados

| Archivo | Rol | Estado |
|---------|-----|--------|
| `internal/license/bind.go` (337 LOC) | Core: License, HMAC, PBKDF2, MachineID, Bind, Verify | ✅ Impecable |
| `internal/license/bind_test.go` | Tests de encode/decode/verify | ✅ Completo |
| `internal/vault/encrypt.go` | AES-256-GCM (KeySize=32 independiente) | ✅ Sin dependencia de license |
| `cmd/cpanel/main.go` | HTTP server :5858, stdlib-only | ✅ Funcional |
| `cmd/cpanel/auth.go` | JWT RS256, rate limiting (5/min/IP) | ✅ Sin integración con license |
| `cmd/cpanel/oauth.go` | Google/GitHub OAuth | ✅ Sin integración con license |
| `cmd/cpanel/routes.go` | REST API `/api/v1/*` | ⚠️ Sin auth middleware en rutas |
| `cmd/cockpit/` (21 archivos) | TUI 8 vistas | ❌ Sin mención de license |
| `cmd/ovav/main.go` | CLI entry point | ❌ Sin chequeo de license |

---

## 4. CRITERIO 1 — ARQUITECTURA DE MEMBRESÍAS (REDISEÑADA)

### 4.1 Flujo completo: Web → API Key → TUI → Sincronización

```
┌─────────────────────────────────────────────────────────────────┐
│  ovav.dev (WEB)                                                  │
│                                                                  │
│  Usuario logueado → Compra/Paga → Sección "API Keys"            │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Tu clave de licencia OVAV:                              │   │
│  │  eyJwcm8iLCJ0aGF2cmVuQGV4YW1wbGUuY29tIiwiMjAyNi0wNi...   │   │
│  │  [Copiar]                                                │   │
│  │                                                          │   │
│  │  Dispositivos activos: 1/2                               │   │
│  │  Tier: Pro · Expira: 2027-06-18                          │   │
│  └──────────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────────┘
                            │ copia la clave
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│  OVAV Cockpit / CLI                                              │
│                                                                  │
│  $ ovav connect                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  🔐 Activar OVAV                                         │   │
│  │                                                          │   │
│  │  Pega tu clave de licencia desde ovav.dev:               │   │
│  │  [eyJwcm8iLCJ0aGF2cmVuQGV4YW1wbGUuY29t...]               │   │
│  │                                                          │   │
│  │  [Activar]  [Cancelar]                                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  1. DecodeLicenseKey() → HMAC verify (local, sin internet)      │
│  2. Device fingerprint: SHA256(machine_id + hostname + OS        │
│     + MAC + kernel_version)                                      │
│  3. POST api.ovav.dev/activate                                   │
│     { license_key, device_fingerprint, hostname, os }            │
│  4. Server verifica: ¿key válida? ¿<2 dispositivos?              │
│     → Respuesta: { status: "active", tier: "pro",                │
│         features: [...], device_token: "dt-abc123" }             │
│  5. Escribe ~/.config/ovav/license/state.json                    │
│  6. ✅ Activado.                                                 │
│                                                                  │
│  Sincronización periódica (background, no bloqueante):           │
│  - Cada 24h o al iniciar Cockpit                                 │
│  - POST api.ovav.dev/sync { device_token }                       │
│  - Respuesta: { tier, features actualizados, revoked? }          │
│  - Si offline >7 días: banner ⚠️ "Sin sincronizar X días"        │
│  - Si revocada: bloquear features premium hasta reconectar       │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Separación de conceptos (antes vs. ahora)

| Concepto | Diseño original (bind.go) | Diseño propuesto |
|----------|--------------------------|-----------------|
| **Autenticación** (¿clave válida?) | HMAC verify local | HMAC verify local + server-side activation |
| **Autorización** (¿qué tier/features?) | En payload de licencia | En `state.json`, refrescado periódicamente |
| **Binding** (¿en qué máquina?) | PBKDF2(key, machine_id) → vault key | Server-side device registry (máx 2) |
| **Datos locales** (vault) | Derivado de licencia | Derivado de `device_token` (post-activación) |
| **Multi-máquina** | ❌ Bloqueado por diseño (PBKDF2) | ✅ Server-side device tracking |

### 4.3 Multi-máquina: Máximo 2 dispositivos, resistente a VPN

**Device fingerprint** (no solo IP — resistente a VPN):

```
fingerprint = SHA256(
    machine_id       +  // /etc/machine-id (Linux) o equiv
    hostname         +  // nombre del equipo
    os               +  // "linux", "darwin", "windows"
    kernel_version   +  // uname -r
    primary_mac      +  // MAC de la primera interfaz física
    cpu_model        +  // modelo de CPU (estable)
)
```

**Lógica server-side:**

```
Al activar:
  IF license_key ya tiene 2 fingerprints registrados:
    IF el fingerprint nuevo matchea uno existente:
      → PERMITIR (es la misma máquina, reinstalación/formateo)
    ELSE:
      → BLOQUEAR. "Ya tienes 2 dispositivos activos. Desvincula uno en ovav.dev."
  ELSE:
    → REGISTRAR fingerprint. Responder con device_token.

VPN resistance:
  - El fingerprint NO incluye IP → mismo dispositivo tras VPN = mismo fingerprint
  - Server puede registrar IPs vistas para analytics, pero no para autorización
  - Si mismo fingerprint aparece desde IPs radicalmente distintas en <5 min:
    → Flag de seguridad (posible robo de device_id), pero no bloqueo automático
  - La seguridad real está en el fingerprint de hardware, no en la IP
```

**Flujo para desvincular dispositivo:**
```
Usuario → ovav.dev → "Mis dispositivos" → [Desvincular] dispositivo X
       → Server marca fingerprint como revoked
       → Próxima sync del dispositivo revocado → state.json marcado invalid
       → Cockpit muestra: "Dispositivo desvinculado. Reactivar."
```

### 4.4 ¿Debe tener login?

**No login tradicional.** La clave de licencia ES la autenticación. Es una prueba criptográfica de compra.

| Método | ¿Se usa? | Razón |
|--------|----------|-------|
| Usuario/contraseña en TUI | ❌ | Fricción innecesaria. La clave HMAC es superior. |
| OAuth en TUI | ❌ | Complejidad innecesaria para activación. Solo web. |
| API Key (clave de licencia) | ✅ | Formato base64+HMAC. Pegar y listo. |
| Device token (post-activación) | ✅ | JWT interno. No visible al usuario. |

### 4.5 La API conecta con OVAV sistema, no con Cockpit

> La API no debe conectar con Cockpit — debe conectar con el sistema OVAV completo. Cockpit es un consumidor más.

```
               ┌──────────────────────────────┐
               │   ~/.config/ovav/license/    │
               │   state.json                 │
               │   {                          │
               │     tier: "pro",             │
               │     features: ["cockpit",    │
               │       "ows_unlimited",       │
               │       "ai_resolve"],         │
               │     device_token: "dt-abc",  │
               │     expires: "2027-06-18",   │
               │     activated_at: "..."      │
               │   }                          │
               └──────┬───────────────────────┘
                      │  (lee en startup)
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   ┌─────────┐  ┌──────────┐  ┌──────────┐
   │ Cockpit │  │ ovav CLI │  │ cPanel   │
   │ gatea   │  │ gatea    │  │ gatea    │
   │ vistas  │  │ comandos │  │ APIs     │
   └─────────┘  └──────────┘  └──────────┘
```

El gating es a nivel de paquete Go, consumido por todas las tools:

```go
// internal/license/state.go (NUEVO)
func LoadState() (*State, error)  // lee state.json
func Tier() string                // "pro", "enterprise", "trial"
func IsEnabled(feature string) bool  // busca en state.Features

// internal/license/gate.go (NUEVO)
func RequireFeature(feature string) error  // retorna error si no habilitado
```

### 4.6 ¿Qué pasa si alguien intenta vulnerarla?

| Ataque | Defensa | Efectividad |
|--------|---------|-------------|
| **Forjar licencia** | HMAC-SHA256. Sin `OVAV_LICENSE_HMAC_KEY` (server-side), imposible. | 🔒 Total |
| **Compartir licencia** | Server-side device registry. Máx 2 fingerprints. El 3ero es rechazado. | 🔒 Total |
| **Suplantar fingerprint** | SHA256 de 6 campos de hardware. Misma máquina reinstalada = mismo fingerprint. Otra máquina = fingerprint distinto. | 🛡️ Alta |
| **VPN para ocultar IP** | Fingerprint no incluye IP. VPN no afecta la autorización. | ✅ Inmune |
| **Modificar binario** | `-ldflags="-s -w"` + garble. El vault depende de device_token, no del binario. | 🛡️ Alta |
| **Man-in-the-middle** | HTTPS + certificate pinning para `api.ovav.dev`. | 🛡️ Alta |
| **Offline bypass** | state.json tiene expiry. Vault requiere device_token derivado. Sin activación inicial → sin acceso a features premium. | 🛡️ Alta |
| **Revocar acceso** | Server marca licencia como revoked. Próxima sync bloquea features. | 🛡️ Alta |

---

## 5. CRITERIO 2 — PLAN ENFORCEMENT (EL PLAN ES LEY)

### 5.1 Filosofía

> El usuario selecciona un plan. El sistema lo sigue. Si el usuario se equivocó de plan, no es un bug — es el sistema funcionando. Cambiar de plan es una acción deliberada, no un accidente.

Esto diferencia a OVAV de cualquier otra herramienta. Nadie en la industria fuerza seguir el plan a nivel git. Graphite, Linear, Sapling — todos usan visibilidad + convención. OVAV va un paso más allá: **gobernanza real.**

### 5.2 Capas de Enforcement

```
CAPA 1 ─── VISIBILIDAD (siempre activa)
  El Cockpit muestra el plan activo en TODO momento.
  owc pregunta: "Próximo planificado: SEG8 · License Activation. ¿Crear worktree? [Y/n]"
  Worktree muestra a qué cap pertenece.

CAPA 2 ─── CONVENCIÓN (activada por defecto)
  Branch naming: {plan_id}-{type}-{short_desc}
  owc actualiza caps.yaml automáticamente (worktree: "feature-SEG8-...")
  Pre-push hook verifica que el branch corresponde a un cap pendiente.

CAPA 3 ─── STRICT MODE (activación manual: cockpit --strict o OVAV_STRICT_MODE=1)
  Bloquea owc si el branch no matchea un plan_id.
  Bloquea commit sin referencia al plan.
  Bloquea owd sin conflict prediction PASS.
  Worktree sin plan → bloqueado hasta vincular vía Cockpit.
```

### 5.3 Binding Worktree ↔ Plan

```
caps.yaml:
  pending:
    SEG8:
      id: SEG8
      name: License Activation Flow
      deps: [SEG7]
      order: 8
      worktree: ""              # ← owc lo llena
      commit: ""                # ← owd lo llena

Al ejecutar: owc SEG8-feat-license-activation
  → caps.yaml se actualiza: SEG8.worktree = "feature-SEG8-license-activation"
  → Cockpit muestra: "Worktree vinculado a SEG8 ✅"
  → Al hacer owd: SEG8.commit = "<merge_commit_hash>"
```

### 5.4 ¿Qué pasa si el usuario inicia un trabajo sin plan?

```
$ owc fix-urgente
⚠️  No hay plan activo.
    Cockpit: [Seleccionar plan] [Crear sin plan] [Cancelar]

Si elige "Crear sin plan":
  - Worktree se crea pero marcado ⚠️ "sin plan"
  - Cockpit lo muestra en gris, no integrable
  - Para integrarlo: debe vincularse a un cap del plan
  - Esto es deliberado: fuerza la disciplina del plan
```

### 5.5 ¿Qué pasa si el plan seleccionado no es el que quería?

> **No es un error. Es el sistema funcionando.**

```
Escenario: Usuario tenía Plan v40.0 activo. Quería trabajar en una idea nueva.

Lo que ocurre:
  1. Cockpit muestra "Plan activo: v40.0 — Python-Free" en cada vista
  2. owc pregunta: "Próximo planificado: F6 Cockpit. ¿Crear worktree? [Y/n]"
  3. El usuario ve que no es lo que quería
  4. Acción deliberada: Cockpit → Planes → [Cambiar plan] o [Crear nuevo plan]

El sistema no adivina la intención del usuario. Respeta su propia estructura.
Si el plan está mal, se cambia — pero como acción explícita, nunca accidental.
```

---

## 6. MATRIZ COMPLETA DE CAPACIDADES POR TIER

| Capacidad | No-dev | Dev | Pro Dev | Negocio | Empresas |
|-----------|:---:|:---:|:---:|:---:|:---:|
| **Vista worktrees** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Modo estándar** (no técnico) | ✅ default | ✅ | ✅ | ✅ | ✅ |
| **Modo Dev** (`Ctrl+T`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Plan obligatorio** | ✅ guiado | ✅ | ✅ | ✅ | ✅ |
| **Crear worktree** (`owc`) | 3 max | 15 max | ∞ | ∞ | ∞ |
| **Integrar** (`owd`) | simple | + diff | + diff | + aprobación | + aprobación |
| **Verificar** (`owv`) | resumen | completo | completo | completo | completo |
| **Listar** (`owl`) | básico | + conflictos | + conflictos | + equipo | + cross-team |
| **Sincronizar** (`ows`) | auto | manual | manual | programado | programado |
| **Abortar** (`owa`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Rescatar** (`owr`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Ruta avanzada** (`owx`) | ❌ | cherry-pick | 4 modos | 4 modos | 4 modos |
| **Bloquear** (`owlk`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Perfiles** | feature, docs | +6 más | + custom | + enterprise | + policies |
| **Diff horizontal** | ❌ | ✅ | ✅ | ✅ | ✅ |
| **Predicción conflictos** | ❌ | íconos | detalle | detalle | detalle |
| **Resolución AI** | ❌ | ❌ | auto | auto | supervisada |
| **Modo autónomo** | ❌ | ❌ | ✅ | ✅ | ✅ |
| **Métrica FMA** | ❌ | personal | personal | equipo | org |
| **Audit trail** | ❌ | ❌ | 10 últimos | exportable | + webhook |
| **Aprobación merge** | ❌ | ❌ | ❌ | ✅ | + jerarquía |
| **Políticas custom** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Vista multi-repo** | ❌ | ❌ | ❌ | ❌ | ✅ |
| **Notificaciones** | ❌ | ❌ | ❌ | email | + webhook |
| **AgentLock visible** | ❌ | ❌ | propio | equipo | cross-team |
| **Exportación** | ❌ | ❌ | ❌ | CSV | + JSON + API |
| **Licencia** | Trial 14d | Pro | Pro+ | Business | Enterprise |

---

## 7. PLAN DE IMPLEMENTACIÓN (7 HITOS — REVISADO)

### ⚠️ PREREQUISITO: F6-LICENSE — Cadena de licencia completa

> **Sin esto, el feature gating (F6-TIER) no puede existir. Cockpit no puede saber qué tier tiene el usuario. Las tools no pueden gatear features.**

```
Nuevos archivos:
  go-runtime/internal/license/state.go       (~150 LOC) LoadState/SaveState + State struct
  go-runtime/internal/license/gate.go        (~100 LOC) IsEnabled/RequireFeature/Tier
  go-runtime/internal/license/fingerprint.go (~120 LOC) DeviceFingerprint()
  go-runtime/internal/license/activate.go    (~180 LOC) HTTP client → api.ovav.dev
  go-runtime/cmd/cpanel/routes_license.go    (~200 LOC) Endpoints server-side

Modificaciones:
  go-runtime/internal/license/bind.go        DecodeLicenseKey ya existe — OK
  go-runtime/internal/vault/encrypt.go        Conectar con state.DeviceToken (hoy independiente)
  go-runtime/cmd/cpanel/routes.go             Agregar middleware de tier
  go-runtime/cmd/cpanel/auth.go               Verificar license en JWT claims

Entregable: state.json como single source of truth.
            Todas las tools leen Tier() e IsEnabled() del mismo lugar.
            api.ovav.dev endpoints de activación y sync.
```

### Hito 1: F6-CORE (~4h) — Vista Worktrees funcional

```
Nuevos archivos:
  go-runtime/cmd/cockpit/worktrees.go    (~350 LOC)

Modificaciones:
  go-runtime/cmd/cockpit/util.go         +ViewWorktrees constant
  go-runtime/cmd/cockpit/view.go         +renderCurrentView case
  go-runtime/cmd/cockpit/update.go       +handleViewKey case
  go-runtime/cmd/cockpit/root.go         +menu item "Worktrees [W]"
  go-runtime/cmd/cockpit/model.go        +worktreeList field

Entregable: Lista de worktrees con estado, dueño, perfil, edad. Navegable con teclas.
```

### Hito 2: F6-PLAN (~2h) — Plan Enforcement (Capas 1+2)

```
Modificaciones:
  go-runtime/internal/ows/handlers.go    owc plan-aware + caps.yaml write
  go-runtime/internal/ows/plan.go        (nuevo) PlanResolver: leer caps.yaml
  go-runtime/cmd/cockpit/worktrees.go    badge de plan en cards

Entregable: owc lee y escribe caps.yaml. Worktrees vinculados a plan.
```

### Hito 3: F6-ACTIVATE (~2h) — Pantalla de activación en Cockpit

```
Nuevos archivos:
  go-runtime/cmd/cockpit/activate.go     (~300 LOC)

Modificaciones:
  go-runtime/cmd/cockpit/model.go        +licenseState field
  go-runtime/cmd/cockpit/welcome.go      detección de licencia al iniciar
  go-runtime/cmd/ovav/main.go            comando ovav connect (CLI)

Entregable: Pantalla de activación. Usuario pega clave → HMAC verify → POST activate →
            state.json escrito → Cockpit desbloqueado según tier.
```

### Hito 4: F6-DIFF (~2h) — Diff horizontal + modo dual

```
Modificaciones:
  go-runtime/cmd/cockpit/worktrees.go    +diffPanel (side-by-side)
  go-runtime/cmd/cockpit/worktrees.go    +Ctrl+T toggle Standard/Dev
  go-runtime/styles/theme.go             +diff colors

Entregable: Diff horizontal al seleccionar worktree. Modo dual funcional.
```

### Hito 5: F6-TIER (~1.5h) — Feature gating por licencia

```
Modificaciones:
  go-runtime/cmd/cockpit/worktrees.go    gateo vía license.IsEnabled()
  go-runtime/cmd/cockpit/model.go        feature flags según tier
  go-runtime/cmd/ovav/main.go            gateo de comandos en CLI

Entregable: Capacidades se habilitan/deshabilitan según tier de licencia.
            Depende de F6-LICENSE (state.json debe existir).
```

### Hito 6: F6-TEAM+F6-ENT (~3h) — Vistas de equipo + enterprise

```
Nuevos archivos:
  go-runtime/cmd/cockpit/team.go         (~250 LOC) vista de equipo
  go-runtime/cmd/cockpit/policy.go       (~200 LOC) editor de políticas
  go-runtime/cmd/cpanel/routes.go        middleware de tier

Entregable: Cockpit full-featured para Negocio y Empresas.
```

### Estimación total revisada

| Fase | Esfuerzo |
|------|----------|
| F6-LICENSE (prerequisito) | ~6.5h |
| F6-CORE | ~4h |
| F6-PLAN | ~2h |
| F6-ACTIVATE | ~2h |
| F6-DIFF | ~2h |
| F6-TIER | ~1.5h |
| F6-TEAM+F6-ENT | ~3h |
| **Total** | **~21h** |

---

## 8. DEPENDENCIAS Y ORDEN (REVISADO)

```
F6-LICENSE ────┬──► F6-CORE ──┬──► F6-PLAN ──┬──► F6-TIER ──► F6-TEAM+F6-ENT
  (6.5h)       │    (4h)      │    (2h)      │    (1.5h)       (3h)
               │              │              │
               └──► F6-ACTIVATE              │
                     (2h)                    │
                                            │
               F6-DIFF ─────────────────────┘
                (2h, independiente)
```

**MVP (No-dev usable):** F6-LICENSE + F6-CORE + F6-PLAN + F6-ACTIVATE = ~14.5h
**Dev completo:** + F6-DIFF + F6-TIER = ~18h
**Enterprise:** + F6-TEAM+F6-ENT = ~21h

---

## 9. MÉTRICAS DE ÉXITO

| Métrica | Objetivo | Cómo se mide |
|---------|----------|--------------|
| FMA (First-attempt Merge Acceptance) | >85% | SQLite audit: owd sin conflictos / total owd |
| Plan adherence | >90% | Worktrees con plan_id / total worktrees |
| Time-to-activation | <2 min | Tiempo desde "Abrir Cockpit" hasta "Licencia activada" |
| Cockpit adoption | >60% de usuarios activos usan Cockpit | Telemetría (opt-in) |
| Worktrees gestionados vía Cockpit | >80% | owc ejecutados desde Cockpit vs CLI directo |
| Multi-machine activations | <0.1% de intentos rechazados por sharing | Server-side analytics |
| Offline grace period | 0% de bloqueos por offline <7 días | Cliente reporta estado offline al reconectar |

---

## 10. PREGUNTAS ABIERTAS PARA EL CEO

1. **Strict mode:** ¿Activado por defecto para empresas o siempre opt-in?
2. **ovav.dev:** ¿El portal web y `api.ovav.dev` existen ya o se construyen junto con F6-LICENSE?
3. **Precios:** ¿Los tiers y precios están definidos? Se necesitan para completar la matriz de features.
4. **Marca:** ¿"Cockpit" y "ovav connect" son los nombres finales?
5. **Prioridad:** ¿Empezamos ya con F6-LICENSE (prerequisito, ~6.5h) o esperamos definiciones de negocio?
6. **Endpoint de activación:** ¿api.ovav.dev/activate se construye en Go (cPanel) o hay otro stack para el backend?
7. **Desvinculación de dispositivos:** ¿El usuario puede autogestionarlo desde ovav.dev o requiere soporte?

---

*Documento generado por Thavren (Platform Engineering) con investigación de Eidren (Research Intelligence).*
*Referencias canónicas: `caps.yaml` v40.0, `OWS_SPEC.md`, `DECISION_BRIEF.md` (Eidren 2026-06-18), Auditoría completa `internal/license/`.*
*Sesión: feature/continue-plan · HEAD `348aea47`*
