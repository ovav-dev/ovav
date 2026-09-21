package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/project"
)

// ── Sync View ─────────────────────────────────────────────────────

type SyncStep int

const (
	SyncStepIdle SyncStep = iota
	SyncStepRunning
	SyncStepDone
	SyncStepError
)

type SyncModel struct {
	step    SyncStep
	output  []string
	cursor  int
	verbose bool
	errMsg  string
}

func NewSyncModel() SyncModel {
	return SyncModel{
		step:    SyncStepIdle,
		output:  []string{},
		verbose: true,
	}
}

func (m Model) syncUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	sm := &m.syncModel

	switch msg.String() {
	case "enter":
		if sm.step == SyncStepIdle || sm.step == SyncStepDone || sm.step == SyncStepError {
			// Start sync
			sm.step = SyncStepRunning
			sm.output = []string{"🚀 Starting projection sync..."}
			sm.cursor = 0
			return m, m.runSync()
		}

	case "v":
		sm.verbose = !sm.verbose

	case "up", "k":
		if sm.cursor > 0 {
			sm.cursor--
		}

	case "down", "j":
		if sm.cursor < len(sm.output)-1 {
			sm.cursor++
		}

	case "q", "esc":
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
	}

	return m, nil
}

// syncResultMsg is sent when sync completes.
type syncResultMsg struct {
	output []string
	err    error
}

func (m Model) runSync() tea.Cmd {
	return func() tea.Msg {
		var output []string
		root := m.ovavRoot
		if root == "" {
			root = findOVAVRoot()
		}

		// Capture sync output by running projection steps individually
		output = append(output, fmt.Sprintf("📁 Root: %s", root))
		output = append(output, "")

		// Step 1: Agents
		output = append(output, "  ① Agent projection...")
		if cleaned, created, err := project.SyncAgents(root, true); err != nil {
			output = append(output, fmt.Sprintf("    ✗ FAILED: %v", err))
		} else {
			output = append(output, fmt.Sprintf("    ✓ %d cleaned, %d projected", cleaned, created))
		}

		// Step 2: ConnectorBus (skills + personnel)
		output = append(output, "  ② Skills & personnel projection...")
		if s, a, err := project.SyncConnectorBus(root, true); err != nil {
			output = append(output, fmt.Sprintf("    ✗ FAILED: %v", err))
		} else {
			output = append(output, fmt.Sprintf("    ✓ %d skills, %d agents synced", s, a))
		}

		// Step 3: Visual
		output = append(output, "  ③ Visual projection...")
		if v, err := project.SyncVisual(root, true); err != nil {
			output = append(output, fmt.Sprintf("    ✗ FAILED: %v", err))
		} else {
			output = append(output, fmt.Sprintf("    ✓ %d artifacts", v))
		}

		// Step 4: MiMo Code
		output = append(output, "  ④ MiMo Code projection...")
		if mc, err := project.SyncMiMoCode(root, true); err != nil {
			output = append(output, fmt.Sprintf("    ✗ FAILED: %v", err))
		} else {
			output = append(output, fmt.Sprintf("    ✓ %d artifacts", mc))
		}

		output = append(output, "")
		output = append(output, "✅ Projection sync complete")

		return syncResultMsg{output: output, err: nil}
	}
}

func (m Model) renderSync() string {
	var sb strings.Builder
	sm := m.syncModel

	sb.WriteString(renderTitleBar("Sync Projection"))
	sb.WriteString("\n\n")

	// Status badge
	switch sm.step {
	case SyncStepIdle:
		sb.WriteString(styles.YellowBorderCompact.Render("  ⏸  Ready to sync — Press Enter to start"))
	case SyncStepRunning:
		sb.WriteString(styles.BlueBorder.Render("  ⏳  Syncing projection..."))
	case SyncStepDone:
		sb.WriteString(styles.GreenBorderCompact.Render("  ✅  Sync complete"))
	case SyncStepError:
		sb.WriteString(styles.ErrorBadge.Render(fmt.Sprintf("  ❌  %s", sm.errMsg)))
	}
	sb.WriteString("\n\n")

	// Output log
	if len(sm.output) > 0 {
		logBox := styles.BlueBorder.
			Width(m.width - 4).
			Height(min(len(sm.output)+2, m.height-10)).
			Render(strings.Join(sm.output, "\n"))
		sb.WriteString(logBox)
		sb.WriteString("\n\n")
	}

	// Model routing info
	sb.WriteString(styles.MutedFg.Render("  Model Routing Active:"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("    Default: %s %s\n",
		styles.GreenFg.Render("openai/gpt-5.6-luna"),
		styles.MutedFg.Render("(primary)")))
	sb.WriteString(fmt.Sprintf("    Fallback: %s %s\n",
		styles.YellowFg.Render("minimax-coding-plan/MiniMax-M3"),
		styles.MutedFg.Render("(MiniMax API)")))
	sb.WriteString("\n")

	// Help
	helpItems := "Enter: Sync  •  v: Verbose  •  ↑↓: Scroll  •  Esc: Back"
	sb.WriteString(renderHelpBar(helpItems))

	return sb.String()
}
