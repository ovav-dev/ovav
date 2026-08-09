---
name: ovav-skill-resolver
description: "Resolve and inject relevant OVAV skills before sub-agent launches. Trigger: squad delegation, workflow calls, actor runs that read/write/review code."
license: Apache-2.0
metadata:
  author: dante (OVAV)
  version: "1.0"
---

# OVAV Skill Resolver — Universal Protocol

Any agent that **delegates work to sub-agents** MUST follow this protocol to resolve and inject relevant OVAV skills. This applies to the squad delegation (ovav-squad-delegation), workflow() calls, and any future agent that launches sub-agents.

## Why This Exists

Sub-agents are born with NO context about what skills exist. Without skill injection, a squad reviewing a React project won't know React 19 patterns, a fix agent won't follow the project's responsive design tokens, and a measurement agent won't run visual-verification-playwright before claiming a fix.

## When to Apply

Before EVERY sub-agent launch that involves **reading, writing, or reviewing code**. Skip only for purely mechanical delegations (e.g., "run this test command").

## The Protocol

### Step 0: Determine Auth State (before skill resolution)

Before any skill resolution, determine the current auth state from session context:

- **LOGIN**: Authenticated session — full autonomous permission
- **NO_LOGIN**: Unauthenticated/restricted — verification-only mode

The auth state comes from the session_greeting chronos block and session credentials. If auth state cannot be determined, default to NO_LOGIN (restricted).

### Step 1: Obtain the Skill Registry (once per session)

The `ovav-skill-registry` skill contains a **Compact Rules** section with pre-digested rules per skill (5-15 lines each). This is what you inject — NOT full SKILL.md paths.

Resolution order:
1. Already cached from earlier in this session? → use cache
2. Load skill `/home/braka/.mimocode/skills/ovav-skill-registry/SKILL.md` directly
3. No registry found? → proceed without skills (but warn the user: "No OVAV skill registry found — sub-agents will work without project-specific standards. Run `skill-registry` to fix this.")

### Step 2: Inject Permission Directives (MUST precede all other skill injection)

Load `ovav-agent-permission-injector` skill and inject auth-state-aware permission directives into the sub-agent prompt:

- **LOGIN mode**: Full autonomous permission — agents may execute write/commit/merge/push without manual gate
- **NO_LOGIN mode**: Restricted verification mode — propose → verify → execute for high-impact operations

The permission directives are prepended FIRST before identity/security/contextual skills.

### Step 3: Match Relevant Skills

Match skills on TWO dimensions:

**A. Code Context** — what files will the sub-agent touch or review?

