# OVAV Sprint Backlog — Ataque 2026

> Generado: 2026-07-31 18:35 UTC-5
> Status: TRACKING — not started

---

## ✅ Sprint 1 — COMPLETADO (`a28061dd`)

- [x] Fix `defend scan` — real filesystem verification, no hardcoded targets
- [x] Pre-commit hook creado — secrets + gofmt guards
- [x] RunActiveScan() — 4 surfaces reales verificadas

---

## 🔴 Sprint 2 — Governor Coverage Sprint (P0)

**Meta:** 80% → 85%+ ✅ **ACHIEVED: 90.9%**

### Cobertura actual (2026-08-02)

| Función | Archivo | Coverage |
|---|---|---|
| `QuickPainScorer` | bridge.go | 100.0% ✅ |
| `CountActiveAlertsQuick` | bridge.go | 100.0% ✅ |
| `CountPendingDelegationsQuick` | bridge.go | 85.7% ✅ |
| `CountOutstandingDecisionsQuick` | bridge.go | 90.9% ✅ |
| `VerifyTrust` | bridge.go | 100.0% ✅ |
| `GetByStatus` | delegation_protocol.go | 100.0% ✅ |
| `NewInterruptEngine` | interrupt_engine.go | 100.0% ✅ |
| `maybeRotate` | session_feed.go | 35.3% ⚠️ P3 |
| `QuickSelfDiagnosis` | bridge.go | 76.9% ⚠️ P3 |

**Owner:** Thavren (Platform Engineering)
**Status:** ✅ COMPLETADO (90.9% > 85% target)

---

## 🟡 Sprint 3 — Wire Validators to CLI (P1)

**Meta:** `ovav validate` expone los 3 validators referenciados en permission_authority.json ✅ **COMPLETED**

### Validators estados

| Validator | CLI | Status |
|---|---|---|
| `check_living_integrity` | `ovav validate living_integrity` | ✅ WIRIRED |
| `check_secrets_hygiene` | `ovav validate secrets_hygiene` | ✅ WIRIRED |
| `check_permission_policy_drift` | `ovav validate permission_drift` | ✅ WIRIRED |

**Owner:** Thavren (Platform Engineering)
**Esfuerzo estimado:** 1-2h
**Status:** ✅ COMPLETADO (2026-08-02)

---

## 🟡 Sprint 4 — external_directory_audit (P1)

**Meta:** Verificar que paths peligrosos tengan deny explícito

### Gap

Total Freedom = `external_directory: {"*": allow}` — excesivamente permisivo, no hay verificación de que `sudo`, `rm -rf /`, `mv /*` estén denegados.

### Validator a crear

`external_directory_audit.go`:
- Liste todos los paths en `external_directory`
- Verifique deny explícito para operaciones super-admin
- Genere alerta si hay super-admin paths sin restricción

**Owner:** Thavren (Platform Engineering)
**Esfuerzo estimado:** 1-2h
**Status:** PENDING

---

## 🟢 Sprint 5 — Agent Permission Fuzzing (P2)

**Meta:** Property-based tests para agent_permission_invariants.go

### Edge cases a fuzzear

- Frabricated role scopes
- Malformed external_directory entries
- Scope injection attacks
- Permission escalation vectors

**Owner:** Thavren (Platform Engineering)
**Esfuerzo estimado:** 2h
**Status:** PENDING

---

## 🔴 Issues Adicionales Detectados

### ISSUE-2026-0731-SPRINT2 — governor zero-coverage functions

**Archivos:** `go-runtime/internal/governor/bridge.go`, `delegation_protocol.go`, `interrupt_engine.go`

**Funciones:** QuickPainScorer, CountActiveAlertsQuick, CountPendingDelegationsQuick, CountOutstandingDecisionsQuick, VerifyTrust, GetByStatus, NewInterruptEngine

**Acción:** Agregar tests unitarios en `bridge_test.go`, `delegation_protocol_test.go`, `interrupt_engine_test.go`

---

### ISSUE-2026-0731-SPRINT3 — validators not wired to CLI

**Archivos:** `go-runtime/cmd/ovav/main.go`, `go-runtime/internal/validators/`

**Gap:** Los 3 validators de security (`check_living_integrity`, `check_secrets_hygiene`, `check_permission_policy_drift`) están en el código pero no expuestos como subcommands de `ovav validate` o `ovav doctor`.

**Acción:** Wirear en `main.go` y crear tests de integración

---

### ISSUE-2026-0731-SPRINT4 — external_directory audit gap

**Gap:** No hay validator para verificar que `external_directory: {"*": allow}` no exponga paths peligrosos.

**Acción:** Crear `external_directory_audit.go` en validators/

---

### ISSUE-2026-0731-CMD-COV — cmd/ovav coverage 54%

**Archivo:** `go-runtime/cmd/ovav/`
**Coverage actual:** 54%
**Meta:** 70%+

**Acción:** Tests de integración para `defense_cli.go`, `govern_cli.go`, `session_greeting_cli.go`

---

### ISSUE-2026-0731-SESSION-FEED — maybeRotate 29.4%

**Archivo:** `go-runtime/internal/governor/session_feed.go:268`
**Coverage actual:** 29.4%
**Función:** `maybeRotate` — rotación de session feed

**Acción:** Agregar tests para edge cases de rotación

---

## 📊 Cobertura Actual (2026-08-02)

| Paquete | Coverage | Meta | Status |
|---|---|---|---|
| `permissions` | 94.7% | ✅ | — |
| `security/defense` | 95.5% | ✅ | — |
| `governor` | 90.9% | 85%+ | ✅ |
| `validators` | ~80% | 80%+ | ✅ |
| `cmd/ovav` | 54.0% | 70%+ | 🔴 PENDING |
| **Overall** | **~83%** | 80%+ | 🟡 |

---

## 🚀 Prioridad de Ejecución

```
Sprint 2 (P0) → Sprint 3 (P1) → Sprint 4 (P1) → Sprint 5 (P2)
```
