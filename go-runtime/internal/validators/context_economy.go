package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ContextEconomy validates context economy contract references across active surfaces.
// Replaces: check_context_economy_and_active_connections.py
type ContextEconomy struct{}

func NewContextEconomy() *ContextEconomy { return &ContextEconomy{} }

func (c *ContextEconomy) ID() string   { return "context_economy" }
func (c *ContextEconomy) Name() string { return "Context Economy Validator" }
func (c *ContextEconomy) Description() string {
	return "Validates context economy contract references in active agent, skill, and command surfaces"
}
func (c *ContextEconomy) Weight() int { return 8 }

var economyActiveSurfaces = []string{
	"go-runtime/internal/runtimes/opencode/agents/area-platform-engineering.md",
	"go-runtime/internal/runtimes/opencode/agents/area-research-intelligence.md",
	".ovav/source/skills/ovav-response-contract/SKILL.md",
	".ovav/source/skills/ovav-platform-session/SKILL.md",
	".ovav/source/skills/ovav-research-session/SKILL.md",
	".ovav/source/skills/ovav-repo-local-work-loop/SKILL.md",
	".ovav/source/skills/ovav-identity-guard/SKILL.md",
	".ovav/source/skills/ovav-context-pack/SKILL.md",
	".opencode/commands/ovav-work.md",
	".opencode/commands/ovav-context.md",
	".opencode/commands/ovav-validate.md",
	".opencode/commands/ovav-close.md",
}

var economyContracts = []string{
	".ovav/service_areas/shared/lead_work_method_contract.yaml",
	".ovav/service_areas/shared/visual_delivery_contract.yaml",
	".ovav/service_areas/shared/safe_stop_contract.yaml",
	".ovav/service_areas/shared/context_economy_contract.yaml",
}

var forbiddenUserVisible = regexp.MustCompile(`\b(?:Thinking|Reasoning):`)
var buildPlanLabel = regexp.MustCompile(`\b(?:BUILD|PLAN)\b`)

const (
	respContractPath = ".ovav/source/skills/ovav-response-contract/SKILL.md"
	workCmdPath      = ".opencode/commands/ovav-work.md"
	contextCmdPath   = ".opencode/commands/ovav-context.md"
)

