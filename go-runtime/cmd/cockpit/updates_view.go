package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/cmd/cockpit/styles"
	"github.com/ovav/ovav/internal/product"
)

// ── Updates View — Work Done & Sync to Product ─────────────────────

type UpdateItem struct {
	Title       string
	Description string
	Status      string // "done" | "pending" | "synced"
	Category    string // "product" | "system" | "config"
}

type UpdatesModel struct {
	items      []UpdateItem
	cursor     int
	syncStatus string // "idle" | "syncing" | "done" | "error"
	syncMsg    string
}

func NewUpdatesModel() UpdatesModel {
	um := UpdatesModel{
		syncStatus: "idle",
	}
	// Start with phase 2 items, load sync status async to avoid blocking render
	um.items = gatherPhase2Items()
	return um
}

// gatherPhase2Items returns the pending Phase 2 items (always present, no network).
func gatherPhase2Items() []UpdateItem {
	return []UpdateItem{
		{
			Title:       "Cross-platform Distribution",
			Description: "Windows, macOS, Linux installer one-click",
			Status:      "pending",
			Category:    "product",
		},
		{
			Title:       "OAuth Connect & Memberships",
			Description: "Login Google/GitHub, planes Free/Pro/Enterprise, multi-máquina",
			Status:      "pending",
			Category:    "product",
		},
		{
			Title:       "F6 Cockpit TUI (Product)",
			Description: "Terminal UI completa para usuarios OVAV Product",
			Status:      "pending",
			Category:    "product",
		},
	}
}

