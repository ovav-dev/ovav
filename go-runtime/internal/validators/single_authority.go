package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SingleAuthority validates that exactly ONE canonical authority source exists
// (.ovav/plan/caps.yaml) and no other file claims authority over system state.
// Replaces: check_single_authority_source.py
type SingleAuthority struct{}

func NewSingleAuthority() *SingleAuthority { return &SingleAuthority{} }

func (s *SingleAuthority) ID() string   { return "single_authority" }
func (s *SingleAuthority) Name() string { return "Single Authority Source" }
func (s *SingleAuthority) Description() string {
	return "Validates single canonical authority: caps.yaml + git HEAD"
}
func (s *SingleAuthority) Weight() int { return 18 }

// Authority keywords that must ONLY appear in caps.yaml.
var authorityKeywords = []string{
	"fuente canónica", "source of truth", "autoridad activa",
	"canonical source", "decide estado", "active authority",
	"current_authority_contract",
}

// Derived files that must NOT contain authority keywords.
var derivedFiles = []string{
	".ovav/context/CURRENT_HANDOFF.md",
	".ovav/context/BUILD1_FINAL_HANDOFF.md",
	".ovav/context/BUILD2_FINAL_HANDOFF.md",
	".ovav/context/BUILD3_CURRENT_HANDOFF.md",
}

// The single canonical plan file.
var planPath = ".ovav/plan/caps.yaml"

// Stale authority contract that must NOT exist.
var staleContractPath = ".ovav/service_areas/shared/current_authority_contract.yaml"

func (s *SingleAuthority) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	var failures []string
	checksPassed := 0
	checksTotal := 4

	// Check 1: caps.yaml exists as the single authority
	planFullPath := filepath.Join(root, planPath)
	if _, err := os.Stat(planFullPath); os.IsNotExist(err) {
		issue := fmt.Sprintf("CRITICAL: %s not found — no canonical plan source", planPath)
		issues = append(issues, issue)
		failures = append(failures, issue)
	} else {
		checksPassed++
		// Verify caps.yaml declares itself as canonical
		data, _ := os.ReadFile(planFullPath)
		content := string(data)
		if !strings.Contains(content, "fuente de datos") && !strings.Contains(content, "canonical") {
			issues = append(issues, "WARNING: caps.yaml does not declare itself as canonical source")
		}
	}

	// Check 2: Stale authority contract is NOT present
	staleFullPath := filepath.Join(root, staleContractPath)
	if _, err := os.Stat(staleFullPath); err == nil {
		issue := fmt.Sprintf("CRITICAL: %s still exists — was replaced by caps.yaml + git HEAD. Must be deleted.", staleContractPath)
		issues = append(issues, issue)
		failures = append(failures, issue)
	} else {
		checksPassed++
	}

	// Check 3: CURRENT_HANDOFF.md is marked as generated
	handoffPath := filepath.Join(root, ".ovav", "context", "CURRENT_HANDOFF.md")
	if data, err := os.ReadFile(handoffPath); err == nil {
		content := string(data)
		if !strings.Contains(content, "GENERADO DESDE") && !strings.Contains(content, "SIN AUTORIDAD") {
			issues = append(issues, "WARNING: CURRENT_HANDOFF.md is not marked as generated (missing GENERADO DESDE / SIN AUTORIDAD)")
		} else {
			checksPassed++
		}
	} else {
		// File doesn't exist — not critical, it'll be generated
		checksPassed++
	}

	// Check 4: No duplicate authority claims in derived files
	duplicateFound := false
	for _, df := range derivedFiles {
		dfPath := filepath.Join(root, df)
		data, err := os.ReadFile(dfPath)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		for _, kw := range authorityKeywords {
			if strings.Contains(content, strings.ToLower(kw)) {
				issues = append(issues, fmt.Sprintf(
					"WARNING: %s contains '%s' — only caps.yaml has authority", df, kw))
				duplicateFound = true
				break
			}
		}
	}
	if !duplicateFound {
		checksPassed++
	}

	if len(failures) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL single authority — %d/%d checks passed", checksPassed, checksTotal),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	if len(issues) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "warn", Weight: s.Weight(),
			Message: fmt.Sprintf("WARN single authority — %d generated-artifact warning(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message: fmt.Sprintf("PASS single authority — %d/%d checks, caps.yaml + git HEAD is canonical",
			checksPassed, checksTotal),
		Duration: time.Since(start),
	}
}

var _ Validator = (*SingleAuthority)(nil)
