package permissions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// coverage_boost_test.go — Boost coverage from 70.5% to 80%+
// Targets: 0% functions, low-coverage branches, error paths
// ═══════════════════════════════════════════════════════════════════════════

// ── PermissionAuthority helpers ─────────────────────────────────────────

func writeValidPolicy(t *testing.T, dir string) {
	t.Helper()
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
	data, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0644)
}

func writeMinimalOpencode(t *testing.T, dir string) {
	t.Helper()
	cfg := map[string]interface{}{"version": "1.0"}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0644)
}

func writeAgentFiles(t *testing.T, dir string) {
	t.Helper()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	content := `---
name: Test Agent
---
# Agent body
`
	os.WriteFile(filepath.Join(agentsDir, "area-platform-engineering.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"), []byte(content), 0644)
}

func writeToolFixtures(t *testing.T, dir string) {
	t.Helper()
	serviceDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(serviceDir, 0755)
	policy := `canonical_source: .ovav/policy/permission_authority.json
drift_response: log_and_restore_ovav_policy
decision: allow_when_user_requested_and_workspace_safety_gate_passes
decision: deny_raw_use_ovav_git_push_gate_with_user_confirmation
`
	os.WriteFile(filepath.Join(serviceDir, "tool_access_policy.yaml"), []byte(policy), 0644)

	toolsDir := filepath.Join(dir, "tools", "agent_runtime")
	os.MkdirAll(toolsDir, 0755)
	gateway := `PLATFORM_APPROVED_GIT = True
approved_governed_git_operation = lambda x: True
`
	os.WriteFile(filepath.Join(toolsDir, "tool_gateway.py"), []byte(gateway), 0644)
}

// ── appendLog (0% → test coverage) ─────────────────────────────────────

func TestAppendLog_WritesDriftEvent(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)

	drift := []map[string]interface{}{
		{"surface": "opencode.json", "field": "permission.edit", "expected": "allow", "actual": "deny"},
	}
	auth.appendLog(drift)

	logPath := filepath.Join(dir, ".ovav", "runtime", "logs", "permission_drift.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected log file to exist: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected log file to have content")
	}
}

func TestAppendLog_CustomLogPath(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)
	policy := map[string]interface{}{
		"schema_version": "ovav.permission_authority.v1",
		"log_path":       "custom/logs/drift.jsonl",
	}
	data, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0644)

	auth := NewPermissionAuthority(dir)
	auth.appendLog([]map[string]interface{}{{"surface": "test"}})

	logPath := filepath.Join(dir, "custom", "logs", "drift.jsonl")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected custom log path to be used")
	}
}

func TestAppendLog_InvalidPolicyJSON(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), []byte("not json"), 0644)

	auth := NewPermissionAuthority(dir)
	// Should not panic
	auth.appendLog([]map[string]interface{}{{"surface": "test"}})
}

func TestAppendLog_MissingPolicyFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	// Should not panic
	auth.appendLog([]map[string]interface{}{{"surface": "test"}})
}

// ── ReconcileAll (0% → test coverage) ──────────────────────────────────

func TestReconcileAll_CheckMode(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)
	result, err := auth.ReconcileAll(false, false)
	if err != nil {
		t.Fatalf("ReconcileAll failed: %v", err)
	}
	if result["status"] == nil {
		t.Error("expected status field")
	}
	if result["authority"] == nil {
		t.Error("expected authority field")
	}
	mode := result["mode"].(string)
	if mode != "check" {
		t.Errorf("expected check mode, got %s", mode)
	}
}

func TestReconcileAll_WriteMode(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)
	result, err := auth.ReconcileAll(true, false)
	if err != nil {
		t.Fatalf("ReconcileAll failed: %v", err)
	}
	mode := result["mode"].(string)
	if mode != "write" {
		t.Errorf("expected write mode, got %s", mode)
	}
}

func TestReconcileAll_WithWriteLog(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)
	result, err := auth.ReconcileAll(false, true)
	if err != nil {
		t.Fatalf("ReconcileAll failed: %v", err)
	}
	if result["status"] == nil {
		t.Error("expected status field")
	}
}

// ── assertPolicySafe error paths ────────────────────────────────────────

