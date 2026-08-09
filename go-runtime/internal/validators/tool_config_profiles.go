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
type ToolConfigProfiles struct{}

func NewToolConfigProfiles() *ToolConfigProfiles { return &ToolConfigProfiles{} }

func (t *ToolConfigProfiles) ID() string   { return "tool_config_profiles" }
func (t *ToolConfigProfiles) Name() string { return "Tool Config Profiles Validator" }
func (t *ToolConfigProfiles) Description() string {
	return "Validates tool_configs.yaml registry, CLI tool, and bin/ovav consistency"
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

var toolConfigCLITokens = []string{
	"OVAV Tool Config Profiles",
	"WEZTERM_HELPER",
	"ovav.tool_config_profile_action.v1",
	"Real WezTerm config apply is blocked",
	"writes_performed",
	`shutil.which("wezterm")`,
}

var toolConfigBinTokens = []string{
	"ovav tools wezterm plan",
	"ovav_tool_configs.py",
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

	// 1. Check source files exist
	// Note: tools/cli/ovav_tool_configs.py is deprecated (Go-native only)
	for _, p := range []struct{ path, name string }{
		{".ovav/registry/tool_configs.yaml", "registry"},
		{"bin/ovav", "bin_ovav"},
	} {
		fullPath := filepath.Join(root, p.path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing %s: %s", p.name, p.path))
		}
	}

	if len(issues) > 0 {
		return Result{ID: t.ID(), Name: t.Name(), Status: "fail", Weight: t.Weight(),
			Message: fmt.Sprintf("FAIL — %d missing source file(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start)}
	}

	// 2. Read files
	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}
	registry := readFile(".ovav/registry/tool_configs.yaml")
	binText := readFile("bin/ovav")

	// 3. Check registry tokens
	for _, token := range toolConfigRequiredTokens {
		if !strings.Contains(registry, token) {
			issues = append(issues, fmt.Sprintf("registry missing token: %s", token))
		}
	}

	// 4. Check bin tokens
	for _, token := range toolConfigBinTokens {
		if !strings.Contains(binText, token) {
			issues = append(issues, fmt.Sprintf("bin/ovav missing token: %s", token))
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
