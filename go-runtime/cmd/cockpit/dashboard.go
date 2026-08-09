package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Dashboard ──────────────────────────────────────────────────────

type DashboardPanel int

const (
	PanelCompleted DashboardPanel = iota
	PanelPending
)

func (m Model) dashboardUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if m.dashboardSearch {
			m.dashboardSearch = false
			m.dashboardFilter = ""
			return m, nil
		}
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
	case "/":
		// Toggle search mode
		m.dashboardSearch = !m.dashboardSearch
		m.dashboardFilter = ""
		return m, nil
	case "enter":
		if m.dashboardSearch {
			// Execute search and close
			m.dashboardSearch = false
			return m, nil
		}
		// Show detail for selected cap
		m.navigateToCapDetail()
	case "up", "k":
		if m.dashboardSearch {
			return m, nil
		}
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.dashboardSearch {
			return m, nil
		}
		total := 0
		if m.caps != nil {
			for _, cap := range m.caps.Caps {
				if cap.Status == "done" {
					total++
				}
			}
			total += len(m.caps.Pending)
		}
		if m.menuCursor < total-1 {
			m.menuCursor++
		}
	case "backspace":
		if m.dashboardSearch && len(m.dashboardFilter) > 0 {
			m.dashboardFilter = m.dashboardFilter[:len(m.dashboardFilter)-1]
		} else if !m.dashboardSearch && m.nav.CanGoBack() {
			m.nav.Pop()
		}
	default:
		// In search mode, append to filter
		if m.dashboardSearch && len(msg.String()) == 1 {
			m.dashboardFilter += msg.String()
		}
	}
	return m, nil
}

// navigateToCapDetail navigates to the detail view for the currently selected cap.
func (m *Model) navigateToCapDetail() {
	if m.caps == nil {
		return
	}
	i := 0
	for _, cap := range m.caps.Caps {
		if cap.Status == "done" {
			if i == m.menuCursor {
				c := cap
				m.planDetail.cap = &c
				m.planDetail.pending = nil
				m.nav.Push(ViewDetail)
				return
			}
			i++
		}
	}
	// Check pending panel
	if m.menuCursor >= i && m.menuCursor < i+len(m.caps.Pending) {
		p := m.caps.Pending[m.menuCursor-i]
		m.planDetail.cap = nil
		m.planDetail.pending = &p
		m.nav.Push(ViewDetail)
	}
}

func (m Model) renderDashboard() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Plan Dashboard"))
	sb.WriteString("\n")

	if m.caps == nil {
		sb.WriteString(styles.ErrorBadge.Render("  ❌ Could not load caps.yaml"))
		sb.WriteString("\n\n")
		sb.WriteString(renderHelpBar("Esc: Back  •  ?: Help"))
		return sb.String()
	}

	// Strategy banner
	banner := styles.BlueBorder.
		Width(m.width - 4).
		Render(fmt.Sprintf("%s  %s  ·  %s",
			styles.BlueFg.Bold(true).Render(m.caps.PlanVersion),
			styles.MutedFg.Render(m.caps.Strategy),
			styles.MutedFg.Render(m.caps.StackTarget.Go),
		))
	sb.WriteString(banner)
	sb.WriteString("\n\n")

	// Search bar (when active)
	if m.dashboardSearch {
		searchBar := styles.YellowBorderCompact.
			Width(m.width - 6).
			Render(fmt.Sprintf("🔍 Filter: %s█", m.dashboardFilter))
		sb.WriteString(searchBar)
		sb.WriteString("\n\n")
	}

	// Two columns: completed | pending
	colWidth := (m.width - 8) / 2

	// ── Completed ──
	completedBox := m.renderCompletedCaps(colWidth)
	// ── Pending ──
	pendingBox := m.renderPendingCaps(colWidth)

	row := lipgloss.JoinHorizontal(lipgloss.Top, completedBox, pendingBox)
	sb.WriteString(row)
	sb.WriteString("\n\n")

	// Contextual help
	helpItems := "↑↓: Navigate  •  Enter: Detail  •  /: Search  •  Esc: Back  •  ?: Help"
	if m.dashboardSearch {
		helpItems = "Type to filter  •  Enter: Apply  •  Backspace: Clear  •  Esc: Cancel"
	}
	sb.WriteString(renderHelpBar(helpItems))
	return sb.String()
}

func (m Model) renderCompletedCaps(width int) string {
	var lines []string
	header := styles.Header.Render("✅ Completed")
	lines = append(lines, header)

	for id, cap := range m.caps.Caps {
		if cap.Status != "done" {
			continue
		}
		bar := renderPctBar(cap.Pct, 8)
		line := fmt.Sprintf("%-4s %-20s %s %3d%%",
			id, truncate(cap.Name, 19), bar, cap.Pct)
		lines = append(lines, line)
	}

	box := styles.GreenBorderCompact.
		Width(width).
		Render(strings.Join(lines, "\n"))
	return box
}

func (m Model) renderPendingCaps(width int) string {
	var lines []string
	header := styles.Header.Render("⬜ Pending")
	lines = append(lines, header)

	for _, p := range m.caps.Pending {
		bar := renderPctBar(p.Pct, 8)
		line := fmt.Sprintf("%-4s %-20s %s %3d%%",
			p.ID, truncate(p.Name, 19), bar, p.Pct)
		lines = append(lines, line)
	}

	box := styles.YellowBorderCompactPad.
		Width(width).
		Render(strings.Join(lines, "\n"))
	return box
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
