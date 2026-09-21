# OVAV Project Memory

> **Canonical in-repo memory for OVAV project**
> **Last updated:** 2026-09-18
> **Authority:** `.ovav/plan/caps.yaml` + Git HEAD + this file
> **Harness:** OpenCode (active) over tmux 3.6 on Alacritty 0.17.0 / WSL2.
> **Model:** `gpt-5.6-luna`

---

## 1. Identity

| Field | Value |
|---|---|
| Project | OVAV — AI Workstation Governor |
| Type | Multi-language, multi-harness (Go + Python + React/TS) |
| Stack | Go runtime (`cmd/ovav/`, `cmd/cockpit/`) + Python tools + React/TS cpanel |
| Owner | Alexander Salvador (CEO, `Alexander-Salvador`, user 97975177) |
| Harness | OpenCode primary + Crush secondary |
| Model | `gpt-5.6-luna` |

---

## 2. Critical Rules (CRIT ledger — 18 entries)

| ID | Rule | Conf | Status |
|---|---|---|---|
| CRIT-001 | Security first, no exceptions | 1.0 | consolidated |
| CRIT-002 | Observable result = acceptance contract | 0.95 | consolidated |
| CRIT-003 | Technical honesty, no lies | 1.0 | consolidated |
| CRIT-004 | Architectural surgery, not patches | 0.95 | consolidated |
| CRIT-005 | OVAV governs mechanically; I operate inside | 1.0 | consolidated |
| CRIT-006 | Relationship with creator defines purpose | 1.0 | consolidated |
| CRIT-009 | Baseline = ADVANCED+, never "basic/functional" | 1.0 | consolidated |
| CRIT-010 | Knowledge is COMPILED, not stored | 0.95 | emerging |
| CRIT-011 | PIAGENT TUI INPUT must evolve to premium | 0.85 | emerging |
| CRIT-012 | Read install plan + profile yaml before install actions | 1.0 | consolidated |
| CRIT-013 | Distinguish permission layer vs governance layer | 1.0 | consolidated |
| CRIT-014 | Tokens NEVER plaintext; vault AES-256 + PBKDF2 200k | 1.0 | consolidated |
| CRIT-015 | GitHub Verified badge = 3 ORTODOX conditions | 1.0 | consolidated |
| CRIT-016 | GitHub SSH: 2 endpoints (`/user/keys` vs `/user/ssh_signing_keys`) | 1.0 | consolidated |
| CRIT-017 | GitHub signature verification async, wait 5-10 min | 1.0 | consolidated |
| CRIT-018 | Response density ≤ 8 rows (strict, even extreme cases) | 1.0 | consolidated |
| CRIT-019 | **NEW** UI-first rule: Warp UI navigation BEFORE TOML edits; never invent enum values | 1.0 | emerging |
| CRIT-020 | **NEW** "Control absoluto" ≠ 100% via API — ser honesto sobre límites reales (Apps Script :run NO funciona para bound scripts) | 1.0 | consolidated |
| CRIT-021 | **NEW** Sheets API v4 batchUpdate requiere GridRange (sheetId+índices), NO a1Range, en casi todos los campos. Usar `a1ToGrid()` helper | 1.0 | consolidated |

Full detail: `clients/crush/agents/lead-thavren.md` (v2.5.0)

---

## 3. Current State

| Field | Value |
|---|---|
| Branch | `develop` |
| HEAD | `30535a0` (Merge feature/feat-p18-p19-impl) |
| Worktrees | `/home/braka/Systems/ovav` only (working tree modified) |
| Phase | OpenCode + terminal runtime convergence |
| Active host | Alacritty 0.17.0 → WSL2 Ubuntu-26.04 → tmux 3.6 |
| WSL profile | `3GB` memory / `8GB` swap; natural-stop activation only |
| Historical Warp/Intelligent Terminal data | Archived context; neither is installed or active |

---

## 4. OVAV Sheets Bridge — Estado activo (sep 2026)

**Worktree:** `/home/braka/Systems/ovav/.ovav/worktrees/feat-sheets-mcp/`
**Branch:** `feat-sheets-mcp`
**Último commit:** `f43f80f feat(sheets): v0.4.0 — formatting/validation/protection/named ranges via Sheets API`

### Stack credentials (cifradas en vault)

