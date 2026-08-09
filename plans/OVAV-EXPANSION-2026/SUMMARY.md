# OVAV — Temario Expansión 2026 (AUDITADO)

> CEO → Thavren | Generado: 2026-08-02 | Estado: AUDITADO vs realidad del codebase

---

## ✅ YA EXISTE EN OVAV — Confirmado por código

---

### OVAV MCP Servers
**Evidence:** `cmd/cpanel/memory_mcp.go` + `internal/browser/mcp_server.go`

- **Memory MCP Relay** en cPanel: HTTP → `memory-mcp` subprocess via stdio MCP bridge
- **Browser MCP Server**: `browser/controller.go` + `mcp_server.go` — JSON-RPC 2.0 over stdio, tool calls para browser automation
- **opencode.json MCP format** validado por `convert/opencode_config.go`
- **Tool readiness matrix** incluye `mcp: active_internal`

→ **Estado:** MCP CORES existen. Lo que NO existe es el **Customer Memory MCP** (tipo influxdb/sqlite para issues del usuario final).

---

### OVAV Vault — FULLY IMPLEMENTED
**Evidence:** `cmd/ovav/main.go:79-84, 1029-1214`

```
ovav vault scan      → vault.ScanAssets() ✅
ovav vault encrypt   → vault.EncryptAllAssets() ✅
ovav vault decrypt   → vault.DecryptAllAssets() ✅
ovav vault gen-key   → vault.GenerateKey() ✅
ovav login           → PBKDF2(machine_id) → vault key → session ✅
```

- AES-256-GCM encryption
- Vault key derivation via PBKDF2-HMAC-SHA256 (600_000 iteraciones)
- Session storage en `~/.local/share/ovav/session`
- Tokens decrypted from vault via `infra.DecryptTokensFromVault()`

→ **Estado:** COMPLETAMENTE IMPLEMENTADO. No hay gap.

---

### OVAV Plan Mode + Worktree System (OWS)
**Evidence:**
- `ovav tailor select` en `main.go:87` → plan selection (nucleo|studio|command)
- `ovav-brainstorm/SKILL.md` → HARD GATE protocol con preguntas por área
- `ovav-worktree-system/SKILL.md` → `owc`, `owd`, `owl`, `owv` lifecycle
- `ovav-repo-local-work-loop/SKILL.md` → route→context→work→validate→close
- `caps.yaml` + `PLAN.md` → sistema de planes por proyecto
- **PL-0 Autonomous Plan Mode** en `AGENTS.md` → NEW IDEA → BRAINSTORM → PLAN → EXECUTE

→ **Estado:** EXISTS. Gap potencial: **area-specific questions más ricas** (ver P3 revised abajo).

---

### OVAV Agents System
**Evidence:**
- `convert_agents` cmd → empaqueta agents a runtimes (opencode, mimocode, cursor, claude code)
- 10 areas, leads, squads con personalidades
- `ovav-squad-delegation/SKILL.md` → workflow + agent() delegation
- `ovav-agent-router/SKILL.md` → domain detection + routing
- `ovav-context-pack/SKILL.md` → compact context injection

→ **Estado:** EXISTS. Funcionalidades cubiertas.

---

### Testing System
**Evidence:** `internal/testing/advance/`
- Probes (security scanning)
- `autonomousAttack()` con gating por `finding.Status == "CONFIRMED"`
- Remediation loop con `applyFix()` (genera patches de comment-only por ahora)
- Security hardening validators (F0-F5)
- `testing_remediation` service area

→ **Estado:** EXISTS. Gap: **interactive test init** que pregunte tipo de proyecto + **E2E visual con playwright-cli**.

---

### OVAV Memory Bridge
**Evidence:** `cmd/cpanel/memory_mcp.go` — MCP relay para memory
- Memory MCP relay HTTP endpoint
- Daily update poller

→ **Estado:** EXISTS. Gap: **customer-facing issue memory** (diferente scope).

---

