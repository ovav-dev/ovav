package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// StageProgress renders a single pipeline stage line.
func StageProgress(name, icon string, stage, current int, pct, barWidth int) string {
	var indicator string
	var style lipgloss.Style

	switch {
	case stage < current:
		indicator = "✅"
		style = styles.SuccessBadge
	case stage == current:
		indicator = icon
		style = styles.ActiveStage
	default:
		indicator = "⬜"
		style = styles.Unselected
	}

	// Progress bar
	var bar string
	if stage < current {
		bar = styles.ProgressFill.Render(strings.Repeat("█", barWidth))
	} else if stage == current {
		filled := pct * barWidth / 100
		bar = styles.ProgressFill.Render(strings.Repeat("█", filled)) +
			styles.ProgressEmpty.Render(strings.Repeat("░", barWidth-filled))
	} else {
		bar = styles.ProgressEmpty.Render(strings.Repeat("░", barWidth))
	}

	return style.Render(fmt.Sprintf("  %s %-10s  %s %3d%%", indicator, name, bar, pct))
}

// VerticalPctBar renders a vertical-layout percentage bar with label.
func VerticalPctBar(label string, pct, barWidth int) string {
	filled := pct * barWidth / 100
	empty := barWidth - filled

	fill := styles.ProgressFill.Render(strings.Repeat("█", filled))
	empt := styles.ProgressEmpty.Render(strings.Repeat("░", empty))

	return fmt.Sprintf("%-22s %s %3d%%", label, fill+empt, pct)
}
