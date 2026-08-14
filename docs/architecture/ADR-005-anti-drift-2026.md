# ADR-005: 2026 Anti-Drift Architecture Plan

**Date:** 2026-08-14
**Status:** Accepted (CEO approval: full 4-phase plan + IT v0.1.4 workarounds)
**Decider:** Thavren + CEO
**Related:** ADR-001 (Go runtime), ADR-003 (OMARS), ADR-004 (tools registry stabilization)

## Context

### Lessons learned from 3 sessions of drift patches (2026-08-12 to 2026-08-14)

CEO reported persistent "keybindings broken" regression across three consecutive
sessions. Investigation revealed systemic gaps:

| Session | Symptom | Patch applied | Why it failed |
|---------|---------|---------------|---------------|
| 1 | "keybindings rotos" | Fixed fragment (17 broken bindings) | Fragment was correct; **deploy to live was missing** |
| 2 | "sigue igual" | Created deploy script | Script had **broken path resolution** (used develop fragment, not worktree) |
| 3 | "Ctrl+V ^V literal" | Fixed bash readline | **Wrong layer** — IT wasn't processing keybindings |
| 3 | "shift+arrow no selecciona" | Added set-mark bindings | **Wrong UX** — CEO wanted IT visual, not bash mark |

### Root systemic problems identified

1. **Drift between source-of-truth and live state.** Fragment is correct, but
   no auto-deploy. Every fix to fragment required manual deploy that often
   didn't happen or happened with wrong paths.
2. **Validators detect, don't auto-fix.** 74 validators passing ≠ features
   working. CEO had to patch manually each time.
3. **Runtime integrity baseline is gitignored.** Per-worktree fragility. No
   audit trail.
4. **WSL cross-FS write bug.** `mv /tmp → /mnt/c` silently fails (verified).
   Only `python open(LIVE, 'w')` or same-FS `Path.replace()` work.
5. **IT v0.1.4 alpha instability.** Keybindings may be processed incorrectly
   by IT itself, not by our configs.
6. **Documentation drift.** 4 documentation directories (docs/, docs-site/,
   docs/internal/, docs/workstation/) without cross-reference enforcement.
7. **Validator count drift.** Validator count changed 3 times in 3 sessions
   (71 → 72 → 73 → 74), each requiring caps.yaml alignment. Manual.

### Why this needs an ADR, not more patches

After 4 patches that addressed symptoms not root causes, CEO explicitly
called out "estás aislando bugs pero no solucionando" (you are isolating
bugs but not solving). This plan addresses root causes through architectural
changes, not more layer-by-layer patches.

## Decision

Implement 4-phase architectural plan across 2026:

### Phase 1 — Q1 2026: Anti-drift core (kill manual deploy)

**Goal:** Eliminate the "fix → patch → fix again" cycle.

| Deliverable | Spec | Effort |
|-------------|------|--------|
| `ovav deploy run` command | Auto-deploy fragment → live + restart IT | 3 sprints |
| CI gate: drift-detect | Reject merge if fragment ≠ live > N days | 1 sprint |
| Runtime integrity → versioned | Commit baseline to repo (not gitignored) | 2 sprints |
| `ovav drift show` | Visual diff fragment vs live + suggested fix | 1 sprint |
| IT reload integration | Trigger IT restart via Win32 API | 2 sprints |

**Success criteria:**
- Deploy automatizado end-to-end (1 command from worktree → live state applied)
- 0 drift incidents in Q1 (measured by CI gate)
- Runtime integrity validable without manual work

### Phase 2 — Q2 2026: Auto-remediation (kill manual patch)

**Goal:** Validators with `--fix` that auto-correct detected issues.

