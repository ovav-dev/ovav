package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/vault/secrets"
)

// ── Vault Panel State ──────────────────────────────────────────────────────

type vaultState int

const (
	vaultStateList vaultState = iota
	vaultStateDetail
	vaultStateAdd
	vaultStateRevoke
	vaultStateRotate
	vaultStateExport
	vaultStateImport
)

type vaultSubModel struct {
	secrets    []*secrets.Secret
	selected   int
	state      vaultState
	key        []byte
	store      *secrets.SecretStore
	graph      *secrets.DependencyGraph
	resultMsg  string
	confirmYes bool
	inputName  string
	inputValue string
	inputType  string
	loading    bool
	loadErr    error
}

// vaultUpdate handles keyboard events for the vault panel.
func (m Model) vaultUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vm := m.vaultModel

	switch msg.String() {
	case "q", "esc":
		if vm.state != vaultStateList {
			vm.state = vaultStateList
			vm.confirmYes = false
			vm.resultMsg = ""
			return m, nil
		}
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
		return m, nil

	case "enter":
		return m, nil

	case "up", "k":
		if vm.state == vaultStateList && len(vm.secrets) > 0 {
			vm.selected = max(0, vm.selected-1)
		}

	case "down", "j":
		if vm.state == vaultStateList && len(vm.secrets) > 0 {
			vm.selected = min(len(vm.secrets)-1, vm.selected+1)
		}

	case "a":
		if vm.state == vaultStateList {
			vm.state = vaultStateAdd
			vm.inputName = ""
			vm.inputValue = ""
			vm.inputType = "api_token"
		}

	case "r":
		if vm.state == vaultStateList && len(vm.secrets) > 0 {
			vm.state = vaultStateRevoke
			vm.confirmYes = false
		}

	case "x":
		if vm.state == vaultStateList && len(vm.secrets) > 0 {
			vm.state = vaultStateRotate
			vm.confirmYes = false
		}

	case "e":
		if vm.state == vaultStateList {
			vm.state = vaultStateExport
		}

	case "i":
		if vm.state == vaultStateList {
			vm.state = vaultStateImport
		}

	case "l":
		if vm.state == vaultStateList {
			return m, m.loadVaultSecrets()
		}

	case "y":
		if vm.confirmYes {
			m.vaultModel.confirmYes = false
			m.vaultModel.state = vaultStateList
			if vm.selected < len(vm.secrets) {
				sec := vm.secrets[vm.selected]
				switch vm.state {
				case vaultStateRevoke:
					report, err := secrets.RevokeSecret(vm.store, vm.graph, sec.Name)
					if err != nil {
						vm.resultMsg = fmt.Sprintf("❌ Revoke failed: %v", err)
					} else {
						var b strings.Builder
						for _, r := range report.Results {
							if r.Status == "revoked" {
								b.WriteString(fmt.Sprintf("  ✅ %s: revoked\n", r.Provider))
							} else if r.Status == "failed" {
								b.WriteString(fmt.Sprintf("  ❌ %s: %s\n", r.Provider, r.Error))
							}
						}
						if report.VaultDeleted {
							b.WriteString("  💾 Removed from vault\n")
						}
						vm.resultMsg = b.String()
						vm.secrets = vm.store.List("")
						vm.selected = 0
					}
				case vaultStateRotate:
					if sec.Rotatable {
						report, err := secrets.RotateSecret(vm.store, vm.graph, sec.Name, vm.key)
						if err != nil {
							vm.resultMsg = fmt.Sprintf("❌ Rotate failed: %v", err)
						} else {
							var b strings.Builder
							for _, r := range report.Results {
								if r.Status == "rotated" {
									b.WriteString(fmt.Sprintf("  ✅ %s: rotated\n", r.Provider))
								} else if r.Status == "failed" {
									b.WriteString(fmt.Sprintf("  ❌ %s: %s\n", r.Provider, r.Error))
								}
							}
							if report.VaultUpdated {
								b.WriteString("  💾 Vault updated\n")
							}
							if report.RollbackOccurred {
								b.WriteString("  ⚠️ Rollback occurred\n")
							}
							vm.resultMsg = b.String()
						}
					} else {
						vm.resultMsg = fmt.Sprintf("⚠️ %s is not rotatable", sec.Name)
					}
				}
			}
			m.vaultModel = vm
			return m, nil
		}

	default:
		// Type input in add/detail states
		if vm.state == vaultStateAdd {
			if msg.String() == "backspace" {
				if len(vm.inputName) > 0 {
					vm.inputName = vm.inputName[:len(vm.inputName)-1]
				}
			} else if len(msg.String()) == 1 {
				vm.inputName += msg.String()
			}
		}
	}

	m.vaultModel = vm
	return m, nil
}

