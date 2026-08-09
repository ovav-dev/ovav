package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/internal/product"
)

// ── Messages ───────────────────────────────────────────────────────

type versionMsg struct {
	info product.UpdateInfo
	err  error
}

type updateDispatchMsg struct {
	ok  bool
	msg string
	err error
}

type sseEventMsg struct {
	event string
	data  map[string]interface{}
	err   error
}

type pollTickMsg struct{}

// syncQueueMsg carries sync queue status from cPanel.
type syncQueueMsg struct {
	itemsQueued int
	err         error
}

// ── Commands ───────────────────────────────────────────────────────

func fetchVersionCmd(cpanelURL string) tea.Cmd {
	return func() tea.Msg {
		info, err := product.CheckForUpdate(cpanelURL)
		if err != nil {
			return versionMsg{info: product.UpdateInfo{}, err: err}
		}
		return versionMsg{info: *info, err: nil}
	}
}

func dispatchUpdateCmd(cpanelURL string) tea.Cmd {
	return func() tea.Msg {
		// Pre-flight: clean stale installation if detected
		if dir, err := product.ProductDir(); err == nil {
			if m, _ := product.LoadManifest(); m != nil && m.Entries != nil {
				missing := 0
				for _, e := range m.Entries {
					if _, err := os.Stat(e.Target); os.IsNotExist(err) {
						missing++
					}
				}
				if missing > 0 || len(m.Entries) == 0 {
					// Stale/incomplete — clean before applying update
					product.ProductInstall(".", "uninstall")
				}
			}
			_ = dir
		}

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Post(cpanelURL+"/api/v1/product/update", "application/json", nil)
		if err != nil {
			return updateDispatchMsg{ok: false, msg: "", err: fmt.Errorf("cPanel unreachable: %w", err)}
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}
		json.Unmarshal(body, &result)

		ok := resp.StatusCode == 200 || resp.StatusCode == 207
		msg := result.Message
		if result.Status == "partial" {
			msg = "⚠️ Partial: " + msg + " — retry or run 'ovav product install' manually"
		}
		return updateDispatchMsg{ok: ok, msg: msg, err: nil}
	}
}

// launchBootstrapCmd runs ovav product launch in the current directory.
func launchBootstrapCmd() tea.Cmd {
	return func() tea.Msg {
		if err := product.BootstrapCWD(); err != nil {
			return updateDispatchMsg{ok: false, msg: "", err: fmt.Errorf("bootstrap: %w", err)}
		}
		return updateDispatchMsg{ok: true, msg: "CWD bootstrapped with OVAV agents", err: nil}
	}
}

// pollCmd returns a tick that fires every 30 seconds for auto-refresh.
func pollCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

// fetchSyncQueueCmd queries cPanel for the current sync queue status.
// GOV-009: Product polls cPanel sync queue for pending updates.
func fetchSyncQueueCmd(cpanelURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(cpanelURL + "/api/v1/product/sync/status")
		if err != nil {
			return syncQueueMsg{itemsQueued: -1, err: err}
		}
		defer resp.Body.Close()

		var status struct {
			Queue struct {
				TotalItems int `json:"total_items"`
				Pending    int `json:"pending"`
			} `json:"queue"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return syncQueueMsg{itemsQueued: -1, err: err}
		}

		return syncQueueMsg{itemsQueued: status.Queue.Pending, err: nil}
	}
}
