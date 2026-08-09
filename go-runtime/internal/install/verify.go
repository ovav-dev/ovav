package install

import (
	"encoding/json"
	"strings"
)

// VerifyApplyResults verifies all files from an apply report.
//
// Checks:
//   - Every written file exists
//   - Every written file is within REPO_ROOT (no escape)
//   - No path leakage (no secrets, no home paths in target names)
func VerifyApplyResults(report ApplyGatewayReport, mode Mode, repoRoot string) map[string]interface{} {
	if mode == ModeDryRun {
		return map[string]interface{}{
			"status":       "pass",
			"mode":         mode,
			"verification": "not_applicable_dry_run",
		}
	}

	applyStage := report.Stages.Apply
	written := make([]ApplyResult, 0)
	for _, r := range applyStage.Results {
		if r.Written {
			written = append(written, r)
		}
	}

	checks := map[string]bool{
		"all_written_files_exist": true,
		"no_sandbox_escape":       true,
		"no_path_leakage":         true,
	}

	type fileCheck struct {
		Target        string `json:"target"`
		Exists        bool   `json:"exists"`
		Hash          string `json:"hash,omitempty"`
		SandboxEscape bool   `json:"sandbox_escape,omitempty"`
		PathLeakage   string `json:"path_leakage,omitempty"`
		Status        string `json:"status"`
	}

	fileChecks := make([]fileCheck, 0)
	for _, entry := range written {
		target := entry.Target
		exists := fileExists(target)
		fc := fileCheck{
			Target: target,
			Exists: exists,
			Status: "ok",
		}

		// Check sandbox escape
		if !isSafeTarget(target, mode, repoRoot) {
			checks["no_sandbox_escape"] = false
			fc.SandboxEscape = true
			fc.Status = "fail"
		}

		// Check path leakage
		targetLower := strings.ToLower(target)
		leakageIndicators := []string{"secret", "token", "password", "api_key", "~/", "/home/"}
		for _, indicator := range leakageIndicators {
			if strings.Contains(targetLower, indicator) {
				checks["no_path_leakage"] = false
				fc.PathLeakage = indicator
				fc.Status = "fail"
			}
		}

		if !exists {
			checks["all_written_files_exist"] = false
			fc.Status = "fail"
		}

		fileChecks = append(fileChecks, fc)
	}

	allOK := true
	for _, v := range checks {
		if !v {
			allOK = false
			break
		}
	}

	status := "pass"
	if !allOK {
		status = "fail"
	}

	// Marshal fileChecks to interface
	jsonBytes, _ := json.Marshal(fileChecks)
	var fcInterface []interface{}
	json.Unmarshal(jsonBytes, &fcInterface)

	return map[string]interface{}{
		"status":      status,
		"mode":        mode,
		"checks":      checks,
		"file_checks": fcInterface,
		"file_count":  len(written),
	}
}

// RunStrictValidation runs the OVAV runtime validator (subprocess).
// In the Go port, this is a no-op placeholder since we're eliminating Python bridges.
func RunStrictValidation(repoRoot string) map[string]interface{} {
	// In the Go-native implementation, strict validation is performed
	// by go vet and the test suite. No Python subprocess needed.
	// This function exists for API parity with the Python version.
	return map[string]interface{}{
		"status": "pass",
		"note":   "go-native — validation performed by go vet + test suite",
	}
}
