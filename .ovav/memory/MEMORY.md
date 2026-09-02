# OVAV Project Memory

> **Canonical in-repo memory for OVAV project**
> **Last updated:** 2026-09-02
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

## 4. Last 5 Architectural Decisions

| # | Date | Decision | Why |
|---|---|---|---|
| 1 | 2026-08-19 | Cancel 4-profile system; use single default profile YOLO + Tab Configs | CEO feedback: profiles invented, not needed |
| 2 | 2026-08-19 | Adopt Warp UI-first rule (CRIT-019) | 3 CRIT-009 violations from invented TOML enums |
| 3 | 2026-08-19 | Read `docs.warp.dev` before any Warp configuration | Prevented 3rd violation on PASO 3 |
| 4 | 2026-08-18 | P19 OWS Warp adapter in Go (`warp://tab_config/<name>` URI) | Native scheme, 4/4 tests passing |
| 5 | 2026-08-18 | P18: 4 Tab Configs (ovav_core/agent/review/systems) | Pre-configured layouts per workflow context |

---

## 5. Active Skills (38 in `.opencode/skills/`)

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

## 6. Canonical Sources

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

---

## 7. Pending Blockers

| Blocker | Status | Owner |
|---|---|---|
| OpenCode restart + visual smoke for Shift+Enter/copy/session bindings | Pending | System/CEO visual confirmation |
| Alacritty live reload confirmation | Pending | System/CEO visual confirmation |
| GitHub push (SSH signing keys uploaded, awaiting async verification) | Async, wait 5-10 min | System |
| Runtime snapshot fix for `/home/braka/Labs/mimocode/*` | Pending session restart | System |

---

*This file is the human-readable authority for OVAV project state. For runtime continuity, see `checkpoint.md` (pending creation).*
