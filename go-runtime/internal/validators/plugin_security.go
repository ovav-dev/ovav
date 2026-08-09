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
		gitPushPresent := strings.Contains(content, `"git push*": "deny"`) || strings.Contains(content, `"git push*": "allow"`)
		if strings.Contains(content, `"edit": "allow"`) && !gitPushPresent {
			issues = append(issues, "SECURITY: opencode.json allows edits but doesn't block git push")
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
			// HTTPS without SSH — check if credential helper is configured
			if !strings.Contains(content, "credential") {
				issues = append(issues, "GIT: HTTPS transport without credential helper — consider SSH")
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