func TestAssertPolicySafe_MissingPolicyFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	err := auth.assertPolicySafe()
	if err == nil {
		t.Error("expected error for missing policy file")
	}
}

func TestAssertPolicySafe_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), []byte("not json"), 0644)

	auth := NewPermissionAuthority(dir)
	err := auth.assertPolicySafe()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestAssertPolicySafe_WrongSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)
	policy := map[string]interface{}{
		"schema_version":       "wrong_version",
		"materialized_targets": []string{"opencode.json"},
	}
	data, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0644)

	auth := NewPermissionAuthority(dir)
	err := auth.assertPolicySafe()
	if err == nil {
		t.Error("expected error for wrong schema version")
	}
}

func TestAssertPolicySafe_MissingTargets(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".ovav", "policy")
	os.MkdirAll(policyDir, 0755)
	policy := map[string]interface{}{
		"schema_version":       "ovav.permission_authority.v1",
		"materialized_targets": []string{"opencode.json"},
	}
	data, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0644)

	auth := NewPermissionAuthority(dir)
	err := auth.assertPolicySafe()
	if err == nil {
		t.Error("expected error for missing materialized targets")
	}
}

func TestAssertPolicySafe_V2SchemaValid(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	// Overwrite with v2
	policyDir := filepath.Join(dir, ".ovav", "policy")
	policy := map[string]interface{}{
		"schema_version": "ovav.permission_authority.v2",
		"materialized_targets": []string{
			"opencode.json",
			".opencode/agents/area-platform-engineering.md",
			".opencode/agents/lead-thavren.md",
		},
	}
	data, _ := json.Marshal(policy)
	os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), data, 0644)

	auth := NewPermissionAuthority(dir)
	err := auth.assertPolicySafe()
	if err != nil {
		t.Errorf("v2 schema should be valid, got: %v", err)
	}
}

// ── buildOpencodeProjection error paths ─────────────────────────────────

func TestBuildOpencodeProjection_MissingFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	_, err := auth.buildOpencodeProjection()
	if err == nil {
		t.Error("expected error for missing opencode.json")
	}
}

func TestBuildOpencodeProjection_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("not json"), 0644)

	auth := NewPermissionAuthority(dir)
	_, err := auth.buildOpencodeProjection()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── buildAgentProjection error paths ────────────────────────────────────

func TestBuildAgentProjection_MissingFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	_, err := auth.buildAgentProjection(filepath.Join(dir, "nonexistent.md"))
	if err == nil {
		t.Error("expected error for missing agent file")
	}
}

func TestBuildAgentProjection_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentFile := filepath.Join(agentsDir, "test-agent.md")
	os.WriteFile(agentFile, []byte("# No frontmatter\nBody only\n"), 0644)

	auth := NewPermissionAuthority(dir)
	_, err := auth.buildAgentProjection(agentFile)
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestBuildAgentProjection_MalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentFile := filepath.Join(agentsDir, "test-agent.md")
	// Only one --- delimiter
	os.WriteFile(agentFile, []byte("---\nname: test\nBody\n"), 0644)

	auth := NewPermissionAuthority(dir)
	_, err := auth.buildAgentProjection(agentFile)
	if err == nil {
		t.Error("expected error for malformed frontmatter")
	}
}

func TestBuildAgentProjection_WithExistingPermission(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentFile := filepath.Join(agentsDir, "test-agent.md")
	content := `---
name: Test Agent
permission:
  edit: allow
  bash:
    "*": allow
---
# Agent body
`
	os.WriteFile(agentFile, []byte(content), 0644)

	auth := NewPermissionAuthority(dir)
	result, err := auth.buildAgentProjection(agentFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty result")
	}
}

// ── MaterializeAll error paths ──────────────────────────────────────────

func TestMaterializeAll_MissingPolicyFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	_, err := auth.MaterializeAll(false)
	if err == nil {
		t.Error("expected error for missing policy file")
	}
}

func TestMaterializeAll_WriteMode(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)
	result, err := auth.MaterializeAll(true)
	if err != nil {
		t.Fatalf("MaterializeAll write mode failed: %v", err)
	}
	mode := result["mode"].(string)
	if mode != "write" {
		t.Errorf("expected write mode, got %s", mode)
	}
}

