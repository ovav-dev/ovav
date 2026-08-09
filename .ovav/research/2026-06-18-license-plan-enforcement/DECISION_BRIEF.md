# DECISION BRIEF — License Activation & Plan Enforcement for Cockpit TUI

**Date:** 2026-06-18 23:54 UTC-5
**Confidence:** HIGH (Topic 1: 8/10, Topic 2: 7/10)
**Recommendation strength:** ADOPT (Topic 1) / ADAPT (Topic 2)

---

## Topic 1: License Activation — ADOPT ✅

### Finding: OVAV's existing system is architecturally correct

The `internal/license/bind.go` implementation (HMAC-signed keys + PBKDF2 machine binding + stdlib-only crypto) matches industry best practice. No architectural change needed. The gap is **activation UX** — the flow from "purchased on website" to "working in Cockpit."

### Recommendation: Implement 2-flow activation

#### Flow A — Interactive (TUI-first)

```
Website: Purchase → Show license key + "Copy to clipboard"
           ↓
Cockpit:  "Welcome — Enter License" screen
           User pastes key (or Cockpit reads clipboard)
           DecodeLicenseKey() → HMAC verify
           Bind() → PBKDF2(key, machine_id) → vault key
           Store: ~/.config/ovav/license/bind.json (vault_hash, machine_id, expiry)
           ✅ Activated
```

**Implementation:**
- New Bubble Tea model: `LicenseActivateModel` with text input (use `bubbles/textarea`)
- Paste detection: read clipboard via `clipboard` Go pkg or `xclip`/`pbpaste`
- On success: write `~/.config/ovav/license/bind.json`, transition to Dashboard
- On failure: show specific error (invalid sig, expired, already bound to other machine)

#### Flow B — Cockpit reads from env/file (headless/CI)

```bash
OVAV_LICENSE_KEY="eyJ..." cockpit
# or
cockpit --license-file ./license.key
```

Same validation chain, no TUI needed.

### Key design decisions

| Decision | Rationale |
|----------|-----------|
| No OAuth server required | HMAC verification is offline. License key IS the proof of purchase. |
| Bind once, verify locally | PBKDF2(key, machine_id) hash stored. Verify on startup without internet. |
| Grace period: 7 days past expiry | Cockpit shows ⚠️ "License expired — X days remaining" banner, continues working. |
| License key format: base64(HMAC payload) | Already implemented. Human-copyable (~300 chars). |
| No remote deactivation | Simpler. Revocation done via expiry. Enterprise: optional revocation list. |

### Anti-tampering: Minimal, pragmatic

| Measure | Priority | Effort |
|---------|----------|--------|
| `-ldflags="-s -w"` in build | HIGH | 1 line in Makefile |
| `garble` obfuscation in release builds | MEDIUM | Add to release pipeline |
| SHA256 self-check at startup | LOW | `go:embed` binary hash |
| Signed binaries (macOS notarization) | LOW | Needed for distribution anyway |

**No binary DRM beyond this.** The HMAC signature already prevents license forgery. Obfuscation makes patching harder but determined attackers will always win. Focus on making honest use easy.

---

## Topic 2: Plan Enforcement — ADAPT 🔧

### Finding: No tool enforces "must follow plan" at git level

Graphite, Linear, and Sapling all use **soft enforcement**: conventions, tooling, and visibility. They don't block commits that deviate. This is intentional — strict git-level enforcement is developer-hostile and breaks offline flow.

### OVAV's situation

OVAV already has the critical pieces:
- **caps.yaml**: canonical plan with `deps`, `order`, `worktree`  
- **OWS (`owc`/`owd`)**: worktree creation and merge workflow
- **Cockpit Dashboard**: plan visualization

The gap: Cockpit shows the plan but OWS doesn't check plan compliance when creating worktrees.

### Recommendation: Add 3 enforcement layers (light → strict)

#### Layer 1: Visibility (IMPLEMENT NOW)

