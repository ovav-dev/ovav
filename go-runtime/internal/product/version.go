// Package product — OVAV Product version tracking and update detection.
//
// GOV-007: Provides version check against cPanel API so Cockpit can
// poll for available updates and trigger the sync pipeline.
package product

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProductVersion is the current OVAV Product release version.
// Bumped on each release that reaches end users.
const ProductVersion = "1.1.0"

// DefaultCPanelURL is the default cPanel API endpoint for update checks.
const DefaultCPanelURL = "http://localhost:5858"

// UpdateInfo contains the result of a version check against cPanel.
type UpdateInfo struct {
	Current     string `json:"current"`
	Available   string `json:"available,omitempty"`
	UpdateReady bool   `json:"update_ready"`
	Channel     string `json:"channel"`
	CheckedAt   string `json:"checked_at"`
	Cached      bool   `json:"cached"`
}

// VersionInfo reads the locally installed version.
// Falls back to ProductVersion constant if manifest is unavailable.
func VersionInfo() string {
	m, err := LoadManifest()
	if err != nil || m == nil || m.Product == "" {
		return ProductVersion
	}
	return m.Product
}

// CheckForUpdate queries the cPanel API for the latest available version
// and compares it against the locally installed version.
//
// If cpanelURL is empty, DefaultCPanelURL is used.
// If the API is unreachable, uses the manifest's last known version.
func CheckForUpdate(cpanelURL string) (*UpdateInfo, error) {
	if cpanelURL == "" {
		cpanelURL = DefaultCPanelURL
	}

	result := &UpdateInfo{
		Current:   VersionInfo(),
		Channel:   "stable",
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Try to reach cPanel
	available, err := fetchVersionFromCPanel(cpanelURL)
	if err != nil {
		result.Cached = true
		// Fallback: check manifest for last known available
		if m, merr := LoadManifest(); merr == nil && m != nil && m.Product != "" {
			available = m.Product
		}
		if available == result.Current {
			return result, nil
		}
	}

	if available != "" && available != result.Current {
		result.Available = available
		result.UpdateReady = true
	}

	return result, nil
}

// NeedsUpdate returns true if a newer version is available.
func NeedsUpdate() bool {
	info, err := CheckForUpdate("")
	if err != nil {
		return false
	}
	return info.UpdateReady
}

// fetchVersionFromCPanel makes an HTTP GET to cPanel's product version endpoint.
func fetchVersionFromCPanel(cpanelURL string) (string, error) {
	url := strings.TrimRight(cpanelURL, "/") + "/api/v1/product/version"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("cPanel unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cPanel returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read cPanel response: %w", err)
	}

	var info UpdateInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("parse cPanel response: %w", err)
	}

	return info.Available, nil
}

// WriteVersionFile writes the product version to the install directory.
// This is called after a successful install to mark the current version.
func WriteVersionFile() error {
	productDir, err := ProductDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(productDir, 0755); err != nil {
		return fmt.Errorf("mkdir product dir: %w", err)
	}
	vf := filepath.Join(productDir, "VERSION")
	return os.WriteFile(vf, []byte(ProductVersion+"\n"), 0644)
}
