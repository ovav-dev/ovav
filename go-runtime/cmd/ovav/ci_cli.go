package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// cmdCI dispatches ovav ci subcommands for CI runners.
//
// Usage: ovav ci <subcommand>
// Subcommands:
//   drift-check    — run drift check, exit non-zero if drift found
//   drift-check --json — JSON output
func cmdCI(args []string) int {
	if len(args) == 0 {
		printCIHelp()
		return 0
	}
	switch args[0] {
	case "drift-check":
		return runCIDriftCheck(args[1:])
	case "help", "--help", "-h":
		printCIHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "OVAV ci: unknown subcommand %q\n", args[0])
		printCIHelp()
		return 2
	}
}

func printCIHelp() {
	fmt.Println(`OVAV ci — CI runner commands

Usage:
  ovav ci drift-check            # exit 0 if no drift, 1 if drift
  ovav ci drift-check --json     # JSON output

Use in CI workflows to gate pushes on drift-free state.
See ADR-009 for details.`)
}

// runCIDriftCheck runs the drift check and exits appropriately.
func runCIDriftCheck(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV ci drift-check: %v\n", err)
		return 2
	}

	jsonOut := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		}
	}

	report, err := buildDriftReport(root, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV ci drift-check: %v\n", err)
		return 2
	}

	if jsonOut {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
	} else {
		// Human summary
		fmt.Printf("OVAV CI Drift Check\n")
		fmt.Printf("  Total targets:   %d\n", report.TotalTargets)
		fmt.Printf("  Drifted targets: %d\n", report.DriftedTargets)
		fmt.Printf("  Drift items:     %d\n", report.TotalItems)
		if report.DriftedTargets > 0 {
			fmt.Println("  Status: ❌ DRIFT DETECTED")
		} else {
			fmt.Println("  Status: ✅ CLEAN")
		}
	}

	// Exit codes: 0 = clean, 1 = drift, 2 = error
	if report.DriftedTargets > 0 {
		return 1
	}
	return 0
}