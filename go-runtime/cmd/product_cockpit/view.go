package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── Styles ──────────────────────────────────────────────────────────

var (
	primary = lipgloss.Color("#2563eb") // Thavren blue
	green   = lipgloss.Color("#10b981")
	yellow  = lipgloss.Color("#f59e0b")
	red     = lipgloss.Color("#ef4444")
	muted   = lipgloss.Color("#6b7280")
	white   = lipgloss.Color("#f9fafb")
	bgDark  = lipgloss.Color("#111827")
	borderC = lipgloss.Color("#374151")

	titleStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(primary).
			Padding(0, 2).
			Bold(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	availableStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	statusOK = lipgloss.NewStyle().
			Foreground(green)

	statusErr = lipgloss.NewStyle().
			Foreground(red)

	mutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	selectedStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(primary).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderC).
			Padding(1, 2)

	successBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(green).
			Padding(1, 2)

	errorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red).
			Padding(1, 2)
)

// ── View ────────────────────────────────────────────────────────────

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// ── Header ──
	sb.WriteString(titleStyle.Render(" OVAV Product "))
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("  v%s  |  channel: %s  |  %s",
		m.version.Current, m.version.Channel, time.Now().Format("15:04"))))
	sb.WriteString("\n\n")

	// ── Connection Status ──
	sb.WriteString(renderConnection(m))
	sb.WriteString("\n")

	// ── Sync Queue Alert (GOV-009) ──
	if m.syncQueueItems > 0 {
		alert := boxStyle.Copy().BorderForeground(yellow).Render(
			availableStyle.Render(fmt.Sprintf("🔔 Sync package ready!  %d items queued for install", m.syncQueueItems)) +
				mutedStyle.Render("\nPress Enter on [Apply Update] to install all changes"))
		sb.WriteString(alert)
		sb.WriteString("\n")
	}

	// ── Version Box ──
	sb.WriteString(renderVersionBox(m))
	sb.WriteString("\n")

	// ── Update Status ──
	if m.updateApplying {
		sb.WriteString(boxStyle.Render(mutedStyle.Render("⏳ Applying update... Please wait.")))
		sb.WriteString("\n\n")
	} else if m.updateResult != "" {
		sb.WriteString(successBox.Render(statusOK.Render("✅ " + m.updateResult)))
		sb.WriteString("\n\n")
	} else if m.updateError != "" {
		sb.WriteString(errorBox.Render(statusErr.Render("❌ " + m.updateError)))
		sb.WriteString("\n\n")
	}

	// ── Actions ──
	sb.WriteString(renderActions(m))
	sb.WriteString("\n\n")

	// ── Footer ──
	sb.WriteString(mutedStyle.Render("OVAV Product Cockpit — ↑↓ select  •  Enter confirm  •  r refresh  •  q quit"))
	sb.WriteString("\n")

	return sb.String()
}

func renderConnection(m Model) string {
	switch m.sseStatus {
	case "connected":
		return statusOK.Render("● Connected to OVAV Systems") +
			mutedStyle.Render(fmt.Sprintf("  (%s)", m.cpanelURL))
	case "connecting":
		return mutedStyle.Render("◌ Connecting to " + m.cpanelURL + "...")
	case "error":
		return statusErr.Render("● Disconnected") +
			mutedStyle.Render(fmt.Sprintf("  (%s)", m.sseErr))
	default:
		return mutedStyle.Render("○ Not connected")
	}
}

func renderVersionBox(m Model) string {
	current := m.version.Current
	if current == "" {
		current = "checking..."
	}

	content := fmt.Sprintf("Installed: %s", versionStyle.Render(current))

	if m.updateAvailable {
		content += fmt.Sprintf("\nAvailable: %s", availableStyle.Render(m.version.Available))
		content += fmt.Sprintf("\n\n%s  Press Enter on [Apply Update] to install",
			availableStyle.Render("🔔 Update ready!"))
	} else if current != "" && current != "checking..." {
		content += fmt.Sprintf("\n%s", statusOK.Render("✓ Up to date"))
	}

	checkedAt := m.version.CheckedAt
	if checkedAt != "" {
		content += mutedStyle.Render(fmt.Sprintf("\n\nLast checked: %s", checkedAt))
	}

	if m.updateAvailable {
		return boxStyle.Copy().BorderForeground(yellow).Render(content)
	}
	return boxStyle.Render(content)
}

func renderActions(m Model) string {
	actions := []struct {
		label  string
		key    string
		active bool
	}{
		{"🔍  Check for Updates", "c", true},
		{"📡  Apply Update", "Enter", m.updateAvailable},
		{"🚀  Launch Agents in CWD", "Enter", true},
		{"❌  Quit", "q", true},
	}

	var sb strings.Builder
	for i, a := range actions {
		style := normalStyle
		prefix := "  "
		if i == m.cursor {
			style = selectedStyle
			prefix = "▸ "
		}
		if !a.active && i == m.cursor {
			// Skip non-active items when navigating
			prefix = "  "
			style = mutedStyle
		}

		label := a.label
		if !a.active {
			label = mutedStyle.Render(a.label + " (no update available)")
		}

		sb.WriteString(prefix + style.Render(label))
		sb.WriteString("\n")
	}

	return sb.String()
}
