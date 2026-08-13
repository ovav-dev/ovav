package main

import (
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/convert"
	"github.com/ovav/ovav/internal/project"
)

type syncOptions struct {
	verbose  bool
	dryRun   bool
	planJSON bool
	help     bool
	step     string
}

func parseSyncArgs(args []string) (syncOptions, error) {
	var opts syncOptions
	for _, arg := range args {
		var step string
		switch arg {
		case "-v", "--verbose":
			opts.verbose = true
		case "--dry-run":
			opts.dryRun = true
		case "--plan-json":
			opts.planJSON = true
		case "--agents":
			step = "agents"
		case "--skills":
			step = "skills"
		case "--visual":
			step = "visual"
		case "--mimocode":
			step = "mimocode"
		case "--help", "-h":
			opts.help = true
		default:
			return syncOptions{}, fmt.Errorf("unknown sync argument %q", arg)
		}
		if step != "" {
			if opts.step != "" && opts.step != step {
				return syncOptions{}, fmt.Errorf("sync steps %q and %q conflict", opts.step, step)
			}
			opts.step = step
		}
	}
	return opts, nil
}

func buildSyncPlan(root, step string) map[string]interface{} {
	steps := []string{"agents", "skills", "visual", "mimocode", "harness_agents", "config"}
	switch step {
	case "agents":
		steps = []string{"agents", "harness_agents"}
	case "skills", "visual", "mimocode":
		steps = []string{step}
	}
	return map[string]interface{}{
		"schema_version":   "ovav.sync_plan.v1",
		"command":          "sync",
		"mode":             "plan",
		"root":             root,
		"steps":            steps,
		"writes_performed": false,
	}
}

// cmdSync handles `ovav sync` — projection sync from canonical source to CLI surfaces.
func cmdSync(args []string) int {
	opts, err := parseSyncArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	if opts.help {
		printSyncHelp()
		return 0
	}

	root, err := findOvavRoot()
	if err != nil || root == "" {
		fmt.Fprintf(os.Stderr, "❌ Cannot find OVAV root directory\n")
		return 1
	}

	if opts.planJSON {
		return jsonOutput(buildSyncPlan(root, opts.step))
	}

	if opts.dryRun {
		fmt.Println("╔══════════════════════════════════════════════════════╗")
		fmt.Println("║  OVAV Sync — Dry Run                                ║")
		fmt.Println("╠══════════════════════════════════════════════════════╣")
		fmt.Printf("║  Root: %s\n", root)
		fmt.Printf("║  Step: %s\n", opts.step)
		fmt.Println("╚══════════════════════════════════════════════════════╝")
		return 0
	}

	fmt.Println("🔄 OVAV Sync Projection")
	fmt.Println()

	totalFailed := 0

	// Run specific step or all
	if opts.step == "" || opts.step == "agents" {
		if cleaned, created, err := project.SyncAgents(root, opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ agents: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ agents: %d cleaned, %d projected\n", cleaned, created)
		}
	}

	if opts.step == "" || opts.step == "skills" {
		if s, a, err := project.SyncConnectorBus(root, opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ skills: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ skills: %d synced, %d agents\n", s, a)
		}
	}

	if opts.step == "" || opts.step == "visual" {
		if v, err := project.SyncVisual(root, opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ visual: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ visual: %d artifacts\n", v)
		}
	}

	if opts.step == "" || opts.step == "mimocode" {
		if mc, err := project.SyncMiMoCode(root, opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ mimocode: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ mimocode: %d artifacts\n", mc)
		}
	}

	if opts.step == "" || opts.step == "agents" {
		if ha, err := project.SyncHarnessAgents(root, opts.verbose); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ harness_agents: %v\n", err)
			totalFailed++
		} else {
			fmt.Printf("  ✓ harness_agents: %d harness AGENTS projected\n", ha)
		}
	}

	if opts.step == "" {
		// OpenCode config generation
		if err := convert.GenerateOpenCodeConfig(root); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ config: %v\n", err)
			totalFailed++
		} else {
			fmt.Println("  ✓ config: generated from .ovav/source/opencode/config.yaml")
		}
		// Validate generated config
		issues, configErr := convert.ValidateOpenCodeConfig(root)
		if configErr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ config validation: %v\n", configErr)
		} else {
			criticals := 0
			for _, issue := range issues {
				if issue.Severity == "critical" {
					criticals++
					fmt.Fprintf(os.Stderr, "  ✗ config: %s — %s\n", issue.Field, issue.Message)
				}
			}
			if criticals == 0 {
				fmt.Println("  ✓ config: valid")
			}
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
	fmt.Println("  ovav sync --plan-json  Emit a machine-readable no-write plan")
	fmt.Println("  ovav sync -v           Verbose output")
	fmt.Println()
	fmt.Println("Also available in cockpit: ovav → Sync Projection")
}