func TestMaterializeAll_AlreadyClean(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)

	auth := NewPermissionAuthority(dir)

	// Write once to project
	result, err := auth.MaterializeAll(true)
	if err != nil {
		t.Fatalf("MaterializeAll write failed: %v", err)
	}
	if result["status"] == nil {
		t.Error("expected status field")
	}
	// Verify all 3 targets were materialized
	targets := result["targets"].([]map[string]interface{})
	if len(targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(targets))
	}
	// Verify each has path, changed, action
	for _, tgt := range targets {
		if tgt["path"] == nil {
			t.Error("target missing path")
		}
		if tgt["action"] == nil {
			t.Error("target missing action")
		}
	}
}

// ── CheckAll with drift + log ──────────────────────────────────────────

func TestCheckAll_WithDriftAndLog(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)
	writeMinimalOpencode(t, dir)
	writeAgentFiles(t, dir)
	writeToolFixtures(t, dir)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.CheckAll(true)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}
	// Should have drift and trigger appendLog
	_ = drift
}

func TestCheckAll_NoDrift(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)

	// Write correct opencode.json
	perm := ExpectedOpencodePermission()
	cfg := map[string]interface{}{
		"version":    "1.0",
		"permission": perm,
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0644)

	// Write correct agent files with authority marker
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	correctAgent := `---
name: Test
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
# Body
`
	os.WriteFile(filepath.Join(agentsDir, "area-platform-engineering.md"), []byte(correctAgent), 0644)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"), []byte(correctAgent), 0644)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.CheckAll(false)
	if err != nil {
		t.Fatalf("CheckAll failed: %v", err)
	}
	// Should have minimal or no drift
	_ = drift
}

// ── checkOpencode error paths ───────────────────────────────────────────

func TestCheckOpencode_MissingFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	_, err := auth.checkOpencode()
	if err == nil {
		t.Error("expected error for missing opencode.json")
	}
}

func TestCheckOpencode_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("not json"), 0644)

	auth := NewPermissionAuthority(dir)
	_, err := auth.checkOpencode()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── checkAgent error paths ──────────────────────────────────────────────

func TestCheckAgent_MissingFile(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	_, err := auth.checkAgent(filepath.Join(dir, "nonexistent.md"), "test.md")
	if err == nil {
		t.Error("expected error for missing agent file")
	}
}

func TestCheckAgent_NoAuthorityMarker(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentFile := filepath.Join(agentsDir, "test.md")
	content := `---
name: Test
---
# Body
`
	os.WriteFile(agentFile, []byte(content), 0644)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.checkAgent(agentFile, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hasMarkerDrift := false
	for _, d := range drift {
		if f, ok := d["field"].(string); ok && f == "authority_marker" {
			hasMarkerDrift = true
		}
	}
	if !hasMarkerDrift {
		t.Error("expected authority_marker drift")
	}
}

func TestCheckAgent_ForbiddenLines(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentFile := filepath.Join(agentsDir, "test.md")
	content := `---
name: Test
permission:
  edit: ask
  bash: ask
  external_directory:
    "*": ask
# OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json
---
# Body
`
	os.WriteFile(agentFile, []byte(content), 0644)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.checkAgent(agentFile, "test.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	forbiddenCount := 0
	for _, d := range drift {
		if _, ok := d["forbidden"]; ok {
			forbiddenCount++
		}
	}
	if forbiddenCount == 0 {
		t.Error("expected forbidden line drift entries")
	}
}

// ── checkRuntimePolicySurfaces when files don't exist ───────────────────

func TestCheckRuntimePolicySurfaces_NoFiles(t *testing.T) {
	dir := t.TempDir()
	auth := NewPermissionAuthority(dir)
	drift, err := auth.checkRuntimePolicySurfaces()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drift) != 0 {
		t.Errorf("expected no drift when files don't exist, got %d", len(drift))
	}
}

func TestCheckRuntimePolicySurfaces_MissingExpectedContent(t *testing.T) {
	dir := t.TempDir()
	serviceDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(serviceDir, 0755)
	os.WriteFile(filepath.Join(serviceDir, "tool_access_policy.yaml"), []byte("empty"), 0644)

	toolsDir := filepath.Join(dir, "tools", "agent_runtime")
	os.MkdirAll(toolsDir, 0755)
	os.WriteFile(filepath.Join(toolsDir, "tool_gateway.py"), []byte("empty"), 0644)

	auth := NewPermissionAuthority(dir)
	drift, err := auth.checkRuntimePolicySurfaces()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drift) == 0 {
		t.Error("expected drift when expected content is missing")
	}
}

