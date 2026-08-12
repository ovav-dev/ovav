package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Delegation Model ────────────────────────────────────────────────

type DelegationModel struct {
	sessions  []Session
	chains    []Chain
	loading   bool
	cursor    int
	width     int
}

type Session struct {
	Lead     string
	Agent    string
	Status   string // "running", "idle", "crashed"
	Duration time.Duration
	Task     string
}

type Chain struct {
	Lead    string
	Agents  []string
	RootCmd string
}

func NewDelegationModel() DelegationModel {
	m := DelegationModel{
		sessions: []Session{
			{Lead: "thavren", Agent: "andres", Status: "running", Duration: 12 * time.Minute, Task: "refactor-validators"},
			{Lead: "thavren", Agent: "clara", Status: "running", Duration: 8 * time.Minute, Task: "test-coverage"},
			{Lead: "eidren", Agent: "carmen", Status: "idle", Duration: 2 * time.Hour, Task: "research-cache"},
			{Lead: "eidren", Agent: "fatima", Status: "running", Duration: 1 * time.Hour, Task: "benchmark-runner"},
		},
		chains: []Chain{
			{Lead: "thavren", Agents: []string{"andres", "clara"}, RootCmd: "refactor + test"},
			{Lead: "eidren", Agents: []string{"carmen", "fatima"}, RootCmd: "research + benchmark"},
		},
	}
	return m
}

func (m DelegationModel) Init() tea.Cmd {
	return nil
}

func (m DelegationModel) Update(msg tea.Msg) (DelegationModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "r":
			// Refresh — would reload from registry
		case "x":
			// Kill selected session
			if m.cursor < len(m.sessions) && m.sessions[m.cursor].Status == "running" {
				m.sessions[m.cursor].Status = "crashed"
			}
		}
	}
	return m, nil
}

func (m DelegationModel) View() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("🎯 Delegation Runtime"))
	sb.WriteString("\n")

	panelW := m.width - 6

	// Active sessions
	sb.WriteString(m.renderActiveSessions(panelW))
	sb.WriteString("\n")

	// Delegation chains
	sb.WriteString(m.renderChains(panelW))
	sb.WriteString("\n")

	// Help bar
	sb.WriteString(renderHelpBar("[↑↓] Navigate  [k] Kill  [r] Refresh  [Esc] Back"))

	return sb.String()
}

func (m DelegationModel) renderActiveSessions(width int) string {
	var lines []string
	header := styles.Header.Render("⚡ Active Sessions")
	lines = append(lines, header)

	statusIcon := map[string]string{
		"running": styles.GreenFg.Render("●"),
		"idle":    styles.MutedFg.Render("○"),
		"crashed": styles.RedFg.Render("✗"),
	}

	for i, s := range m.sessions {
		icon := statusIcon[s.Status]
		highlight := ""
		cursorMarker := "  "
		if i == m.cursor {
			highlight = styles.PrimaryFg.Render()
			cursorMarker = "▸ "
		}

		duration := fmt.Sprintf("%dm", int(s.Duration.Minutes()))
		if s.Duration.Hours() >= 1 {
			duration = fmt.Sprintf("%.1fh", s.Duration.Hours())
		}

		line := fmt.Sprintf("%s%s%-8s ──► %-8s %s %s %s",
			cursorMarker, highlight+s.Lead, styles.MutedFg.Render("→"), s.Agent, icon, duration, styles.MutedFg.Render(s.Task))
		if i == m.cursor {
			line += " " + styles.YellowFg.Render("◀")
		}
		lines = append(lines, line)
	}

	return styles.PrimaryBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m DelegationModel) renderChains(width int) string {
	var lines []string
	header := styles.Header.Render("🔗 Delegation Chains")
	lines = append(lines, header)

	for _, c := range m.chains {
		chainStr := c.Lead
		for _, a := range c.Agents {
			chainStr += styles.MutedFg.Render(" → ") + a
		}
		lines = append(lines, fmt.Sprintf("  %s", chainStr))
		lines = append(lines, styles.MutedFg.Render(fmt.Sprintf("     └─ %s", c.RootCmd)))
		lines = append(lines, "")
	}

	return styles.AccentBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

// ── Render Wrapper (called from cockpit view.go) ─────────────────────

func (m Model) renderDelegation() string {
	return m.delegationModel.View()
}
