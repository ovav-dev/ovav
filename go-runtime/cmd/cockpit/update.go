package main

import tea "github.com/charmbracelet/bubbletea"

// ── Update ─────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	view := m.nav.Current()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6
		m.ready = true

	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "?":
			// Quick Help overlay — toggle from any view
			if view != ViewHelp {
				m.toggleHelp()
				return m, nil
			}

		case "esc":
			if view == ViewQuit {
				m.quitting = true
				return m, tea.Quit
			}
			if view == ViewHelp {
				m.nav.Pop()
				return m, nil
			}
			if m.nav.CanGoBack() {
				m.nav.Pop()
				return m, nil
			}
			return m, nil
		}

		// Route to view handler
		return m.handleViewKey(msg, view)

	case tea.MouseMsg:
		return m.handleMouse(msg, view)

	case goBackMsg:
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
		return m, nil

	case installTickMsg:
		im, subCmd := m.installModel.Update(msg)
		m.installModel = im
		return m, subCmd

	case tailorDoneMsg:
		m.tailorModel.step = TailorDone
		return m, nil

	case syncResultMsg:
		m.syncModel.output = msg.output
		if msg.err != nil {
			m.syncModel.step = SyncStepError
			m.syncModel.errMsg = msg.err.Error()
		} else {
			m.syncModel.step = SyncStepDone
		}
		return m, nil

	case productSyncMsg:
		m.updatesModel.syncStatus = "done"
		if msg.err != nil {
			m.updatesModel.syncStatus = "error"
			m.updatesModel.syncMsg = msg.err.Error()
		} else {
			m.updatesModel.syncMsg = msg.msg
			for i := range m.updatesModel.items {
				if m.updatesModel.items[i].Status == "done" {
					m.updatesModel.items[i].Status = "synced"
				}
			}
		}
		return m, nil

	case syncItemsMsg:
		// GOV-009: Sync engine status loaded (non-blocking)
		if len(msg.items) > 0 {
			m.updatesModel.items = append(msg.items, gatherPhase2Items()...)
		}
		return m, nil

	case updateCheckMsg:
		// GOV-007: cPanel version check result
		if msg.err != nil {
			m.updateInfo.Error = msg.err.Error()
		} else {
			m.updateInfo = msg.info
		}
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// ── View Key Router ────────────────────────────────────────────────

func (m Model) handleViewKey(msg tea.KeyMsg, view string) (tea.Model, tea.Cmd) {
	switch view {
	case ViewWelcome:
		return m.welcomeUpdate(msg)
	case ViewRoot:
		return m.rootUpdate(msg)
	case ViewDashboard:
		return m.dashboardUpdate(msg)
	case ViewHealth:
		return m.healthUpdate(msg)
	case ViewVault:
		return m.vaultUpdate(msg)
	case ViewInstall:
		m.installModel.ovavRoot = m.ovavRoot
		im, cmd := m.installModel.Update(msg)
		m.installModel = im
		return m, cmd
	case ViewTailor:
		tm, cmd := m.tailorModel.Update(msg)
		m.tailorModel = tm
		return m, cmd
	case ViewDetail:
		return m.planDetailUpdate(msg)
	case ViewCLI:
		return m.cliUpdate(msg)
	case ViewSync:
		return m.syncUpdate(msg)
	case ViewConfig:
		return m.configUpdate(msg)
	case ViewUpdates:
		return m.updatesUpdate(msg)
	case ViewHelp:
		return m.helpUpdate(msg)
	case ViewQuit:
		if msg.String() == "enter" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// ── Mouse Router ──────────────────────────────────────────────────

func (m Model) handleMouse(msg tea.MouseMsg, view string) (tea.Model, tea.Cmd) {
	// Only handle left clicks
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	switch view {
	case ViewRoot:
		// Safe click on menu items with bounds checking
		row := msg.Y - 4
		if row >= 0 && row < len(menuItems) {
			m.menuCursor = row
			item := menuItems[row]
			switch item.id {
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
			}
			return m, nil
		}

	case ViewDashboard:
		// Click anywhere in dashboard → no-op (use keyboard)
		return m, nil

	case ViewHelp:
		m.nav.Pop()
		return m, nil

	case ViewInstall:
		m.installModel.ovavRoot = m.ovavRoot
		im, cmd := m.installModel.Update(msg)
		m.installModel = im
		return m, cmd

	case ViewSync:
		// Click on sync view → start sync if idle
		if m.syncModel.step == SyncStepIdle || m.syncModel.step == SyncStepDone || m.syncModel.step == SyncStepError {
			m.syncModel.step = SyncStepRunning
			m.syncModel.output = []string{"🚀 Starting projection sync..."}
			return m, m.runSync()
		}
		return m, nil

	case ViewTailor:
		tm, cmd := m.tailorModel.Update(msg)
		m.tailorModel = tm
		return m, cmd

	case ViewWelcome:
		m.nav.Replace(ViewRoot)
		return m, nil

	case ViewQuit:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}
