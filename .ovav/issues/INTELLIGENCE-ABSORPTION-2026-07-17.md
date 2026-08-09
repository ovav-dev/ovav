# Intelligence Absorption Report — MiMoCode → OVAV Systems

**Date:** 2026-07-17 23:30 UTC-5
**Lead:** thavren
**Status:** ACTIVE — absorption in progress

---

## 1. Why MiMoCode Beat OpenCode

MiMoCode is OpenCode's fork. The delta that made it superior:

| Dimension | OpenCode (before) | MiMoCode (after) |
|-----------|-------------------|-------------------|
| **Memory** | None | SQLite FTS5, project/session/global, auto-inject, dream/distill |
| **Context** | Manual handoff | Auto-checkpoint, rebuild, compaction with thresholds |
| **Subagents** | None | Actor system with task binding, lifecycle hooks, parallel dispatch |
| **Workflows** | None | JS orchestration with sandbox, convergence, adversarial jury |
| **Self-modification** | None | Tools/hooks/skills/workflows/TUI, all hot-reloaded |
| **Compose** | None | 14-skill spec-to-ship pipeline with review gates |
| **Skills** | None | 17 builtin + 14 compose + user-defined, progressive disclosure |
| **Plugins** | Basic | 13 hook events, tool/session/actor/LLM interception |
| **Permissions** | Basic | Per-tool glob maps, last-match-wins, external_directory control |
| **Model routing** | Manual | Named groups, provider-aware, per-task tier selection |
| **Scheduled work** | None | Cron/loop with keepalive budget, autonomous mode |
| **Verification** | None | Adversarial jury, spec-anchored review, evidence gates |

**Key insight:** MiMoCode's advantage is not raw capability — it's **governance patterns**. Every feature has hard gates, evidence requirements, and rollback paths. This IS what OVAV already does, but MiMoCode does it at the tool/agent level while OVAV does it at the system level.

---

## 2. OVAV's Current Brain (what we have)

