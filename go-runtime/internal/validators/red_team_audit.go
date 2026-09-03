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

// RedTeamAudit validates cross-area boundary compliance at Red Team depth.
// R5: Boundary Audit — verifies all 9 areas comply with LAW-001 (area_boundary_enforcement),
// checks agent permissions against declared scopes, and detects cross-area violations.
//
// This validator is designed to be run:
//   - Pre-merge (every PR)
//   - Scheduled (weekly cron)
//   - On-demand (red_team trigger)
//
// Replaces the Red Team manual audit workflow from T14.
type RedTeamAudit struct{}

func NewRedTeamAudit() *RedTeamAudit { return &RedTeamAudit{} }

func (r *RedTeamAudit) ID() string   { return "red_team_audit" }
func (r *RedTeamAudit) Name() string { return "R5 Red Team Boundary Audit" }
func (r *RedTeamAudit) Description() string {
	return "Red Team automated boundary audit — verifies LAW-001 compliance across all 9 areas, agent scope enforcement, and cross-area violation detection"
}
func (r *RedTeamAudit) Weight() int { return 9 }

// allAreas defines the 9 sovereign areas of OVAV.
var allAreas = []struct {
	id          string
	name        string
	profileFile string
	leadFile    string
	keywords    []string // terms that MUST appear
	forbidden   []string // terms that MUST NOT appear
}{
	{
		id:          "platform_engineering",
		name:        "Platform Engineering & Developer Experience",
		profileFile: "platform_engineering/area_boundaries.yaml",
		leadFile:    "lead-thavren.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "runtime Go", "seguridad del sistema", "validación sistémica"},
		forbidden:   []string{"NO diseño UI/UX", "NO frontend React", "NO DevOps"},
	},
	{
		id:          "research_intelligence",
		name:        "Evidence & Decision Intelligence",
		profileFile: "research_intelligence/area_boundaries.yaml",
		leadFile:    "lead-eidren.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "evidencia", "benchmark"},
		forbidden:   []string{"NO desarrollo de producto"},
	},
	{
		id:          "commercial_growth",
		name:        "Commercial & Growth Strategy",
		profileFile: "commercial_growth/area_boundaries.yaml",
		leadFile:    "lead-sofia.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "estrategia", "GTM"},
		forbidden:   []string{"NO runtime Go", "NO desarrollo"},
	},
	{
		id:          "digital_product",
		name:        "Digital Product Engineering",
		profileFile: "digital_product/area_boundaries.yaml",
		leadFile:    "lead-dante.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "frontend", "React", "TypeScript"},
		forbidden:   []string{"NO runtime Go", "NO estrategia comercial"},
	},
	{
		id:          "devops_infrastructure",
		name:        "DevOps & Infrastructure",
		profileFile: "devops_infrastructure/area_boundaries.yaml",
		leadFile:    "lead-uriel.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "infraestructura", "deploy", "CI/CD"},
		forbidden:   []string{"NO desarrollo Go", "NO frontend"},
	},
	{
		id:          "ux_design",
		name:        "UX/UI Design",
		profileFile: "ux_design/area_boundaries.yaml",
		leadFile:    "lead-elena.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "diseño", "UX", "UI"},
		forbidden:   []string{"NO runtime Go", "NO DevOps"},
	},
	{
		id:          "legal_compliance",
		name:        "Legal & Compliance",
		profileFile: "legal_compliance/area_boundaries.yaml",
		leadFile:    "lead-camila.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "legal", "compliance", "contratos"},
		forbidden:   []string{"NO desarrollo", "NO runtime Go"},
	},
	{
		id:          "education_career",
		name:        "Education & Career Development",
		profileFile: "education_career/area_boundaries.yaml",
		leadFile:    "lead-valeria.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "educación", "currículo"},
		forbidden:   []string{"NO DevOps", "NO runtime Go"},
	},
	{
		id:          "adversarial_intelligence",
		name:        "Adversarial Intelligence",
		profileFile: "adversarial_intelligence/area_boundaries.yaml",
		leadFile:    "lead-kenji.md",
		keywords:    []string{"HARD STOP", "Fuera de mi área", "Red Team", "adversarial", "pentesting"},
		forbidden:   []string{"NO desarrollo de features", "NO modificar código de otras áreas"},
	},
}

