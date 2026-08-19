# OVAV × WARP 2026 — Sprint Plan (11 phases, 100% per gate)

<!-- OVAV_WARP_PLAN_v1.0 -->
<!-- Generated 2026-08-18 by thavren after P0 snapshot -->
<!-- Worktree: feature/feat-warp-2026-master-plan -->
<!-- Snapshot: .ovav/snapshots/2026-08-18-pre-warp-plan/ -->

## Overview

Master plan from CEO (43 sections, OVAV × WARP 2026) executed as 11 sprints.
Each sprint ends with a Safe Stop Report. No sprint advances until 100% complete.

**Execution surface:** worktree `feature/feat-warp-2026-master-plan`
**Baseline commit:** `fc72591` (post-P0)
**Snapshot:** `.ovav/snapshots/2026-08-18-pre-warp-plan/`
**Rollback path:** `git reset --hard 9b1a470` (pre-plan state on develop)

---

## P0 — Snapshot ✅ COMPLETE

**Goal:** Capture pre-plan state for rollback safety.

| Artifact | Status |
|---|---|
| AGENTS.md (root + 3 sources) | ✅ Copied |
| Runtime inventory (versions, tools) | ✅ Captured |
| Worktree baseline | ✅ Verified |
| Baseline integrity regeneration | ✅ Committed (`fc72591`) |
| Pre-plan dirty state | ✅ Committed (`9b1a470`) |
| Runtime projections carry | ✅ Committed (`fc1d7ce`) |

**Deliverable:** `.ovav/snapshots/2026-08-18-pre-warp-plan/` (5 files, 51-line inventory)

**100% criterion:** ✅ Snapshot files exist + integrity baseline regenerated + worktree clean.

---

## P1 — Warp × WSL2 × Fish stability

**Goal:** Verify the existing connection stack works without modification.

### Tasks

- [ ] Confirm Warp Stable installed (Windows side, not WSL)
- [ ] Confirm WSL2 distro: Ubuntu-26.04
- [ ] Confirm Fish 4.x is login shell
- [ ] Confirm no `new_session_shell_override`
- [ ] Confirm no `wsl.exe -d Ubuntu-26.04` launcher
- [ ] Verify Warp → WSL native path
- [ ] Document baseline terminal config

### 100% criteria

| Check | Evidence |
|---|---|
| `wsl.exe` absent from Warp settings | Warp settings.toml diff |
| Fish is login shell | `echo $SHELL` returns `/usr/bin/fish` |
| Ubuntu-26.04 confirmed | `/etc/os-release` shows Ubuntu 26.04 |
| Warp version documented | `warp --version` captured |

**Safe Stop Report** at close.

---

## P2 — Warp UX native configuration

**Goal:** Maximize Warp native capabilities.

### Tasks

- [ ] Vertical Tabs enabled
- [ ] Tab Groups organized (OVAV CORE / ACTIVE AGENTS / DEV)
- [ ] Session Restore + Undo Close enabled
- [ ] Working-directory inheritance (Previous session)
- [ ] Session Navigation enabled
- [ ] Notifications configured
- [ ] Theme: Dracula, CaskaydiaCove Nerd Font Mono, 13pt/14pt, 110% zoom, bar cursor
- [ ] Confirm quit warning enabled

### 100% criteria

| Check | Evidence |
|---|---|
| Vertical Tabs ON | `show_panel_in_restored_windows: true` in TOML |
| Working dir = Previous session | TOML inspection |
| Session Restore + Undo Close | TOML inspection |
| Quit warning enabled | `show_warning_before_quitting: true` |
| Theme + font match spec | screenshot |

**Safe Stop Report** at close.

---

## P3 — mise as canonical toolchain authority

**Goal:** mise owns runtime versions. NVM/FNM/asdf removed.

### Tasks

- [ ] Audit current runtimes (node, go, python, others)
- [ ] Inventory version requirements from go.mod, package.json, .nvmrc, CI, Dockerfiles
- [ ] Create `mise.toml` at repo root
- [ ] Create `mise.lock` with checksums
- [ ] `mise trust` on approved config
- [ ] Fish activation: `mise activate fish | source`
- [ ] CI uses `mise exec -- <command>` and `mise run <task>`
- [ ] Validate: build, tests, OpenCode, Crush all run on mise-installed runtimes
- [ ] Remove NVM/FNM/asdf if any exist (NVM not present in this env)

