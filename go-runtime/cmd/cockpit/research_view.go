package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Research Model ───────────────────────────────────────────────────

type ResearchModel struct {
	benchmarks []Benchmark
	evidence   []EvidenceScore
	sources    SourceStatus
	loading    bool
	cursor     int
	width      int
}

type Benchmark struct {
	Name    string
	Latency string
	Pct     int
	Level   string // "fastest", "fast", "medium", "slow"
}

type EvidenceScore struct {
	KB    string
	Score float64
}

type SourceStatus struct {
	Verified int
	Pending  int
	Failed   int
}

func NewResearchModel() ResearchModel {
	return ResearchModel{
		benchmarks: []Benchmark{
			{Name: "kc-compile", Latency: "12ms", Pct: 100, Level: "fastest"},
			{Name: "memory-bridge", Latency: "89ms", Pct: 62, Level: "medium"},
			{Name: "context-pack", Latency: "4ms", Pct: 100, Level: "fastest"},
			{Name: "ovav-validate", Latency: "234ms", Pct: 30, Level: "slow"},
		},
		evidence: []EvidenceScore{
			{KB: "KB-eidren", Score: 0.94},
			{KB: "KB-health", Score: 0.87},
			{KB: "KB-platform", Score: 0.91},
		},
		sources: SourceStatus{Verified: 12, Pending: 3, Failed: 1},
	}
}

func (m ResearchModel) Init() tea.Cmd {
	return nil
}

func (m ResearchModel) Update(msg tea.Msg) (ResearchModel, tea.Cmd) {
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
			if m.cursor < len(m.benchmarks)-1 {
				m.cursor++
			}
		case "b":
			// New benchmark — would trigger benchmark runner
		case "v":
			// Verify sources
		}
	}
	return m, nil
}

func (m ResearchModel) View() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("🔬 Research & Evidence"))
	sb.WriteString("\n")

	panelW := m.width - 6

	// Benchmarks
	sb.WriteString(m.renderBenchmarks(panelW))
	sb.WriteString("\n")

	// Two column: evidence scores + source status
	evBox := m.renderEvidenceScores(panelW/2 - 2)
	srcBox := m.renderSourceStatus(panelW/2 - 2)
	row := lipgloss.JoinHorizontal(lipgloss.Top, evBox, srcBox)
	sb.WriteString(row)
	sb.WriteString("\n\n")

	// Help bar
	sb.WriteString(renderHelpBar("[↑↓] Navigate  [b] +Benchmark  [v] Verify  [Esc] Back"))

	return sb.String()
}

func (m ResearchModel) renderBenchmarks(width int) string {
	var lines []string
	header := styles.Header.Render("📊 Benchmarks")
	lines = append(lines, header)

	levelColor := map[string]lipgloss.Style{
		"fastest": styles.GreenFg,
		"fast":    styles.GreenFg,
		"medium":  styles.YellowFg,
		"slow":    styles.RedFg,
	}

	sparkline := map[string]string{
		"fastest": "████████████████████",
		"fast":    "██████████████████░░",
		"medium":  "████████████░░░░░░░░",
		"slow":    "██████░░░░░░░░░░░░░░",
	}

	for i, b := range m.benchmarks {
		cursorMarker := "  "
		if i == m.cursor {
			cursorMarker = "▸ "
		}
		bar := levelColor[b.Level].Render(sparkline[b.Level])
		level := levelColor[b.Level].Render(strings.ToUpper(b.Level))
		line := fmt.Sprintf("%s%-16s %s %5s  %s", cursorMarker, b.Name, bar, b.Latency, level)
		lines = append(lines, line)
	}

	return styles.PrimaryBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m ResearchModel) renderEvidenceScores(width int) string {
	var lines []string
	header := styles.Header.Render("🧠 Evidence Scores")
	lines = append(lines, header)

	for _, e := range m.evidence {
		scoreColor := styles.GreenFg
		if e.Score < 0.8 {
			scoreColor = styles.YellowFg
		}
		if e.Score < 0.6 {
			scoreColor = styles.RedFg
		}
		line := fmt.Sprintf("  %-12s %s", e.KB, scoreColor.Render(fmt.Sprintf("%.2f", e.Score)))
		lines = append(lines, line)
	}

	return styles.BlueBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m ResearchModel) renderSourceStatus(width int) string {
	var lines []string
	header := styles.Header.Render("🔍 Source Status")
	lines = append(lines, header)

	lines = append(lines, fmt.Sprintf("  %s %d sources", styles.GreenFg.Render("✓"), m.sources.Verified))
	lines = append(lines, fmt.Sprintf("  %s %d pending", styles.YellowFg.Render("◐"), m.sources.Pending))
	lines = append(lines, fmt.Sprintf("  %s %d failed", styles.RedFg.Render("✗"), m.sources.Failed))

	return styles.YellowBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

// ── Render Wrapper (called from cockpit view.go) ─────────────────────

func (m Model) renderResearch() string {
	return m.researchModel.View()
}
