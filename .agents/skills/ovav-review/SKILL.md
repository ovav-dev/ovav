---
name: ovav-review
description: Code review workflow with Warp Code Review integration.
trigger: review, code review, pr review, warp review
---

# ovav-review

Pre-commit code review workflow. Warp Code Review is the human-facing gate; OWS is the gating authority.

## When to use

- Before every merge to develop
- Before every push to protected branch
- Before any PR creation

## Workflow

```bash
# 1. Local review (in worktree)
git diff develop..HEAD --stat

# 2. Warp Code Review (manual or via Workflow)
# Open Warp → Code Review → attach to branch
# Review comments go back to OpenCode automatically

# 3. Address feedback
git commit --amend --no-edit

# 4. Re-verify
ovav worktree owv

# 5. Merge if approved
ovav worktree owd
```

## Rules

- Code Review is part of the gate — NOT optional
- Comments land back in OpenCode via Warp ↔ OpenCode integration
- OWS decides WHEN to integrate, not code review
- Always review BEFORE `owd`

## Review checklist

| Check | Owner |
|---|---|
| 8-row responses (CRIT-018) | self |
| Tests pass | self |
| Lint clean | self |
| No secrets in diff | `check_secrets_hygiene` |
| Breaking changes documented | reviewer |
| Migration plan if needed | reviewer |

## Failure modes

- Reviewer rejected → address comments, request re-review
- "OWS denied merge" → check `owv` output, fix issues
- "Warp comments not flowing" → check Warp OpenCode plugin
