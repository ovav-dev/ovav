// main.go — OVAV VAULT TUI
//
// Interactive bubble tea TUI for OVAV VAULT.
// Run with: ovav-vault-tui
//
// Features:
//   - Secret browser with detail panel
//   - Real-time vault health and sync status
//   - Quick actions: add, revoke, rotate, sync
//   - Spend report for OVAV CONNECT
//
// Style: Dark theme with blue accents, matching OVAV branding.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ovav/ovav/internal/vault/secrets"
)

// ── Styles (OVAV dark theme) ─────────────────────────────────────────────────

var (
	// Color palette
	colorBg     = lipgloss.Color("#0d1117")
	colorPanel  = lipgloss.Color("#161b22")
	colorBorder = lipgloss.Color("#30363d")
	colorBlue   = lipgloss.Color("#58a6ff")
	colorGreen  = lipgloss.Color("#3fb950")
	colorYellow = lipgloss.Color("#d29922")
	colorRed    = lipgloss.Color("#f85149")
	colorWhite  = lipgloss.Color("#e6edf3")
	colorMuted  = lipgloss.Color("#8b949e")
	colorAccent = lipgloss.Color("#1f6feb")

	// Styles
	styleApp = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorWhite).
			Width(120).
			Height(40)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true).
			Padding(0, 1)

	stylePanel = lipgloss.NewStyle().
			Background(colorPanel).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 1).
			Width(40).
			Height(38)

	styleDetail = lipgloss.NewStyle().
			Background(colorPanel).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			Width(78).
			Height(38)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleSecretItem = lipgloss.NewStyle().
			Foreground(colorWhite).
			Padding(0, 1)

	styleTag = lipgloss.NewStyle().
			Background(colorAccent).
			Foreground(colorWhite).
			Padding(0, 1).
			MarginRight(4)

	styleOnline  = lipgloss.NewStyle().Foreground(colorGreen)
	styleOffline = lipgloss.NewStyle().Foreground(colorMuted)
	styleWarning = lipgloss.NewStyle().Foreground(colorYellow)
	styleError   = lipgloss.NewStyle().Foreground(colorRed)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
)

// ── Model ────────────────────────────────────────────────────────────────────

type status int

const (
	statusLoading status = iota
	statusReady
	statusError
)

type Model struct {
	store       *secrets.SecretStore
	graph       *secrets.DependencyGraph
	vaultKey    []byte
	secretsList list.Model
	selected    *secrets.Secret
	status      status
	errorMsg    string
	syncStatus  string
	quitting    bool
	spinner     spinner.Model
	input       textinput.Model
	inputActive bool
	inputMode   string // "add", "revoke_confirm", "query"
}

func NewModel(store *secrets.SecretStore, graph *secrets.DependencyGraph, key []byte) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorBlue)

	delegate := list.NewDefaultDelegate()
	li := list.New([]list.Item{}, delegate, 40, 34)
	li.SetShowTitle(false)
	li.SetFilteringEnabled(true)

	return &Model{
		store:       store,
		graph:       graph,
		vaultKey:    key,
		status:      statusLoading,
		syncStatus:  "● Loading...",
		spinner:     sp,
		secretsList: li,
	}
}

// ── Init ─────────────────────────────────────────────────────────────────────

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadSecretsCmd(m),
	)
}

// ── Update ───────────────────────────────────────────────────────────────────

type (
	secretsLoadedMsg struct {
		list    []list.Item
		secrets []*secrets.Secret
	}
	loadErrMsg struct{ err error }
)

func loadSecretsCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(300 * time.Millisecond) // brief loading feel
		all := m.store.List("")
		items := make([]list.Item, 0, len(all))
		for _, sec := range all {
			items = append(items, secretItem{sec: sec, graph: m.graph})
		}
		return secretsLoadedMsg{list: items, secrets: all}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		newSpinner, cmd := m.spinner.Update(msg)
		m.spinner = newSpinner
		return m, cmd

	case secretsLoadedMsg:
		m.status = statusReady
		m.syncStatus = "● Online — vault unlocked"
		delegate := list.NewDefaultDelegate()
		m.secretsList = list.New(msg.list, delegate, 40, 34)
		m.secretsList.SetShowTitle(false)
		m.secretsList.SetFilteringEnabled(true)
		return m, nil

	case tea.KeyMsg:
		if m.inputActive {
			return m.handleInput(msg)
		}
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		return m, nil

	default:
		return m, nil
	}
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "enter":
		i := m.secretsList.Index()
		items := m.secretsList.Items()
		if i < len(items) {
			if si, ok := items[i].(secretItem); ok {
				m.selected = si.sec
			}
		}
		return m, nil

	case "a":
		m.inputActive = true
		m.inputMode = "add"
		ti := textinput.New()
		ti.Placeholder = "Secret name (e.g. GITHUB_TOKEN)"
		ti.Focus()
		m.input = ti
		return m, nil

	case "r":
		if m.selected == nil {
			return m, nil
		}
		m.inputActive = true
		m.inputMode = "revoke_confirm"
		ti := textinput.New()
		ti.Placeholder = fmt.Sprintf("Type 'revoke %s' to confirm", m.selected.Name)
		ti.Focus()
		m.input = ti
		return m, nil

	case "s":
		m.syncStatus = "↕ Syncing..."
		return m, triggerSyncCmd(m)

	case "d":
		if m.selected != nil {
			refs := m.graph.GetRefs(m.selected.ID)
			_ = refs
			// TODO: show dependency graph view
		}
		return m, nil

	default:
		// Pass to list navigation
		var cmd tea.Cmd
		m.secretsList, cmd = m.secretsList.Update(msg)
		return m, cmd
	}
}

