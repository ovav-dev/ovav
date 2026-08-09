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

// F3Roles validates F3 Role Governance:
//   - Lead agent files have valid YAML frontmatter with mode, hidden, description
//   - Team agent files have mode: subagent
//   - Research profile, sandbox rules, and temporal limits exist
//
// Replaces: tools/validators/check_f3_roles.py
type F3Roles struct{}

func NewF3Roles() *F3Roles { return &F3Roles{} }

func (v *F3Roles) ID() string   { return "f3_roles" }
func (v *F3Roles) Name() string { return "F3 Role Governance" }
func (v *F3Roles) Description() string {
	return "Validates F3 roles: lead/team agent frontmatter, research profile, sandbox rules, temporal limits"
}
func (v *F3Roles) Weight() int { return 8 }

// requiredLeadFields are the YAML frontmatter fields required for lead agents.
var requiredLeadFields = []string{"mode", "hidden", "description"}

func (v *F3Roles) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	agentsDir := filepath.Join(root, "go-runtime", "internal", "runtimes", "opencode", "agents")

	// 1. Validate lead agent files
	issues = append(issues, v.validateLeadAgents(agentsDir)...)

	// 2. Validate team agent files
	issues = append(issues, v.validateTeamAgents(agentsDir)...)

	// 3. Research profile rules
	rpPath := filepath.Join(root, ".ovav", "governance", "research_profile.yaml")
	if _, err := os.Stat(rpPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/governance/research_profile.yaml — research profile rules not found")
	} else {
		data, err := os.ReadFile(rpPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read research_profile.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: research_profile.yaml invalid YAML: %v", err))
			}
		}
	}

	// 4. Sandbox governance
	sbPath := filepath.Join(root, ".ovav", "governance", "sandbox_rules.yaml")
	if _, err := os.Stat(sbPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/governance/sandbox_rules.yaml — sandbox governance rules not found")
	} else {
		data, err := os.ReadFile(sbPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read sandbox_rules.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: sandbox_rules.yaml invalid YAML: %v", err))
			}
		}
	}

	// 5. Temporal/scoped limits
	tlPath := filepath.Join(root, ".ovav", "governance", "temporal_limits.yaml")
	if _, err := os.Stat(tlPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .ovav/governance/temporal_limits.yaml — temporal/scoped limits not found")
	} else {
		data, err := os.ReadFile(tlPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: Cannot read temporal_limits.yaml: %v", err))
		} else {
			var node yaml.Node
			if err := yaml.Unmarshal(data, &node); err != nil {
				issues = append(issues, fmt.Sprintf("ERROR: temporal_limits.yaml invalid YAML: %v", err))
			}
		}
	}

	// 6. Check permission_authority.json for role_surfaces
	paPath := filepath.Join(root, ".ovav", "policy", "permission_authority.json")
	if data, err := os.ReadFile(paPath); err == nil {
		var pa map[string]interface{}
		if err := yaml.Unmarshal(data, &pa); err == nil { // yaml.v3 can parse JSON too
			if _, ok := pa["role_surfaces"]; !ok {
				issues = append(issues, "ERROR: permission_authority.json missing 'role_surfaces' section")
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
		msg := fmt.Sprintf("FAIL F3 role governance — %d issue(s)", len(issues))
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
		Message:  "PASS F3 role governance — all agent files, role configs valid",
		Duration: time.Since(start),
	}
}

// validateLeadAgents checks that all lead-*.md files have valid YAML frontmatter
// with required fields: mode, hidden, description.
func (v *F3Roles) validateLeadAgents(agentsDir string) []string {
	var issues []string
	pattern := filepath.Join(agentsDir, "lead-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		// Try alternate patterns
		matches2, _ := filepath.Glob(filepath.Join(agentsDir, "area-*.md"))
		matches = matches2
	}
	if len(matches) == 0 {
		issues = append(issues, "ERROR: No lead agent files found (lead-*.md or area-*.md in go-runtime/internal/runtimes/opencode/agents/)")
		return issues
	}

	for _, f := range matches {
		fileIssues := v.validateAgentFrontmatter(f, requiredLeadFields, "lead")
		issues = append(issues, fileIssues...)
	}
	return issues
}

// validateTeamAgents checks that all team-*.md files have mode: subagent.
func (v *F3Roles) validateTeamAgents(agentsDir string) []string {
	var issues []string
	pattern := filepath.Join(agentsDir, "team-*.md")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return nil // Not an error — team agents are optional
	}

	for _, f := range matches {
		fm, err := parseFrontmatter(f)
		if err != nil {
			issues = append(issues, fmt.Sprintf("ERROR: %s — invalid YAML frontmatter: %v", filepath.Base(f), err))
			continue
		}
		if fm == nil {
			issues = append(issues, fmt.Sprintf("ERROR: %s — missing YAML frontmatter (must start with ---)", filepath.Base(f)))
			continue
		}
		mode, ok := fm["mode"].(string)
		if !ok || mode != "subagent" {
			issues = append(issues, fmt.Sprintf("ERROR: %s — mode must be 'subagent' (got: %v)", filepath.Base(f), fm["mode"]))
		}
	}
	return issues
}

// validateAgentFrontmatter checks a single agent file for required frontmatter fields.
func (v *F3Roles) validateAgentFrontmatter(path string, requiredFields []string, agentType string) []string {
	var issues []string
	base := filepath.Base(path)

	fm, err := parseFrontmatter(path)
	if err != nil {
		issues = append(issues, fmt.Sprintf("ERROR: %s — invalid YAML frontmatter: %v", base, err))
		return issues
	}
	if fm == nil {
		issues = append(issues, fmt.Sprintf("ERROR: %s — missing YAML frontmatter (%s agents must start with --- delimited frontmatter)", base, agentType))
		return issues
	}

	for _, field := range requiredFields {
		if _, ok := fm[field]; !ok {
			issues = append(issues, fmt.Sprintf("ERROR: %s — missing required frontmatter field '%s'", base, field))
		}
	}

	// Validate mode field specifically
	if mode, ok := fm["mode"].(string); ok {
		if agentType == "lead" && mode == "subagent" {
			issues = append(issues, fmt.Sprintf("ERROR: %s — lead agent has mode='subagent' (should be a lead mode)", base))
		}
	}

	return issues
}

// parseFrontmatter extracts YAML frontmatter from a markdown file.
// Frontmatter is delimited by --- at the start and end.
// Returns nil if no frontmatter is found.
func parseFrontmatter(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	// Frontmatter must start with --- on the first line
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return nil, nil
	}

	// Find closing ---
	rest := content[3:] // skip first ---
	// Handle Windows line endings
	rest = strings.TrimPrefix(rest, "\r")
	if !strings.HasPrefix(rest, "\n") {
		return nil, nil
	}
	rest = rest[1:] // skip \n

	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		// Try without leading newline
		endIdx = strings.Index(rest, "---")
	}
	if endIdx == -1 {
		return nil, fmt.Errorf("unclosed frontmatter (missing closing ---)")
	}

	yamlContent := rest[:endIdx]

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &result); err != nil {
		return nil, err
	}

	return result, nil
}

var _ Validator = (*F3Roles)(nil)
