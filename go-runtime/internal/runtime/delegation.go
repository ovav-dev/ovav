// Package runtime implements OVAV delegation runtime gates and routers.
// These components form the A2A mesh delegation layer.
//
// OVAV Signature: internal/runtime — stabilized 2026-08-02
// Security fix: path traversal sanitization on profileName.
package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RepoRoot returns the OVAV repo root by walking up for .git
func RepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// DelegationTriggerResult is the output of h_delegation_trigger
type DelegationTriggerResult struct {
	ShouldDelegate bool     `json:"should_delegate"`
	Score          int      `json:"score"`
	Threshold      int      `json:"threshold"`
	Reasons        []string `json:"reasons"`
	Task           string   `json:"task"`
	Blocked        bool     `json:"blocked"`
	BlockReason    string   `json:"block_reason,omitempty"`
}

// EvaluateTaskComplexity evaluates whether a task warrants delegation
func EvaluateTaskComplexity(task string, filesCount int, architectureChange, authSecurityChange, globalConfigChange bool, researchSources int) DelegationTriggerResult {
	score := 0
	var reasons []string

	// Positive triggers for delegation
	if filesCount >= 4 {
		score += 30
		reasons = append(reasons, fmt.Sprintf("files_count=%d >= threshold", filesCount))
	}
	if filesCount >= 2 {
		score += 20
		reasons = append(reasons, fmt.Sprintf("non_trivial_files=%d", filesCount))
	}
	if architectureChange {
		score += 50
		reasons = append(reasons, "architecture_change=true")
	}
	if authSecurityChange {
		score += 40
		reasons = append(reasons, "auth_security_change=true")
	}
	if globalConfigChange {
		score += 35
		reasons = append(reasons, "global_config_change=true")
	}
	if researchSources >= 4 {
		score += 25
		reasons = append(reasons, fmt.Sprintf("research_sources=%d", researchSources))
	}

	threshold := 40
	should := score >= threshold

	return DelegationTriggerResult{
		ShouldDelegate: should,
		Score:          score,
		Threshold:      threshold,
		Reasons:        reasons,
		Task:           task,
		Blocked:        !should,
	}
}

// DoNotDelegateResult is the output of h_do_not_delegate_guard
type DoNotDelegateResult struct {
	ShouldDelegate bool   `json:"should_delegate"`
	Blocked        bool   `json:"blocked"`
	Reason         string `json:"reason,omitempty"`
	Task           string `json:"task"`
	FilesCount     int    `json:"files_count"`
}

// EvaluateDoNotDelegate checks if task is trivially delegable
func EvaluateDoNotDelegate(task string, filesCount int) DoNotDelegateResult {
	taskLower := strings.ToLower(task)
	words := strings.Fields(taskLower)

	// Pure explanation check (English)
	explanationPrefixes := []string{"explain", "what is", "how does", "can you explain", "what does", "why does", "when is", "where is"}
	for _, prefix := range explanationPrefixes {
		if strings.HasPrefix(taskLower, prefix) {
			return DoNotDelegateResult{
				ShouldDelegate: false,
				Blocked:        true,
				Reason:         "pure_explanation",
				Task:           task,
				FilesCount:     filesCount,
			}
		}
	}

	// Pure explanation check (Spanish)
	spanishExplanations := []string{"explica", "explica que", "que es", "que es un", "que es una", "como funciona", "como se hace", "para que sirve", "defiende", "describeme", "dime que es", "hazme un resumen"}
	for _, phrase := range spanishExplanations {
		if strings.HasPrefix(taskLower, phrase) || strings.Contains(taskLower, phrase) {
			// Only block if very short (under 6 words) — real tasks can have these words too
			if len(words) < 6 {
				return DoNotDelegateResult{
					ShouldDelegate: false,
					Blocked:        true,
					Reason:         "pure_explanation_spanish",
					Task:           task,
					FilesCount:     filesCount,
				}
			}
		}
	}

	// Very short questions
	if len(words) <= 4 && (strings.HasPrefix(taskLower, "what") || strings.HasPrefix(taskLower, "how") ||
		strings.HasPrefix(taskLower, "why") || strings.HasPrefix(taskLower, "when") ||
		strings.HasPrefix(taskLower, "where") || strings.HasPrefix(taskLower, "who")) {
		return DoNotDelegateResult{
			ShouldDelegate: false,
			Blocked:        true,
			Reason:         "direct_user_question_with_known_answer",
			Task:           task,
			FilesCount:     filesCount,
		}
	}

	// Single-file trivial fix — any fix/correct + typo/misspell pattern
	// Infer single-file from task mentioning a specific file
	singleFilePatterns := []string{".md", ".go", ".ts", ".tsx", ".js", ".json", ".yaml", ".yml", ".txt", ".py", "readme", "license", "changelog", "gitignore", "dockerfile", "makefile"}
	inferFiles := 0
	if filesCount == 0 {
		for _, pattern := range singleFilePatterns {
			if strings.Contains(taskLower, pattern) {
				inferFiles++
			}
		}
	}
	effectiveFiles := filesCount
	if effectiveFiles == 0 && inferFiles > 0 {
		effectiveFiles = inferFiles
	}

	if effectiveFiles == 1 {
		hasFix := strings.Contains(taskLower, "fix ") || strings.Contains(taskLower, "corregir ") ||
			strings.Contains(taskLower, "arreglar ") || strings.Contains(taskLower, "typo")
		hasSubject := strings.Contains(taskLower, "typo") || strings.Contains(taskLower, "misspell") ||
			strings.Contains(taskLower, "whitespace") || strings.Contains(taskLower, "comment")
		if hasFix && hasSubject {
			return DoNotDelegateResult{
				ShouldDelegate: false,
				Blocked:        true,
				Reason:         "typo_or_copy_fix",
				Task:           task,
				FilesCount:     effectiveFiles,
			}
		}
	}

	return DoNotDelegateResult{
		ShouldDelegate: true,
		Blocked:        false,
		Task:           task,
		FilesCount:     filesCount,
	}
}

