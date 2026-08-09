package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ovav/ovav/cmd/cockpit/styles"
)

// ── Math ───────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ── Rendering ──────────────────────────────────────────────────────

func renderTitleBar(title string) string {
	w := lipgloss.Width
	_ = w
	bar := styles.TitleBar.Render(fmt.Sprintf(" OVAV Cockpit  —  %s ", title))
	return bar + "\n\n"
}

func renderHelpBar(items string) string {
	return styles.Help.Render(items) + "\n"
}

func styleHelp(s string) string {
	return styles.MutedFg.Render(s)
}

func renderPctBar(pct, width int) string {
	if width <= 0 {
		width = 10
	}
	filled := pct * width / 100
	empty := width - filled

	fill := styles.ProgressFill.Render(strings.Repeat("█", filled))
	empt := styles.ProgressEmpty.Render(strings.Repeat("░", empty))
	return fill + empt
}

// ── View IDs ───────────────────────────────────────────────────────

const (
	ViewWelcome   = "welcome"
	ViewRoot      = "root"
	ViewDashboard = "dashboard"
	ViewHealth    = "health"
	ViewVault     = "vault"
	ViewInstall   = "install"
	ViewTailor    = "tailor"
	ViewDetail    = "detail"
	ViewCLI       = "cli_selector"
	ViewSync      = "sync"
	ViewConfig    = "config"
	ViewUpdates   = "updates"
	ViewQuit      = "quit"
	ViewHelp      = "help"
)

// ── OVAV Root Detection ─────────────────────────────────────────────

var (
	ovavRootCache    string
	ovavRootCacheSet bool
	maxFindDepth     = 50 // max directories to traverse upward
)

func findOVAVRoot() string {
	// Fast path: env var override
	if root := os.Getenv("OVAV_ROOT"); root != "" {
		return root
	}
	// Cache hit
	if ovavRootCacheSet {
		return ovavRootCache
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	for dir, depth := cwd, 0; dir != "/" && dir != "." && depth < maxFindDepth; dir, depth = filepath.Dir(dir), depth+1 {
		capsPath := filepath.Join(dir, ".ovav", "plan", "caps.yaml")
		if _, err := os.Stat(capsPath); err == nil {
			ovavRootCache = dir
			ovavRootCacheSet = true
			return dir
		}
	}
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		capsPath := filepath.Join(dir, ".ovav", "plan", "caps.yaml")
		if _, err := os.Stat(capsPath); err == nil {
			ovavRootCache = dir
			ovavRootCacheSet = true
			return dir
		}
	}
	ovavRootCache = cwd
	ovavRootCacheSet = true
	return cwd
}
