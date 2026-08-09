// Package permissions implements OVAV permission governance in Go.
// Migrated from tools/permissions/ (Python).
//
// Components:
//   - rego_engine.go:          OPA/Rego-style policy evaluation (F1.1)
//   - permission_authority.go: Canonical policy materialization + drift (F1)
//   - governors.go:            Bash, claims, config, new_states, plugin,
//     sandbox, system_paths, simulate, verify
package permissions
