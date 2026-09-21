# OVAV CLI Commands

> **Auto-generated** from `cmd/ovav/*.go`. DO NOT EDIT MANUALLY.
> Run `ovav docs generate` to refresh.

| Command | Description |
|---------|-------------|
| `ovav validate` | Run validation gates |
| `ovav validate --fix` | Auto-remediate SAFE_FIX validators (ADR-011) |
| `ovav drift` | Fragment vs live drift detection (ADR-007) |
| `ovav deploy` | Auto-deploy fragments to live (ADR-008) |
| `ovav ci` | CI runner commands (ADR-009) |
| `ovav hooks` | Git hook management |
| `ovav it` | Intelligent Terminal ops (ADR-010) |
| `ovav memory` | Agent memory queries |
| `ovav worktree` | OWS worktree lifecycle |
| `ovav status` | System status |
| `ovav sbom` | SBOM generation/verification |
| `ovav integrity` | Runtime integrity baseline (ADR-006) |
| `ovav drift show` | Visual diff fragment vs live |
| `ovav drift catalog` | Drift history |
| `ovav drift targets` | List registered drift targets |
| `ovav deploy run` | Execute deploy pipeline |
| `ovav deploy status` | Last deploy summary |
| `ovav deploy list` | All recent deploys |
| `ovav deploy rollback` | Restore from snapshot |
| `ovav ci drift-check` | CI-friendly drift check |
| `ovav hooks install-pre-commit` | Install baseline freshness hook |
| `ovav hooks install-pre-push` | Install drift gate hook |
| `ovav hooks install-all` | Install all OVAV hooks |
| `ovav hooks status` | Show all hook states |
| `ovav it reload` | IT reload via Win32 API |
| `ovav it status` | Check if IT is running |
| `ovav docs generate` | Generate auto-generated docs |
| `ovav docs check` | Verify docs are up-to-date |
