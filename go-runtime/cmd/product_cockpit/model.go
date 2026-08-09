package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/internal/product"
)

// Model holds the Product Cockpit state.
type Model struct {
	cpanelURL string
	width     int
	height    int
	ready     bool

	// Version info from cPanel
	version product.UpdateInfo

	// SSE connection status
	sseStatus string // "disconnected", "connecting", "connected", "error"
	sseErr    string

	// Update state
	updateAvailable bool
	updateApplying  bool
	updateResult    string
	updateError     string

	// Post-update
	launchAfter bool

	// Sync queue (GOV-009)
	syncQueueItems int
	syncQueueErr   string

	// UI
	cursor   int // 0=check, 1=apply, 2=launch, 3=quit
	quitting bool
}

func NewModel(cpanelURL string) Model {
	return Model{
		cpanelURL: cpanelURL,
		sseStatus: "disconnected",
		cursor:    0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("OVAV Product"),
		fetchVersionCmd(m.cpanelURL),
		fetchSyncQueueCmd(m.cpanelURL),
	)

}
