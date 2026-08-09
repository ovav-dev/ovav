package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LeadScope validates that each lead agent file in go-runtime/internal/runtimes/opencode/agents/lead-*.md
// includes a scope definition section (## Funciones Autorizadas or equivalent).
// Replaces: check_lead_scope.py
type LeadScope struct{}

func NewLeadScope() *LeadScope { return &LeadScope{} }

func (l *LeadScope) ID() string   { return "lead_scope" }
func (l *LeadScope) Name() string { return "Lead Scope Validator" }
func (l *LeadScope) Description() string {
	return "Validates that each lead agent file defines its authorized scope section"
}
func (l *LeadScope) Weight() int { return 5 }

const leadAgentsDir = "go-runtime/internal/runtimes/claude-code/agents"

// Scope section headers that satisfy the requirement.
var scopeHeaders = []string{
	"## Funciones Autorizadas",
	"## Authorized Functions",
	"## Scope",
	"## Scope Definition",
	"## Authorized Scope",
	"## Funciones",
	"## Autorizadas",
	"## Ámbito Autorizado",
}

func (l *LeadScope) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	agentsDir := filepath.Join(root, leadAgentsDir)
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("cannot read lead agents directory: %s: %v", agentsDir, err))
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
			Message:  "FAIL lead scope — cannot access lead agents directory",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	leadCount := 0
	missingCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "lead-") || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		leadCount++
		fullPath := filepath.Join(agentsDir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("cannot read lead file: %s: %v", entry.Name(), err))
			continue
		}

		content := string(data)
		hasScope := false
		for _, header := range scopeHeaders {
			if strings.Contains(content, header) {
				hasScope = true
				break
			}
		}

		if !hasScope {
			missingCount++
			issues = append(issues, fmt.Sprintf(
				"lead file %s missing scope definition section (expected: '## Funciones Autorizadas' or equivalent)",
				entry.Name(),
			))
		}
	}

	if missingCount > 0 {
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
			Message: fmt.Sprintf(
				"FAIL lead scope — %d of %d lead files missing scope definition",
				missingCount, leadCount,
			),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	if leadCount == 0 {
		issues = append(issues, "no lead-*.md files found in agents directory")
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
			Message:  "FAIL lead scope — no lead files found",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID: l.ID(), Name: l.Name(), Status: "pass", Weight: l.Weight(),
		Message:  fmt.Sprintf("PASS lead scope — all %d lead files have scope definitions", leadCount),
		Duration: time.Since(start),
	}
}

var _ Validator = (*LeadScope)(nil)
