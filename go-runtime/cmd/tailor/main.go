// OVAV Tailor Composer — CLI.
//
// Port of tools/cli/ovav_tailor_composer.py (Python).
// Provides workstation profile composition: plans, tools, roles with plan-gating.
// Can be used standalone or driven by the Cockpit TUI.
//
// Stack: Go stdlib-only. No external dependencies.
//
// Build: go build -o bin/tailor ./cmd/tailor/
// Run:   ./bin/tailor

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ovav/ovav/internal/tailor"
)

func main() {
	s := tailor.NewState(nil)

	if len(os.Args) < 2 {
		interactive(s)
		return
	}

	exitCode, err := dispatch(s, os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// ── Dispatch (non-interactive command processing) ───────────────────

// dispatch processes a single non-interactive command.
// args should be os.Args[1:] (command name + arguments).
// Returns exit code (0 = success) and any error.
func dispatch(s *tailor.State, args []string, w io.Writer) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("no command provided")
	}

	switch args[0] {
	case "status":
		printStatus(w, s)
		return 0, nil

	case "select", "plan":
		if len(args) < 2 {
			fmt.Fprintln(w, "Usage: ovav tailor select <plan>")
			return 1, nil
		}
		if err := s.SelectPlan(args[1]); err != nil {
			return 1, err
		}
		printStatus(w, s)
		return 0, nil

	case "toggle":
		if len(args) < 2 {
			fmt.Fprintln(w, "Usage: ovav tailor toggle <item>")
			return 1, nil
		}
		toggleByID(s, args[1], w)
		printStatus(w, s)
		return 0, nil

	case "preview":
		printPreview(w, s)
		return 0, nil

	case "apply":
		results := s.ApplySelection()
		printResults(w, results)
		return 0, nil

	case "help", "-h", "--help":
		printHelp(w)
		return 0, nil

	default:
		fmt.Fprintf(w, "Unknown command: %s\n", args[0])
		printHelp(w)
		return 1, nil
	}
}

// ── Interactive mode ────────────────────────────────────────────────

func interactive(s *tailor.State) {
	fmt.Println("OVAV Tailor Composer")
	fmt.Println(strings.Repeat("─", 40))
	printStatus(os.Stdout, s)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		cont, err := handleLine(s, line, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		if !cont {
			break
		}
	}
}

// handleLine processes a single interactive input line.
// Returns true to continue the loop, false to quit.
func handleLine(s *tailor.State, line string, w io.Writer) (bool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return true, nil
	}

	parts := strings.Fields(line)
	switch parts[0] {
	case "status", "s":
		printStatus(w, s)

	case "plan", "select":
		if len(parts) < 2 {
			fmt.Fprintln(w, "Usage: plan <nucleo|studio|command>")
			return true, nil
		}
		if err := s.SelectPlan(parts[1]); err != nil {
			fmt.Fprintf(w, "Error: %v\n", err)
			return true, nil
		}
		fmt.Fprintf(w, "✓ Plan %s selected.\n", tailor.PlanLabel(s.SelectedPlan))

	case "toggle", "t":
		if len(parts) < 2 {
			fmt.Fprintln(w, "Usage: toggle <item-id>")
			fmt.Fprintln(w, "Available items:")
			for _, r := range s.SelectableRows() {
				if r.Type == "item" && r.Kind != "plan" {
					mark := " "
					if r.Active {
						mark = "✓"
					}
					fmt.Fprintf(w, "  [%s] %s (%s)\n", mark, r.ID, r.Label)
				}
			}
			return true, nil
		}
		toggleByID(s, parts[1], w)

	case "preview", "p":
		printPreview(w, s)

	case "apply", "a":
		results := s.ApplySelection()
		fmt.Fprintln(w, "\n✓ Configuration applied:")
		printResults(w, results)

	case "help", "h", "?":
		fmt.Fprintln(w, "Commands: status, plan <id>, toggle <id>, preview, apply, help, quit")

	case "quit", "q", "exit":
		fmt.Fprintln(w, "Goodbye.")
		return false, nil

	default:
		fmt.Fprintf(w, "Unknown: %s (type 'help' for commands)\n", parts[0])
	}

	return true, nil
}

