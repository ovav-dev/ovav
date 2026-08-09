package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.updateApplying {
			return m, nil // block input during apply
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			m.cursor = max(0, m.cursor-1)

		case "down", "j":
			m.cursor = min(3, m.cursor+1)

		case "enter", " ":
			return m, m.handleEnter()

		case "c":
			// Force version check
			return m, fetchVersionCmd(m.cpanelURL)

		case "r":
			// Reset state
			m.updateApplying = false
			m.updateResult = ""
			m.updateError = ""
			return m, fetchVersionCmd(m.cpanelURL)
		}

	// Version fetched from cPanel
	case versionMsg:
		if msg.err != nil {
			m.sseStatus = "error"
			m.sseErr = msg.err.Error()
		} else {
			m.version = msg.info
			m.sseStatus = "connected"
			m.sseErr = ""
			m.updateAvailable = msg.info.UpdateReady
		}
		// Start polling if connected
		if m.sseStatus == "connected" {
			return m, pollCmd()
		}
		return m, nil

	// Update dispatch result
	case updateDispatchMsg:
		m.updateApplying = false
		if msg.err != nil {
			m.updateResult = ""
			m.updateError = msg.err.Error()
		} else {
			m.updateResult = msg.msg
			m.updateError = ""
			m.launchAfter = msg.ok
			// Refresh version after update
			return m, fetchVersionCmd(m.cpanelURL)
		}

	// Auto-poll refresh
	case pollTickMsg:
		if !m.updateApplying {
			return m, tea.Batch(
				fetchVersionCmd(m.cpanelURL),
				fetchSyncQueueCmd(m.cpanelURL),
				pollCmd(),
			)
		}
		return m, pollCmd()

	// Sync queue status from cPanel (GOV-009)
	case syncQueueMsg:
		if msg.err != nil {
			m.syncQueueErr = msg.err.Error()
			m.syncQueueItems = -1
		} else {
			m.syncQueueItems = msg.itemsQueued
			m.syncQueueErr = ""
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleEnter() tea.Cmd {
	switch m.cursor {
	case 0: // Check for updates
		return fetchVersionCmd(m.cpanelURL)

	case 1: // Apply update
		if !m.updateAvailable {
			return nil
		}
		m.updateApplying = true
		m.updateResult = ""
		m.updateError = ""
		return dispatchUpdateCmd(m.cpanelURL)

	case 2: // Launch bootstrap in CWD
		m.updateApplying = true
		return launchBootstrapCmd()

	case 3: // Quit
		m.quitting = true
		return tea.Quit
	}
	return nil
}