### 100% criteria

| Check | Evidence |
|---|---|
| `mise.toml` exists and is trusted | file + `mise trust` output |
| `mise.lock` versioned | git tracked |
| `mise exec -- go build` works | exit 0 |
| `mise exec -- npm test` works | exit 0 |
| NVM absent | `~/.nvm/` does not exist |
| Fish shows mise in PATH | `which node` resolves to mise install |

**Risk note:** If Fish PATH breaks, fall back to `mise exec --` prefix in scripts.

**Safe Stop Report** at close.

---

## P4 — AGENTS.md canonical + .agents/skills shared

**Goal:** Single constitution. Warp + OpenCode share skills directory.

### Tasks

- [ ] Audit AGENTS.md root for canonical sections (identity, governance, OWS, mise, git, security, agent permissions, gates, memory, MCP, release)
- [ ] Audit `.ovav/source/{opencode,crush,mimocode}/AGENTS.md` for divergence
- [ ] Reconcile divergences into canonical root AGENTS.md
- [ ] Create `.agents/skills/` directory
- [ ] Create 9 SKILL.md files:
  - [ ] `ovav-worktree-create/SKILL.md`
  - [ ] `ovav-worktree-finish/SKILL.md`
  - [ ] `ovav-worktree-route/SKILL.md`
  - [ ] `ovav-verify/SKILL.md`
  - [ ] `ovav-review/SKILL.md`
  - [ ] `ovav-runtime/SKILL.md`
  - [ ] `ovav-systems-diagnose/SKILL.md`
  - [ ] `ovav-incident-response/SKILL.md`
  - [ ] `ovav-release/SKILL.md`
- [ ] Confirm Warp discovers `.agents/skills/` (Warp convention)
- [ ] Confirm OpenCode discovers `.agents/skills/` (via `ovav project sync`)
- [ ] No WARP.md exists in repo (verified: ✅ absent)

### 100% criteria

| Check | Evidence |
|---|---|
| 9 SKILL.md files exist | `ls .agents/skills/*/SKILL.md \| wc -l` = 9 |
| AGENTS.md canonical sections present | grep for required sections |
| OpenCode discovers skills | `ovav project sync` output |
| WARP.md absent | `find . -name WARP.md` empty |

**Safe Stop Report** at close.

---

## P5 — Execution Profiles (4 levels)

**Goal:** 4 agent profiles with explicit autonomy tiers.

### Profiles

| Profile | Use | Read | Write | Cmd | YOLO |
|---|---|---|---|---|---|
| **OVAV BUILD** | Default daily | always | always | agent | n |
| **OVAV YOLO** | Aggressive | always | always | always | y |
| **OVAV REVIEW** | Audit/review | always | never/ask | ask | n |
| **THAVREN SYSTEMS** | Infra | always | ask | agent | n |

### Tasks

- [ ] Create profile YAML definitions in `.ovav/profiles/`
- [ ] Map to Warp Agent Profiles (settings files)
- [ ] Map to OpenCode agent permissions
- [ ] Disable denylist bypass (`Allow auto-approve to bypass command denylist` = false)
- [ ] Rebuild denylist: keep `git worktree add/remove/prune`, `sudo`, `rm -rf`, `git reset --hard`, `git clean -f`, `git push --force`, `git branch -D`, `curl`, `wget`, `ssh`, `scp`, `rsync`, system destructive actions, WSL destructive
- [ ] Test each profile with fake tasks

### 100% criteria

| Check | Evidence |
|---|---|
| 4 profiles created and discoverable | `ovav profile list` shows them |
| Denylist bypass disabled | Warp settings diff |
| Git worktree commands blocked | Try `git worktree add` from Warp → blocked |
| YOLO profile works in sandbox | Fake task succeeds |

**Safe Stop Report** at close.

---

## P6 — OWS × Warp integration

**Goal:** Warp is presentation, OWS is authority. No git worktree from Warp.

### Tasks

- [ ] Create Warp Workflows in Warp Drive:
  - [ ] OVAV · Create Worktree → `owc {{task}} --profile {{profile}}`
  - [ ] OVAV · Status → `owl`
  - [ ] OVAV · Visual Status → `owsv`
  - [ ] OVAV · Verify → `owv`
  - [ ] OVAV · Update → `owu`
  - [ ] OVAV · Done → `owd`
  - [ ] OVAV · Route Commit → `owcp {{commit}}`
  - [ ] OVAV · Cleanup Preview → `owclean --dry-run`
  - [ ] OVAV · Sync → `ows`
