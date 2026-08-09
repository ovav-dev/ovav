// OVAV Cockpit — Go Native TUI with Bubble Tea.
//
// Replaces tools/cli/ovav_first_run_cockpit.py (3435 LOC Python+curses).
// Elm Architecture. Mouse native. Zero Python bridge.
//
// Stack: Go stdlib + charmbracelet/bubbletea + charmbracelet/bubbles + charmbracelet/lipgloss.
//
// Build: go build -o bin/cockpit ./cmd/cockpit/
// Run:   ./bin/cockpit

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create the program with initial model
	p := tea.NewProgram(
		NewModel(),
		tea.WithAltScreen(),       // Full alternate screen
		tea.WithMouseCellMotion(), // Mouse support
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "OVAV Cockpit error: %v\n", err)
		os.Exit(1)
	}
}
