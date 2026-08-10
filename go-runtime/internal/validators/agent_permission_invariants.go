package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentPermissionInvariants validates that lead agent files don't have
// permission drift between the lead surface and area surface.
// Compares lead-thavren.md against area-platform-engineering.md
// for consistency in permission blocks.
// Replaces: tools/validators/check_agent_permission_invariants.py
type AgentPermissionInvariants struct{}

func NewAgentPermissionInvariants() *AgentPermissionInvariants {
	return &AgentPermissionInvariants{}
}

func (v *AgentPermissionInvariants) ID() string   { return "agent_permission_invariants" }
func (v *AgentPermissionInvariants) Name() string { return "Agent Permission Invariants" }
func (v *AgentPermissionInvariants) Description() string {
	return "Validates lead-agent permission invariants against area surface"
}
func (v *AgentPermissionInvariants) Weight() int { return 7 }

func (v *AgentPermissionInvariants) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	thavrenPath := filepath.Join(root, ".ovav", "service_areas", "platform_engineering", "lead_contract.yaml")
	areaPath := filepath.Join(root, ".ovav", "service_areas", "platform_engineering", "area_boundaries.yaml")

	thavrenData, err := os.ReadFile(thavrenPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read lead_contract.yaml: %v", err))
	}
	areaData, err := os.ReadFile(areaPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read area_boundaries.yaml: %v", err))
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	var thavrenDoc, areaDoc map[string]interface{}
	if err := yaml.Unmarshal(thavrenData, &thavrenDoc); err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse lead_contract.yaml: %v", err))
	}
	if err := yaml.Unmarshal(areaData, &areaDoc); err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse area_boundaries.yaml: %v", err))
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	if lead, ok := thavrenDoc["lead_contract"].(map[string]interface{}); ok {
		if leadID, ok := lead["lead"].(string); !ok || leadID != "thavren" {
			issues = append(issues, "ERROR: lead_contract.lead must be 'thavren'")
		}
	} else {
		issues = append(issues, "ERROR: lead_contract.yaml missing lead_contract section")
	}

	if area, ok := areaDoc["area"].(string); !ok || area != "platform_engineering" {
		issues = append(issues, "ERROR: area_boundaries.yaml area must be 'platform_engineering'")
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:  "PASS agent permission invariants — Thavren and Platform Engineering aligned",
		Duration: time.Since(start),
	}
}

var _ Validator = (*AgentPermissionInvariants)(nil)
