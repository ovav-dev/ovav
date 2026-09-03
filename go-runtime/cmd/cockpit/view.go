package main

import "github.com/ovav/ovav/cmd/cockpit/styles"

// ── View Router ────────────────────────────────────────────────────

func (m Model) View() string {
	if m.quitting {
		return styles.App.Render("Exiting OVAV Cockpit...\n")
	}
	if !m.ready {
		return "Initializing...\n"
	}

	return m.renderCurrentView()
}

func (m Model) renderCurrentView() string {
	switch m.nav.Current() {
	case ViewWelcome:
		return m.renderWelcome()
	case ViewRoot:
		return m.renderRoot()
	case ViewDashboard:
		return m.renderDashboard()
	case ViewHealth:
		return m.renderHealth()
	case ViewVault:
		return m.renderVault()
	case ViewInstall:
		return m.renderInstall()
	case ViewTailor:
		return m.renderTailor()
	case ViewCLI:
		return m.renderCLI()
	case ViewSync:
		return m.renderSync()
	case ViewConfig:
		return m.renderConfig()
	case ViewUpdates:
		return m.renderUpdates()
	case ViewDetail:
		return m.renderPlanDetail()
	case ViewQuit:
		return m.renderQuit()
	case ViewHelp:
		return m.renderHelp()
	case ViewTesting:
		return m.renderTesting()
	case ViewDelegation:
		return m.renderDelegation()
	case ViewResearch:
		return m.renderResearch()
	case ViewAdversarial:
		return m.renderAdversarial()
	case ViewPerformance:
		return m.renderPerformance()
	default:
		return m.renderWelcome()
	}
}
