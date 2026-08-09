package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ovav/ovav/cmd/cockpit/data"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Model ──────────────────────────────────────────────────────────

type Model struct {
	// Navigation
	nav NavStack

	// Dimensions
	width  int
	height int
	ready  bool

	// OVAV root path
	ovavRoot string

	// Data
	caps *data.CapsData
	sys  *data.SystemInfo

	// Sub-models
	installModel InstallModel
	tailorModel  TailorModel
	planDetail   PlanDetailModel
	cliModel     CLISelectorModel
	syncModel    SyncModel
	configModel  ConfigModel
	updatesModel UpdatesModel
	vaultModel   vaultSubModel
	updateInfo   ProductVersionInfo // GOV-007: cPanel version check result
	menuCursor   int

	// UI state
	quitting        bool
	dashboardSearch bool
	dashboardFilter string

	// Viewport
	viewport viewport.Model
}

// ── Constructor ────────────────────────────────────────────────────

func NewModel() Model {
	ovavRoot := findOVAVRoot()

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary)

	m := Model{
		nav:          NewNavStack(ViewWelcome),
		ovavRoot:     ovavRoot,
		viewport:     vp,
		menuCursor:   0,
		installModel: NewInstallModel(),
		tailorModel:  NewTailorModel(),
		planDetail:   NewPlanDetailModel(),
		cliModel:     NewCLISelectorModel(ovavRoot),
		syncModel:    NewSyncModel(),
		configModel:  NewConfigModel(),
		updatesModel: NewUpdatesModel(),
		vaultModel:   vaultSubModel{state: vaultStateList, selected: 0, loading: true},
	}

	// Load data
	if caps, err := data.LoadCaps(ovavRoot); err == nil {
		m.caps = caps
	}
	sys := data.GatherSystemInfo(ovavRoot, m.caps, true)
	m.sys = &sys

	return m
}

// ── Init ───────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("OVAV Cockpit"),
	)
}
