package doctor

import (
	"strings"
	"testing"

	"github.com/ovav/ovav/internal/cli"
)

func TestRunQuick(t *testing.T) {
	results := Run(true)
	if len(results) == 0 {
		t.Fatal("Run(quick=true) returned no results")
	}

	// Quick mode should include these core checks
	required := []string{"go-runtime", "git-available", "git-repo", "branch-safety", "ovav-root"}
	for _, name := range required {
		found := false
		for _, r := range results {
			if r.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Quick mode missing required check: %s", name)
		}
	}
}

func TestRunFull(t *testing.T) {
	results := Run(false)
	if len(results) == 0 {
		t.Fatal("Run(quick=false) returned no results")
	}

	// Full mode should have more checks than quick
	quickResults := Run(true)
	if len(results) <= len(quickResults) {
		t.Errorf("Full mode (%d checks) should have more than quick mode (%d)",
			len(results), len(quickResults))
	}
}

func TestAllChecksHaveValidStatus(t *testing.T) {
	results := Run(false)
	validStatuses := map[string]bool{
		"pass": true, "fail": true, "warn": true, "skip": true,
	}

	for _, r := range results {
		if !validStatuses[r.Status] {
			t.Errorf("Check %q has invalid status: %q", r.Name, r.Status)
		}
		if r.Name == "" {
			t.Error("Check has empty name")
		}
		if r.Detail == "" {
			t.Errorf("Check %q has empty detail", r.Name)
		}
	}
}

func TestNoDuplicateChecks(t *testing.T) {
	results := Run(false)
	seen := make(map[string]bool)
	for _, r := range results {
		if seen[r.Name] {
			t.Errorf("Duplicate check name: %q", r.Name)
		}
		seen[r.Name] = true
	}
}

func TestFormatResults(t *testing.T) {
	results := Run(false)
	output := FormatResults(results)

	// Should contain the header
	if !strings.Contains(output, "OVAV Doctor") {
		t.Error("FormatResults missing header")
	}

	// Should contain pass/fail counts
	if !strings.Contains(output, "passed") {
		t.Error("FormatResults missing pass count")
	}
}

func TestFormatResultsEmpty(t *testing.T) {
	output := FormatResults(nil)
	if !strings.Contains(output, "OVAV Doctor") {
		t.Error("FormatResults empty: missing header")
	}
}

func TestCheckGoRuntime(t *testing.T) {
	r := checkGoRuntime()
	if r.Status != "pass" {
		t.Errorf("go-runtime check should pass, got %s: %s", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "Go") {
		t.Error("go-runtime detail should mention Go version")
	}
}

func TestCheckGitAvailable(t *testing.T) {
	r := checkGitAvailable()
	if r.Status != "pass" && r.Status != "fail" {
		t.Errorf("git-available: unexpected status %s", r.Status)
	}
}

func TestCheckBranchSafety(t *testing.T) {
	r := checkBranchSafety()
	if r.Status == "" {
		t.Error("branch-safety: empty status")
	}
	// In a task branch, should be "pass"
	branch, _, _ := cli.GitInfo()
	if strings.HasPrefix(branch, "task/") || strings.HasPrefix(branch, "feature/") {
		if r.Status != "pass" {
			t.Logf("branch-safety on task branch: %s — %s", r.Status, r.Detail)
		}
	}
}

func TestCheckOVAVRoot(t *testing.T) {
	r := checkOVAVRoot()
	if r.Status == "fail" && strings.Contains(r.Detail, "Not in an OVAV repository") {
		t.Skip("Not in OVAV repo — skipping root check")
	}
	if r.Status != "pass" && r.Status != "warn" {
		t.Errorf("ovav-root: unexpected status %s: %s", r.Status, r.Detail)
	}
}

func TestQuickModeFewerChecks(t *testing.T) {
	full := Run(false)
	quick := Run(true)

	if len(quick) >= len(full) {
		t.Errorf("Quick mode (%d) should have fewer checks than full (%d)",
			len(quick), len(full))
	}

	// Quick mode should not include slow checks
	slowChecks := []string{"go-version", "disk-space", "authority-contract"}
	for _, name := range slowChecks {
		for _, r := range quick {
			if r.Name == name {
				t.Errorf("Quick mode should not include %q", name)
			}
		}
	}
}
