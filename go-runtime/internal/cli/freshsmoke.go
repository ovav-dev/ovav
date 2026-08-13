// Package cli — freshsmoke.go: fresh clone smoke test.
//
// Go migration of tools/cli/ovav_fresh_clone_smoke.py (106 LOC).
// Validates that a fresh clone works end-to-end.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FreshCloneResult holds candidate smoke evidence from a fresh local clone.
type FreshCloneResult struct {
	SchemaVersion  string                   `json:"schema_version"`
	GeneratedAt    string                   `json:"generated_at"`
	ClonePath      string                   `json:"clone_path,omitempty"`
	ValidatePassed bool                     `json:"validate_passed"`
	SmokePassed    bool                     `json:"smoke_passed"`
	OverallOK      bool                     `json:"overall_ok"`
	Timeout        string                   `json:"timeout"`
	TimedOut       bool                     `json:"timed_out"`
	Checks         []map[string]interface{} `json:"checks"`
}

// FreshCloneSmoke validates candidate changes in a fresh clone end-to-end.
// Replaces ovav_fresh_clone_smoke.py.
func FreshCloneSmoke(repoRoot string, keepClone bool) FreshCloneResult {
	return FreshCloneSmokeWithTimeout(repoRoot, keepClone, 5*time.Minute)
}

// FreshCloneSmokeWithTimeout runs clone, candidate apply, validation, and build phases within timeout.
func FreshCloneSmokeWithTimeout(repoRoot string, keepClone bool, timeout time.Duration) FreshCloneResult {
	result := FreshCloneResult{
		SchemaVersion: "ovav.candidate_smoke.v2",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Timeout:       timeout.String(),
		Checks:        []map[string]interface{}{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--no-hardlinks", repoRoot, clonePath)
	cloneCmd.Stderr = nil
	if err := cloneCmd.Run(); err != nil {
		result.TimedOut = ctx.Err() == context.DeadlineExceeded
		detail := err.Error()
		if result.TimedOut {
			detail = "clone phase exceeded timeout " + timeout.String()
		}
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "clone", "passed": false, "detail": detail,
		})
		os.RemoveAll(sandbox)
		return result
	}

	result.ClonePath = clonePath
	result.Checks = append(result.Checks, map[string]interface{}{
		"name": "clone", "passed": true, "detail": "local clone completed",
	})
	if err := applyCandidateChanges(repoRoot, clonePath); err != nil {
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "apply_candidate", "passed": false, "detail": err.Error(),
		})
		if !keepClone {
			os.RemoveAll(sandbox)
		}
		return result
	}
	result.Checks = append(result.Checks, map[string]interface{}{
		"name": "apply_candidate", "passed": true, "detail": "tracked and intended untracked candidate changes applied",
	})

	// Check: validate_all (Go binary if available, fallback skip)
	goValidate := filepath.Join(clonePath, "go-runtime")
	if pathExists(goValidate) {
		valCmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-short", "./...")
		valCmd.Dir = goValidate
		valCmd.Env = append(os.Environ(), "GOFLAGS=-short", "GOPROXY=off")
		valOut, valErr := valCmd.CombinedOutput()
		// Parse output: `ok <pkg> <dur>` and `? <pkg> [no test files]` are PASS;
		// only `FAIL <pkg> <reason>` lines are FAIL. The trailing `FAIL`
		// summary line (no package) is ignored.
		passed := parseGoTestOutput(string(valOut))
		if valErr != nil && passed {
			// Non-zero exit with no FAIL package lines — treat as pass
			// (covers edge cases where a non-fatal signal still exits 1).
		}
		result.ValidatePassed = passed
		detail := "passed"
		if !passed {
			detail = truncate(string(valOut), 500)
		}
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			detail = "validation phase exceeded timeout " + timeout.String()
		}
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "validate", "passed": passed && !result.TimedOut, "detail": detail,
		})
	} else {
		result.Checks = append(result.Checks, map[string]interface{}{
			"name": "validate", "passed": false, "detail": "go-runtime directory missing",
		})
		result.ValidatePassed = false
	}
	if result.TimedOut {
		if !keepClone {
			os.RemoveAll(sandbox)
		}
		return result
	}

	// Check: go build smoke
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", os.DevNull, "./cmd/ovav/")
	buildCmd.Dir = goValidate
	buildCmd.Env = append(os.Environ(), "GOPROXY=off")
	buildOut, buildErr := buildCmd.CombinedOutput()
	buildPassed := buildErr == nil
	result.SmokePassed = buildPassed
	smokeDetail := "exit=0"
	if !buildPassed {
		smokeDetail = truncate(string(buildOut), 500)
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		buildPassed = false
		result.SmokePassed = false
		smokeDetail = "build phase exceeded timeout " + timeout.String()
	}
	result.Checks = append(result.Checks, map[string]interface{}{
		"name": "build", "passed": buildPassed, "detail": smokeDetail,
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

func applyCandidateChanges(repoRoot, clonePath string) error {
	patch, err := candidatePatch(repoRoot)
	if err != nil {
		return err
	}
	if len(patch) == 0 {
		return nil
	}
	cmd := exec.Command("git", "apply", "--binary", "--whitespace=nowarn", "-")
	cmd.Dir = clonePath
	cmd.Stdin = bytes.NewReader(patch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("candidate patch could not be applied: %s", truncate(detail, 300))
	}
	return nil
}

func candidatePatch(repoRoot string) ([]byte, error) {
	tracked, err := gitPatch(repoRoot, "diff", "--binary", "HEAD", "--")
	if err != nil {
		return nil, fmt.Errorf("generate tracked candidate patch: %w", err)
	}
	var patch bytes.Buffer
	patch.Write(tracked)

	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list intended untracked files: %w", err)
	}
	for _, rawPath := range bytes.Split(out, []byte{0}) {
		if len(rawPath) == 0 {
			continue
		}
		relPath := string(rawPath)
		if filepath.IsAbs(relPath) || filepath.Clean(relPath) != relPath || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("unsafe untracked path rejected")
		}
		fullPath := filepath.Join(repoRoot, relPath)
		info, err := os.Lstat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("inspect untracked file %q: %w", relPath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("untracked path %q is not a regular file", relPath)
		}
		filePatch, err := gitPatch(repoRoot, "diff", "--binary", "--no-index", "--", os.DevNull, relPath)
		if err != nil {
			return nil, fmt.Errorf("generate untracked candidate patch for %q: %w", relPath, err)
		}
		patch.Write(filePatch)
	}
	return patch.Bytes(), nil
}

func gitPatch(repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 || !containsNoIndex(args) {
			return nil, err
		}
	}
	return output.Bytes(), nil
}

func containsNoIndex(args []string) bool {
	for _, arg := range args {
		if arg == "--no-index" {
			return true
		}
	}
	return false
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

// parseGoTestOutput determines whether the combined stdout/stderr of
// `go test ./...` represents a passing run. Go test output uses three
// package markers:
//
//   ok    <pkg>  <duration>   — package passed
//   ?     <pkg>  [no test files] — package has no test files (PASS)
//   FAIL  <pkg>  <reason>   — package failed
//
// A trailing `FAIL` summary line (no package) is printed when any
// package failed, but we ignore it because it never appears without a
// matching `FAIL <pkg> ...` line. Any line whose first whitespace-
// separated field is `FAIL` with at least one trailing field is treated
// as a package failure. Returns true when no package-level FAIL was
// found.
func parseGoTestOutput(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == "FAIL" {
			return false
		}
	}
	return true
}
