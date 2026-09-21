# ADR-004: Tools Registry & Baseline Stabilization

**Date:** 2026-08-14
**Status:** Active
**Decider:** Thavren
**Related commits:** `9b9ef44`, `fd84907`, `3794a83`, `5c7e243`, `2849065`, `026cc48`

## Context

On 2026-08-14, `ovav validate` reported a `Supply Chain Integrity FAIL` with 7 baseline/security issues, plus several WARN-level drifts in the Workspace Isolation validator. Investigation revealed a deeper architectural drift:

1. **Registry drift:** `.ovav/registry/tool_configs.yaml` and `.ovav/registry/tool_changelog.yaml` referenced 604 Python tools (`tools/harnesses/*.py`, `tools/agent_runtime/*.py`, `tools/governor/*.py`, etc.) that had been removed during the Python → Go runtime migration. The actual `tools/` directory contained only `cpanel/`, `extensions/`, `infra/`, `mcp/`, `web/` — five subdirectories, not 604 Python files.

2. **Policy drift:** `config/workstation/ovav-wezterm-workspace-isolation.yaml` referenced `tools/workstation/ovav_wezterm_workspace.py` and `tools/validators/check_ovav_wezterm_workspace_isolation.py` — both removed. The actual Go validators lived at `go-runtime/internal/validators/workspace_isolation.go` and `wezterm_path_integrity.go`.

3. **SBOM baseline staleness:** `.ovav/registry/sbom.json` was generated on 2026-06-10 (2 months stale) and flagged 3 `workstation/scripts/*.sh` files as `UNEXPECTED_TRACKED` plus 3 `HASH_MISMATCH` for workstation configs that had been modified since.

4. **6 stale branches:** `feature/feat-baseline-final`, `feature/feat-baseline-regen`, `feature/feat-cleanup-100pct`, `feature/feat-node-health-fix`, `feature/feat-yolo-trusted-domain`, `fix/sbom-baseline` — all 0 commits ahead of `develop`, abandoned snapshots with no actual work.

5. **Hardcoded paths in scripts:** The 3 `workstation/scripts/*.sh` files (registered as UNEXPECTED_TRACKED) contained hardcoded `/mnt/c/Users/Alexa/AppData/Local/Packages/Microsoft.IntelligentTerminal_8wekyb3d8bbwe/...` paths, hardcoded `$HOME/.ovav-backups`, hardcoded GUID namespaces, and implicit dependencies on `python3` and `powershell.exe`.

This drift was not a security leak — no actual secrets were exposed — but it represented **loss of architectural control**: the registry declared one reality, the filesystem another, and no validator could reconcile them.

## Decision

### 1. Registry resync (single source of truth)

Replace all references to removed Python helpers with explicit `helper_tool_status: removed_python_helper` markers and current Go validator paths. This makes the drift visible and intentional rather than invisible and accidental.

**`tool_configs.yaml`:**
```yaml
# Before
helper_tool: tools/workstation/ovav_wezterm_workspace.py
validator: tools/validators/check_ovav_wezterm_workspace_isolation.py

# After
helper_tool_status: removed_python_helper
helper_tool_note: "Python helper removed during Go runtime migration; wezterm workspace isolation is now enforced by go-runtime/internal/validators/workspace_isolation.go (no global writes)."
validator: go-runtime/internal/validators/workspace_isolation.go
path_integrity_validator: go-runtime/internal/validators/wezterm_path_integrity.go
```

### 2. Source-local script registration (formal profile)

Register the 3 workstation maintenance scripts as a new tool profile under `tool_configs.yaml` → `ovav_workstation_scripts`, with:
- Explicit env-var contracts (`OVAV_IT_SETTINGS`, `OVAV_BACKUP_DIR`, etc.)
- Documented external dependencies (`bash ≥ 4.0`, `python3`, `powershell`)
- `apply_posture: user_invoked_only` (never auto-run from OVAV governor)
- Boundary `no_hardcoded_absolute_paths: true`

