package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ovav/ovav/internal/convert"
	"github.com/ovav/ovav/internal/hostsync"
	"github.com/ovav/ovav/internal/project"
)

type syncOptions struct {
	verbose          bool
	dryRun           bool
	planJSON         bool
	help             bool
	step             string
	hostProfile      string
	rollbackJournal  string
	windowsHome      string
	apply            bool
	approveHostWrite bool
}

func parseSyncArgs(args []string) (syncOptions, error) {
	var opts syncOptions
	for index := 0; index < len(args); index++ {
		arg := args[index]
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
		case "--host-profile":
			if opts.hostProfile != "" {
				return syncOptions{}, errors.New("--host-profile may be specified only once")
			}
			index++
			if index >= len(args) || args[index] == "" {
				return syncOptions{}, errors.New("--host-profile requires a value")
			}
			opts.hostProfile = args[index]
		case "--rollback-journal":
			if opts.rollbackJournal != "" {
				return syncOptions{}, errors.New("--rollback-journal may be specified only once")
			}
			index++
			if index >= len(args) || args[index] == "" {
				return syncOptions{}, errors.New("--rollback-journal requires a value")
			}
			opts.rollbackJournal = args[index]
		case "--windows-home":
			if opts.windowsHome != "" {
				return syncOptions{}, errors.New("--windows-home may be specified only once")
			}
			index++
			if index >= len(args) || args[index] == "" {
				return syncOptions{}, errors.New("--windows-home requires a value")
			}
			opts.windowsHome = args[index]
		case "--apply":
			opts.apply = true
		case "--approve-host-write":
			opts.approveHostWrite = true
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
	if opts.help {
		return opts, nil
	}
	if err := validateHostSyncOptions(opts); err != nil {
		return syncOptions{}, err
	}
	return opts, nil
}

func validateHostSyncOptions(opts syncOptions) error {
	hostMode := opts.hostProfile != "" || opts.rollbackJournal != "" || opts.windowsHome != "" || opts.apply || opts.approveHostWrite
	if !hostMode {
		return nil
	}
	if opts.step != "" || opts.dryRun || opts.planJSON || opts.verbose {
		return errors.New("host projection flags conflict with repository sync flags")
	}
	if opts.hostProfile != "" && opts.rollbackJournal != "" {
		return errors.New("--host-profile and --rollback-journal are mutually exclusive")
	}
	if opts.hostProfile != "" {
		windows, ok := hostsync.RequiresWindowsHome(opts.hostProfile)
		if !ok {
			return fmt.Errorf("unknown host profile %q", opts.hostProfile)
		}
		if windows && opts.windowsHome == "" {
			return fmt.Errorf("host profile %q requires --windows-home", opts.hostProfile)
		}
		if !windows && opts.windowsHome != "" {
			return fmt.Errorf("--windows-home is not valid for host profile %q", opts.hostProfile)
		}
	}
	if opts.apply && opts.hostProfile == "" {
		return errors.New("--apply requires --host-profile")
	}
	if opts.apply && !opts.approveHostWrite {
		return errors.New("--apply requires --approve-host-write")
	}
	if opts.rollbackJournal != "" && !opts.approveHostWrite {
		return errors.New("--rollback-journal requires --approve-host-write")
	}
	if opts.approveHostWrite && !opts.apply && opts.rollbackJournal == "" {
		return errors.New("--approve-host-write requires --apply or --rollback-journal")
	}
	if opts.hostProfile == "" && opts.rollbackJournal == "" {
		return errors.New("host projection flags require --host-profile or --rollback-journal")
	}
	if opts.rollbackJournal != "" && (!filepath.IsAbs(opts.rollbackJournal) || filepath.Clean(opts.rollbackJournal) != opts.rollbackJournal) {
		return errors.New("--rollback-journal must be an absolute, traversal-free path")
	}
	if opts.windowsHome != "" && (!filepath.IsAbs(opts.windowsHome) || filepath.Clean(opts.windowsHome) != opts.windowsHome) {
		return errors.New("--windows-home must be an absolute, traversal-free path")
	}
	return nil
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
	if opts.hostProfile != "" || opts.rollbackJournal != "" {
		return cmdHostSync(root, opts)
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

func cmdHostSync(root string, opts syncOptions) int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: resolve home: %v\n", err)
		return 1
	}
	result, err := hostsync.Run(hostsync.Request{
		RepoRoot: root, Home: home, WindowsHome: opts.windowsHome, Profile: opts.hostProfile,
		Apply: opts.apply, ApproveHostWrite: opts.approveHostWrite,
		RollbackJournal: opts.rollbackJournal, Now: time.Now(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return jsonOutput(result)
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
	fmt.Println("  ovav sync --host-profile <name>")
	fmt.Println("                          Plan an exact host projection; emits JSON")
	fmt.Println("  ovav sync --host-profile <name> --apply --approve-host-write")
	fmt.Println("                          Apply an approved host projection")
	fmt.Println("  ovav sync --rollback-journal <absolute> --approve-host-write")
	fmt.Println("                          Roll back an approved projection journal")
	fmt.Println("  --windows-home <absolute>")
	fmt.Println("                          Required only for wsl2-resource-policy and warp-wsl-tab")
	fmt.Println("  Host profiles: opencode-bootstrap, wsl2-resource-policy, warp-wsl-tab")
	fmt.Println("  ovav sync -v           Verbose output")
	fmt.Println()
	fmt.Println("Also available in cockpit: ovav → Sync Projection")
}