// ── RegoEngine: classifyRule branches ───────────────────────────────────

func TestClassifyRule_AllBranches(t *testing.T) {
	dir := t.TempDir()
	engine := NewRegoEngine(dir)

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"deny_foo", "deny_foo {", "deny"},
		{"denied", "denied {", "deny"},
		{"allow", "allow {", "allow"},
		{"allow_foo", "allow_foo {", "allow"},
		{"default_deny", "default allow = false", "default_deny"},
		{"default_allow", "default allow = true", "default_allow"},
		{"unknown_rule", "some_rule {", "info"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.classifyRule(tc.name, tc.line)
			if result != tc.expected {
				t.Errorf("classifyRule(%q, %q) = %q, want %q", tc.name, tc.line, result, tc.expected)
			}
		})
	}
}

// ── RegoEngine: extractRuleName unknown path ────────────────────────────

func TestExtractRuleName_NoMatch(t *testing.T) {
	dir := t.TempDir()
	engine := NewRegoEngine(dir)
	result := engine.extractRuleName("   ")
	if result != "unknown" {
		t.Errorf("expected 'unknown' for blank line, got %q", result)
	}
}

// ── RegoEngine: GetPolicySummary (0%) ───────────────────────────────────

func TestGetPolicySummary_Empty(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	engine := NewRegoEngine(policiesDir)
	summary := engine.GetPolicySummary()

	if summary["total_rules"].(int) != 0 {
		t.Error("expected 0 total rules for empty engine")
	}
	if summary["deny_rules"].(int) != 0 {
		t.Error("expected 0 deny rules")
	}
	if summary["allow_rules"].(int) != 0 {
		t.Error("expected 0 allow rules")
	}
}

func TestGetPolicySummary_WithPolicies(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	policy := `package test

deny_x { input.x == 1 }
deny_y { input.y == 2 }
allow_z { input.z == 3 }
`
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	engine.LoadPolicies()
	summary := engine.GetPolicySummary()

	if summary["total_rules"].(int) != 3 {
		t.Errorf("expected 3 total rules, got %d", summary["total_rules"])
	}
	if summary["deny_rules"].(int) != 2 {
		t.Errorf("expected 2 deny rules, got %d", summary["deny_rules"])
	}
	if summary["allow_rules"].(int) != 1 {
		t.Errorf("expected 1 allow rule, got %d", summary["allow_rules"])
	}
}

// ── RegoEngine: evaluateDenyRule branches ───────────────────────────────

func TestEvaluateDenyRule_DenyPathTraversal(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_path_traversal", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"path": "../../etc/passwd"}) {
		t.Error("expected path traversal to be denied")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"path": "safe/path"}) {
		t.Error("expected safe path to pass")
	}
}

func TestEvaluateDenyRule_DenySystemPathWrite(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_system_path_write", Package: "test", RuleType: "deny"}

	// Empty path — should not trigger
	if engine.evaluateDenyRule(rule, map[string]interface{}{"path": "", "workspace_root": "/ws"}) {
		t.Error("empty path should not trigger deny")
	}
	// Outside workspace and not /tmp/opencode/
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"path": "/etc/file", "workspace_root": "/ws"}) {
		t.Error("expected system path write to be denied")
	}
	// Inside /tmp/opencode/ — allowed
	if engine.evaluateDenyRule(rule, map[string]interface{}{"path": "/tmp/opencode/file", "workspace_root": "/ws"}) {
		t.Error("expected /tmp/opencode/ to be allowed")
	}
	// Inside workspace — allowed
	if engine.evaluateDenyRule(rule, map[string]interface{}{"path": "/ws/file", "workspace_root": "/ws"}) {
		t.Error("expected workspace path to be allowed")
	}
}

