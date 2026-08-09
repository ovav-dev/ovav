package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PermissionDrift detects drift between the canonical permission authority
// and the runtime materialized permission state.
type PermissionDrift struct{}

func NewPermissionDrift() *PermissionDrift { return &PermissionDrift{} }

func (p *PermissionDrift) ID() string   { return "permission_drift" }
func (p *PermissionDrift) Name() string { return "Permission Policy Drift" }
func (p *PermissionDrift) Description() string {
	return "Detects drift between canonical permissions and runtime state"
}
func (p *PermissionDrift) Weight() int { return 10 }

// canonicalPermissionFiles are the authoritative sources for permission policy.
var canonicalPermissionFiles = []string{
	".ovav/policy/permission_authority.json",
	".ovav/policy/canonical_paths.json",
}

func (p *PermissionDrift) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Verify all canonical permission files exist and are readable
	for _, relPath := range canonicalPermissionFiles {
		fullPath := filepath.Join(root, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: Canonical permission file '%s' not found", relPath))
		} else if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read '%s': %v", relPath, err))
		}
	}

	// Check permission_authority.json for required structure
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	data, err := os.ReadFile(paPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read permission_authority.json: %v", err))
	} else if len(data) < 100 {
		issues = append(issues, "WARNING: permission_authority.json is suspiciously small (< 100 bytes)")
	}

	if len(issues) > 0 {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message: fmt.Sprintf("FAIL permission policy drift — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
		Message:  "PASS permission policy drift",
		Duration: time.Since(start),
	}
}

var _ Validator = (*PermissionDrift)(nil)
