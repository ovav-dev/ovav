# ADR-012: Chaos Testing for Deploy Pipeline

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 3 / D2), ADR-008 (deploy pipeline)
**Decider:** Thavren + CEO

## Context

The deploy pipeline (ADR-008) handles critical operations:
- Atomic file writes
- Snapshot creation
- Rollback on failure
- Audit logging

Real-world failures include:
- Disk full mid-write
- Permission denied on live path
- Concurrent deploys (race condition)
- Snapshot directory corruption
- Process killed mid-deploy

Phase 3 / D2 requires chaos testing: **deliberately inject failures**
and verify the pipeline handles them gracefully.

## Decision

Add `ovav deploy chaos` command that runs the deploy pipeline against
synthetic failure scenarios:

1. **Disk full** — pre-fill disk, attempt deploy
2. **Permission denied** — chmod live path 0000
3. **Concurrent deploys** — two deploys in parallel
4. **Snapshot corruption** — write garbage to snapshot file
5. **Process kill mid-deploy** — cancel via context

Each scenario:
- Expects specific failure mode (rollback, error, no-op)
- Verifies state invariant: no partial writes, snapshot intact
- Records outcome in chaos_history.jsonl

## Architecture

```
ovav deploy chaos [--scenario=X] [--list]
   │
   ▼
[Setup synthetic failure]
   │
   ▼
[Run deploy pipeline]
   │
   ▼
[Verify outcome matches expectation]
   │
   ▼
[Restore environment]
   │
   ▼
[Log to .ovav/registry/chaos_history.jsonl]
```

## Invariants tested

1. **Atomic write invariant** — live file is never partial
2. **Snapshot invariant** — snapshot file always exists for active deploys
3. **Rollback invariant** — failed deploys leave no traces
4. **Concurrency invariant** — parallel deploys don't corrupt state
5. **Recovery invariant** — system recovers from all failure modes

## References

- ADR-008 deploy pipeline (consumed)
- ADR-011 auto-remediation (related)
