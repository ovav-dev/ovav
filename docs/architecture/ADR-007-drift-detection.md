# ADR-007: Drift Detection Architecture

**Date:** 2026-08-14
**Status:** Accepted
**Related:** ADR-005 (Phase 1 anti-drift core), ADR-006 (baseline versioning)
**Decider:** Thavren + CEO

## Context

After 4 sessions of patches that addressed symptoms not root causes (ADR-005),
we identified the core problem: **drift between source-of-truth (fragment in
repo) and live state (system configs)**. The pattern repeats:

1. User reports something broken
2. We patch the fragment (correct fix)
3. Deploy to live fails or is incomplete
4. User reports "sigue igual"
5. Repeat

The missing piece is **visibility** — operators need to SEE what drift exists
before deciding what to do about it.

## Decision

Add `ovav drift show` command that produces a **visual + machine-readable**
report of all drift between fragment (canonical) and live (deployed) state.

### Architecture: Target Registry

A target is a pair (fragment_path, live_path) plus a comparison function.
Targets are registered in code (initially) and can be added via API later.

```go
type DriftTarget struct {
    ID           string                                          // "it-keybindings"
    Name         string                                          // "IT Keybindings"
    FragmentPath string                                          // "workstation/configs/intelligent-terminal/settings-fragment.json"
    LivePath     string                                          // "/mnt/c/.../LocalState/settings.json"
    LivePathEnv  string                                          // "OVAV_LIVE_IT_SETTINGS"
    Compare      func(fragment, live []byte) ([]DriftItem, error) // target-specific
    CanAutoFix   bool                                            // "ovav deploy run" can fix
}
```

### Initial targets (5)

| ID | Fragment | Live | Auto-fix |
|----|----------|------|----------|
| `it-keybindings` | `workstation/configs/intelligent-terminal/settings-fragment.json` | `/mnt/c/.../settings.json` | ✅ |
| `bash-inputrc` | `workstation/configs/inputrc/ovav.inputrc` | `~/.inputrc` | ✅ |
| `tool-configs` | `.ovav/registry/tool_configs.yaml` | `bin/ovav` (CLI binary) | ❌ (rebuild) |
| `runtime-baseline` | `.ovav/integrity_backups/baseline.json` | current file hashes | ✅ (regen) |
| `pinned-baseline` | `.ovav/integrity_backups/baseline.pinned.json` | current vs pinned | ⚠️ (CEO approve) |

### DriftItem structure

```go
type DriftItem struct {
    Type    string // "missing_in_live" | "missing_in_fragment" | "modified" | "added"
    Path    string // JSON path (e.g., "keybindings[5].keys")
    Fragment interface{} // current fragment value (nil if missing)
    Live     interface{} // current live value (nil if missing)
    SuggestedFix string // command to apply fix (e.g., "ovav deploy run --target=it-keybindings")
}
```

### Output formats

1. **Human-readable (default)** — color-coded table with:
   - Target header (ID, fragment, live path, file sizes)
   - Drift summary (X missing, Y modified, Z added)
   - Per-item detail (type, path, fragment vs live diff)
   - Suggested fixes (commands to run)
2. **JSON (`--json`)** — machine-readable for CI/dashboards:
   ```json
   {
     "timestamp": "2026-08-14T...",
     "total_targets": 5,
     "drifted_targets": 2,
     "targets": [
       {
         "id": "it-keybindings",
         "drift_count": 3,
         "items": [...]
       }
     ]
   }
   ```
3. **Markdown (`--md`)** — for embedding in PRs/comments:
   ```markdown
   ## IT Keybindings Drift
   - 3 items
   - 2 missing in live, 1 modified
   ```

### Drift catalog

Each `ovav drift show` run appends to `.ovav/registry/drift_catalog.jsonl`
(one line per run, JSONL format). Enables:
- Drift over time analysis
- "Is drift growing or shrinking?"
- Auto-correlate with commits (last_seen_clean < commit_time < last_seen_drift)

## Consequences

### Positive

- **Visibility** — operators SEE drift before users notice
- **Actionability** — each drift has a suggested fix command
- **Auditability** — drift_catalog.jsonl captures history
- **CI-ready** — `--json` output for automated gates
- **Composable** — `ovav deploy run --from-drift-report=path.json` (Phase 1/D1)

### Negative

- **Code surface** — new package, new tests, new docs
- **Path fragility** — Windows paths, WSL mounts, env vars
- **Comparison complexity** — each target has different structure (JSON vs INI vs YAML)

### Mitigations

- Reuse existing validators (it_keybindings, it_live_keybindings, bash_readline_bindings)
- Env var overrides for live paths (OVAV_LIVE_IT_SETTINGS, OVAV_LIVE_INPUTRC, etc.)
- Each target owns its comparison function (no god-comparator)

## References

- ADR-005 Phase 1 anti-drift core (D4)
- ADR-006 baseline versioning (D3)
- `go-runtime/internal/validators/it_keybindings.go`
- `go-runtime/internal/validators/it_live_keybindings.go`
- `go-runtime/internal/validators/bash_readline_bindings.go`
