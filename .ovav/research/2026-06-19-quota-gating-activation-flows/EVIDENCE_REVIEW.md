# EVIDENCE REVIEW — Quota Gating & Activation Flows

---

## TOPIC 1: Usage-Based Feature Gating (Quota/Trial Premium)

### 1. OpenAI — ChatGPT Free vs Plus

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | Server-side rate counter. Free tier: limited GPT-5.5 messages per 5-hour rolling window. Separate counters for DALL-E (image gen), data analysis, and file uploads. |
| **Where tracked** | Server-side only. Account-scoped counters in OpenAI's backend. Client-side session token authenticates requests; server increments per-request counter, checks threshold, returns HTTP 429 or allows. |
| **Anti-tampering** | Client never owns the counter value. No local state to tamper with. Request authentication via session JWT — forged requests without valid session fail auth before reaching rate limiter. |
| **UI display** | In-conversation banner: "You've reached the free limit for GPT-5.5. You can try again in X hours." Shows "Get Plus" CTA button. For tool-specific limits (DALL-E), separate banner with tool-specific reset time. |
| **Graceful degradation** | Full lockout of rate-limited feature until window resets. No partial access. Message states exact reset time ("tomorrow at X time"). Upgrade instantly resets all counters. |

**Pattern quality:** ⭐⭐⭐⭐ — Server-side is correct. 5-hour window is user-hostile (why not a visible counter?). Upgrade-as-reset is clever monetization.

---

### 2. GitHub Copilot — AI Credits System

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | AI Credits (1 credit = $0.01 USD). Each model interaction consumes tokens → converted to credits at model-specific rates. Plans have base credits + flex allotment. Free: 2000 code completions/month + limited AI credits. |
| **Where tracked** | Server-side exclusively. GitHub's billing backend tracks token consumption per interaction. Credits debited after each interaction. Usage dashboard shows real-time balance. |
| **Anti-tampering** | Fully server-side. Client sends interaction, server processes model call, measures tokens consumed, debits credits. Client never sees credit balance internals. |
| **UI display** | Usage dashboard on GitHub.com (not in IDE). Shows: base credits remaining, flex allotment status, consumption history per model. IDE shows basic Copilot status indicator. |
| **Graceful degradation** | When credits exhausted: premium model access stops. User can purchase more credits (pay-as-you-go). Code completions remain unlimited for all paid tiers. Free tier: completions capped at 2000/month. |

**Pattern quality:** ⭐⭐⭐⭐⭐ — Best-in-class. Credit abstraction decouples pricing from implementation. Base+flex structure allows pricing model evolution. Pay-as-you-go exhaust path respects user choice.

---

### 3. Vercel — Hard Resource Caps

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | Hobby: hard caps on resources (4h CPU, 360 GB-hrs memory, 1M invocations, 5K image transforms, 50K analytics events). Pro: included amounts with on-demand overage billing. Enterprise: custom. |
| **Where tracked** | Server-side at infrastructure level. CPU time, memory, bandwidth measured at Vercel's edge/runtime. No client counter. |
| **Anti-tampering** | Impossible to tamper — resources metered by the infrastructure running the code. |
| **UI display** | Dashboard: usage meters with progress bars. Email alerts at configurable thresholds (70%, 85%, 95%, 100%). `vercel` CLI: `vercel projects ls` shows basic status. |
| **Graceful degradation** | **Hard stop.** Hobby: functions return errors when CPU cap hit. Image optimization returns placeholders. No continued service at reduced quality. Pro: continues serving, bills overage (soft landing). |

**Pattern quality:** ⭐⭐⭐ — Correct enforcement but hostile degradation on Hobby. Good: Pro overage is seamless. Bad: no "you have X remaining" in CLI/deploy flow. For OVAV: hard stop is NOT the right pattern for a TUI tool.

---

