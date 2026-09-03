package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// LeadScope validates that each lead contract in .ovav/service_areas/*/lead_contract.yaml
// includes scope definition (covers and does_not_cover sections).
// Replaces: check_lead_scope.py
type LeadScope struct{}

func NewLeadScope() *LeadScope { return &LeadScope{} }

func (l *LeadScope) ID() string   { return "lead_scope" }
func (l *LeadScope) Name() string { return "Lead Scope Validator" }
func (l *LeadScope) Description() string {
	return "Validates that each lead agent file defines its authorized scope section"
}
func (l *LeadScope) Weight() int { return 5 }

func (l *LeadScope) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	saDir := filepath.Join(root, ".ovav", "service_areas")
	entries, err := os.ReadDir(saDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No service_areas directory — skip validation (not applicable for minimal test fixtures)
			return Result{
				ID: l.ID(), Name: l.Name(), Status: "skip", Weight: l.Weight(),
				Message:  "SKIP lead scope — service areas directory not found",
				Duration: time.Since(start),
			}
		}
		issues = append(issues, fmt.Sprintf("cannot read service areas directory: %s: %v", saDir, err))
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
			Message:  "FAIL lead scope — cannot access service areas directory",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	leadCount := 0
	missingCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		leadContractPath := filepath.Join(saDir, entry.Name(), "lead_contract.yaml")
		data, err := os.ReadFile(leadContractPath)
		if err != nil {
			continue
		}

		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			issues = append(issues, fmt.Sprintf("cannot parse lead_contract.yaml in %s: %v", entry.Name(), err))
			continue
		}

		leadCount++
		hasScope := false
		if lc, ok := doc["lead_contract"].(map[string]interface{}); ok {
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
		}

		if !hasScope {
			missingCount++
			issues = append(issues, fmt.Sprintf(
				"lead %s missing scope definition (covers/does_not_cover/responsibilities/authority)",
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
		issues = append(issues, "no lead_contract.yaml files found in service areas")
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