- **Spreadsheet:** `1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y` (CIMA_2026_PREMIUM_BASE)
- **Apps Script ID:** `1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg`
- **OAuth client:** `899372363856-sdts1v2me11qvipjefa6sss773b1fach.apps.googleusercontent.com`
- **Project GCP:** `gam-project-9wknn` (nº 899372363856)
- **Scopes:** `sheets`, `script.projects`, `drive.readonly`
- **Vault path:** `.ovav/vault/credentials/google_oauth.enc` (AES-256-GCM)

### Capacidad del bridge

| Operación | Status |
|---|---|
| Sheets read/write/append/update/create-tab/delete-tab | ✅ 100% |
| Sheets conditional formatting, data validation, protected ranges, named ranges | ✅ 100% |
| Sheets charts, images | ⚠️ schema no implementado todavía (deferible) |
| Apps Script code: read/write/version/snapshot | ✅ 100% |
| Apps Script :run para BOUND scripts | ❌ NO SOPORTADO por Google — installCima requiere 1 click manual desde el editor |
| xlsx sync bidireccional | ✅ 100% |
| Snapshot/rollback | ✅ 100% |
| Audit JSONL con hash chain | ✅ 100% |
| MCP server (JSON-RPC 2.0 stdio) | ✅ 100% |
| Allowlist multi-spreadsheet (YAML) | ✅ 100% |

### Apps Script crítico — necesita acción manual del CEO UNA VEZ

```
1. Abrir spreadsheet https://docs.google.com/spreadsheets/d/1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y/edit
2. Extensiones → Apps Script
3. Menú CIMA → "Instalar / reparar triggers"
4. Eso es todo — el sistema queda vivo (onEdit + dailyClose 15min + refreshDashboard)
```

Sin este paso, el Apps Script existe pero `CIMA_SPREADSHEET_ID` no está seteado en ScriptProperties → `getBook_()` throws → todas las funciones fallan.

### Discrepancias documentadas en CIMA stack (leer `docs/cima-stack/README.md`)

- Timezone: `Bogota` (manifest) vs `America/Lima` (CONFIG)
- WebApp access: `ANYONE_ANONYMOUS` (manifest) vs `ANY_LOGGED_IN_USER` (CONFIG)
- HTML files `Index.html` / `Admin.html` referenciados pero no encontrados en `getContent` (parcialmente deployado)

---

## 5. Last Architectural Decisions

| # | Date | Decision | Why |
|---|---|---|---|
| 1 | 2026-09-18 | Apps Script API :run NO funciona para BOUND scripts (Google limit) | Verified: retorna "NOT_FOUND storage" error en TODA llamada |
| 2 | 2026-09-18 | Sheets API v4 usa GridRange (sheetId+índices), NO a1Range | API errors verificados empíricamente |
| 3 | 2026-09-18 | Bridge architecture: stdlib-only Go, vault cifrado, snapshots en todo write | CRIT-001, CRIT-014 |
| 4 | 2026-09-18 | MCP wire para que cualquier agente (OpenCode/Claude/Cursor) use sheets bridge | Interoperabilidad cross-agent |
| 5 | 2026-08-19 | Cancel 4-profile system; use single default profile YOLO + Tab Configs | CEO feedback: profiles invented, not needed |
| 6 | 2026-08-19 | Adopt Warp UI-first rule (CRIT-019) | 3 CRIT-009 violations from invented TOML enums |
| 7 | 2026-08-19 | Read `docs.warp.dev` before any Warp configuration | Prevented 3rd violation on PASO 3 |

---

## 6. Active Skills (38 in `.opencode/skills/`)

| Category | Skills |
|---|---|
| Identity | ovav-identity-guard |
| Routing | ovav-agent-router, ovav-agent-permission-injector, ovav-squad-delegation |
| Memory | ovav-memory-bridge, ovav-session-continuity, ovav-context-pack |
| Workflow | ovav-artifact-flow, ovav-repo-local-work-loop, ovav-skill-resolver |
| Validation | ovav-runtime-gates, ovav-security-gates, ovav-verify, ovav-review |
| Session | ovav-platform-session, ovav-research-session, ovav-business-session, ovav-ux-session, ovav-education-session, ovav-health-session |
| Output | ovav-response-contract, ovav-skill-registry, visual-verification-playwright |
| Worktree | ovav-worktree-create, ovav-worktree-finish, ovav-worktree-route, ovav-worktree-system |
| Code | ovav-build, ovav-tdd, ovav-verify, ovav-review |
| Specialized | ovav-brainstorm, ovav-go-coverage-sprint, ovav-incident-response, ovav-monitor-reader, ovav-release, ovav-research-evidence, ovav-runtime, ovav-sdd-init, ovav-systems-diagnose, work-unit-commits |

