package main

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ovav/ovav/cmd/cockpit/data"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

const (
	inactivityTickSeconds = 1
	inactivityTimeoutMins = 15
	inactivityMaxTicks    = inactivityTimeoutMins * 60 / inactivityTickSeconds
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
	installModel   InstallModel
	tailorModel    TailorModel
	planDetail     PlanDetailModel
	cliModel       CLISelectorModel
	syncModel      SyncModel
	configModel    ConfigModel
	updatesModel   UpdatesModel
	vaultModel     vaultSubModel
	testingModel     TestingModel
	delegationModel  DelegationModel
	researchModel    ResearchModel
	adversarialModel AdversarialModel
	performanceModel PerformanceModel
	updateInfo       ProductVersionInfo
	menuCursor     int

	// UI state
	quitting        bool
	dashboardSearch bool
	dashboardFilter string
	loading         bool

	// Inactivity timeout
	inactivityCounter int

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
		nav:            NewNavStack(ViewWelcome),
		ovavRoot:       ovavRoot,
		viewport:       vp,
		menuCursor:     0,
		loading:        true,
		inactivityCounter: 0,
		installModel:   NewInstallModel(),
		tailorModel:    NewTailorModel(),
		planDetail:     NewPlanDetailModel(),
		cliModel:       NewCLISelectorModel(ovavRoot),
		syncModel:      NewSyncModel(),
		configModel:    NewConfigModel(),
		updatesModel:   NewUpdatesModel(),
		vaultModel:     vaultSubModel{state: vaultStateList, selected: 0, loading: true},
		testingModel:     NewTestingModel(),
		delegationModel:  NewDelegationModel(),
		researchModel:    NewResearchModel(),
		adversarialModel: NewAdversarialModel(),
		performanceModel: NewPerformanceModel(),
	}

	return m
}

// ── Init ───────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("OVAV Cockpit"),
		m.loadDataAsync(),
		m.startInactivityTicker(),
	)
}

// ── Async Data Loading ─────────────────────────────────────────────

type dataLoadedMsg struct {
	caps *data.CapsData
	sys  *data.SystemInfo
}

func (m Model) loadDataAsync() tea.Cmd {
	return func() tea.Msg {
		ovavRoot := m.ovavRoot

		var caps *data.CapsData
		if c, err := data.LoadCaps(ovavRoot); err == nil {
			caps = c
		}

		sys := data.GatherSystemInfo(ovavRoot, caps, true)
		return dataLoadedMsg{caps: caps, sys: &sys}
	}
}

func (m Model) handleDataLoaded(msg dataLoadedMsg) {
	m.caps = msg.caps
	m.sys = msg.sys
	m.loading = false
	m.ready = true
}

// ── Inactivity Timeout ────────────────────────────────────────────

type inactivityTickMsg struct{}

func (m Model) startInactivityTicker() tea.Cmd {
	return tea.Tick(time.Duration(inactivityTickSeconds)*time.Second, func(t time.Time) tea.Msg {
		return inactivityTickMsg{}
	})
}

func (m Model) handleInactivityTick() (Model, tea.Cmd) {
	m.inactivityCounter++
	if m.inactivityCounter > inactivityMaxTicks {
		m.vaultModel.clearKey()
		m.inactivityCounter = 0
	}
	return m, m.startInactivityTicker()
}

// ResetInactivity resets the inactivity counter on user activity
func (m *Model) ResetInactivity() {
	m.inactivityCounter = 0
}