### 4. Supabase — Database-Level Quotas

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | Free: 500MB DB, 50K MAU, 5GB egress, 1GB file storage, 2 active projects. Pro: 8GB DB, 100K MAU, 250GB egress, 100GB storage. Overage available on Pro at per-unit pricing. |
| **Where tracked** | Server-side: Postgres-level disk quota (DB size), Auth service (MAU counting), API gateway (egress metering). |
| **Anti-tampering** | Database-level enforcement — Postgres enforces disk quota, not application code. MAU counted by their Auth service on login events. Egress measured at network level. |
| **UI display** | Dashboard: usage bars for each metric (DB size, MAU, egress, storage). Billing alerts. In-product warnings when approaching limits. |
| **Graceful degradation** | Free projects paused after 1 week of inactivity. When DB limit exceeded: inserts fail (disk full error). API rate limits apply on egress. Pro: overage charges apply, service continues. |

**Pattern quality:** ⭐⭐⭐⭐ — Clean separation: infrastructure enforces, dashboard reports. Good: Pro overage = no disruption. Bad: Free DB full = writes fail with cryptic Postgres errors (poor UX for non-DBAs).

---

### 5. Linear — Simple Counter Limits

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | Free: hard cap of 250 issues total, 2 teams. Basic ($10/user): 5 teams, unlimited issues. Business ($16/user): unlimited teams, AI features unlocked. |
| **Where tracked** | Server-side. Issue count is a database constraint. Team count checked on team creation action. |
| **Anti-tampering** | Server-side API validation. Issue creation endpoint checks plan limit before insert. |
| **UI display** | Notification when approaching limit. Upgrade prompts in sidebar and issue creation dialog. "You've used 248 of 250 issues" type messaging. |
| **Graceful degradation** | When 250-issue limit reached: "Create Issue" button becomes disabled with upgrade prompt. Must archive old issues or upgrade to continue. No partial access — hard lock. |

**Pattern quality:** ⭐⭐⭐ — Simple and transparent, but hard lock is frustrating. Good: visible counter with clear threshold. Bad: no grace period, no "archive to create more" automated workflow.

---

### 6. Midjourney — Time-Based GPU Quota

| Dimension | Evidence |
|-----------|----------|
| **Quota mechanism** | Fast GPU hours per billing period. Basic: 3.3h/month, Standard: 15h, Pro: 30h, Mega: 60h. Each generation consumes GPU seconds (varies by resolution, steps, model). Relax mode: unlimited but queued at lower priority. |
| **Where tracked** | Server-side. GPU allocation managed by Midjourney's job queue. Fast hours tracked per account per billing period. |
| **Anti-tampering** | Server-side GPU allocation — jobs run on their infrastructure. Client only sends prompts via Discord or web app. |
| **UI display** | `/info` command in Discord: shows remaining Fast hours, subscription status, renewal date. Web app: account panel shows hours. Notification when Fast hours deplete. |
| **Graceful degradation** | **Best pattern observed.** Standard plan+ users: Fast hours exhausted → automatically falls back to Relax mode (same quality, slower generation). Basic plan: no Relax mode available — must wait for monthly reset or upgrade. Upgrade instantly provides more Fast hours. |

**Pattern quality:** ⭐⭐⭐⭐⭐ — Best graceful degradation in the study. Fallback to slower-but-functional mode preserves value. Users never hit a brick wall. Monetization: upgrade restores speed. This is the ideal model for tiered premium OVAV features.

---

### Topic 1 — Pattern Summary

| Pattern | Example | Quality | OVAV Applicability |
|---------|---------|---------|-------------------|
| Server-side tracking | OpenAI, GitHub, Midjourney | Mandatory | Must implement via license server API |
| Credit abstraction | GitHub Copilot AI Credits | Excellent | Adopt: "OVAV Points" for premium operations |
| Hard resource cap | Vercel, Supabase, Linear | Good for infra, bad for tools | Avoid for TUI — use soft quotas |
| Time-based quota | Midjourney Fast hours | Excellent with fallback | Adopt: monthly ai_resolve pool with graceful exhaustion |
| Counter-based limit | Linear (250 issues) | Simple but rigid | Use for team/project limits, not feature access |
| Fallback mode | Midjourney Relax mode | Best pattern | Adopt: degraded-but-functional mode when quota exhausted |
| Overage billing | GitHub, Supabase, Vercel Pro | Good for paid tiers | Pro/Pro+ tiers: allow overage purchase |

---

## TOPIC 2: API Key Activation Flow — Best UX Patterns