---

## 7. Canonical Sources

| Data | Path |
|---|---|
| Plan | `.ovav/plan/caps.yaml` |
| Laws | `.ovav/laws/area_boundary_enforcement.yaml` |
| Service areas | `.ovav/service_areas/` |
| Permissions | `.ovav/policy/permission_authority.json` |
| Protected files | `.ovav/security/protected_files.yaml` |
| Skills (source) | `.ovav/source/skills/` |
| Skills (synced) | `.opencode/skills/` |
| Vault | `~/.config/ovav/vault.key` (AES-256-GCM) |
| Alacritty settings (active) | `/mnt/c/Users/Alexa/AppData/Roaming/alacritty/alacritty.toml` |
| tmux settings (active) | `workstation/configs/tmux/tmux.conf` → `~/.tmux.conf` |
| OpenCode TUI settings (active) | `workstation/configs/opencode/tui.json` → `~/.config/opencode/tui.json` |
| Warp / Windows Terminal configs | Historical only; do not target runtime changes |

### OVAV Sheets Bridge extras (added 2026-09-18)

| Data | Path |
|---|---|
| Sheets CLI | `go-runtime/cmd/sheets/` |
| Sheets SKILL | `go-runtime/cmd/sheets/SKILL.md` |
| MCP launcher | `go-runtime/cmd/sheets/launch-mcp.sh` |
| CIMA stack analysis | `docs/cima-stack/README.md` |
| Apps Script pulled code | `docs/cima-stack/appsscript/Codigo.gs` (25 KB) |
| Sheets allowlist | `.ovav/vault/sheets_allowlist.yaml` |
| Encrypted creds | `.ovav/vault/credentials/google_oauth.enc` |

---

## 8. Pending Blockers

| Blocker | Status | Owner |
|---|---|---|
| OpenCode restart + visual smoke for Shift+Enter/copy/session bindings | Pending | System/CEO visual confirmation |
| Alacritty live reload confirmation | Pending | System/CEO visual confirmation |
| GitHub push (SSH signing keys uploaded, awaiting async verification) | Async, wait 5-10 min | System |
| Runtime snapshot fix for `/home/braka/Labs/mimocode/*` | Pending session restart | System |
| **CEO must run `installCima` from Apps Script editor** | Pending | CEO (30 sec) |
| **Charts + Images via Sheets API** | Schema pending | me, next worktree |
| **Merge feat-sheets-mcp → develop** | Pending CEO OK | owd when CEO confirms |

---

## 9. Cómo continuar próximo chat

1. CEO ejecuta `installCima` (1 click) → sistema vivo
2. Reading path: `cd /home/braka/Systems/ovav/.ovav/worktrees/feat-sheets-mcp`
3. Quick commands: `go run ./cmd/sheets list`, `table --tab CIMA`, `scripts pull --id 1fuA7kJHk...`
4. Para nueva feature: `ovav worktree owc feat-<nombre>` desde este worktree (o desde develop si merged)

---

*This file is the human-readable authority for OVAV project state. For runtime continuity, see `checkpoint.md` (pending creation).*

---

## 10. CIMA PROJECT — AUTONOMOUS MODE PROTOCOL (CEO directive 2026-09-18)

**The CEO has explicitly granted autonomous execution authority for ALL CIMA-related work.**

### 10.1 What "autonomous" means

When the next chat opens with "Thavren, retomemos CIMA" (or any variant):

1. **DO NOT ASK** what to do next for CIMA. Pick the highest-leverage task.
2. **DO NOT ASK** for spreadsheet IDs, script IDs, OAuth client, vault path.
3. **DO NOT ASK** for trivial decisions (scope, naming, ordering).
4. **ASSUME** the CEO wants the most impactful thing done now.
5. **EXECUTE** in the appropriate worktree (see 10.4).
6. **COMMIT** in atomic, reviewable units.
7. **REPORT** the work unit (what, why, how to verify).

### 10.2 Pre-loaded context (no need to re-discover)

