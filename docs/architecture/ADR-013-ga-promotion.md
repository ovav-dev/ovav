# ADR-013: GA Promotion Strategy

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 4), ADR-003 (launch verification)
**Decider:** Thavren + CEO

## Context

Phase 1-3 are complete. System is in `launch_verification_blocked` state.
Per ADR-003, before claiming production-ready, we need:
1. Final smoke evidence captured
2. Final tag created
3. Launch verification closed
4. CEO waiver on "production ready" claims

## Decision

GA promotion follows a 4-step ceremony:

### Step 1 — Evidence Collection
```bash
# Run comprehensive smoke
./bin/ovav smoke-all > .ovav/registry/launch_evidence/smoke-all.txt

# Capture validator state
./bin/ovav validate --json > .ovav/registry/launch_evidence/validate.json

# Verify no drift
./bin/ovav ci drift-check
```

### Step 2 — Final Tag
```bash
git tag -a v1.0.0-rc.1 -m "OVAV v1.0.0 Release Candidate 1"
git push origin v1.0.0-rc.1
```

### Step 3 — Launch Verification Close
Requires CEO waiver (`--ceo-waiver` flag) to:
- Update `.ovav/plan/caps.yaml` to remove `launch_verification_blocked`
- Update posture from `source-local launch candidate` to `production ready`
- Add tag hash to canonical authority

### Step 4 — Community Prep
- README + installation guide
- Troubleshooting playbook
- Contribution guide

## Consequences

### Positive
- **Structured ceremony** — no surprise claims
- **CEO gate** — final decision stays with human
- **Evidence trail** — every claim backed by artifact

### Negative
- **Manual ceremony** — not auto-executable
- **CEO dependency** — blocks until human approves

## References

- ADR-003 launch verification
- ADR-005 Phase 4 deliverables
- OVAV invariant: CEO decisions stay human
