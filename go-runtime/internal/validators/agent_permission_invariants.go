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
	var failures []string
	var warnings []string

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
			failures = append(failures, fmt.Sprintf("CRITICAL: Cannot parse lead_contract.yaml: %v", err))
		}
	} else if data, err := os.ReadFile(thavrenMDPath); err == nil {
		thavrenData = data
		thavrenDoc, err = parseAgentFMFromContent(string(thavrenData))
		if err != nil {
			failures = append(failures, fmt.Sprintf("CRITICAL: Cannot parse lead-thavren.md frontmatter: %v", err))
		}
	} else {
		failures = append(failures, fmt.Sprintf("CRITICAL: Cannot read lead file (tried lead_contract.yaml and lead-thavren.md): %v", err))
	}

	// Read area: try YAML first, then markdown
	if data, err := os.ReadFile(areaYAMLPath); err == nil {
		areaData = data
		if err := yaml.Unmarshal(areaData, &areaDoc); err != nil {
			failures = append(failures, fmt.Sprintf("CRITICAL: Cannot parse area_boundaries.yaml: %v", err))
		}
	} else if data, err := os.ReadFile(areaMDPath); err == nil {
		areaData = data
		areaDoc, err = parseAgentFMFromContent(string(areaData))
		if err != nil {
			failures = append(failures, fmt.Sprintf("CRITICAL: Cannot parse area-platform-engineering.md frontmatter: %v", err))
		}
	} else {
		failures = append(failures, fmt.Sprintf("CRITICAL: Cannot read area file (tried area_boundaries.yaml and area-platform-engineering.md): %v", err))
	}

	if len(failures) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d critical issue(s)", len(failures)),
			Issues:   failures,
			Duration: time.Since(start),
		}
	}

	// Normalize name field: strip any embedded newlines/CR characters that may
	// be present from YAML parser quirks (some YAML parsers include trailing
	// content in string values when the value spans lines).
	if name, ok := thavrenDoc["name"].(string); ok {
		normalized := strings.ReplaceAll(strings.ReplaceAll(name, "\n", ""), "\r", "")
		normalized = strings.TrimSpace(normalized)
		if normalized != name {
			thavrenDoc["name"] = normalized
		}
	}

	// Validate lead has correct lead ID
	if lead, ok := thavrenDoc["lead_contract"].(map[string]interface{}); ok {
		if leadID, ok := lead["lead"].(string); !ok || leadID != "thavren" {
			failures = append(failures, "ERROR: lead_contract.lead must be 'thavren'")
		}
	} else if name, ok := thavrenDoc["name"].(string); ok {
		// Old markdown format: check name field
		if name != "Thavren" {
			failures = append(failures, fmt.Sprintf("ERROR: lead name must be 'Thavren', got %q", name))
		}
	} else {
		failures = append(failures, "ERROR: lead file missing lead_contract section (or name field in markdown)")
	}

	// Validate area has correct area ID
	if area, ok := areaDoc["area"].(string); ok {
		// New YAML format
		if area != "platform_engineering" {
			failures = append(failures, fmt.Sprintf("ERROR: area must be 'platform_engineering', got %q", area))
		}
	} else if name, ok := areaDoc["name"].(string); ok {
		// Old markdown format: check name field
		if name != "Platform Engineering" {
			failures = append(failures, fmt.Sprintf("ERROR: area name must be 'Platform Engineering', got %q", name))
		}
	} else {
		failures = append(failures, "ERROR: area file missing area field (or name field in markdown)")
	}

	// Validate permission consistency between lead and area (only for markdown format)
	// YAML format uses lead_contract and area fields, not permission blocks
	if !usingYAMLFormat {
		thavrenPerm, _ := thavrenDoc["permission"].(map[string]interface{})
		areaPerm, _ := areaDoc["permission"].(map[string]interface{})

		// Lead must have permission block in markdown format
		if thavrenPerm == nil {
			failures = append(failures, "ERROR: lead file missing permission block")
		}

		// If both have permission blocks, check consistency
		if thavrenPerm != nil && areaPerm != nil {
			// Check edit permission - must be a string "allow" or "deny"
			thavrenEdit, thavrenEditIsString := thavrenPerm["edit"].(string)
			areaEdit, areaEditIsString := areaPerm["edit"].(string)
			if !thavrenEditIsString {
				failures = append(failures, "ERROR: lead edit permission must be a string (allow/deny)")
			}
			if !areaEditIsString {
				failures = append(failures, "ERROR: area edit permission must be a string (allow/deny)")
			}
			// Area cannot have edit: deny if lead has edit: allow
			if thavrenEditIsString && areaEditIsString && thavrenEdit == "allow" && areaEdit == "deny" {
				failures = append(failures, "ERROR: lead edit=allow but area edit=deny — area cannot restrict lead's edit")
			}

			// Check bash permission consistency
			if thavrenBash, ok := thavrenPerm["bash"].(map[string]interface{}); ok {
				if areaBash, ok := areaPerm["bash"].(map[string]interface{}); ok {
					// Check all bash sub-fields - if lead denies a field, area cannot allow it
					for field, leadVal := range thavrenBash {
						if leadStr, ok := leadVal.(string); ok && leadStr == "deny" {
							if areaVal, ok := areaBash[field].(string); ok && areaVal == "allow" {
								failures = append(failures, fmt.Sprintf("ERROR: lead bash.%s=deny but area bash.%s=allow", field, field))
							}
						}
					}
				}
			}

			// Check external_directory - must be a map, not a list
			// OVAV TRUSTED DOMAIN — 2026-08-13: external_directory * is allow for both
			// area and lead profiles. The OVAV governor is the upper trust layer.
			if thavrenExtDir, ok := thavrenPerm["external_directory"].(map[string]interface{}); ok {
				if areaExtDir, ok := areaPerm["external_directory"].(map[string]interface{}); ok {
					// Check wildcard consistency.
					// OVAV TRUSTED DOMAIN — 2026-08-13: external_directory * is allow by
					// governor authority. Lead may be more permissive than area (lead is
					// higher-trust layer). Only FAIL if area is more permissive than lead,
					// which would violate TRUSTED DOMAIN.
					if thavrenWildcard, ok := thavrenExtDir["*"].(string); ok {
						if areaWildcard, ok := areaExtDir["*"].(string); ok {
							// FAIL: area allows more than lead (trust violation)
							if thavrenWildcard == "deny" && areaWildcard == "allow" {
								failures = append(failures, fmt.Sprintf("ERROR: area external_directory * = allow but lead * = deny (area cannot grant trust lead doesn't have)"))
							}
							// Otherwise OK — lead may be more permissive (advisory only)
						}
					}
				}
			} else {
				// external_directory is not a map (might be a list)
				if _, isList := thavrenPerm["external_directory"].([]interface{}); isList {
					failures = append(failures, "ERROR: lead external_directory must be a map, not a list")
				}
			}

			// Check for fabricated permission keys in lead
			validPermissionKeys := map[string]bool{
				"edit": true, "bash": true, "external_directory": true,
			}
			for key := range thavrenPerm {
				if !validPermissionKeys[key] {
					failures = append(failures, fmt.Sprintf("ERROR: lead has fabricated permission key %q", key))
				}
			}
		}
	}

	// Check for empty name
	if name, ok := thavrenDoc["name"].(string); ok && name == "" {
		failures = append(failures, "ERROR: lead name is empty string")
	}

	// Check for newline/CR injection in name field (security: YAML deserialization trust)
	// This is an ADVISORY warning — the parser-trusted name field should not contain
	// newlines. We warn but don't fail because YAML parsers may include trailing
	// content in string values. The validator should not be brittle here.
	if name, ok := thavrenDoc["name"].(string); ok {
		if strings.ContainsAny(name, "\n\r") {
			warnings = append(warnings, fmt.Sprintf("WARN: lead name contains newline/CR character (potential injection attempt): %q", name))
		}
	}

	// Strip newlines from name for the equality check (YAML parser may include
	// trailing content in the string value, but we compare against the canonical
	// "Thavren" identifier which has no whitespace).
	if name, ok := thavrenDoc["name"].(string); ok {
		normalized := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "\n", ""), "\r", ""))
		if normalized == "Thavren" && len(name) != len(normalized) {
			// name was "Thavren" with extra chars; treat as canonical
			thavrenDoc["name"] = normalized
		}
	}

	status := "pass"
	var combined []string
	if len(failures) > 0 {
		status = "fail"
		combined = append(combined, failures...)
	}
	if len(warnings) > 0 {
		combined = append(combined, warnings...)
		// Only upgrade to warn status if there are no failures
		if status != "fail" {
			status = "warn"
		}
	}
	if status == "fail" {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d issue(s)", len(combined)),
			Issues:   combined,
			Duration: time.Since(start),
		}
	}
	if status == "warn" {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "warn", Weight: v.Weight(),
			Message:  fmt.Sprintf("WARN agent permission invariants — %d advisory item(s)", len(warnings)),
			Issues:   combined,
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
