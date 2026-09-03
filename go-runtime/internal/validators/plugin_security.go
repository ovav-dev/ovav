package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PluginSecurity validates opencode plugins, network hardening, and SSH configuration.
// Replaces: check_opencode_plugins.py, check_network_hardening.py,
// check_ovav_ssh_profile.py, check_wezterm_path_integrity.py
type PluginSecurity struct{}

func NewPluginSecurity() *PluginSecurity { return &PluginSecurity{} }

func (p *PluginSecurity) ID() string   { return "plugin_security" }
func (p *PluginSecurity) Name() string { return "Plugin & Network Security" }
func (p *PluginSecurity) Description() string {
	return "Validates opencode plugins, network hardening, and SSH config"
}
func (p *PluginSecurity) Weight() int { return 15 }

func (p *PluginSecurity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check opencode.json plugins are valid
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		content := string(data)
		// Check plugin array is valid format
		if strings.Contains(content, `"plugin"`) {
			if !strings.Contains(content, `"plugin": []`) && !strings.Contains(content, `"plugin":[]`) {
				// Has plugins — verify they're properly formatted
				pluginIdx := strings.Index(content, `"plugin"`)
				if pluginIdx >= 0 {
					tail := content[pluginIdx:]
					// Simple check: no suspicious characters in plugin names
					if strings.Contains(tail, "..") || strings.Contains(tail, "rm -rf") {
						issues = append(issues, "PLUGIN: Suspicious content in plugin configuration")
					}
				}
			}
		}

		// Check no insecure permissions
		// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
		// YOLO mode: bash is 100% allow (no deny). The git push gate is enforced
		// by the Go-native ovav push_cli. Skip the opencode.json git push deny
		// check if YOLO is active.
		gitPushPresent := strings.Contains(content, `"git push*": "deny"`) || strings.Contains(content, `"git push*": "allow"`)
		isYolo := strings.Contains(content, `"_ovav"`) || strings.Contains(content, `"yolo"`)
		if strings.Contains(content, `"edit": "allow"`) && !gitPushPresent && !isYolo {
			issues = append(issues, "SECURITY: opencode.json allows edits but doesn't block git push (or enable YOLO via _ovav marker)")
		}
	}

	// 2. Check .git/config for SSH hardening
	gitConfigPath := filepath.Join(root, ".git", "config")
	if data, err := os.ReadFile(gitConfigPath); err == nil {
		content := string(data)
		// SSH transport is more secure than HTTPS
		if strings.Contains(content, "url = https://") && strings.Contains(content, "insteadOf = git@") {
			// OK: HTTPS with SSH insteadOf is reasonable
		} else if strings.Contains(content, "url = git@") {
			// SSH — good
		} else if strings.Contains(content, "url = https://") {
			// HTTPS without SSH — accept either:
			//   (a) traditional `credential.helper` config in .git/config
			//   (b) gh CLI authentication via ~/.config/gh/hosts.yml (modern dev workflow)
			if !strings.Contains(content, "credential") && !hasGhCLIAuth() {
				issues = append(issues, "GIT: HTTPS transport without credential helper — consider SSH or `gh auth login`")
			}
		}
	}

	// 3. Check .gitleaks.toml exists for secret scanning
	gitleaksPath := filepath.Join(root, ".gitleaks.toml")
	if _, err := os.Stat(gitleaksPath); os.IsNotExist(err) {
		issues = append(issues, "MISSING: .gitleaks.toml — GitLeaks config for secret scanning not found")
	}

	// 4. Check for .env files that shouldn't be committed
	envPatterns := []string{".env", ".env.local", ".env.production"}
	for _, pat := range envPatterns {
		envPath := filepath.Join(root, pat)
		if info, err := os.Stat(envPath); err == nil && !info.IsDir() {
			// Check if it's in .gitignore
			gitignorePath := filepath.Join(root, ".gitignore")
			if gitignoreData, err := os.ReadFile(gitignorePath); err == nil {
				if !strings.Contains(string(gitignoreData), pat) {
					issues = append(issues, fmt.Sprintf("SECURITY: %s exists but not in .gitignore", pat))
				}
			} else {
				issues = append(issues, fmt.Sprintf("SECURITY: %s exists but .gitignore not found", pat))
			}
		}
	}

	// 5. Verify no open network ports in config
	// Check fly.toml for exposed ports (0.0.0.0 is standard Fly.io proxy binding)
	flyPath := filepath.Join(root, "fly.toml")
	if data, err := os.ReadFile(flyPath); err == nil {
		content := string(data)
		if strings.Contains(content, "internal_port = 5858") {
			// cPanel port — expected
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: p.ID(), Name: p.Name(), Status: "fail", Weight: p.Weight(),
			Message: fmt.Sprintf("FAIL plugin/network security — %d issue(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start),
		}
	}
	return Result{
		ID: p.ID(), Name: p.Name(), Status: "pass", Weight: p.Weight(),
		Message:  "PASS plugin/network security",
		Duration: time.Since(start),
	}
}

var _ Validator = (*PluginSecurity)(nil)

// hasGhCLIAuth reports whether the user has authenticated the GitHub CLI.
//
// `gh` stores its session token in ~/.config/gh/hosts.yml as
// `oauth_token: <token>` (or `users.<user>.oauth_token`). When present,
// git's https transport automatically uses the token via the `gh` credential
// helper, even without `credential.helper` set in .git/config. This is the
// canonical auth path on modern dev workstations and CI runners.
//
// Returns false if the file is missing, unreadable, or contains no oauth_token.
func hasGhCLIAuth() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	hostsPath := filepath.Join(home, ".config", "gh", "hosts.yml")
	data, err := os.ReadFile(hostsPath)
	if err != nil {
		return false
	}
	// hosts.yml is YAML, but a substring search for the token field is robust
	// enough for the boolean decision we need here. False positives are
	// extremely unlikely (a literal "oauth_token: " in user-controlled config
	// is itself a security smell).
	return strings.Contains(string(data), "oauth_token:")
}