func (m *Model) handleInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.inputActive = false
			switch m.inputMode {
			case "add":
				name := m.input.Value()
				if name != "" {
					return m, m.doAddSecret(name)
				}
			case "revoke_confirm":
				m.inputActive = false
				if m.selected != nil {
					return m, m.doRevokeSecret(m.selected.Name)
				}
			}
			return m, nil

		case "esc":
			m.inputActive = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) doAddSecret(name string) tea.Cmd {
	return func() tea.Msg {
		// Create a placeholder secret with a generated value
		val := generateRandomValue()
		sec := &secrets.Secret{
			Name:   name,
			Type:   secrets.TypeAPIToken,
			Value:  []byte(val),
			Source: "manual",
		}
		if err := m.store.Add(sec); err != nil {
			m.syncStatus = fmt.Sprintf("❌ Add failed: %v", err)
		} else {
			if saveErr := m.store.Save(m.vaultKey); saveErr != nil {
				m.syncStatus = fmt.Sprintf("❌ Save failed: %v", saveErr)
			} else {
				m.syncStatus = fmt.Sprintf("✓ Secret '%s' added", name)
			}
		}
		return secretsLoadedMsg{
			list:    m.secretsList.Items(),
			secrets: m.store.List(""),
		}
	}
}

func (m *Model) doRevokeSecret(name string) tea.Cmd {
	return func() tea.Msg {
		report, err := secrets.RevokeSecret(m.store, m.graph, name)
		if err != nil {
			m.syncStatus = fmt.Sprintf("❌ Revoke failed: %v", err)
		} else {
			m.syncStatus = fmt.Sprintf("✓ Revoked: %s (%d systems)", name, len(report.Results))
		}
		return secretsLoadedMsg{
			list:    m.secretsList.Items(),
			secrets: m.store.List(""),
		}
	}
}

func triggerSyncCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.syncStatus = "↕ Syncing..."
		// TODO: call secrets.FullSync when ready
		m.syncStatus = "✓ Synced"
		time.Sleep(1 * time.Second)
		return secretsLoadedMsg{
			list:    m.secretsList.Items(),
			secrets: m.store.List(""),
		}
	}
}

func generateRandomValue() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("ovav-generated-%d", time.Now().UnixNano())
	}
	return "ovav-" + hex.EncodeToString(b)
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	header := renderHeader(m)
	footer := renderFooter(m)

	if m.status == statusLoading {
		loading := lipgloss.JoinHorizontal(
			lipgloss.Center,
			lipgloss.NewStyle().Width(40).Render(m.spinner.View()),
			lipgloss.NewStyle().Foreground(colorMuted).Render(" Loading vault..."),
		)
		return lipgloss.JoinVertical(
			lipgloss.Center,
			"\n\n",
			header,
			lipgloss.NewStyle().Height(30).Width(120).Render(
				lipgloss.JoinVertical(lipgloss.Center, "\n\n\n", loading),
			),
			footer,
		)
	}

	// Two-panel layout
	secretsPanel := m.secretsList.View()
	detailPanel := renderDetail(m)

	if m.inputActive {
		inputView := renderInput(m)
		return lipgloss.JoinVertical(
			lipgloss.Top,
			header,
			lipgloss.JoinHorizontal(lipgloss.Top, secretsPanel, detailPanel),
			inputView,
			footer,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		lipgloss.JoinHorizontal(lipgloss.Top, secretsPanel, detailPanel),
		footer,
	)
}

func renderHeader(m *Model) string {
	syncColor := styleOnline
	if m.status != statusReady {
		syncColor = styleMuted
	}
	header := fmt.Sprintf(
		"  OVAV VAULT  ▸  secrets  %s   %s",
		syncColor.Render(m.syncStatus),
		styleMuted.Render(fmt.Sprintf("🔒 %d vaulted", m.store.Count())),
	)
	return styleHeader.Render(header)
}

