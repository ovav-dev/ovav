---
name: work-unit-commits
description: "Plan commits as reviewable work units. Trigger: implementation, commit splitting, keeping tests and docs with code, or avoiding 400+ line PRs."
license: Apache-2.0
metadata:
  author: dante (OVAV) — adapted from gentleman-programming work-unit-commits
  version: "1.0"
---

# Work Unit Commits — OVAV Edition

Plan commits as **reviewable work units**, not by file type. Each commit should be a single deliverable that the repo can absorb independently.

## When to Use

Load this skill when deciding what belongs in each commit or PR.

Use it for:

- Splitting a feature into reviewable work.
- Preparing commits before opening a PR.
- Turning a large change into chained or stacked PRs.
- Keeping reviewer cognitive load healthy.
- Applying changes without accidentally producing a PR above 400 changed lines.

## Critical Rules

| Rule | Requirement |
|------|-------------|
| Commit by work unit | A commit represents a deliverable behavior, fix, migration, or docs unit. |
| Do not commit by file type | Avoid `models`, then `services`, then `tests` if none works alone. |
| Keep tests with code | Tests belong in the same commit as the behavior they verify. |
| Keep docs with the user-visible change | Docs belong with the feature or workflow they explain. |
| Tell a story | A reviewer should understand why each commit exists from its diff and message. |
| Future PR-ready | Each commit should be a candidate chained PR when the change grows. |
| OVAV commit format | Use `type(scope): subject` with conventional commit types: feat, fix, refactor, chore, docs, style, perf, test, build, ci, revert. |
| 400-line guard | If a change approaches 400 changed lines, split into chained PR slices BEFORE implementation, not after. |
| One fix per commit | User mandate (SESSION 53+): "Avancemos 1 por 1". Each commit = ONE focused fix. If a fix doesn't work, `git reset --hard HEAD~1` and try a different layer. |
| Visual fixes require measurement | User mandate (SESSION 57+): before claiming a visual fix is done, run visual-verification-playwright and report measured pixel numbers. No "should be fixed" claims. |

## Work Unit Checklist

Before committing, confirm:

- [ ] The commit has one clear purpose.
- [ ] The repo still makes sense after applying only this commit.
- [ ] Tests or docs for this unit are included when relevant.
- [ ] Rollback is reasonable without reverting unrelated work.
- [ ] The commit message explains the outcome, not the file list.
- [ ] Visual claims in the message are backed by measured pixel widths (not "should be fixed").
- [ ] TypeScript: 0 new errors.
- [ ] Working tree clean post-commit.
- [ ] No `!important` added (use specificity or clean up conflicting rules).
- [ ] No dead CSS or hardcoded styles left behind.

## Split Examples

| Weak split | Better work-unit split |
|------------|------------------------|
| `add models` | `feat(auth): add token validation domain model and tests` |
| `add services` | `feat(auth): wire token validation into login flow` |
| `add tests` | Tests included with each behavior commit |
| `update docs` | Docs included with the user-visible change they explain |
| `fix(CSS): 5 properties` | `fix(input): height + font`, then `fix(input): centered text`, then `fix(input): no X button` (one property per commit) |
| `fix(border): reduce opacity` | `fix(border): remove double ring`, then `fix(border): softer border`, then `fix(border): hover-only shadow` (each anti-pattern fix separate) |

## OVAV-Specific Commit Pattern

For bt-sys-react (Bitel Agent) project, use this commit format:

```
<type>(<scope>): <subject>

<body explaining the outcome, not the file list>

<footer with measured evidence for visual changes>

# --- example ---
fix(SmartLocation): sl-container width matches cascade in both states

The sl-container was shrinking 86px (396→310) when the cascade
completed and turned into display mode. Root cause: .generic-field
parent had min-width:0 and shrink-wrapped to content width.

Fix: applied flex: 0 0 100% + min-width: 100% + align-self: stretch
to .generic-field so it maintains the parent's width.

Measured via Playwright (visual-verification-playwright):
  slContainer: 396 → 396 (Δ 0.00px) ← fix verified
  inner row:   396 → 396 (Δ 0.00px)
  copySection: 420 → 420 (Δ 0.00px)
```

## PR Relationship

Use work-unit commits as the foundation for chained PRs:

1. Build the smallest independent work unit.
2. Include verification for that unit.
3. Commit it with a Conventional Commit message.
4. If the PR approaches 400 changed lines, promote commits or groups of commits into chained PRs.

## Visual Fix Discipline (OVAV-specific extension)

For ANY visual change (CSS, layout, width, height, font, color):

1. Apply the fix in one commit.
2. **BEFORE claiming it's done**, run visual-verification-playwright.
3. Report measured pixel widths in the commit body or PR description.
4. If the diff is non-zero in the suspected elements, the fix didn't work — revert with `git reset --hard HEAD~1` and try a different layer.
5. Never commit a "should be fixed" claim.

## Commands

```bash
# Review the story before committing
git diff --stat
git diff --cached --stat

# Check recent commit style
git log --oneline -5

# Verify TypeScript 0 errors
pnpm typecheck

# Verify visual fix (if visual)
node /tmp/inspect-app3.mjs
```

## When to Reset

- User says "el error persiste" → reset immediately, don't compound more fixes
- User says "no arreglaste nada" → reset and use visual-verification-playwright BEFORE next attempt
- User identifies a class/element that doesn't actually have the bug (e.g., says "es este" pointing to wrapper, but measurement shows wrapper holds width) → measure at all wrapper chain levels, fix at the correct level

## Anti-Patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Bundling "while I'm at it" changes | Hard to revert one without losing others | One commit per change |
| Committing "should be fixed" without proof | User has rejected 5+ commits for this | Use visual-verification-playwright |
| Reverting via `git revert` instead of `git reset --hard HEAD~1` | Leaves the broken state in history | `git reset --hard HEAD~1` keeps history clean |
| Mixing CSS changes with logic changes | Reviewer can't tell what to focus on | Split CSS-only vs logic-only commits |
| Long commit messages with full file lists | Makes git log useless | Subject in 50 chars, body explains why |

## See Also

- `ovav-skill-registry` — for matching this skill to its trigger
- `visual-verification-playwright` — for the visual measurement step
- `ovav-runtime-gates` — for validation before commits
- `ovav-skill-resolver` — for injecting this skill into sub-agent prompts