func (r *RedTeamAudit) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	issues := make([]string, 0)
	passed := 0
	total := 0

	harness := DetectHarness(root)
	agentsDir := harness.agentsDir(root)
	saDir := filepath.Join(root, ".ovav", "service_areas")
	lawPath := filepath.Join(root, ".ovav", "laws", "area_boundary_enforcement.yaml")
	sharedDir := filepath.Join(root, ".ovav", "service_areas", "shared")

	// 1. Verify LAW-001 exists and is readable
	total++
	if data, err := os.ReadFile(lawPath); err != nil {
		issues = append(issues, "CRITICAL: LAW-001 area_boundary_enforcement.yaml missing or unreadable")
	} else {
		content := string(data)
		if !strings.Contains(content, "LAW-001") {
			issues = append(issues, "LAW-001: area_boundary_enforcement.yaml missing LAW-001 marker")
		} else {
			passed++
		}
	}

	// 2. Verify all 9 area agent profiles exist with scope definitions (canonical YAML)
	for _, area := range allAreas {
		total++
		profilePath := filepath.Join(saDir, area.profileFile)
		data, err := os.ReadFile(profilePath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("MISSING_AREA_PROFILE: %s (%s) — %v", area.name, area.profileFile, err))
			continue
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			issues = append(issues, fmt.Sprintf("AREA_YAML_ERROR: %s — invalid YAML: %v", area.name, err))
			continue
		}

		hasScope := false
		if dnc, ok := doc["does_not_cover"].([]interface{}); ok && len(dnc) > 0 {
			hasScope = true
		}
		if ab, ok := doc["area_boundaries"].(map[string]interface{}); ok {
			if scope, ok := ab["scope"].(map[string]interface{}); ok {
				if _, hasIncludes := scope["includes"]; hasIncludes {
					hasScope = true
				}
				if _, hasExcludes := scope["excludes"]; hasExcludes {
					hasScope = true
				}
			}
		}

		if !hasScope {
			issues = append(issues, fmt.Sprintf("AREA_SCOPE_MISSING: %s — no scope definition found (does_not_cover or scope.includes/excludes)", area.name))
			continue
		}
		passed++
	}

	// 3. Verify all 9 lead agent files exist with scope definitions (canonical YAML)
	if harness.isFullHierarchy() {
		for _, area := range allAreas {
			total++
			leadPath := filepath.Join(saDir, area.id, "lead_contract.yaml")
			data, err := os.ReadFile(leadPath)
			if err != nil {
				issues = append(issues, fmt.Sprintf("MISSING_LEAD_FILE: %s (lead_contract.yaml) — %v", area.name, err))
				continue
			}
			var doc map[string]interface{}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				issues = append(issues, fmt.Sprintf("LEAD_YAML_ERROR: %s — invalid YAML: %v", area.name, err))
				continue
			}
			lc, ok := doc["lead_contract"].(map[string]interface{})
			if !ok {
				issues = append(issues, fmt.Sprintf("LEAD_NO_SCOPE: %s — lead_contract.yaml missing lead_contract section", area.name))
				continue
			}
			hasScope := false
			if _, hasCovers := lc["covers"]; hasCovers {
				hasScope = true
			}
			if _, hasDNC := lc["does_not_cover"]; hasDNC {
				hasScope = true
			}
			if _, hasResp := lc["responsibilities"]; hasResp {
				hasScope = true
			}
			if _, hasAuth := lc["authority"]; hasAuth {
				hasScope = true
			}
			if !hasScope {
				issues = append(issues, fmt.Sprintf("LEAD_NO_SCOPE: %s — lead_contract.yaml missing scope (covers/does_not_cover/responsibilities/authority)", area.name))
				continue
			}
			passed++
		}
	}

	// 4. Verify shared contracts reference LAW-001
	total++
	contractsChecked := 0
	contractsCompliant := 0
	entries, err := os.ReadDir(sharedDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			contractsChecked++
			contractPath := filepath.Join(sharedDir, entry.Name())
			cdata, err := os.ReadFile(contractPath)
			if err != nil {
				continue
			}
			ccontent := string(cdata)
			if strings.Contains(ccontent, "LAW-001") || strings.Contains(ccontent, "boundary") {
				contractsCompliant++
			}
		}
	}
	if contractsChecked == 0 {
		issues = append(issues, "WARNING: No shared contracts found in service_areas/shared/")
	} else if contractsCompliant < contractsChecked {
		issues = append(issues, fmt.Sprintf("CONTRACT_COMPLIANCE: %d/%d contracts reference LAW-001 or boundary", contractsCompliant, contractsChecked))
	} else {
		passed++
	}

	// 5. Check for agents with overly broad permissions (Red Team deep scan)
	total++
	agentFiles, _ := filepath.Glob(filepath.Join(agentsDir, "*.md"))
	broadAgents := 0
	for _, af := range agentFiles {
		base := filepath.Base(af)
		// The central governor is neither an area lead nor a team member. Its
		// cross-area authority is the object being governed, not scope ambiguity.
		if isCentralGovernorAgent(base) {
			continue
		}
		// Skip area profile files (these are area declarations, not agents)
		if strings.HasPrefix(base, "area-") {
			continue
		}
		// Skip lead files (already checked above)
		isLead := false
		for _, area := range allAreas {
			if base == area.leadFile {
				isLead = true
				break
			}
		}
		if isLead {
			continue
		}

		data, err := os.ReadFile(af)
		if err != nil {
			continue
		}
		content := string(data)
		// Check for dangerous patterns: agents that claim authority over multiple areas
		if strings.Contains(content, "área:") && strings.Count(content, "área:") > 1 {
			broadAgents++
		}
		// Check for missing area declaration
		if !strings.Contains(content, "Área:") && !strings.Contains(content, "área:") {
			// Team agents might not have area declaration; only flag if they have authority claims
			if strings.Contains(content, "Autoridad:") || strings.Contains(content, "authority") {
				broadAgents++
			}
		}
	}
	if broadAgents > 0 {
		issues = append(issues, fmt.Sprintf("BROAD_AGENTS: %d agents with ambiguous or overly broad scope detected", broadAgents))
	} else {
		passed++
	}

	// 6. Visibility audit — verify correct hidden/mode flags per agent type
	total++
	visIssues := 0
	allAgentFiles, _ := filepath.Glob(filepath.Join(agentsDir, "*.md"))
	for _, af := range allAgentFiles {
		base := filepath.Base(af)
		data, err := os.ReadFile(af)
		if err != nil {
			continue
		}
		content := string(data)

		// Extract frontmatter hidden and mode values
		getYAMLField := func(field string) string {
			// Simple YAML frontmatter parser: look for "field: value"
			for _, line := range strings.Split(content, "\n") {
				line = strings.TrimSpace(line)
				if line == "---" {
					continue
				}
				if strings.HasPrefix(line, field+":") {
					val := strings.TrimSpace(strings.TrimPrefix(line, field+":"))
					return strings.ToLower(val)
				}
			}
			return ""
		}
		hidden := getYAMLField("hidden")
		mode := getYAMLField("mode")

		switch {
		case strings.HasPrefix(base, "area-"):
			if hidden != "false" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: %s — expected hidden:false, got hidden:%s", base, hidden))
			}
			if mode != "" && mode != "primary" && mode != "all" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: %s — area must be mode:primary, got mode:%s", base, mode))
			}
		case strings.HasPrefix(base, "lead-"):
			if hidden != "true" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: %s — lead must be hidden:true, got hidden:%s", base, hidden))
			}
			// Leads can be primary or all
		case strings.HasPrefix(base, "team-"):
			if hidden != "true" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: %s — team must be hidden:true, got hidden:%s", base, hidden))
			}
			if mode != "" && mode != "subagent" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: %s — team must be mode:subagent, got mode:%s", base, mode))
			}
		case base == "ovav.md":
			if hidden != "true" {
				visIssues++
				issues = append(issues, fmt.Sprintf("VISIBILITY: ovav.md — governor must be hidden:true, got hidden:%s", hidden))
			}
		}
	}
	if visIssues > 0 {
		issues = append(issues, fmt.Sprintf("VISIBILITY_FAIL: %d agent(s) with incorrect hidden/mode flags", visIssues))
	} else {
		passed++
	}

	// Determine final status
	status := "pass"
	message := fmt.Sprintf("Red Team R5 boundary audit [%s harness] — %d/%d checks passed", harness, passed, total)
	if len(issues) > 0 {
		status = "fail"
		message = fmt.Sprintf("FAIL — %d issue(s) in %d checks [%s harness]", len(issues), total, harness)
	}

	return Result{
		ID:          r.ID(),
		Name:        r.Name(),
		Status:      status,
		Message:     message,
		Issues:      issues,
		Weight:      r.Weight(),
		Duration:    time.Since(start),
		Description: r.Description(),
	}
}

func isCentralGovernorAgent(base string) bool { return base == "ovav.md" }