// fetchSyncItemsCmd returns a tea.Cmd that loads sync engine status asynchronously.
// GOV-009: Non-blocking — avoids flickering during initial render.
func fetchSyncItemsCmd() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(product.DefaultCPanelURL + "/api/v1/product/sync/status")
		if err != nil {
			return syncItemsMsg{items: nil}
		}
		defer resp.Body.Close()

		var status struct {
			Detected struct {
				TotalItems int `json:"total_items"`
				Synced     int `json:"synced"`
				Pending    int `json:"pending"`
				Items      []struct {
					ID       string `json:"id"`
					Path     string `json:"path"`
					Category string `json:"category"`
				} `json:"items"`
			} `json:"detected"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return syncItemsMsg{items: nil}
		}

		items := []UpdateItem{{
			Title:       fmt.Sprintf("Sync Engine: %d items ready", status.Detected.Pending),
			Description: fmt.Sprintf("%d agents, skills, configs, CLI tools", status.Detected.TotalItems),
			Status:      "synced",
			Category:    "product",
		}}

		cats := map[string]int{}
		for _, item := range status.Detected.Items {
			cats[item.Category]++
		}
		for cat, count := range cats {
			items = append(items, UpdateItem{
				Title:       fmt.Sprintf("%s: %d files", cat, count),
				Description: fmt.Sprintf("%d %s files for OVAV Product", count, cat),
				Status:      "synced",
				Category:    "config",
			})
		}
		return syncItemsMsg{items: items}
	}
}

type syncItemsMsg struct{ items []UpdateItem }

func (m Model) updatesUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	um := &m.updatesModel

	switch msg.String() {
	case "up", "k":
		um.cursor = max(0, um.cursor-1)

	case "down", "j":
		um.cursor = min(len(um.items)-1, um.cursor+1)

	case "c":
		// GOV-007: Check for updates from cPanel
		um.syncStatus = "syncing"
		um.syncMsg = "Checking cPanel..."
		return m, tea.Batch(
			fetchSyncItemsCmd(),
			checkForUpdates(DefaultCPanelAddr),
		)

	case "enter":
		// GOV-007: Trigger full update dispatch via cPanel
		if um.syncStatus == "syncing" {
			return m, nil
		}
		um.syncStatus = "syncing"
		um.syncMsg = "Dispatching update via cPanel → OVAV Product..."
		return m, triggerUpdateDispatch(DefaultCPanelAddr)

	case "s":
		// Sync to Product — reinstall with fresh agents
		um.syncStatus = "syncing"
		um.syncMsg = "Projecting + reinstalling OVAV Product..."
		return m, m.runProductSync()

	case "r":
		um.items = gatherPhase2Items()
		um.syncStatus = "idle"

	case "q", "esc":
		if m.nav.CanGoBack() {
			m.nav.Pop()
		}
	}

	return m, nil
}

type productSyncMsg struct {
	err error
	msg string
}

func (m Model) runProductSync() tea.Cmd {
	return func() tea.Msg {
		root := m.ovavRoot
		_ = root

		// Run product reinstall with new code
		productDir, err := product.ProductDir()
		if err != nil {
			return productSyncMsg{err: err, msg: "Failed to find product dir"}
		}

		// Install with fresh agents
		result, err := product.ProductInstall(root, "install")
		if err != nil {
			return productSyncMsg{err: err, msg: "Install failed"}
		}

		msg := fmt.Sprintf("✅ Product updated: %d files, %d symlinks at %s",
			result.FilesCopied, result.LinksCreated, productDir)
		return productSyncMsg{err: nil, msg: msg}
	}
}

func (m Model) renderUpdates() string {
	var sb strings.Builder
	um := m.updatesModel

	sb.WriteString(renderTitleBar("Work Done & Updates"))
	sb.WriteString("\n")

	// ── Summary Bar ──
	done := 0
	pending := 0
	for _, item := range um.items {
		if item.Status == "done" {
			done++
		} else {
			pending++
		}
	}

	summary := fmt.Sprintf("  %s  %s  %s  %s  %s",
		styles.TagGreen.Render(fmt.Sprintf(" %d Ready ", done)),
		styles.TagYellow.Render(fmt.Sprintf(" %d Pending ", pending)),
		styles.MutedFg.Render("Press s to sync"),
		styles.ActionBtn.Render(" [ s ] Sync "),
		styles.ActionBtn.Render(" [ c ] Check cPanel "),
	)
	sb.WriteString(summary)
	sb.WriteString("\n")

	// ── cPanel Version Info (GOV-007) ──
	if m.updateInfo.UpdateReady {
		sb.WriteString(styles.YellowBorderCompact.Width(m.width - 4).Render(
			fmt.Sprintf("  🔔 Update available! %s → %s  (Checked: %s)",
				m.updateInfo.Current, m.updateInfo.Available, m.updateInfo.CheckedAt)))
		sb.WriteString("\n")
	} else if m.updateInfo.Current != "" {
		sb.WriteString(styles.MutedFg.Render(
			fmt.Sprintf("  ✓ OVAV Product %s — up to date  (channel: %s)", m.updateInfo.Current, m.updateInfo.Channel)))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// ── Sync Status ──
	switch um.syncStatus {
	case "syncing":
		sb.WriteString(styles.YellowBorderCompact.
			Width(m.width - 4).
			Render("  ⏳ " + um.syncMsg))
		sb.WriteString("\n\n")
	case "done":
		sb.WriteString(styles.GreenBorderCompact.
			Width(m.width - 4).
			Render("  ✅ " + um.syncMsg))
		sb.WriteString("\n\n")
	case "error":
		sb.WriteString(styles.ErrorBadge.
			Render("  ❌ " + um.syncMsg))
		sb.WriteString("\n\n")
	}

	// ── Items ──
	categories := []string{"product", "system", "config"}
	categoryLabels := map[string]string{
		"product": "📦 OVAV Product",
		"system":  "⚙️ OVAV Systems",
		"config":  "🔧 Configuration",
	}

	for _, cat := range categories {
		catItems := filterByCategory(um.items, cat)
		if len(catItems) == 0 {
			continue
		}

		sb.WriteString(styles.PurpleCategory.Render(categoryLabels[cat]))
		sb.WriteString("\n")

		globalIdx := categoryStartIndex(um.items, cat)
		for _, item := range catItems {
			isSelected := globalIdx == um.cursor
			line := m.renderUpdateItem(item, isSelected)
			sb.WriteString(line)
			sb.WriteString("\n")
			globalIdx++
		}
		sb.WriteString("\n")
	}

	// ── Time ──
	sb.WriteString(styles.MutedFg.Render(fmt.Sprintf("  Generated: %s", time.Now().Format("2006-01-02 15:04"))))
	sb.WriteString("\n")
	sb.WriteString(renderHelpBar("s/c: Sync/Check  •  Enter: Dispatch  •  ↑↓: Navigate  •  r: Refresh  •  Esc: Back"))

	return sb.String()
}

func (m Model) renderUpdateItem(item UpdateItem, isSelected bool) string {
	icon := "  "
	itemStyle := styles.Unselected

	if isSelected {
		icon = "▸ "
		itemStyle = styles.Selected
	}

	// Status indicator
	statusIcon := "✅"
	statusStyle := styles.GreenFg
	if item.Status == "pending" {
		statusIcon = "⬜"
		statusStyle = styles.YellowFg
	} else if item.Status == "synced" {
		statusIcon = "📡"
		statusStyle = styles.BlueFg
	}

	line := fmt.Sprintf("%s%s  %s  %s",
		icon,
		statusIcon,
		styles.BoldWhite.Render(item.Title),
		statusStyle.Render(item.Status))

	if isSelected {
		line += fmt.Sprintf("\n     %s", styles.MutedItalic.Render(item.Description))
	}

	return itemStyle.Render(line)
}

func filterByCategory(items []UpdateItem, cat string) []UpdateItem {
	var result []UpdateItem
	for _, item := range items {
		if item.Category == cat {
			result = append(result, item)
		}
	}
	return result
}

func categoryStartIndex(items []UpdateItem, cat string) int {
	for i, item := range items {
		if item.Category == cat {
			return i
		}
	}
	return 0
}
