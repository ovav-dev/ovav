package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ToolReadiness validates tool readiness matrix, capability boundary, and surface constraints.
// Replaces: check_tool_readiness_matrix.py
type ToolReadiness struct{}

func NewToolReadiness() *ToolReadiness { return &ToolReadiness{} }

func (t *ToolReadiness) ID() string   { return "tool_readiness" }
func (t *ToolReadiness) Name() string { return "Tool Readiness Matrix Validator" }
func (t *ToolReadiness) Description() string {
	return "Validates tool readiness matrix, capability boundary, and surface constraints"
}
func (t *ToolReadiness) Weight() int { return 12 }

var requiredCapabilities = []string{
	"memory_gateway", "mcp", "a2a", "sdd_artifact_first",
	"external_adapters", "install_apply_deploy", "plugin_installation",
	"global_config_writes", "web_browser_api_connectors", "vector_rag_stores",
	"observability_evals", "opencode_ux_extensions", "tui_ui",
	"package_manager_installs", "protected_internal_agent_aliases",
}

var validStates = map[string]bool{
	"active_internal": true, "active_internal_gated": true,
	"blocked": true, "gated_candidate": true,
	"future_candidate": true, "historical_only": true, "deprecated": true,
}

var notActiveByDefault = map[string]bool{
	"mcp": true, "a2a": true,
	"tui_ui": true, "plugin_installation": true, "global_config_writes": true,
}

func (t *ToolReadiness) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check tool_readiness_matrix.yaml exists and valid
	matrixPath := filepath.Join(root, ".ovav", "service_areas", "shared", "tool_readiness_matrix.yaml")
	if _, err := os.Stat(matrixPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: tool_readiness_matrix.yaml")
	} else {
		data, err := os.ReadFile(matrixPath)
		if err == nil {
			var matrix map[string]interface{}
			if err := yaml.Unmarshal(data, &matrix); err != nil {
				issues = append(issues, fmt.Sprintf("tool_readiness_matrix.yaml: YAML parse error: %v", err))
			} else {
				wrapper, ok := matrix["tool_readiness_matrix"].(map[string]interface{})
				if !ok {
					issues = append(issues, "tool_readiness_matrix.yaml: missing top-level tool_readiness_matrix mapping")
				} else {
					caps, ok := wrapper["capabilities"].(map[string]interface{})
					if !ok {
						issues = append(issues, "tool_readiness_matrix.yaml: missing capabilities mapping")
					} else {
						// Check required capabilities
						for _, cap := range requiredCapabilities {
							if _, ok := caps[cap]; !ok {
								issues = append(issues, fmt.Sprintf("TOOL_MATRIX: missing required capability: %s", cap))
							}
						}
						// Check not-active-by-default
						for capName := range notActiveByDefault {
							if entry, ok := caps[capName].(map[string]interface{}); ok {
								state, _ := entry["current_state"].(string)
								if state == "active_internal" || state == "active_internal_gated" {
									issues = append(issues, fmt.Sprintf("TOOL_MATRIX: %s must not be active by default (current: %s)", capName, state))
								}
							}
						}
						// Check valid states
						for name, entry := range caps {
							if entryMap, ok := entry.(map[string]interface{}); ok {
								state, _ := entryMap["current_state"].(string)
								if state != "" && !validStates[state] {
									issues = append(issues, fmt.Sprintf("TOOL_MATRIX: %s has invalid state: %s", name, state))
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Check advanced_capability_boundary.yaml
	boundaryPath := filepath.Join(root, ".ovav", "service_areas", "shared", "advanced_capability_boundary.yaml")
	if _, err := os.Stat(boundaryPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: advanced_capability_boundary.yaml")
	} else {
		data, err := os.ReadFile(boundaryPath)
		if err == nil {
			var boundary map[string]interface{}
			if yaml.Unmarshal(data, &boundary) == nil {
				wrapper, _ := boundary["advanced_capability_boundary"].(map[string]interface{})
				if wrapper != nil {
					core, _ := wrapper["core_independence"].(map[string]interface{})
					if core != nil {
						if independent, _ := core["ovav_core_does_not_depend_on_advanced_tools"].(bool); !independent {
							issues = append(issues, "BOUNDARY: core independence not declared")
						}
					}
				}
			}
		}
	}

	// 3. Check package install denies in opencode.json
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		var config map[string]interface{}
		if json.Unmarshal(data, &config) == nil {
			if perms, ok := config["permission"].(map[string]interface{}); ok {
				if bashPerms, ok := perms["bash"].(map[string]interface{}); ok {
					for _, cmd := range []string{"pip install *", "npm install *", "apt install *"} {
						if deny, ok := bashPerms[cmd].(string); !ok || deny != "deny" {
							issues = append(issues, fmt.Sprintf("opencode.json: must deny '%s' in bash permissions", cmd))
						}
					}
				}
			}
		}
	}

	// 4. Check for dangerous active surface claims in agent files
	dangerousPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(mcp|a2a)\b[^\n]{0,80}\b(active|available|enabled)\b`),
	}
	agentsDir := filepath.Join(root, ".opencode", "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(agentsDir, e.Name()))
			if err != nil {
				continue
			}
			text := string(content)
			for _, pat := range dangerousPatterns {
				if pat.MatchString(text) {
					boundaryWords := regexp.MustCompile(`(?i)\b(blocked|denied|gated|boundary)\b`)
					if !boundaryWords.MatchString(text) {
						issues = append(issues, fmt.Sprintf("SURFACE: %s mentions advanced capability without boundary language", e.Name()))
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: t.ID(), Name: t.Name(), Status: "fail", Weight: t.Weight(),
			Message:  fmt.Sprintf("FAIL tool readiness — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: t.ID(), Name: t.Name(), Status: "pass", Weight: t.Weight(),
		Message:  "PASS tool readiness — matrix and boundary verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*ToolReadiness)(nil)
