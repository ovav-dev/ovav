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

var requiredPermissionKeys = map[string]bool{
	"edit":               true,
	"bash":               true,
	"external_directory": true,
}

// parseAgentFrontmatter reads the YAML frontmatter from an agent .md file.
func parseAgentFrontmatter(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil, fmt.Errorf("missing YAML frontmatter")
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("malformed frontmatter")
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, fmt.Errorf("YAML parse error: %w", err)
	}
	return fm, nil
}

func (v *AgentPermissionInvariants) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	thavrenPath := filepath.Join(root, "clients", "opencode", "agents", "lead-thavren.md")
	areaPath := filepath.Join(root, "clients", "opencode", "agents", "area-platform-engineering.md")

	thavrenData, err := parseAgentFrontmatter(thavrenPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse lead-thavren.md: %v", err))
	}
	areaData, err := parseAgentFrontmatter(areaPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("CRITICAL: Cannot parse area-platform-engineering.md: %v", err))
	}

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// Verify names
	if name, ok := thavrenData["name"].(string); !ok || name != "Thavren" {
		issues = append(issues, "ERROR: lead-thavren.md name must be 'Thavren'")
	}
	if name, ok := areaData["name"].(string); !ok || name != "Platform Engineering" {
		issues = append(issues, "ERROR: area-platform-engineering.md name must be 'Platform Engineering'")
	}

	// Extract permission blocks
	thavrenPerm, _ := thavrenData["permission"].(map[string]interface{})
	areaPerm, _ := areaData["permission"].(map[string]interface{})

	if thavrenPerm == nil {
		issues = append(issues, "ERROR: lead-thavren.md missing permission block in frontmatter")
	}
	if areaPerm == nil {
		issues = append(issues, "ERROR: area-platform-engineering.md missing permission block in frontmatter")
	}

	if thavrenPerm == nil || areaPerm == nil {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL agent permission invariants — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	// Check required permission keys
	thavrenKeys := mapKeys(thavrenPerm)
	areaKeys := mapKeys(areaPerm)
	if !setsEqual(thavrenKeys, requiredPermissionKeys) {
		issues = append(issues, fmt.Sprintf("ERROR: lead-thavren.md permission keys drifted: %v", sortedKeys(thavrenKeys)))
	}
	if !setsEqual(areaKeys, requiredPermissionKeys) {
		issues = append(issues, fmt.Sprintf("ERROR: area-platform-engineering.md permission keys drifted: %v", sortedKeys(areaKeys)))
	}

	// edit must be "allow" for both
	if edit, ok := thavrenPerm["edit"].(string); !ok || edit != "allow" {
		issues = append(issues, "ERROR: lead-thavren.md permission.edit must be 'allow'")
	}
	if edit, ok := areaPerm["edit"].(string); !ok || edit != "allow" {
		issues = append(issues, "ERROR: area-platform-engineering.md permission.edit must be 'allow'")
	}

	// bash permissions must be identical
	thavrenBash, _ := thavrenPerm["bash"].(map[string]interface{})
	areaBash, _ := areaPerm["bash"].(map[string]interface{})
	if thavrenBash == nil {
		issues = append(issues, "ERROR: lead-thavren.md permission.bash must be a mapping")
	}
	if areaBash == nil {
		issues = append(issues, "ERROR: area-platform-engineering.md permission.bash must be a mapping")
	}
	if thavrenBash != nil && areaBash != nil {
		if !mapsEqual(thavrenBash, areaBash) {
			issues = append(issues, "ERROR: Platform Engineering and Thavren bash permissions must be identical")
		}
	}

	// external_directory checks
	thavrenExt, _ := thavrenPerm["external_directory"].(map[string]interface{})
	areaExt, _ := areaPerm["external_directory"].(map[string]interface{})
	if thavrenExt == nil {
		issues = append(issues, "ERROR: lead-thavren.md external_directory must be a mapping")
	}
	if areaExt == nil {
		issues = append(issues, "ERROR: area-platform-engineering.md external_directory must be a mapping")
	}

	if thavrenExt != nil {
		if wildcard, ok := thavrenExt["*"].(string); !ok || wildcard != "allow" {
			issues = append(issues, "ERROR: lead-thavren.md external_directory '*' must be 'allow'")
		}
	}
	if areaExt != nil {
		if wildcard, ok := areaExt["*"].(string); !ok || wildcard != "deny" {
			issues = append(issues, "ERROR: area-platform-engineering.md external_directory '*' must be 'deny'")
		}
		// Area must have at least one explicit allow (besides *)
		hasExplicitAllow := false
		for k, v := range areaExt {
			if k != "*" {
				if s, ok := v.(string); ok && s == "allow" {
					hasExplicitAllow = true
					break
				}
			}
		}
		if !hasExplicitAllow {
			issues = append(issues, "ERROR: area-platform-engineering.md must have at least one explicit external_directory allow")
		}
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

// ── helpers ──────────────────────────────────────────────────────────────────

func mapKeys(m map[string]interface{}) map[string]bool {
	result := make(map[string]bool, len(m))
	for k := range m {
		result[k] = true
	}
	return result
}

func setsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple bubble sort for small key sets
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", av) != fmt.Sprintf("%v", bv) {
			return false
		}
	}
	return true
}

var _ Validator = (*AgentPermissionInvariants)(nil)