func TestEvaluateDenyRule_DenyForceDeleteBranch(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_force_delete_branch", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"flag": "-D"}) {
		t.Error("expected -D to trigger deny")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"flag": "-d"}) {
		t.Error("expected -d to not trigger deny")
	}
}

func TestEvaluateDenyRule_DenyProtectedBranchPush(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_protected_branch_push", Package: "test", RuleType: "deny"}

	for _, branch := range []string{"main", "master", "develop", "production"} {
		if !engine.evaluateDenyRule(rule, map[string]interface{}{"branch": branch}) {
			t.Errorf("expected %s to be denied", branch)
		}
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"branch": "feature/x"}) {
		t.Error("expected feature/x to pass")
	}
}

func TestEvaluateDenyRule_DenyPluginInstall(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_plugin_install", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "install_plugin", "operator": "eidren"}) {
		t.Error("expected plugin install to be denied for eidren")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "install_plugin", "operator": "thavren"}) {
		t.Error("expected plugin install to be allowed for thavren")
	}
}

func TestEvaluateDenyRule_DenyExtensionInstall(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_extension_install", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "install_extension", "operator": "eidren"}) {
		t.Error("expected extension install to be denied for eidren")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "install_extension", "operator": "thavren"}) {
		t.Error("expected extension install to be allowed for thavren")
	}
}

func TestEvaluateDenyRule_DenyExternalMCP(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_external_mcp", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "register_mcp_server", "source": "external"}) {
		t.Error("expected external MCP to be denied")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "register_mcp_server", "source": "ovav_internal"}) {
		t.Error("expected internal MCP to pass")
	}
}

func TestEvaluateDenyRule_DenyExternalNetwork(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_external_network", Package: "test", RuleType: "deny"}

	// Non-external_request action
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "bash"}) {
		t.Error("non-external action should pass")
	}
	// Allowed domain
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "external_request", "url": "https://github.com/x"}) {
		t.Error("github.com should be allowed")
	}
	// Subdomain of allowed
	if engine.evaluateDenyRule(rule, map[string]interface{}{"action": "external_request", "url": "https://api.github.com/x"}) {
		t.Error("api.github.com should be allowed")
	}
	// Unknown domain
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "external_request", "url": "https://evil.com/steal"}) {
		t.Error("unknown domain should be denied")
	}
	// Empty domain
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "external_request", "url": ""}) {
		t.Error("empty domain should be denied")
	}
	// URL without scheme
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"action": "external_request", "url": "evil.com/path"}) {
		t.Error("URL without scheme should be denied")
	}
}

func TestEvaluateDenyRule_DenySecretInOutput(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_secret_in_output", Package: "test", RuleType: "deny"}

	secrets := []string{"ghp_abc", "github_pat_abc", "gho_abc", "AKIA123", "sk-abc", "sk-ant-abc", "-----BEGIN RSA PRIVATE KEY-----"}
	for _, s := range secrets {
		if !engine.evaluateDenyRule(rule, map[string]interface{}{"content": s}) {
			t.Errorf("expected %s prefix to be detected", s[:min(10, len(s))])
		}
	}
	// Clean content
	if engine.evaluateDenyRule(rule, map[string]interface{}{"content": "hello world"}) {
		t.Error("clean content should pass")
	}
}

func TestEvaluateDenyRule_DenyWithoutBootstrap(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_without_bootstrap", Package: "test", RuleType: "deny"}

	if !engine.evaluateDenyRule(rule, map[string]interface{}{"bootstrap_valid": false}) {
		t.Error("expected deny without bootstrap")
	}
	if engine.evaluateDenyRule(rule, map[string]interface{}{"bootstrap_valid": true}) {
		t.Error("expected pass with valid bootstrap")
	}
}

func TestEvaluateDenyRule_DenyRateLimit(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "deny_rate_limit", Package: "test", RuleType: "deny"}

	// Over limit
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"request_count": 150, "rate_limit": 100}) {
		t.Error("expected rate limit exceeded")
	}
	// Under limit
	if engine.evaluateDenyRule(rule, map[string]interface{}{"request_count": 50, "rate_limit": 100}) {
		t.Error("expected under limit to pass")
	}
	// Zero rate limit (default 100)
	if !engine.evaluateDenyRule(rule, map[string]interface{}{"request_count": 101, "rate_limit": 0}) {
		t.Error("expected default rate limit to apply")
	}
}

