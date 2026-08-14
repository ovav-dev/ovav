package main

import "fmt"

// cmdDrift is a placeholder for the upcoming drift-show command (D4).
// Per ADR-005 Phase 1: visual diff between fragment (source-of-truth) and
// live state. Full implementation lands in feat-drift-show worktree.
//
// Usage: ovav drift [show] [--json]
func cmdDrift(args []string) int {
	fmt.Println("⚠️  ovav drift: command not yet implemented")
	fmt.Println("   This is a placeholder. Full implementation lands in")
	fmt.Println("   feat-drift-show worktree (Phase 1 / D4 / ADR-005).")
	fmt.Println("")
	fmt.Println("Planned subcommands:")
	fmt.Println("  show [target]   — visual diff fragment vs live")
	fmt.Println("  show --json     — JSON output for CI")
	fmt.Println("  fix             — auto-apply suggested fixes (Phase 2)")
	return 1
}