package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/data"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Welcome — Premium Splash ────────────────────────────────────────

func (m Model) welcomeUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.nav.Replace(ViewRoot)
		return m, nil
	case "s":
		// Quick sync from welcome
		m.nav.Push(ViewUpdates)
		return m, nil
	case "q", "ctrl+c":
		m.nav.Push(ViewQuit)
		return m, nil
	}
	return m, nil
}

func (m Model) renderWelcome() string {
	w := m.width
	if w < 40 {
		w = 80
	}

	var sb strings.Builder

	// ── Logo ──
	logo := `
   ╔══════════════════════════════╗
   ║     █▀█ █░█ ▄▀█ █░█        ║
   ║     █▄█ ▀▄▀ █▀█ ▀▄▀        ║
   ║                              ║
   ║  AI Workstation Governor     ║
   ║  Go Native Cockpit v2.0      ║
   ╚══════════════════════════════╝
`
	logoStyle := styles.Selected.
		Width(w - 4).
		Align(lipgloss.Center).
		Render(logo)
	sb.WriteString(logoStyle)
	sb.WriteString("\n")

	// ── System Status Bar ──
	status := ""
	if m.sys != nil {
		gitIcon := "●"
		gitColor := styles.GreenFg
		gitLabel := "clean"
		if m.sys.Dirty != "clean" {
			gitIcon = "●"
			gitColor = styles.YellowFg
			gitLabel = "dirty"
		}
		goVer := styles.PrimaryDimFg.Render(m.sys.GoVersion)

		status = fmt.Sprintf("  %s  %s  %s  %s  %s",
			styles.TagBlue.Render(" OVAV Systems "),
			styles.TagGreen.Render(fmt.Sprintf(" %s git ", gitLabel)),
			gitColor.Render(gitIcon+" "+m.sys.Branch),
			styles.TagPurple.Render(fmt.Sprintf(" %s ", m.sys.PlanVersion)),
			styles.MutedFg.Render(goVer),
		)
	} else {
		status = fmt.Sprintf("  %s  %s",
			styles.TagBlue.Render(" OVAV Systems "),
			styles.MutedFg.Render("Initializing..."),
		)
	}
	sb.WriteString(status)
	sb.WriteString("\n\n")

	// ── Quick Actions ──
	actions := []string{
		"[ Enter ]  🚀  Open Dashboard",
		"[   s   ]  ⚙️  Sync to Product",
		"[   q   ]  ⏏  Quit Cockpit",
	}
	for _, a := range actions {
		sb.WriteString(fmt.Sprintf("  %s\n", styles.BrightFg.Render(a)))
	}
	sb.WriteString("\n")

	// ── Project Summary (if loaded) ──
	if m.caps != nil {
		done := countDone(m.caps)
		total := done + len(m.caps.Pending)
		pct := 0
		if total > 0 {
			pct = done * 100 / total
		}

		bar := styles.AccentBorder.Width(w - 4).Render(
			fmt.Sprintf("  📊  Plan: %s  ·  %d/%d complete  ·  %d%%  ·  %s",
				styles.BoldWhite.Render(m.caps.PlanVersion),
				done, total, pct,
				styles.MutedFg.Render(m.caps.Strategy),
			))
		sb.WriteString(bar)
		sb.WriteString("\n\n")
	}

	// ── Footer ──
	sb.WriteString(renderHelpBar("Enter: Start  •  s: Quick Sync  •  q: Quit  •  ?: Help  •  Ctrl+C: Force quit"))

	return sb.String()
}

func countDone(caps *data.CapsData) int {
	n := 0
	for _, c := range caps.Caps {
		if c.Status == "done" {
			n++
		}
	}
	return n
}
