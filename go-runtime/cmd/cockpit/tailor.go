package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/tailor"
)

// ── Tailor Model ───────────────────────────────────────────────────

type TailorStep int

const (
	TailorSelect TailorStep = iota
	TailorPreview
	TailorConfirm
	TailorDone
)

type TailorModel struct {
	step        TailorStep
	state       *tailor.State
	cursor      int // index into state.SelectableRows()
	actionFocus int
	width       int
}

// tailorDoneMsg signals that the tailor install step is complete.
type tailorDoneMsg struct{}

func NewTailorModel() TailorModel {
	return TailorModel{
		step:   TailorSelect,
		state:  tailor.NewState(nil),
		cursor: 0,
	}
}

// SelectableCount returns the number of interactive rows.
func (tm TailorModel) SelectableCount() int {
	return tm.state.SelectableCount()
}

// ── Tailor Update ──────────────────────────────────────────────────

func (tm TailorModel) Update(msg tea.Msg) (TailorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch tm.step {
		case TailorSelect:
			return tm.selectUpdate(msg)
		case TailorPreview:
			return tm.previewUpdate(msg)
		case TailorConfirm:
			return tm.confirmUpdate(msg)
		}

	case tea.MouseMsg:
		return tm.mouseUpdate(msg)
	}
	return tm, nil
}

func (tm TailorModel) selectUpdate(msg tea.KeyMsg) (TailorModel, tea.Cmd) {
	n := tm.SelectableCount()
	if n == 0 {
		return tm, nil
	}

	switch msg.String() {
	case "up", "k":
		tm.cursor = (tm.cursor - 1 + n) % n
	case "down", "j":
		tm.cursor = (tm.cursor + 1) % n
	case " ":
		tm.state.ToggleAt(tm.cursor)
	case "enter":
		if tm.state.SelectedPlan != "" {
			tm.step = TailorPreview
			tm.actionFocus = 0
		}
	}
	return tm, nil
}

func (tm TailorModel) previewUpdate(msg tea.KeyMsg) (TailorModel, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		tm.actionFocus = max(0, tm.actionFocus-1)
	case "right", "l":
		tm.actionFocus = min(1, tm.actionFocus+1)
	case "enter":
		if tm.actionFocus == 0 {
			// Install
			tm.state.ApplySelection()
			tm.step = TailorConfirm
			return tm, func() tea.Msg { return tailorDoneMsg{} }
		}
		// Back
		tm.step = TailorSelect
		return tm, nil
	}
	return tm, nil
}

func (tm TailorModel) confirmUpdate(msg tea.KeyMsg) (TailorModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		tm.step = TailorDone
		return tm, func() tea.Msg { return goBackMsg{} }
	}
	return tm, nil
}

func (tm TailorModel) mouseUpdate(msg tea.MouseMsg) (TailorModel, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if tm.step == TailorSelect {
			n := tm.SelectableCount()
			if n > 0 {
				tm.cursor = (tm.cursor - 1 + n) % n
			}
		}
	case tea.MouseButtonWheelDown:
		if tm.step == TailorSelect {
			n := tm.SelectableCount()
			if n > 0 {
				tm.cursor = (tm.cursor + 1) % n
			}
		}
	case tea.MouseButtonLeft:
		if tm.step == TailorSelect {
			row := msg.Y - 3 // offset for title bar + header
			if row >= 0 && row < tm.SelectableCount() {
				tm.cursor = row
				tm.state.ToggleAt(row)
			}
		}
	}
	return tm, nil
}

// ── Tailor View Router ─────────────────────────────────────────────

func (m Model) renderTailor() string {
	// Inject width into tailor model for layout
	m.tailorModel.width = m.width

	tm := m.tailorModel
	switch tm.step {
	case TailorSelect:
		return m.renderTailorSelect()
	case TailorPreview:
		return m.renderTailorPreview()
	case TailorConfirm, TailorDone:
		return m.renderTailorConfirm()
	default:
		return m.renderTailorSelect()
	}
}

// ── Tailor Select ─────────────────────────────────────────────────

