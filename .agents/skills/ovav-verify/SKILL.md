---
name: ovav-verify
description: Run full OVAV validation pipeline (tests, lint, security).
trigger: owv, verify, validate, tests, lint
---

# ovav-verify

Runs the full OVAV validator pipeline: secrets, lint, tests, security, integrity.

## When to use

- Before every merge to develop
- As part of `owd` automatically
- After any change to validation-protected surfaces

## Workflow

```bash
# From inside worktree
ovav worktree owv

# Detailed
ovav validate

# Specific package tests
ovav worktree owv --target go-runtime
```

## Pipeline gates

| Gate | Tool | Required |
|---|---|---|
| Secrets | `check_secrets_hygiene` | yes |
| Forbidden | wordlist scan | yes |
| Tests | `go test ./...` | yes |
| Lint | `golangci-lint` | yes |
| Security | OWASP scan | yes |
| Integrity | `check_living_integrity` | yes |
| GPG | signature check | yes |
| Reviewer | pre-commit review | yes |

## Rules

- Any failure blocks merge
- Cannot bypass except with `--ceowaiver` (CEO only)
- Results cached for 5 minutes
- Re-run after fixes

## Expected output

```
✓ secrets_hygiene
✓ forbidden_words
✓ tests (243 PASS, 0 FAIL)
✓ lint (0 issues)
✓ security (0 critical)
✓ integrity (100%)
✓ gpg (signed)
✓ reviewer (approved)
VALIDATION PASSED
```

## Failure modes

- "tests failed" → `ovav worktree ows` to sync, then `go test -v`
- "integrity <100%" → `ovav integrity baseline --write`
- "GPG missing" → re-sign commits
