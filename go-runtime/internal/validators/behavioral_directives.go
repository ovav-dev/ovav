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

// BehavioralDirectives validates behavioral directive integrity across agent
// files and the BEHAVIORAL_DIRECTIVES.yaml configuration.
// Replaces: tools/validators/check_behavioral_directives.py
type BehavioralDirectives struct{}

func NewBehavioralDirectives() *BehavioralDirectives { return &BehavioralDirectives{} }

func (v *BehavioralDirectives) ID() string   { return "behavioral_directives" }
func (v *BehavioralDirectives) Name() string { return "Behavioral Directives" }
func (v *BehavioralDirectives) Description() string {
	return "Validates behavioral directive integrity across agent files and BEHAVIORAL_DIRECTIVES.yaml"
}
func (v *BehavioralDirectives) Weight() int { return 7 }

// Required scopes that must be present in behavioral directives.
var requiredScopes = map[string]bool{
	"personality":    true,
	"delivery":       true,
	"work_execution": true,
	"safety":         true,
}

// Required behavioral governance markers that must appear in agent files.
var requiredMarkers = []string{
	"OVAV_INTEGRITY_SEAL",
	"soberano",
	"Redirigir a",
}

func (v *BehavioralDirectives) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// ── 1. Validate BEHAVIORAL_DIRECTIVES.yaml ───────────────────────────────
	directivesPath := filepath.Join(root, ".ovav", "context", "BEHAVIORAL_DIRECTIVES.yaml")
	data, err := os.ReadFile(directivesPath)
	if err != nil {
		issues = append(issues, "MISSING: BEHAVIORAL_DIRECTIVES.yaml not found — no behavioral context for sessions")
	} else {
		var directives struct {
			Directives []struct {
				Rule       string      `yaml:"rule"`
				Confidence interface{} `yaml:"confidence"`
				Scope      string      `yaml:"scope"`
			} `yaml:"directives"`
		}
		if err := yaml.Unmarshal(data, &directives); err != nil {
			issues = append(issues, fmt.Sprintf("SYNTAX_ERROR: BEHAVIORAL_DIRECTIVES.yaml invalid YAML: %v", err))
		} else if len(directives.Directives) == 0 {
			issues = append(issues, "EMPTY: BEHAVIORAL_DIRECTIVES.yaml has no directives defined")
		} else {
			// Count active directives (confidence >= 0.5)
			active := 0
			scopesPresent := make(map[string]bool)
			for i, d := range directives.Directives {
				if d.Rule == "" {
					issues = append(issues, fmt.Sprintf("INCOMPLETE: Directive #%d missing 'rule' field", i+1))
				}
				conf := toFloat(d.Confidence)
				if conf < 0 || conf > 1 {
					issues = append(issues, fmt.Sprintf("INVALID: Directive #%d confidence must be 0.0-1.0, got %v", i+1, d.Confidence))
				}
				if conf >= 0.5 {
					active++
					if d.Scope != "" {
						scopesPresent[d.Scope] = true
					}
				}
			}
			issues = append(issues, fmt.Sprintf("INFO: %d/%d behavioral directives active", active, len(directives.Directives)))

			// Check scope coverage
			var missingScopes []string
			for scope := range requiredScopes {
				if !scopesPresent[scope] {
					missingScopes = append(missingScopes, scope)
				}
			}
			if len(missingScopes) > 0 {
				issues = append(issues, fmt.Sprintf("WARN: Missing behavioral directives for scopes: %s", strings.Join(missingScopes, ", ")))
			}
		}
	}

	// ── 2. Validate agent files contain governance markers ───────────────────
	agentsDir := filepath.Join(root, "clients", "opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("ERROR: Cannot read agents directory: %v", err))
	} else {
		var agentsWithoutMarkers []string
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			agentPath := filepath.Join(agentsDir, e.Name())
			content, err := os.ReadFile(agentPath)
			if err != nil {
				continue
			}
			text := string(content)
			missing := []string{}
			for _, marker := range requiredMarkers {
				if !strings.Contains(text, marker) {
					// Boundary law can also be expressed as "HARD STOP" or "Limitaciones"
					if marker == "soberano" && (strings.Contains(text, "HARD STOP") || strings.Contains(text, "Limitaciones Explícitas")) {
						continue
					}
					if marker == "Redirigir a" && strings.Contains(text, "HARD STOP") {
						continue
					}
					missing = append(missing, marker)
				}
			}
			if len(missing) > 0 {
				agentsWithoutMarkers = append(agentsWithoutMarkers, fmt.Sprintf("%s (missing: %s)", e.Name(), strings.Join(missing, ", ")))
			}
		}
		if len(agentsWithoutMarkers) > 0 {
			for _, a := range agentsWithoutMarkers {
				issues = append(issues, fmt.Sprintf("BEHAVIORAL: agent %s", a))
			}
		}
	}

	// ── 3. Determine result ──────────────────────────────────────────────────
	hasError := false
	for _, issue := range issues {
		if strings.HasPrefix(issue, "MISSING:") || strings.HasPrefix(issue, "SYNTAX_ERROR:") ||
			strings.HasPrefix(issue, "EMPTY:") || strings.HasPrefix(issue, "INVALID:") ||
			strings.HasPrefix(issue, "INCOMPLETE:") || strings.HasPrefix(issue, "ERROR:") {
			hasError = true
			break
		}
	}
	if hasError {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:  fmt.Sprintf("FAIL behavioral directives — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:  "PASS behavioral directives — YAML valid, agent governance markers present",
		Duration: time.Since(start),
	}
}

// toFloat converts an interface{} to float64, handling int and float64.
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

var _ Validator = (*BehavioralDirectives)(nil)