// loadVaultSecrets loads secrets from the vault store.
func (m Model) loadVaultSecrets() tea.Cmd {
	return func() tea.Msg {
		// Load vault key from env or file
		key, err := loadVaultKeyForCockpit()
		if err != nil {
			return vaultLoadedMsg{err: err}
		}
		store, graph, err := openSecretsStoreForCockpit(key)
		if err != nil {
			return vaultLoadedMsg{err: err}
		}
		all := store.List("")
		return vaultLoadedMsg{secrets: all, store: store, graph: graph, key: key}
	}
}

type vaultLoadedMsg struct {
	secrets []*secrets.Secret
	store   *secrets.SecretStore
	graph   *secrets.DependencyGraph
	key     []byte
	err     error
}

func (m Model) confirmVaultAction() (tea.Model, tea.Cmd) {
	vm := m.vaultModel
	vm.confirmYes = true

	switch vm.state {
	case vaultStateRevoke:
		if vm.selected < len(vm.secrets) {
			sec := vm.secrets[vm.selected]
			report, err := secrets.RevokeSecret(vm.store, vm.graph, sec.Name)
			if err != nil {
				vm.resultMsg = fmt.Sprintf("❌ Revoke failed: %v", err)
			} else {
				var b strings.Builder
				for _, r := range report.Results {
					if r.Status == "revoked" {
						b.WriteString(fmt.Sprintf("  ✅ %s: revoked\n", r.Provider))
					} else if r.Status == "failed" {
						b.WriteString(fmt.Sprintf("  ❌ %s: %s\n", r.Provider, r.Error))
					}
				}
				if report.VaultDeleted {
					b.WriteString("  💾 Removed from vault\n")
				}
				vm.resultMsg = b.String()
				// Reload secrets
				vm.secrets = vm.store.List("")
				vm.selected = 0
			}
		}
		vm.state = vaultStateList

	case vaultStateRotate:
		if vm.selected < len(vm.secrets) {
			sec := vm.secrets[vm.selected]
			if !sec.Rotatable {
				vm.resultMsg = fmt.Sprintf("⚠️ %s is not rotatable — no provider refs", sec.Name)
				vm.state = vaultStateList
			} else {
				report, err := secrets.RotateSecret(vm.store, vm.graph, sec.Name, vm.key)
				if err != nil {
					vm.resultMsg = fmt.Sprintf("❌ Rotate failed: %v", err)
				} else {
					var b strings.Builder
					for _, r := range report.Results {
						if r.Status == "rotated" {
							b.WriteString(fmt.Sprintf("  ✅ %s: rotated\n", r.Provider))
						} else if r.Status == "failed" {
							b.WriteString(fmt.Sprintf("  ❌ %s: %s\n", r.Provider, r.Error))
						}
					}
					if report.VaultUpdated {
						b.WriteString("  💾 Vault updated\n")
					}
					if report.RollbackOccurred {
						b.WriteString("  ⚠️ Rollback occurred\n")
					}
					vm.resultMsg = b.String()
				}
			}
			vm.state = vaultStateList
		}
	}

	m.vaultModel = vm
	return m, nil
}

// renderVault renders the vault panel.
func (m Model) renderVault() string {
	vm := m.vaultModel

	header := renderTitleBar("🔐  OVAV Vault — Intelligent Secrets Manager")
	help := styles.MutedFg.Render("  [a] add  [r] revoke  [x] rotate  [e] export  [i] import  [l] reload  [q] back  [?] help")

	if vm.loading {
		return header + "\n\n  ⏳ Loading vault...\n\n" + help
	}

	if vm.loadErr != nil {
		return header + fmt.Sprintf("\n\n  ❌ %v\n\n  %s\n\n%s\n", vm.loadErr, styles.MutedFg.Render("  Vault key required — run 'ovav login' first"), help)
	}

	if vm.resultMsg != "" {
		return header + fmt.Sprintf("\n\n%s\n\n%s\n", vm.resultMsg, help)
	}

	switch vm.state {
	case vaultStateList:
		return m.renderVaultList()
	case vaultStateDetail:
		return m.renderVaultDetail()
	case vaultStateAdd:
		return m.renderVaultAdd()
	case vaultStateRevoke:
		return m.renderVaultRevoke()
	case vaultStateRotate:
		return m.renderVaultRotate()
	case vaultStateExport:
		return m.renderVaultExport()
	case vaultStateImport:
		return m.renderVaultImport()
	}

	return header + "\n\n  Use keys to navigate.\n\n" + help
}

