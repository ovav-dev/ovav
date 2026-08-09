# TASK8 — Delegation: Web Payments, Login, Access & Membership

| Field | Value |
|---|---|
| Handoff ID | TASK8-PAYMENTS-MEMBERSHIPS |
| Delegate To | **Dante** (Digital Product Engineering) + **Sofía** (Commercial & Growth Strategy) |
| Delegated By | Thavren / Platform Engineering |
| Date | 2026-06-16 |
| Priority | P0 — Commercial Launch Blocking |
| Branch | task/tasknext-ceo-task8 |
| Source Caps | caps.yaml L85-153 (SOFIA-PRICING), L130-153 (Enterprise tier), L207-220 (market data) |

---

## 1. Task Specification

Implement the complete web payment, authentication, membership tier access control, and purchase flow for OVAV packages across all surfaces. This is a **greenfield implementation** — no payment or membership infrastructure exists today.

### 1.1 What to Build

| Component | Description |
|---|---|
| **Payment Flow (Stripe)** | Checkout for 3 tiers (Free / Pro $19/mo / Enterprise $49/seat). Stripe Checkout or Elements. Annual discount (2 months free). Webhook handling for Stripe events. |
| **Authentication** | User registration + login. Upgrade existing Google/GitHub OAuth from cPanel-only to full account system. Email/password fallback. |
| **Membership Tier Enforcement** | RBAC based on Stripe subscription status. Surface-level and feature-level gating. Free: 2 profiles. Pro: 8 profiles + unlimited models. Enterprise: SSO + audit logs + SLA. |
| **Purchase Flow** | Landing page (ovav.dev) → Pricing section → Checkout → Account creation/upgrade → cPanel access with tier enforcement. |
| **Account Management** | Profile page: upgrade/downgrade, billing history, cancel subscription. |
| **Enterprise Flow** | Contact form → enterprise@ovav.dev → manual onboarding pipeline. |

### 1.2 Tiers Reference (from `business_model.yaml` v2)

| Tier | Price | Key Includes | Key Limitations |
|---|---|---|---|
| **Free** | $0 | CLI completa, 2 perfiles, modelos comunitarios | 2 perfiles activos, sin SSO, sin audit logs |
| **Pro** | $19/mo ($190/yr) | 8 perfiles, modelos ilimitados, Eidren evidence, Tailor Composer, beta channel | Sin SSO/SAML, sin audit logs, sin SLA |
| **Enterprise** | $49/user/mo (min 10 seats) | Todo Pro + SSO/SAML, audit logs, self-hosting, SLA 99.5%, onboarding guiado | Mínimo 10 seats |

### 1.3 Surfaces Affected

- **ovav.dev** (landing) — Pricing section CTA → checkout flow
- **cpanel.ovav.dev** (dashboard) — Auth gate, tier-aware feature visibility, account page
- **Go runtime cPanel API** — New endpoints for user accounts, subscriptions, tier enforcement
- **Database** — New: users table, subscriptions table, tier-membership mapping

---

## 2. Current State — What Platform Engineering Has Done

### 2.1 cPanel OAuth (Google + GitHub) — ✅ ALREADY BUILT

The cPanel at `cpanel.ovav.dev` already has working OAuth:

- **Backend**: `go-runtime/cmd/cpanel/oauth.go` — Google OAuth2 + GitHub OAuth flow handlers. CSRF state verification (one-time use, 10min TTL). Token exchange against provider APIs. `io.LimitReader(1MB)` on all provider responses.
- **Backend**: `go-runtime/cmd/cpanel/auth.go` — JWT RS256 token management. Session tracking with expiry. Rate limiting (5 attempts/min/IP, X-Forwarded-For aware). Token minimum 32 characters (hardened from 4).
- **Frontend**: `tools/cpanel/src/components/Login.tsx` — OAuth Google/GitHub buttons, token auth fallback, CSRF state generation. Redirect + callback handling.
- **Config**: `config/oauth_config.yaml` — Provider client IDs, redirect URIs (→ cpanel.ovav.dev).

**What you can reuse:**
- The full OAuth provider integration (Google + GitHub) is production-ready.
- JWT session management is battle-tested (107 tests on cPanel, 65.5% coverage, 0 data races).
- The OAuth redirect URLs are already pointed at `cpanel.ovav.dev` and SSL-verified.
- You need to **extend** this from cPanel-only login to a system-wide account identity. The OAuth infrastructure itself does not need to be rebuilt.

### 2.2 cPanel API Surface — ✅ ALREADY BUILT

- **Go runtime** with 17 route handlers (auth, OAuth, profiles, validators, git, memory, agents, security, system, events, SSE, status).
- Full CORS, CSRF, rate limiting, path traversal protection (Phase 3 hardening complete).
- 10 Go packages with tests, 0 data races.
- **Frontend**: React 18 + Vite SPA (`tools/cpanel/`), 10 sections, Toast notifications, ErrorBoundary.

### 2.3 Landing Page — ✅ ALREADY DEPLOYED

- Next.js 14 + Tailwind, static export ~112KB, 6 sections (hero, pricing with 3 tiers, 8 profiles, competitive moat, CTA, footer).
- Deployed on Cloudflare Pages → `ovav.dev`. DNS verified, SSL OK.
- Copy from Sofía's `landing_copy_brief.yaml`.
- The pricing section already displays the 3 tiers visually. The "Start governing" CTA links to `#pricing`.

