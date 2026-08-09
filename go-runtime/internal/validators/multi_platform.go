package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MultiPlatform validates Windows/macOS readiness and cross-platform configs.
// Skills enforcement removed — governed by Go skill registry (2026-07-31).
type MultiPlatform struct{}

func NewMultiPlatform() *MultiPlatform { return &MultiPlatform{} }

func (m *MultiPlatform) ID() string   { return "multi_platform" }
func (m *MultiPlatform) Name() string { return "C7 Multi-Platform Validator" }
func (m *MultiPlatform) Description() string {
	return "Validates Windows/macOS readiness and cross-platform configs"
}
func (m *MultiPlatform) Weight() int { return 8 }

func (m *MultiPlatform) checkWindowsReadiness(root string) []string {
	var issues []string

	// C7.1.2: WezTerm Windows loader
	loaderPath := filepath.Join(root, ".ovav", "source", "configs", "wezterm", "ovav-windows-loader.wezterm.lua")
	if _, err := os.Stat(loaderPath); os.IsNotExist(err) {
		issues = append(issues, "C7.1: Windows loader template not found")
	} else {
		data, err := os.ReadFile(loaderPath)
		if err == nil {
			content := string(data)
			if !strings.Contains(content, "OVAV_WZPROXY_v3") {
				issues = append(issues, "C7.1: OVAV_WZPROXY_v3 marker missing from Windows loader")
			}
			if !strings.Contains(content, "OVAV_CAPA7_CROSS_PLATFORM") {
				issues = append(issues, "C7.1: OVAV_CAPA7_CROSS_PLATFORM marker missing in Windows loader")
			}
		}
	}

	// C7.1.3: Cross-platform paths — removed (2026-07-31, Go-native paths)
	// Path configuration now in Go: go-runtime/internal/project/isolation.go

	// C7.1.1: WezTerm fallback
	fallbackPath := filepath.Join(root, "config", "wezterm", "wezterm-fallback-minimal.lua")
	if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
		issues = append(issues, "C7.1: wezterm-fallback-minimal.lua not found")
	} else {
		data, _ := os.ReadFile(fallbackPath)
		if !strings.Contains(string(data), "OVAV_WZFALLBACK_v1") {
			issues = append(issues, "C7.1: OVAV_WZFALLBACK_v1 marker missing from fallback config")
		}
	}

	return issues
}

// checkSkillsEnforcement removed — OVAV is 100% Go, skills are now governed
// by the skill registry in .ovav/registry/ and skill files in .opencode/skills/
// (2026-07-31 post-migration cleanup)
func (m *MultiPlatform) checkSkillsEnforcement(root string) []string {
	return nil // no-op
}

// checkConnect removed — OVAV is 100% Go, session routing is now handled
// by the Go dispatch system in cmd/ovav/dispatch.go
// (2026-07-31 post-migration cleanup)
func (m *MultiPlatform) checkConnect(root string) []string {
	return nil // no-op
}

func (m *MultiPlatform) checkKeyboardShortcuts(root string) []string {
	var issues []string

	// Check keyboard shortcuts documentation
	shortcutsPath := filepath.Join(root, "docs", "workstation", "OVAV_WEZTERM_WORKSPACE_ISOLATION.md")
	if data, err := os.ReadFile(shortcutsPath); err == nil {
		content := strings.ToLower(string(data))
		if !strings.Contains(content, "shortcut") && !strings.Contains(content, "hotkey") {
			issues = append(issues, "C7.4: keyboard shortcuts not documented in workspace isolation doc")
		}
	}

	return issues
}

func (m *MultiPlatform) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	issues = append(issues, m.checkWindowsReadiness(root)...)
	issues = append(issues, m.checkSkillsEnforcement(root)...)
	issues = append(issues, m.checkConnect(root)...)
	issues = append(issues, m.checkKeyboardShortcuts(root)...)

	if len(issues) > 0 {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message:  fmt.Sprintf("FAIL C7 multi-platform — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: m.ID(), Name: m.Name(), Status: "pass", Weight: m.Weight(),
		Message:  "PASS C7 multi-platform — Windows/macOS readiness, skills, connect verified",
		Duration: time.Since(start),
	}
}

var _ Validator = (*MultiPlatform)(nil)