### 1. Vercel CLI — Zero-Friction OAuth

| Step | Action |
|------|--------|
| 1 | User runs `vercel login` in terminal |
| 2 | Browser opens automatically to Vercel auth page |
| 3 | User authenticates (already logged in → instant) |
| 4 | Browser redirects with success message |
| 5 | CLI auto-detects completion — no paste, no code entry |
| 6 | Token stored at `~/.vercel/auth.json` |
| 7 | ✅ Authenticated |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | Nearly zero. Requires browser. Headless/SSH: alternative token flow via `--token`. |
| **Clean factor** | One command. Browser opens automatically. Zero copy-paste. User just clicks "Allow." |
| **Security** | OAuth 2.0 standard flow. Token scoped to Vercel account capabilities. Local storage in JSON file. No OS keychain integration (weaker than GitHub). |
| **OVAV applicability** | Requires OAuth server + web app. Not directly applicable since OVAV uses HMAC-signed keys (no auth server needed). |

---

### 2. GitHub CLI — Polished Dual-Mode Flow

| Step | Action (Browser mode) | Action (Token mode) |
|------|----------------------|---------------------|
| 1 | `gh auth login` | `gh auth login --with-token` |
| 2 | Interactive: select "GitHub.com" | Interactive prompts (skip browser choice) |
| 3 | Select "HTTPS" | Terminal asks: "Paste your authentication token:" |
| 4 | Select "Login with a web browser" | User navigates GitHub → Settings → Tokens → Generate |
| 5 | Browser opens. Code displayed: `ABCD-1234` | Pastes token into terminal (hidden input via `survey` pkg) |
| 6 | User enters code on GitHub device flow page | ✅ Token validated, stored in OS keychain |
| 7 | CLI polls for completion | Done |
| 8 | Token stored in OS keychain (macOS Keychain, Windows Credential Manager, Linux `pass`) | |
| 9 | ✅ Authenticated | |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | Browser mode: manual code entry (4 chars, simple). Interactive prompts (4 questions) can be streamlined with `--hostname`, `--scopes` flags. Token mode: finding token page requires navigating 3 levels of GitHub Settings. |
| **Clean factor** | Highly polished. Browser URL pre-fills code via query param. Progress spinner while polling. Success message with account name. OS keychain integration makes storage transparent. |
| **Security** | Best-in-class. Device flow OAuth (RFC 8628). Token stored in OS secure credential store, not plaintext. Scoped to specific permissions. Multiple account support with `gh auth switch`. |
| **OVAV applicability** | Device flow pattern: Cockpit shows code → user visits OVAV site → enters code → Cockpit polls for license delivery. Works with HMAC keys: server sends key on code verification. |

---

### 3. Supabase CLI — Browser Token + Manual Paste

| Step | Action |
|------|--------|
| 1 | `supabase login` |
| 2 | Browser opens to Supabase dashboard login |
| 3 | User authenticates |
| 4 | Browser displays: "Your access token: sup_xxxx... Copy this token" |
| 5 | User copies token from browser |
| 6 | Pastes back into terminal (hidden input) |
| 7 | ✅ Authenticated. Token stored locally. |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | Manual copy-paste required. Token is long (~50 chars). Context switch: browser → terminal. Error-prone (whitespace, truncation). |
| **Clean factor** | Simpler than GitHub's token flow (no navigating Settings). Less clean than Vercel's OAuth (extra paste step). |
| **Security** | Token-based. Stored locally. Scoped to Management API. No OAuth token exchange. Token visible in browser — screen capture risk. |
| **OVAV applicability** | Closest to OVAV's current model (license key + paste). But OVAV's license key is ~300 chars (HMAC payload) — much longer than Supabase's token. Paste UX degrades with key length. |

---

### 4. OpenCode — Web Purchase + API Key Paste

