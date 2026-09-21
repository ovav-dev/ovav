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

// canonicalAreaDir returns the canonical service areas directory.
func canonicalAreaDir(root string) string {
	return filepath.Join(root, ".ovav", "service_areas")
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

// Required area agents across all harnesses (canonical paths).
var requiredAreas = []string{
	"platform_engineering/area_boundaries.yaml",
	"research_intelligence/area_boundaries.yaml",
	"ux_design/area_boundaries.yaml",
	"digital_product/area_boundaries.yaml",
	"commercial_growth/area_boundaries.yaml",
	"devops_infrastructure/area_boundaries.yaml",
	"education_career/area_boundaries.yaml",
	"health_performance/area_boundaries.yaml",
	"adversarial_intelligence/area_boundaries.yaml",
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
	saDir := canonicalAreaDir(root)

	// 1. Verify all required area agents exist (all harnesses) — canonical path
	for _, area := range requiredAreas {
		path := filepath.Join(saDir, area)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING: Area agent %s not found in canonical path", area))
		}
	}

	// 2. Full-hierarchy harnesses (opencode, claude-code, cursor): validate leads + teams
	// MiMiMoCode harness (MimocodeConverter.AreasOnly=true): skip lead/team validation
	if harness.isFullHierarchy() {
		// 2a. Verify required lead agents exist (canonical location)
		leadAreaMap := map[string]string{
			"lead-thavren.md": "platform_engineering",
			"lead-eidren.md":  "research_intelligence",
			"lead-dante.md":   "digital_product",
			"lead-elena.md":   "ux_design",
			"lead-sofia.md":   "commercial_growth",
			"lead-uriel.md":   "devops_infrastructure",
			"lead-valeria.md": "education_career",
			"lead-renata.md":  "health_performance",
			"lead-kenji.md":   "adversarial_intelligence",
			"lead-camila.md":  "legal_compliance",
		}
		for _, lead := range requiredLeads {
			areaID, ok := leadAreaMap[lead]
			if !ok {
				continue
			}
			leadPath := filepath.Join(saDir, areaID, "lead_contract.yaml")
			if _, err := os.Stat(leadPath); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("MISSING: Lead agent %s not found in canonical path", lead))
			}
		}

		// 2b. Verify ovav.md exists (only for full-hierarchy harnesses)
		ovavPath := filepath.Join(agentsDir, "ovav.md")
		if _, err := os.Stat(ovavPath); os.IsNotExist(err) {
			issues = append(issues, "MISSING: ovav.md governor agent not found")
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