func (c *ContextEconomy) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}

	// 1. Check economy contracts exist
	for _, rel := range economyContracts {
		if _, err := os.Stat(filepath.Join(root, rel)); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing contract: %s", rel))
		}
	}

	// 2. Check specific surface files for required terms
	workText := readFile(workCmdPath)
	contextText := readFile(contextCmdPath)
	respText := readFile(respContractPath)
	safeStopText := readFile(".ovav/service_areas/shared/safe_stop_contract.yaml")
	economyText := readFile(".ovav/service_areas/shared/context_economy_contract.yaml")

	// ovav-context command
	contextTiers := []string{"T0_none", "T1_tiny", "T2_compact", "T3_focused", "T4_full_scoped", "T5_closure_grade"}
	for _, tier := range contextTiers {
		if !strings.Contains(contextText, tier) {
			issues = append(issues, fmt.Sprintf("ovav-context missing tier: %s", tier))
		}
	}

	// ovav-work command
	workTerms := []string{"Service Area Router first", "before loading repo/internal context", "Context Economy"}
	for _, term := range workTerms {
		if !strings.Contains(workText, term) {
			issues = append(issues, fmt.Sprintf("ovav-work missing: %s", term))
		}
	}
	// Ordering checks
	sarIdx := strings.Index(workText, "Service Area Router first")
	cgIdx := strings.Index(workText, "Context Gateway")
	if sarIdx != -1 && cgIdx != -1 && sarIdx > cgIdx {
		issues = append(issues, "ovav-work orders 'Service Area Router first' after 'Context Gateway'")
	}

	// response-contract
	if !strings.Contains(respText, "visual_delivery_contract.yaml") {
		issues = append(issues, "response-contract missing visual_delivery_contract.yaml")
	}
	if !strings.Contains(respText, "no visible reasoning") {
		issues = append(issues, "response-contract missing 'no visible reasoning'")
	}

	// safe_stop
	safeStopTerms := []string{"Host Runtime", "OVAV Runtime", "routers, gateways, capsules, handoffs, validators"}
	for _, term := range safeStopTerms {
		if !strings.Contains(safeStopText, term) {
			issues = append(issues, fmt.Sprintf("safe_stop_contract missing: %s", term))
		}
	}

	// context_economy
	economyTerms := []string{"Every escalation must have a reason", "Research Intelligence",
		"must not load repo/internal OVAV context by default", "Platform Engineering"}
	for _, term := range economyTerms {
		if !strings.Contains(economyText, term) {
			issues = append(issues, fmt.Sprintf("context_economy_contract missing: %s", term))
		}
	}

	// 3. Check area agents reference contracts
	areaContracts := []string{"visual_delivery_contract.yaml", "safe_stop_contract.yaml", "context_economy_contract.yaml"}
	for _, rel := range []string{"go-runtime/internal/runtimes/opencode/agents/area-platform-engineering.md", "go-runtime/internal/runtimes/opencode/agents/area-research-intelligence.md"} {
		text := readFile(rel)
		if text == "" {
			issues = append(issues, fmt.Sprintf("missing: %s", rel))
			continue
		}
		for _, ct := range areaContracts {
			if !strings.Contains(text, ct) {
				issues = append(issues, fmt.Sprintf("%s missing contract reference: %s", rel, ct))
			}
		}
	}

	// 4. Scan team agent files (subagents — must not be always-on)
	teamDir := filepath.Join(root, ".opencode", "agents")
	if entries, err := os.ReadDir(teamDir); err == nil {
		hasTeam := false
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), "team-") {
				continue
			}
			hasTeam = true
			teamText := readFile(filepath.Join(".opencode", "agents", entry.Name()))
			lower := strings.ToLower(teamText)
			if !strings.Contains(lower, "subagent") && !strings.Contains(lower, "delegation router") {
				issues = append(issues, fmt.Sprintf("%s missing Delegation Router boundary", entry.Name()))
			}
			if !strings.Contains(lower, "not always-on") && !strings.Contains(lower, "no eres always-on") && !strings.Contains(lower, "hidden: true") {
				issues = append(issues, fmt.Sprintf("%s does not prove it is not always-on", entry.Name()))
			}
			unsafeTerms := []string{"always-on by default", "always on by default", "activate full squad by default", "activate all squad by default"}
			for _, ut := range unsafeTerms {
				if strings.Contains(lower, ut) {
					issues = append(issues, fmt.Sprintf("%s contains unsafe team default: %s", entry.Name(), ut))
				}
			}
		}
		if !hasTeam {
			issues = append(issues, "no team surfaces found")
		}
	}

	// 5. Check all active surfaces for forbidden markers
	allSurfaces := economyActiveSurfaces
	if entries, err := os.ReadDir(teamDir); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "team-") {
				allSurfaces = append(allSurfaces, filepath.Join(".opencode", "agents", entry.Name()))
			}
		}
	}
	for _, rel := range allSurfaces {
		text := readFile(rel)
		if text == "" {
			issues = append(issues, fmt.Sprintf("missing active surface: %s", rel))
			continue
		}
		if matches := forbiddenUserVisible.FindAllString(text, -1); len(matches) > 0 {
			issues = append(issues, fmt.Sprintf("%s allows user-visible output marker: %v", rel, matches))
		}
		if matches := buildPlanLabel.FindAllStringIndex(text, -1); len(matches) > 0 {
			for _, m := range matches {
				line := strings.Count(text[:m[0]], "\n") + 1
				word := text[m[0]:m[1]]
				issues = append(issues, fmt.Sprintf("%s:%d contains active segment label: %s", rel, line, word))
			}
		}
	}

	if len(issues) > 0 {
		return Result{ID: c.ID(), Name: c.Name(), Status: "fail", Weight: c.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: c.ID(), Name: c.Name(), Status: "pass", Weight: c.Weight(),
		Message: "PASS — context economy and active connections valid", Duration: time.Since(start)}
}

var _ Validator = (*ContextEconomy)(nil)
