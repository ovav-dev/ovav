// Package runtime — NEW IDEA gate component.
// Detects new project/feature/scope deviation patterns in input text.
// This is an AUTOMATIC runtime component that fires on EVERY input
// before any other processing (PL-0 [1] NEW IDEA DETECTOR).
package runtime

import (
	"regexp"
	"strings"
)

// NewIdeaResult is the output of the NEW IDEA detector
type NewIdeaResult struct {
	IsNewIdea       bool     `json:"is_new_idea"`
	MatchedPatterns []string `json:"matched_patterns"`
	Category        string   `json:"category"`       // "new_project" | "feature_request" | "scope_deviation" | "external_system" | "goal_redefinition"
	Severity        string   `json:"severity"`       // "hard" | "soft"
	Recommendation  string   `json:"recommendation"` // "BRAINSTORM" | "EXECUTE" | "ESCALATE"
	Task            string   `json:"task"`
}

// newIdeaPatterns defines all NEW IDEA detection patterns
type newIdeaPatterns struct {
	// New project patterns
	newProject []*regexp.Regexp
	// Feature request patterns
	featureRequest []*regexp.Regexp
	// Scope deviation patterns
	scopeDeviation []*regexp.Regexp
	// External system patterns
	externalSystem []*regexp.Regexp
	// Goal redefinition patterns
	goalRedefinition []*regexp.Regexp
}

func newPattern(patterns []string) []*regexp.Regexp {
	var result []*regexp.Regexp
	for _, p := range patterns {
		result = append(result, regexp.MustCompile(`(?i)`+p))
	}
	return result
}

var patterns = newIdeaPatterns{
	newProject: newPattern([]string{
		`quiero hacer`,
		`nueva app`,
		`nuevo sistema`,
		`construir`,
		`crear un`,
		`crear una`,
		`empezar un proyecto`,
		`iniciar un proyecto`,
		`new project`,
		`new app`,
		`build a`,
		`creating`,
		`start a project`,
		`lets build`,
		`vamos a construir`,
		`nuevo producto`,
		`nueva funcionalidad`,
		`quiero un`,
		`quiero una`,
		`sistema completo`,
		`todo el sistema`,
		`todo un`,
		`toda la app`,
		`completo de`,
		`full system`,
		`complete system`,
		`entire system`,
	}),

	featureRequest: newPattern([]string{
		`agregar`,
		`implementar`,
		`crear`,
		`añadir`,
		`add`,
		`implement`,
		`create new`,
		`nueva feature`,
		`nuevo comando`,
		`nueva funcionalidad`,
		`agregar soporte`,
		`add support for`,
		`vamos a agregar`,
		`quiero agregar`,
		`me gustaria agregar`,
		`would like to add`,
		`feature:`,
		`feat:`,
	}),

	scopeDeviation: newPattern([]string{
		`en vez de`,
		`cambiar a`,
		`migrar a`,
		`instead of`,
		`change to`,
		`migrate to`,
		`switch to`,
		`en lugar de`,
		`pasar de`,
		`dejar de usar`,
		`abandonar`,
		`reemplazar`,
		`replace`,
		`desviacion`,
		`scope change`,
		`scope drift`,
	}),

	externalSystem: newPattern([]string{
		`conectar a`,
		`integrar con`,
		`connect to`,
		`integrate with`,
		`conectar con`,
		`usar api de`,
		`usar el api`,
		`consumir api`,
		`webhook`,
		`external service`,
		`servicio externo`,
		`conectar un`,
		`integrar un`,
		`llamar a`,
		`autenticar con`,
	}),

	goalRedefinition: newPattern([]string{
		`el objetivo es`,
		`lo que necesito es`,
		`el proposito es`,
		`la meta es`,
		`the goal is`,
		`what i need is`,
		`the purpose is`,
		`basically`,
		`lo que quiero es`,
		`necesito que`,
		`quiero que`,
		`el objetivo principal`,
		`objetivo principal`,
		`reiniciar proyecto`,
		`empezar desde cero`,
	}),
}

// EvaluateNewIdea implements PL-0 [1] NEW IDEA DETECTOR
// Returns whether the input matches any NEW IDEA patterns
func EvaluateNewIdea(task string) NewIdeaResult {
	taskLower := strings.ToLower(task)
	var matchedPatterns []string
	var categories []string

	// Check new project patterns
	for _, p := range patterns.newProject {
		if p.MatchString(taskLower) {
			matchedPatterns = append(matchedPatterns, "new_project:"+p.String())
			categories = append(categories, "new_project")
		}
	}

	// Check feature request patterns
	for _, p := range patterns.featureRequest {
		if p.MatchString(taskLower) {
			matchedPatterns = append(matchedPatterns, "feature_request:"+p.String())
			categories = append(categories, "feature_request")
		}
	}

	// Check scope deviation patterns
	for _, p := range patterns.scopeDeviation {
		if p.MatchString(taskLower) {
			matchedPatterns = append(matchedPatterns, "scope_deviation:"+p.String())
			categories = append(categories, "scope_deviation")
		}
	}

	// Check external system patterns
	for _, p := range patterns.externalSystem {
		if p.MatchString(taskLower) {
			matchedPatterns = append(matchedPatterns, "external_system:"+p.String())
			categories = append(categories, "external_system")
		}
	}

	// Check goal redefinition patterns
	for _, p := range patterns.goalRedefinition {
		if p.MatchString(taskLower) {
			matchedPatterns = append(matchedPatterns, "goal_redefinition:"+p.String())
			categories = append(categories, "goal_redefinition")
		}
	}

	// Determine severity
	severity := "soft"
	for _, cat := range categories {
		if cat == "new_project" || cat == "scope_deviation" {
			severity = "hard"
			break
		}
	}

	// Determine recommendation
	rec := "EXECUTE"
	if len(categories) > 0 {
		rec = "BRAINSTORM"
	}

	// Category is the most severe matched
	category := ""
	if len(categories) > 0 {
		// Priority order
		priority := map[string]int{
			"new_project":       5,
			"scope_deviation":   4,
			"external_system":   3,
			"goal_redefinition": 2,
			"feature_request":   1,
		}
		best := ""
		bestP := 0
		for _, c := range categories {
			if p := priority[c]; p > bestP {
				bestP = p
				best = c
			}
		}
		category = best
	}

	return NewIdeaResult{
		IsNewIdea:       len(categories) > 0,
		MatchedPatterns: matchedPatterns,
		Category:        category,
		Severity:        severity,
		Recommendation:  rec,
		Task:            task,
	}
}