// DelegationRouterResult is the output of the delegation router
type DelegationRouterResult struct {
	ServiceArea     string   `json:"service_area_detected"`
	LeadResolved    string   `json:"lead_resolved"`
	AgentID         string   `json:"agent_id"`
	AgentName       string   `json:"agent_name"`
	AgentArea       string   `json:"agent_area"`
	AgentKind       string   `json:"agent_kind"`
	RoutingSequence []string `json:"routing_sequence"`
	Task            string   `json:"task"`
	RoutedAt        string   `json:"routed_at"`
}

// DetectServiceArea detects service area from task keywords
func DetectServiceArea(task string) string {
	taskLower := strings.ToLower(task)

	areaKeywords := map[string][]string{
		"platform_engineering":     {"backend", "api", "go", "python", "database", "sql", "runtime", "cli", "governor", "command"},
		"ux_design":                {"frontend", "ui", "ux", "diseno", "css", "react", "vista", "design", "interfaz"},
		"research_intelligence":    {"research", "analisis", "benchmark", "estudio", "fuentes", "investigacion", "analizar"},
		"commercial_growth":        {"money", "revenue", "pricing", "customer", "growth", "venta", "monetizacion", "business"},
		"devops_infrastructure":    {"devops", "kubernetes", "docker", "cloud", "aws", "deploy", "infra", "servidor"},
		"health_performance":       {"health", "metricas", "performance", "latencia", "salud", "medicion"},
		"legal_compliance":         {"legal", "contract", "compliance", "gdpr", "contrato", "legal"},
		"adversarial_intelligence": {"security", "hack", "penetration", "red team", "seguridad", "vulnerabilidad", "injection", "sql", "cmd", "command", "xss", "csrf", "authentication", "authorization", "path traversal", "xxe", "deserialization", "credential", "hardcoded", "weak random", "cryptographic"},
		"education_career":         {"education", "course", "tutoring", "curriculum", "educacion", "aprendizaje"},
		"digital_product":          {"fullstack", "app", "producto", "web", "mobile", "aplicacion"},
	}

	scores := make(map[string]int)
	for area, keywords := range areaKeywords {
		for _, kw := range keywords {
			if strings.Contains(taskLower, kw) {
				scores[area]++
			}
		}
	}

	if len(scores) == 0 {
		return "platform_engineering"
	}

	maxScore := 0
	bestArea := "platform_engineering"
	for area, score := range scores {
		if score > maxScore {
			maxScore = score
			bestArea = area
		}
	}
	return bestArea
}

// LeadForArea maps service area to lead ID
func LeadForArea(area string) string {
	areaLeadMap := map[string]string{
		"platform_engineering":     "lead-thavren",
		"ux_design":                "lead-elena",
		"research_intelligence":    "lead-eidren",
		"commercial_growth":        "lead-sofia",
		"devops_infrastructure":    "lead-uriel",
		"health_performance":       "lead-renata",
		"legal_compliance":         "lead-camila",
		"adversarial_intelligence": "lead-kenji",
		"education_career":         "lead-valeria",
		"digital_product":          "lead-dante",
	}
	if lead, ok := areaLeadMap[area]; ok {
		return lead
	}
	return "lead-thavren"
}

// RouteDelegation implements the full delegation routing sequence
func RouteDelegation(task string, agentID string) DelegationRouterResult {
	serviceArea := DetectServiceArea(task)
	lead := agentID
	if lead == "" {
		lead = LeadForArea(serviceArea)
	}

	// Determine kind
	kind := "lead"
	if strings.HasPrefix(lead, "team-") {
		kind = "team"
	} else if strings.HasPrefix(lead, "lead-") {
		kind = "lead"
	}

	// Extract name from lead ID
	name := strings.TrimPrefix(lead, "lead-")
	name = strings.TrimPrefix(name, "team-")
	name = strings.ReplaceAll(name, "-", " ")

	return DelegationRouterResult{
		ServiceArea:  serviceArea,
		LeadResolved: lead,
		AgentID:      lead,
		AgentName:    name,
		AgentArea:    serviceArea,
		AgentKind:    kind,
		RoutingSequence: []string{
			"Service Area Router",
			"Lead resolves first",
			"Delegation Router",
			"Context Gateway",
			"Tool Gateway",
			"Delivery Contract",
			"Observability Trace",
		},
		Task:     task,
		RoutedAt: time.Now().UTC().Format(time.RFC3339) + "Z",
	}
}