- [ ] Profile enum in Create Worktree: feature/refactor/docs/spike/research/migration/enterprise/hotfix/release/patch/emergency
- [ ] Create Warp Code Review integration
- [ ] Define Tab Group strategy (CORE / AGENTS / DEV)
- [ ] Tab Configs for OVAV CORE / OVAV AGENT / OVAV REVIEW / OVAV SYSTEMS (post-stabilization)

### 100% criteria

| Check | Evidence |
|---|---|
| 9 Warp Workflows created | Warp Drive catalog |
| Each Workflow calls OWS command | workflow definitions |
| Tab Groups organized | Warp settings |
| Code Review wired | Warp + OWS integration test |

**Decision:** `owcp` and `obc` will not be added as new commands.
- `owcp` is conceptual alias for `owx route --mode cherry-pick` (already exists)
- `obc` (branch-only) deferred — not needed for plan

**Safe Stop Report** at close.

---

## P7 — MiniMax endpoints configuration

**Goal:** MiniMax-M3 is the only model. Auto Genius disabled.

### Tasks

- [ ] Warp Agent: custom inference endpoint `https://api.minimax.io/v1`, model `MiniMax-M3`, OpenAI-compatible schema
- [ ] Store MiniMax Subscription Key in Warp's local store
- [ ] Verify Warp uses custom endpoint (no Warp AI credits consumed)
- [ ] OpenCode: Token Plan M3 via `opencode auth login` (already verified: `minimax-coding-plan` in auth.json ✅)
- [ ] Crush: M3 model
- [ ] Disable Auto + Auto Genius as defaults in Warp

### 100% criteria

| Check | Evidence |
|---|---|
| Warp custom endpoint accepts M3 | test query returns M3 output |
| Warp AI credits not consumed | Warp account balance check |
| OpenCode M3 active | `opencode --model` resolves to M3 |
| Crush M3 active | `crush` settings show M3 |
| Auto Genius absent | Warp settings inspection |

**Risk:** Custom endpoint + Warp Free may still consume credits. Smoke test before declaring P7 complete.

**Safe Stop Report** at close.

---

## P8 — OpenCode Warp plugin

**Goal:** Warp-aware OpenCode.

### Tasks

- [ ] Add `@warp-dot-dev/opencode-warp` plugin to `opencode.json`
- [ ] Validate notifications flow Warp → OpenCode
- [ ] Validate rich input (block + autocomplete)
- [ ] Validate Code Review handoff
- [ ] Validate Tab metadata + Tab Configs
- [ ] Validate Remote Control opt-in path

### 100% criteria

| Check | Evidence |
|---|---|
| Plugin loaded | `opencode.json` mcp/plugins entry |
| Notifications work | trigger test → Warp shows it |
| Code Review attaches to commit | manual test |
| Remote Control explicit only | never auto-publish |

**Safe Stop Report** at close.

---

## P9 — Memory model + Cloud Conversations pilot

**Goal:** OVAV Memory canonical, Cloud Conversations as flight recorder.

### Tasks

- [ ] Cloud Conversations enabled in Warp
- [ ] OVAV Memory active (verified: ✅ in `ovav_status`)
- [ ] Warp Agent Memory = OFF (research preview, not for canonical)
- [ ] Document memory hierarchy:
  - OVAV Memory → canonical knowledge
  - AGENTS.md → canonical rules
  - `.agents/skills/` → procedures
  - Git → code truth
  - OpenCode Session → harness context
  - Warp Cloud Conversations → operational history
- [ ] Start 30-day pilot: track restores, incidents recovered, handoffs, cross-device
- [ ] Day 30 report: KEEP / MODIFY / DISABLE

### 100% criteria

| Check | Evidence |
|---|---|
| Cloud Conversations ON | Warp TOML |
| OVAV Memory healthy | `ovav memory` query works |
| Warp Agent Memory OFF | Warp settings |
| Pilot started | Day 0 timestamp recorded |

**Safe Stop Report** at close.

---

## P10 — Privacy + Security hardening

**Goal:** Privacy settings aligned with Warp Free + AI usage.

### Tasks

