package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Menu Items ─────────────────────────────────────────────────────

type menuItem struct {
	id       string
	label    string
	icon     string
	category string
	desc     string
	hotkey   string
}

var menuItems = []menuItem{
	{id: "updates", label: "Work Done & Updates", icon: "🚀", category: "PRIMARY", desc: "See completed work ready to send to OVAV Product", hotkey: "u"},
	{id: "dashboard", label: "Plan Dashboard", icon: "📊", category: "SYSTEM", desc: "Implementation plan, caps & milestones", hotkey: "d"},
	{id: "health", label: "Health & Status", icon: "💚", category: "SYSTEM", desc: "System diagnostics, git, Go runtime", hotkey: "h"},
	{id: "vault", label: "OVAV Vault", icon: "🔐", category: "SECURITY", desc: "Intelligent secrets manager — credentials, keys, tokens", hotkey: "v"},
	{id: "sync", label: "Sync Projection", icon: "🔄", category: "SYSTEM", desc: "Project agents, skills, themes — sync with Product", hotkey: "s"},
	{id: "config", label: "Configuration", icon: "⚙️", category: "SYSTEM", desc: "Models, permissions, providers, routing", hotkey: "c"},
	{id: "install", label: "Install Pipeline", icon: "📦", category: "SYSTEM", desc: "Guided setup & deployment", hotkey: "i"},
	{id: "tailor", label: "Tailor Composer", icon: "🧩", category: "SYSTEM", desc: "Profile & workspace customization", hotkey: "t"},
	{id: "cli", label: "CLI Runtimes", icon: "⚡", category: "RUNTIMES", desc: "MiMo Code · OpenCode · Claude Code · Cursor", hotkey: "r"},
	{id: "testing", label: "Testing & Coverage", icon: "🧪", category: "DEVELOPER", desc: "Coverage sprint · test suites · loop detect", hotkey: "e"},
	{id: "delegation", label: "Delegation", icon: "🎯", category: "DEVELOPER", desc: "Active agents · delegation chains", hotkey: "g"},
	{id: "research", label: "Research", icon: "🔬", category: "DEVELOPER", desc: "Benchmarks · evidence scores", hotkey: "n"},
	{id: "adversarial", label: "Adversarial", icon: "🛡️", category: "DEVELOPER", desc: "Security gates · threat model", hotkey: "a"},
	{id: "performance", label: "Performance", icon: "📈", category: "DEVELOPER", desc: "Runtime metrics · profiling", hotkey: "p"},
}

// ── Root Update ────────────────────────────────────────────────────

func (m Model) rootUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.menuCursor < len(menuItems) {
			m.navigateToMenuItem(menuItems[m.menuCursor].id)
		}
		return m, nil

	case "u":
		m.navigateToMenuItem("updates")
		return m, nil

	case "d":
		m.menuCursor = 1 // dashboard position
		return m, nil

	case "h":
		m.menuCursor = 2
		return m, nil

	case "v":
		m.menuCursor = 3
		return m, nil

	case "s":
		m.menuCursor = 4
		return m, nil

	case "c":
		m.menuCursor = 5
		return m, nil

	case "i":
		m.menuCursor = 6
		return m, nil

	case "t":
		m.menuCursor = 7
		return m, nil

	case "r":
		m.menuCursor = 8
		return m, nil

	case "e":
		m.menuCursor = 9
		return m, nil

	case "g":
		m.menuCursor = 10
		return m, nil

	case "n":
		m.menuCursor = 11
		return m, nil

	case "a":
		m.menuCursor = 12
		return m, nil

	case "p":
		m.menuCursor = 13
		return m, nil

	case "up", "k":
		m.menuCursor = max(0, m.menuCursor-1)
		return m, nil

	case "down", "j":
		m.menuCursor = min(len(menuItems)-1, m.menuCursor+1)
		return m, nil

	case "q":
		m.nav.Push(ViewQuit)
		return m, nil
	}
	return m, nil
}

func (m *Model) navigateToMenuItem(id string) {
	switch id {
	case "updates":
		m.nav.Push(ViewUpdates)
	case "dashboard":
		m.nav.Push(ViewDashboard)
	case "health":
		m.nav.Push(ViewHealth)
	case "vault":
		m.nav.Push(ViewVault)
	case "install":
		m.nav.Push(ViewInstall)
	case "sync":
		m.nav.Push(ViewSync)
	case "config":
		m.nav.Push(ViewConfig)
	case "tailor":
		m.nav.Push(ViewTailor)
	case "cli":
		m.nav.Push(ViewCLI)
	case "testing":
		m.nav.Push(ViewTesting)
	case "delegation":
		m.nav.Push(ViewDelegation)
	case "research":
		m.nav.Push(ViewResearch)
	case "adversarial":
		m.nav.Push(ViewAdversarial)
	case "performance":
		m.nav.Push(ViewPerformance)
	}
}

// ── Root Render — Modern ───────────────────────────────────────────

const (
	rootMenuOffset = 7
	devListOffset  = 6
)

func (m Model) renderRoot() string {
	w := m.width
	if w < 60 {
		w = 60
	}

	var sb strings.Builder
	sb.WriteString(renderTitleBar(fmt.Sprintf("🚀  %s", "OVAV Cockpit")))
	sb.WriteString("\n")

	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  %s", m.ovavRoot)))

	if crumb := m.nav.Breadcrumb(); crumb != "" {
		sb.WriteString("  ")
		sb.WriteString(styles.Breadcrumb.Render(crumb))
	}
	sb.WriteString("\n\n")

	// ── Menu ──
	currentCategory := ""
	for i, item := range menuItems {
		// Category header
		if item.category != currentCategory {
			if currentCategory != "" {
				sb.WriteString("\n")
			}
			currentCategory = item.category
			catLabel := ""
			switch currentCategory {
			case "PRIMARY":
				catLabel = styles.TagGreen.Render(" ★ PRIMARY ")
			case "SYSTEM":
				catLabel = styles.PurpleCategory.Render("  SYSTEM")
			case "RUNTIMES":
				catLabel = styles.CyanFg.Bold(true).Render("  RUNTIMES")
			}
			sb.WriteString(fmt.Sprintf("  %s\n", catLabel))
		}

		isSelected := i == m.menuCursor

		// Build item with visual hierarchy
		line := m.buildMenuItem(item, isSelected)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// ── Help / Tips ──
	helpBox := styles.PrimaryHelpBorder.
		Width(w - 4).
		Render(fmt.Sprintf(" %s  ↑↓ Move  Enter Select  Hotkeys (u,d,h,v,s,c,i,t,r,e,g,n,a,p)  ? Help  q Quit  Esc Back ",
			styles.PrimaryDimFg.Render(fmt.Sprintf("%d/%d", m.menuCursor+1, len(menuItems)))))

	sb.WriteString(styles.MutedFg.Render(helpBox))

	return sb.String()
}

func (m Model) buildMenuItem(item menuItem, isSelected bool) string {
	if isSelected {
		hotkey := fmt.Sprintf("[%s]", styles.BoldWhite.Render(item.hotkey))
		left := fmt.Sprintf("▸ %s  %s  %s", item.icon, styles.BoldWhite.Render(item.label), styles.MutedFg.Render(hotkey))
		right := styles.MutedItalic.Render(item.desc)

		return styles.Selected.
			Width(m.width - 4).
			Render(fmt.Sprintf("  %-50s  %s", left, right))
	} else {
		left := fmt.Sprintf("  %s  %s", item.icon, styles.MutedFg.Render(item.label))
		return styles.Unselected.
			Render(fmt.Sprintf("  %s", left))
	}
}