// GitContext holds git workspace state
type GitContext struct {
	Branch        string   `json:"git_branch"`
	Head          string   `json:"git_head"`
	Status        string   `json:"git_status"`
	ModifiedFiles []string `json:"modified_files"`
	StagedFiles   []string `json:"staged_files"`
	WorktreeRoot  string   `json:"worktree_root"`
}

// GetGitContext returns current git workspace state
func GetGitContext() GitContext {
	ctx := GitContext{
		WorktreeRoot: RepoRoot(),
	}

	run := func(name string, args ...string) string {
		cmd := exec.Command(name, args...)
		cmd.Dir = ctx.WorktreeRoot
		out, err := cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}

	if head := run("git", "rev-parse", "--abbrev-ref", "HEAD"); head != "" {
		ctx.Branch = head
	}
	if hash := run("git", "rev-parse", "HEAD"); hash != "" {
		ctx.Head = hash[:8]
	}
	if status := run("git", "status", "--porcelain"); status != "" {
		lines := strings.Split(status, "\n")
		for _, line := range lines {
			if len(line) < 3 {
				continue
			}
			statusFlag := line[:2]
			file := line[3:]
			if statusFlag[0] != '?' {
				ctx.StagedFiles = append(ctx.StagedFiles, file)
			}
			if statusFlag[1] == 'M' || statusFlag[0] == 'M' {
				ctx.ModifiedFiles = append(ctx.ModifiedFiles, file)
			}
		}
		ctx.Status = "dirty"
	} else {
		ctx.Status = "clean"
	}

	return ctx
}

// DelegationPayload is the full payload for subagent delegation
type DelegationPayload struct {
	AgentID   string       `json:"agent_id"`
	AgentName string       `json:"agent_name"`
	AgentArea string       `json:"agent_area"`
	AgentKind string       `json:"agent_kind"`
	Task      string       `json:"task"`
	Workspace GitContext   `json:"workspace_context"`
	Profile   AgentProfile `json:"profile"`
	Generated string       `json:"generated_at"`
	Generator string       `json:"generator"`
}

// AgentProfile holds the loaded agent profile
type AgentProfile struct {
	SystemPrompt string `json:"system_prompt"`
	FilePath     string `json:"file_path,omitempty"`
}

// BuildDelegationPayload builds complete delegation payload for an agent
func BuildDelegationPayload(agentID string, task string) (*DelegationPayload, error) {
	route := RouteDelegation(task, agentID)

	// Load agent profile — check leads first, then teams
	var profile AgentProfile
	repoRoot := RepoRoot()
	// Normalize agentID to filename: eidren → lead-eidren, team-clara → team-clara
	profileName := agentID
	if !strings.HasPrefix(profileName, "lead-") && !strings.HasPrefix(profileName, "team-") {
		profileName = "lead-" + profileName
	}

	// SECURITY: Sanitize profileName to prevent path traversal.
	// An agentID like "lead-../../etc" could escape the intended directory
	// if ".." is not removed before filepath.Join.
	profileName = strings.ReplaceAll(profileName, "..", "")

	// Try leads directory first
	profilePath := filepath.Join(repoRoot, "ovav", "agents", "leads", profileName+".yaml")
	if _, err := os.Stat(profilePath); err == nil {
		if data, err := os.ReadFile(profilePath); err == nil {
			profile.SystemPrompt = string(data)
			profile.FilePath = profilePath
		}
	}

	// If not found and it's a team, try teams directory
	if profile.SystemPrompt == "" && strings.HasPrefix(profileName, "team-") {
		teamsPath := filepath.Join(repoRoot, "ovav", "agents", "teams", profileName+".yaml")
		if _, err := os.Stat(teamsPath); err == nil {
			if data, err := os.ReadFile(teamsPath); err == nil {
				profile.SystemPrompt = string(data)
				profile.FilePath = teamsPath
			}
		}
	}

	return &DelegationPayload{
		AgentID:   route.AgentID,
		AgentName: route.AgentName,
		AgentArea: route.AgentArea,
		AgentKind: route.AgentKind,
		Task:      task,
		Workspace: GetGitContext(),
		Profile:   profile,
		Generated: time.Now().UTC().Format(time.RFC3339) + "Z",
		Generator: "go-runtime/internal/runtime",
	}, nil
}

// LoadDelegationRules loads delegation_rules.yaml
func LoadDelegationRules() map[string]interface{} {
	repoRoot := RepoRoot()
	rulesPath := filepath.Join(repoRoot, ".ovav", "registry", "delegation_rules.yaml")
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return map[string]interface{}{}
	}
	var rules map[string]interface{}
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return map[string]interface{}{}
	}
	return rules
}
