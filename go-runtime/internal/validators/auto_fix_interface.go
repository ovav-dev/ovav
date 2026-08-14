package validators

// Fixable is the interface for validators that can auto-correct their issues.
//
// Validators opt-in by:
//  1. Implementing Fix() error
//  2. Marking the comment "// SAFE_FIX: <description>" above the struct
//  3. Adding the validator ID to safeFixRegistry in auto_fix.go
//
// The auto-remediation pipeline (ovav validate --fix) calls Fix() for each
// registered validator, applies the change, verifies no regression,
// and rolls back if needed.
type Fixable interface {
	// Validator is the existing interface
	Validator
	// Fix returns nil on success, error on failure.
	// Must be idempotent (safe to call multiple times).
	Fix(root string) error
	// FixDescription returns a human-readable description of what Fix does.
	FixDescription() string
}

// FixResult is one auto-fix attempt outcome.
type FixResult struct {
	ValidatorID string `json:"validator_id"`
	Description string `json:"description"`
	Outcome     string `json:"outcome"` // applied | skipped | failed | no-op | rollback
	Error       string `json:"error,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
	Files       []string `json:"files,omitempty"` // files touched by this fix
}

// SafeFixEntry is one entry in the safe-fix registry (whitelist).
type SafeFixEntry struct {
	ValidatorID   string   `   string:"validator_id"`
	Description   string   `   string:"description"`
	RiskLevel     string   `   string:"risk_level"` // low | medium | high
	RequiresWaiver bool   `   string:"requires_waiver"` // CEO waiver needed
	ProtectedTargets []string `   string:"protected_targets,omitempty"`
}

// AutoFixRegistry is the whitelist of validators allowed to auto-fix.
type AutoFixRegistry struct {
	Entries []SafeFixEntry `json:"entries"`
}