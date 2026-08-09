# OVAV TESTING & REMEDIATION — Area Boundaries
# Declared: 2026-08-01

## What TESTING & REMEDIATION DOES

- "**Detection:** Coverage gaps, security vulns, code quality, supply chain"
- "**Generation:** Real Go tests that call actual package functions"
- "**Fix application:** Auto-apply patches with backup/rollback"
- "**Verification:** `go test` post-apply to confirm no regressions"
- "**Delegation:** Route findings to correct OVAV AGENTS (kenji/diana/thavren)"
- "**Intercom:** Live event log + JSON tickets for agent consumption"

## What TESTING & REMEDIATION DOES NOT DO

- ❌ Does NOT modify production code without backup
- ❌ Does NOT auto-apply fixes without `applyFix()` confirmation
- ❌ Does NOT replace manual code review
- ❌ Does NOT handle legal/compliance findings (→ legal_compliance)
- ❌ Does NOT handle DevOps/infrastructure (→ devops_infrastructure)
- ❌ Does NOT handle UX/design (→ ux_design)
- ❌ Does NOT generate non-Go code (Python, JS, etc.)
- ❌ Does NOT run in production environments

## Cross-Area Routing

| Finding Type | Routed To | Reason |
|---|---|---|
| SQL/CMD/XXE Injection, Auth, Path Traversal | adversarial_intelligence (kenji) | Offensive security expertise |
| Crypto failures, Hardcoded creds, Misconfigs | security_auditor (diana) | Defensive security expertise |
| Coverage gaps in Go packages | platform_engineering (thavren) | Runtime/go-runtime ownership |
| General vulnerabilities | platform_engineering (thavren) | Default routing |

## Boundary Rules

1. **Coverage contamination:** Test files go to `/tmp/` only — never to package dirs
2. **Backup before fix:** Every `applyFix()` must `os.Rename` original to `.bak`
3. **Verification gate:** `go test` must pass before keeping fix — otherwise rollback
4. **No force-push:** Testing subsystem never pushes to protected branches
5. **No production writes:** Only writes to `/tmp/` and explicit source files with backup

## Intercom Boundaries

- **Input:** Source code analysis, coverprofile data, OWASP probe results
- **Output:** JSON tickets → `/tmp/ovav_fable5_tasks/`, events → `/tmp/ovav_intercom.log`
- **Agents:** Tickets consumed by lead-kenji, lead-diana, lead-thavren

## Status

**BOUNDARIES ACTIVE:** true
**CROSS-AREA ROUTING:** verified
**AUTO-REMEDIATION:** partial (tickets generated; auto-apply pending)
