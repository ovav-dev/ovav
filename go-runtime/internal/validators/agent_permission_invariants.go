package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// parseAgentFMFromContent extracts YAML frontmatter from markdown content string
func parseAgentFMFromContent(content string) (map[string]interface{}, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil, fmt.Errorf("no frontmatter found")
	}
	// Find closing ---
	var fmLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		fmLines = append(fmLines, lines[i])
	}
	if len(fmLines) == 0 {
		return nil, fmt.Errorf("empty frontmatter")
	}
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(strings.Join(fmLines, "\n")), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (v *AgentPermissionInvariants) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Try new structure first: .ovav/service_areas/platform_engineering/
	saDir := filepath.Join(root, ".ovav", "service_areas", "platform_engineering")
	thavrenYAMLPath := filepath.Join(saDir, "lead_contract.yaml")
	areaYAMLPath := filepath.Join(saDir, "area_boundaries.yaml")

	// Also check old structure: clients/opencode/agents/
	agentsDir := filepath.Join(root, "clients", "opencode", "agents")
	thavrenMDPath := filepath.Join(agentsDir, "lead-thavren.md")
	areaMDPath := filepath.Join(agentsDir, "area-platform-engineering.md")

	var thavrenDoc, areaDoc map[string]interface{}
	var thavrenData, areaData []byte
	usingYAMLFormat := false

	// Read thavren: try YAML first, then markdown
	if data, err := os.ReadFile(thavrenYAMLPath); err == nil {
		thavrenData = data
		usingYAMLFormat = true
		if err := yaml.Unmarshal(thavrenData, &thavrenDoc); err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse lead_contract.yaml: %v", err))
		}
	} else if data, err := os.ReadFile(thavrenMDPath); err == nil {
		thavrenData = data
		thavrenDoc, err = parseAgentFMFromContent(string(thavrenData))
		if err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse lead-thavren.md frontmatter: %v", err))
		}
	} else {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read lead file (tried lead_contract.yaml and lead-thavren.md): %v", err))
	}

	// Read area: try YAML first, then markdown
	if data, err := os.ReadFile(areaYAMLPath); err == nil {
		areaData = data
		if err := yaml.Unmarshal(areaData, &areaDoc); err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse area_boundaries.yaml: %v", err))
		}
	} else if data, err := os.ReadFile(areaMDPath); err == nil {
		areaData = data
		areaDoc, err = parseAgentFMFromContent(string(areaData))
		if err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse area-platform-engineering.md frontmatter: %v", err))
		}
	} else {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read area file (tried area_boundaries.yaml and area-platform-engineering.md): %v", err))
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// Validate lead has correct lead ID
	if lead, ok := thavrenDoc["lead_contract"].(map[string]interface{}); ok {
		if leadID, ok := lead["lead"].(string); !ok || leadID != "thavren" {
			issues = append(issues, "ERROR: lead_contract.lead must be 'thavren'")
		}
	} else if name, ok := thavrenDoc["name"].(string); ok {
		// Old markdown format: check name field
		if name != "Thavren" {
			issues = append(issues, fmt.Sprintf("ERROR: lead name must be 'Thavren', got %q", name))
		}
	} else {
		issues = append(issues, "ERROR: lead file missing lead_contract section (or name field in markdown)")
	}

	// Validate area has correct area ID
	if area, ok := areaDoc["area"].(string); ok {
		// New YAML format
		if area != "platform_engineering" {
			issues = append(issues, fmt.Sprintf("ERROR: area must be 'platform_engineering', got %q", area))
		}
	} else if name, ok := areaDoc["name"].(string); ok {
		// Old markdown format: check name field
		if name != "Platform Engineering" {
			issues = append(issues, fmt.Sprintf("ERROR: area name must be 'Platform Engineering', got %q", name))
		}
	} else {
		issues = append(issues, "ERROR: area file missing area field (or name field in markdown)")
	}

	// Validate permission consistency between lead and area (only for markdown format)
	// YAML format uses lead_contract and area fields, not permission blocks
	if !usingYAMLFormat {
		thavrenPerm, _ := thavrenDoc["permission"].(map[string]interface{})
		areaPerm, _ := areaDoc["permission"].(map[string]interface{})

		// Lead must have permission block in markdown format
		if thavrenPerm == nil {
			issues = append(issues, "ERROR: lead file missing permission block")
		}

		// If both have permission blocks, check consistency
		if thavrenPerm != nil && areaPerm != nil {
			// Check edit permission - must be a string "allow" or "deny"
			thavrenEdit, thavrenEditIsString := thavrenPerm["edit"].(string)
			areaEdit, areaEditIsString := areaPerm["edit"].(string)
			if !thavrenEditIsString {
				issues = append(issues, "ERROR: lead edit permission must be a string (allow/deny)")
			}
			if !areaEditIsString {
				issues = append(issues, "ERROR: area edit permission must be a string (allow/deny)")
			}
			// Area cannot have edit: deny if lead has edit: allow
			if thavrenEditIsString && areaEditIsString && thavrenEdit == "allow" && areaEdit == "deny" {
				issues = append(issues, "ERROR: lead edit=allow but area edit=deny — area cannot restrict lead's edit")
			}

			// Check bash permission consistency
			if thavrenBash, ok := thavrenPerm["bash"].(map[string]interface{}); ok {
				if areaBash, ok := areaPerm["bash"].(map[string]interface{}); ok {
					// Check all bash sub-fields - if lead denies a field, area cannot allow it
					for field, leadVal := range thavrenBash {
						if leadStr, ok := leadVal.(string); ok && leadStr == "deny" {
							if areaVal, ok := areaBash[field].(string); ok && areaVal == "allow" {
								issues = append(issues, fmt.Sprintf("ERROR: lead bash.%s=deny but area bash.%s=allow", field, field))
							}
						}
					}
				}
			}

			// Check external_directory - must be a map, not a list
			if thavrenExtDir, ok := thavrenPerm["external_directory"].(map[string]interface{}); ok {
				if areaExtDir, ok := areaPerm["external_directory"].(map[string]interface{}); ok {
					// Check wildcard consistency
					if thavrenWildcard, ok := thavrenExtDir["*"].(string); ok {
						if areaWildcard, ok := areaExtDir["*"].(string); ok {
							if thavrenWildcard == "deny" && areaWildcard == "allow" {
								issues = append(issues, "ERROR: lead external_directory * = deny but area * = allow")
							}
						}
					}
				}
			} else {
				// external_directory is not a map (might be a list)
				if _, isList := thavrenPerm["external_directory"].([]interface{}); isList {
					issues = append(issues, "ERROR: lead external_directory must be a map, not a list")
				}
			}

			// Check for fabricated permission keys in lead
			validPermissionKeys := map[string]bool{
				"edit": true, "bash": true, "external_directory": true,
			}
			for key := range thavrenPerm {
				if !validPermissionKeys[key] {
					issues = append(issues, fmt.Sprintf("ERROR: lead has fabricated permission key %q", key))
				}
			}
		}
	}

	// Check for empty name
	if name, ok := thavrenDoc["name"].(string); ok && name == "" {
		issues = append(issues, "ERROR: lead name is empty string")
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
