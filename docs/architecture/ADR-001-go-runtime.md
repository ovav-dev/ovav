# ADR-001: Go Runtime as Primary Product Stack

**Date:** 2026-06-17  
**Status:** Accepted  
**Decider:** Alexander Salvador (CEO) + Thavren (Platform Engineering)

## Context

OVAV transitioned from a Python monolith to a Go-native runtime for all product surfaces. Python remains as governance/operational layer.

## Decision

- Go runtime (`go-runtime/`): All product code — cPanel backend, Cockpit TUI, Tailor pipeline, vault, validators, install gateway
- TypeScript (`tools/cpanel/`): Web frontend (React 18 + Vite) — browser-only runtime, inamovible
- Python (`tools/`): Governance validators, harnesses, agent runtime, security tools — operational, NOT product

## Consequences

- Zero Python in product surface ✅
- 5 Go binaries cross-compiled for linux/darwin/windows
- 30 Go validators (38% of 79 migrated)
- 0 data races, gofmt/vet clean
