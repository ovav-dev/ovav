// OVAV Coverage Gate — CLI tool for CI coverage enforcement.
//
// Runs go test -coverprofile in go-runtime/, parses total coverage,
// and enforces a minimum threshold. Designed for CI/CD pipelines
// and local pre-push hooks.
//
// USAGE:
//
//	coverage_gate                        # default: 70% threshold
//	coverage_gate --min 80               # custom threshold
//	coverage_gate --json                  # JSON output
//	coverage_gate --quick                 # skip race detector (faster)
//
// Exit codes:
//
//	0 — coverage passes threshold
//	1 — coverage below threshold
//	2 — runtime error (build failed, etc.)
//
// Stack: Go 1.25+, stdlib only. Zero Python dependencies.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Output represents the structured output of the coverage gate.
type Output struct {
	Total     float64 `json:"total_pct"`
	Threshold float64 `json:"threshold_pct"`
	Pass      bool    `json:"pass"`
	GoRuntime string  `json:"go_runtime_dir"`
	CoverFile string  `json:"cover_file"`
	Error     string  `json:"error,omitempty"`
}

func main() {
	minPct := flag.Float64("min", 70.0, "Minimum coverage percentage")
	jsonOut := flag.Bool("json", false, "JSON output")
	quick := flag.Bool("quick", false, "Skip -race flag (faster)")
	goRuntime := flag.String("dir", "", "go-runtime directory (default: auto-detect)")
	flag.Parse()

	// Find go-runtime directory
	runtimeDir := *goRuntime
	if runtimeDir == "" {
		runtimeDir = findGoRuntime()
	}
	if runtimeDir == "" {
		exitError("cannot find go-runtime/ directory", *jsonOut)
	}

	absDir, err := filepath.Abs(runtimeDir)
	if err != nil {
		exitError(fmt.Sprintf("cannot resolve path: %v", err), *jsonOut)
	}

	// Run go test with coverage (exclude coverage_gate itself to avoid circular test)
	coverFile := filepath.Join(os.TempDir(), "ovav-coverage-gate.out")
	args := []string{"test", "-count=1", "-coverprofile=" + coverFile}
	if !*quick {
		args = append(args, "-race")
	}
	// Test all packages except coverage_gate itself
	args = append(args, "./internal/...", "./cmd/cockpit/...", "./cmd/cpanel/...",
		"./cmd/convert_agents/...", "./cmd/ovav/...", "./cmd/tailor/...",
		"./cmd/chronos_gate/...", "./cmd/output_guard/...")

	cmd := exec.Command("go", args...)
	cmd.Dir = absDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := stderrBuf.String()
		exitError(fmt.Sprintf("go test failed: %v\nstderr: %s", err, strings.TrimSpace(stderr)), *jsonOut)
	}

	// Parse total coverage from cover profile
	total, err := parseTotalCoverage(coverFile)
	if err != nil {
		exitError(fmt.Sprintf("cannot parse coverage: %v", err), *jsonOut)
	}

	pass := total >= *minPct

	out := Output{
		Total:     total,
		Threshold: *minPct,
		Pass:      pass,
		GoRuntime: absDir,
		CoverFile: coverFile,
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		icon := "✅"
		if !pass {
			icon = "❌"
		}
		fmt.Printf("%s Coverage: %.1f%% (threshold: %.0f%%) — %s\n",
			icon, total, *minPct, passStr(pass))
	}

	if !pass {
		os.Exit(1)
	}
}

func passStr(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

func exitError(msg string, jsonOut bool) {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(Output{Error: msg})
	} else {
		fmt.Fprintf(os.Stderr, "❌ coverage_gate: %s\n", msg)
	}
	os.Exit(2)
}

// parseTotalCoverage extracts the total coverage percentage from a coverage.out file.
func parseTotalCoverage(path string) (float64, error) {
	// Use go tool cover -func to get the total line
	cmd := exec.Command("go", "tool", "cover", "-func="+path)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("go tool cover failed: %w", err)
	}

	// Last line looks like: "total: (statements) 68.4%"
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	lastLine := lines[len(lines)-1]
	fields := strings.Fields(lastLine)
	if len(fields) < 2 {
		return 0, fmt.Errorf("unexpected format: %q", lastLine)
	}

	// The last field should be "XX.X%"
	pctStr := strings.TrimSuffix(fields[len(fields)-1], "%")
	return strconv.ParseFloat(pctStr, 64)
}

// findGoRuntime walks up from cwd looking for go-runtime/ directory with go.mod.
func findGoRuntime() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	// Also check common relative paths from repo root
	for _, dir := range []string{
		cwd,
		filepath.Join(cwd, "go-runtime"),
	} {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	// Walk up looking for a repo with go-runtime/
	dir := cwd
	for range 12 {
		candidate := filepath.Join(dir, "go-runtime")
		if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
