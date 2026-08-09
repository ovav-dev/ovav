// Package main implements the ovav test command.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/testing/advance"
)

func main() {
	// Get root directory
	root := os.Getenv("OVAV_ROOT")
	if root == "" {
		root = "."
	}

	// Parse flags
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	interactive := runCmd.Bool("interactive", false, "Enable interactive mode")
	autonomous := runCmd.Bool("autonomous", true, "Enable autonomous mode")
	targetCoverage := runCmd.Float64("coverage", 0.8, "Target coverage (0.0-1.0)")
	timeout := runCmd.Duration("timeout", 5*time.Minute, "Timeout per test")
	packages := runCmd.String("packages", "", "Comma-separated packages to test")
	ovavSystem := runCmd.Bool("ovav-system", true, "Enable OVAV SYSTEM integration")
	securityOnly := runCmd.Bool("security", false, "Run security-only tests")

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run(runCmd, *interactive, *autonomous, *targetCoverage, *timeout, *packages, *ovavSystem, *securityOnly, root)
	case "list":
		list(root)
	case "coverage":
		coverage_cmd(*targetCoverage, root)
	default:
		printUsage()
		os.Exit(1)
	}
}

func run(cmd *flag.FlagSet, interactive, autonomous bool, targetCov float64, timeout time.Duration, packages string, ovavSystemFlag, securityOnly bool, root string) {
	cmd.Parse(os.Args[2:])
	if cmd.NArg() > 0 {
		cmd.Usage()
		os.Exit(1)
	}

	// Build config
	cfg := &advance.Config{
		Interactive:     interactive,
		Autonomous:      autonomous,
		TargetCoverage:  targetCov,
		Timeout:         timeout,
		OVAVSystem:      ovavSystemFlag,
	}

	if packages != "" {
		cfg.Packages = strings.Split(packages, ",")
	}

	// Create advance
	adv := advance.New(cfg)

	fmt.Println("🧪 OVAV Testing Advance")
	fmt.Println()
	fmt.Printf("  Mode: %s\n", modeString(interactive, autonomous))
	fmt.Printf("  Target Coverage: %.0f%%\n", targetCov*100)
	fmt.Printf("  Timeout: %s\n", timeout)
	fmt.Printf("  OVAV System: %v\n", ovavSystemFlag)
	fmt.Println()

	// Run tests
	fmt.Println("Running tests...")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var report *advance.Report
	var err error

	if securityOnly {
		report, err = adv.RunSecurityOnly(ctx, cfg.Packages)
	} else {
		report, err = adv.Run(ctx, cfg.Packages)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running tests: %v\n", err)
		os.Exit(1)
	}

	elapsed := report.EndTime.Sub(report.StartTime)

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Test Results")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Duration: %s\n", elapsed.Round(time.Second))
	fmt.Printf("  Packages: %d\n", len(report.Packages))
	fmt.Printf("  Iterations: %d\n", report.Iterations)
	fmt.Printf("  Baseline Coverage: %.1f%%\n", report.BaselineCoverage*100)
	fmt.Printf("  Final Coverage: %.1f%%\n", report.FinalCoverage*100)
	fmt.Printf("  Alerts: %d\n", len(report.Alerts))

	if report.FinalCoverage >= targetCov {
		fmt.Println()
		fmt.Printf("✅ Coverage target (%.0f%%) met!\n", targetCov*100)
	} else {
		fmt.Println()
		fmt.Printf("⚠️  Coverage target (%.0f%%) not met (%.1f%%)\n", targetCov*100, report.FinalCoverage*100)
	}

	if len(report.Alerts) > 0 {
		fmt.Println()
		fmt.Println("🚨 Alerts:")
		for _, alert := range report.Alerts {
			fmt.Printf("  [%s] %s\n", alert.Level, alert.Title)
		}
	}

	if report.MutationReport != nil {
		fmt.Println()
		fmt.Println("🧬 Mutation Testing:")
		fmt.Printf("  Total: %d\n", report.MutationReport.MutationsTotal)
		fmt.Printf("  Alive: %d\n", report.MutationReport.MutationsAlive)
		fmt.Printf("  Killed: %d\n", report.MutationReport.MutationsKilled)
		fmt.Printf("  Score: %.1f%%\n", report.MutationReport.Score*100)
	}

	if report.FutureReport != nil && len(report.FutureReport.Predictions) > 0 {
		fmt.Println()
		fmt.Println("🔮 Future Vulnerabilities Predicted:")
		for _, p := range report.FutureReport.Predictions {
			fmt.Printf("  ⚠️  %s (%s)\n", p.Pattern, p.Severity)
			fmt.Printf("     %s\n", p.Reason)
		}
	}

	if report.FutureReport != nil && report.FutureReport.SecurityReport != nil {
		sr := report.FutureReport.SecurityReport
		if sr.Total > 0 {
			fmt.Println()
			fmt.Println("🔒 Security Findings:")
			fmt.Printf("  Total: %d\n", sr.Total)
			if sr.Critical != nil {
				fmt.Printf("  Critical: %d\n", len(sr.Critical))
			}
			if sr.High != nil {
				fmt.Printf("  High: %d\n", len(sr.High))
			}
			if sr.Medium != nil {
				fmt.Printf("  Medium: %d\n", len(sr.Medium))
			}
			if sr.Low != nil {
				fmt.Printf("  Low: %d\n", len(sr.Low))
			}
		}
	}

	os.Exit(0)
}

