package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// F2Infrastructure validates F2 Infrastructure Governance:
//   - System path rules exist
//   - Plugin governance config exists with security fields
//   - Live behavior config exists
//   - Permission authority JSON is valid
//   - Claims enforcement file exists
//
// Replaces: tools/validators/check_f2_infrastructure.py
type F2Infrastructure struct{}

func NewF2Infrastructure() *F2Infrastructure { return &F2Infrastructure{} }

func (v *F2Infrastructure) ID() string   { return "f2_infrastructure" }
func (v *F2Infrastructure) Name() string { return "F2 Infrastructure Governance" }
func (v *F2Infrastructure) Description() string {
	return "Validates F2 infrastructure: system paths, plugin governance, live behavior, config authority, claims"
}
func (v *F2Infrastructure) Weight() int { return 8 }

func (v *F2Infrastructure) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. System path rules
	sysPathRules := filepath.Join(root, ".ovav", "governance", "system_path_rules.yaml")
	if _, err := os.Stat(sysPathRules); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/governance/system_path_rules.yaml — system path governance rules not found")
	} else {
		data, err := os.ReadFile(sysPathRules)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read system_path_rules.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: system_path_rules.yaml invalid YAML: %v", err))
			}
		}
	}

	// 2. Plugin governance — opencode.json or opencode.jsonc must exist with security fields
	pluginConfigFound := false
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		cfgPath := filepath.Join(root, name)
		if _, err := os.Stat(cfgPath); err == nil {
			pluginConfigFound = true
			data, err := os.ReadFile(cfgPath)
			if err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: Cannot read %s: %v", name, err))
				break
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: %s invalid JSON: %v", name, err))
				break
			}
			// Check for security-relevant fields
			securityFields := []string{"security", "blocked_surfaces", "permissions", "plugin"}
			hasSecurityField := false
			for _, field := range securityFields {
				if _, ok := cfg[field]; ok {
					hasSecurityField = true
					break
				}
			}
			if !hasSecurityField {
				issues = append(issues, fmt.Sprintf("WARNING: %s missing security governance fields (security, blocked_surfaces, permissions, or plugin)", name))
			}
			break
		}
	}
	if !pluginConfigFound {
		issues = append(issues, "MISSING: opencode.json or opencode.jsonc — plugin governance config not found")
	}

	// 3. Live behavior config
	liveBehaviorPath := filepath.Join(root, ".ovav", "governance", "live_behavior.yaml")
	if _, err := os.Stat(liveBehaviorPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/governance/live_behavior.yaml — live behavior config not found")
	} else {
		data, err := os.ReadFile(liveBehaviorPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read live_behavior.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: live_behavior.yaml invalid YAML: %v", err))
			}
		}
	}

	// 4. Config authority — permission_authority.json
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if _, err := os.Stat(paPath); os.IsNotExist(err) {
		issues = append(issues, "CRITICAL: .ovav/policy/permission_authority.json — config authority not found")
	} else {
		data, err := os.ReadFile(paPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("CRITICAL: Cannot read permission_authority.json: %v", err))
		} else {
			var pa map[string]interface{}
			if err := json.Unmarshal(data, &pa); err != nil {
				issues = append(issues, fmt.Sprintf("CRITICAL: permission_authority.json invalid JSON: %v", err))
			} else {
				// Verify infrastructure_surfaces section
				if _, ok := pa["infrastructure_surfaces"]; !ok {
					issues = append(issues, "ERROR: permission_authority.json missing 'infrastructure_surfaces' section")
				}
			}
		}
	}

	// 5. Claims enforcement
	claimsPath := filepath.Join(root, ".ovav", "policy", "claims.yaml")
	if _, err := os.Stat(claimsPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/policy/claims.yaml — claims enforcement file not found")
	} else {
		data, err := os.ReadFile(claimsPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read claims.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: claims.yaml invalid YAML: %v", err))
			}
		}
	}

	hasCritical := false
	for _, i := range issues {
		if strings.HasPrefix(i, "CRITICAL:") {
			hasCritical = true
			break
		}
	}

	if hasCritical || len(issues) > 0 {
		status := "fail"
		msg := fmt.Sprintf("FAIL F2 infrastructure governance — %d issue(s)", len(issues))
		return Result{
			ID:       v.ID(),
			Name:     v.Name(),
			Status:   status,
			Weight:   v.Weight(),
			Message:  msg,
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID:       v.ID(),
		Name:     v.Name(),
		Status:   "pass",
		Weight:   v.Weight(),
		Message:  "PASS F2 infrastructure governance — all config files present and valid",
		Duration: time.Since(start),
	}
}

var _ Validator = (*F2Infrastructure)(nil)
