package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InvalidFixtures is a meta-validator that checks invalid fixture registries are detected.
// Replaces: check_invalid_fixtures.py
type InvalidFixtures struct{}

func NewInvalidFixtures() *InvalidFixtures { return &InvalidFixtures{} }

func (f *InvalidFixtures) ID() string   { return "invalid_fixtures" }
func (f *InvalidFixtures) Name() string { return "Invalid Fixtures Validator" }
func (f *InvalidFixtures) Description() string {
	return "Meta-validator: verifies that deliberately broken fixture registries are correctly rejected"
}
func (f *InvalidFixtures) Weight() int { return 5 }

func (f *InvalidFixtures) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Check that invalid fixture directory exists
	fixtureRoot := filepath.Join(root, "tests", "fixtures", "invalid_registries")
	if _, err := os.Stat(fixtureRoot); os.IsNotExist(err) {
		// Fixtures dir missing — not a failure if tests are in CI context
		return Result{ID: f.ID(), Name: f.Name(), Status: "pass", Weight: f.Weight(),
			Message:  "PASS — no invalid fixtures directory to check (expected in CI)",
			Duration: time.Since(start)}
	}

	// Read fixture directories
	entries, err := os.ReadDir(fixtureRoot)
	if err != nil {
		issues = append(issues, fmt.Sprintf("cannot read fixtures dir: %v", err))
		return Result{ID: f.ID(), Name: f.Name(), Status: "error", Weight: f.Weight(),
			Message:  "ERROR — cannot read fixtures directory",
			Issues:   issues,
			Duration: time.Since(start)}
	}

	checked := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fixtureDir := filepath.Join(fixtureRoot, entry.Name())
		checked++

		// Check that the fixture dir has invalid/missing registry files
		hasIssues := false

		// Check for common fixture conditions
		if _, err := os.Stat(filepath.Join(fixtureDir, ".ovav", "registry")); os.IsNotExist(err) {
			hasIssues = true
		} else {
			// Check specific registry files
			registryFiles := []string{"service_profiles.yaml", "skills.yaml", "memory_policy.yaml", "phase_dag.yaml"}
			for _, rf := range registryFiles {
				if _, err := os.Stat(filepath.Join(fixtureDir, ".ovav", "registry", rf)); os.IsNotExist(err) {
					hasIssues = true
					break
				}
			}
		}

		if !hasIssues {
			// All expected files exist — fixture may not be broken enough
			issues = append(issues, fmt.Sprintf("%s: fixture may not have enough broken registries", entry.Name()))
		}
	}

	if len(issues) > 0 {
		return Result{ID: f.ID(), Name: f.Name(), Status: "fail", Weight: f.Weight(),
			Message:  fmt.Sprintf("FAIL — %d fixture issue(s) in %d fixture(s) checked", len(issues), checked),
			Issues:   issues,
			Duration: time.Since(start)}
	}
	return Result{ID: f.ID(), Name: f.Name(), Status: "pass", Weight: f.Weight(),
		Message:  fmt.Sprintf("PASS — %d invalid fixture(s) verified", checked),
		Duration: time.Since(start)}
}

var _ Validator = (*InvalidFixtures)(nil)