func list(root string) {
	fmt.Println("📦 Available Test Packages")
	fmt.Println()

	// Find test packages
	testDir := filepath.Join(root, "go-runtime")
	packages := findTestPackages(testDir)

	if len(packages) == 0 {
		fmt.Println("  No test packages found.")
		return
	}

	for _, pkg := range packages {
		rel, _ := filepath.Rel(testDir, pkg)
		fmt.Printf("  • %s\n", rel)
	}
}

func coverage_cmd(targetCov float64, root string) {
	cfg := &advance.Config{
		TargetCoverage: targetCov,
	}

	adv := advance.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	report, err := adv.Run(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running coverage: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Coverage: %.1f%%\n", report.FinalCoverage*100)
	fmt.Printf("Target: %.1f%%\n", targetCov*100)

	if report.FinalCoverage >= targetCov {
		fmt.Println("✅ Coverage target met!")
	} else {
		fmt.Println("❌ Coverage target not met")
	}
}

func findTestPackages(dir string) []string {
	var packages []string

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			testFile := filepath.Join(path, "core_test.go")
			if _, err := os.Stat(testFile); err == nil {
				packages = append(packages, path)
			}
		}
		return nil
	})

	return packages
}

func modeString(interactive, autonomous bool) string {
	if interactive {
		return "Interactive"
	}
	if autonomous {
		return "Autonomous"
	}
	return "Manual"
}

func printUsage() {
	fmt.Print(`OVAV Testing - Autonomous Test Runner

Usage:
  ovav test <command>

Commands:
  run           Run all tests with autonomous vulnerability detection
  run --security Run security-only tests
  list          List available test packages
  coverage      Check coverage report

Options:
  --interactive    Enable interactive mode
  --autonomous    Enable autonomous mode (default: true)
  --coverage N    Target coverage (0.0-1.0, default: 0.8)
  --timeout D     Timeout per test (default: 5m)
  --packages P    Comma-separated packages to test
  --ovav-system   Enable OVAV SYSTEM integration (default: true)
  --security      Run security-only tests

Examples:
  ovav test run                # Full autonomous test run
  ovav test run --coverage 0.9 # Higher coverage target
  ovav test run --security     # Security tests only
  ovav test list               # Show available packages
  ovav test coverage           # Coverage report
`)
}
