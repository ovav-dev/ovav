package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cmdDrift dispatches the ovav drift subcommands.
//
// Usage: ovav drift <subcommand>
// Subcommands:
//
//	show [target]      — visual diff fragment vs live (D4)
//	show --json        — JSON output for CI
//	show --md          — markdown output for PR comments
//	catalog            — show drift history (drift_catalog.jsonl)
//	targets            — list registered drift targets
func cmdDrift(args []string) int {
	if len(args) == 0 {
		return runDriftShow(args)
	}
	switch args[0] {
	case "show":
		return runDriftShow(args[1:])
	case "catalog":
		return runDriftCatalog(args[1:])
	case "targets":
		return runDriftTargets(args[1:])
	case "help", "--help", "-h":
		printDriftHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV drift: unknown subcommand %q\n", args[0])
		printDriftHelp()
		return 2
	}
}

func printDriftHelp() {
	fmt.Println(`OVAV drift — visibility into fragment vs live state

Usage:
  ovav drift show [target]      # visual diff (default)
  ovav drift show --json        # JSON output for CI
  ovav drift show --md          # markdown for PR comments
  ovav drift catalog            # show drift history
  ovav drift targets            # list registered targets

Targets:
  it-keybindings    IT settings.json (Intelligent Terminal v0.1.4+)
  bash-inputrc      ~/.inputrc (bash readline)
  runtime-baseline  .ovav/integrity_backups/baseline.json
  pinned-baseline   .ovav/integrity_backups/baseline.pinned.json
  tool-configs      .ovav/registry/tool_configs.yaml vs bin/ovav

Each drift item includes a suggested_fix command.`)
}

// runDriftShow produces the drift report.
func runDriftShow(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV drift: %v\n", err)
		return 1
	}

	// Parse flags
	jsonOut := false
	mdOut := false
	targetFilter := ""
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--md":
			mdOut = true
		case "--help", "-h":
			printDriftHelp()
			return 0
		default:
			if !strings.HasPrefix(a, "-") {
				targetFilter = a
			}
		}
	}

	report, err := buildDriftReport(root, targetFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV drift: %v\n", err)
		return 1
	}

	if jsonOut {
		return outputDriftJSON(report)
	}
	if mdOut {
		return outputDriftMarkdown(report)
	}
	outputDriftHuman(report)

	// Append to catalog
	catalogPath := filepath.Join(root, ".ovav", "registry", "drift_catalog.jsonl")
	entry := DriftCatalogEntry{
		Timestamp:      report.Timestamp,
		RepoRoot:       report.RepoRoot,
		TotalTargets:   report.TotalTargets,
		DriftedTargets: report.DriftedTargets,
		TotalItems:     report.TotalItems,
	}
	appendCatalog(catalogPath, entry)

	// Exit code: 0 if no drift, 1 if drift found (CI-friendly)
	if report.DriftedTargets > 0 {
		return 1
	}
	return 0
}

// buildDriftReport runs all (filtered) targets and returns a DriftReport.
func buildDriftReport(root, filter string) (DriftReport, error) {
	targets := DefaultTargets(root)
	report := DriftReport{
		Timestamp: nowISO(),
		RepoRoot:  root,
		Targets:   []DriftTargetReport{},
	}

	for _, t := range targets {
		if filter != "" && t.ID != filter {
			continue
		}
		targetReport := runTarget(t, root)
		report.Targets = append(report.Targets, targetReport)
		report.TotalTargets++
		if len(targetReport.Items) > 0 {
			report.DriftedTargets++
			report.TotalItems += len(targetReport.Items)
		}
	}
	return report, nil
}

// runTarget runs a single target's Compare function.
func runTarget(t DriftTarget, root string) DriftTargetReport {
	report := DriftTargetReport{Target: t}

	fragPath := filepath.Join(root, t.FragmentRel)
	fragData, err := os.ReadFile(fragPath)
	if err != nil {
		report.FragmentOK = false
		// Cannot compare if fragment missing — return early
		return report
	}
	report.FragmentOK = true

	livePath := t.resolveLivePath()
	if livePath == "" {
		// Live path not configured (e.g., runtime-baseline is dynamic)
		// Skip this target — handled elsewhere
		return report
	}

	liveData, err := os.ReadFile(livePath)
	if err != nil {
		report.LiveOK = false
		// Cannot compare if live missing
		return report
	}
	report.LiveOK = true

	items, err := t.Compare(fragData, liveData)
	if err != nil {
		// Add as synthetic drift item so user sees the error
		items = []DriftItem{{
			Type:         DriftModified,
			Path:         "(compare error)",
			FragmentJSON: err.Error(),
		}}
	}
	report.Items = items
	return report
}

// outputDriftJSON writes the report as JSON.
func outputDriftJSON(report DriftReport) int {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV drift: marshal: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	// CI-friendly: exit non-zero if drift detected
	if report.DriftedTargets > 0 {
		return 1
	}
	return 0
}