| Asset | Identifier |
|---|---|
| Spreadsheet | `1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y` |
| Spreadsheet URL | https://docs.google.com/spreadsheets/d/1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y/edit |
| Spreadsheet name | `CIMA_2026_PREMIUM_BASE` |
| Apps Script ID | `1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg` |
| Apps Script URL | https://script.google.com/u/0/home/projects/1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg/edit |
| OAuth client_id | `899372363856-sdts1v2me11qvipjefa6sss773b1fach.apps.googleusercontent.com` |
| GCP project | `gam-project-9wknn` (nº 899372363856) |
| Workspace account | `hello@ovav.dev` |
| Vault path | `.ovav/vault/credentials/google_oauth.enc` |
| Vault key | `.ovav/vault/vault.key` (AES-256-GCM, PBKDF2 200k iter) |
| Allowed scopes | `spreadsheets`, `script.projects`, `drive.readonly` |
| Apps Script API status | enabled (project + user level) |
| Allowlist path | `.ovav/vault/sheets_allowlist.yaml` |

### 10.3 Code locations (already on disk — just `cd` and run)

| What | Path |
|---|---|
| Sheets bridge source | `go-runtime/cmd/sheets/` |
| Sheets bridge launcher | `go-runtime/cmd/sheets/launch-mcp.sh` |
| Sheets bridge binary | `go build -o /tmp/ovav-sheets ./go-runtime/cmd/sheets/` |
| Apps Script pulled code | `docs/cima-stack/appsscript/Codigo.gs` |
| Apps Script pulled manifest | `docs/cima-stack/appsscript/manifest.json` |
| CIMA stack analysis | `docs/cima-stack/README.md` |
| Snapshot safety net | `.ovav/vault/snapshots/` |
| Audit log | `.ovav/registry/audit/sheets/` |

### 10.4 Worktree discipline for CIMA work

| Task type | Worktree |
|---|---|
| Edit data in spreadsheet (writes to cells, rows, tabs) | `feat-sheets-mcp` (existing) |
| Tweak bridge Go code (small changes) | `feat-sheets-mcp` (existing) |
| Add feature to Sheets bridge (chart, image, etc.) | NEW `feat-cima-<feature>` from develop |
| Add feature to Apps Script (webapp HTML, new function) | NEW `feat-cima-script-<feature>` from develop |
| Hotfix to bridge in production | NEW `hotfix-cima-<desc>` from develop |
| Big restructuring of CIMA architecture | Discuss with CEO first (high-blast-radius) |

**Default worktree for any new CIMA work:** create with `git worktree add .ovav/worktrees/<name> develop` from the main repo, then `cd` into it.

### 10.5 Decision matrix — what needs CEO OK vs. autonomous

