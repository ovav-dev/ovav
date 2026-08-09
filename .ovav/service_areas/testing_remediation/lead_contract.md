# OVAV TESTING & REMEDIATION — Lead Contract
# Lead: thavren (Platform Engineering)
# Declared: 2026-08-01

## Lead Responsibilities

1. **Maintain the testing engine** — `testing-advance` binary, probes, testgen
2. **Own the remediation loop** — close the cycle: detect → confirm → apply → verify
3. **Route findings correctly** — kenji (injection/path), diana (crypto/creds), thavren (general)
4. **Keep intercom alive** — `/tmp/ovav_intercom.log` must always reflect current state
5. **Ensure coverage quality** — zero-value tests are NOT acceptable; use real function calls

## Service Level Agreement

| Metric | Target | Current |
|---|---|---|
| Security scan time | < 5s | 0.5s ✅ |
| Coverage boost per iteration | > 0.1pp | 0pp ❌ |
| Fix application success | > 95% | partial |
| Intercom latency | < 100ms | real-time ✅ |
| Tickets routed correctly | 100% | 100% ✅ |

## Current State (2026-08-01)

**Testing engine:** OPERATIONAL
- Binary: `/tmp/testing-advance`
- OWASP probes: 22 probes across A01-A10
- Security findings: 87 (27 critical, 59 high, 1 medium)
- Fix tickets: 85 generated

**Remediation loop:** PARTIAL
- `applyFix()` generates JSON tickets ✅
- `applyFix()` does NOT write to source files ❌
- Auto-apply with backup not wired ❌
- Rollback not implemented ❌

**Coverage boost:** DEGRADED
- `testgen.go` generates zero-value tests
- Zero-value tests don't improve coverage
- Real improvement needs mocks + integration tests

## Next Actions (Priority Order)

1. **Wire auto-apply** — `applyFix()` writes real patches to source with `.bak` backup
2. **Verify fixes** — `go test` post-apply, rollback on failure
3. **Close coverage loop** — replace zero-values with smart argument generation
4. **Agent integration** — OVAV AGENTS consume tickets and apply real fixes

## Metrics to Track

- `testing_findings_total` — all vulnerabilities found
- `testing_fixes_applied` — fixes auto-applied
- `testing_coverage_delta` — coverage change per iteration
- `testing_agents_active` — concurrent agent delegations

## Escalation

If testing engine fails → escalate to platform_engineering (thavren)
If security findings require legal action → escalate to legal_compliance (camila)
If findings affect production → escalate to devops_infrastructure (uriel)
