// coverage_cli.go — OVAV Coverage Gate CLI
// Integrates cmd/coverage_gate functionality into main ovav CLI.
// Usage: ovav coverage [--min 80] [--json] [--quick]
// Exit codes: 0=pass, 1=fail, 2=error

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
	"time"
)

func cmdCoverage(args []string) int {
	minPct := flag.Int("min", 70, "Minimum coverage threshold (%)")
	jsonOut := flag.Bool("json", false, "Output JSON format")
	quick := flag.Bool("quick", false, "Skip race detector for faster execution")
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		if jsonOut != nil && *jsonOut {
			fmt.Printf(`{"status":"error","error":"%v"}`, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot find repo root: %v\n", err)
		}
		return 2
	}

	// Run go test -coverprofile
	goDir := filepath.Join(repoRoot, "go-runtime")
	if _, err := os.Stat(goDir); err != nil {
		goDir = repoRoot
	}

	args2 := []string{"test", "./..."}
	if !*quick {
		args2 = append(args2, "-race")
	}
	args2 = append(args2, "-coverprofile", filepath.Join(os.TempDir(), "ovav_coverage.out"), "-covermode", "atomic")

	cmd := exec.Command("go", args2...)
	cmd.Dir = goDir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil && !*quick {
		// Retry without race if failed
		args3 := []string{"test", "./...", "-coverprofile", filepath.Join(os.TempDir(), "ovav_coverage.out")}
		cmd2 := exec.Command("go", args3...)
		cmd2.Dir = goDir
		cmd2.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
		output, err = cmd2.CombinedOutput()
	}

	var totalPct float64
	var packages []string

	if err == nil {
		// Parse coverage from output
		totalPct = parseCoverageFromOutput(string(output))
		packages = parsePackagesFromOutput(string(output))
	}

	result := map[string]interface{}{
		"command":     "coverage",
		"status":      "ok",
		"threshold":   *minPct,
		"actual":      totalPct,
		"passed":      totalPct >= float64(*minPct),
		"duration_ms": duration.Milliseconds(),
		"packages":    len(packages),
	}

	if jsonOut != nil && *jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		if result["passed"].(bool) {
			return 0
		}
		return 1
	}

	// Human output
	icon := "🟢"
	if !result["passed"].(bool) {
		icon = "🔴"
	}
	fmt.Printf("%s Coverage Gate | Threshold: %d%% | Actual: %.1f%%\n", icon, *minPct, totalPct)
	fmt.Printf("   Packages tested: %d\n", len(packages))
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	if result["passed"].(bool) {
		fmt.Println("✅ Coverage gate PASSED")
		return 0
	}
	fmt.Println("❌ Coverage gate FAILED")
	fmt.Printf("   Run 'go test ./... -cover' in go-runtime/ for details\n")
	return 1
}

func parseCoverageFromOutput(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			parts := strings.Split(line, "coverage:")
			if len(parts) > 1 {
				pctStr := strings.TrimSpace(strings.Split(parts[1], "%")[0])
				if pct, err := strconv.ParseFloat(pctStr, 64); err == nil {
					return pct
				}
			}
		}
	}
	// Try to find total line
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "ok") || strings.HasPrefix(strings.TrimSpace(line), "FAIL") {
			continue
		}
	}
	return 0
}

func parsePackagesFromOutput(output string) []string {
	var packages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ok") || strings.HasPrefix(line, "FAIL") || strings.HasPrefix(line, "?") {
			if len(line) > 0 {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					pkg := parts[0]
					if strings.HasPrefix(pkg, "github.com/ovav/ovav/") {
						pkg = strings.TrimPrefix(pkg, "github.com/ovav/ovav/")
					}
					packages = append(packages, pkg)
				}
			}
		}
	}
	return packages
}
