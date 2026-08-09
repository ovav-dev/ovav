package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WorkspaceIsolation validates OVAV WezTerm workspace isolation artifacts.
// Replaces: check_ovav_wezterm_workspace_isolation.py
type WorkspaceIsolation struct{}

func NewWorkspaceIsolation() *WorkspaceIsolation { return &WorkspaceIsolation{} }

func (w *WorkspaceIsolation) ID() string   { return "workspace_isolation" }
func (w *WorkspaceIsolation) Name() string { return "Workspace Isolation Validator" }
func (w *WorkspaceIsolation) Description() string {
	return "Validates WezTerm workspace isolation, source template, and governed config"
}
func (w *WorkspaceIsolation) Weight() int { return 10 }

// Required workspace isolation artifacts.
var workspaceArtifacts = []string{
	"docs/workstation/OVAV_WEZTERM_WORKSPACE_ISOLATION.md",
	"config/workstation/ovav-wezterm-workspace-isolation.yaml",
	".ovav/source/configs/wezterm/ovav-workspace-isolation.wezterm.lua.example",
	"config/wezterm/wezterm.lua",
	".ovav/source/configs/wezterm/ovav-windows-loader.wezterm.lua",
	"tools/workstation/ovav_wezterm_workspace.py",
	".ovav/registry/tool_configs.yaml",
}

// Required documentation tokens — must be present in doc.
var docRequiredTokens = []string{
	"Workspace WezTerm",
	"repo + branch + root_hash",
	"cross_branch_attach",
	"launch-command",
	"check-current",
	"paleta tipo OVAV PI",
	"sin pane inactivo negro",
	"canvas sketchbook",
	"scroll Chrome-like",
	"tabs largas",
}

// Required policy tokens.
var policyRequiredTokens = []string{
	"profile_id: ovav-wezterm-workspace-isolation",
	"workspace_scope: repo_branch_root_hash",
	"workspace_name_shape: ovav-{repo_slug}-{branch_slug}-{root_hash}",
	"source_template:",
	"governed_payload: config/wezterm/wezterm.lua",
	"windows_loader:",
	"helper_tool: tools/workstation/ovav_wezterm_workspace.py",
	"blocked_until_explicit_install_approval",
	"color_profile: ovav_pi_eye_comfort",
	"inactive_pane_dimming: disabled_brightness_1_0",
	"chrome_like_scrollbar",
	"sketchbook_canvas",
	"OVAV_WZPROXY_v3",
}

// Required tool tokens.
var toolRequiredTokens = []string{
	"def workspace_name(",
	"def launch_command(",
	"def check_current(",
	"def blocked_apply()",
	"workspace_scope",
	"commands_executed",
	"writes_performed",
	"Real WezTerm apply is blocked",
	"no_wezterm_process_launch",
}

func (w *WorkspaceIsolation) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	found := 0
	missing := 0

	// 1. Check all required artifacts exist
	for _, artifact := range workspaceArtifacts {
		fullPath := filepath.Join(root, artifact)
		if info, err := os.Stat(fullPath); os.IsNotExist(err) {
			missing++
			issues = append(issues, fmt.Sprintf("MISSING: %s", artifact))
		} else if info.Size() == 0 {
			issues = append(issues, fmt.Sprintf("EMPTY: %s", artifact))
		} else {
			found++
		}
	}

	// 2. Validate doc tokens
	docPath := filepath.Join(root, "docs", "workstation", "OVAV_WEZTERM_WORKSPACE_ISOLATION.md")
	if data, err := os.ReadFile(docPath); err == nil {
		content := string(data)
		for _, token := range docRequiredTokens {
			if !strings.Contains(content, token) {
				issues = append(issues, fmt.Sprintf("DOC_TOKEN_MISSING: '%s' not found in workspace isolation doc", token))
			}
		}
	}

	// 3. Validate policy tokens
	policyPath := filepath.Join(root, "config", "workstation", "ovav-wezterm-workspace-isolation.yaml")
	if data, err := os.ReadFile(policyPath); err == nil {
		content := string(data)
		for _, token := range policyRequiredTokens {
			if !strings.Contains(content, token) {
				issues = append(issues, fmt.Sprintf("POLICY_TOKEN_MISSING: '%s' not found in workspace isolation policy", token))
			}
		}
	}

	// 4. Validate tool tokens
	toolPath := filepath.Join(root, "tools", "workstation", "ovav_wezterm_workspace.py")
	if data, err := os.ReadFile(toolPath); err == nil {
		content := string(data)
		for _, token := range toolRequiredTokens {
			if !strings.Contains(content, token) {
				issues = append(issues, fmt.Sprintf("TOOL_TOKEN_MISSING: '%s' not found in workstation tool", token))
			}
		}
		// Check that apply is truly blocked
		if strings.Contains(content, "subprocess.run([\"wezterm\"") {
			issues = append(issues, "TOOL: workstation tool may launch real wezterm process — apply must remain blocked")
		}
	}

	// 5. Validate governed wezterm config — check for OVAV branding and proper module pattern
	governedLua := filepath.Join(root, "config", "wezterm", "wezterm.lua")
	if data, err := os.ReadFile(governedLua); err == nil {
		content := string(data)
		// Check for OVAV branding (required in all OVAV-governed configs)
		if !strings.Contains(content, "OVAV") {
			issues = append(issues, "GOVERNED_WEZTERM: OVAV branding marker missing in wezterm.lua")
		}
		// Check for proper WezTerm module loading (require is correct for WezTerm v20240203+)
		if !strings.Contains(content, "require") {
			issues = append(issues, "GOVERNED_WEZTERM: require 'wezterm' pattern missing — use module loading")
		}
	}

	if len(issues) > 0 {
		status := "fail"
		// If only missing artifacts (not token violations), it's informational
		onlyMissing := true
		for _, issue := range issues {
			if !strings.HasPrefix(issue, "MISSING:") && !strings.HasPrefix(issue, "EMPTY:") {
				onlyMissing = false
				break
			}
		}
		if onlyMissing {
			status = "warn"
		}
		return Result{
			ID: w.ID(), Name: w.Name(), Status: status, Weight: w.Weight(),
			Message:  fmt.Sprintf("FAIL workspace isolation — %d/%d artifacts found, %d issue(s)", found, len(workspaceArtifacts), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: w.ID(), Name: w.Name(), Status: "pass", Weight: w.Weight(),
		Message:  fmt.Sprintf("PASS workspace isolation — %d/%d artifacts verified", found, len(workspaceArtifacts)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*WorkspaceIsolation)(nil)
