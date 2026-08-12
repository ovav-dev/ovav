// Package validators provides the OVAV validation pipeline in Go.
// Each validator implements the Validator interface and is self-contained,
// with no Python dependencies.
//
// Architecture:
//   - validators.go:       Interface + shared types + registry
//   - secrets_hygiene.go:  Scans codebase for plaintext secrets
//   - exfil_patterns.go:   Checks for exfiltration patterns in logs
//   - supply_chain.go:     Verifies dependency hashes and SBOM integrity
//   - protected_branch.go: Blocks writes on protected git branches
//   - workspace_safety.go: Validates workspace safety before writes
//   - git_push.go:         Enforces push gate rules
//   - permission_drift.go: Detects permission policy drift
//   - living_integrity.go: Orchestrator — runs all F0 validators
//   - runtime_integrity.go: Verifies file hashes against baseline
//   - contract_freshness.go: Checks contract staleness and integrity
//   - credential_governance.go: Validates credential vault, provider config, scope isolation
//   - security_hardening.go: Validates F4 bash commands, unsafe selectors, deny-by-default
//   - zero_trust.go: Validates L6 F0 hardening baseline, quarantine, risk thresholds
//   - advanced_hardening.go: Validates F5 new states, gate liberation, advanced surfaces
//   - runtime_wiring.go: Validates OpenCode surface files, governance terms, stale patterns
//   - context_firewall.go: Validates injection detection, L5 firewall, context economy
//   - registry_drift.go: Validates registry YAML declarations against real filesystem
//   - config_syntax.go: Validates YAML and JSON syntax across all config surfaces
//   - single_authority.go: Validates single canonical authority source (caps.yaml)
//   - release_gate.go: Validates version tag release readiness
//   - network_hardening.go: Validates network allowlist and critical domain coverage
//   - handoff_sync.go: Validates git-tracked state files consistency
package validators

import (
	"context"
	"time"
)

// Result represents the outcome of a single validator execution.
type Result struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"` // "pass", "fail", "error", "skip"
	Message     string        `json:"message,omitempty"`
	Issues      []string      `json:"issues,omitempty"`
	Suggestions []string      `json:"suggestions,omitempty"` // domains suggested for approval
	Weight      int           `json:"weight"`
	Duration    time.Duration `json:"duration_ms"`
	Description string        `json:"description,omitempty"`
}

// Validator is the interface all validators must implement.
type Validator interface {
	// ID returns the unique identifier for this validator.
	ID() string
	// Name returns the human-readable name.
	Name() string
	// Description returns a one-line description of what this validator checks.
	Description() string
	// Weight returns the importance weight (used by orchestrator for scoring).
	Weight() int
	// Validate executes the validation and returns the result.
	Validate(ctx context.Context, root string) Result
}

// Registry holds all registered validators.
type Registry struct {
	validators []Validator
}

// NewRegistry creates a validator registry with the given validators.
func NewRegistry(validators ...Validator) *Registry {
	return &Registry{validators: validators}
}

// All returns all registered validators.
func (r *Registry) All() []Validator {
	return r.validators
}

// Run executes all validators and returns their results.
func (r *Registry) Run(ctx context.Context, root string) []Result {
	results := make([]Result, 0, len(r.validators))
	for _, v := range r.validators {
		results = append(results, v.Validate(ctx, root))
	}
	return results
}