Companion policy: `config/workstation/ovav-workstation-scripts.yaml` documents the path-externalization strategy and idempotency contracts.

### 3. Path externalization (no hardcoded anything)

Refactor all 3 scripts so that:
- Every absolute path comes from an env var (REQUIRED env vars have NO defaults)
- Every backup destination is configurable
- Every GUID namespace is configurable
- Sibling script paths are derived from `BASH_SOURCE` (not hardcoded)
- External dependencies are checked with `command -v` and fail with clear errors if missing
- Every script has a header documenting required/optional env vars and dependencies

**Example (audit-it-guids.sh):**
```bash
# Before
IT_SETTINGS="/mnt/c/Users/Alexa/AppData/Local/Packages/.../settings.json"
python3 << 'PYEOF'  # implicit dependency
with open("/mnt/c/Users/Alexa/.../settings.json") as f: ...
PYEOF

# After
: "${OVAV_IT_SETTINGS:?OVAV_IT_SETTINGS env var is required}"
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 required"; exit 3; }
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
```

### 4. SBOM baseline regeneration

Run `ovav sbom generate` after registry changes. This anchors the SHA-256 baseline to the current HEAD and registers the 3 workstation scripts as known/expected tracked files. The new identity is recorded in `.ovav/registry/sbom.json`.

### 5. Branch hygiene

Delete 6 stale local branches that are 0 commits ahead of `develop` (safe delete). Delete 1 remote branch (`origin/feature/yolo-full-advance-2026-100pct-cleanup`) that was already merged into `develop` (`git merge-base --is-ancestor` confirmed). The `worktrees/` directory was added to `.gitignore` to prevent accidental commits.

## Consequences

### Positive

- **`ovav validate`: 67 PASS / 4 warned / 0 failed** (up from 66/4/1)
- Supply Chain Integrity validator no longer fails on UNEXPECTED_TRACKED or HASH_MISMATCH
- Workspace Isolation validator no longer warns on `CONFIG_PROJECTION_STALE`
- Workstation scripts are formally recognized as part of OVAV (registered in `tool_configs.yaml`)
- Zero hardcoded paths in workstation scripts — fully portable across machines/operators
- 7 fewer stale branches confusing the audit trail
- Clear documentation of dependencies (bash, python3, powershell) instead of implicit assumptions

### Negative

- Scripts now require operators to set `OVAV_IT_SETTINGS` env var before invocation (was: just ran)
- The `bin/ovav` validator check (`tool_config_profiles.go`) is fragile — fails in worktrees without rebuilding. This is a known wart noted for future fix but out of scope for this stabilization.
- 5 WARN-level results remain (all `INTENTIONALLY_GATED` markers, e.g., "no current Go WezTerm workspace launch helper"). These are intentional, not fixable without architectural changes outside this ADR's scope.

### Risks mitigated

- **Drift recurrence:** The `helper_tool_status: removed_python_helper` marker makes future drift visible rather than silent
- **Path portability:** Operators can run the scripts on any machine with their own settings.json path
- **Dependency assumptions:** `command -v` checks fail fast with clear messages if `python3` or `powershell` is missing
- **Idempotency preserved:** GUID replacement uses uuid5 with configurable namespace (deterministic across re-runs)

## References

- **Validators resolved:** `Supply Chain Integrity` (FAIL → PASS), `Workspace Isolation` (WARN → clean), `Tool Config Profiles` (PASS)
- **Commits:** see related commits listed above
- **CHANGELOG entry:** Top section of `CHANGELOG.md`
- **Policy files:** `config/workstation/ovav-wezterm-workspace-isolation.yaml`, `config/workstation/ovav-workstation-scripts.yaml`
- **Registry:** `.ovav/registry/tool_configs.yaml` (2 profiles: `wezterm_workspace_isolation`, `ovav_workstation_scripts`)
- **Related ADR:** `ADR-003-validator-migration-strategy.md` (Python → Go validators)
