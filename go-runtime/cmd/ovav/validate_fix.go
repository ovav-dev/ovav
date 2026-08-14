package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/validators"
)

// cmdValidateFix is the `ovav validate --fix` command.
// Per ADR-011: auto-remediation for safe-fix validators.
//
// Usage:
//   ovav validate --fix [--dry-run] [--strategy=atomic|best-effort] [--ceo-waiver]
//
// Subcommands (when --fix not present):
//   ovav validate --fix --list     # List safe-fix validators
func cmdValidateFix(args []string) int {
	root, err := cliFindRepoRootSafe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV validate: %v\n", err)
		return 1
	}

	// Parse flags
	dryRun := false
	ceoWaiver := false
	listOnly := false
	strategy := "best-effort"

	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--ceo-waiver":
			ceoWaiver = true
		case "--list":
			listOnly = true
		case "--strategy=atomic":
			strategy = "atomic"
		case "--strategy=best-effort":
			strategy = "best-effort"
		case "--help", "-h":
			printValidateFixHelp()
			return 0
		}
	}

	if listOnly {
		return listSafeFix(root)
	}

	orchestrator := validators.NewAutoFixOrchestrator(root)
	if dryRun {
		orchestrator.WithDryRun()
	}
	if ceoWaiver {
		orchestrator.WithCEOWaiver()
	}

	results, err := orchestrator.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "OVAV validate --fix: %v\n", err)
		return 1
	}

	// Output summary
	fmt.Printf("OVAV validate --fix (strategy: %s)\n\n", strategy)
	applied, skipped, failed := 0, 0, 0
	for _, r := range results {
		icon := "⏭️ "
		switch r.Outcome {
		case "applied":
			applied++
			icon = "✅"
		case "skipped":
			skipped++
			icon = "⏭️ "
		case "dry-run":
			icon = "🔍"
		case "failed":
			failed++
			icon = "❌"
		case "rollback":
			failed++
			icon = "↩️ "
		}
		fmt.Printf("%s %s: %s (%d ms)\n", icon, r.ValidatorID, r.Outcome, r.DurationMs)
		if r.Error != "" {
			fmt.Printf("   error: %s\n", r.Error)
		}
	}
	fmt.Printf("\nSummary: %d applied, %d skipped, %d failed\n", applied, skipped, failed)

	if failed > 0 {
		return 1
	}
	return 0
}

// listSafeFix prints the safe-fix registry.
func listSafeFix(root string) int {
	entries := validators.GetSafeFixRegistry()
	fmt.Printf("Safe-fix validators (%d):\n\n", len(entries))
	for _, e := range entries {
		fmt.Printf("• %s [%s]\n", e.ValidatorID, e.RiskLevel)
		fmt.Printf("  %s\n", e.Description)
		fmt.Println("")
	}
	return 0
}

func printValidateFixHelp() {
	fmt.Println(`OVAV validate --fix — auto-remediation for safe-fix validators

Per ADR-011:
  Snapshot → Apply → Verify → Rollback-on-regression

Usage:
  ovav validate --fix                       # Apply all safe-fixes
  ovav validate --fix --dry-run             # Show what would change
  ovav validate --fix --list                # List safe-fix validators
  ovav validate --fix --strategy=atomic     # All-or-nothing rollback
  ovav validate --fix --ceo-waiver          # Allow fixing protected files

Safe-fix validators (3):
  bash_readline_bindings         Add 'deliberately UNBOUND' marker to ~/.inputrc
  runtime_integrity_baseline_fresh   Regenerate baseline.json
  supply_chain                   Regenerate sbom.json

Exit codes:
  0 = all fixes applied successfully
  1 = at least one fix failed or rolled back

Logs: .ovav/registry/auto_fix_history.jsonl
Snapshots: .ovav/registry/snapshots/fix-*/`)
}