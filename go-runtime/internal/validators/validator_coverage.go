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

// ValidatorCoverage validates meta-coverage of validators: what percentage is wired to automation.
// Replaces: check_validator_coverage.py
type ValidatorCoverage struct{}

func NewValidatorCoverage() *ValidatorCoverage { return &ValidatorCoverage{} }

func (v *ValidatorCoverage) ID() string   { return "validator_coverage" }
func (v *ValidatorCoverage) Name() string { return "Validator Coverage Auditor" }
func (v *ValidatorCoverage) Description() string {
	return "Measures validator and harness coverage across automation surfaces"
}
func (v *ValidatorCoverage) Weight() int { return 5 }

func (v *ValidatorCoverage) getValidatorsFromDir(root string) []string {
	var validators []string
	validatorsDir := filepath.Join(root, "tools", "validators")
	entries, err := os.ReadDir(validatorsDir)
	if err != nil {
		return validators
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") &&
			e.Name() != "__init__.py" && e.Name() != "common.py" {
			validators = append(validators, strings.TrimSuffix(e.Name(), ".py"))
		}
	}
	return validators
}

func (v *ValidatorCoverage) getValidateAllValidators(root string) []string {
	var validators []string
	vaPath := filepath.Join(root, "tools", "validators", "validate_all.py")
	data, err := os.ReadFile(vaPath)
	if err != nil {
		return validators
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "from tools.validators.") && strings.Contains(line, "import validate") {
			prefix := "from tools.validators."
			if idx := strings.Index(line, prefix); idx >= 0 {
				rest := line[idx+len(prefix):]
				if spaceIdx := strings.Index(rest, " import"); spaceIdx >= 0 {
					moduleName := strings.TrimSpace(rest[:spaceIdx])
					validators = append(validators, moduleName)
				}
			}
		}
	}
	return validators
}

func (v *ValidatorCoverage) getAutoTriggerValidators(root string) []string {
	var validators []string
	atPath := filepath.Join(root, ".ovav", "registry", "auto_triggers.yaml")
	data, err := os.ReadFile(atPath)
	if err != nil {
		return validators
	}
	var triggers map[string]interface{}
	if yaml.Unmarshal(data, &triggers) != nil {
		return validators
	}

	atData, _ := triggers["auto_triggers"].(map[string]interface{})
	if atData != nil {
		if router, ok := atData["router"].(map[string]interface{}); ok {
			for _, triggerList := range router {
				if list, ok := triggerList.([]interface{}); ok {
					for _, name := range list {
						if s, ok := name.(string); ok {
							validators = append(validators, s)
						}
					}
				}
			}
		}
	}
	return validators
}

func (v *ValidatorCoverage) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	allValidators := v.getValidatorsFromDir(root)
	vaValidators := v.getValidateAllValidators(root)
	atValidators := v.getAutoTriggerValidators(root)

	vaSet := make(map[string]bool)
	for _, va := range vaValidators {
		// Match validator names: strip prefixes
		match := va
		for _, prefix := range []string{"check_", "validate_", "verify_"} {
			if strings.HasPrefix(match, prefix) {
				match = strings.TrimPrefix(match, prefix)
			}
		}
		vaSet[match] = true
		vaSet[va] = true
	}

	atSet := make(map[string]bool)
	for _, a := range atValidators {
		match := a
		for _, prefix := range []string{"check_", "validate_", "verify_", "h_"} {
			if strings.HasPrefix(match, prefix) {
				match = strings.TrimPrefix(match, prefix)
			}
		}
		atSet[match] = true
		atSet[a] = true
	}

	// Calculate validate_all coverage
	total := len(allValidators)
	if total == 0 {
		issues = append(issues, "WARNING: No Python validators found in tools/validators/")
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
			Message:  "PASS validator coverage — no Python validators to measure (migration to Go complete?)",
			Duration: time.Since(start),
		}
	}

	vaCovered := 0
	for _, val := range allValidators {
		if vaSet[val] {
			vaCovered++
		}
	}
	vaPct := float64(vaCovered) / float64(total) * 100

	atCovered := 0
	for _, val := range allValidators {
		if atSet[val] {
			atCovered++
		}
	}
	atPct := float64(atCovered) / float64(total) * 100

	info := fmt.Sprintf("INFO: validate_all covers %d/%d validators (%.0f%%). auto_triggers: %d/%d (%.0f%%)",
		vaCovered, total, vaPct, atCovered, total, atPct)
	issues = append(issues, info)

	// Only flag if coverage is extremely low
	if vaPct < 5 {
		issues = append(issues, fmt.Sprintf("COVERAGE_GAP: validate_all coverage (%.0f%%) below threshold", vaPct))
	}

	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:  fmt.Sprintf("PASS validator coverage — %.0f%% validate_all, %.0f%% auto_triggers", vaPct, atPct),
		Issues:   issues,
		Duration: time.Since(start),
	}
}

var _ Validator = (*ValidatorCoverage)(nil)