func (m Model) renderVaultList() string {
	vm := m.vaultModel
	header := renderTitleBar("🔐  OVAV Vault")
	nav := styles.MutedFg.Render("  ← back to menu")
	help := styles.MutedFg.Render("  [a] add  [r] revoke  [x] rotate  [↑↓] select  [q] back")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	if len(vm.secrets) == 0 {
		sb.WriteString(styles.MutedFg.Render("  Vault is empty — press [a] to add a secret\n"))
		sb.WriteString(help)
		return sb.String()
	}

	// Stats bar
	var rotatable, expiring int
	now := time.Now()
	for _, sec := range vm.secrets {
		if sec.Rotatable {
			rotatable++
		}
		if !sec.ExpiresAt.IsZero() && sec.ExpiresAt.Before(now.Add(7*24*time.Hour)) {
			expiring++
		}
	}
	stats := fmt.Sprintf("  %d secrets | %d rotatable | %d expiring soon", len(vm.secrets), rotatable, expiring)
	sb.WriteString(styles.TagBlue.Render(stats))
	sb.WriteString("\n\n")

	// Secret list
	for i, sec := range vm.secrets {
		selected := i == vm.selected
		icon := "🔑"
		if sec.Type == secrets.TypeCloudKey {
			icon = "☁️"
		} else if sec.Type == secrets.TypeOAuthCreds {
			icon = "🔐"
		} else if sec.Type == secrets.TypeDBCredential {
			icon = "🗄️"
		} else if sec.Type == secrets.TypeTunnelToken {
			icon = "🌐"
		}

		tagStr := ""
		if len(sec.Tags) > 0 {
			tagStr = " [" + strings.Join(sec.Tags, ", ") + "]"
		}
		rotatable := ""
		if sec.Rotatable {
			rotatable = " 🔄"
		}
		refs := vm.graph.GetRefs(sec.ID)
		orphan := ""
		if len(refs) == 0 {
			orphan = " ⚠️ orphan"
		}

		nameStyle := lipgloss.NewStyle()
		if selected {
			nameStyle = styles.Selected
		}
		line := fmt.Sprintf("  %s %-32s %s%s%s%s", icon, nameStyle.Render(sec.Name), sec.Type, tagStr, rotatable, orphan)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("  [a] add  [r] revoke  [x] rotate  [e] export  [i] import  [↑↓] select  [l] reload"))
	return sb.String()
}

func (m Model) renderVaultDetail() string {
	vm := m.vaultModel
	header := renderTitleBar("🔐  Secret Detail")
	nav := styles.MutedFg.Render("  ← back (esc)")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	if vm.selected >= len(vm.secrets) {
		return sb.String()
	}
	sec := vm.secrets[vm.selected]

	age := time.Since(sec.CreatedAt)
	hash := sec.Hash
	if len(hash) > 16 {
		hash = hash[:16] + "..."
	}

	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Name:", sec.Name))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Type:", sec.Type))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Provider:", sec.Provider))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Source:", sec.Source))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Hash:", hash))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Created:", sec.CreatedAt.Format("2006-01-02 15:04 MST")))
	sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Age:", age.Round(time.Hour).String()))
	if !sec.ExpiresAt.IsZero() {
		sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Expires:", sec.ExpiresAt.Format("2006-01-02 15:04 MST")))
	}
	if !sec.LastUsed.IsZero() {
		sb.WriteString(fmt.Sprintf("  %-12s %s\n", "Last used:", sec.LastUsed.Format("2006-01-02 15:04 MST")))
	}
	if sec.Rotatable {
		sb.WriteString(fmt.Sprintf("  %-12s ✅ yes\n", "Rotatable:"))
	}
	if len(sec.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("  %-12s %v\n", "Tags:", sec.Tags))
	}

	refs := vm.graph.GetRefs(sec.ID)
	if len(refs) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %d provider reference(s):\n", len(refs)))
		for _, ref := range refs {
			sb.WriteString(fmt.Sprintf("  • %s: %s (%s)\n", ref.System, ref.Path, ref.EnvVar))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("  Value hidden — use 'ovav vault secrets get' in CLI to reveal"))
	return sb.String()
}

func (m Model) renderVaultAdd() string {
	vm := m.vaultModel
	header := renderTitleBar("🔐  Add Secret")
	nav := styles.MutedFg.Render("  ← cancel (esc)  [enter] confirm")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("  Name:  %s\n", lipgloss.NewStyle().Width(40).Render(vm.inputName)))
	sb.WriteString(fmt.Sprintf("  Type:  %s\n", vm.inputType))
	sb.WriteString("\n")
	sb.WriteString(styles.MutedFg.Render("  Type a secret name. Use [tab] to cycle types.\n"))
	sb.WriteString(styles.MutedFg.Render("  Value will be prompted in CLI with 'ovav vault secrets add'\n"))

	return sb.String()
}