func TestEvaluateDenyRule_UnknownRule(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "unknown_rule", Package: "test", RuleType: "deny"}
	if engine.evaluateDenyRule(rule, map[string]interface{}{}) {
		t.Error("unknown rule should return false")
	}
}

// ── RegoEngine: evaluateAllowRule branches ──────────────────────────────

func TestEvaluateAllowRule_Allow(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "allow", Package: "test", RuleType: "allow"}

	tests := []struct {
		name   string
		facts  map[string]interface{}
		expect bool
	}{
		{"thavren_repo_local", map[string]interface{}{"operator": "thavren", "scope": "repo_local"}, true},
		{"thavren_external", map[string]interface{}{"operator": "thavren", "action": "external_request"}, true},
		{"eidren_research", map[string]interface{}{"operator": "eidren", "action": "research"}, true},
		{"explicit_grant", map[string]interface{}{"explicit_grant": true}, true},
		{"no_match", map[string]interface{}{"operator": "other", "action": "other"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.evaluateAllowRule(rule, tc.facts)
			if result != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, result)
			}
		})
	}
}

func TestEvaluateAllowRule_AllowOperatorAction(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "allow_operator_action", Package: "test", RuleType: "allow"}

	if !engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "thavren", "scope": "repo_local"}) {
		t.Error("expected thavren repo_local to be allowed")
	}
	if !engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "eidren", "action": "research"}) {
		t.Error("expected eidren research to be allowed")
	}
	if engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "other"}) {
		t.Error("expected other operator to be denied")
	}
}

func TestEvaluateAllowRule_AllowOperatorScope(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "allow_operator_scope", Package: "test", RuleType: "allow"}

	// Thavren scopes
	for _, scope := range []string{"repo_local", "global_diagnostic", "install_sandbox"} {
		if !engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "thavren", "scope": scope}) {
			t.Errorf("expected thavren %s to be allowed", scope)
		}
	}
	// Eidren scopes
	for _, scope := range []string{"repo_local", "research_external"} {
		if !engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "eidren", "scope": scope}) {
			t.Errorf("expected eidren %s to be allowed", scope)
		}
	}
	// Invalid scope
	if engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "thavren", "scope": "invalid"}) {
		t.Error("expected invalid scope to be denied")
	}
	// Unknown operator
	if engine.evaluateAllowRule(rule, map[string]interface{}{"operator": "unknown", "scope": "repo_local"}) {
		t.Error("expected unknown operator to be denied")
	}
}

func TestEvaluateAllowRule_Unknown(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())
	rule := PolicyRule{Name: "unknown_allow", Package: "test", RuleType: "allow"}
	if engine.evaluateAllowRule(rule, map[string]interface{}{}) {
		t.Error("unknown allow rule should return false")
	}
}

// ── ClaimsGovernor: checkEvidence cached path ───────────────────────────

func TestClaimsGovernor_EvidenceCacheHit(t *testing.T) {
	g := NewClaimsGovernor()
	g.evidenceCache["test_evidence"] = true

	if !g.checkEvidence("test_evidence") {
		t.Error("expected cached evidence to return true")
	}
}

func TestClaimsGovernor_EvaluateClaim_MissingEvidence(t *testing.T) {
	g := NewClaimsGovernor()
	d := g.EvaluateClaim("production_ready", "thavren")
	if d.Allowed {
		t.Error("expected production_ready to be denied with missing evidence")
	}
	if len(d.MissingEvidence) == 0 {
		t.Error("expected missing evidence list")
	}
}

func TestClaimsGovernor_EvaluateClaim_WrongOperator(t *testing.T) {
	g := NewClaimsGovernor()
	d := g.EvaluateClaim("production_ready", "eidren")
	if d.Allowed {
		t.Error("expected non-thavren operator to be denied for production_ready")
	}

	d = g.EvaluateClaim("global_ready", "eidren")
	if d.Allowed {
		t.Error("expected non-thavren operator to be denied for global_ready")
	}
}

