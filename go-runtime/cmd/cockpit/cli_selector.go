package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/convert"
)

// ── CLI Selector Model ──────────────────────────────────────────────

type cliTarget struct {
	Target  convert.Target
	Label   string
	Desc    string
	Status  string // "generated", "available", "not_generated"
	GenDate string
}

type CLISelectorModel struct {
	cursor    int
	targets   []cliTarget
	generated bool
	message   string
	ovavRoot  string
}

func NewCLISelectorModel(ovavRoot string) CLISelectorModel {
	targets := []cliTarget{
		{Target: convert.TargetMimocode, Label: "MiMo Code", Desc: "Xiaomi MiMo — free model, auto-bootstrap ★DEFAULT"},
		{Target: convert.TargetOpenCode, Label: "OpenCode", Desc: "Primary CLI — markdown agents in .opencode/agents/"},
		{Target: convert.TargetClaude, Label: "Claude Code", Desc: "Anthropic Claude Code — agents in .claude/agents/"},
		{Target: convert.TargetCursor, Label: "Cursor", Desc: "Cursor IDE — rules in .cursor/rules/"},
	}
	// Check status
	for i := range targets {
		targets[i].Status = checkTargetStatus(ovavRoot, targets[i].Target)
	}
	return CLISelectorModel{
		targets:  targets,
		ovavRoot: ovavRoot,
	}
}

func checkTargetStatus(root string, target convert.Target) string {
	c, err := convert.GetConverter(target)
	if err != nil {
		return "unavailable"
	}
	outputDir := filepath.Join(root, c.OutputDir())
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return "not_generated"
	}
	// Check if there are files
	entries, err := os.ReadDir(outputDir)
	if err != nil || len(entries) == 0 {
		return "not_generated"
	}
	return "generated"
}

// ── CLI View ─────────────────────────────────────────────────────────

func (m Model) renderCLI() string {
	var sb strings.Builder
	sb.WriteString(renderTitleBar("CLI Runtime Selector"))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("Generate OVAV agents for any supported CLI runtime."))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("Canonical source: ovav/agents/{areas,leads,teams}/*.yaml"))
	sb.WriteString("\n\n")

	model := &m.cliModel

	// Status message
	if model.message != "" {
		sb.WriteString(styles.GreenFg.Render(model.message))
		sb.WriteString("\n\n")
	}

	for i, t := range model.targets {
		isSelected := i == model.cursor
		cursor := "  "
		itemStyle := styles.Unselected

		if isSelected {
			cursor = "▸ "
			itemStyle = styles.Selected
		}

		// Status indicator
		statusIcon := ""
		statusStyle := styles.MutedFg
		switch t.Status {
		case "generated":
			statusIcon = "✅"
			statusStyle = styles.GreenFg
		case "not_generated":
			statusIcon = "⬜"
			statusStyle = styles.MutedFg
		case "unavailable":
			statusIcon = "❌"
			statusStyle = styles.RedFg
		}

		line := fmt.Sprintf("%s%s %s %s — %s",
			cursor, statusIcon, t.Label, statusStyle.Render(t.Status), styles.MutedFg.Render(t.Desc))
		sb.WriteString(itemStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("g — Generate files for selected CLI  |  Enter — Select & Preview"))
	sb.WriteString("\n")
	sb.WriteString(styleHelp(styles.PrimaryHelpBorder.Render(
		fmt.Sprintf(" %d/%d  ↑↓ navigate  g generate  Esc back  q quit ",
			model.cursor+1, len(model.targets)))))
	sb.WriteString("\n\n")

	// Show generation preview if requested
	if model.generated {
		sb.WriteString(renderGenerationPreview(model.ovavRoot, model.targets[model.cursor]))
	}

	return sb.String()
}

func renderGenerationPreview(root string, target cliTarget) string {
	var sb strings.Builder
	sb.WriteString(styles.PurpleCategory.Render("📁 Files that will be generated:"))
	sb.WriteString("\n")

	c, err := convert.GetConverter(target.Target)
	if err != nil {
		return styles.RedFg.Render(fmt.Sprintf("Error: %v", err))
	}

	outputDir := filepath.Join(root, c.OutputDir())
	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  Output: %s/", c.OutputDir())))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  Extension: %s", c.FileExtension())))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  Full path: %s", outputDir)))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  Files: 10 areas + 10 leads + 50 teams + 1 governor = 71 total")))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("  Source: ovav/agents/{areas,leads,teams}/*.yaml (canonical)"))
	sb.WriteString("\n")

	return sb.String()
}

// ── CLI Update ───────────────────────────────────────────────────────

func (m Model) cliUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	model := &m.cliModel

	switch msg.String() {
	case "up", "k":
		if model.cursor > 0 {
			model.cursor--
		}
		return m, nil

	case "down", "j":
		if model.cursor < len(model.targets)-1 {
			model.cursor++
		}
		return m, nil

	case "enter":
		// Preview toggle
		model.generated = !model.generated
		return m, nil

	case "g":
		// Generate files for selected target
		target := model.targets[model.cursor]
		if target.Status == "unavailable" {
			model.message = fmt.Sprintf("❌ %s converter not available", target.Label)
			return m, nil
		}

		err := convert.GenerateAll(
			filepath.Join(m.ovavRoot, "ovav", "agents"),
			target.Target,
			m.ovavRoot,
		)
		if err != nil {
			model.message = fmt.Sprintf("❌ Generation failed: %v", err)
		} else {
			model.message = fmt.Sprintf("✅ %s runtime files generated successfully!", target.Label)
			model.generated = true
			// Update status
			model.targets[model.cursor].Status = "generated"
		}
		return m, nil
	}
	return m, nil
}

// ── Helpers ──────────────────────────────────────────────────────────

// IsCLIView checks if we're in the CLI selector view
func isCLIView(view string) bool {
	return view == ViewCLI
}
