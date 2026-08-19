// Command ovav-adapter-warp bridges OWS with Warp presentation.
//
// Per plan §19:
//   - OWS owns git worktree lifecycle (create, merge, prune, move).
//   - This binary is the PRESENTATION layer — opens Warp tabs/Code Review
//     for existing worktrees. It NEVER touches git state.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ovav/ovav/internal/adapter/warp"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "open-tab":
		fs := flag.NewFlagSet("open-tab", flag.ExitOnError)
		config := fs.String("config", "", "Tab Config name (required)")
		if err := fs.Parse(args); err != nil {
			os.Exit(2)
		}
		if *config == "" {
			fmt.Fprintln(os.Stderr, "open-tab: --config is required")
			usage()
			os.Exit(2)
		}
		adapter := warp.New()
		if err := adapter.OpenTabConfig(context.Background(), *config); err != nil {
			fmt.Fprintf(os.Stderr, "open-tab failed: %v\n", err)
			os.Exit(1)
		}

	case "open-worktree":
		fs := flag.NewFlagSet("open-worktree", flag.ExitOnError)
		path := fs.String("path", "", "Absolute worktree path (required)")
		config := fs.String("config", "", "Tab Config name (required)")
		if err := fs.Parse(args); err != nil {
			os.Exit(2)
		}
		if *path == "" || *config == "" {
			fmt.Fprintln(os.Stderr, "open-worktree: --path and --config are required")
			usage()
			os.Exit(2)
		}
		adapter := warp.New()
		if err := adapter.OpenWorktree(context.Background(), *path, *config); err != nil {
			fmt.Fprintf(os.Stderr, "open-worktree failed: %v\n", err)
			os.Exit(1)
		}

	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `ovav-adapter-warp — OWS → Warp presentation bridge

Usage:
  ovav-adapter-warp open-tab --config <name>
  ovav-adapter-warp open-worktree --path <abs> --config <name>

This binary is READ-ONLY on git state. OWS owns worktree lifecycle.`)
}