```
owc → before creating worktree:
  Check caps.yaml for next pending cap (lowest order without completed deps)
  Show Cockpit: "Next planned: SEG8 · 'License Activation' · deps: [SEG7 ✅]"
  Ask: "Create worktree for SEG8? [Y/n]"
  
  If user creates non-plan worktree:
  Show Cockpit: "⚠️  Worktree 'fix-urgent-bug' not in plan. Link to plan item? [Select/Unplanned]"
```

**Effort:** ~200 LOC in `go-runtime/internal/ows/`. Cockpit integration via `PlanResolution()` already in `resolve.go`.

#### Layer 2: Branch Naming Convention (IMPLEMENT SOON)

```
Branch format: {plan_id}-{type}-{short_desc}
Example: SEG8-feat-license-activation

Pre-push hook (or owc gate):
  - Extract plan_id from branch name
  - Verify plan_id exists in caps.yaml pending[]
  - Verify deps are completed
  - If check fails: warn (not block) with Cockpit notification
```

**Effort:** ~150 LOC. Hook placed in `.ovav/hooks/pre-push` (checked into repo).

#### Layer 3: Strict Mode (FUTURE — opt-in flag)

```
cockpit --strict-mode
# or OVAV_STRICT_MODE=1

In strict mode:
  - owc REJECTS branch not matching plan_id
  - Pre-commit REJECTS commit without plan reference
  - owd requires conflict prediction PASS before merge
  - Worktree without caps entry: blocked until linked via Cockpit
```

Activated per-project or per-session. Off by default.

### Worktree ↔ Plan binding

```
caps.yaml pending entry:
  SEG8:
    id: SEG8
    name: License Activation Flow
    deps: [SEG7]
    order: 8
    worktree: ""              # ← filled by owc on creation
    commit: ""                # ← filled on merge

owc SEG8-feat-license-activation:
  → Creates worktree at ../feature-SEG8-license-activation
  → Updates caps.yaml: SEG8.worktree = "feature-SEG8-license-activation"
  → Updates Cockpit: pending cap now shows linked worktree
```

This already exists in the caps data model (`PendingCap.Worktree`). Just needs an `owc` update to write it.

### What NOT to do

- ❌ Block commits that don't reference a plan item (kills hotfixes)
- ❌ Require linear task ordering when deps allow parallelism
- ❌ Enforce at merge without human override (some work IS unplanned)
- ❌ Server-side plan validation for local git operations

---

## Implementation Priority

| # | Task | Effort | Impact | When |
|---|------|--------|--------|------|
| 1 | Cockpit License Activation screen (Flow A) | ~300 LOC | 🔴 Critical path | Next sprint |
| 2 | `owc` plan-awareness (Layer 1 visibility) | ~200 LOC | 🟡 High | Next sprint |
| 3 | `-ldflags="-s -w"` in Makefile | 1 line | 🟢 Medium | This week |
| 4 | Branch naming convention hook (Layer 2) | ~150 LOC | 🟢 Medium | Following sprint |
| 5 | Offline grace period (7-day expiry warning) | ~50 LOC | 🟡 High | With #1 |
| 6 | Worktree→caps.yaml auto-link on `owc` | ~80 LOC | 🟢 Medium | With #2 |
| 7 | Strict mode (Layer 3) | ~300 LOC | 🔵 Low | Post-v2 |

---

## Summary

| Topic | Verdict | Key Action |
|-------|---------|------------|
| License activation | **ADOPT** existing HMAC+PBKDF2 system. Add TUI activation screen. | Build `LicenseActivateModel` in Cockpit. |
| Anti-tampering | **ADOPT** minimal: strip symbols + optional obfuscation. | Add `-ldflags="-s -w"` to Makefile. |
| Plan enforcement | **ADAPT** existing caps.yaml+OWS. Add visibility layer, NOT strict blocking. | Make `owc` read caps.yaml before creating worktrees. |
| Plan↔git binding | **ADOPT** branch naming convention + worktree auto-link. | Write worktree name to caps.yaml on `owc` create. |

**The OVAV license system is already best-in-class for Go TUIs. The activation flow is the only missing piece.**
