package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SSHProfile validates the OVAV/Thavren SSH profile artifacts.
// Replaces: check_ovav_ssh_profile.py
type SSHProfile struct{}

func NewSSHProfile() *SSHProfile { return &SSHProfile{} }

func (s *SSHProfile) ID() string   { return "ssh_profile" }
func (s *SSHProfile) Name() string { return "SSH Profile Validator" }
func (s *SSHProfile) Description() string {
	return "Validates OVAV Thavren SSH profile artifacts — docs, templates, policy, and install plan"
}
func (s *SSHProfile) Weight() int { return 6 }

var sshRequiredFiles = []struct {
	path string
	name string
}{
	{"docs/workstation/OVAV_THAVREN_SSH_PROFILE.md", "doc"},
	{"docs/workstation/OVAV_WORKSTATION_ACCESS_PROFILE.md", "access_doc"},
	{"docs/workstation/OVAV_THAVREN_SSH_INSTALL_PLAN.md", "install_doc"},
	{"config/ssh/ovav-thavren.ssh.config.example", "ssh_template"},
	{"config/fish/ovav-thavren-ssh-agent.fish.example", "fish_template"},
	{"config/workstation/ovav-thavren-ssh-profile.yaml", "policy"},
	{"config/workstation/ovav-thavren-ssh-install-plan.yaml", "install_plan"},
	{"tools/workstation/ovav_workstation_access.py", "workstation_tool"},
}

func (s *SSHProfile) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	readFile := func(rel string) string {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(data)
	}

	// 1. Check all required files exist
	for _, f := range sshRequiredFiles {
		fullPath := filepath.Join(root, f.path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing %s: %s", f.name, f.path))
		}
	}
	if len(issues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: fmt.Sprintf("FAIL — %d missing file(s)", len(issues)),
			Issues:  issues, Duration: time.Since(start)}
	}

	// 2. Read all files
	sshText := readFile("config/ssh/ovav-thavren.ssh.config.example")
	fishText := readFile("config/fish/ovav-thavren-ssh-agent.fish.example")
	policyText := readFile("config/workstation/ovav-thavren-ssh-profile.yaml")
	installPlanText := readFile("config/workstation/ovav-thavren-ssh-install-plan.yaml")
	workstationText := readFile("tools/workstation/ovav_workstation_access.py")
	docText := readFile("docs/workstation/OVAV_THAVREN_SSH_PROFILE.md")
	accessDocText := readFile("docs/workstation/OVAV_WORKSTATION_ACCESS_PROFILE.md")
	installDocText := readFile("docs/workstation/OVAV_THAVREN_SSH_INSTALL_PLAN.md")

	// 3. Check SSH template tokens
	sshTokens := []string{
		"Host github-ovav-thavren", "HostName github.com", "User git",
		"IdentityFile ~/.ssh/ovav_thavren_ed25519", "IdentitiesOnly yes", "AddKeysToAgent yes",
	}
	for _, token := range sshTokens {
		if !strings.Contains(sshText, token) {
			issues = append(issues, fmt.Sprintf("ssh_template missing: %s", token))
		}
	}

	// 4. Check fish template tokens
	fishTokens := []string{
		"set -gx OVAV_SSH_HOST_ALIAS github-ovav-thavren",
		"set -gx OVAV_SSH_KEY_PATH",
		"set -gx OVAV_SSH_KEY_COMMENT ovav-thavren-github",
		"set -gx OVAV_SSH_AGENT_LIFETIME 24h",
		"function ovav_ssh_unlock",
		"function ovav_ssh_status",
		`ssh-add -t "$OVAV_SSH_AGENT_LIFETIME"`,
		"Intentionally not auto-unlocking on shell startup",
	}
	for _, token := range fishTokens {
		if !strings.Contains(fishText, token) {
			issues = append(issues, fmt.Sprintf("fish_template missing: %s", token))
		}
	}

	// 5. Check policy YAML tokens
	policyTokens := []string{
		"profile_id: ovav-thavren-ssh-profile",
		"transport: ssh",
		"passphrase_required: true",
		"host_alias: github-ovav-thavren",
		"remote_url_shape: git@github-ovav-thavren:ORG/REPO.git",
		"shell_profile_template: config/fish/ovav-thavren-ssh-agent.fish.example",
		"unlock_command: ovav_ssh_unlock",
		"expected_prompt_behavior: ask_passphrase_once_per_24h_when_key_is_missing_or_expired",
		"must_not_store:",
		"blocked_until_explicit_install_approval:",
	}
	for _, token := range policyTokens {
		if !strings.Contains(policyText, token) {
			issues = append(issues, fmt.Sprintf("policy missing: %s", token))
		}
	}

	// 6. Check install plan tokens
	installPlanTokens := []string{
		"plan_id: ovav-thavren-ssh-install-plan",
		"status: source_local_dry_run_only",
		"ssh_config_fragment: ~/.ssh/config.d/ovav-thavren.conf",
		"fish_agent_helper: ~/.config/fish/conf.d/ovav-thavren-ssh-agent.fish",
		"restore_previous_git_remote_url",
		"real_install_remains_blocked_until_explicit_install_gate",
	}
	for _, token := range installPlanTokens {
		if !strings.Contains(installPlanText, token) {
			issues = append(issues, fmt.Sprintf("install_plan missing: %s", token))
		}
	}

	// 7. Check workstation tool tokens
	workstationTokens := []string{
		"def build_plan()", "def diagnose()", "def verify_source()",
		"def blocked_apply()", "writes_performed", "Real workstation apply is blocked",
	}
	for _, token := range workstationTokens {
		if !strings.Contains(workstationText, token) {
			issues = append(issues, fmt.Sprintf("workstation_tool missing: %s", token))
		}
	}

	// 8. Check doc boundaries
	if !strings.Contains(docText, "~/.ssh/config") || !strings.Contains(docText, "no autoriza") {
		issues = append(issues, "doc missing no-global-write boundary")
	}
	if !strings.Contains(accessDocText, "No es un almacén de secretos") {
		issues = append(issues, "access_doc missing no-secret-custody statement")
	}
	if !strings.Contains(installDocText, "No aplica cambios reales todavía") {
		issues = append(issues, "install_doc missing blocked-real-apply statement")
	}

	// 9. Check no HTTPS remote in policy
	if strings.Contains(policyText, "https://github.com") {
		issues = append(issues, "policy references HTTPS remote (should be SSH)")
	}

	// 10. Parse policy YAML for structural check
	var policyDoc map[string]interface{}
	if err := yaml.Unmarshal([]byte(policyText), &policyDoc); err != nil {
		issues = append(issues, fmt.Sprintf("policy YAML parse error: %v", err))
	}

	if len(issues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message: "PASS — SSH profile artifacts valid", Duration: time.Since(start)}
}

var _ Validator = (*SSHProfile)(nil)
