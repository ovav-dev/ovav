package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ToolConfigProfiles validates the tool_configs.yaml registry and CLI surface files.
// Replaces: check_tool_config_profiles.py
//
// Migration note (2026-08): Python CLI helper (tools/cli/ovav_tool_configs.py) and
// Python source bin/ovav were removed in the Python→Go migration. The validator now
// focuses on the tool_configs.yaml registry schema only. Build artifacts
// (go-runtime/build/ovav) are validated by `make test` + `make vet`, not here.
// bin/ovav is intentionally gitignored — see .gitignore.
type ToolConfigProfiles struct{}

func NewToolConfigProfiles() *ToolConfigProfiles { return &ToolConfigProfiles{} }

func (t *ToolConfigProfiles) ID() string   { return "tool_config_profiles" }
func (t *ToolConfigProfiles) Name() string { return "Tool Config Profiles Validator" }
func (t *ToolConfigProfiles) Description() string {
	return "Validates tool_configs.yaml registry schema (build artifacts checked by make test/vet)"
}
func (t *ToolConfigProfiles) Weight() int { return 6 }

var toolConfigRequiredTokens = []string{
	"tool_config_profiles:",
	"wezterm_workspace_isolation:",
	"category: terminal",
	"ovav_tailor",
	"ovav tools wezterm plan",
	"ovav tools wezterm verify",
	"ovav_installs_wezterm: false",
	"writes_user_home_now: false",
	"launches_real_wezterm_now: false",
}

var toolConfigBlockedTokens = []string{
	"write_text(",
	`subprocess.run(["wezterm"`,
	"~/..ovav/source/configs/wezterm/wezterm.lua",
	"/mnt/c/Users",
}

func (t *ToolConfigProfiles) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check that the tool_configs.yaml registry exists.
	// bin/ovav is intentionally gitignored (Go build artifact) and not checked here.
	registryPath := filepath.Join(root, ".ovav/registry/tool_configs.yaml")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return Result{ID: t.ID(), Name: t.Name(), Status: "fail", Weight: t.Weight(),
			Message:    "FAIL — missing registry: .ovav/registry/tool_configs.yaml",
			Issues:     []string{"missing registry: .ovav/registry/tool_configs.yaml"},
			Duration:   time.Since(start)}
	}

	// 2. Read registry and verify required schema tokens.
	registryBytes, err := os.ReadFile(registryPath)
	if err != nil {
		return Result{ID: t.ID(), Name: t.Name(), Status: "fail", Weight: t.Weight(),
			Message:    fmt.Sprintf("FAIL — cannot read registry: %v", err),
			Issues:     []string{fmt.Sprintf("registry read error: %v", err)},
			Duration:   time.Since(start)}
	}
	registry := string(registryBytes)

	for _, token := range toolConfigRequiredTokens {
		if !strings.Contains(registry, token) {
			issues = append(issues, fmt.Sprintf("registry missing token: %s", token))
		}
	}

	if len(issues) > 0 {
		return Result{ID: t.ID(), Name: t.Name(), Status: "fail", Weight: t.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: t.ID(), Name: t.Name(), Status: "pass", Weight: t.Weight(),
		Message: "PASS — tool config profiles valid", Duration: time.Since(start)}
}

var _ Validator = (*ToolConfigProfiles)(nil)
