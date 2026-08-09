# DECISION BRIEF — Quota Gating & API Key Activation for Cockpit TUI

**Date:** 2026-06-19 02:15 UTC-5
**Confidence:** HIGH (Topic 1: 9/10, Topic 2: 8/10)
**Recommendation strength:** ADOPT (quota patterns) / ADAPT (activation UX)

---

## 🎯 Topic 1: Usage-Based Feature Gating — ADOPT ✅

### Executive Finding
OVAV needs server-side quota tracking for tiered premium features. Midjourney's model (quota pool + graceful fallback) is the best pattern. GitHub's AI Credits model (abstracted consumption unit) is the best billing abstraction. **Both patterns should be adopted.**

### Recommendation: Dual-Mode Quota System

#### Architecture

```
┌──────────────────────────────────────────────────┐
│                  OVAV License Server               │
│                                                    │
│  ┌──────────────────┐   ┌───────────────────────┐ │
│  │  Quota Pool       │   │  Feature Gates         │ │
│  │  (monthly reset)  │   │  (tier → capabilities) │ │
│  │                    │   │                       │ │
│  │  Tier │ ai_resolve │   │  Free → resolve: 30   │ │
│  │  Free │    30      │   │  Pro  → resolve: 300  │ │
│  │  Pro  │   300      │   │  Pro+ → resolve: ∞    │ │
│  │  Pro+ │   ∞        │   │  Biz  → resolve: 3000 │ │
│  │  Biz  │  3000      │   │                       │ │
│  └──────────────────┘   └───────────────────────┘ │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │  Client Sync (HMAC-signed state blob)         │ │
│  │  "quota": {"ai_resolve": {"used": 12, "max":  │ │
│  │   30}, "reset": "2026-07-01T00:00:00Z"}      │ │
│  │  + HMAC signature over full state            │ │
│  └──────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
         │
         │ Cockpit syncs quota state on startup
         │ + periodic (every 60 min) + pre-operation
         ▼
┌──────────────────────────────────────────────────┐
│                 Cockpit TUI (client)               │
│                                                    │
│  Cockpit Dashboard:                                │
│  ┌───────────────────────────────────────────┐   │
│  │ 🟢 ai_resolve: 12/30 used this month      │   │
│  │    Resets in 12 days                      │   │
│  │                                           │   │
│  │ ⚡ Premium: 30 resolves/mo on Free tier   │   │
│  │ [Upgrade to Pro → 300/mo]                 │   │
│  └───────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

### Quota Mechanism (Adopt from Midjourney + GitHub)

| Component | Pattern Source | Implementation |
|-----------|---------------|----------------|
| **Quota unit** | GitHub AI Credits | `ai_credits` — abstract unit (1 credit = 1 ai_resolve call). Different operations cost different credits. |
| **Monthly pool** | Midjourney Fast hours | Per-tier monthly allocation (Free: 30, Pro: 300, Pro+: ∞). Reset on billing date. |
| **Client state** | HMAC-signed blob | Server sends `{used, max, reset, tier}` with HMAC signature. Cockpit verifies locally, displays, enforces. |
| **Graceful degradation** | Midjourney Relax mode | When credits exhausted → reduced-priority mode: operations still work but queued after premium users. Banner: "⚡ On Free tier limit. Upgrade for instant processing." |
| **Overage path** | GitHub Copilot | Pro/Pro+ users: exhaust pool → "Buy 100 more credits for $X?" → purchase → instant refill. |
| **Reset behavior** | OpenAI upgrade-reset | Free → Pro upgrade: pool resets to Pro tier amount instantly. Pro-rated billing. |

### Anti-Tampering Strategy

| Layer | Mechanism | Priority |
|-------|-----------|----------|
| **State integrity** | HMAC signature over `{used, max, reset, tier}`. Cockpit verifies before displaying or enforcing. | 🔴 Critical |
| **Clock binding** | Quota state includes server timestamp. Cockpit rejects state older than 90 minutes (forces sync). | 🟡 High |
| **Operation gating** | Premium operations call `license.VerifyQuota()` before execution. Checks: HMAC valid, not expired, `used < max`. | 🔴 Critical |
| **Increment reporting** | Cockpit reports usage increment to server. Server debits counter. Cockpit stores HMAC-signed confirmation. | 🟡 High |
| **Offline buffer** | Cockpit caches last-known quota state. Allows 5 operations without server contact (optimistic decrement). Re-syncs on next connection. | 🟢 Medium |

### UI Display in Cockpit (Bubble Tea)

```
┌─── OVAV Cockpit ─────────────────────────────────┐
│ Free Tier · 12/30 ai_resolve used · Resets Jul 1 │
│                                                   │
│ ▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░░░ 40% used                 │
│                                                   │
│ 💡 Upgrade to Pro for 300 ai_resolve/month        │
│    and priority processing.                       │
└───────────────────────────────────────────────────┘
```

**When quota at 90%:** ⚠️ amber banner in Dashboard footer.
**When quota exhausted:** 🔴 banner + "operations will queue. [Upgrade]" CTA.
**Post-exhaustion behavior:** Operations complete but with 5-second artificial delay + "processed on Free tier queue" note.

### What to Avoid

| Anti-pattern | Why |
|--------------|-----|
| ❌ Client-side counter without server sync | Trivially bypassed. GitHub, OpenAI, Midjourney all use server-side. |
| ❌ Hard stop on quota exhaustion (Vercel model) | Hostile UX. Kills user's workflow. Midjourney's Relax mode proves fallback works. |
| ❌ Placing feature gate check only on startup | User can leave Cockpit running for weeks. Check on every premium operation. |
| ❌ Hidden quota (no visible counter) | OpenAI's "try again in 5 hours" without visible counter creates anxiety. Show exact numbers. |

---

## 🎯 Topic 2: API Key Activation — ADAPT 🔧

### Executive Finding
OVAV's HMAC-signed key system is architecturally correct (offline verify → machine bind). The gap is the purchase-to-activation UX. GitHub CLI's device flow is the best adaptation for OVAV's architecture.

### Recommendation: 3-Path Activation System

#### Path A: Device Flow (Recommended Default) | 🥇 Rank

```
┌─ Terminal ─────────────────────┐    ┌─ Browser ──────────────────────┐
│                                │    │                                │
│ $ cockpit activate             │    │ https://ovav.dev/activate     │
│                                │    │                                │
│ ┌──────────────────────────┐   │    │ Enter code: XK9M-2PLQ         │
│ │ Open https://ovav.dev/    │   │    │ [Continue]                    │
│ │ activate                  │   │    │         ↓                     │
│ │ Code: XK9M-2PLQ           │   │    │ ✅ Purchase verified           │
│ │                           │   │    │ License key generated         │
│ │ [Waiting for activation]  │   │    │                                │
│ │  ⠋ Polling...             │   │    │ ┌────────────────────────────┐│
│ │                           │   │    │ │ Copy this key to clipboard  ││
│ │ ✅ License activated       │   │    │ │ [Copy to clipboard]        ││
│ │ Free tier · Machine bound │   │    │ │ Or: auto-deliver to Cockpit ││
│ └──────────────────────────┘   │    │ └────────────────────────────┘│
└────────────────────────────────┘    └────────────────────────────────┘
```

**Step by step:**
1. `cockpit activate` → Cockpit displays 8-char device code + URL
2. User visits `ovav.dev/activate` on any device (phone works)
3. Enters code → OVAV verifies purchase (Stripe session)
4. Server generates HMAC-signed license key, returns it
5. Cockpit polls activation endpoint every 3 seconds
6. Cockpit receives key → HMAC verify → PBKDF2 bind → writes `bind.json`
7. ✅ Activated

**Why this wins:**
- Works cross-device (activate on phone while Cockpit runs on server)
- No paste of 300-char key (HMAC key delivered via HTTPS)
- Device code is short (8 chars) and disposable (expires 10 min)
- User never sees the raw license key (security win)
- Same pattern as GitHub CLI, Apple TV, Netflix device activation

#### Path B: Clipboard Auto-Detect (Quick Path) | 🥈 Rank

```
┌─ Browser (same machine) ──┐    ┌─ Cockpit ─────────────────────────┐
│                            │    │                                   │
│ Purchase complete          │    │ cockpit activate --quick         │
│ [Copy license key]         │    │                                   │
│     ↓                      │    │ ┌───────────────────────────────┐│
│ Key in clipboard           │    │ │ License key detected in        ││
│                            │    │ │ clipboard. Activate? [Y/n]     ││
│                            │    │ │                               ││
│                            │    │ │ ✅ HMAC valid                  ││
│                            │    │ │ 🔗 Binding to this machine     ││
│                            │    │ │ Free tier · Expires Jul 2027   ││
│                            │    │ └───────────────────────────────┘│
└────────────────────────────┘    └───────────────────────────────────┘
```

**Step by step:**
1. User purchases on ovav.dev → clicks "Copy license key"
2. User runs `cockpit activate --quick`
3. Cockpit reads system clipboard (via `xclip`/`pbpaste`/PowerShell)
4. Detects license key format (`eyJ...` prefix)
5. Validates HMAC → binds → done

**Friction:** User must still press a button on the website. But Cockpit auto-detects. No manual paste.

#### Path C: Environment Variable (Headless/CI) | 🥉 Rank

```bash
OVAV_LICENSE_KEY="eyJhbGciOiJIUzI1NiIs..." cockpit
# or
cockpit --license-file ./ovav-license.key
```

Same architecture as current design. For automated deployments and CI.

### Security Considerations

| Concern | Mitigation |
|---------|-----------|
| Device code brute-force | 8-char alphanumeric = 36^8 combinations. 10-minute expiry + 5-attempt lockout. |
| License key in clipboard | Auto-clear clipboard after activation. Key is HMAC-signed (can't forge). Machine binding prevents key reuse. |
| HTTPS interception | Device flow uses HTTPS. HMAC signature verified client-side even if TLS compromised. |
| Replay attack | Device code single-use. Expires after successful activation. |
| Offline activation | Path A requires internet (device flow). Path B offline if key already in clipboard. Path C fully offline (env var). |

### Activation UX Ranking for OVAV

| Rank | Path | Friction | User steps | OVAV compatibility |
|------|------|----------|------------|-------------------|
| 🥇 | Device Flow (A) | Very low | 3 (run cmd, enter code, click link) | ✅ HMAC compatible |
| 🥈 | Clipboard detect (B) | Low | 2 (copy key, run cmd) | ✅ HMAC compatible |
| 🥉 | Env var (C) | Medium | 2-3 (set var, run cmd) | ✅ HMAC compatible |
| — | Manual paste (OpenCode model) | High | 5+ | ❌ Avoid — 300-char paste is hostile |

---

## 📋 Concrete Implementation Plan

### Phase 1: Quota Foundation (Sprint Next)

| # | Task | Effort | Priority | Depends on |
|---|------|--------|----------|------------|
| 1.1 | `QuotaState` struct in `internal/license/quota.go` | ~100 LOC | 🔴 Critical | — |
| 1.2 | HMAC sign/verify for quota blobs (`quota.Sign()`, `quota.Verify()`) | ~80 LOC | 🔴 Critical | 1.1 |
| 1.3 | Cockpit quota display component (Bubble Tea progress bar + counter) | ~150 LOC | 🔴 Critical | 1.2 |
| 1.4 | `quota.Consume(operation)` — optimistic decrement + server sync | ~120 LOC | 🟡 High | 1.2 |

### Phase 2: Graceful Degradation (Sprint Next + 1)

| # | Task | Effort | Priority | Depends on |
|---|------|--------|----------|------------|
| 2.1 | Exhaustion mode: artificial delay + queue banner | ~80 LOC | 🟡 High | 1.4 |
| 2.2 | "Upgrade" CTA component with tier comparison | ~100 LOC | 🟡 High | 1.3 |
| 2.3 | 90% threshold warning banner | ~40 LOC | 🟢 Medium | 1.3 |

### Phase 3: Device Flow Activation (Sprint Next + 1)

| # | Task | Effort | Priority | Depends on |
|---|------|--------|----------|------------|
| 3.1 | `cockpit activate` command + device code generation | ~150 LOC | 🔴 Critical | — |
| 3.2 | Polling loop for activation endpoint | ~80 LOC | 🔴 Critical | 3.1 |
| 3.3 | Web activation page (`ovav.dev/activate`) | ~200 LOC | 🔴 Critical | 3.1 |
| 3.4 | Clipboard auto-detect (`activate --quick`) | ~80 LOC | 🟢 Medium | — |

### Phase 4: Polish (Sprint Next + 2)

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 4.1 | Offline quota buffer (5 ops without sync) | ~100 LOC | 🟢 Medium |
| 4.2 | Multi-machine quota sync (same account, different machines) | ~150 LOC | 🔵 Low |
| 4.3 | Usage history view in Cockpit | ~100 LOC | 🔵 Low |

---

## 📊 Summary

| Topic | Verdict | Key Pattern | Implementation Priority |
|-------|---------|-------------|------------------------|
| Quota gating | **ADOPT** Midjourney + GitHub patterns | Monthly credit pool + graceful fallback + HMAC-signed state | Phase 1–2 (critical path to launch) |
| Activation UX | **ADAPT** GitHub device flow + clipboard detect | Device code (8 chars) → polling → auto-deliver key | Phase 3 (activation is revenue gate) |
| Anti-tampering | **ADOPT** server-side quota + HMAC state | Client verifies HMAC before enforcing. Clock binding prevents replay. | Phase 1 |
| Graceful degradation | **ADOPT** Midjourney Relax mode | Slow-but-works fallback. Never hard-stop a user. | Phase 2 |

### 🚫 Anti-Patterns to Avoid

1. **❌ Client-side quota tracking without server sync** — Every major competitor uses server-side. OpenAI, GitHub, Midjourney, Supabase — zero exceptions.
2. **❌ Hard stop on exhaustion** — Vercel Hobby is the poster child for hostile UX. Users in flow should never hit a wall.
3. **❌ Manual license key paste (OpenCode model)** — 300-character base64 paste into a TUI is a conversion killer. Device flow eliminates this.
4. **❌ Hidden quota counters** — OpenAI's "try again later" without visible counter creates anxiety. Show exact remaining count.

### ✅ OVAV's Competitive Advantage
OVAV's HMAC-signed license system is already architecturally superior to OpenCode's plaintext API key. Adding server-side quota + device flow activation places OVAV's activation UX alongside GitHub CLI (ranked #2) — ahead of Supabase CLI (#3) and far ahead of OpenCode (#6).
