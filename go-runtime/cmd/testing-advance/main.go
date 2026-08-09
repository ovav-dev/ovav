// Package advance — OVAV Testing Advance CLI
//
// Usage:
//
//	go run ./cmd/testing-advance/ [flags]
//
// Flags:
//
//	-packages strings   packages to test (default all internal + cmd)
//	-target float      target coverage 0.0-1.0 (default 0.80)
//	-iterations int    max iterations (default 0 = unlimited)
//	-autonomous        run without operator input
//	-interactive       enable interactive mode
//	-ovav-system      enable OVAV SYSTEM alerts (default true)
//	-report string     output file for JSON report
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ovav/ovav/internal/testing/advance"
)

func main() {
	var packagesStr string
	var target float64
	var maxIterations int
	var autonomous, interactive, ovavSystem bool
	var reportFile string

	flag.StringVar(&packagesStr, "packages", "", "comma-separated packages (default: all)")
	flag.Float64Var(&target, "target", 0.80, "target coverage 0.0-1.0")
	flag.IntVar(&maxIterations, "iterations", 0, "max attack iterations (0=unlimited)")
	flag.BoolVar(&autonomous, "autonomous", false, "fully autonomous mode")
	flag.BoolVar(&interactive, "interactive", false, "interactive operator mode")
	flag.BoolVar(&ovavSystem, "ovav-system", true, "enable OVAV SYSTEM integration")
	flag.StringVar(&reportFile, "report", "", "output file for JSON report")
	securityOnly := flag.Bool("security-only", false, "skip coverage loop, run security probes only (fast mode)")
	flag.Parse()

	cfg := &advance.Config{
		TargetCoverage: target,
		MaxIterations:  maxIterations,
		Autonomous:     autonomous,
		Interactive:    interactive,
		OVAVSystem:     ovavSystem,
		Timeout:        10 * time.Minute,
	}

	var packages []string
	if packagesStr != "" {
		for _, p := range splitComma(packagesStr) {
			packages = append(packages, p)
		}
	}

	fmt.Printf("🚀 OVAV Testing Advance\n")
	if *securityOnly {
		fmt.Printf("   Mode: 🔒 SECURITY-ONLY (fast)\n")
	} else {
		fmt.Printf("   Mode: FULL (coverage + security)\n")
	}
	fmt.Printf("   Target: %.0f%%\n", target*100)
	fmt.Printf("   Packages: %d\n", len(packages))
	fmt.Printf("   Autonomous: %v\n", autonomous)
	fmt.Printf("   OVAV System: %v\n\n", ovavSystem)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Always set a generous timeout; cfg.Timeout gates per-command timeouts
	ctx, cancel = context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()

	adv := advance.New(cfg)
	var report *advance.Report
	var err error
	if *securityOnly {
		report, err = adv.RunSecurityOnly(ctx, packages)
	} else {
		report, err = adv.Run(ctx, packages)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("\n✅ Testing Advance Complete\n")
	fmt.Printf("   Iterations: %d\n", report.Iterations)
	fmt.Printf("   Baseline:   %.1f%%\n", report.BaselineCoverage*100)
	fmt.Printf("   Final:      %.1f%%\n", report.FinalCoverage*100)
	fmt.Printf("   Gain:       %+.1fpp\n", (report.FinalCoverage-report.BaselineCoverage)*100)
	fmt.Printf("   Duration:   %s\n", report.EndTime.Sub(report.StartTime).Round(time.Second))

	if report.MutationReport != nil && report.MutationReport.MutationsTotal > 0 {
		fmt.Printf("\n🧪 Mutation Testing\n")
		fmt.Printf("   Total:    %d\n", report.MutationReport.MutationsTotal)
		fmt.Printf("   Killed:   %d\n", report.MutationReport.MutationsKilled)
		fmt.Printf("   Alive:    %d\n", report.MutationReport.MutationsAlive)
		fmt.Printf("   Score:    %.1f%%\n", report.MutationReport.Score*100)
	}

	if report.FutureReport != nil && len(report.FutureReport.Predictions) > 0 {
		fmt.Printf("\n🔮 Future Predictions: %d\n", len(report.FutureReport.Predictions))
		for _, p := range report.FutureReport.Predictions[:min(5, len(report.FutureReport.Predictions))] {
			fmt.Printf("   [%s] %s:%d — %s (%s)\n", p.Severity, p.File, p.Line, p.Pattern, p.Reason)
		}
	}

	// Print security report if autonomous mode ran
	if report.FutureReport != nil && report.FutureReport.SecurityReport != nil {
		sr := report.FutureReport.SecurityReport
		fmt.Printf("\n🚨 Security Report (Autonomous Attack)\n")
		fmt.Printf("   Total findings:   %d\n", sr.Total)
		fmt.Printf("   Critical:        %d\n", len(sr.Critical))
		fmt.Printf("   High:            %d\n", len(sr.High))
		fmt.Printf("   Medium:          %d\n", len(sr.Medium))
		fmt.Printf("   Low:             %d\n", len(sr.Low))
		if len(sr.Critical) > 0 {
			fmt.Printf("\n   🚨 CRITICAL VULNERABILITIES:\n")
			for _, f := range sr.Critical {
				fmt.Printf("   [%s] %s:%d — %s\n", f.Severity, f.File, f.Line, f.Pattern)
			}
		}
		if len(sr.High) > 0 {
			fmt.Printf("\n   ⚠️ HIGH VULNERABILITIES:\n")
			for _, f := range sr.High {
				fmt.Printf("   [%s] %s:%d — %s\n", f.Severity, f.File, f.Line, f.Pattern)
			}
		}
		if len(sr.Medium) > 0 {
			fmt.Printf("\n   🔶 MEDIUM VULNERABILIDADES:\n")
			for _, f := range sr.Medium {
				fmt.Printf("   [%s] %s:%d — %s\n", f.Severity, f.File, f.Line, f.Pattern)
			}
		}
		if len(sr.Low) > 0 {
			fmt.Printf("\n   🔹 LOW VULNERABILIDADES:\n")
			for _, f := range sr.Low {
				fmt.Printf("   [%s] %s:%d — %s\n", f.Severity, f.File, f.Line, f.Pattern)
			}
		}
	}

	// Save report
	if reportFile != "" {
		data, _ := report.ReportJSON()
		os.WriteFile(reportFile, data, 0644)
		fmt.Printf("\n📄 Report saved: %s\n", reportFile)
	}
}

func splitComma(s string) []string {
	var result []string
	for _, part := range splitString(s, ",") {
		part = trimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for {
		idx := indexString(s, sep, start)
		if idx < 0 {
			result = append(result, s[start:])
			break
		}
		result = append(result, s[start:idx])
		start = idx + len(sep)
	}
	return result
}

func indexString(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}
