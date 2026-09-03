package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BehavioralDirectives validates source-level identity and area boundary truth.
// Replaces: tools/validators/check_behavioral_directives.py
type BehavioralDirectives struct{}

func NewBehavioralDirectives() *BehavioralDirectives { return &BehavioralDirectives{} }

func (v *BehavioralDirectives) ID() string   { return "behavioral_directives" }
func (v *BehavioralDirectives) Name() string { return "Behavioral Directives" }
func (v *BehavioralDirectives) Description() string {
	return "Validates canonical identity guard generation and source-level area hard-stop contracts"
}
func (v *BehavioralDirectives) Weight() int { return 7 }

func (v *BehavioralDirectives) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. The converter is canonical for projection identity guards.
	if !fileContainsAll(root, "go-runtime/internal/convert/convert.go", "OVAV_IDENTITY_GUARD", "WriteIdentityGuard") {
		issues = append(issues, "ERROR: canonical converter missing source-level identity guard")
	}
	if !fileContainsAll(root, "go-runtime/internal/convert/opencode.go", "WriteIdentityGuard") {
		issues = append(issues, "ERROR: OpenCode converter does not inject the canonical identity guard")
	}

	// 2. Validate canonical area/lead source, never the generated projections.
	areaDir := filepath.Join(root, ".ovav", "source", "agents", "areas")
	areaEntries, err := os.ReadDir(areaDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("ERROR: cannot read canonical area agents: %v", err))
	} else {
		for _, entry := range areaEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(areaDir, entry.Name()))
			if readErr != nil || !containsAnyText(string(data), "No cubre", "Hard boundaries", "HARD STOP") {
				issues = append(issues, fmt.Sprintf("ERROR: canonical area %s missing scope hard-stop boundary", entry.Name()))
			}
		}
	}

	leadDir := filepath.Join(root, ".ovav", "source", "agents", "leads")
	leadEntries, err := os.ReadDir(leadDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("ERROR: cannot read canonical lead agents: %v", err))
	} else {
		for _, entry := range leadEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(leadDir, entry.Name()))
			if readErr != nil || !containsAnyText(strings.ToLower(string(data)), "hard stop", "handoff", "redirigir") {
				issues = append(issues, fmt.Sprintf("ERROR: canonical lead %s missing out-of-scope routing contract", entry.Name()))
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
		Message:  "PASS behavioral directives — canonical identity guard and area hard-stop sources verified",
		Duration: time.Since(start),
	}
}

func containsAnyText(content string, alternatives ...string) bool {
	for _, alternative := range alternatives {
		if strings.Contains(content, alternative) {
			return true
		}
	}
	return false
}

// toFloat remains for compatibility with historical validator fixtures.
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
