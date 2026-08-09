package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// Button renders a selectable action button.
type Button struct {
	Label    string
	Active   bool
	Disabled bool
}

func RenderButton(label string, active bool) string {
	if active {
		return styles.ActiveButton.Render(fmt.Sprintf(" ▶ %s ", label))
	}
	return styles.InactiveButton.Render(fmt.Sprintf("   %s ", label))
}

// ActionRow renders a horizontal row of buttons.
func ActionRow(buttons []string, focus int) string {
	var parts []string
	for i, label := range buttons {
		parts = append(parts, RenderButton(label, i == focus))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}