// outputDriftMarkdown writes the report as markdown.
func outputDriftMarkdown(report DriftReport) int {
	fmt.Printf("# OVAV Drift Report — %s\n\n", report.Timestamp)
	fmt.Printf("**Targets**: %d total, %d with drift, %d items\n\n",
		report.TotalTargets, report.DriftedTargets, report.TotalItems)
	for _, t := range report.Targets {
		fmt.Printf("## %s\n\n", t.Target.Name)
		if !t.FragmentOK {
			fmt.Println("**Fragment missing** — fragment file not found at", t.Target.FragmentRel)
			fmt.Println("")
			continue
		}
		if !t.LiveOK {
			fmt.Println("**Live missing** — live file not found at", t.Target.resolveLivePath())
			fmt.Println("")
			continue
		}
		if len(t.Items) == 0 {
			fmt.Println("✅ No drift")
			fmt.Println("")
			continue
		}
		fmt.Printf("**%d drift items**\n\n", len(t.Items))
		fmt.Println("| Type | Path | Fix |")
		fmt.Println("|------|------|-----|")
		for _, item := range t.Items {
			fmt.Printf("| %s | `%s` | `%s` |\n", item.Type, item.Path, item.SuggestedFix)
		}
		fmt.Println("")
	}
	return 0
}

// outputDriftHuman writes the report as a human-readable table.
func outputDriftHuman(report DriftReport) int {
	fmt.Printf("OVAV Drift Report — %s\n", report.Timestamp)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Targets: %d total, %d with drift, %d items\n\n",
		report.TotalTargets, report.DriftedTargets, report.TotalItems)

	for _, t := range report.Targets {
		fmt.Printf("📦 %s [%s]\n", t.Target.Name, t.Target.ID)
		fmt.Printf("   Fragment: %s\n", t.Target.FragmentRel)
		livePath := t.Target.resolveLivePath()
		if livePath != "" {
			fmt.Printf("   Live:     %s\n", livePath)
		} else {
			fmt.Printf("   Live:     (dynamic — see validator)\n")
		}

		if !t.FragmentOK {
			fmt.Printf("   ❌ Fragment missing\n\n")
			continue
		}
		if !t.LiveOK {
			fmt.Printf("   ⚠️  Live missing (no drift check possible)\n\n")
			continue
		}
		if len(t.Items) == 0 {
			fmt.Printf("   ✅ No drift\n\n")
			continue
		}

		// Count by type
		byType := map[DriftType]int{}
		for _, item := range t.Items {
			byType[item.Type]++
		}
		fmt.Printf("   Drift: %d items", len(t.Items))
		if len(byType) > 0 {
			fmt.Printf(" (")
			first := true
			for dt, count := range byType {
				if !first {
					fmt.Print(", ")
				}
				fmt.Printf("%d %s", count, dt)
				first = false
			}
			fmt.Printf(")")
		}
		fmt.Println("")

		// Show first 5 items
		showCount := len(t.Items)
		if showCount > 5 {
			showCount = 5
		}
		for i := 0; i < showCount; i++ {
			item := t.Items[i]
			fmt.Printf("   - %s: %s\n", item.Type, item.Path)
			if item.FragmentJSON != "" {
				fmt.Printf("     fragment: %s\n", truncate(item.FragmentJSON, 60))
			}
			if item.LiveJSON != "" {
				fmt.Printf("     live:     %s\n", truncate(item.LiveJSON, 60))
			}
			if item.SuggestedFix != "" {
				fmt.Printf("     fix:      %s\n", item.SuggestedFix)
			}
		}
		if len(t.Items) > 5 {
			fmt.Printf("   ... and %d more (use --json for full list)\n", len(t.Items)-5)
		}
		fmt.Println("")
	}

	return 0
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func nowISO() string {
	// ISO 8601 with timezone
	return fmt.Sprintf("%s", nowRFC3339())
}

// nowRFC3339 returns current time as RFC3339.
// Wrapper to avoid importing "time" in tests that don't need it.
func nowRFC3339() string {
	// Simple implementation: use date command or fallback
	out, err := osExecute("date", "-u", "+%Y-%m-%dT%H:%M:%SZ")
	if err != nil {
		return "1970-01-01T00:00:00Z"
	}
	return strings.TrimSpace(string(out))
}

// osExecute runs a command and returns stdout.
func osExecute(name string, args ...string) ([]byte, error) {
	// Use Go's exec via package import (avoid circular issues)
	return goExec(name, args...)
}

// appendCatalog appends a JSONL entry to the catalog.
func appendCatalog(path string, entry DriftCatalogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	// Ensure directory exists
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	// Append
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(data)
	f.Write([]byte("\n"))
}

// runDriftCatalog shows the drift history.
func runDriftCatalog(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV drift catalog: %v\n", err)
		return 1
	}
	path := filepath.Join(root, ".ovav", "registry", "drift_catalog.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("No catalog yet at %s\n", path)
		return 0
	}
	fmt.Printf("Drift catalog (%s):\n\n", path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry DriftCatalogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			fmt.Printf("  (parse error: %v)\n", err)
			continue
		}
		fmt.Printf("  %s — %d targets, %d drifted, %d items\n",
			entry.Timestamp, entry.TotalTargets, entry.DriftedTargets, entry.TotalItems)
	}
	return 0
}

// runDriftTargets lists registered drift targets.
func runDriftTargets(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV drift targets: %v\n", err)
		return 1
	}
	targets := DefaultTargets(root)
	fmt.Printf("Registered drift targets (%d):\n\n", len(targets))
	for _, t := range targets {
		fixIcon := "✅"
		if !t.AutoFixable {
			fixIcon = "⚠️"
		}
		fmt.Printf("%s %s — %s\n", fixIcon, t.ID, t.Name)
		fmt.Printf("   fragment: %s\n", t.FragmentRel)
		livePath := t.resolveLivePath()
		if livePath != "" {
			fmt.Printf("   live:     %s\n", livePath)
		}
		if t.LiveEnv != "" {
			fmt.Printf("   env:      %s\n", t.LiveEnv)
		}
		fmt.Println("")
	}
	return 0
}
