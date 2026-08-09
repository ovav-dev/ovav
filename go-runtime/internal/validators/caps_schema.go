package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CapsSchema validates that caps.yaml has the canonical structure required
// for the OVAV plan matrix: header fields, required sections, no stale artifacts.
// Added v41.0 to prevent structural degradation of the plan matrix.
type CapsSchema struct{}

func NewCapsSchema() *CapsSchema { return &CapsSchema{} }

func (v *CapsSchema) ID() string   { return "caps_schema" }
func (v *CapsSchema) Name() string { return "Caps Canonical Schema" }
func (v *CapsSchema) Description() string {
	return "Ensures caps.yaml has the required canonical structure"
}
func (v *CapsSchema) Weight() int { return 15 }

// requiredTopLevel are fields that MUST exist at the root of caps.yaml.
var requiredTopLevel = []string{
	"version",
	"canonical",
	"updated_at",
	"updated_by",
	"plan_version",
	"next_phase",
}

// requiredSections are top-level sections that must exist.
var requiredSections = []string{
	"current_state",
	"architecture",
	"governance_workflow",
	"governance_wiring",
	"subsidiary_plans",
}

func (v *CapsSchema) Validate(ctx context.Context, root string) Result {
	root = resolveRepoRoot(root)
	capsPath := filepath.Join(root, ".ovav", "plan", "caps.yaml")

	data, err := os.ReadFile(capsPath)
	if err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot read caps.yaml",
			Issues: []string{err.Error()},
		}
	}

	var caps map[string]interface{}
	if err := yaml.Unmarshal(data, &caps); err != nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status: "fail", Message: "Cannot parse caps.yaml",
			Issues: []string{err.Error()},
		}
	}

	var issues []string

	// Check required top-level fields
	for _, field := range requiredTopLevel {
		if _, ok := caps[field]; !ok {
			issues = append(issues, fmt.Sprintf("missing required field: %q", field))
		}
	}

	// Check canonical flag
	if canonical, ok := caps["canonical"]; ok {
		if c, ok := canonical.(bool); !ok || !c {
			issues = append(issues, "'canonical' must be true — caps.yaml is the single source of truth")
		}
	}

	// Check required sections exist
	for _, section := range requiredSections {
		if _, ok := caps[section]; !ok {
			issues = append(issues, fmt.Sprintf("missing required section: %q", section))
		}
	}

	// Check for known stale sections that should not exist
	staleSections := []string{
		"active_context_ledger",
		"engram_config",
		"capsule_runtime",
		"go_migration_plan", // deprecated — consolidated into current_state + architecture
	}
	for _, stale := range staleSections {
		if _, ok := caps[stale]; ok {
			issues = append(issues, fmt.Sprintf("stale section detected: %q — this section is deprecated and must be removed", stale))
		}
	}

	// Duplicate plan_version check: should only be at root, not in current_state
	if cs, ok := caps["current_state"].(map[string]interface{}); ok {
		if _, ok := cs["plan_version"]; ok {
			issues = append(issues, "current_state.plan_version is forbidden — plan_version is top-level only. Remove this duplicate.")
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
			Status:  "fail",
			Message: fmt.Sprintf("%d schema violations in caps.yaml", len(issues)),
			Issues:  issues,
		}
	}

	return Result{
		ID: v.ID(), Name: v.Name(), Weight: v.Weight(),
		Status:  "pass",
		Message: "caps.yaml schema valid — all required fields and sections present",
	}
}
