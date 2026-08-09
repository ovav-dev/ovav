// Package main — Cockpit Update Checker (GOV-007)
//
// cPanel HTTP client for version checking. Polls cPanel's
// /api/v1/product/version endpoint to detect available updates.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ovav/ovav/internal/product"
)

// DefaultCPanelAddr is the default cPanel server address.
// Deprecated: use product.DefaultCPanelURL directly.
const DefaultCPanelAddr = product.DefaultCPanelURL

// ProductVersionInfo is the response from cPanel's version endpoint.
type ProductVersionInfo struct {
	Current     string `json:"current"`
	Available   string `json:"available,omitempty"`
	UpdateReady bool   `json:"update_ready"`
	Channel     string `json:"channel"`
	CheckedAt   string `json:"checked_at"`
	Error       string `json:"error,omitempty"`
}

// updateCheckMsg is a tea.Msg carrying the version check result.
type updateCheckMsg struct {
	info ProductVersionInfo
	err  error
}

// checkForUpdates returns a tea.Cmd that queries cPanel for available updates.
func checkForUpdates(cpanelURL string) tea.Cmd {
	if cpanelURL == "" {
		cpanelURL = DefaultCPanelAddr
	}

	return func() tea.Msg {
		info := ProductVersionInfo{
			Channel:   "stable",
			CheckedAt: time.Now().UTC().Format(time.RFC3339),
		}

		url := cpanelURL + "/api/v1/product/version"
		client := &http.Client{Timeout: 3 * time.Second}

		resp, err := client.Get(url)
		if err != nil {
			// cPanel not running — use local info
			info.Error = fmt.Sprintf("cPanel not reachable (%v)", err)
			info.Current = "local"
			return updateCheckMsg{info: info, err: nil}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			info.Error = fmt.Sprintf("cPanel returned %d", resp.StatusCode)
			return updateCheckMsg{info: info, err: nil}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			info.Error = fmt.Sprintf("read error: %v", err)
			return updateCheckMsg{info: info, err: nil}
		}

		if err := json.Unmarshal(body, &info); err != nil {
			info.Error = fmt.Sprintf("parse error: %v", err)
			return updateCheckMsg{info: info, err: nil}
		}

		return updateCheckMsg{info: info, err: nil}
	}
}

// triggerUpdateDispatch sends a POST to cPanel to trigger the full update pipeline.
func triggerUpdateDispatch(cpanelURL string) tea.Cmd {
	if cpanelURL == "" {
		cpanelURL = DefaultCPanelAddr
	}

	return func() tea.Msg {
		url := cpanelURL + "/api/v1/product/update"
		client := &http.Client{Timeout: 30 * time.Second}

		resp, err := client.Post(url, "application/json", nil)
		if err != nil {
			return productSyncMsg{err: err, msg: fmt.Sprintf("Dispatch failed: %v", err)}
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		return productSyncMsg{
			err: nil,
			msg: fmt.Sprintf("✅ Update dispatched — %s", string(body)),
		}
	}
}
