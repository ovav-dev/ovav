# RESEARCH SCOPE — License & Plan Enforcement for Cockpit TUI

**Session:** 2026-06-18 23:54 UTC-5
**Lead:** Eidren (Research Intelligence)
**Requestor:** CEO
**Status:** COMPLETE

## Scope

Competitive analysis for OVAV Cockpit TUI (Go + Bubble Tea) covering two topics:

### Topic 1: Membership/License Management in TUI Tools
- License activation flows in commercial CLI/TUI tools
- Authentication mechanisms (OAuth device flow, API keys, license files, machine binding)
- Offline operation after verification
- Anti-tampering mechanisms in Go binaries
- Web↔TUI membership state sync
- Simplest purchase→activation flow

### Topic 2: Plan/Document Enforcement Systems
- How PM tools enforce work follows plan
- Git branch/worktree ↔ task/plan binding mechanisms
- Stacked PR/commit enforcement
- Git-level "must follow declared plan" tools

## Existing OVAV Assets Reviewed

| Asset | Path | Relevance |
|-------|------|-----------|
| License bind system | `go-runtime/internal/license/bind.go` | HMAC-signed keys, PBKDF2 machine binding, stdlib-only crypto |
| Cockpit TUI | `go-runtime/cmd/cockpit/` | Bubble Tea dashboard, caps.yaml reader, navigation |
| Plan data model | `.ovav/plan/caps.yaml` | Pending caps with deps/order/worktree/tasks |
| Plan data loader | `go-runtime/cmd/cockpit/data/caps.go` | YAML→Go struct, Cockpit integration |

## Evidence Sources

7 sources consulted (see SOURCE_MAP.md). Priority: official docs > technical blogs > practitioner patterns.

## Out of Scope
- Web checkout/payment implementation
- OAuth server deployment
- License key generation service
- Full DRM system
