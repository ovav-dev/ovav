# ISSUE-2026-0804-GOVERNANCE-PY-GO-MIGRATION — Governance surface Python→Go migration: 70% remaining

**Severity:** 🟡 Governance Debt
**Status:** 🆕 OPEN
**Detected:** 2026-08-04
**Affects:** `tools/validators/`, `tools/security/`, `tools/github/`, `.ovav/connector_bus/`

---

## Problem

OVAV governing system runs on **100% Go runtime** since Phase 6. However, **governance tooling** (hooks, validators, CI scripts) still contains **39 Python files** that must be migrated per project rule:

> "Cada detección de un sistema de py <- debe crearse una issue y atacarla en la brevedidad posible, ya que el sistema trabajado en .go nativo al 100%"

**Current state** (from `caps.yaml`):
- User-facing surface: 98% Go ✅
- Governance surface: **30% migrated** | **70% remaining** ❌

---

## Scope

### Critical (blocks CI reliability)
| File | Purpose | Risk if unfixed |
|---|---|---|
| `tools/validators/validate_all.py` | Pre-commit/CI validation | Incompatible with pure-Go CI |
| `tools/validators/check_secrets.py` | Secrets hygiene | Drift from Go secrets validator |
| `tools/validators/check_hooks.py` | Hook validation | May conflict with new hooks |

### High (governance gaps)
| File | Purpose | Risk |
|---|---|---|
| `tools/security/sbom.py` | SBOM generation | **Already migrated** to `cmd/sbom_regen/` ✅ |
| `tools/github/*.py` | GitHub API scripts | No longer needed if `gh` CLI covers all |
| `.ovav/connector_bus/bus.py` | Connector bus | Must be replaced by Go-native connector |
| `tools/validators/check_*.py` | Individual check scripts | ~8 files, consolidate into Go |

### Already Migrated ✅
- SBOM generation → `go-runtime/cmd/sbom_regen/`
- Vault subsystem → `go-runtime/internal/vault/`
- Secrets hygiene → `go-runtime/internal/validators/secrets_hygiene.go`
- Config integrity → `go-runtime/internal/validators/config_integrity.go`
- Supply chain → `go-runtime/internal/validators/supply_chain.go`
- Email dispatch → `go-runtime/internal/email/dispatcher.go`

---

## Root Cause

The previous session fixed the **post-commit hook** calling `tools/security/sbom.py` (which doesn't exist), replacing it with `go run sbom_regen`. This exposed the broader problem: many governance scripts still in Python.

---

## Proposed Fix

1. **Phase 1 (this issue):** Audit and document all remaining Python governance files
2. **Phase 2:** Migrate `tools/validators/` to Go validators (`go-runtime/internal/validators/`)
3. **Phase 3:** Migrate `tools/github/` scripts to `gh` CLI + Go wrappers
4. **Phase 4:** Migrate `connector_bus/bus.py` to Go-native connector

---

## GitHub Issue

Cross-reference: GitHub issue [#67](https://github.com/ovav-dev/ovav-systems/issues/67) created simultaneously.

## Sync

This file is synced to GitHub via `.github/workflows/issue-sync.yml`.
Any change to `status:` field here should reflect in GitHub and vice versa.