### 2.4 Business Model — ✅ ALREADY APPROVED

- `business_model.yaml` v2 (610 loc): 3 pricing tiers, pricing justification, market data, GTM strategy.
- `landing_copy_brief.yaml` (245 loc): exact copy for landing page.
- CEO-approved pricing: Free / Pro $19/mo / Enterprise $49/user/mo.
- Competitive positioning document from Eidren (OVAV 96/100 vs 5 competitors × 10 dimensions).

---

## 3. What Is NOT Done (Your Scope)

| Gap | Details |
|---|---|
| **Stripe integration** | No Stripe Checkout, no webhook handling, no subscription tracking |
| **User account database** | No persistent user storage. Current auth is session-based JWT with no user profile. |
| **Membership tier enforcement** | No tier-awareness in cPanel. No feature gating. cPanel shows all sections to all users. |
| **Purchase flow** | No path from landing CTA → checkout → account → cPanel |
| **Account management** | No billing page, no upgrade/downgrade, no cancellation flow |
| **Enterprise contact flow** | `enterprise@ovav.dev` exists but no pipeline/tracking |
| **Email integration** | No transactional emails (receipts, welcome, password reset) |

---

## 4. Architecture Guidance (Non-Binding)

```
User visits ovav.dev
  → Clicks tier CTA (e.g., "Start Pro")
  → Stripe Checkout Session (pricing info from business_model.yaml)
  → Payment succeeds → Stripe webhook → Go backend creates/updates subscription
  → Account created (OAuth Google/GitHub) or linked to existing
  → Redirect to cpanel.ovav.dev with JWT
  → cPanel reads tier from JWT claims → renders allowed sections
  → Account page at cPanel: billing history, upgrade/downgrade, cancel
```

```
Data model (suggested):
  users: id, email, name, avatar_url, oauth_provider, oauth_id, created_at
  subscriptions: id, user_id, stripe_customer_id, stripe_subscription_id,
                tier (free|pro|enterprise), status (active|past_due|canceled|trialing),
                current_period_start, current_period_end, created_at
```

**Key decisions Dante + Sofía own:**
- Stripe product/price IDs and their mapping to OVAV tiers
- Whether to use Stripe Checkout (hosted) or Stripe Elements (embedded)
- Database choice (SQLite via Go, Cloudflare D1, or something else)
- Whether accounts span surfaces or are per-surface
- Email provider (Resend, SendGrid, Cloudflare Email Service)

---

## 5. Acceptance Criteria

1. User can sign up via Google OAuth or GitHub OAuth on ovav.dev
2. User can purchase Pro tier via Stripe checkout from landing page CTA
3. Successful payment → user is redirected to cPanel with Pro-tier access (8 profiles visible)
4. Free tier user sees only 2 profiles in cPanel; Pro sees all 8; Enterprise sees +SSO/audit sections
5. Enterprise inquiry form sends to enterprise@ovav.dev with tracking
6. User can view billing history and cancel subscription from cPanel account page
7. Stripe webhooks handle: checkout completed, subscription updated, subscription canceled, payment failed
8. cPanel enforces tier access server-side (JWT claims + DB lookup, not just frontend hiding)
9. Welcome email sent on first purchase; receipt email on renewal
10. Downgrade flow: Pro → Free preserves account but restricts to Free limits

---

## 6. Context References

| Artifact | Path | Relevant Sections |
|---|---|---|
| Business Model (canonical) | `.ovav/plan/business_model.yaml` | Pricing lines 80-153, all 3 tiers + justification |
| Landing Copy Brief | `.ovav/plan/landing_copy_brief.yaml` | Hero + Pricing section copy |
| Competitive Analysis | `docs/research/competitive_analysis_2026.md` | Pricing comparables (6 tools) |
| cPanel OAuth (Go) | `go-runtime/cmd/cpanel/oauth.go` | Existing OAuth handlers |
| cPanel Auth (Go) | `go-runtime/cmd/cpanel/auth.go` | JWT + session management |
| cPanel Login (TS) | `tools/cpanel/src/components/Login.tsx` | Frontend OAuth UI |
| cPanel API Config | `tools/cpanel/src/config.ts` | API_BASE for endpoints |
| cPanel Types | `tools/cpanel/src/types/index.ts` | TypeScript interfaces |
| OAuth Config | `config/oauth_config.yaml` | Provider client IDs + redirect URIs |
| cPanel Dockerfile | `Dockerfile.cpanel` | Build/deploy config |
| Fly.io config | `fly.toml` | cPanel hosting (DFW, 2 machines) |

---

## 7. Notes for Delegates

- **Dante**: La infraestructura OAuth ya está sólida y testeada (107 tests, 0 data races). Construí sobre eso. El cPanel frontend en React 18 + Vite ya tiene un `Login.tsx` funcional con Google/GitHub — extendelo para registro + linking de cuentas. El pricing ya está en la landing (Next.js 14) y el copy es canónico en `landing_copy_brief.yaml`.
- **Sofía**: Los 3 tiers ya están definidos y CEO-aprobados. Cualquier ajuste de precios o estructura de tiers requiere tu aprobación como Commercial Lead. El reframing a "Professional Development Governance" debe reflejarse en todo el UX de compra.
- **Ambos**: Este es el último bloqueante comercial antes del lanzamiento en Product Hunt (Jul 7). Sin pagos no hay ingresos. Sin membresías no hay diferenciación de producto.