| Step | Action |
|------|--------|
| 1 | Purchase model access on opencode.ai ($5+) |
| 2 | Login to OpenCode panel |
| 3 | Navigate to API keys section |
| 4 | Generate/copy API key |
| 5 | Open `opencode.jsonc` in editor |
| 6 | Paste API key into config file |
| 7 | Restart OpenCode or reload config |
| 8 | ✅ Model access working |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | 8-step flow with multiple context switches. Manual file editing required. No validation until first API call fails. Long key (~300 chars base64). No `opencode login` command available. |
| **Clean factor** | Low. This is the baseline — what you get without building an activation flow. Functional but feels unpolished for a commercial product. |
| **Security** | Config file plaintext storage (`.jsonc` is readable). No OS keychain. No machine binding (key works on any machine). |
| **OVAV applicability** | **This is the anti-pattern.** OVAV already has a better foundation (HMAC + PBKDF2 bind). The baseline flow should NOT feel like this. |

---

### 5. Warp Terminal — Account Sign-In Activation

| Step | Action (inferred from pricing model + docs) |
|------|--------|
| 1 | Download and install Warp |
| 2 | Launch → prompted to sign in |
| 3 | Sign in with GitHub/Google/email |
| 4 | OAuth flow in browser |
| 5 | Select plan (Free/Pro/Team) |
| 6 | Pro features unlocked per account |
| 7 | ✅ Activated |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | Requires internet for initial activation. Free tier requires sign-in (privacy concern for terminal users). |
| **Clean factor** | Account-based, not key-based. Plan follows account, not machine. Activation is "sign in, done." |
| **Security** | OAuth-based. Account-scoped. Features gated server-side by plan. |
| **OVAV applicability** | Warp's model is cloud-dependent. OVAV's HMAC system is offline-verifiable — a different architectural choice. But the "account → plan → features" server-side gating is relevant for quota tracking. |

---

### 6. Zed Editor — Account + Usage-Based Tiers

| Step | Action |
|------|--------|
| 1 | Download and launch Zed |
| 2 | Sign in with GitHub or email |
| 3 | Select plan: Personal (free, 2000 edit predictions) / Pro ($10, unlimited + $5 tokens) |
| 4 | Features gated by account plan |
| 5 | ✅ Activated |

| Dimension | Assessment |
|-----------|------------|
| **Friction points** | Sign-in required even for free tier. Token-based billing is per-account, not per-machine. |
| **Clean factor** | Account-based model eliminates license key concept entirely. Subscription managed on zed.dev portal. Editor just checks account status. |
| **Security** | Server-side. Plan enforcement at API level. Usage counters server-side. No client-side secrets beyond OAuth token. |
| **OVAV applicability** | Zed's approach (server-side plan + usage tracking) is architecturally different from OVAV's offline HMAC system. The UX pattern of "sign in once, plan follows" is appealing but requires always-online architecture OVAV deliberately avoids. |

---

### Topic 2 — UX Ranking

| Rank | Tool | Score | Key strength | Key weakness |
|------|------|-------|-------------|--------------|
| 🥇 1 | **Vercel CLI** | 9.5/10 | Zero friction. One command, browser click, done. | Requires browser. Headless flow is secondary. |
| 🥈 2 | **GitHub CLI** | 9/10 | Polished device flow. OS keychain. Multiple modes. | 4 interactive prompts. Token mode requires navigating Settings. |
| 🥉 3 | **Supabase CLI** | 7/10 | Simple. Browser + paste. Fast. | Manual paste. Token visible in browser. |
| 4 | Warp | 7/10 | Account-based. Sign in, done. | Requires cloud connection. Privacy concern. |
| 5 | Zed | 7/10 | Clean account-based model. | Always-online. Requires sign-in. |
| 6 | OpenCode | 3/10 | Functional. | Manual file edit. No validation. No activation command. |

---

### Topic 2 — Pattern Summary

| Pattern | Example | Friction | Security | OVAV Fit |
|---------|---------|----------|----------|----------|
| Browser OAuth auto-detect | Vercel CLI | None | OAuth token, local file | Requires auth server — no |
| Device flow + polling | GitHub CLI | Code entry (4 chars) | OAuth + OS keychain | ⭐ Best fit for HMAC model |
| Browser token + paste | Supabase CLI | Copy-paste | Token in plaintext | Works but high friction for 300-char keys |
| Manual config edit | OpenCode | File editing | Config plaintext | ❌ Anti-pattern to avoid |
| Account-based sign-in | Warp, Zed | Sign-in required | Server-side | Requires always-online — conflicts with offline design |
