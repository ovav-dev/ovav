// Package cli — freshsmoke.go: fresh clone smoke test.
//
// Go migration of tools/cli/ovav_fresh_clone_smoke.py (106 LOC).
// Validates that a fresh clone works end-to-end.
package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FreshCloneResult holds the smoke test result.
type FreshCloneResult struct {
	SchemaVersion  string                   `json:"schema_version"`
	GeneratedAt    string                   `json:"generated_at"`
	ClonePath      string                   `json:"clone_path,omitempty"`
	ValidatePassed bool                     `json:"validate_passed"`
	SmokePassed    bool                     `json:"smoke_passed"`
	OverallOK      bool                     `json:"overall_ok"`
	Checks         []map[string]interface{} `json:"checks"`
}

// FreshCloneSmoke validates that a fresh clone works end-to-end.
// Replaces ovav_fresh_clone_smoke.py.
func FreshCloneSmoke(repoRoot string, keepClone bool) FreshCloneResult {
	result := FreshCloneResult{
		SchemaVersion: "ovav.fresh_clone_smoke.v1",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Checks:        []map[string]interface{}{},
	}

	// Create temp directory
	sandbox, err := os.MkdirTemp("", "ovav-fresh-clone-")
	if err != nil {
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "clone_successful", "passed": false, "detail": err.Error(),
		})
		return result
	}
	clonePath := filepath.Join(sandbox, "OVAV")

	// Clone
	cloneCmd := exec.Command("git", "clone", repoRoot, clonePath)
	cloneCmd.Stderr = nil
	if err := cloneCmd.Run(); err != nil {
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "clone_successful", "passed": false, "detail": err.Error(),
		})
		if !keepClone {
			os.RemoveAll(sandbox)
		}
		return result
	}

	result.ClonePath = clonePath
	result.Checks = append(result.Checks, map[string]interface{}{
		"name": "clone_successful", "passed": true,
	})

	// Check: validate_all (Go binary if available, fallback skip)
	goValidate := filepath.Join(clonePath, "go-runtime")
	if pathExists(goValidate) {
		valCmd := exec.Command("go", "test", "-count=1", "-short", "./...")
		valCmd.Dir = goValidate
		valCmd.Env = append(os.Environ(), "GOFLAGS=-short")
		valOut, valErr := valCmd.CombinedOutput()
		passed := valErr == nil
		result.ValidatePassed = passed
		detail := "passed"
		if !passed {
			detail = truncate(string(valOut), 500)
		}
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "validate_all", "passed": passed, "detail": detail,
		})
	} else {
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "validate_all", "passed": true, "detail": "skipped (no go-runtime)",
		})
		result.ValidatePassed = true
	}

	// Check: go build smoke
	buildCmd := exec.Command("go", "build", "-o", "/dev/null", "./cmd/ovav/")
	buildCmd.Dir = goValidate
	buildOut, buildErr := buildCmd.CombinedOutput()
	buildPassed := buildErr == nil
	result.SmokePassed = buildPassed
	smokeDetail := "exit=0"
	if !buildPassed {
		smokeDetail = truncate(string(buildOut), 500)
	}
	result.Checks = append(result.Checks, map[string]interface{}{
		"name": "smoke_test", "passed": buildPassed, "detail": smokeDetail,
	})

	// Overall
	result.OverallOK = true
	for _, c := range result.Checks {
		if !c["passed"].(bool) {
			result.OverallOK = false
			break
		}
	}

	if !keepClone {
		os.RemoveAll(sandbox)
	}

	return result
}

// FreshCloneSmokeJSON runs the smoke test and returns JSON bytes.
func FreshCloneSmokeJSON(repoRoot string, keepClone bool) ([]byte, error) {
	result := FreshCloneSmoke(repoRoot, keepClone)
	return json.MarshalIndent(result, "", "  ")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
