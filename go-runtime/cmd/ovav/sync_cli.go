package main

import (
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/project"
)

// cmdSync handles `ovav sync` — projection sync from canonical source to CLI surfaces.
func cmdSync(args []string) int {
	verbose := false
	dryRun := false
	step := ""

	for _, arg := range args {
		switch arg {
		case "-v", "--verbose":
			verbose = true
		case "--dry-run":
			dryRun = true
		case "--agents":
			step = "agents"
		case "--skills":
			step = "skills"
		case "--visual":
			step = "visual"
		case "--mimocode":
			step = "mimocode"
		case "--help", "-h":
			printSyncHelp()
			return 0
		}
	}

	root, err := findOvavRoot()
	if err != nil || root == "" {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root directory\n")
		return 1
	}

	if dryRun {
		fmt.Println("╔══════════════════════════════════════════════════════╗")
		fmt.Println("║  OVAV Sync — Dry Run                                ║")
		fmt.Println("╠══════════════════════════════════════════════════════╣")
		fmt.Printf("║  Root: %s\n", root)
		fmt.Printf("║  Step: %s\n", step)
		fmt.Println("╚══════════════════════════════════════════════════════╝")
		return 0
	}

	fmt.Println("🔄 OVAV Sync Projection")
	fmt.Println()

	totalFailed := 0

	// Run specific step or all
	if step == "" || step == "agents" {
		if cleaned, created, err := project.SyncAgents(root, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ agents: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ agents: %d cleaned, %d projected\n", cleaned, created)
		}
	}

	if step == "" || step == "skills" {
		if s, a, err := project.SyncConnectorBus(root, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ skills: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ skills: %d synced, %d agents\n", s, a)
		}
	}

	if step == "" || step == "visual" {
		if v, err := project.SyncVisual(root, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ visual: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ visual: %d artifacts\n", v)
		}
	}

	if step == "" || step == "mimocode" {
		if mc, err := project.SyncMiMoCode(root, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ mimocode: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ mimocode: %d artifacts\n", mc)
		}
	}

	fmt.Println()
	if totalFailed > 0 {
		fmt.Printf("❌ Sync completed with %d errors\n", totalFailed)
		return 1
	}
	fmt.Println("✅ Sync projection complete")
	return 0
}

func printSyncHelp() {
	fmt.Println("ovav sync — Project canonical sources to CLI surfaces")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ovav sync              Run full sync (all steps)")
	fmt.Println("  ovav sync --agents     Sync agents only")
	fmt.Println("  ovav sync --skills     Sync skills + personnel only")
	fmt.Println("  ovav sync --visual     Sync themes + plugins only")
	fmt.Println("  ovav sync --mimocode   Sync MiMo Code artifacts only")
	fmt.Println("  ovav sync --dry-run    Preview without writing")
	fmt.Println("  ovav sync -v           Verbose output")
	fmt.Println()
	fmt.Println("Also available in cockpit: ovav → Sync Projection")
}