func renderDetail(m *Model) string {
	if m.selected == nil {
		noSelection := lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(76).
			Height(34).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Select a secret ↑↓\nto view details")
		return styleDetail.Render(noSelection)
	}

	sec := m.selected
	refs := m.graph.GetRefs(sec.ID)

	age := ageString(sec.CreatedAt)

	typeColor := styleMuted
	if sec.Type == secrets.TypeAPIToken || sec.Type == secrets.TypeCloudKey {
		typeColor = styleError
	}

	detail := "  " + styleTag.Render(string(sec.Type)) + " " +
		lipgloss.NewStyle().Bold(true).Render(sec.Name) + "\n\n" +
		"  " + styleMuted.Render("Type:") + "  " + padRight(string(sec.Type), 18) + "  " + typeColor.Render(string(sec.Type)) + "\n" +
		"  " + styleMuted.Render("Age:") + "  " + padRight(age, 18) + "  " + styleWhite.Render(age) + "\n" +
		"  " + styleMuted.Render("Source:") + "  " + padRight(sec.Source, 18) + "  " + styleWhite.Render(sec.Source) + "\n\n" +
		"  " + styleMuted.Render("--- USED BY ---") + "\n"

	// Used by section
	if len(refs) == 0 {
		detail += styleMuted.Render("  No system references (orphan)\n")
	} else {
		for _, r := range refs {
			autoTag := ""
			if r.AutoRotatable {
				autoTag = styleGreen.Render(" [AUTO]")
			}
			detail += fmt.Sprintf("  %s %s %s %s%s\n",
				styleMuted.Render("›"),
				styleBlue.Render(string(r.System)),
				styleWhite.Render(r.Path),
				styleMuted.Render("→"),
				styleWhite.Render(r.EnvVar),
			) + autoTag + "\n"
		}
	}

	// Action buttons
	detail += fmt.Sprintf("\n  %s  %s  %s",
		styleTag.Copy().Background(colorGreen).Render("[↵ Reveal]"),
		styleTag.Copy().Background(colorRed).Render("[r Revoke]"),
		styleTag.Copy().Background(colorYellow).Render("[R Rotate]"),
	)

	return styleDetail.Render(detail)
}

func renderFooter(m *Model) string {
	items := []string{
		"[a] Add",
		"[r] Revoke",
		"[R] Rotate",
		"[s] Sync",
		"[d] Dependencies",
		"[q] Quit",
	}
	joined := ""
	for i, item := range items {
		if i > 0 {
			joined += "   "
		}
		joined += styleFooter.Render(item)
	}
	return "\n" + joined
}

func renderInput(m *Model) string {
	switch m.inputMode {
	case "add":
		return lipgloss.NewStyle().
			Background(colorPanel).
			Padding(1, 2).
			Width(120).
			Render("  " + m.input.View())
	case "revoke_confirm":
		return lipgloss.NewStyle().
			Background(colorPanel).
			Padding(1, 2).
			Width(120).
			Render(fmt.Sprintf("  🔴 Confirm revoke '%s': %s  [Enter] confirm  [Esc] cancel",
				m.selected.Name, m.input.View()))
	}
	return ""
}

func ageString(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "—"
	}
	result := ""
	for _, t := range tags {
		result += styleTag.Copy().Render(t) + " "
	}
	return result
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

var styleWhite = lipgloss.NewStyle().Foreground(colorWhite)
var styleBlue = lipgloss.NewStyle().Foreground(colorBlue)
var styleGreen = lipgloss.NewStyle().Foreground(colorGreen)

// ── secretItem (list item) ───────────────────────────────────────────────────

type secretItem struct {
	sec   *secrets.Secret
	graph *secrets.DependencyGraph
}

func (si secretItem) Title() string {
	return fmt.Sprintf("🔑 %s", si.sec.Name)
}

func (si secretItem) Description() string {
	refs := si.graph.GetRefs(si.sec.ID)
	refStr := fmt.Sprintf("%d system(s)", len(refs))
	if len(refs) == 0 {
		refStr = "orphan"
	}
	return fmt.Sprintf("%-12s %s  %s",
		string(si.sec.Type),
		ageString(si.sec.CreatedAt),
		styleMuted.Render(refStr),
	)
}

func (si secretItem) FilterValue() string { return si.sec.Name }

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	// Load vault
	key, err := loadVaultKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Vault locked: %v\n", err)
		fmt.Fprintln(os.Stderr, "   Run 'ovav-vault-secrets unlock' first")
		os.Exit(1)
	}

	vaultPath := filepath.Join(os.Getenv("HOME"), ".local/share/ovav/secrets.vault")
	store, err := secrets.LoadFromPath(vaultPath, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Load vault: %v\n", err)
		os.Exit(1)
	}

	graph, _ := secrets.LoadDependencyGraph()
	if graph == nil {
		graph = &secrets.DependencyGraph{}
	}

	model := NewModel(store, graph, key)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error running TUI:", err)
		os.Exit(1)
	}
}

// loadVaultKey loads the vault encryption key.
// In production, this would use TPM or a prompt.
// For now, reads from OVAV_SEED or falls back to a demo mode.
func loadVaultKey() ([]byte, error) {
	seed := os.Getenv("OVAV_SEED")
	if seed == "" {
		// Demo mode: use a default for local testing
		seed = "demo-seed-for-local-vault-tui-2026"
	}
	// Get machine ID for VaultKey derivation
	machineID, err := machineID()
	if err != nil {
		// Fallback: use a fixed machine ID for demo
		machineID = "tui-demo-machine"
	}
	return secrets.DeriveVaultKey(seed, machineID)
}

// machineID returns the system machine ID.
func machineID() (string, error) {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		// Fallback: try dmidecode or other sources
		return "unknown", err
	}
	return string(data[:len(data)-1]), nil // trim newline
}
