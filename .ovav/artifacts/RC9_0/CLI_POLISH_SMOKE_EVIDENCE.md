# RC9.0 CLI polish smoke evidence

Status: PASS for source-local CLI polish checks. This is not a production/global-ready claim.

## Scope

- Baseline commit resumed from: `1328778 fix(cli): restore plan route contracts`
- Branch: `task/cli-living-experience-rc9`
- Changed surfaces:
  - `VERSION`
  - `.ovav/tasks/cli-living-experience/WORK_TRACKER.md`
  - `.ovav/artifacts/RC9_0/CLI_POLISH_SMOKE_EVIDENCE.md`
  - `.ovav/artifacts/CLI_LIVING_EXPERIENCE/SEG_03_TAILOR_TOGGLES_EVIDENCE.md`

## Results

- CLI syntax: PASS — cockpit and router compile.
- Install JSON smoke: PASS — `python3 -B bin/ovav install --json`.
- Tailor static smoke: PASS — `python3 -B bin/ovav tailor --no-interactive`.
- Tailor toggle flow: PASS — source-local simulation for Space toggle, Enter preview/apply and state preservation.
- Tailor feedback revision: PASS — plan gating, dimmed unavailable options, compact Tailor surface, dynamic footer and install confirmation.
- Practical sandbox smoke: PASS — `python3 -B tools/cli/ovav_practical_smoke.py --json`.
- Install sandbox smoke: PASS — `python3 -B tools/cli/ovav_install_smoke.py --json`.
- Fresh clone smoke: PASS — `python3 -B tools/cli/ovav_fresh_clone_smoke.py --json`.
- Runtime validation: PASS — `OVAV_EVIDENCE_MODE=strict python3 tools/ovav_runtime.py validate`.
- Final launch authority validators: PASS — current authority and runtime authority.

## Boundaries

- Source-local only.
- No global OpenCode config writes.
- No plugin install.
- No production/global-ready claim.
- Final launch closure remains blocked until final launch verification is closed and final tag exists.
