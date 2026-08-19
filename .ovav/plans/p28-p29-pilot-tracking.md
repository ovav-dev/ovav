# P28 + P29 — 30-day Pilot Tracking

## P28 — Cloud Conversations Pilot (30 days)

Per plan §28, 30-day measurement period.

### Metrics to track

| Metric | Description | Storage |
|---|---|---|
| `restores_useful` | Times Cloud Conv restored chat when needed | Warp Drive |
| `incidents_recovered` | Times Conv replay helped debug | Warp Drive |
| `handoffs` | Cloud Conv used to hand off work | Warp Drive |
| `historical_searches` | Searches in Conv history | Warp Drive |
| `cross_device_continuations` | Same Conv resumed on different device | Warp Drive |
| `ovav_memory_duplication` | Times Conv captured decisions that should have been in OVAV Memory | Audit log |

### Day 30 decision tree

```
IF restores_useful > 5x/week → KEEP
IF historical_searches > 10/week → KEEP
IF ovav_memory_duplication > 0 → MODIFY (off for decisions, on for chat)
IF cross_device_continuations = 0 → DISABLE (no cross-device use)
```

## P29 — Warp Agent Memory (research preview)

Per plan §29, **DECISION: NOT INTEGRATED.**

### Rationale

- Warp Agent Memory = research preview
- Competes conceptually with OVAV Memory
- Third-party harness support limited
- Can request access ONLY for isolated evaluation

### Status

✅ Warp Agent Memory = OFF (no flag in settings.toml)
✅ OVAV Memory = ACTIVE (per `ovav status`)
✅ Cloud Conversations = ON (per P9)
✅ No integration of Warp Agent Memory in P29 cycle

### Future state

- Monitor Warp's third-party support roadmap
- If support expands and auditability increases, re-evaluate in 2027
- Until then, OVAV Memory remains canonical

## Files

- `p28-cloud-conv-pilot.md` (30-day tracking)
- `p29-warp-agent-memory.md` (decision log)

## Status

✅ P28 + P29 100% — documented and tracked.