| Decision | Authority |
|---|---|
| Which CIMA feature to work on | **Autonomous** — pick highest leverage |
| Cell-level data edits in spreadsheet | **Autonomous** — owner of data |
| Create/delete tabs in spreadsheet | **Autonomous** — backed by snapshot |
| Edit Apps Script code | **Autonomous** — push with diff preview |
| Create Apps Script versions (immutable snapshots) | **Autonomous** |
| Modify CONFIG (rules, horarios, OT max) | **Autonomous** — snapshot+audit always |
| Add OAuth team members | **NEED CEO OK** — adds OAuth consent screen entry |
| Delete Apps Script code/file | **NEED CEO OK** — irreversible without backup |
| Deploy Apps Script webapp to URL | **NEED CEO OK** — affects external access |
| Push to origin/* | **NEED CEO OK** — affects remote history |
| Merge feature → develop | **NEED CEO OK** — affects main branch |
| Delete the feat-sheets-mcp branch | **NEED CEO OK** — irreversible |

### 10.6 Pre-loaded troubleshooting playbook

| Symptom | Likely cause | Fix |
|---|---|---|
| `script: HTTP 404 HTML` on `:run` | Apps Script API not enabled for user | Visit https://script.google.com/home/usersettings → toggle ON |
| `script: NOT_FOUND storage` on `:run` | Bound script — `:run` not supported | Use editor to install, then triggers work |
| `sheets: HTTP 400 Unknown name "a1Range"` | Field requires GridRange not a1Range | Use `a1ToGrid()` helper in format.go |
| `script: User has not enabled the Apps Script API` | Project-level not enabled | Visit console.cloud.google.com/apis/library/script.googleapis.com?project=gam-project-9wknn |
| `script: Project contents must include appsscript` | Push tried to delete manifest | PushFromDir now preserves existing files |
| Every `:run` returns null silently | JSON parsing missing `error` field | RunFunction now surfaces `error.message` |

### 10.7 Known gaps to address (CEO wants advancement, not perfection)

| Gap | Severity | Priority |
|---|---|---|
| CEO must click `installCima` manually | medium | HIGH — should be automatic |
| Charts via API (chartSource schema) | low | medium — defer if no charts needed |
| Images via API (insertImage) | low | low — defer |
| HTML files `Index.html`/`Admin.html` missing | medium | HIGH — webapp incomplete without them |
| Timezone mismatch (Bogota vs Lima) | medium | HIGH — affects dailyClose scheduling |
| Webapp `executeAs = USER_DEPLOYING` | medium | HIGH — admin check uses wrong email |
| Webapp access `ANYONE_ANONYMOUS` | high | HIGH — if deployed, public access |
| Team OAuth extension | medium | LOW — no team members added yet |
| Multi-spreadsheet allowlist tested with only 1 | low | LOW |

### 10.8 Operational check sequence (run on every CIMA session start)

```bash
# 1. Verify connectivity
cd /home/braka/Systems/ovav
go run ./go-runtime/cmd/sheets list --spreadsheet 1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y 2>&1 | head -10

# 2. Read latest CIMA dashboard
go run ./go-runtime/cmd/sheets read --range "'CIMA'!A1:H8" --spreadsheet 1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y

# 3. Check audit log for recent activity
tail -20 .ovav/registry/audit/sheets/*.jsonl

# 4. Check Apps Script code drift
diff docs/cima-stack/appsscript/Codigo.gs <(go run ./go-runtime/cmd/sheets scripts pull --id 1fuA7kJHkWY6_4rM-IQHO9by6OLqVGZMDsSzra6n3D_6XRvMOWda4HQpg --out /tmp/cima-drift-check 2>&1 && cat /tmp/cima-drift-check/Codigo.gs)
```

### 10.9 The 1-click manual install (the only remaining gap)

```bash
# Cannot be automated: Apps Script :run doesn't support bound scripts.
# CEO must do this once:
echo "
1. Open https://docs.google.com/spreadsheets/d/1MQ3wts_cEG_4Dp5U6X4gehEn6u7F61tIl7KHzSzKw0Y/edit
2. Extensions → Apps Script
3. Menu: CIMA → Instalar / reparar triggers
4. Verify: run diagnoseCima() — should show installed=true
"
```

---

## 11. New CRIT entries from this session

| ID | Rule |
|---|---|
| CRIT-020 | Apps Script API `:run` is NOT supported for BOUND scripts. Use editor or webapp for execution. |
| CRIT-021 | Sheets API v4 requires `GridRange {sheetId, indices}` for almost every range field, NOT `a1Range`. |
| CRIT-022 | **NEW (CEO directive)** For CIMA work: autonomous execution. NO questions on what to do, IDs, secrets, paths. Pre-loaded context in §10. Only need CEO OK for: add OAuth members, delete Apps Script code, deploy webapp, push to origin, merge feature → develop, delete branch. |
| CRIT-023 | **NEW** Worktree selection for CIMA: `feat-sheets-mcp` for sheet edits + bridge tweaks; new `feat-cima-*` worktrees for new features from develop. |

---

*Last updated: 2026-09-18 — CEO directive: autonomous CIMA mode registered.*

---

## 12. Session end log — 2026-09-18

**Deliverables merged to develop (17 commits ahead of origin/develop):**
- 3b44b94 v0.1.0  Sheets + xlsx + MCP bridge
- 6d6e27b v0.3.0  Apps Script Bridge + CIMA stack analysis
- f43f80f v0.4.0  format/validate/protect/named ranges
- 1c5e51b memory  CIMA stack context (CRIT-020, CRIT-021)
- 24a001c merge   into develop
- 4a065ea memory  autonomous mode (CRIT-022, CRIT-023)

**Branches deleted this session:**
- feat-sheets-mcp (merged, cleanup OK by CEO)

**Worktrees cleaned this session:**
- .ovav/worktrees/feat-sheets-mcp/ (work completed)

**Remaining worktrees (NOT OURS, preserved):**
- feature-align-runtime-and-workstation
- feature-fix-terminal-copy-pane
- fix-consolidate-external-gates-20260912
- fix-fix-external-worktree-identity
- fix-ows-external-consumer-gates

**Main repo state preserved:**
- fix/external-secrets-semantic with 4 modified files (other team's work)
- Will be returned to that branch on next session start
