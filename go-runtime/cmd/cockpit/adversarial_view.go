package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Adversarial Model ──────────────────────────────────────────────

type AdversarialModel struct {
	gates     []Gate
	threats   ThreatModel
	audits    []AuditEntry
	loading   bool
	cursor    int
	width     int
}

type Gate struct {
	Name        string
	Status      string // "pass", "warn", "fail"
	Description string
}

type ThreatModel struct {
	AttackSurface int
	Critical      int
	High          int
	Medium        int
	Low           int
}

type AuditEntry struct {
	Date    string
	Name    string
	Status  string
	Details string
}

func NewAdversarialModel() AdversarialModel {
	return AdversarialModel{
		gates: []Gate{
			{Name: "workspace-safety", Status: "pass", Description: "Auto-trigger on write"},
			{Name: "git-push-gate", Status: "pass", Description: "HTTPS-only, no force push"},
			{Name: "protected-branch", Status: "pass", Description: "main/develop/staging protected"},
			{Name: "secrets-hygiene", Status: "pass", Description: "No plaintext in tracked files"},
			{Name: "coverage-gate", Status: "warn", Description: "78% < 80% threshold"},
			{Name: "F0-validate", Status: "pass", Description: "All F0 validators green"},
		},
		threats: ThreatModel{
			AttackSurface: 12,
			Critical:      0,
			High:          2,
			Medium:        4,
			Low:           6,
		},
		audits: []AuditEntry{
			{Date: "2026-08-11", Name: "secrets-scan", Status: "CLEAN", Details: ""},
			{Date: "2026-08-10", Name: "dependency", Status: "CLEAN", Details: ""},
			{Date: "2026-08-09", Name: "push-audit", Status: "CLEAN", Details: ""},
			{Date: "2026-08-08", Name: "scope-risk", Status: "2 WARN", Details: "elevated"},
		},
	}
}

func (m AdversarialModel) Init() tea.Cmd {
	return nil
}

func (m AdversarialModel) Update(msg tea.Msg) (AdversarialModel, tea.Cmd) {
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
			if m.cursor < len(m.gates)-1 {
				m.cursor++
			}
		case "a":
			// Run full audit
		case "t":
			// View threat model detail
		case "s":
			// Scan secrets
		}
	}
	return m, nil
}

func (m AdversarialModel) View() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("🛡️ Adversarial & Security"))
	sb.WriteString("\n")

	panelW := m.width - 6

	// Security gates
	sb.WriteString(m.renderGates(panelW))
	sb.WriteString("\n")

	// Two column: threat model + recent audits
	threatBox := m.renderThreatModel(panelW/2 - 2)
	auditBox := m.renderAudits(panelW/2 - 2)
	row := lipgloss.JoinHorizontal(lipgloss.Top, threatBox, auditBox)
	sb.WriteString(row)
	sb.WriteString("\n\n")

	// Help bar
	sb.WriteString(renderHelpBar("[↑↓] Navigate  [a] Audit  [t] Threat  [s] Secrets  [Esc] Back"))

	return sb.String()
}

func (m AdversarialModel) renderGates(width int) string {
	var lines []string
	header := styles.Header.Render("🔒 Security Gates")
	lines = append(lines, header)

	statusIcon := map[string]string{
		"pass": styles.GreenFg.Render("🟢"),
		"warn": styles.YellowFg.Render("🟡"),
		"fail": styles.RedFg.Render("🔴"),
	}

	for i, g := range m.gates {
		cursorMarker := "  "
		if i == m.cursor {
			cursorMarker = "▸ "
		}
		icon := statusIcon[g.Status]
		line := fmt.Sprintf("%s%s %-18s %s %s", cursorMarker, icon, g.Name, styles.MutedFg.Render(g.Description), styles.MutedFg.Render("[Auto]"))
		lines = append(lines, line)
	}

	return styles.PrimaryBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m AdversarialModel) renderThreatModel(width int) string {
	var lines []string
	header := styles.Header.Render("⚠️ Threat Model")
	lines = append(lines, header)

	lines = append(lines, fmt.Sprintf("  %s %2d pts", styles.MutedFg.Render("Surface:"), m.threats.AttackSurface))
	lines = append(lines, fmt.Sprintf("  %s %d", styles.RedFg.Render("Critical:"), m.threats.Critical))
	lines = append(lines, fmt.Sprintf("  %s %d", styles.YellowFg.Render("High:"), m.threats.High))
	lines = append(lines, fmt.Sprintf("  %s %d", styles.MutedFg.Render("Medium:"), m.threats.Medium))
	lines = append(lines, fmt.Sprintf("  %s %d", styles.MutedFg.Render("Low:"), m.threats.Low))

	return styles.PrimaryBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m AdversarialModel) renderAudits(width int) string {
	var lines []string
	header := styles.Header.Render("📋 Recent Audits")
	lines = append(lines, header)

	statusColor := map[string]lipgloss.Style{
		"CLEAN":  styles.GreenFg,
		"2 WARN": styles.YellowFg,
		"FAIL":   styles.RedFg,
	}

	for _, a := range m.audits {
		color, ok := statusColor[a.Status]
		if !ok {
			color = styles.MutedFg
		}
		line := fmt.Sprintf("  %s %s  %s", a.Date, a.Name, color.Render(a.Status))
		lines = append(lines, line)
	}

	return styles.BlueBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

// ── Render Wrapper (called from cockpit view.go) ─────────────────────

func (m Model) renderAdversarial() string {
	return m.adversarialModel.View()
}
