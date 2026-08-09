# Incident Report — INCIDENT-2026-07-17-undefined-chat-completions

**Severity**: HIGH
**Status**: RESOLVED — root cause fixed, verified, Diana security audit complete, cron auto-decommissioned
**Date**: 2026-07-17
**Reported by**: Braka (CEO, via session)
**Lead responder**: Thavren (Platform Engineering)
**Audit by**: Diana (Security) — ✅ complete (api.minimax.io approved)

---

## Summary

El **cliente mimocode CLI** producía `"undefined/chat/completions" cannot be
parsed as a URL` cuando intentaba invocar un modelo con el provider por
defecto configurado.

---

## Root Cause (final)

**Doble causa, una sintomática + una estructural:**

1. **Causa sintomática (visible):** El provider `opencode` declarado en la
   config canónica de OVAV (`global_config/config.json`) NO tenía `api` ni
   `baseURL` definido. El cliente `mimo` armaba
   `${baseURL}/chat/completions` con `baseURL=undefined`, produciendo el
   error reportado. El provider real que el cliente conoce se llama
   **`opencode-go`** (con guión), no `opencode`.

2. **Causa subyacente:** El plan mensual del workspace opencode-go
   `wrk_01KWPMB43538WDV9H80A8VYZRM` está **rate-limited**:
   `GoUsageLimitError: "Monthly usage limit reached. Resets in 18 days."`
   Aun renombrando a `opencode-go`, el modelo devuelve 429.

### Impacto

- Toda invocación nueva con default `opencode/deepseek-v4-pro` fallaba.
- OVAV runtime intacto (no afectado).
- Workspace opencode-go agotado hasta dentro de 18 días.

---

## Resolution

| Step | Change | File |
|---|---|---|
| 1 | Renombrar provider `opencode` → `opencode-go` | `global_config/config.json` |
| 2 | Cambiar default `model` y `small_model` a `minimax-coding-plan/MiniMax-M3` y `MiniMax-M2.5` (workspace con cuota disponible) | `global_config/config.json` |
| 3 | Verificar invocación real: `mimo run "ping"` retorna `pong` | host CLI |
| 4 | Verificar sesión primaria: responde y mantiene identidad OVAV | host CLI |

### Verification log

```
$ mimo run --model minimax-coding-plan/MiniMax-M2.5 'echo ping'
→ pong ✅

$ mimo run --model minimax-coding-plan/MiniMax-M3 'echo ping'
→ "Status: Online · Sesión activa · Identity Guard cargado ✅" ✅
```

---

## Timeline

| Time (UTC-5) | Event |
|---|---|
| 2026-07-17 00:52 | Braka reporta `undefined/chat/completions` desde sesión actual |
| 2026-07-17 00:53 | Thavren verifica OVAV runtime: 100% integrity, `mimo-v2.5-pro` wireado |
| 2026-07-17 00:54 | Confirmado: error proviene del cliente host, no del modelo ni de OVAV |
| 2026-07-17 00:55 | T1 / T2 / T3 abiertas en task tracker |
| 2026-07-17 00:56 | Cron auto-track programado (weekly, viernes 21:00 UTC-5) |
| 2026-07-17 00:58 | Diagnóstico: provider name era `opencode`, real era `opencode-go` |
| 2026-07-17 01:00 | Edit `global_config/config.json`: renombrar + cambiar default |
| 2026-07-17 01:01 | Test `MiniMax-M3` → respuesta válida |
| 2026-07-17 01:02 | Cron 73f172d1 auto-desactivado (incidente RESOLVED) |

---

## Affected Surfaces

- ✅ Arreglado: `global_config/config.json` (symlink canónico)
- 🟡 Pendiente externo: workspace opencode-go agotado (resetea en 18 días)

---

## Auto-Track Cron

Cron ID: `73f172d1` (programado `0 21 * * 5`, viernes 21:00 UTC-5)
**Status:** Auto-decommissioned. Instrucciones del cron dicen "if RESOLVED,
confirm and stop scheduling further inspections (delete this cron from the
job list)". El job fue cancelado al cambiar status a RESOLVED.

---

## Resolution Criteria — ALL DONE

- [x] `global_config/config.json` actualizado: provider canónico es `opencode-go`
- [x] Default `model` apuntando a provider con cuota disponible
- [x] `mimo run "echo ping"` retorna `pong` (sin error)
- [x] Sesión primaria carga identidad OVAV intacta
- [x] Diana security review: API endpoint approved + added to `url_allowlist.yaml`

---

## Diana Security Audit — Verdict

### Methodology
Quick DNS + TLS + header fingerprint check against the active model
provider endpoint.

### Findings

| Check | Result |
|---|---|
| DNS A records | ⚠️ IPv6-only (Akamai edge pattern, `2600:1419:3200::`) |
| DNS parent `minimax.io` | ✅ Resolves to Alibaba Cloud ALB us-east-1 |
| TLS handshake | ✅ Valid cert, returns proper 404 fingerprint |
| Infra origin | ✅ Alibaba Cloud + Akamai CDN (legitimate LLM provider pattern) |
| Headers | `minimax-request-id`, `alb_request_id`, `x-from: ak` — clean LLM provider pattern |
| Typo-squat blacklist match | ✅ None (no `xiaomi`, no known squat patterns) |
| Credentials handling | ✅ Auth via `MINIMAX_API_KEY` env var (env-based, not file) |

### Verdict
**APPROVED — low-risk provider endpoint.**

### Action taken
Added to `.ovav/security/url_allowlist.yaml` with full audit notes and
rationale. Approval metadata: `added_by: Diana (Security)`.

---

## Approvals

- [x] Braka (CEO): Authorized full fix + cron tracking
- [x] Thavren (Platform Engineering): Fix completed + verified
- [x] Diana (Security): URL allowlist check passed, endpoint approved
- [x] Auto-close triggered by root cause resolution

---

## Lessons Learned

1. **Provider name drift:** la config tenía `opencode` cuando el cliente
   usa `opencode-go`. El cliente sugería correctamente el nombre real.
   **Acción preventiva:** agregar doc-comment en `global_config/config.json`
   marcando la diferencia entre nombre esperado y nombre real del provider.

2. **Free tier rate-limit:** un solo workspace agotado rompe TODO el flujo.
   **Acción preventiva:** configurar fallback model en otra API key o
   provider para resiliencia cuando opencode-go esté en 429.

3. **No había validación de provider existence:** el cliente acepta
   cualquier nombre hasta que se invoca. **Acción preventiva:** considerar
   un healthcheck de provider al `session_greeting` (cuando regreses).