## 🎯 NUEVAS IDEAS REALES — post-audit

---

### P1r. OVAV Customer Memory MCP (REVISED — es genuinamente nuevo)

**Lo que existe:** `memory_mcp.go` — relay para memory del developer
**Lo que falta:** Issue memory focado en el **usuario final / cliente**

**Concepto:**
```
Cliente reporta: "bash tool broken"
→ OVAV Customer Memory detecta patrón
→ Injecta: "02/05/2024 — bash tool issue → clean reinstall mimocode → SOLUCIONADO"
→ AI aplica fix en minutos vs días
```

**Diferencia clave:**
- Memory actual = developer context (commits, decisiones técnicas)
- Customer Memory = issues del USUARIO FINAL, errores recurrentes, soluciones aplicadas

**Requiere:** Plan nuevo subsistema.

---

### P3r. OVAV Brainstorm — Área-Specific Questions (REVISED)

**Lo que existe:** `ovav-brainstorm/SKILL.md` con HARD GATE + preguntas genéricas
**Lo que falta:** Preguntas específicas por cada área profesional

**Concepto:**
```
NEW IDEA detectada → BRAINSTORM activa
→ Si área = UX/UI → "¿WCAG AA o AAA? ¿Mobile-first? ¿Design system?"
→ Si área = Backend → "¿ORM o raw SQL? ¿Monorepo? ¿Auth: JWT o sessions?"
→ Si área = Platform → "¿Go modules o single module? ¿Testing strategy?"
→ Genera DESIGN.md → PLAN.md → EXECUTE
```

**Beneficio medido:** CEO detecta 5x menor gasto de tokens con plan vs implementación directa.

**Requiere:** Extender `ovav-brainstorm/SKILL.md` con preguntas por área. Skill existente, mejora incremental.

---

### P4. CRM WhatsApp — Genuinamente nuevo
**No existe nada de esto en OVAV.**
WhatsApp Business API integration, webhooks, retry logic, conversation management.

---

### P7. Kimi CLI Research — Investigar para adopción
**No hay investigación de Kimi CLI en OVAV todavía.**
`kimi-k7` adapter existe en `adapters.go` pero features de Kimi CLI (/yolo, /swarm, skills.sh) no fueron analizadas.

→ Handoff a **Eidren** (Research Intelligence)

---

## 🔵 INFRAESTRUCTURA — Quick wins

### Wezterm Nightly
**Acción:** `wezterm --version` → si es old, update.
No requiere plan.

---

## 📊 RESUMEN POST-AUDIT

| # | Tema | Estado | Acción |
|---|---|---|---|
| P1r | OVAV Customer Memory MCP | 🎯 NUEVO (diferente scope que memory_mcp.go) | Generar plan |
| P3r | Brainstorm + área questions | ⚡ MEJORA (skill existe) | Extender skill preguntas |
| P4 | CRM WhatsApp | 🎯 NUEVO | Generar plan → handoff Dante |
| P7 | Kimi CLI /yolo /swarm | 🔍 INVESTIGAR | Handoff Eidren |
| Wezterm | Wezterm Nightly | ✅ ACCIONABLE | 1 comando |
| **VAULT-2026** | **Secrets subsystem upgrade** | **🎯 PLAN CREADO** | **plans/OVAV-VAULT-2026/PLAN.md** |
| OWS/Plan/Agents/MCP cores | Implementado | ✅ HECHO | Ninguna |

---

## 🗺️ PRÓXIMOS PASOS

1. **Customer Memory MCP** → plan completo
2. **Brainstorm preguntas por área** → extensión skill (bajo esfuerzo)
3. **CRM WhatsApp** → plan → handoff Dante
4. **Kimi CLI research** → handoff Eidren
5. **Wezterm** → update ahora
6. **OVAV Vault 2026** → ejecutar Phase 1-5 (plans/OVAV-VAULT-2026/PLAN.md)

---

*Thavren — Platform Engineering — 2026-08-02 (post-audit)*