### 2.1 Go Runtime (33 packages, 39K+ LOC)
- **validators/** — 78 files, 19K LOC, 79+ governance validators
- **ows/** — 15 files, 12K LOC, SQLite-backed worktree state machine
- **governor/** — 8 files, 3.3K LOC, autonomous governance cycle
- **convert/** — 5 files, 3.5K LOC, 4-runtime agent converter
- **install/** — 13 files, 1.2K LOC, install/apply/rollback pipeline

### 2.2 MiMoCode Integration (already adopted)
- **24 OVAV skills** in `.mimocode/skills/`
- **4 OVAV plugins** using hook API (governance, security, status, monitor)
- **2 OVAV workflows** using JS orchestration
- **11 OVAV agents** (areas only — TAB picker workaround)
- **7 OVAV commands** (status, work, context, verify, refresh-skills, validate, close)

### 2.3 What's Missing (gaps identified)
1. **No adversarial verification pattern** in OVAV's own validators
2. **No spec-anchored review** for OVAV's own code changes
3. **No model groups** for tiered routing
4. **No cron/loop** for periodic governance checks
5. **No checkpoint rebuild** for context window overflow
6. **No snapshot/undo** for filesystem state

---

## 3. Absorption Plan — What to Bring Into OVAV

### Phase A: Immediate Absorption (this session)

#### A1. Adversarial Jury Pattern → `ovav-validators`
**Source:** MiMoCode deep-research workflow (adversarial jury: 3 jurors, 2-of-3 reject kills fact)
**Target:** `internal/validators/` — add adversarial review to critical validators
**Pattern:** Each critical claim gets 3 independent checks. 2-of-3 reject = finding dismissed.
**Impact:** Reduces false positives in governance checks

#### A2. Spec-Anchored Review Gate → `work-unit-commits`
**Source:** MiMoCode compose:subagent (every task maps to spec sections, evidence required)
**Target:** OVAV commit discipline — every commit must map to caps.yaml section
**Pattern:** Commit message includes `Refs: caps.yaml#section-id`. Reviewer checks spec alignment.
**Impact:** Prevents drift between code and plan

#### A3. Model Groups → `permission_authority.json`
**Source:** MiMoCode model_groups config
**Target:** OVAV model routing — define `ovav-ultra`, `ovav-standard`, `ovav-lite`
**Pattern:** Subagent tasks reference groups, not literal model strings
**Impact:** Model switching without config edits

#### A4. Cron/Loop → Governance Monitoring
**Source:** MiMoCode cron tool + loop skill
**Target:** Periodic governance checks (coverage drift, secrets hygiene, branch protection)
**Pattern:** Scheduled prompts fire governance validators on cadence
**Impact:** Automated compliance monitoring

### Phase B: Structural Absorption (next sessions)

#### B1. Checkpoint Rebuild → Context Economy
**Source:** MiMoCode auto-checkpoint + rebuild
**Target:** `internal/memory/` — add context rebuild for long sessions
**Pattern:** Auto-distill conversation at thresholds, rebuild from checkpoints
**Impact:** Prevents context window overflow in long OVAV sessions

#### B2. Snapshot/Undo → Worktree Safety
**Source:** MiMoCode git-based snapshots
**Target:** `internal/ows/` — add snapshot before risky operations
**Pattern:** `git stash` + state snapshot before merge/push, auto-restore on failure
**Impact:** Atomic worktree operations with rollback

#### B3. Progressive Disclosure → Skill Loading
**Source:** MiMoCode skill frontmatter → body → references
**Target:** OVAV skill system — implement 3-tier loading
**Pattern:** Frontmatter (always) → SKILL.md (on match) → references/ (on demand)
**Impact:** Faster skill loading, less context waste

### Phase C: Innovation Absorption (future)

#### C1. Voice Input → Terminal Integration
#### C2. Self-Modification Triggers → Auto-evolve
#### C3. Adversarial Jury → Full Validator Pipeline

---

## 4. Stabilization Plan — "Pulir antes de avanzar"

### Pre-requisites
- ✅ caps.yaml aligned (v78.0, HEAD de7018f)
- ✅ Memory current
- ✅ Git HEAD certified
- ✅ MiMoCode intelligence mapped

### Stabilization Units (SUs)

| SU | What | Effort | Priority |
|----|------|--------|----------|
| **STAB-01** | Update caps.yaml: remove stale product refs, align all timestamps | Done | ✅ |
| **STAB-02** | Audit `.mimocode/` vs `.opencode/` — sync agent definitions | 30min | HIGH |
| **STAB-03** | Absorb A1: adversarial pattern into validators | 1h | HIGH |
| **STAB-04** | Absorb A2: spec-anchored commit messages | 30min | HIGH |
| **STAB-05** | Absorb A3: model groups config | 30min | MEDIUM |
| **STAB-06** | Absorb A4: cron governance monitoring | 1h | MEDIUM |
| **STAB-07** | Sync all 4 runtimes (opencode, mimocode, claude-code, cursor) | 15min | HIGH |
| **STAB-08** | Run full validation suite (F0-F5) | 30min | HIGH |
| **STAB-09** | Update MEMORY.md with absorption discoveries | 15min | HIGH |
| **STAB-10** | Commit stabilization batch | 15min | HIGH |

**Total estimated:** ~5h focused work

### Definition of Done
- [ ] caps.yaml HEAD matches real HEAD (verified, committed)
- [ ] All 4 runtimes regenerated and synced
- [ ] At least 3 MiMoCode patterns absorbed (A1-A4)
- [ ] Full validation suite passes
- [ ] MEMORY.md updated with absorption discoveries
- [ ] No stale references to withdrawn OVAV Product

---

## 5. Environment Context

**Current setup:**
- Windows Terminal + PowerShell 7 (new)
- WSL2 Ubuntu (OVAV lives here)
- MiMoCode CLI in terminal (new)
- OpenCode CLI (legacy, still functional)

**OVAV's role:** Layer ABOVE these CLIs — provides governance, validation, security, and advanced toolchains. Not a replacement, but an enhancement layer.

---

*Report generated by thavren — Platform Engineering Lead*
*OVAV Systems v78.0 — 2026-07-17*