// DefaultRegistry returns the standard set of validators used in production.
//
// NOTE: As of 2026-08-09, many validators have been migrated to OMARS
// (OVAV Monitoring & Auto-Remediation System). These validators now delegate
// to monitors or return SKIP to avoid double-checking.
//
// Deprecated validators removed from default registry:
//   - ContextFirewallV2 (duplicate of ContextFirewall)
//   - MergeReadiness (→ HygieneMonitor in OMARS)
//   - ReleaseGate (→ HygieneMonitor in OMARS)
//   - HandoffSync (→ HygieneMonitor in OMARS)
//   - HeadIntegrity (hash drift normal between sessions)
//   - ArchitectureGuardian (directory structure not critical)
//   - CapsChronosAlignment (stale caps.yaml is WARN, not FAIL)
//   - CrossTargetConsistency (→ AgentProjectionMonitor in OMARS)
func DefaultRegistry() *Registry {
	return NewRegistry(
		NewSecretsHygiene(),
		NewExfilPatterns(),
		NewSupplyChain(),
		NewProtectedBranch(),
		NewGitPush(),
		NewPermissionDrift(),
		NewRuntimeIntegrity(),
		NewContractFreshness(),
		NewInstallVerification(),
		NewSecurityPolicy(),
		NewConfigIntegrity(),
		NewAgentGovernance(),
		NewPluginSecurity(),
		NewCredentialGovernance(),
		NewSecurityHardening(),
		NewZeroTrust(),
		NewAdvancedHardening(),
		NewRuntimeWiring(),
		NewContextFirewall(),
		NewRegistryDrift(),
		NewConfigSyntax(),
		NewSingleAuthority(),
		NewNetworkHardening(),
		// NOTE: ArchitectureCompliance → merged into ArchitectureGovernance
		NewContractEnforcement(),
		NewArchitectureGovernance(),
		// Batch 5 — Python→Go migration (remaining useful)
		NewThoughtFirewall(),
		NewSessionContextGuard(),
		NewGateSelfProtection(),
		NewModelPolicy(),
		NewHostConfigDrift(),
		NewWorkspaceIsolation(),
		NewLedgerWritePath(),
		NewSurfaceDrift(),
		NewAgentSurfaceHierarchy(),
		NewToolReadiness(),
		NewAgentRuntimeEnforcement(),
		NewHarnessIntegrity(),
		NewFeedbackLoop(),
		NewRegoPolicies(),
		NewMultiPlatform(),
		NewValidatorCoverage(),
		// Batch 6 — Python→Go migration (remaining useful)
		NewLedgerDeprecation(),
		NewServiceAreaGovernance(),
		NewRegistryValidator(),
		NewLeadScope(),
		NewAgentPermissionInvariants(),
		NewF1Architecture(),
		NewBehavioralDirectives(),
		// NOTE: CrossTargetConsistency → AgentProjectionMonitor in OMARS
		// NOTE: ContextFirewallV2 → duplicate of ContextFirewall
		// NOTE: HeadIntegrity → removed (hash drift normal)
		// NOTE: HandoffSync → redundant with HygieneMonitor
		// NOTE: ReleaseGate → redundant with MergeReadiness
		// NOTE: ArchitectureGuardian → directory structure not critical
		// NOTE: CapsChronosAlignment → stale is WARN not FAIL
		NewCanonicalIntegrity(),
		NewF2Infrastructure(),
		NewF3Roles(),
		// Batch 7 — final migration (15 validators, 100% complete)
		NewHarnessContractAlignment(),
		NewMemoryPolicy(),
		NewPhaseDAG(),
		NewBootstrapChain(),
		NewSkills(),
		NewServiceProfiles(),
		NewSquadNormalization(),
		NewToolConfigProfiles(),
		NewAgentUXVisualDelivery(),
		NewContextEconomy(),
		NewServiceAreaRouter(),
		NewStaleArtifactReferences(),
		NewInvalidFixtures(),
		NewSSHProfile(),
		NewWeztermPathIntegrity(),
		NewWeztermWorkspaceIsolation(),
		// Batch 8 — T15 Red Team automation
		NewRedTeamAudit(),
		// Batch 9 — v41.0 Caps authority blindaje
		NewCapsSingleNext(),
		// NOTE: CapsChronosAlignment deprecated - stale caps.yaml is INFO not FAIL
		NewCapsSchema(),
		// Batch 10 — Phase 3 innovation (absorbed from external systems)
		NewAdversarialVerification(),
	)
}

// e2e test