Map file patterns to OVAV skills (common examples — always defer to the registry's Trigger field as the source of truth):
- `.tsx`, `.jsx` → visual-verification-playwright
- `.ts`, `.tsx` (type errors) → ovav-runtime-gates
- `tokens.css`, design system files → ovav-ux-session + visual-verification-playwright
- `vite.config.*`, build setup → ovav-repo-local-work-loop + ovav-runtime-gates
- `*.test.*`, `*.spec.*`, playwright scripts → visual-verification-playwright
- `.claude/`, `.mimocode/` → ovav-platform-session
- Changes to scope/identity/agent files → ovav-identity-guard

**B. Task Context** — what ACTIONS will the sub-agent perform?

| Sub-agent action | Match skills with triggers mentioning... |
|-----------------|------------------------------------------|
| Create a PR | "PR", "pull request" |
| Write/review code | The specific framework/language |
| Make a visual claim | visual-verification-playwright (ALWAYS before claim) |
| Handoff between agents | ovav-artifact-flow + ovav-session-continuity |
| Create work units | (new) work-unit-commits |
| Spawn sub-agents | ovav-squad-delegation + ovav-agent-router |
| Apply governance/security rules | ovav-security-gates + ovav-runtime-gates |

### Step 4: Inject into Sub-Agent Prompt

From the registry's **Compact Rules** section, copy the matching skill blocks directly into the sub-agent's prompt:

```
## OVAV Standards (auto-resolved)

{paste compact rules blocks for each matching skill, in precedence order from registry §Precedence Order}

## Working Notes

- Dante's team is exclusive to web/product topics.
- NEVER claim a visual fix is done without measured pixel widths from Playwright (visual-verification-playwright skill).
- All outputs pass the ovav-response-contract Quality Gates before delivery.
```

### Step 5: Track What You Injected

In your response to the orchestrator, log:
```
Skills injected into <sub-agent-name>: [skill1, skill2, ...]
- Trigger reason for each
- Which file paths the sub-agent will touch
```

## Hard Rules

- **Identity is sacred**: `ovav-identity-guard` MUST be injected for any agent that exposes user-facing identity (LEAD name, service area). Never let an agent break character.
- **Security is non-negotiable**: `ovav-security-gates` MUST be injected for any sub-agent that touches credentials, secrets, network endpoints, or production systems.
- **Visual claims require proof**: `visual-verification-playwright` MUST be injected for any sub-agent that will claim CSS widths/heights/fonts/colors. The sub-agent must report measured pixel numbers in its output, not "should be fixed".
- **No defaults to `actor.run` mimocode agents for web/product work**: Use Dante's squad (Sergio, Elena-frontend, Uriel-devops, Nora) via workflow+agent instead. Route other domains to their LEADs.

## Examples

### Example 1: Visual fix for SmartLocation

User: "the location dropdown text is too small, fix it".

You launch sub-agent:
- Match: `.css` file in `CaseManager/partials/` → visual-verification-playwright
- Match: user-facing visual symptom → visual-verification-playwright (must measure)
- Match: `var(--font-)` references → ovav-ux-session (design tokens)

Inject into prompt:
```
## OVAV Standards (auto-resolved)

[visual-verification-playwright compact rules]
[ovav-ux-session compact rules]

## Working Notes

- You're modifying SmartLocation.css in /home/braka/Work/web/products/worktrees/bt-sys-react-design/
- Braka rejects "should be fixed" claims. Measure with Playwright before declaring done.
- Use the login credentials from visual-verification-playwright skill.
- Report measured pixel widths for the font change in the final output.
```

### Example 2: Brand new feature on a web app

User: "add a dark mode toggle to the sidebar".

You launch sub-agent:
- Match: front-end feature → visual-verification-playwright + ovav-ux-session
- Match: `theme.css`, design tokens → ovav-ux-session
- Match: scoped component changes → ovav-identity-guard (no breaking color identity)

Inject into prompt:
```
## OVAV Standards (auto-resolved)

[ovav-identity-guard compact rules - leadership is Dante]
[visual-verification-playwright compact rules]
[ovav-ux-session compact rules]

## Working Notes

- You're adding dark mode toggle in src/Sidebar/
- Maintain existing design tokens (--bg, --card-glass, etc.).
- After changes, use Playwright to verify contrast ratios meet WCAG 2.1 AA (auto-resolved via ovav-ux-session).
```

## Anti-Patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| "You're an expert, just do it" with no skill injection | Sub-agent invents its own conventions, deviates from project | Always inject at least `ovav-runtime-gates` and `ovav-identity-guard` |
| Injecting all 20 skills | Sub-agent wastes tokens reading irrelevant rules | Use registry's `Trigger` field to match precisely |
| "should be fixed" without pixel numbers | User has rejected 5+ commits for this | Use `visual-verification-playwright` skill |
| Skipping skills on "obvious" tasks | Even simple CSS changes need anti-shrink pattern | Default minimum: identity-guard + response-contract |

## See Also

- `ovav-skill-registry` — the master catalog
- `ovav-squad-delegation` — for delegating to Dante's squad
- `visual-verification-playwright` — for measuring visual claims
- `ovav-runtime-gates` — for validation gates before commits
