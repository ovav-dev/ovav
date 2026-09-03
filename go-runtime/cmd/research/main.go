// Package main implements the ovav research command.
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ovav/ovav/internal/autonomous/engine"
	"github.com/ovav/ovav/internal/autonomous/scheduler"
)

const exampleDir = "/tmp/ovav-research"

func main() {
	// Get root directory
	root := os.Getenv("OVAV_ROOT")
	if root == "" {
		root = "."
	}

	cfg := engine.Config{
		DataDir: root + "/.ovav/intelligence",
		Timeout: 30 * time.Second,
	}

	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing research engine: %v\n", err)
		os.Exit(1)
	}

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		run(eng)
	case "status":
		status(eng)
	case "findings":
		findings(eng)
	case "targets":
		targets(eng)
	case "analyze":
		analyze(eng)
	default:
		printUsage()
		os.Exit(1)
	}
}

func run(eng *engine.Engine) {
	fmt.Println("🔍 Running OVAV Autonomous Research...")
	fmt.Println()

	result, err := eng.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Research cycle complete\n")
	fmt.Printf("   Duration: %dms\n", result.DurationMs)
	fmt.Printf("   URLs scraped: %d\n", len(result.URLsScraped))
	fmt.Printf("   Findings: %d\n", len(result.Findings))
	fmt.Printf("   Changes: %d\n", len(result.Changes))
	if len(result.Errors) > 0 {
		fmt.Printf("   Errors: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("     - %s\n", e)
		}
	}
}

func status(eng *engine.Engine) {
	st := eng.Status()

	fmt.Println("📊 OVAV Research Status")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Next scheduled:\t%s\n", st.NextScheduled.Format(time.RFC3339))
	fmt.Fprintf(w, "Total findings:\t%d\n", st.TotalFindings)
	fmt.Fprintf(w, "Running:\t%v\n", st.Running)
	fmt.Fprintf(w, "\nTargets:\n")
	fmt.Fprintf(w, "ID\tName\tFrequency\tEnabled\tLast Run\n")
	fmt.Fprintf(w, "---\t----\t---------\t-------\t--------\n")
	for _, t := range st.Targets {
		lastRun := "Never"
		if !t.LastRun.IsZero() {
			lastRun = t.LastRun.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", t.ID, t.Name, t.Frequency, t.Enabled, lastRun)
	}
	w.Flush()
}

func findings(eng *engine.Engine) {
	findings, err := eng.ListFindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Println("No findings yet. Run 'ovav research run' first.")
		return
	}

	fmt.Printf("📋 Research Findings (%d total)\n\n", len(findings))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tSeverity\tCategory\tTitle\n")
	fmt.Fprintf(w, "---\t--------\t--------\t-----\n")
	for _, f := range findings {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", f.ID, f.Severity, f.Category, f.Title)
	}
	w.Flush()
}

func targets(eng *engine.Engine) {
	st := eng.Status()

	fmt.Println("🎯 Research Targets")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tName\tURL\tFrequency\n")
	fmt.Fprintf(w, "---\t----\t---\t---------\n")
	for _, t := range st.Targets {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Name, t.URL, t.Frequency)
	}
	w.Flush()
}

func analyze(eng *engine.Engine) {
	fmt.Println("🧠 Running AI-Powered Analysis...")
	fmt.Println()

	intel := eng.GetIntelligenceLayer()

	findings, err := eng.ListFindings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading findings: %v\n", err)
		os.Exit(1)
	}

	if len(findings) == 0 {
		fmt.Println("No findings to analyze. Run 'ovav research run' first.")
		return
	}

	// Predictive analysis
	predictions, err := intel.PredictiveAnalysis(findings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in predictive analysis: %v\n", err)
		os.Exit(1)
	}

	if len(predictions) > 0 {
		fmt.Println("📈 Predictive Analysis:")
		fmt.Println()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Category\tTrend\tConfidence\tActions\n")
		fmt.Fprintf(w, "--------\t-----\t----------\t-------\n")
		for _, p := range predictions {
			actions := strings.Join(p.Actions, ", ")
			if len(actions) > 50 {
				actions = actions[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%.0f%%\t%s\n", p.Category, p.Direction, p.Confidence*100, actions)
		}
		w.Flush()
		fmt.Println()
	}

	// Correlations
	correlations := intel.CorrelateFindings(findings)
	if len(correlations) > 0 {
		fmt.Printf("🔗 Found %d correlations between findings:\n", len(correlations))
		fmt.Println()
		for i, c := range correlations {
			if i >= 5 {
				break
			}
			fmt.Printf("  • %s ↔ %s (strength: %.0f%%, type: %s)\n",
				c.Finding1, c.Finding2, c.Strength*100, c.Type)
		}
		fmt.Println()
	}

	// Target prioritization
	st := eng.Status()
	var targets []scheduler.Target
	for _, t := range st.Targets {
		targets = append(targets, scheduler.Target{
			ID:        t.ID,
			Name:      t.Name,
			URL:       t.URL,
			Frequency: t.Frequency,
			LastRun:   t.LastRun,
			NextRun:   t.NextRun,
			Enabled:   t.Enabled,
		})
	}

	prioritized := intel.PrioritizeTargets(targets, findings)
	if len(prioritized) > 0 {
		fmt.Println("🎯 Prioritized Research Targets:")
		fmt.Println()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Rank\tTarget\tScore\tPriority\n")
		fmt.Fprintf(w, "----\t------\t-----\t--------\n")
		for _, pt := range prioritized {
			priority := "Normal"
			if pt.Score > 1.5 {
				priority = "High"
			} else if pt.Score > 1.0 {
				priority = "Medium"
			}
			fmt.Fprintf(w, "#%d\t%s\t%.2f\t%s\n", pt.Rank, pt.Target.Name, pt.Score, priority)
		}
		w.Flush()
	}
}

func printUsage() {
	fmt.Print(`OVAV Research - Autonomous Research System

Usage:
  ovav research <command>

Commands:
  run      Execute a research cycle for all due targets
  status   Show research system status and targets
  findings List all collected findings
  targets  List all research targets

Examples:
  ovav research run       # Run full research cycle
  ovav research status    # Check system status
  ovav research findings  # View all findings
` + "\n")
}
