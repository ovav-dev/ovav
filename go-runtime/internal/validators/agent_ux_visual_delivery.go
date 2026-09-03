package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentUXVisualDelivery validates contract references in agent and skill files.
// Replaces: check_agent_ux_visual_delivery.py
type AgentUXVisualDelivery struct{}

func NewAgentUXVisualDelivery() *AgentUXVisualDelivery { return &AgentUXVisualDelivery{} }

func (a *AgentUXVisualDelivery) ID() string   { return "agent_ux_visual_delivery" }
func (a *AgentUXVisualDelivery) Name() string { return "Agent UX Visual Delivery Validator" }
func (a *AgentUXVisualDelivery) Description() string {
	return "Validates OVAV contract references in core agent files and response skill"
}
func (a *AgentUXVisualDelivery) Weight() int { return 8 }

var uxContracts = []string{
	".ovav/service_areas/shared/lead_work_method_contract.yaml",
	".ovav/service_areas/shared/visual_delivery_contract.yaml",
	".ovav/service_areas/shared/safe_stop_contract.yaml",
	".ovav/service_areas/shared/context_economy_contract.yaml",
}

var coreAgents = []string{
	".ovav/service_areas/platform_engineering/area_boundaries.yaml",
	".ovav/service_areas/research_intelligence/area_boundaries.yaml",
}

type termCheck struct {
	file     string
	required []string
}

var uxTermChecks = []termCheck{
	{
		file: ".ovav/service_areas/shared/lead_work_method_contract.yaml",
		required: []string{
			"receive_request", "understand_intent", "route_service_area",
			"choose_context_tier", "lead_only", "skill_only", "focused_squad",
			"full_squad", "critical_squad", "Context Gateway", "Tool Gateway",
			"produce_human_visual_delivery", "validate_as_needed", "close_with_next_action",
		},
	},
	{
		file: ".ovav/service_areas/shared/visual_delivery_contract.yaml",
		required: []string{
			"100% human", "50% shorter", "half_length_response", "result first",
			"H1", "tables or cards", "semantic labels",
			"never depend only on color", "no visible reasoning",
			"chain-of-thought", "private scratchpad",
		},
	},
	{
		file: ".ovav/service_areas/shared/safe_stop_contract.yaml",
		required: []string{
			"Safe Stop Report", "PARTIAL", "SAFE STOP", "READY FOR COMMIT",
			"Host Runtime", "OVAV Runtime", "Commit allowed", "Exact next action",
		},
	},
	{
		file: ".ovav/service_areas/shared/context_economy_contract.yaml",
		required: []string{
			"T0_none", "T1_tiny", "T2_compact", "T3_focused", "T4_full_scoped", "T5_closure_grade",
			"Do not preload full build plans", "Do not activate squads by default",
			"Research Intelligence", "Platform Engineering",
		},
	},
}

func (a *AgentUXVisualDelivery) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}

	// 1. Check contracts exist
	for _, rel := range uxContracts {
		if _, err := os.Stat(filepath.Join(root, rel)); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing contract: %s", rel))
		}
	}

	// 2. Check contract terms
	for _, tc := range uxTermChecks {
		text := strings.ToLower(readFile(tc.file))
		for _, term := range tc.required {
			if !strings.Contains(text, strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("%s missing required term: %s", tc.file, term))
			}
		}
	}

	// 3. Check core agents reference contracts
	coreAgentTerms := []string{
		"visual_delivery_contract.yaml", "safe_stop_contract.yaml",
		"context_economy_contract.yaml", "50% shorter",
		"Host Runtime", "OVAV Runtime", "no visible reasoning",
	}
	for _, rel := range coreAgents {
		text := readFile(rel)
		if text == "" {
			issues = append(issues, fmt.Sprintf("missing core agent: %s", rel))
			continue
		}
		lower := strings.ToLower(text)
		for _, term := range coreAgentTerms {
			if !strings.Contains(lower, strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("%s missing required term: %s", rel, term))
			}
		}
	}

	// 4. Check response contract skill
	respPath := ".ovav/source/skills/ovav-response-contract/SKILL.md"
	respText := readFile(respPath)
	respTerms := []string{
		"visual_delivery_contract.yaml", "half-length response",
		"no visible reasoning", "no thinking narration",
		"no chain-of-thought", "safe_stop_contract.yaml",
		"Host Runtime", "OVAV Runtime",
	}
	for _, term := range respTerms {
		if !strings.Contains(strings.ToLower(respText), strings.ToLower(term)) {
			issues = append(issues, fmt.Sprintf("%s missing required term: %s", respPath, term))
		}
	}

	if len(issues) > 0 {
		return Result{ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message: "PASS — agent UX visual delivery contracts valid", Duration: time.Since(start)}
}

var _ Validator = (*AgentUXVisualDelivery)(nil)