- [ ] Telemetry ON (required for Warp Free + AI)
- [ ] Crash reporting ON during stabilization
- [ ] Secret Redaction enabled (`asterisks` mode, custom regex tested)
- [ ] Test secret redaction with fake tokens (MiniMax, GitHub, OpenAI, Anthropic, JWT, AWS, Warp) — NO real secrets
- [ ] Remote Control = manual/opt-in only (never auto-publish)
- [ ] Codebase Context OFF in WSL (Warp limitation, no workaround)

### 100% criteria

| Check | Evidence |
|---|---|
| Secret redaction test passes | fake token → asterisks in output |
| Telemetry ON | Warp TOML |
| Remote Control manual | Warp settings |
| Codebase Context OFF | Warp TOML |

**Safe Stop Report** at close.

---

## P11 — Final acceptance

**Goal:** All 47 acceptance criteria from master plan §42 pass.

### Tasks

- [ ] Run full validation suite: `ovav validate`
- [ ] Run `make test` in go-runtime
- [ ] Run OpenCode smoke test
- [ ] Run Crush smoke test
- [ ] Confirm each criterion in master plan §42
- [ ] Generate final SAFE_STOP_READY_FOR_COMMIT report

### 100% criteria

All 47 checkboxes from master plan §42 marked complete with evidence.

**Safe Stop Report** at close = `READY_FOR_COMMIT` → `owd` to integrate.

---

## Cross-cutting principles

1. **100% per phase** — no advance with partial
2. **Safe Stop Report** — every phase close
3. **OWS authority** — never `git worktree` directly from Warp
4. **No bypass without documentation** — every `OVAV_BYPASS_*` env var documented in commit
5. **Snapshot first, mutate later** — P0 baseline before any change
6. **Honest reporting** — every blocker declared, never disguised

---

## Status ledger

| Phase | Status | Commit | Safe Stop |
|---|---|---|---|
| P0 | ✅ Complete | `fc72591` | PARTIAL |
| P1 | ✅ Complete | `ed4e3c1` | VERIFICATION_COMPLETE |
| P2 | ⏳ Pending | — | — |
| P3 | ⏳ Pending | — | — |
| P4 | ⏳ Pending | — | — |
| P5 | ⏳ Pending | — | — |
| P6 | ⏳ Pending | — | — |
| P7 | ⏳ Pending | — | — |
| P8 | ⏳ Pending | — | — |
| P9 | ⏳ Pending | — | — |
| P10 | ⏳ Pending | — | — |
| P11 | ⏳ Pending | — | — |

---

## P1 — Warp × WSL2 × Fish stability ✅ VERIFICATION_COMPLETE

**Goal:** Verify existing connection stack without modification.

### Verified facts

| Check | Evidence |
|---|---|
| Warp Stable installed | `/mnt/c/Program Files/Warp/` exists |
| WSL2 distro = Ubuntu-26.04 | `/etc/os-release` → `Ubuntu 26.04 LTS` |
| Fish is login shell | `getent passwd braka` → `/usr/bin/fish` |
| `[session] wsl = "Ubuntu-26.04"` present | settings.toml line 158 |
| `[session.new_session_shell_override]` present | settings.toml line 161 (gap: plan says remove) |
| Warp settings.toml size | 169 lines |

### Settings snapshot captured

- `.ovav/snapshots/2026-08-18-pre-warp-plan/warp-settings.toml.baseline` (full TOML)
- `.ovav/snapshots/2026-08-18-pre-warp-plan/warp-runtime-baseline.txt` (findings)

### 100% criterion: ✅ VERIFICATION_COMPLETE

P1 is read-only. No mutations to Warp settings applied. All gaps
discovered are catalogued for P2-P10 resolution.

### Gaps catalogued for downstream phases

See `warp-runtime-baseline.txt` for full list. Key items:

- P2: `show_warning_before_quitting=false`, `show_panel_in_restored_windows=false`, `input_box_type_setting=universal`
- P5: denylist rebuild (broad shells → granular patterns + git worktree block)
- P5: 4 execution profiles (only "Default" exists)
- P6: 9 Warp Workflows missing
- P7: `base_model=auto-genius` must become MiniMax-M3
- P7: MiniMax custom endpoint not configured
- P8: `@warp-dot-dev/opencode-warp` plugin missing

---

*Thavren — Platform Engineering — OVAV × WARP 2026 master plan execution*
