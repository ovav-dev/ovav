# OVAV Project Memory

> **Canonical in-repo memory for OVAV project**
> **Last updated:** 2026-08-19
> **Authority:** `.ovav/plan/caps.yaml` + Git HEAD + this file
> **Harness:** OpenCode (active). Warp 2026 (terminal).
> **Model:** `minimax-coding-plan/MiniMax-M3`

---

## 1. Identity

| Field | Value |
|---|---|
| Project | OVAV — AI Workstation Governor |
| Type | Multi-language, multi-harness (Go + Python + React/TS) |
| Stack | Go runtime (`cmd/ovav/`, `cmd/cockpit/`) + Python tools + React/TS cpanel |
| Owner | Alexander Salvador (CEO, `Alexander-Salvador`, user 97975177) |
| Harness | OpenCode primary + Crush secondary |
| Model | `minimax-coding-plan/MiniMax-M3` |

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
| Worktrees | `/home/braka/Systems/ovav` only (clean) |
| Phase | Warp 2026 master plan — UI config phase (PASO 4 pending) |
| P11 suite | 24 PASS / 0 FAIL / 7 DEFERRED / 31 total |
| Warp settings.toml | Restored from `settings.toml.pre-p5v2-20260819-000213` |
| MiniMax endpoint | ✅ Configured (PASO 2 done) |
| Terminal mode | ✅ Configured (PASO 3 done) |

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
| Warp settings | `%LOCALAPPDATA%\warp\Warp\config\settings.toml` |

---

## 7. Pending Blockers

| Blocker | Status | Owner |
|---|---|---|
| PASO 4 — Default profile YOLO + Tab Configs import | Awaiting CEO | CEO (manual UI) |
| P10 — Secret redaction mode = asterisks | Awaiting CEO | CEO (manual UI) |
| GitHub push (SSH signing keys uploaded, awaiting async verification) | Async, wait 5-10 min | System |
| Runtime snapshot fix for `/home/braka/Labs/mimocode/*` | Pending session restart | System |

---

*This file is the human-readable authority for OVAV project state. For runtime continuity, see `checkpoint.md` (pending creation).*