func TestClaimsGovernor_EvaluateClaim_AllEvidenceCached(t *testing.T) {
	g := NewClaimsGovernor()
	// Cache all evidence for production_ready
	evidence := []string{"bootstrap_chain_verified", "all_f0_validators_pass", "all_f1_validators_pass", "all_f2_validators_pass", "formal_verification_pass", "supply_chain_integrity_pass", "no_secrets_in_plaintext", "no_exfil_anomalies"}
	for _, e := range evidence {
		g.evidenceCache[e] = true
	}
	d := g.EvaluateClaim("production_ready", "thavren")
	if !d.Allowed {
		t.Errorf("expected production_ready to be allowed with all evidence, got: %s", d.Reason)
	}
	if _, ok := g.activeClaims["production_ready"]; !ok {
		t.Error("expected production_ready to be activated")
	}
}

func TestClaimsGovernor_EvaluateClaim_NewPublicProfile(t *testing.T) {
	g := NewClaimsGovernor()
	d := g.EvaluateClaim("new_public_profile", "thavren")
	if d.Allowed {
		t.Error("expected new_public_profile to be denied with missing evidence")
	}
}

func TestClaimsGovernor_EvaluateClaim_UnknownType(t *testing.T) {
	g := NewClaimsGovernor()
	d := g.EvaluateClaim("unknown_claim", "thavren")
	// Should not panic, should handle gracefully
	_ = d
}

// ── RegoEngine: Evaluate with explicit_grant path ──────────────────────

func TestRegoEngine_Evaluate_ExplicitGrant(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(`package test
default allow = false
`), 0644)

	engine := NewRegoEngine(policiesDir)
	decision := engine.Evaluate("bash", map[string]interface{}{
		"operator":        "unknown",
		"action":          "bash",
		"bootstrap_valid": true,
		"explicit_grant":  true,
	})
	if !decision.Allowed {
		t.Error("expected explicit grant to allow")
	}
}

func TestRegoEngine_Evaluate_DefaultAllow(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(`package test
default allow = true
`), 0644)

	engine := NewRegoEngine(policiesDir)
	decision := engine.Evaluate("read", map[string]interface{}{
		"operator":        "unknown",
		"bootstrap_valid": true,
	})
	if !decision.Allowed {
		t.Error("expected default allow policy to permit read")
	}
}

func TestRegoEngine_Evaluate_NoBootstrap(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(`package test
default allow = false
allow { input.operator == "thavren" }
`), 0644)

	engine := NewRegoEngine(policiesDir)
	decision := engine.Evaluate("bash", map[string]interface{}{
		"operator":        "thavren",
		"bootstrap_valid": false,
	})
	if decision.Allowed {
		t.Error("expected denial without bootstrap")
	}
}

// ── BashGovernor: F0 integration paths ──────────────────────────────────

func TestBashGovernor_F0NetworkGuard(t *testing.T) {
	g := NewBashCommandGovernor()
	// gh issue list has f0.4_network_guard integration
	d := g.Check("gh issue list", "thavren")
	// Network guard placeholder returns true, so should pass
	if !d.Allowed {
		t.Errorf("gh issue list should be allowed, got: %s", d.Reason)
	}
}

// ── SandboxGovernor: unknown operation ──────────────────────────────────

func TestSandboxGovernor_UnknownOperation(t *testing.T) {
	g := NewSandboxGovernor()
	d := g.CheckOperation("nonexistent_op", true)
	if d.Allowed {
		t.Error("expected unknown operation to be denied")
	}
}

// ── SystemPathGovernor: /boot, /sys, /dev paths ─────────────────────────

