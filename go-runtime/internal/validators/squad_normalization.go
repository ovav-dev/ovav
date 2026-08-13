package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SquadNormalization validates squad, operator, and runtime governance files.
// Replaces: check_squad_normalization.py
type SquadNormalization struct{}

func NewSquadNormalization() *SquadNormalization { return &SquadNormalization{} }

func (s *SquadNormalization) ID() string   { return "squad_normalization" }
func (s *SquadNormalization) Name() string { return "Squad Normalization Validator" }
func (s *SquadNormalization) Description() string {
	return "Validates squad registry, operators, delegation rules, and runtime governance files"
}
func (s *SquadNormalization) Weight() int { return 7 }

var requiredSquadFiles = []string{
	".ovav/registry/squads.yaml",
	".ovav/registry/operators.yaml",
	".ovav/registry/service_profiles.yaml",
	".ovav/registry/delegation_rules.yaml",
	".ovav/service_areas/shared/delegation_policy.yaml",
	".ovav/service_areas/shared/squad_roles.yaml",
	".ovav/service_areas/shared/observability_policy.yaml",
	"go-runtime/internal/agents/service_area.go",
	"go-runtime/internal/agents/context.go",
	"go-runtime/internal/agents/tool_gateway.go",
	"go-runtime/internal/agents/delegation.go",
	"go-runtime/internal/agents/handoff.go",
	"go-runtime/internal/agents/observability.go",
	"go-runtime/internal/validators/context_firewall_v2.go",
}

var requiredDelegationModes = []string{
	"lead_only", "skill_only", "focused_squad", "full_squad", "critical_squad",
}

var requiredGovernanceTerms = []string{
	"Service Area Router", "Delegation Router", "Context Gateway",
	"Tool Gateway", "Delivery Contract", "Observability Trace",
}

var unsafeSquadTerms = []string{
	"always on by default", "always-on by default",
	"activate all squad by default", "activate full squad by default",
}

var requiredSquads = []string{"systems_architecture_squad", "research_intelligence_squad"}
var requiredOperators = []string{"thavren", "eidren"}

func (s *SquadNormalization) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check required files exist
	for _, path := range requiredSquadFiles {
		fullPath := filepath.Join(root, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing required file: %s", path))
		}
	}

	// 2. Read critical files
	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}

	squads := readFile(".ovav/registry/squads.yaml")
	operators := readFile(".ovav/registry/operators.yaml")
	profiles := readFile(".ovav/registry/service_profiles.yaml")
	rules := readFile(".ovav/registry/delegation_rules.yaml")
	sharedRoles := readFile(".ovav/service_areas/shared/squad_roles.yaml")
	delegationPolicy := readFile(".ovav/service_areas/shared/delegation_policy.yaml")
	router := readFile("go-runtime/internal/agents/delegation.go")
	runtimeSurface := strings.Join([]string{
		readFile("go-runtime/internal/agents/service_area.go"),
		readFile("go-runtime/internal/agents/context.go"),
		readFile("go-runtime/internal/agents/tool_gateway.go"),
		readFile("go-runtime/internal/agents/handoff.go"),
		readFile("go-runtime/internal/agents/observability.go"),
		readFile("go-runtime/internal/validators/context_firewall_v2.go"),
		readFile(".ovav/service_areas/shared/observability_policy.yaml"),
	}, "\n")

	// 3. Check delegation modes
	combinedDelegation := strings.Join([]string{squads, rules, delegationPolicy, router}, "\n")
	for _, mode := range requiredDelegationModes {
		if !strings.Contains(combinedDelegation, mode) {
			issues = append(issues, fmt.Sprintf("missing delegation mode: %s", mode))
		}
	}

	// 4. Check governed squad references
	for _, squad := range requiredSquads {
		if !strings.Contains(squads, squad) && !strings.Contains(operators, squad) && !strings.Contains(profiles, squad) {
			issues = append(issues, fmt.Sprintf("missing governed squad reference: %s", squad))
		}
	}

	// 5. Check operators
	operatorsLower := strings.ToLower(operators)
	for _, op := range requiredOperators {
		if !strings.Contains(operatorsLower, op) {
			issues = append(issues, fmt.Sprintf("operators.yaml missing %s", op))
		}
	}

	// 6. Check governance terms
	combinedSurface := strings.Join([]string{squads, rules, sharedRoles, runtimeSurface}, "\n")
	// Also scan opencode agents
	agentsDir := filepath.Join(root, "clients", "opencode", "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".md") {
				content := readFile(filepath.Join("clients", "opencode", "agents", entry.Name()))
				combinedSurface += "\n" + content
			}
		}
	}

	for _, term := range requiredGovernanceTerms {
		if !strings.Contains(combinedSurface, term) {
			issues = append(issues, fmt.Sprintf("missing runtime governance term: %s", term))
		}
	}

	// 7. Check unsafe language
	lowered := strings.ToLower(combinedSurface)
	for _, term := range unsafeSquadTerms {
		if strings.Contains(lowered, strings.ToLower(term)) {
			issues = append(issues, fmt.Sprintf("unsafe always-on squad language found: %s", term))
		}
	}

	// 8. Check delegation guard
	if !strings.Contains(rules, "do_not_delegate_when") && !strings.Contains(rules, "lead_only") {
		issues = append(issues, "delegation_rules.yaml does not prove small-task no-delegation behavior")
	}
	if !strings.Contains(delegationPolicy, "squad_usage") && !strings.Contains(router, "squad_usage") {
		issues = append(issues, "delegation policy/router missing squad_usage")
	}

	if !fileContainsAll(root, "go-runtime/internal/agents/observability.go",
		"type TraceID string", "type TraceEvent struct", "type TraceSink interface",
		"NewTraceEvent", "Validate() error", "MarshalJSON", "NewFileTraceSink",
		"NewMemoryTraceSink", "RouteRequestWithTrace") {
		issues = append(issues, "observability runtime is missing required implementation signatures")
	}

	if len(issues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message: "PASS — squad normalization valid", Duration: time.Since(start)}
}

var _ Validator = (*SquadNormalization)(nil)
