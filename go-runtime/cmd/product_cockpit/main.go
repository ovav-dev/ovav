// OVAV Product Cockpit — End-user update receiver and applier.
//
// GOV-007: Minimal TUI for OVAV Product users. Connects via SSE to
// OVAV Systems cPanel, receives update notifications, and applies
// updates with a single keystroke.
//
// Usage:
//
//	ovav product cockpit              # Default: connects to localhost:5858
//	ovav product cockpit --url URL    # Custom cPanel URL
//
// The binary is installed as part of `ovav product install` alongside
// agents, skills, and identity files.

package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/internal/product"
)

func main() {
	url := flag.String("url", "", "cPanel URL (default: http://localhost:5858)")
	flag.Parse()

	cpanelURL := *url
	if cpanelURL == "" {
		cpanelURL = product.DefaultCPanelURL
	}
	if env := os.Getenv("OVAV_CPANEL_URL"); env != "" {
		cpanelURL = env
	}

	p := tea.NewProgram(
		NewModel(cpanelURL),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "product cockpit: %v\n", err)
		os.Exit(1)
	}

	// Post-exit: if update was applied, bootstrap CWD
	if product.NeedsUpdate() {
		fmt.Println("\n📡 Running ovav product launch to bootstrap CWD...")
		product.BootstrapCWD()
	}
}