// ── Output helpers ──────────────────────────────────────────────────

func printStatus(w io.Writer, s *tailor.State) {
	fmt.Fprintf(w, "\nPlan: %s\n", tailor.PlanLabel(s.SelectedPlan))
	fmt.Fprintln(w, strings.Repeat("─", 40))

	// Tools
	fmt.Fprintln(w, "Tools:")
	for _, t := range s.Tools {
		mark := " "
		if t.Active {
			mark = "✓"
		}
		allowed := s.IsAllowed(t.MinPlan)
		if !allowed {
			fmt.Fprintf(w, "  [%s] %-12s %-20s (requires %s)\n", "✗", t.Label, t.Note, tailor.PlanLabel(t.MinPlan))
		} else {
			dnote := ""
			if t.Detected {
				dnote = " [detected]"
			}
			fmt.Fprintf(w, "  [%s] %-12s %-20s%s\n", mark, t.Label, t.Note, dnote)
		}
	}

	// Roles
	fmt.Fprintln(w, "\nRoles:")
	for _, r := range s.Roles {
		mark := " "
		if r.Active {
			mark = "✓"
		}
		allowed := s.IsAllowed(r.MinPlan)
		if !allowed {
			fmt.Fprintf(w, "  [%s] %-24s %-20s (requires %s)\n", "✗", r.Label, r.Note, tailor.PlanLabel(r.MinPlan))
		} else {
			fmt.Fprintf(w, "  [%s] %-24s %s\n", mark, r.Label, r.Note)
		}
	}

	if s.LastMessage != "" {
		fmt.Fprintf(w, "\n  %s\n", s.LastMessage)
	}
}

func printPreview(w io.Writer, s *tailor.State) {
	changes := s.PreviewChanges()
	if changes == nil || len(changes) == 0 {
		fmt.Fprintln(w, "No pending changes.")
		return
	}
	fmt.Fprintln(w, "\nPending changes:")
	for _, c := range changes {
		mark := "+"
		if !c.After {
			mark = "-"
		}
		fmt.Fprintf(w, "  %s %s: %s\n", mark, c.Label, c.Summary)
	}
}

func printResults(w io.Writer, rows []tailor.ResultRow) {
	for _, r := range rows {
		fmt.Fprintf(w, "  %-14s %s\n", r.Label+":", r.Value)
	}
}

func toggleByID(s *tailor.State, id string, w io.Writer) {
	// Search selectable rows first
	rows := s.SelectableRows()
	for i, r := range rows {
		if r.ID == id && r.Type == "item" {
			fmt.Fprintln(w, s.ToggleAt(i))
			return
		}
	}
	// Try numeric index
	if idx, err := strconv.Atoi(id); err == nil {
		if idx > 0 && idx <= len(rows) {
			fmt.Fprintln(w, s.ToggleAt(idx-1))
			return
		}
	}
	// Item may exist but be hidden by plan gating. Search all items.
	for _, t := range s.Tools {
		if t.ID == id {
			if !s.IsAllowed(t.MinPlan) {
				fmt.Fprintf(w, "%s requires plan %s. Switch plan first.\n", t.Label, tailor.PlanLabel(t.MinPlan))
				return
			}
		}
	}
	for _, r := range s.Roles {
		if r.ID == id {
			if !s.IsAllowed(r.MinPlan) {
				fmt.Fprintf(w, "%s requires plan %s. Switch plan first.\n", r.Label, tailor.PlanLabel(r.MinPlan))
				return
			}
		}
	}
	fmt.Fprintf(w, "Item not found: %s\n", id)
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `OVAV Tailor Composer

Usage:
  tailor                  Interactive mode
  tailor status           Show current state
  tailor select <plan>    Select workstation plan (nucleo, studio, command)
  tailor toggle <item>    Toggle a tool or role
  tailor preview          Show pending changes
  tailor apply            Apply current selection
  tailor help             This help

Interactive commands:
  status, plan <id>, toggle <id>, preview, apply, help, quit`)
}