func (m Model) renderVaultRevoke() string {
	vm := m.vaultModel
	header := renderTitleBar("⚠️  Revoke Secret")
	nav := styles.MutedFg.Render("  ← cancel (esc)")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	if vm.selected >= len(vm.secrets) {
		return sb.String()
	}
	sec := vm.secrets[vm.selected]
	refs := vm.graph.GetRefs(sec.ID)

	sb.WriteString(fmt.Sprintf("  Revoke %q?\n\n", sec.Name))
	if len(refs) == 0 {
		sb.WriteString(styles.MutedFg.Render("  No provider refs — will remove from vault only.\n"))
	} else {
		sb.WriteString(fmt.Sprintf("  Will revoke from %d provider(s):\n", len(refs)))
		for _, ref := range refs {
			sb.WriteString(fmt.Sprintf("  • %s: %s\n", ref.System, ref.Path))
		}
	}
	sb.WriteString("\n")
	sb.WriteString(styles.WarningBadge.Render("  ⚠️  This cannot be undone. Press [y] to confirm.\n"))

	return sb.String()
}

func (m Model) renderVaultRotate() string {
	vm := m.vaultModel
	header := renderTitleBar("🔄  Rotate Secret")
	nav := styles.MutedFg.Render("  ← cancel (esc)")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	if vm.selected >= len(vm.secrets) {
		return sb.String()
	}
	sec := vm.secrets[vm.selected]

	if !sec.Rotatable {
		sb.WriteString(fmt.Sprintf("  %q is not rotatable.\n\n", sec.Name))
		sb.WriteString(styles.MutedFg.Render("  Track it in the dependency graph first with 'ovav vault deps track'\n"))
		return sb.String()
	}

	refs := vm.graph.GetRefs(sec.ID)
	sb.WriteString(fmt.Sprintf("  Rotate %q?\n\n", sec.Name))
	sb.WriteString(fmt.Sprintf("  Will push new value to %d provider(s):\n", len(refs)))
	for _, ref := range refs {
		sb.WriteString(fmt.Sprintf("  • %s: %s\n", ref.System, ref.Path))
	}
	sb.WriteString("\n")
	sb.WriteString(styles.WarningBadge.Render("  ⚠️  Press [y] to confirm rotation.\n"))

	return sb.String()
}

func (m Model) renderVaultExport() string {
	header := renderTitleBar("📦  Export Vault")
	nav := styles.MutedFg.Render("  ← back (esc)")

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	sb.WriteString(styles.MutedFg.Render("  Use the CLI to export:\n\n"))
	codeStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	sb.WriteString(fmt.Sprintf("  %s\n", codeStyle.Render("ovav vault secrets export backup.airgap --expires 30d")))
	sb.WriteString("\n")
	sb.WriteString("  Creates an encrypted .airgap package:\n")
	sb.WriteString("  • AES-256-GCM encrypted\n")
	sb.WriteString("  • HMAC-SHA256 authenticated\n")
	sb.WriteString("  • Optional password + expiration\n")
	sb.WriteString("  • Self-contained — no network needed to restore\n")

	return sb.String()
}

func (m Model) renderVaultImport() string {
	header := renderTitleBar("📥  Import Vault")
	nav := styles.MutedFg.Render("  ← back (esc)")
	codeStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(nav)
	sb.WriteString("\n\n")

	sb.WriteString(styles.MutedFg.Render("  Use the CLI to import:\n\n"))
	sb.WriteString(fmt.Sprintf("  %s\n", codeStyle.Render("ovav vault secrets import backup.airgap")))
	sb.WriteString("\n")
	sb.WriteString("  Merges secrets from an .airgap file into the vault.\n")
	sb.WriteString("  • Expired packages are rejected\n")
	sb.WriteString("  • Duplicate secrets are skipped\n")
	sb.WriteString("  • HMAC verified before merge\n")

	return sb.String()
}

// loadVaultKeyForCockpit loads the vault key for cockpit use.
func loadVaultKeyForCockpit() ([]byte, error) {
	// Try OVAV_VAULT_KEY env
	hexKey := os.Getenv("OVAV_VAULT_KEY")
	if hexKey != "" {
		return hex.DecodeString(hexKey)
	}
	// Fallback: vault_key_export from login
	home, _ := os.UserHomeDir()
	exportPath := filepath.Join(home, ".local/share/ovav/vault_key_export")
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("vault key not found: run 'ovav login' first")
	}
	return hex.DecodeString(strings.TrimSpace(string(data)))
}

func openSecretsStoreForCockpit(key []byte) (*secrets.SecretStore, *secrets.DependencyGraph, error) {
	store, err := secrets.Load(key)
	if err != nil {
		return nil, nil, fmt.Errorf("open vault: %w", err)
	}
	graph, _ := secrets.LoadDependencyGraph()
	if graph == nil {
		graph = &secrets.DependencyGraph{}
	}
	return store, graph, nil
}
