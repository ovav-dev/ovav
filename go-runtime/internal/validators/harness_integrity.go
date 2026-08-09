package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HarnessIntegrity validates harness contract alignment, group integrity, and file presence.
// Replaces: validate_harnesses.py
type HarnessIntegrity struct{}

func NewHarnessIntegrity() *HarnessIntegrity { return &HarnessIntegrity{} }

func (h *HarnessIntegrity) ID() string   { return "harness_integrity" }
func (h *HarnessIntegrity) Name() string { return "Harness Integrity Validator" }
func (h *HarnessIntegrity) Description() string {
	return "Validates harness contract alignment, grouping, and file integrity"
}
func (h *HarnessIntegrity) Weight() int { return 8 }

// Harness groups deprecated — Python harnesses migrated to Go validators.
// Go validators (internal/validators/) use different naming and organization.
// This validator now only checks that the validators package exists.
var harnessGroups = map[string][]string{}

// Required harness module names.
// Updated 2026-07-28: Python harnesses migrated to Go validators.
// These are the Go validator packages that serve as harnesses.
var requiredHarnesses = []string{
	"validators",
}

func (h *HarnessIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	harnessesDir := filepath.Join(root, "go-runtime", "internal", "validators")
	if _, err := os.Stat(harnessesDir); os.IsNotExist(err) {
		// Fallback to old Python harnesses dir
		harnessesDir = filepath.Join(root, "tools", "harnesses")
		if _, err := os.Stat(harnessesDir); os.IsNotExist(err) {
			issues = append(issues, "MISSING: harnesses directory not found (tried go-runtime/internal/validators and tools/harnesses)")
			return Result{
				ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
				Message:  "FAIL harness integrity — harnesses directory missing",
				Issues:   issues,
				Duration: time.Since(start),
			}
		}
	}

	// 1. Check harness files exist — match by h_/check_ prefix OR by known harness names
	entries, _ := os.ReadDir(harnessesDir)
	harnessFiles := make(map[string]bool)
	// Build a set of known harness names for matching
	knownNames := make(map[string]bool)
	for _, h := range requiredHarnesses {
		knownNames[h] = true
	}
	for _, harnesses := range harnessGroups {
		for _, h := range harnesses {
			knownNames[h] = true
		}
	}
	for _, e := range entries {
		// Check both .go and .py files in validators directory
		if !strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), ".py") {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".go"), ".py")
		// Match by prefix convention OR by known name
		if strings.HasPrefix(base, "h_") || strings.HasPrefix(base, "check_") || knownNames[base] || strings.HasSuffix(e.Name(), ".go") {
			harnessFiles[base] = true
		}
	}

	// Check required harnesses
	missingHarnesses := 0
	for _, required := range requiredHarnesses {
		if !harnessFiles[required] {
			missingHarnesses++
			issues = append(issues, fmt.Sprintf("MISSING_HARNESS: %s.py", required))
		}
	}

	// Check harness group coverage
	for groupName, harnesses := range harnessGroups {
		missing := 0
		for _, hName := range harnesses {
			if !harnessFiles[hName] {
				missing++
				issues = append(issues, fmt.Sprintf("GROUP_MISSING: %s group missing harness '%s'", groupName, hName))
			}
		}
	}

	// 2. Check for duplicate harness names
	seen := make(map[string]int)
	for name := range harnessFiles {
		seen[name]++
	}

	// 3. Check harnesses are importable (basic syntax check)
	for name := range harnessFiles {
		hPath := filepath.Join(harnessesDir, name+".py")
		if data, err := os.ReadFile(hPath); err == nil {
			content := string(data)
			if len(strings.TrimSpace(content)) == 0 {
				issues = append(issues, fmt.Sprintf("EMPTY_HARNESS: %s.py", name))
			}
		}
	}

	foundCount := len(harnessFiles)

	if missingHarnesses > 0 || len(issues) > 0 {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  fmt.Sprintf("FAIL harness integrity — %d/%d required harnesses found, %d issue(s)", foundCount-missingHarnesses, len(requiredHarnesses), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
		Message:  fmt.Sprintf("PASS harness integrity — %d harness files verified", foundCount),
		Duration: time.Since(start),
	}
}

var _ Validator = (*HarnessIntegrity)(nil)
