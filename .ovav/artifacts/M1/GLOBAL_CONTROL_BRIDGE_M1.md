# M1 — Global Control Bridge

**Milestone:** M1
**Status:** prepared source-local
**Authority:** Final Launch Verification on top of B23 Tool Readiness Matrix.

## Outcome

M1 adds a source-local Global Control Bridge for global-intent triage. It can
classify global-control requests, emit deterministic decisions and write
repo-local evidence. It does **not** perform global writes.

## Bridge Contract

- Source-local decision and evidence only.
- No global config writes.
- No user HOME config/local-state writes.
- No OpenCode global configuration or plugin installation.
- No real install/apply/backup/rollback behavior.
- No external service, UI/TUI, MCP/A2A or live Engram behavior.
- Consent does not override blocked surfaces.

## M1 Decisions

- `opencode_global_config`, `global_config_writes`, `plugin_installation`,
  `package_manager_installs`, live Engram, UI/TUI, MCP/A2A, external services
  and production/global-ready claims are blocked.
- `install_apply_deploy` is plan-only for `inspect`, `plan`, `dry_run`,
  `validate` and `smoke` actions.
- Mutating global actions such as `apply`, `install`, `write`, `deploy`,
  `backup`, `restore`, `publish` and `configure` remain blocked.
- Repo-local artifacts, validators and harness checks are allowed only after
  the workspace safety gate passes.

## Files

- `tools/harnesses/global_control_bridge_m1.py`
- `tools/harnesses/check_global_control_bridge_m1.py`
- `tests/evals/test_global_control_bridge_m1.py`
- `.ovav/artifacts/M1/evidence/GLOBAL_CONTROL_BRIDGE_M1_REPORT.json`
- `.ovav/artifacts/M1/evidence/GLOBAL_CONTROL_BRIDGE_M1_CHECK_REPORT.json`

## Validation

- `python3 tools/harnesses/check_global_control_bridge_m1.py`
- `python3 tests/evals/test_global_control_bridge_m1.py`
- `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate`

## Next Safe Step

Use M1 as the bridge for global-intent triage while actual global behavior stays
blocked until a future, explicit install/global segment opens the required gates.
