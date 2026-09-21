# Python Test Suite — Broken Imports Post-Refactor

**Status:** Defecto arquitectónico (CRIT-004)
**Detected:** 2026-08-19
**Severity:** HIGH — 137 tests broken, 21 pass, 54 skip

---

## Symptom

```
$ python -m pytest tests/
137 failed, 21 passed, 54 skipped
```

All failures: `ModuleNotFoundError: No module named 'tools.X'`

## Root cause

`tools/` was refactored. Current structure:
```
tools/
├── cpanel/         # React/TS control panel
├── extensions/     # Premium input extension
├── infra/          # Infrastructure scripts
├── mcp/            # MCP server
└── web/            # Web backend
```

**Removed during refactor** (tests still reference):
- `tools.agent_runtime` → tests/evals/*, tests/test_runtime_safety_governor.py
- `tools.pain_scorer` → tests/test_pain_scorer_d1.py
- `tools.decision_brief` → tests/evals/test_decision_brief.py
- `tools.evidence_scoring` → tests/evals/test_evidence_scoring.py
- `tools.observability_engine` → tests/evals/test_observability_engine.py
- `tools.protocol_gate` → tests/evals/test_protocol_gate_*
- `tools.nerve_bus` → tests/test_pain_scorer_d1.py
- ~25 other modules

## Decision (CRIT-004)

This is **architectural defect, not bug**. Per CRIT-004: "3+ repetitions = architectural defect, not patch".

Two options:
1. ❌ Restore old paths (wrong — bloats repo, conflicts with new architecture)
2. ✅ Mark legacy tests as quarantined, document, invest in new suite

**Chosen:** Option 2.

## Action plan

1. **Now (this PR):**
   - Move broken tests to `tests/_legacy_broken_imports/`
   - Add pytest.ini config to skip legacy by default
   - Run only the 21 passing tests in CI
   - Document why each module was removed

2. **Next sprint:**
   - Build new Python test suite aligned with current `tools/` architecture
   - Test cpanel (Vitest already covers TS components)
   - Test infra scripts
   - Test extensions

## Affected files (137 tests)

```
tests/evals/test_benchmark_matrix.py
tests/evals/test_decision_brief.py
tests/evals/test_evidence_scoring.py
tests/evals/test_global_control_bridge_m1.py
tests/evals/test_host_config_drift.py
tests/evals/test_model_policy_validator.py
tests/evals/test_model_task_router.py
tests/evals/test_observability_engine.py
tests/evals/test_protocol_*.py (8 files)
tests/evals/test_source_verification.py
tests/test_runtime_safety_governor.py
tests/test_pain_scorer_d1.py
... and ~115 more in test_pain_scorer_d1.py
```

## Working tests (21 passed)

See `tests/` excluding the broken files above.

---

*Generated 2026-08-19 — Thavren, CRIT-019 + CRIT-004 enforced.*