# SOURCE MAP — Quota Gating & Activation Flows

## Topic 1 Sources

| # | Company | Source | URL | Type | Quality | Notes |
|---|---------|--------|-----|------|---------|-------|
| 1 | OpenAI | ChatGPT Free Tier FAQ (Help Center) | https://help.openai.com/en/articles/9275245 | Official docs | HIGH | Documents quota mechanism, 5h windows, separate tool limits, UI notifications |
| 2 | OpenAI | API Rate Limits (Platform docs) | https://platform.openai.com/docs/guides/rate-limits | Official docs | HIGH | Headers-based rate limiting (x-ratelimit-*), server-side enforcement |
| 3 | GitHub | Copilot subscription plans | https://docs.github.com/en/copilot/about-github-copilot/subscription-plans | Official docs | HIGH | Tier comparison with AI Credits allowance per plan |
| 4 | GitHub | Usage-based billing for individuals | https://docs.github.com/en/copilot/concepts/billing/usage-based-billing-for-individuals | Official docs | HIGH | AI Credits mechanism: base+flex, overage purchase, token→credit conversion |
| 5 | GitHub | Models and pricing for Copilot | https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing | Official docs | MEDIUM | Per-model token pricing (referenced from source #4) |
| 6 | Vercel | Pricing page | https://vercel.com/docs/pricing | Official docs | HIGH | Hobby hard caps, Pro overage pricing, resource-by-resource meters |
| 7 | Vercel | Resource usage calculation | https://vercel.com/docs/pricing/how-does-vercel-calculate-usage-of-resources | Official docs | MEDIUM | Detailed metric definitions (referenced from pricing page) |
| 8 | Supabase | Pricing page | https://supabase.com/pricing | Official docs | HIGH | Plan tiers, compute add-ons, disk storage limits, MAU/eGress quotas |
| 9 | Linear | Pricing page | https://linear.app/pricing | Official page | HIGH | Issue limits, team limits, feature gates per tier |
| 10 | Midjourney | Plan comparison | https://docs.midjourney.com/docs/plans | Official docs | HIGH | Fast GPU hours per plan, Relax mode availability, video resolution gates |
| 11 | Midjourney | Fast vs Relax hours | https://docs.midjourney.com/docs/plans (referenced guide) | Official docs | HIGH | How Fast hours are consumed per generation, Relax mode queuing |

## Topic 2 Sources

| # | Company | Source | URL | Type | Quality | Notes |
|---|---------|--------|-----|------|---------|-------|
| 12 | Vercel | CLI login command | https://vercel.com/docs/cli/login | Official docs | HIGH | OAuth flow, auto-detection, no paste needed |
| 13 | GitHub | gh auth login manual | https://cli.github.com/manual/gh_auth_login | Official docs | HIGH | Device flow, web browser + token paste dual mode |
| 14 | GitHub | Personal access tokens | https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens | Official docs | MEDIUM | Token management, fine-grained vs classic (context for paste flow) |
| 15 | Supabase | CLI getting started | https://supabase.com/docs/guides/cli/getting-started | Official docs | HIGH | `supabase login` flow, token generation + paste |
| 16 | Supabase | CLI managing environments | https://supabase.com/docs/guides/cli/managing-environments | Official docs | HIGH | Login + link flow, project ID usage |
| 17 | OpenCode | Documentation intro | https://opencode.ai/docs/ | Official docs | MEDIUM | General docs, purchase flow inferred from config structure |
| 18 | Warp | Pricing page | https://www.warp.dev/pricing | Official page | MEDIUM | Tier structure, Pro activation implied via account sign-in |
| 19 | Zed | Pricing page | https://zed.dev/pricing | Official page | HIGH | Account-based tiers, 2000 edit prediction cap on free, usage-based on Pro |

## OVAV Internal Sources

| # | Asset | Path | Relevance |
|---|-------|------|-----------|
| I1 | License bind system | `go-runtime/internal/license/bind.go` | Current HMAC+PBKDF2 implementation |
| I2 | Previous research | `.ovav/research/2026-06-18-license-plan-enforcement/DECISION_BRIEF.md` | Prior ADOPT recommendation for license activation |
| I3 | Cockpit TUI | `go-runtime/cmd/cockpit/` | Current dashboard architecture |

## Source Quality Summary

| Quality Tier | Count | Sources |
|--------------|-------|---------|
| HIGH (official docs, primary source) | 14 | #1-6, #8-15, #19 |
| MEDIUM (official but partial/inferred) | 5 | #7, #14, #16-18 |
| LOW (speculative/guessed) | 0 | — |

**Overall confidence:** HIGH (12/19 sources are direct official documentation; 0 speculative sources)