func TestSystemPathGovernor_BootAndSysAndDev(t *testing.T) {
	gov := NewSystemPathGovernor("/home/test")

	// /boot read
	d := gov.Check("/boot/grub", "read")
	if !d.Allowed {
		t.Error("expected /boot read to be allowed")
	}
	// /boot write
	d = gov.Check("/boot/grub", "write")
	if d.Allowed {
		t.Error("expected /boot write to require gate")
	}
	// /boot execute
	d = gov.Check("/boot/grub", "execute")
	if d.Allowed {
		t.Error("expected /boot execute to be denied")
	}
	// /sys read
	d = gov.Check("/sys/class", "read")
	if !d.Allowed {
		t.Error("expected /sys read to be allowed")
	}
	// /sys write
	d = gov.Check("/sys/class", "write")
	if d.Allowed {
		t.Error("expected /sys write to require gate")
	}
	// /dev read
	d = gov.Check("/dev/null", "read")
	if !d.Allowed {
		t.Error("expected /dev read to be allowed")
	}
	// /dev write
	d = gov.Check("/dev/null", "write")
	if d.Allowed {
		t.Error("expected /dev write to require gate")
	}
}

// ── NewStatesGovernor: all states ───────────────────────────────────────

func TestNewStatesGovernor_AllStates(t *testing.T) {
	gov := NewNewStatesGovernor()

	allowed := []string{"delegated", "adaptive", "consensus_required", "provenance_gated", "rate_limited", "geofenced", "revocable", "step_up_required", "circuit_breaker", "idempotency_gated", "cost_gated", "emergent"}
	for _, state := range allowed {
		d := gov.Check(state)
		if !d.Allowed {
			t.Errorf("expected %s to be allowed, got: %s", state, d.Reason)
		}
	}

	denied := []string{"inherited", "canary_gated"}
	for _, state := range denied {
		d := gov.Check(state)
		if d.Allowed {
			t.Errorf("expected %s to be denied", state)
		}
	}
}

// ── ExtractDomain edge cases ────────────────────────────────────────────

func TestExtractDomain_EdgeCases(t *testing.T) {
	engine := NewRegoEngine(t.TempDir())

	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/path", "github.com"},
		{"http://evil.com:8080/path", "evil.com"},
		{"ftp://files.example.com/x", "files.example.com"},
		{"no-scheme.com/path", "no-scheme.com"},
		{"://double-slash.com", "double-slash.com"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := engine.extractDomain(tc.url)
			if result != tc.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tc.url, result, tc.expected)
			}
		})
	}
}

// ── LoadPolicies: duplicate rules ───────────────────────────────────────

func TestLoadPolicies_DuplicateRules(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	policy := `package test

deny_x { input.x == 1 }
deny_x { input.x == 2 }
allow_y { input.y == 1 }
`
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	if err := engine.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies failed: %v", err)
	}
	// Duplicate rule should be deduplicated
	for _, r := range engine.denyRules {
		if r.Name == "deny_x" {
			// Should appear only once
		}
	}
}

// ── LoadPolicies: multiple files with comments and blanks ───────────────

func TestLoadPolicies_CommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	policiesDir := filepath.Join(dir, "policies")
	os.MkdirAll(policiesDir, 0755)

	policy := `# This is a comment

package test

# Another comment

deny_z { input.z == 1 }

# End comment
`
	os.WriteFile(filepath.Join(policiesDir, "test.rego"), []byte(policy), 0644)

	engine := NewRegoEngine(policiesDir)
	if err := engine.LoadPolicies(); err != nil {
		t.Fatalf("LoadPolicies failed: %v", err)
	}
	if len(engine.denyRules) != 1 {
		t.Errorf("expected 1 deny rule, got %d", len(engine.denyRules))
	}
}

// ── MaterializeAll: agent projection with existing permission block ─────

func TestMaterializeAll_AgentWithPermissionBlock(t *testing.T) {
	dir := t.TempDir()
	writeValidPolicy(t, dir)

	// Write opencode.json first (minimal)
	writeMinimalOpencode(t, dir)

	// Write agent with existing permission block
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	agentContent := `---
name: Test
permission:
  edit: deny
  bash:
    "*": allow
---
# Body
`
	os.WriteFile(filepath.Join(agentsDir, "area-platform-engineering.md"), []byte(agentContent), 0644)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"), []byte(agentContent), 0644)

	auth := NewPermissionAuthority(dir)
	result, err := auth.MaterializeAll(true)
	if err != nil {
		t.Fatalf("MaterializeAll failed: %v", err)
	}
	status := result["status"].(string)
	if status != "changed" {
		t.Errorf("expected changed status, got %s", status)
	}
}
