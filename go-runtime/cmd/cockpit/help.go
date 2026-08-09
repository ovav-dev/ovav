package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Quick Help Overlay ───────────────────────────────────────────────
// Accessible from any view with '?' key.
// Shows keyboard shortcuts and navigation tips contextual to current view.

func (m Model) helpUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		// Close help overlay
		m.nav.Pop()
		return m, nil
	}
	return m, nil
}

func (m Model) renderHelp() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Quick Help"))
	sb.WriteString("\n\n")

	// Global shortcuts
	sb.WriteString(styles.Header.Render("🌐 Global"))
	sb.WriteString("\n")
	sb.WriteString(renderShortcut("Esc", "Go back / close overlay"))
	sb.WriteString(renderShortcut("Ctrl+C", "Force quit"))
	sb.WriteString(renderShortcut("?", "Toggle this help"))
	sb.WriteString("\n")

	// Navigation
	sb.WriteString(styles.Header.Render("🧭 Navigation"))
	sb.WriteString("\n")
	sb.WriteString(renderShortcut("↑↓ / j k", "Move cursor up/down"))
	sb.WriteString(renderShortcut("Enter", "Select / drill in"))
	sb.WriteString(renderShortcut("Esc / Backspace", "Go back"))
	sb.WriteString(renderShortcut("q", "Quit (from main menu)"))
	sb.WriteString("\n")

	// Contextual shortcuts based on current view
	current := m.nav.PeekBelow()
	switch current {
	case ViewDashboard:
		sb.WriteString(styles.Header.Render("📊 Dashboard"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("↑↓", "Navigate caps"))
		sb.WriteString(renderShortcut("Enter", "View cap details"))
		sb.WriteString(renderShortcut("/", "Search/filter caps"))
		sb.WriteString(renderShortcut("Click", "Select cap with mouse"))
		sb.WriteString("\n")
	case ViewHealth:
		sb.WriteString(styles.Header.Render("💚 Health"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("r", "Refresh data"))
		sb.WriteString(renderShortcut("Enter", "Run full diagnostic"))
		sb.WriteString("\n")
	case ViewInstall:
		sb.WriteString(styles.Header.Render("📦 Install Pipeline"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("Enter / Click", "Start pipeline"))
		sb.WriteString(renderShortcut("Tab", "Next step"))
		sb.WriteString(renderShortcut("Space", "Confirm / Toggle"))
		sb.WriteString("\n")
	case ViewTailor:
		sb.WriteString(styles.Header.Render("🧩 Tailor Composer"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("↑↓ / Scroll", "Navigate options"))
		sb.WriteString(renderShortcut("Space / Click", "Toggle selection"))
		sb.WriteString(renderShortcut("Enter", "Next step"))
		sb.WriteString("\n")
	case ViewCLI:
		sb.WriteString(styles.Header.Render("🔄 CLI Selector"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("↑↓", "Select target"))
		sb.WriteString(renderShortcut("g", "Generate files"))
		sb.WriteString(renderShortcut("Enter", "Preview generation"))
		sb.WriteString("\n")
	case ViewRoot:
		sb.WriteString(styles.Header.Render("🏠 Main Menu"))
		sb.WriteString("\n")
		sb.WriteString(renderShortcut("↑↓ / j k", "Navigate menu"))
		sb.WriteString(renderShortcut("Enter / Click", "Open section"))
		sb.WriteString(renderShortcut("q", "Quit"))
		sb.WriteString("\n")
	}

	// Productivity
	sb.WriteString(styles.Header.Render("⚡ Productivity"))
	sb.WriteString("\n")
	sb.WriteString(renderShortcut("Mouse", "Click buttons, select items"))
	sb.WriteString(renderShortcut("Scroll", "Navigate long lists"))
	sb.WriteString("\n")

	// Footer
	sb.WriteString(styles.MutedFg.Italic(true).Render(
		"  OVAV Cockpit v1.0 · Go Native · Bubble Tea TUI · Zero Python",
	))
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("?: Close help  •  All shortcuts work in every view"))

	return sb.String()
}

func renderShortcut(key, desc string) string {
	return fmt.Sprintf("  %-22s %s\n",
		styles.PrimaryFg.Bold(true).Render(key),
		styles.MutedFg.Render(desc),
	)
}

// ── Help overlay dispatch ────────────────────────────────────────────

// toggleHelp pushes or pops the help overlay.
func (m *Model) toggleHelp() {
	if m.nav.Current() == ViewHelp {
		m.nav.Pop()
	} else {
		m.nav.Push(ViewHelp)
	}
}

// PeekBelow returns the view that would be visible if help was dismissed.
// Used to show contextual shortcuts.
func (ns *NavStack) PeekBelow() string {
	if ns.Depth() < 2 {
		return ViewRoot
	}
	return ns.stack[ns.Depth()-2]
}
