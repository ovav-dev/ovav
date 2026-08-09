package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/data"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Plan Detail Model ──────────────────────────────────────────────

type PlanDetailModel struct {
	cap      *data.Cap
	pending  *data.PendingCap
	isActive bool
}

func NewPlanDetailModel() PlanDetailModel {
	return PlanDetailModel{}
}

func (m Model) renderPlanDetail() string {
	var sb strings.Builder
	pdm := m.planDetail

	if pdm.cap != nil {
		sb.WriteString(renderTitleBar(fmt.Sprintf("Cap: %s", pdm.cap.Name)))
		sb.WriteString("\n")

		info := styles.GreenBorder.
			Width(m.width - 6).
			Render(fmt.Sprintf("%s\n\nStatus:    %s\nType:      %s\nItems:     %d\nProgress:  %s %d%%\nMerged:    %s\nCommit:    %s\n\n%s",
				styles.Header.Render(pdm.cap.Name),
				styles.SuccessBadge.Render("completed"),
				pdm.cap.Type,
				pdm.cap.Items,
				renderPctBar(pdm.cap.Pct, 10),
				pdm.cap.Pct,
				pdm.cap.MergedAt,
				styles.MutedFg.Render(pdm.cap.Merge),
				styles.MutedFg.Italic(true).Render(pdm.cap.Summary),
			))
		sb.WriteString(info)

	} else if pdm.pending != nil {
		sb.WriteString(renderTitleBar(fmt.Sprintf("Cap: %s", pdm.pending.Name)))
		sb.WriteString("\n")

		tasks := strings.Join(pdm.pending.Tasks, "\n  • ")

		info := styles.YellowBorder.
			Width(m.width - 6).
			Render(fmt.Sprintf("%s\n\nStatus:    %s\nType:      %s\nOrder:     %d\nDeps:      %v\nWorktree:  %s\nStack:     %s\n\nTasks:\n  • %s\n\n%s",
				styles.Header.Render(pdm.pending.Name),
				styles.WarningBadge.Render("pending"),
				pdm.pending.Type,
				pdm.pending.Order,
				pdm.pending.Deps,
				pdm.pending.Worktree,
				styles.MutedFg.Render(pdm.pending.Stack),
				tasks,
				styles.MutedFg.Italic(true).Render(pdm.pending.Summary),
			))
		sb.WriteString(info)
	} else {
		sb.WriteString(renderTitleBar("Plan Detail"))
		sb.WriteString("\n")
		sb.WriteString(styles.MutedFg.Render("  No cap selected."))
	}

	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("Esc: Back  •  ?: Help"))
	return sb.String()
}

// ── Plan Detail Update ─────────────────────────────────────────────

func (m Model) planDetailUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "backspace":
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
	}
	return m, nil
}