func (m Model) renderTailorSelect() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Tailor Composer"))
	sb.WriteString("\n")

	tm := m.tailorModel
	s := tm.state

	// Plan badge
	planStyle := styles.BlueBorder.
		Width(min(m.width-6, 70)).
		Render(fmt.Sprintf("%s  Plan: %s  |  %d tools · %d roles  |  %s",
			styles.BlueFg.Bold(true).Render("🧩"),
			styles.WhiteFg.Bold(true).Render(tailor.PlanLabel(s.SelectedPlan)),
			s.ActiveToolCount(), s.ActiveRoleCount(),
			styles.MutedFg.Italic(true).Render(tailor.PlanLabel(s.SelectedPlan)+" · "+s.InstallSummary()),
		))
	sb.WriteString(planStyle)
	sb.WriteString("\n\n")

	// Sectioned rows rendering
	rows := s.SectionedRows()
	selectableIdx := 0

	for _, row := range rows {
		switch row.Type {
		case "section":
			sb.WriteString(styles.Header.Render("▸ " + row.Label))
			sb.WriteString("\n")

		case "item":
			isFocused := selectableIdx == tm.cursor
			selectableIdx++

			// Determine checkbox
			checkbox := "[ ]"
			if row.Active {
				checkbox = "[✓]"
			}
			if row.Kind == "plan" && row.Active {
				checkbox = "[●]"
			}

			// Determine style
			itemStyle := styles.Unselected
			cursorMark := "  "
			if isFocused {
				itemStyle = styles.Selected
				cursorMark = "▸ "
			}

			// Status color
			status := styles.MutedFg
			if row.Active {
				if row.Kind == "plan" {
					status = styles.BlueFg
				} else {
					status = styles.GreenFg
				}
			}

			// Detected note
			extra := ""
			if row.DetectedNote == "detected" {
				extra = styles.GreenFg.Render(" ● detected")
			} else if row.Kind == "tool" {
				extra = styles.MutedFg.Render(" ○ " + row.DetectedNote)
			}
			if row.Kind == "plan" {
				extra = styles.MutedFg.Italic(true).Render(row.Note)
			}

			line := fmt.Sprintf("%s%s %s %-20s %s",
				cursorMark, checkbox,
				status.Render(row.Label),
				"",
				extra,
			)
			sb.WriteString(itemStyle.Width(m.width - 4).Render(line))
			sb.WriteString("\n")

		case "gap":
			sb.WriteString("\n")

		case "action":
			isFocused := selectableIdx == tm.cursor
			selectableIdx++

			actionStyle := styles.Unselected
			cursorMark := "  "
			if isFocused {
				actionStyle = styles.Selected
				cursorMark = "▸ "
			}

			allowed := row.Allowed
			actionLabel := styles.GreenFg.Bold(true).Render(row.Label)
			if !allowed {
				actionLabel = styles.MutedFg.Render(row.Label + " (select a plan first)")
			}

			line := fmt.Sprintf("%s  %s %s",
				cursorMark,
				actionLabel,
				styles.MutedFg.Italic(true).Render(row.Note),
			)
			sb.WriteString(actionStyle.Width(m.width - 4).Render(line))
			sb.WriteString("\n")
		}
	}

	// Hint bar
	hint := tm.state.RowHint(tm.cursor)
	sb.WriteString("\n")
	sb.WriteString(styleHelp(styles.PurpleHelpBorder.
		Render(" " + hint + " ")))
	sb.WriteString("\n")
	sb.WriteString(renderHelpBar("↑↓: Navigate  ·  Space: Toggle  ·  Enter: Preview  ·  Esc: Back"))
	return sb.String()
}

// ── Tailor Preview ─────────────────────────────────────────────────

func (m Model) renderTailorPreview() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Tailor Preview"))
	sb.WriteString("\n")

	tm := m.tailorModel
	s := tm.state

	// Summary card
	changes := s.PreviewChanges()
	nChanges := len(changes)
	summaryStyle := styles.BlueBorder.
		Width(min(m.width-6, 60))

	if nChanges == 0 {
		sb.WriteString(summaryStyle.Render(
			fmt.Sprintf("%s  No changes pending.\n     Current plan: %s",
				styles.MutedFg.Render("ℹ"),
				styles.WhiteFg.Bold(true).Render(tailor.PlanLabel(s.SelectedPlan)),
			)))
	} else {
		sb.WriteString(summaryStyle.Render(
			fmt.Sprintf("%s  %d change(s) pending:", styles.BlueFg.Render("◆"), nChanges)))
	}
	sb.WriteString("\n\n")

	// Change list
	for _, ch := range changes {
		mark := "+"
		color := styles.GreenFg
		if !ch.After {
			mark = "−"
			color = styles.MutedFg
		}
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n",
			color.Render(mark),
			styles.WhiteFg.Render(ch.Label),
			styles.MutedFg.Render(ch.Summary),
		))
	}

	// Install summary
	sb.WriteString("\n")
	sb.WriteString(styleHelp(styles.PurpleHelpBorder.
		Render(fmt.Sprintf(" Will install: %s", s.InstallSummary()))))
	sb.WriteString("\n\n")

	// Actions
	sb.WriteString(ActionRow([]string{"Install", "Back"}, tm.actionFocus))
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("← →: Select  ·  Enter: Confirm  ·  Esc: Back"))
	return sb.String()
}

// ── Tailor Confirm ─────────────────────────────────────────────────

func (m Model) renderTailorConfirm() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Tailor Complete"))
	sb.WriteString("\n")

	tm := m.tailorModel
	s := tm.state

	// Results
	rows := s.InstallConfirmationRows()
	var resultLines []string
	for _, r := range rows {
		resultLines = append(resultLines, fmt.Sprintf("  %-14s %s",
			styles.MutedFg.Render(r.Label),
			styles.WhiteFg.Render(r.Value),
		))
	}

	card := styles.GreenBorderLarge.
		Width(min(m.width-6, 60)).
		Render(fmt.Sprintf("%s\n\n%s\n\n%s\n  • ovav doctor to verify\n  • Restart terminal to activate",
			styles.SuccessBadge.Render("✅ Tailor Complete"),
			styles.Header.Render("Configuration Summary"),
			strings.Join(resultLines, "\n"),
		))

	sb.WriteString(card)
	sb.WriteString("\n\n")
	sb.WriteString(ActionRow([]string{"Back to Menu"}, 0))
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("Enter: Back  ·  Esc: Back"))
	return sb.String()
}
