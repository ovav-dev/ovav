package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WeztermPathIntegrity enforces single canonical WezTerm path architecture.
// Replaces: check_wezterm_path_integrity.py
type WeztermPathIntegrity struct{}

func NewWeztermPathIntegrity() *WeztermPathIntegrity { return &WeztermPathIntegrity{} }

func (w *WeztermPathIntegrity) ID() string   { return "wezterm_path_integrity" }
func (w *WeztermPathIntegrity) Name() string { return "WezTerm Path Integrity Validator" }
func (w *WeztermPathIntegrity) Description() string {
	return "Enforces single canonical WezTerm path — no duplicates, proxy markers present, blocked paths clean"
}
func (w *WeztermPathIntegrity) Weight() int { return 7 }

// Canonical paths
const (
	canonicalWSLPath     = "/home/braka/.config/wezterm/wezterm.lua"
	canonicalWindowsUser = "%USERPROFILE%\\.wezterm.lua"
	wslDistro            = "Ubuntu-24.04"
)

// Blocked/deprecated paths
var blockedPaths = []string{
	"C:\\Users\\Alexa\\AppData\\Roaming\\wezterm\\.wezterm.lua",
	"%APPDATA%\\wezterm\\.wezterm.lua",
	"/mnt/c/Users/Alexa/AppData/Roaming/wezterm/.wezterm.lua",
	"~/..ovav/source/configs/wezterm/wezterm.lua",
	"/mnt/c/Users/<user>/.config/wezterm/wezterm.lua",
}

// Proxy marker requirements
var proxyMarkers = map[string][]string{
	".ovav/source/configs/wezterm/ovav-windows-loader.wezterm.lua": {
		"OVAV_WZPROXY_v3", "OVAV_PROXY_MARKER", "OVAV_CANONICAL_PATH_WSL",
		"OVAV_CANONICAL_UNC", "OVAV_WSL_DISTRO", "OVAV_FALLBACK_PATH",
	},
	"config/wezterm/wezterm-fallback-minimal.lua": {
		"OVAV_WZFALLBACK_v1", "OVAV_FALLBACK_MARKER",
	},
}

// Scan files for path references
var weztermScanFiles = []string{
	"docs/workstation/OVAV_WEZTERM_WORKSPACE_ISOLATION.md",
	"config/workstation/ovav-wezterm-workspace-isolation.yaml",
	".ovav/source/configs/workstation/ovav-wezterm-workspace-isolation.yaml",
	".ovav/registry/work_ledger.yaml",
	".ovav/registry/tool_configs.yaml",
	"tools/workstation/ovav_wezterm_workspace.py",
	"tools/validators/check_ovav_wezterm_workspace_isolation.py",
	"tools/harnesses/deployment_claim_audit.py",
}

var (
	docBlockedMarkers = []string{
		"blocked_path", "blocked_note", "windows_blocked", "wezterm_blocked",
		"# blocked", "eliminado", "eliminar", "eliminated", "removed", "removido",
		"must be removed", "must not exist", "deprecated", "bloqueado", "bloquea",
		"redundante", "blindado",
	}
)

func (w *WeztermPathIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}

	// 1. Check proxy marker files
	for relPath, markers := range proxyMarkers {
		fullPath := filepath.Join(root, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("MISSING_PROXY: %s", relPath))
			continue
		}
		content := readFile(relPath)
		if content == "" {
			issues = append(issues, fmt.Sprintf("UNREADABLE_PROXY: %s", relPath))
			continue
		}
		for _, marker := range markers {
			if !strings.Contains(content, marker) {
				issues = append(issues, fmt.Sprintf("MISSING_MARKER: %s missing '%s'", relPath, marker))
			}
		}
	}

	// 2. Check fallback file integrity
	fallbackPath := filepath.Join(root, "config/wezterm/wezterm-fallback-minimal.lua")
	if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING_FALLBACK: config/wezterm/wezterm-fallback-minimal.lua")
	} else {
		fallbackContent := readFile("config/wezterm/wezterm-fallback-minimal.lua")
		fallbackTokens := []string{
			"OVAV_FALLBACK_MARKER", "require 'wezterm'", "OVAV_WZFALLBACK_v1",
		}
		for _, token := range fallbackTokens {
			if !strings.Contains(fallbackContent, token) {
				issues = append(issues, fmt.Sprintf("FALLBACK_MISSING_TOKEN: %s", token))
			}
		}
	}

	// 3. Scan files for blocked paths
	for _, scanFile := range weztermScanFiles {
		content := readFile(scanFile)
		if content == "" {
			continue
		}
		for _, blocked := range blockedPaths {
			if !strings.Contains(content, blocked) {
				continue
			}
			// Check if this is documentation of the blocked path
			lines := strings.Split(content, "\n")
			isDocContext := false
			for _, line := range lines {
				if !strings.Contains(line, blocked) {
					continue
				}
				lower := strings.ToLower(strings.TrimSpace(line))
				docContext := false
				for _, dm := range docBlockedMarkers {
					if strings.Contains(lower, dm) {
						docContext = true
						break
					}
				}
				if !docContext {
					isDocContext = true
				}
			}
			if isDocContext {
				issues = append(issues, fmt.Sprintf("BLOCKED_PATH: %s references '%s'", scanFile, blocked))
			}
		}
	}

	// 4. Check work_ledger consistency
	ledgerContent := readFile(".ovav/registry/work_ledger.yaml")
	if strings.Contains(ledgerContent, "wezterm_windows_path:") {
		idx := strings.Index(ledgerContent, "wezterm_windows_path:")
		rest := ledgerContent[idx:]
		if nl := strings.Index(rest, "\n"); nl > 0 {
			line := rest[:nl]
			if strings.Contains(line, "AppData") {
				issues = append(issues, "LEDGER: work_ledger.yaml wezterm_windows_path references deprecated APPDATA path")
			}
		}
	}

	if len(issues) > 0 {
		return Result{ID: w.ID(), Name: w.Name(), Status: "fail", Weight: w.Weight(),
			Message:  fmt.Sprintf("FAIL — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start)}
	}
	return Result{ID: w.ID(), Name: w.Name(), Status: "pass", Weight: w.Weight(),
		Message:  fmt.Sprintf("PASS — WezTerm path integrity valid (canonical: %s)", canonicalWSLPath),
		Duration: time.Since(start)}
}

var _ Validator = (*WeztermPathIntegrity)(nil)