| Deliverable | Spec | Effort |
|-------------|------|--------|
| `ovav validate --fix` | Apply auto-corrections for issues marked SAFE_FIX | 3 sprints |
| Safe-fix registry | Whitelist of validators with auto-fix allowed | 1 sprint |
| Rollback on auto-fix failure | Snapshot before fix → atomic rollback | 2 sprints |
| Documentation auto-gen | `docs/validator-registry.md` from registry/*.yaml | 1 sprint |

**Success criteria:**
- 80% of detected issues have auto-fix available
- 0 false positives in auto-fix (measured by rollback rate)
- Docs always in sync with validators

### Phase 3 — Q3 2026: Launch hardening (close launch verification)

**Goal:** Close Final Launch Verification per ADR-003 + robust OpenCode smoke.

| Deliverable | Spec | Effort |
|-------------|------|--------|
| Fuzz testing for validators | go-fuzz on JSON/YAML inputs | 2 sprints |
| Chaos testing for deploy pipeline | Kill process mid-deploy, verify rollback | 1 sprint |
| IT upgrade path | Migrate from v0.1.4 alpha → GA when available | 2 sprints (external) |
| OpenCode smoke expansion | Cover all OpenCode entry points | 2 sprints |
| ADR-005 published | This document, finalized | 1 sprint |

**Success criteria:**
- Final smoke evidence captured
- Final tag created
- Launch verification closed → enables GA claims

### Phase 4 — Q4 2026: GA promotion + stabilization

**Goal:** Only if Q1-Q3 succeed. If not, this phase becomes "iterate Phase 1-3".

| Deliverable | Spec | Effort |
|-------------|------|--------|
| GA release tag | v1.0.0 (no production-ready claim before) | 1 sprint |
| Community prep | README, install guide, troubleshooting | 1 sprint |
| Bug bashing week | Triage all known issues | 1 sprint |
| Roadmap 2027 | Inputs from 2026 lessons | 1 sprint |

**Success criteria:**
- GA release
- 0 critical bugs
- Roadmap 2027 signed

### IT v0.1.4 decision: workarounds (not downgrade)

CEO chose to **keep IT v0.1.4** and **add workarounds** (vs downgrade to
Windows Terminal). Rationale:

- IT is OVAV's strategic direction
- Workarounds documented in `workstation/docs/IT_V014_WORKAROUNDS.md`
- Upgrade path to IT GA tracked in Phase 3
- If IT alpha bugs become blockers, Q1 anti-drift core still helps

## Consequences

### Positive

- **Eliminates drift cycle.** Once Phase 1 lands, the "fragment correct but
  live stale" bug class is structurally impossible.
- **Self-healing governance.** Phase 2 `--fix` reduces manual patches.
- **Auditable integrity.** Phase 1 versioned runtime integrity baseline means
  every commit has a record.
- **Launch verification closeable.** Phase 3 directly addresses Final Launch
  Verification blockers.
- **Documentation co-evolves.** Phase 2 auto-gen keeps docs in sync with code.

### Negative

- **Effort: ~22 sprints / 1 year.** Significant commitment.
- **Phase 1 risk:** WSL cross-FS write bug is undocumented. Building
  reliable deploy on top requires careful testing.
- **Phase 2 risk:** Auto-fix could make things worse if not atomic.
  Snapshot + rollback mandatory from day 1.
- **External dependency:** IT v0.1.4 GA release date unknown. Workarounds
  accumulate technical debt.
- **Scope creep risk:** 4 phases × 5 deliverables = 20 work items. Easy to
  lose focus.

### Risks mitigated

| Risk | Mitigation |
|------|-----------|
| Auto-fix breaks things | Mandatory snapshot + atomic rollback in Phase 2 |
| IT v0.1.4 alpha bugs | Phase 3 upgrade path + per-bug workarounds documented |
| Deploy pipeline regressions | Phase 3 chaos testing |
| Scope creep | Quarterly review at end of each phase |
| Validator count drift | Phase 2 auto-gen eliminates manual caps.yaml updates |

## Implementation discipline

To prevent the "4 patches that didn't fix the problem" pattern:

1. **Each phase ships with:** validators + tests + docs + ADR update.
2. **Each deliverable requires:** test coverage ≥ 80% + smoke evidence.
3. **No silent changes:** Every fix commits + docs + caps + sbom.
4. **CEO approval gate:** Each phase end requires explicit CEO sign-off
   before next phase starts.

## References

- **ADR-001:** Go runtime migration (foundation)
- **ADR-003:** OMARS validator monitoring system
- **ADR-004:** Tools registry stabilization (2026-08-14 — first anti-drift work)
- **docs/architecture/IT_V014_WORKAROUNDS.md:** IT v0.1.4 specific workarounds
- **docs/workstation/IT_DEPLOY_PIPELINE.md:** Existing deploy documentation
- **docs/workstation/IT_KEYBINDINGS_CONTRACT.md:** Keybinding contract
- **docs/workstation/BASH_READLINE_CONTRACT.md:** Readline contract

## Change history

| Date | Change | Commit |
|------|--------|--------|
| 2026-08-14 | Initial plan accepted by CEO | This commit |
