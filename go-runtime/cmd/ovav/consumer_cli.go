package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ovav/ovav/internal/consumers"
	"github.com/ovav/ovav/internal/validators"
)

// cmdConsumer manages central-only state for independent consumer projects.
// It never writes to the consumer repository.
func cmdConsumer(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println(`OVAV consumer — independent repository governance

Usage:
  ovav consumer status
  ovav consumer baseline --plan
  ovav consumer baseline --write

Baselines are stored only in OVAV's central consumer registry state.`)
		return 0
	}

	root, err := findConsumerRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ consumer: %v\n", err)
		return 1
	}
	profile := consumers.Resolve(root)
	if !profile.External {
		fmt.Fprintln(os.Stderr, "❌ consumer: this command is only for independent repositories")
		return 1
	}
	if !profile.Active() {
		fmt.Fprintf(os.Stderr, "❌ consumer: %v\n", profile.Err)
		return 1
	}

	switch args[0] {
	case "status":
		fmt.Printf("✅ consumer %s registered\n", profile.Consumer.ID)
		fmt.Printf("   registry: %s\n", profile.RegistryPath)
		fmt.Printf("   central state: %s\n", profile.StateDir())
		return 0
	case "baseline":
		return cmdConsumerBaseline(root, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "❌ consumer: unknown subcommand %q\n", args[0])
		return 2
	}
}

func cmdConsumerBaseline(root string, args []string) int {
	plan := false
	write := false
	for _, arg := range args {
		switch arg {
		case "--plan":
			plan = true
		case "--write":
			write = true
		default:
			fmt.Fprintf(os.Stderr, "❌ consumer baseline: unknown option %s\n", arg)
			return 2
		}
	}
	if plan == write {
		fmt.Fprintln(os.Stderr, "❌ consumer baseline: choose exactly one of --plan or --write")
		return 2
	}
	if write {
		if dirty, err := consumerWorktreeDirty(root); err != nil {
			fmt.Fprintf(os.Stderr, "❌ consumer baseline: git status failed: %v\n", err)
			return 1
		} else if dirty {
			fmt.Fprintln(os.Stderr, "❌ consumer baseline: worktree must be clean; baseline is anchored to committed HEAD")
			return 1
		}
	}
	sbom, integrity, err := validators.PlanExternalBaselines(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ consumer baseline: %v\n", err)
		return 1
	}
	if plan {
		fmt.Printf("✅ no-write plan: SBOM %d bytes, integrity baseline %d bytes\n", len(sbom), len(integrity))
		return 0
	}
	if err := validators.WriteExternalBaselines(root); err != nil {
		fmt.Fprintf(os.Stderr, "❌ consumer baseline: %v\n", err)
		return 1
	}
	profile := consumers.Resolve(root)
	fmt.Printf("✅ central baselines written for %s\n", profile.Consumer.ID)
	fmt.Printf("   %s\n   %s\n", profile.StateDir()+"/sbom.json", profile.StateDir()+"/integrity.json")
	return 0
}

func findConsumerRepoRoot() (string, error) {
	root, err := findRepoRootForCommand()
	if err != nil {
		return "", err
	}
	return root, nil
}

func findRepoRootForCommand() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func consumerWorktreeDirty(root string) (bool, error) {
	cmd := exec.CommandContext(context.Background(), "git", "status", "--porcelain", "--untracked-files=all")
	cmd.Dir = root
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)) != "", err
}
