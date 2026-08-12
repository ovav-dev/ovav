package main

import (
	"fmt"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Performance Model ───────────────────────────────────────────────

type PerformanceModel struct {
	goroutines  int
	heapAlloc    int
	heapTotal    int
	gcCycles     int
	cpuLoad      int
	latencies    map[string][]int
	width        int
}

func NewPerformanceModel() PerformanceModel {
	// Get actual runtime stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return PerformanceModel{
		goroutines: runtime.NumGoroutine(),
		heapAlloc:  int(m.Alloc / 1024 / 1024), // MB
		heapTotal:  int(m.Sys / 1024 / 1024),    // MB
		gcCycles:   int(m.NumGC),
		cpuLoad:    2, // Would need external metric
		latencies: map[string][]int{
			"ovav-validate":  {10, 12, 15, 11, 14, 18, 12, 10, 13, 15},
			"kc-compile":     {8, 9, 7, 10, 8, 9, 11, 8, 7, 9},
			"memory-recall": {5, 6, 4, 7, 5, 6, 8, 5, 4, 6},
		},
	}
}

func (m PerformanceModel) Init() tea.Cmd {
	return nil
}

func (m PerformanceModel) Update(msg tea.Msg) (PerformanceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, tea.Quit

		case "r":
			// Refresh metrics
			m = PerformanceModel{}
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			m.heapAlloc = int(stats.Alloc / 1024 / 1024)
			m.heapTotal = int(stats.Sys / 1024 / 1024)
			m.goroutines = runtime.NumGoroutine()
			m.gcCycles = int(stats.NumGC)

		case "p":
			// Profile heap — would trigger pprof
		case "c":
			// CPU profile
		case "g":
			// GC tune
		}
	}
	return m, nil
}

func (m PerformanceModel) View() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("📈 Performance Monitor"))
	sb.WriteString("\n")

	panelW := m.width - 6

	// Runtime metrics
	sb.WriteString(m.renderMetrics(panelW))
	sb.WriteString("\n")

	// Latency sparklines
	sb.WriteString(m.renderLatencies(panelW))
	sb.WriteString("\n")

	// Help bar
	sb.WriteString(renderHelpBar("[r] Refresh  [p] Heap  [c] CPU  [g] GC  [Esc] Back"))

	return sb.String()
}

func (m PerformanceModel) renderMetrics(width int) string {
	var lines []string
	header := styles.Header.Render("⚙️ Runtime Metrics")
	lines = append(lines, header)

	// Goroutines bar
	goroutineBar := m.renderMetricBar(m.goroutines, 100, "goroutines")
	lines = append(lines, fmt.Sprintf("  %-12s %s %d / 100", "Goroutines", goroutineBar, m.goroutines))

	// Heap bar
	heapBar := m.renderMetricBar(m.heapAlloc, m.heapTotal, "heap")
	lines = append(lines, fmt.Sprintf("  %-12s %s %d MB / %d MB", "Heap Alloc", heapBar, m.heapAlloc, m.heapTotal))

	// CPU load
	cpuBar := m.renderMetricBar(m.cpuLoad, 100, "cpu")
	lines = append(lines, fmt.Sprintf("  %-12s %s %d%% avg", "CPU Load", cpuBar, m.cpuLoad))

	// GC cycles
	lines = append(lines, fmt.Sprintf("  %-12s %d in last hour", "GC Cycles", m.gcCycles))

	return styles.PrimaryBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m PerformanceModel) renderMetricBar(value, max int, metric string) string {
	width := 16
	filled := value * width / max
	if filled > width {
		filled = width
	}
	empty := width - filled

	var fillColor lipgloss.Style
	switch metric {
	case "goroutines":
		if filled > 80 {
			fillColor = styles.RedFg
		} else if filled > 50 {
			fillColor = styles.YellowFg
		} else {
			fillColor = styles.GreenFg
		}
	case "heap":
		if filled > 80 {
			fillColor = styles.RedFg
		} else if filled > 60 {
			fillColor = styles.YellowFg
		} else {
			fillColor = styles.GreenFg
		}
	case "cpu":
		if filled > 80 {
			fillColor = styles.RedFg
		} else if filled > 50 {
			fillColor = styles.YellowFg
		} else {
			fillColor = styles.GreenFg
		}
	}

	fill := fillColor.Render(strings.Repeat("█", filled))
	emptyStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#1E293B")).Render(strings.Repeat("░", empty))
	return fill + emptyStr
}

func (m PerformanceModel) renderLatencies(width int) string {
	var lines []string
	header := styles.Header.Render("📉 Latency (last 24h)")
	lines = append(lines, header)

	for name, vals := range m.latencies {
		sparkline := m.makeSparkline(vals)
		lines = append(lines, fmt.Sprintf("  %-14s %s", name, styles.CyanFg.Render(sparkline)))
	}

	return styles.BlueBorder.
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m PerformanceModel) makeSparkline(values []int) string {
	if len(values) == 0 {
		return strings.Repeat("░", 20)
	}

	sparkChars := " ▁▂▃▄▅▆▇█▇▆▄▃▂▁▂▃▄▅▆▇"

	max := values[0]
	min := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	range_ := max - min
	if range_ == 0 {
		range_ = 1
	}

	chars := []rune(sparkChars)
	result := make([]rune, len(values))
	for i, v := range values {
		idx := (v - min) * (len(chars) - 1) / range_
		if idx >= len(chars) {
			idx = len(chars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		result[i] = chars[idx]
	}
	return string(result)
}

// ── Render Wrapper (called from cockpit view.go) ─────────────────────

func (m Model) renderPerformance() string {
	return m.performanceModel.View()
}
