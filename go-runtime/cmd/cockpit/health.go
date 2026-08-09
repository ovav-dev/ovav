package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/data"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Health ─────────────────────────────────────────────────────────

func (m Model) healthUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		sys := data.GatherSystemInfo(m.ovavRoot, m.caps, true)
		m.sys = &sys
		return m, nil
	case "q", "esc", "backspace":
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
	}
	return m, nil
}

func (m Model) renderHealth() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Health & Status"))
	sb.WriteString("\n")

	if m.sys == nil {
		sb.WriteString(styles.ErrorBadge.Render("  ❌ No system info available"))
		sb.WriteString("\n\n")
		sb.WriteString(renderHelpBar("Esc: Back  •  ?: Help"))
		return sb.String()
	}

	s := m.sys
	panelW := m.width - 6

	// ── Top row: Identity + Git ──
	identityCard := renderCard("🆔 Identity", panelW/2-2,
		kv("Plan", s.PlanVersion),
		kv("Strategy", truncate(s.Strategy, 40)),
		kv("Root", truncate(s.OVAVRoot, 40)),
	)

	gitStatus := styles.SuccessBadge
	if s.Dirty != "clean" {
		gitStatus = styles.WarningBadge
	}
	branchColor := styles.PrimaryFg
	if s.Branch == "develop" || s.Branch == "main" {
		branchColor = styles.YellowFg
	}

	gitCard := renderCard("🔀 Git", panelW/2-2,
		kv("Branch", branchColor.Render(s.Branch)),
		kv("HEAD", styles.MutedFg.Render(s.SHA)),
		kv("Status", gitStatus.Render(s.Dirty)),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, identityCard, gitCard)
	sb.WriteString(topRow)
	sb.WriteString("\n\n")

	// ── Bottom row: System + Doctor + Caps ──
	systemCard := renderCard("⚙️ System", panelW/3-2,
		kv("Go", s.GoVersion),
		kv("Doctor", fmt.Sprintf("%d/%d pass", s.DoctorPass, s.DoctorTotal)),
	)

	doctorPct := 0
	if s.DoctorTotal > 0 {
		doctorPct = s.DoctorPass * 100 / s.DoctorTotal
	}
	doctorBar := VerticalPctBar("Health", doctorPct, 12)
	doctorCard := renderCard("🩺 Doctor", panelW/3-2,
		doctorBar,
		kv("Pass", styles.GreenFg.Render(fmt.Sprintf("%d", s.DoctorPass))),
		kv("Fail", styles.RedFg.Render(fmt.Sprintf("%d", s.DoctorFail))),
		kv("Warn", styles.YellowFg.Render(fmt.Sprintf("%d", s.DoctorWarn))),
	)

	capsCard := renderCard("📋 Caps", panelW/3-2,
		kv("Completed", styles.GreenFg.Render(fmt.Sprintf("%d", s.CapsDone))),
		kv("Pending", styles.YellowFg.Render(fmt.Sprintf("%d", s.CapsPending))),
	)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, systemCard, doctorCard, capsCard)
	sb.WriteString(bottomRow)
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("r: Refresh  •  Esc: Back  •  ?: Help"))
	return sb.String()
}

// ── Card helper ────────────────────────────────────────────────────

func renderCard(title string, width int, lines ...string) string {
	header := styles.CardHeader.Render(title)

	body := strings.Join(lines, "\n")

	content := header + "\n" + body

	return styles.PrimaryBorder.
		Width(width).
		Render(content)
}

func kv(key, value string) string {
	k := styles.KVKey.Render(key)
	v := styles.KVValue.Render(value)
	return fmt.Sprintf("  %s %s", k, v)
}
