package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegoEngine_BuiltinTests(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	// Create a minimal policy file
	policy := `package ovav.security

default allow = false

deny_bash[rules] {
    input.command == "sudo"
}

deny_external_network {
    input.action == "external_request"
    not ovav_allowed_domain
}

ovav_allowed_domain {
    allowed := {"github.com", "api.github.com", "pypi.org", "files.pythonhosted.org", "docs.python.org", "arxiv.org", "scholar.google.com", "ovav.dev"}
    allowed[input.domain]
}

allow {
    input.operator == "thavren"
    input.scope == "repo_local"
}
`
	os.WriteFile(filepath.Join(policiesDir, "security.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	if err := engine.LoadPolicies(); err != nil {
		t.Fatalf("failed to load policies: %v", err)
	}

	tests := BuiltinTests()
	results := engine.TestPolicy(tests)

	passed := results["passed"].(int)
	failed := results["failed"].(int)
	total := results["total"].(int)

	t.Logf("Rego engine tests: %d/%d passed", passed, total)

	if failed > 0 {
		t.Errorf("Expected all tests to pass, but %d failed", failed)
		for _, r := range results["results"].([]map[string]interface{}) {
			if !r["passed"].(bool) {
				t.Errorf("  FAIL: %s (expected=%v, actual=%v, reason=%s)",
					r["name"], r["expected"], r["actual"], r["reason"])
			}
		}
	}
}

func TestRegoEngine_DenyRules(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	policy := `package ovav.security

default allow = false

deny_bash[rules] {
    input.command == "sudo"
}

deny_path_traversal {
    input.path == "../etc/passwd"
}

deny_force_push {
    input.flags == "--force"
}

allow {
    input.operator == "thavren"
    input.scope == "repo_local"
}
`
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	engine.LoadPolicies()

	// Test sudo blocked
	decision := engine.Evaluate("bash", map[string]interface{}{
		"command":         "sudo rm -rf /",
		"operator":        "thavren",
		"scope":           "repo_local",
		"bootstrap_valid": true,
	})
	if decision.Allowed {
		t.Error("Expected sudo to be blocked")
	}

	// Test path traversal blocked
	decision = engine.Evaluate("file_write", map[string]interface{}{
		"path":            "../etc/passwd",
		"operator":        "thavren",
		"bootstrap_valid": true,
	})
	if decision.Allowed {
		t.Error("Expected path traversal to be blocked")
	}

	// Test force push blocked
	decision = engine.Evaluate("git_push", map[string]interface{}{
		"flags":           "--force",
		"operator":        "thavren",
		"bootstrap_valid": true,
	})
	if decision.Allowed {
		t.Error("Expected force push to be blocked")
	}

	// Test safe operation allowed
	decision = engine.Evaluate("bash", map[string]interface{}{
		"command":         "ls",
		"operator":        "thavren",
		"scope":           "repo_local",
		"bootstrap_valid": true,
	})
	if !decision.Allowed {
		t.Errorf("Expected safe operation to be allowed, got: %s", decision.Reason)
	}
}

func TestRegoEngine_BootstrapRequired(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	policy := `package ovav.security

default allow = false

allow {
    input.operator == "thavren"
}
`
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	engine.LoadPolicies()

	// Without bootstrap, should be denied
	decision := engine.Evaluate("bash", map[string]interface{}{
		"command":         "ls",
		"operator":        "thavren",
		"bootstrap_valid": false,
	})
	if decision.Allowed {
		t.Error("Expected denial without bootstrap verification")
	}

	// With bootstrap, should be allowed (scope required by allow rule)
	decision = engine.Evaluate("bash", map[string]interface{}{
		"command":         "ls",
		"operator":        "thavren",
		"scope":           "repo_local",
		"bootstrap_valid": true,
	})
	if !decision.Allowed {
		t.Errorf("Expected allow with bootstrap, got: %s", decision.Reason)
	}
}

func TestBashCommandGovernor_AllowRules(t *testing.T) {
	gov := NewBashCommandGovernor()

	tests := []struct {
		command string
		allowed bool
	}{
		{"git status", true},
		{"git log --oneline", true},
		{"git diff HEAD", true},
		{"ls -la", true},
		{"cat file.txt", true},
		{"python3 tools/validators/check.py", true},
		{"git push --force", false},
		{"sudo rm -rf /", false},
		{"pip install malware", false},
		{"unknown_command", false},
	}

	for _, tc := range tests {
		decision := gov.Check(tc.command, "thavren")
		if decision.Allowed != tc.allowed {
			t.Errorf("Command %q: expected allowed=%v, got %v (rule=%s, reason=%s)",
				tc.command, tc.allowed, decision.Allowed, decision.MatchedRule, decision.Reason)
		}
	}
}

func TestBashCommandGovernor_Summary(t *testing.T) {
	gov := NewBashCommandGovernor()
	summary := gov.GetSummary()

	total := summary["total_rules"].(int)
	allowed := summary["allowed"].(int)
	denied := summary["denied"].(int)

	if total != 16 {
		t.Errorf("Expected 16 total rules, got %d", total)
	}
	if allowed != 9 {
		t.Errorf("Expected 9 allow rules, got %d", allowed)
	}
	if denied != 7 {
		t.Errorf("Expected 7 deny rules, got %d", denied)
	}
}

func TestNewStatesGovernor_AllowStates(t *testing.T) {
	gov := NewNewStatesGovernor()

	// Test allowed states
	allowedStates := []string{"delegated", "adaptive", "consensus_required", "revocable"}
	for _, state := range allowedStates {
		decision := gov.Check(state)
		if !decision.Allowed {
			t.Errorf("Expected state %q to be allowed, got: %s", state, decision.Reason)
		}
	}

	// Test denied states
	deniedStates := []string{"inherited", "canary_gated"}
	for _, state := range deniedStates {
		decision := gov.Check(state)
		if decision.Allowed {
			t.Errorf("Expected state %q to be denied", state)
		}
	}

	// Test unknown state
	decision := gov.Check("unknown_state")
	if decision.Allowed {
		t.Error("Expected unknown state to be denied")
	}
}

func TestPluginGovernor_Authorization(t *testing.T) {
	gov := NewPluginGovernor()

	// Thavren can install plugins
	decision := gov.AuthorizePlugin("thavren", "test-plugin", "plugin")
	if !decision.Allowed {
		t.Errorf("Expected thavren to be authorized, got: %s", decision.Reason)
	}
	if len(decision.RequiresGates) != 3 {
		t.Errorf("Expected 3 required gates, got %d", len(decision.RequiresGates))
	}

	// Eidren cannot install plugins
	decision = gov.AuthorizePlugin("eidren", "test-plugin", "plugin")
	if decision.Allowed {
		t.Error("Expected eidren to be denied")
	}

	// MCP servers are always blocked
	decision = gov.AuthorizePlugin("thavren", "test-mcp", "mcp_server")
	if decision.Allowed {
		t.Error("Expected MCP server to be blocked")
	}
}

func TestSandboxGovernor_Operations(t *testing.T) {
	gov := NewSandboxGovernor()

	// Test allowed operations in sandbox
	decision := gov.CheckOperation("live_probe", true)
	if !decision.Allowed {
		t.Errorf("Expected live_probe in sandbox to be allowed, got: %s", decision.Reason)
	}

	// Test sandbox_runner always denied
	decision = gov.CheckOperation("sandbox_runner", true)
	if decision.Allowed {
		t.Error("Expected sandbox_runner to be permanently denied")
	}

	// Test operation requiring sandbox but not in sandbox
	decision = gov.CheckOperation("live_probe", false)
	if decision.Allowed {
		t.Error("Expected live_probe outside sandbox to be denied")
	}

	// Test operation not requiring sandbox
	decision = gov.CheckOperation("redaction_policy", false)
	if !decision.Allowed {
		t.Errorf("Expected redaction_policy outside sandbox to be allowed, got: %s", decision.Reason)
	}
}

func TestSystemPathGovernor_SystemPaths(t *testing.T) {
	gov := NewSystemPathGovernor("/home/test")

	// Test /etc read allowed
	decision := gov.Check("/etc/passwd", "read")
	if !decision.Allowed {
		t.Errorf("Expected /etc read to be allowed, got: %s", decision.Reason)
	}

	// Test /etc write requires gate
	decision = gov.Check("/etc/config", "write")
	if decision.Allowed {
		t.Error("Expected /etc write to require gate")
	}
	if decision.RequiresGate != "requires_security_gate" {
		t.Errorf("Expected requires_security_gate, got: %s", decision.RequiresGate)
	}

	// Test /proc write denied
	decision = gov.Check("/proc/1/cmdline", "write")
	if decision.Allowed {
		t.Error("Expected /proc write to be denied")
	}

	// Test unknown path
	decision = gov.Check("/unknown/path", "read")
	if decision.Allowed {
		t.Error("Expected unknown path to be denied")
	}
}

func TestPermissionAuthority_Materialize(t *testing.T) {
	dir := t.TempDir()

	// Create minimal structure
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)

	policy := map[string]interface{}{
		"schema_version": "ovav.permission_authority.v1",
		"materialized_targets": []string{
			"opencode.json",
			".opencode/agents/area-platform-engineering.md",
			".opencode/agents/lead-thavren.md",
		},
	}
	policyJSON, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), policyJSON, 0644)

	// Create opencode.json
	opencode := map[string]interface{}{"version": "1.0"}
	opencodeJSON, _ := json.Marshal(opencode)
	os.WriteFile(filepath.Join(dir, "opencode.json"), opencodeJSON, 0644)

	// Create agent files
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentContent := `---
# Agent frontmatter
---
# Agent body
`
	os.WriteFile(filepath.Join(agentsDir, "area-platform-engineering.md"), []byte(agentContent), 0644)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"), []byte(agentContent), 0644)

	auth := NewPermissionAuthority(dir)
	result, err := auth.MaterializeAll(false)
	if err != nil {
		t.Fatalf("MaterializeAll failed: %v", err)
	}

	status := result["status"].(string)
	if status != "changed" {
		t.Errorf("Expected status 'changed', got: %s", status)
	}

	changedCount := result["changed_count"].(int)
	if changedCount != 3 {
		t.Errorf("Expected 3 changed targets, got: %d", changedCount)
	}
}

