package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Harness represents a supported OVAV harness target.
// Each harness has its own agent hierarchy, conversion rules, and surface expectations.
type Harness string

const (
	HarnessOpenCode   Harness = "opencode"
	HarnessClaudeCode Harness = "claude-code"
	HarnessMiMoCode   Harness = "mimocode"
	HarnessCursor     Harness = "cursor"
)

// DetectHarness determines the active harness from environment and filesystem.
// Priority: OVAV_HARNESS env var > .mimocode config > default opencode.
func DetectHarness(root string) Harness {
	if h := os.Getenv("OVAV_HARNESS"); h != "" {
		switch Harness(h) {
		case HarnessOpenCode, HarnessClaudeCode, HarnessMiMoCode, HarnessCursor:
			return Harness(h)
		}
	}
	// MiMoCode writes its config to .mimocode/global_config/config.json
	mimocodeCfg := filepath.Join(root, ".mimocode", "global_config", "config.json")
	if data, err := os.ReadFile(mimocodeCfg); err == nil {
		var cfg struct {
			Harness string `json:"harness"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Harness != "" {
			return Harness(cfg.Harness)
		}
	}
	// Default: opencode (legacy behavior)
	return HarnessOpenCode
}

// harnessAgentsDir returns the agents directory for a given harness.
// After Aug 2026 repo restructure, all harness agents live under go-runtime/internal/.
func (h Harness) agentsDir(root string) string {
	return filepath.Join(root, "go-runtime", "internal", "runtimes", string(h), "agents")
}

// AgentGovernance validates agent permission invariants, boundary law compliance,
// and agent file consistency. It is HARNESS-AWARE: validates against the
// correct harness-specific paths and expectations.
//
// For MiMoCode harness: validates areas-only (MimocodeConverter.AreasOnly=true)
// For OpenCode/ClaudeCode/Cursor: validates full hierarchy (leads + areas + teams)
type AgentGovernance struct{}

func NewAgentGovernance() *AgentGovernance { return &AgentGovernance{} }

func (a *AgentGovernance) ID() string   { return "agent_governance" }
func (a *AgentGovernance) Name() string { return "Agent Governance" }
func (a *AgentGovernance) Description() string {
	return "Validates agent permissions, boundary laws, and file consistency (harness-aware)"
}
func (a *AgentGovernance) Weight() int { return 15 }

// Required leads only apply to full-hierarchy harnesses (opencode, claude-code, cursor).
// MiMoCode harness (MimocodeConverter.AreasOnly=true) does NOT produce lead agents.
var requiredLeads = []string{
	"lead-thavren.md",
	"lead-eidren.md",
	"lead-dante.md",
	"lead-elena.md",
	"lead-sofia.md",
	"lead-uriel.md",
	"lead-valeria.md",
	"lead-renata.md",
	"lead-kenji.md",
	"lead-camila.md",
}

// Required area agents across all harnesses.
var requiredAreas = []string{
	"area-platform-engineering.md",
	"area-research-intelligence.md",
	"area-ux-design.md",
	"area-digital-product.md",
	"area-commercial-growth.md",
	"area-devops-infrastructure.md",
	"area-education-career.md",
	"area-health-performance.md",
	"area-adversarial-intelligence.md",
}

// isFullHierarchy returns true if the harness produces the full agent hierarchy
// (areas + leads + teams). MiMoCode is areas-only; all others are full hierarchy.
func (h Harness) isFullHierarchy() bool {
	return h != HarnessMiMoCode
}

func (a *AgentGovernance) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	harness := DetectHarness(root)
	agentsDir := harness.agentsDir(root)

	// 1. Verify all required area agents exist (all harnesses)
	for _, area := range requiredAreas {
		path := filepath.Join(agentsDir, area)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: Area agent %s not found in %s", area, harness))
		}
	}

	// 2. Full-hierarchy harnesses (opencode, claude-code, cursor): validate leads + teams
	// MiMoCode harness (MimocodeConverter.AreasOnly=true): skip lead/team validation
	if harness.isFullHierarchy() {
		// 2a. Verify required lead agents exist
		for _, lead := range requiredLeads {
			path := filepath.Join(agentsDir, lead)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("MISSING: Lead agent %s not found in %s", lead, harness))
			}
		}

		// 2b. Verify ovav.md exists (only for full-hierarchy harnesses)
		ovavPath := filepath.Join(agentsDir, "ovav.md")
		if _, err := os.Stat(ovavPath); os.IsNotExist(err) {
			issues = append(issues, "MISSING: ovav.md governor agent not found")
		}

		// 2c. Check each lead agent has boundary law (hard stops)
		for _, lead := range requiredLeads {
			path := filepath.Join(agentsDir, lead)
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			content := string(data)
			if !strings.Contains(content, "LO QUE NO") && !strings.Contains(content, "HARD STOP") && !strings.Contains(content, "Limitaciones") {
				issues = append(issues, fmt.Sprintf("BOUNDARY: %s missing boundary law / hard stops", lead))
			}
		}

		// 2d. Verify team agents exist for each lead (at least 2 per area)
		teamCount := 0
		teamEntries, _ := os.ReadDir(agentsDir)
		for _, e := range teamEntries {
			if strings.HasPrefix(e.Name(), "team-") {
				teamCount++
			}
		}
		if teamCount < 18 { // 9 areas × 2 minimum
			issues = append(issues, fmt.Sprintf("TEAM_COUNT: only %d team agents found (expected >= 18)", teamCount))
		}
	}

	leadCount := len(requiredLeads)
	areaCount := len(requiredAreas)
	if !harness.isFullHierarchy() {
		// MiMoCode harness: MimocodeConverter.AreasOnly=true — no lead/team hierarchy
		leadCount = 0
	}

	if len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message: fmt.Sprintf("FAIL agent governance — %d issue(s) [%s harness]", len(issues), harness),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  fmt.Sprintf("PASS agent governance — %s harness: %d leads, %d areas verified", harness, leadCount, areaCount),
		Duration: time.Since(start),
	}
}

var _ Validator = (*AgentGovernance)(nil)
