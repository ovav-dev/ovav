# OVAV — Fase 2: Arquitectura de Producto

**Versión:** 1.0 · Consolidado final de sesión
**Fecha:** 2026-06-19 04:30 UTC-5
**Sesión:** feature/continue-plan · HEAD `348aea47`
**Autor:** thavren (Platform Engineering Lead) + Eidren (Research Intelligence)
**Decisión CEO:** Split real Core/Product · OAuth connect · Membresías por tier · Plan mandatorio · Cuotas de prueba por feature
**Archivos relacionados:**
- `split_audit.yaml` — auditoría de dependencias y clasificación
- `f6_cockpit_action_plan.md` — plan detallado de F6 Cockpit v2.0
- `.ovav/research/2026-06-18-license-plan-enforcement/` — investigación Eidren v1
- `.ovav/research/2026-06-19-quota-gating-activation-flows/` — investigación Eidren v2
- `caps.yaml` v40.0 — plan canónico activo

**Estado:** 🟡 PLAN APROBADO — Esperando completar Python→Go + estabilización para iniciar Fase 2

---

## ÍNDICE

1. [Visión General](#1-visión-general)
2. [Decisiones del CEO](#2-decisiones-del-ceo)
3. [Arquitectura Core vs Product](#3-arquitectura-core-vs-product)
4. [Flujos de Usuario](#4-flujos-de-usuario)
5. [Sistema de Membresías y Licencias](#5-sistema-de-membresías-y-licencias)
6. [Sistema de Cuotas por Tier](#6-sistema-de-cuotas-por-tier)
7. [Plan Enforcement (El Plan es Ley)](#7-plan-enforcement)
8. [Matriz de Capacidades por Tier](#8-matriz-de-capacidades-por-tier)
9. [F6 Cockpit — Especificación](#9-f6-cockpit--especificación)
10. [Auditoría de Split — Resultados](#10-auditoría-de-split--resultados)
11. [Plan de Implementación Completo](#11-plan-de-implementación-completo)
12. [Métricas de Éxito](#12-métricas-de-éxito)
13. [Dependencias y Prerrequisitos](#13-dependencias-y-prerrequisitos)

---

## 1. VISIÓN GENERAL

OVAV Fase 2 transforma el sistema de una herramienta de desarrollo interno a un producto comercial instalable. La Fase 1 (actual) completa la migración Python→Go y estabiliza el sistema. La Fase 2 ejecuta:

1. **Split arquitectónico** — Separar Mother Root (desarrollo) de Product (usuario final)
2. **F6 Cockpit** — Vista visual de Worktrees, connect OAuth, diff, plan enforcement
3. **Sistema de membresías** — Tiers Free/Pro/Pro+/Business/Enterprise con feature gating
4. **Plan mandatorio** — El sistema obliga a trabajar con un plan. Sin plan no hay integración.
5. **Prueba en Windows** — Validar el producto en entorno real

### Cronograma macro

```
FASE 1 (en curso):     Python→Go · Estabilización · cobertura
         ↓
FASE 2 (este plan):    Split Core/Product (~26h) → F6 Cockpit (~19h) → Windows (~3h)
         ↓
FASE 3 (futuro):       ovav.dev web · Stripe pagos · Marketing · Launch
```

---

## 2. DECISIONES DEL CEO

### Criterios establecidos durante la sesión

| # | Decisión | Detalle |
|---|----------|---------|
| **D1** | Split real, no build tags | Dos repos separados. Nada de atajos. |
| **D2** | OAuth vía web, no API key manual | `ovav connect` → navegador → login → tier auto-detectado |
| **D3** | Plan es mandatorio y sticky | Sin plan no hay `owd`. Cambio de plan es acción deliberada. |
| **D4** | Tiers: Free/Pro/Pro+/Business/Enterprise | Cada uno hereda del anterior. |
| **D5** | Cuotas de prueba, no créditos | Free tier prueba features Pro/Pro+ N veces/mes. Sin abstracción de créditos. |
| **D6** | cPanel en Core con Sync Module | cPanel推送 versiones a Product. Product auto-update. |
| **D7** | Multi-máquina (máx 2) | Device fingerprint de hardware. Resistente a VPN. |
| **D8** | No login en TUI | La autenticación ocurre en ovav.dev (web). TUI solo recibe el resultado. |
| **D9** | Separar lo pesado | El usuario no recibe 77 validadores. Solo lo que necesita. |
| **D10** | Probar en Windows ASAP | Build mínimo → testear → iterar agregando features. |

---

## 3. ARQUITECTURA CORE VS PRODUCT

```
┌──────────────────────────────────────────────────────────────────┐
│  OVAV-CORE (Mother Root) — github.com/ovav/core                  │
│  Desarrollo interno · cPanel sync server · Gobernanza canónica   │
│                                                                  │
│  .ovav/                 Gobernanza: leyes, contratos, plan       │
│  cmd/cpanel/            HTTP :5858 · JWT · OAuth · Sync Module   │
│  cmd/ovav-dev/          CLI para desarrolladores OVAV            │
│  internal/validators/   77 validadores F0-F5                     │
│  internal/license/      Server-side: EncodeLicenseKey, HMAC sign │
│  tools/                 Python legacy en migración                │
│  docs/                  Documentación canónica                   │
│                                                                  │
│  ~188K LOC (mayormente Python legacy + validadores)              │
└──────────────────────────┬───────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    ┌──────────┐   ┌──────────────┐   ┌──────────────┐
    │ Convert  │   │ Sync Module  │   │ OAuth Server │
    │ Engine   │   │ (cPanel)     │   │ (cPanel)     │
    │ exports  │   │ version push │   │ connect flow │
    └────┬─────┘   └──────┬───────┘   └──────┬───────┘
         │                │                  │
         ▼                ▼                  ▼
┌──────────────────────────────────────────────────────────────────┐
│  OVAV-SHARED — github.com/ovav/shared                            │
│  Interfaces extraídas de 6 paquetes puente (~600 LOC)            │
│                                                                  │
│  format/ · install_contract/ · sbom_types/                       │
│  vault_crypto/ · git_operations/ · agent_formats/                │
└──────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  OVAV-PRODUCT — github.com/ovav/product                          │
│  Lo que el usuario instala · ~13,900 LOC                         │
│                                                                  │
│  cmd/cockpit/           TUI Bubble Tea · 8 vistas                │
│  cmd/ovav/              CLI usuario: ovav, connect, update       │
│  internal/ows/          Workflow System F1-F9                    │
│  internal/gitflow/      Git operations engine                    │
│  internal/license/      Client-side: DecodeLicenseKey, Verify    │
│  internal/vault/        AES-256-GCM client-side                  │
│  internal/plugins/      🆕 Sistema de plugins (carga lazy)       │
│  internal/connect/      🆕 OAuth connect flow                    │
│  internal/update/       🆕 Auto-update client                    │
│  [+ 10 paquetes más]    chronos, doctor, economy, profile, etc.  │
└──────────────────────────────────────────────────────────────────┘
```

### Límite de seguridad

| Datos en CORE (nunca en product) | Datos en PRODUCT |
|----------------------------------|------------------|
| HMAC signing key | HMAC verify (clave pública) |
| License generation (server) | License decode + verify (client) |
| caps.yaml canónico | Plan export (solo lectura) |
| Validator internals | NUNCA acceso |
| cPanel auth server | Solo HTTP client |

---

## 4. FLUJOS DE USUARIO

### 4.1 Instalación

```
$ git clone https://github.com/ovav/product.git
$ cd product && make install
✅ OVAV Product v1.0.0 instalado
$ ovav
→ Abre Cockpit TUI (Free tier)
```

### 4.2 Conexión de cuenta (OAuth)

```
Terminal                          Navegador
────────                          ─────────
$ ovav connect
  Código: XK9M-2PLQ      →       ovav.dev/connect?code=XK9M-2PLQ
  ⠋ Esperando...                  → Iniciar sesión
                                   → "¿Conectar?" [Confirmar]
  ✅ Conectado                     ← Tier detectado: Pro
  user@email.com · Pro
```

**Flujo técnico:**
1. `ovav connect` → genera device code 8 chars
2. GET `/api/v1/connect/status?code=XK9M-2PLQ` (polling 2s)
3. Usuario inicia sesión en ovav.dev
4. Server asocia code → user session → tier + features
5. Terminal recibe `{ email, tier, features, quotas }`
6. Escribe `~/.config/ovav/license/state.json`
7. Cockpit desbloquea tier

### 4.3 Uso diario

```
$ ovav
→ Cockpit muestra: tier, plan activo, worktrees
→ Auto-check update contra cPanel /api/v1/product/version
→ Si hay nueva versión: banner "🔄 Update v1.1.0"
$ ovav update
→ Download + checksum verify + reemplazo
```

### 4.4 Actualización de producto

```
cPanel (CORE)                     Product (cliente)
────────────                      ────────
Nueva versión publicada           En startup: GET /api/v1/product/version
  ↓                               WebSocket /watch → notificación instantánea
/api/v1/product/version           Cockpit: banner "🔄 v1.1.0"
/api/v1/product/watch (WS)        $ ovav update → download → verify → replace
```

---

## 5. SISTEMA DE MEMBRESÍAS Y LICENCIAS

### 5.1 Flujo de activación

```
ovav.dev (WEB)                              OVAV Product (TUI)
─────────────                               ──────────────────
1. Usuario se registra/loguea
2. Selecciona tier (Free/Pro/Pro+/...)
3. Paga (Stripe)
4. Panel muestra:
   "Tu membresía: Pro"
   "Dispositivos: 1/2"
   "API Key: ovav_live_..."
                          ↓
                    $ ovav connect
                    → Navegador: ovav.dev/connect
                    → Login → "¿Conectar?" → ✅
                    → state.json escrito
                    → Cockpit desbloquea tier
```

### 5.2 Multi-máquina (máx 2 dispositivos)

**Device fingerprint** (resistente a VPN):
```
SHA256(machine_id + hostname + os + kernel_version + primary_mac + cpu_model)
```

**Lógica server-side:**
- Misma licencia → máx 2 fingerprints registrados
- Mismo fingerprint desde distintas IPs → permitido (es la misma máquina tras VPN)
- 3er fingerprint distinto → bloqueado. "Desvincula un dispositivo en ovav.dev"
- Desvinculación: usuario → ovav.dev → Mis dispositivos → [Desvincular]

### 5.3 state.json — Single source of truth

```json
{
  "tier": "pro",
  "email": "user@email.com",
  "features": ["cockpit", "ows_unlimited", "ai_resolve", "diff_horizontal"],
  "quotas": {
    "ai_resolve": { "used": 12, "max": 300, "reset": "2026-07-18T00:00:00Z" }
  },
  "device_token": "dt-abc123",
  "activated_at": "2026-06-18T22:30:00Z",
  "expires_at": "2026-07-18T00:00:00Z"
}
```

Todas las tools (Cockpit, CLI, cPanel) leen de este archivo. Feature gating:

```go
if license.IsEnabled("ai_resolve") { ... }
if license.CanUse("ai_resolve") { ... }  // chequea quota también
```

### 5.4 Seguridad anti-vulneración

| Ataque | Defensa |
|--------|---------|
| Forjar licencia | HMAC-SHA256. Sin clave server-side, imposible. |
| Compartir licencia | Server-side device registry. Máx 2 fingerprints. |
| Suplantar fingerprint | SHA256 de 6 campos de hardware. Otra máquina = otro hash. |
| VPN para ocultar IP | Fingerprint no incluye IP. VPN no afecta autorización. |
| Modificar binario | state.json requiere device_token derivado de activación. |

---

## 6. SISTEMA DE CUOTAS POR TIER

### 6.1 Filosofía

OVAV no vende créditos de AI. Las cuotas son **marketing + sampling**:
- Free tier prueba features premium N veces/mes
- El usuario experimenta el valor real → quiere comprar el tier superior
- Sin abstracción de "créditos". Sin compra de créditos extra.

### 6.2 Cuotas por tier

| Feature | Free | Pro | Pro+ | Business | Enterprise |
|---------|:---:|:---:|:---:|:---:|:---:|
| ai_resolve | 30/mes | 300/mes | ∞ | 3000/mes | ∞ |
| owx emergency | 5/mes | 20/mes | ∞ | 200/mes | ∞ |
| cpanel local | ❌ | ❌ | ❌ | ✅ | ✅ |

### 6.3 Comportamiento al agotar

```
ai_resolve: 30/30 usado este mes
┌──────────────────────────────────────────┐
│ ⚡ Límite de prueba alcanzado            │
│                                          │
│ Probaste ai_resolve 30 veces este mes.   │
│ En Pro tenés 300/mes + priority.         │
│                                          │
│ [Upgrade a Pro]  [Seguir en Free]        │
└──────────────────────────────────────────┘
```

- **Nunca hard stop.** El usuario no choca con un muro.
- La feature se deshabilita hasta el próximo ciclo mensual.
- El resto del sistema sigue funcionando normalmente.

### 6.4 Anti-tampering

- Cuota trackeada server-side (cPanel)
- Cliente recibe HMAC-signed quota blob
- Cliente verifica HMAC antes de enforce
- Clock binding: blob expira a los 90 min (fuerza re-sync)
- Offline buffer: 5 operaciones sin contacto con server

---

## 7. PLAN ENFORCEMENT (EL PLAN ES LEY)

### 7.1 Filosofía

> Si el usuario dejó un plan seleccionado que no quería, no es un error del sistema — es el sistema funcionando. El plan es ley. Cambiarlo debe ser deliberado, nunca accidental.

### 7.2 Capas de enforcement

| Capa | Comportamiento | Activación |
|------|---------------|------------|
| **Visibilidad** | Cockpit muestra plan activo SIEMPRE. `owc` pregunta antes de crear. | Siempre |
| **Convención** | Branch naming: `{plan_id}-{type}-{desc}`. `owc` escribe a `caps.yaml`. | Por defecto |
| **Strict** | Bloquea `owc` sin plan_id. Bloquea commit sin referencia. | Opt-in (`--strict`) |

### 7.3 Binding Worktree ↔ Plan

```
caps.yaml:
  pending:
    SEG8:
      worktree: ""              # ← owc lo llena
      commit: ""                # ← owd lo llena

owc SEG8-feat-license-activation
  → caps.yaml: SEG8.worktree = "feature-SEG8-license-activation"
  → owd: SEG8.commit = "<merge_commit_hash>"
```

### 7.4 Escenarios

**Sin plan activo:**
```
$ owc fix-urgente
⚠️  No hay plan activo.
    [Seleccionar plan] [Crear sin plan ⚠️ no integrable]
```

**Plan equivocado:**
```
Cockpit muestra: "Plan activo: v40.0" en cada vista.
owc pregunta: "Próximo planificado: F6 Cockpit. ¿Crear?"
El usuario ve que no es lo que quería → cambia el plan.
```

---

## 8. MATRIZ DE CAPACIDADES POR TIER

| Capacidad | Free | Pro | Pro+ | Business | Enterprise |
|-----------|:---:|:---:|:---:|:---:|:---:|
| Vista worktrees | ✅ | ✅ | ✅ | ✅ | ✅ |
| Modo estándar | ✅ | ✅ | ✅ | ✅ | ✅ |
| Modo Dev (`Ctrl+T`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| Plan obligatorio | ✅ | ✅ | ✅ | ✅ | ✅ |
| Worktrees (`owc`) | 3 max | 15 max | ∞ | ∞ | ∞ |
| Integrar (`owd`) | simple | + diff | + diff | + aprobación | + aprobación |
| Verificar (`owv`) | resumen | completo | completo | completo | completo |
| Listar (`owl`) | básico | + conflictos | + conflictos | + equipo | + cross-team |
| Sincronizar (`ows`) | auto | manual | manual | programado | programado |
| Abortar (`owa`) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Rescatar (`owr`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| Ruta (`owx`) | ❌ | cherry-pick | 4 modos | 4 modos | 4 modos |
| Bloquear (`owlk`) | ❌ | ✅ | ✅ | ✅ | ✅ |
| Perfiles | feature, docs | +6 más | + custom | + enterprise | + policies |
| Diff horizontal | ❌ | ✅ | ✅ | ✅ | ✅ |
| Predicción conflictos | ❌ | íconos | detalle | detalle | detalle |
| Resolución AI | ❌ | ❌* | auto | auto | supervisada |
| Modo autónomo | ❌ | ❌ | ✅ | ✅ | ✅ |
| Métrica FMA | ❌ | personal | personal | equipo | org |
| Audit trail | ❌ | ❌ | 10 últimos | exportable | + webhook |
| Aprobación merge | ❌ | ❌ | ❌ | ✅ | + jerarquía |
| Políticas custom | ❌ | ❌ | ❌ | ❌ | ✅ |
| Vista multi-repo | ❌ | ❌ | ❌ | ❌ | ✅ |
| Notificaciones | ❌ | ❌ | ❌ | email | + webhook |
| AgentLock visible | ❌ | ❌ | propio | equipo | cross-team |
| Exportación | ❌ | ❌ | ❌ | CSV | + JSON + API |
| Cuota ai_resolve | 30/mes | 300/mes | ∞ | 3000/mes | ∞ |

*Pro tiene 30 ai_resolve/mes de prueba. Pro+ tiene ilimitado.

---

## 9. F6 COCKPIT — ESPECIFICACIÓN

### 9.1 Qué es

La 9ª vista del Cockpit TUI (Bubble Tea). Un dashboard visual para orquestar worktrees. El motor OWS (F1-F5, F7-F9) ya está 100% completo. El Cockpit ya tiene 8 vistas (21 archivos, 1691 LOC). F6 agrega la vista de Worktrees.

### 9.2 Hitos de implementación

| Hito | Descripción | Esfuerzo | Archivos nuevos |
|------|-------------|----------|-----------------|
| **F6-CORE** | Tarjetas de worktree + lista + integración OWS | 4h | `worktrees.go` (~350 LOC) |
| **F6-PLAN** | Plan enforcement capas 1+2 · owc plan-aware | 2h | `plan.go` (~150 LOC) |
| **F6-ACTIVATE** | Pantalla de activación · `ovav connect` | 1.5h | `activate.go` (~250 LOC) |
| **F6-DIFF** | Diff horizontal side-by-side · modo dual `Ctrl+T` | 2h | En `worktrees.go` |
| **F6-TIER** | Feature gating por `license.IsEnabled()` | 1.5h | En `worktrees.go` + `model.go` |
| **F6-TEAM+ENT** | Vistas de equipo · políticas · multi-repo | 3h | `team.go`, `policy.go` |
| **Total F6** | | **~14h** | |

### 9.3 Vista previa del Cockpit

```
┌──────────────────────────────────────────────────────────────┐
│  🛩️ OVAV Cockpit · Pro · user@email.com      Ctrl+T Dev    │
│  Plan: v40.0 — Python-Free                                   │
│──────────────────────────────────────────────────────────────│
│  ⬤ feature/continue-plan  🟢 clean  👤 thavren  🕐 6.7h     │
│     v40.0 · F6 Cockpit · ⚡1 conflicto                        │
│     [owd] [owv] [owlk] [owr]                                 │
│                                                              │
│  ⬤ hotfix/login-error     🟡 dirty  👤 soren   🕐 1.2h     │
│     v40.0 · F6 Cockpit · ✅ sin conflictos                    │
│     [owd] [owv] [owlk] [owr]                                 │
│──────────────────────────────────────────────────────────────│
│  owc:New  owd:Done  owv:Verify  owx:Route  ows:Prune        │
│  owl:List owu:Sync  owa:Abort   owr:Rescue owlk:Lock        │
│──────────────────────────────────────────────────────────────│
│  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░  ai_resolve: 45/300 · 15% usado    │
│  🔄 Update v1.1.0 disponible                     [ovav upd]  │
└──────────────────────────────────────────────────────────────┘
```

---

## 10. AUDITORÍA DE SPLIT — RESULTADOS

### 10.1 Inventario

| Métrica | Valor |
|---------|-------|
| Archivos Go analizados | 171 (sin tests) |
| LOC total | 35,460 |
| Paquetes PRODUCT | 17 · ~13,900 LOC |
| Paquetes CORE | 2 · ~13,000 LOC |
| Paquetes PUENTE | 6 · 5,793 LOC |

### 10.2 Puentes a refactorizar

| # | Paquete | LOC | Riesgo | Horas | Solución |
|---|---------|-----|--------|-------|----------|
| 1 | `cli` | 273 | BAJO | 1-2h | `shared/format` + `shared/filesystem` |
| 2 | `vault` | 383 | BAJO | 1-3h | `shared/vault_crypto` (interface) |
| 3 | `sbom` | 401 | MEDIO | 2-4h | `shared/sbom_types` (structs) |
| 4 | `convert` | 1224 | MEDIO | 2-4h | `shared/agent_formats` (interface) |
| 5 | `gitflow` | 1027 | ALTO | 3-6h | `shared/git_operations` (interface) |
| 6 | `install` | 2485 | ALTO | 4-8h | `shared/install_contract` (~50 LOC) |
| **Total** | | **5,793** | | **13-27h** | |

### 10.3 Estructura post-split

```
github.com/ovav/core        → dev tools + cPanel + validators
github.com/ovav/shared      → 6 interfaces (~600 LOC)
github.com/ovav/product     → Cockpit + OWS + todo usuario
```

Archivo completo de auditoría: `.ovav/plan/split_audit.yaml`

---

## 11. PLAN DE IMPLEMENTACIÓN COMPLETO

### Fase 2A — Split Arquitectónico

| Paso | Descripción | Esfuerzo |
|------|-------------|----------|
| S1 | Auditoría de dependencias | 3h ✅ |
| S2 | Bootstrap ovav-product (repo + go.mod + 17 paquetes) | 4h |
| S3 | Bootstrap ovav-core (repo + cPanel + sync module) | 4h |
| S4 | Connect flow (`internal/connect/` + endpoints cPanel) | 3h |
| S5 | Update client (`internal/update/` + WebSocket) | 3h |
| S6 | Convert exports (expandir engine + `//go:embed`) | 3h |
| S7 | Plugin system (`internal/plugins/` + carga lazy) | 3h |
| S8 | Windows build + smoke test | 3h |
| **Subtotal Split** | | **~26h** |

### Fase 2B — Refactor de Puentes

| Paso | Descripción | Esfuerzo |
|------|-------------|----------|
| R1 | Extraer `shared/cli` (format + filesystem) | 2h |
| R2 | Extraer `shared/vault_crypto` | 2h |
| R3 | Extraer `shared/sbom_types` | 3h |
| R4 | Extraer `shared/agent_formats` | 3h |
| R5 | Extraer `shared/git_operations` | 4h |
| R6 | Extraer `shared/install_contract` | 5h |
| **Subtotal Puentes** | | **~19h** |

### Fase 2C — F6 Cockpit

| Paso | Descripción | Esfuerzo |
|------|-------------|----------|
| F6-LICENSE | state.json + feature gating + fingerprint + activation client/server | 8h |
| F6-CORE | Worktree cards + lista + integración OWS | 4h |
| F6-PLAN | Plan enforcement capas 1+2 | 2h |
| F6-ACTIVATE | Pantalla connect en Cockpit | 1.5h |
| F6-DIFF | Diff horizontal + modo dual | 2h |
| F6-TIER | Feature gating por tier | 1.5h |
| F6-TEAM+ENT | Vistas equipo + enterprise | 3h |
| **Subtotal F6** | | **~22h** |

### Total Fase 2

| Bloque | Esfuerzo |
|--------|----------|
| Split (2A) | ~26h |
| Puentes (2B) | ~19h |
| F6 Cockpit (2C) | ~22h |
| **Total** | **~67h** |

### Orden recomendado

```
S1-S3 (bootstrap) → R1-R6 (puentes) → S4-S7 (connect+update+exports+plugins)
    ↓
F6-LICENSE → F6-CORE → F6-PLAN → F6-ACTIVATE → F6-DIFF → F6-TIER
    ↓
S8 (Windows) → F6-TEAM+ENT
```

---

## 12. MÉTRICAS DE ÉXITO

| Métrica | Objetivo | Medición |
|---------|----------|----------|
| FMA (First-attempt Merge Acceptance) | >85% | SQLite audit |
| Plan adherence | >90% | Worktrees con plan_id / total |
| Time-to-activation | <2 min | Click en "Conectar" hasta Cockpit desbloqueado |
| Cockpit adoption | >60% | Usuarios que usan Cockpit vs CLI directo |
| Windows smoke test | 0 crashes | Sesión de 30 min sin errores |
| Product binary size | <25 MB | `ls -lh ovav` |
| Multi-machine rejection rate | <0.1% | Intentos bloqueados por sharing |

---

## 13. DEPENDENCIAS Y PRERREQUISITOS

### Lo que DEBE estar completo antes de iniciar Fase 2

| Prerrequisito | Estado | Bloquea |
|---------------|--------|---------|
| Python→Go migración completa | 🔴 En curso | Todo Fase 2 |
| OWS F1-F9 100% funcional | ✅ Completo | F6-CORE |
| Fish v6 15 comandos | ✅ Completo | — |
| cPanel server funcional | ✅ Completo | S4, S5 |
| Convert engine funcional | ✅ Completo | S6 |
| Sistema estable (0 flaky tests) | 🟡 Pendiente verificación final | Todo |
| caps.yaml actualizado con referencia a este plan | 🟡 Este commit | Tracking |

### Lo que NO bloquea (puede construirse en paralelo)

- ovav.dev web (se necesita para S4 pero puede ser un mock inicial)
- Stripe integración (post-Fase 2)
- Windows CI/CD (S8 prueba manual primero)

---

## REFERENCIAS

- `caps.yaml` v40.0 — Plan canónico activo (Fase 1)
- `.ovav/plan/split_audit.yaml` — Auditoría de dependencias y clasificación
- `.ovav/plan/f6_cockpit_action_plan.md` — Plan detallado de F6 Cockpit v2.0
- `.ovav/plan/architecture_split_core_product.md` — Diseño arquitectónico del split
- `.ovav/research/2026-06-18-license-plan-enforcement/DECISION_BRIEF.md` — Eidren v1
- `.ovav/research/2026-06-19-quota-gating-activation-flows/DECISION_BRIEF.md` — Eidren v2

---

*Documento maestro de Fase 2. Generado por Thavren (Platform Engineering Lead).*
*Sesión: feature/continue-plan · HEAD `348aea47`*
*Próxima actualización: al completar Python→Go + estabilización.*
