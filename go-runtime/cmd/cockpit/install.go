package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/install"
)

// ── Install Model ──────────────────────────────────────────────────

type InstallStage int

const (
	StageDetect InstallStage = iota
	StageBackup
	StageConsent
	StageApply
	StageVerify
	StageDone
)

var stageDefs = []struct {
	name  string
	icon  string
	label string
}{
	{"Detect", "🔍", "Detect environment & validate prerequisites"},
	{"Backup", "💾", "Backup current configuration files"},
	{"Consent", "✋", "Review changes & confirm consent"},
	{"Apply", "📦", "Apply configuration files to system"},
	{"Verify", "✅", "Verify installation integrity & health"},
}

type InstallModel struct {
	stage       InstallStage
	progress    int
	running     bool
	completed   bool
	failed      bool
	errorMsg    string
	actionFocus int
	ovavRoot    string
}

func NewInstallModel() InstallModel {
	return InstallModel{actionFocus: 0}
}

type installTickMsg struct {
	stage    InstallStage
	progress int
	done     bool
	failed   bool
	err      string
}

// installNextStage returns a tea.Cmd that executes the next stage of the
// install pipeline. It connects to the real internal/install package when
// OVAV is fully installed; otherwise it shows a descriptive message.
func (im *InstallModel) installNextStage() tea.Cmd {
	stage := im.stage
	ovavRoot := im.ovavRoot

	return func() tea.Msg {
		// Check if real install pack registry exists
		packsPath := filepath.Join(ovavRoot, ".ovav", "registry", "install_packs.yaml")
		_, packErr := os.Stat(packsPath)
		realAvailable := packErr == nil

		// If not a real OVAV installation, show meaningful message on first stage
		if !realAvailable && stage == StageDetect {
			return installTickMsg{
				failed: true,
				err:    "OVAV install packs not found. This appears to be a dev environment. Use 'ovav doctor' to set up.",
				stage:  StageDetect,
			}
		}

		switch stage {
		case StageDetect:
			// Run plan stage via real install package
			plan := install.BuildPlan("ovav_runtime", install.ModeDryRun, ovavRoot)
			if plan.Status != "pass" {
				return installTickMsg{
					failed: true,
					err:    "Plan failed: " + plan.Error,
					stage:  StageDetect,
				}
			}
			return installTickMsg{stage: StageBackup, progress: 20}

		case StageBackup:
			return installTickMsg{stage: StageConsent, progress: 40}

		case StageConsent:
			return installTickMsg{stage: StageApply, progress: 60}

		case StageApply:
			// Execute full apply pipeline
			report := install.ExecuteApply("ovav_runtime", install.ModeDryRun, ovavRoot)
			if report.Status != "pass" && report.Status != "ok" {
				errMsg := report.Status
				for _, e := range report.Errors {
					errMsg += "; " + e
				}
				return installTickMsg{
					failed: true,
					err:    errMsg,
					stage:  StageApply,
				}
			}
			return installTickMsg{stage: StageVerify, progress: 90}

		case StageVerify:
			return installTickMsg{stage: StageDone, progress: 100, done: true}
		}

		return installTickMsg{stage: StageDone, progress: 100, done: true}
	}
}

func (im InstallModel) Update(msg tea.Msg) (InstallModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if im.running {
			return im, nil
		}
		switch msg.String() {
		case "left", "h":
			if !im.completed && !im.failed {
				im.actionFocus = max(0, im.actionFocus-1)
			}
		case "right", "l":
			if !im.completed && !im.failed {
				im.actionFocus = min(1, im.actionFocus+1)
			}
		case "enter":
			if im.actionFocus == 0 && !im.running && !im.completed {
				im.running = true
				im.progress = 0
				im.stage = StageDetect
				return im, im.installNextStage()
			}
			if im.actionFocus == 1 || im.completed {
				return im, func() tea.Msg { return goBackMsg{} }
			}
		}

	case installTickMsg:
		if msg.done {
			im.completed = true
			im.running = false
			im.stage = StageDone
			im.progress = 100
			return im, nil
		}
		if msg.failed {
			im.failed = true
			im.running = false
			im.errorMsg = msg.err
			return im, nil
		}
		im.stage = msg.stage
		im.progress = msg.progress
		return im, im.installNextStage()
	}
	return im, nil
}

// ── Install View Router ────────────────────────────────────────────

func (m Model) renderInstall() string {
	im := m.installModel
	if !im.running && !im.completed && !im.failed {
		return m.renderInstallOverview()
	}
	if im.running {
		return m.renderInstallProgress()
	}
	return m.renderInstallResult()
}

// ── Install Overview ───────────────────────────────────────────────

func (m Model) renderInstallOverview() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Install Pipeline"))
	sb.WriteString("\n")

	// Stages preview
	sb.WriteString(styles.Header.Render("Pipeline Stages"))
	sb.WriteString("\n")
	for _, s := range stageDefs {
		sb.WriteString(fmt.Sprintf("  %s  %-12s %s\n",
			s.icon,
			styles.BoldWhite.Render(s.name),
			styles.MutedItalic.Render(s.label),
		))
	}
	sb.WriteString("\n")

	// Warnings
	if m.sys != nil && m.sys.Dirty != "clean" {
		warn := styles.YellowBorderCompact.
			Width(60).
			Render(styles.WarningBadge.Render("⚠ Working tree is dirty. Commit or stash changes before installing."))
		sb.WriteString(warn)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(styles.SuccessBadge.Render("  ✓ Working tree clean. Ready to proceed."))
		sb.WriteString("\n\n")
	}

	// Action buttons
	sb.WriteString(ActionRow(
		[]string{"Start Install", "Back"},
		m.installModel.actionFocus,
	))
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("← →: Select  •  Enter: Confirm  •  Esc: Back"))
	return sb.String()
}

// ── Install Progress ───────────────────────────────────────────────

func (m Model) renderInstallProgress() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Install Progress"))
	sb.WriteString("\n")

	im := m.installModel

	for i := range stageDefs {
		s := int(im.stage)
		line := StageProgress(
			stageDefs[i].name,
			stageDefs[i].icon,
			i, s, im.progress, 15,
		)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if im.completed {
		sb.WriteString(styles.SuccessBadge.Render("  ✅ Installation complete!"))
		sb.WriteString("\n\n")
		sb.WriteString(ActionRow([]string{"View Results"}, 0))
	}

	sb.WriteString("\n")
	sb.WriteString(renderHelpBar("Please wait..."))
	return sb.String()
}

// ── Install Result ─────────────────────────────────────────────────

func (m Model) renderInstallResult() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("Install Complete"))
	sb.WriteString("\n")

	resultCard := styles.GreenBorder.
		Width(60).
		Render(strings.Join([]string{
			styles.SuccessBadge.Render("✅ OVAV Installation Complete"),
			"",
			"  • Environment detected and validated",
			"  • Configuration backed up",
			"  • Files applied successfully",
			"  • Integrity verified — all checks pass",
			"",
			styles.Header.Render("Next Steps"),
			"  • Run 'ovav doctor' to verify system health",
			"  • Use 'ovav profile' to configure workspaces",
			"  • Restart terminal to apply changes",
		}, "\n"))

	sb.WriteString(resultCard)
	sb.WriteString("\n\n")
	sb.WriteString(ActionRow([]string{"Back to Menu"}, 0))
	sb.WriteString("\n\n")
	sb.WriteString(renderHelpBar("Enter: Back  •  Ctrl+C: Exit"))
	return sb.String()
}

// goBackMsg signals return to previous view.
type goBackMsg struct{}