// PlanExistsResult checks if a plan exists for the given project name
type PlanExistsResult struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path,omitempty"`
}

// CheckPlanExists verifies if a plan exists for the given project identifier
func CheckPlanExists(projectName string) PlanExistsResult {
	// This requires filesystem access — use only in CLI context
	return PlanExistsResult{Exists: false}
}

// MultiAreaResult detects if input spans multiple service areas
type MultiAreaResult struct {
	IsMultiArea bool     `json:"is_multi_area"`
	AreasFound  []string `json:"areas_found"`
	PrimaryArea string   `json:"primary_area"`
	LeadsFound  []string `json:"leads_found"`
}

// DetectMultiArea implements PL-0 [3] AREA SCOPE DETECTION
func DetectMultiArea(task string) MultiAreaResult {
	serviceArea := DetectServiceArea(task)
	lead := LeadForArea(serviceArea)

	// Check for keywords from multiple areas
	taskLower := strings.ToLower(task)
	multiKeywords := map[string][]string{
		"platform_engineering":  {"backend", "api", "go", "python", "database", "sql"},
		"ux_design":             {"frontend", "ui", "ux", "css", "react", "design"},
		"research_intelligence": {"research", "analisis", "benchmark"},
		"commercial_growth":     {"money", "revenue", "pricing", "customer"},
		"devops_infrastructure": {"devops", "kubernetes", "docker", "cloud"},
		"digital_product":       {"fullstack", "app", "producto", "web"},
	}

	areasFound := []string{}
	for area := range multiKeywords {
		for _, kw := range multiKeywords[area] {
			if strings.Contains(taskLower, kw) {
				areasFound = append(areasFound, area)
				break
			}
		}
	}

	// Deduplicate
	seen := map[string]bool{}
	unique := []string{}
	for _, a := range areasFound {
		if !seen[a] {
			seen[a] = true
			unique = append(unique, a)
		}
	}

	return MultiAreaResult{
		IsMultiArea: len(unique) > 1,
		AreasFound:  unique,
		PrimaryArea: serviceArea,
		LeadsFound:  []string{lead},
	}
}

// EffortCalibration evaluates task complexity
type EffortCalibration struct {
	Effort  string   `json:"effort"` // "simple" | "moderate" | "complex" | "epic"
	Signals []string `json:"signals"`
}

// CalibrateEffort implements PL-0 [4] EFFORT CALIBRATION
func CalibrateEffort(task string) EffortCalibration {
	taskLower := strings.ToLower(task)
	var signals []string

	// Epic signals
	epicSignals := []string{"completo", "full system", "todo el", "entire", "whole system", "todo el sistema"}
	for _, s := range epicSignals {
		if strings.Contains(taskLower, s) {
			signals = append(signals, "epic:"+s)
		}
	}

	// Complex signals
	complexSignals := []string{"arquitectura", "architecture", "nuevo sistema", "new system", "redisenio", "redesign"}
	for _, s := range complexSignals {
		if strings.Contains(taskLower, s) {
			signals = append(signals, "complex:"+s)
		}
	}

	// Moderate signals
	moderateSignals := []string{"feature", "funcionalidad", "agregar", "implementar", "build", "implement"}
	for _, s := range moderateSignals {
		if strings.Contains(taskLower, s) {
			signals = append(signals, "moderate:"+s)
		}
	}

	// Simple signals
	simpleSignals := []string{"fix", "typo", "small", "pequeno", "quick", "un fix"}
	for _, s := range simpleSignals {
		if strings.Contains(taskLower, s) {
			signals = append(signals, "simple:"+s)
		}
	}

	// Determine effort
	effort := "moderate"
	if len(signals) == 0 {
		effort = "moderate"
	}
	for _, sig := range signals {
		if strings.HasPrefix(sig, "epic:") {
			effort = "epic"
			break
		}
		if strings.HasPrefix(sig, "complex:") {
			effort = "complex"
		}
	}
	for _, sig := range signals {
		if strings.HasPrefix(sig, "simple:") && effort != "epic" && effort != "complex" {
			effort = "simple"
		}
	}

	return EffortCalibration{
		Effort:  effort,
		Signals: signals,
	}
}
