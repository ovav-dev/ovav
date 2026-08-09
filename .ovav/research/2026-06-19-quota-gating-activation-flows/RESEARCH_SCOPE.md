# RESEARCH SCOPE — Quota Gating & API Key Activation for Cockpit TUI

**Session:** 2026-06-19 02:15 UTC-5
**Lead:** Eidren (Research Intelligence)
**Requestor:** CEO
**Status:** COMPLETE

## Scope

Competitive analysis for OVAV Cockpit TUI (Go + Bubble Tea) covering two topics:

### Topic 1: Usage-Based Feature Gating (Quota/Trial Premium)
How companies implement "lower tier gets limited access to premium features":
- OpenAI: Free vs Plus — GPT-5.5 message quotas
- GitHub Copilot: Free/Pro/Pro+/Max — AI Credits system
- Vercel: Hobby vs Pro — hard resource caps with meter-based overage
- Supabase: Free vs Pro — DB size, MAU, egress quotas
- Linear: Free vs Basic — issue and team count limits
- Midjourney: Basic/Standard/Pro/Mega — Fast GPU hours

### Topic 2: API Key Activation Flow — Best UX Patterns
Analyze the "buy on web → activate in app" flow:
- OpenCode: Web purchase → API key → manual paste
- Vercel CLI: `vercel login` → browser OAuth → auto-detected
- GitHub CLI: `gh auth login` → device flow or token paste
- Supabase CLI: `supabase login` → browser token → paste back
- Warp terminal: license activation via account sign-in
- Zed editor: account-based activation with usage tiers

## OVAV Context
Building a commercial AI workstation governor with Bubble Tea TUI (Cockpit). Existing Go license system uses HMAC-signed keys + PBKDF2 machine binding. Designing tiered subscriptions (Free/Pro/Pro+/Business/Enterprise). Need to allow lower tiers limited access to premium features (e.g., Free tier gets 30 ai_resolve calls/month).

## Existing OVAV Assets Reviewed

| Asset | Path | Relevance |
|-------|------|-----------|
| License bind system | `go-runtime/internal/license/bind.go` | HMAC-signed keys, PBKDF2 machine binding, stdlib-only crypto |
| Cockpit TUI | `go-runtime/cmd/cockpit/` | Bubble Tea dashboard framework |
| Plan data model | `.ovav/plan/caps.yaml` | Feature segmentation model |
| Previous research | `.ovav/research/2026-06-18-license-plan-enforcement/` | License activation flow recommendation (ADOPT HMAC system) |

## Evidence Sources
12 sources consulted across web documentation and official docs (see SOURCE_MAP.md).
Priority: official docs > pricing pages > help centers.

## Out of Scope
- Payment processor integration (Stripe/Paddle)
- License key generation service architecture
- Web checkout UI implementation
- Full billing system design