func TestPermissionAuthority_CheckDrift(t *testing.T) {
	dir := t.TempDir()

	// Create minimal structure
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)

	policy := map[string]interface{}{
		"schema_version":       "ovav.permission_authority.v1",
		"materialized_targets": []string{"opencode.json"},
	}
	policyJSON, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), policyJSON, 0644)

	// Create opencode.json with wrong permissions
	opencode := map[string]interface{}{
		"version": "1.0",
		"permission": map[string]interface{}{
			"edit": "deny", // Should be "allow"
		},
	}
	opencodeJSON, _ := json.Marshal(opencode)
	os.WriteFile(filepath.Join(dir, "opencode.json"), opencodeJSON, 0644)

	// Create agent fixture files required by CheckAll()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentContent := `---
name: Area Platform Engineering
permission:
  edit: allow
  bash:
    "git push*": deny
    "git push --force *": deny
    "gh auth token*": deny
    "sudo *": deny
    "go run -C go-runtime ./cmd/ovav validate*": allow
    "go run -C go-runtime ./internal/validators/cmd/validate*": allow
    "git commit*": allow
    "git ls-remote *": allow
    "gh pr create*": allow
    "*": allow
  external_directory:
    "/tmp/opencode/*": allow
    "*": deny
# OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json
---
# Agent body
`
	os.WriteFile(filepath.Join(agentsDir, "area-platform-engineering.md"), []byte(agentContent), 0644)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"), []byte(agentContent), 0644)

	// Create tool access policy and gateway fixture files
	serviceDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(serviceDir, 0755)
	toolPolicy := `canonical_source: .ovav/policy/permission_authority.json
drift_response: log_and_restore_ovav_policy
decision: allow_when_user_requested_and_workspace_safety_gate_passes
decision: deny_raw_use_ovav_git_push_gate_with_user_confirmation
`
	os.WriteFile(filepath.Join(serviceDir, "tool_access_policy.yaml"), []byte(toolPolicy), 0644)

	toolsDir := filepath.Join(dir, "tools", "agent_runtime")
	os.MkdirAll(toolsDir, 0755)
	toolGateway := `PLATFORM_APPROVED_GIT = True
approved_governed_git_operation = lambda x: True
`
	os.WriteFile(filepath.Join(toolsDir, "tool_gateway.py"), []byte(toolGateway), 0644)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.CheckAll(false)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}

	if len(drift) == 0 {
		t.Error("Expected drift to be detected")
	}

	// Check that drift includes permission.edit
	foundEditDrift := false
	for _, d := range drift {
		if field, ok := d["field"].(string); ok && field == "permission.edit" {
			foundEditDrift = true
			break
		}
	}
	if !foundEditDrift {
		t.Error("Expected drift for permission.edit field")
	}
}
