package convert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ovav/ovav/internal/permissions"
)

func TestValidateOpenCodeConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":       "https://opencode.ai/config.json",
		"model":         "openai/gpt-5.6-luna",
		"instructions":  []string{"AGENTS.md"},
		"default_agent": "Platform Engineering",
		"agent":         map[string]any{},
		"mcp": map[string]any{
			"ovav-budget": map[string]any{
				"type":    "local",
				"command": []any{"python3", "tools/mcp/ovav_mcp_server.py", "budget"},
				"enabled": true,
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) > 0 {
		for _, i := range issues {
			t.Errorf("unexpected issue: [%s] %s: %s", i.Severity, i.Field, i.Message)
		}
	}
}

func TestValidateOpenCodeConfig_LegacyMCPFormat(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
		"agent":        map[string]any{},
		"mcp": map[string]any{
			"ovav-budget": map[string]any{
				"command":     "python3",
				"args":        []any{"tools/mcp/ovav_mcp_server.py", "budget"},
				"description": "OVAV Budget tracking",
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Severity == "critical" && issue.Field == "mcp.ovav-budget" {
			found = true
			if !contains(issue.Message, "FORMATO OBSOLETO") {
				t.Errorf("expected OBSOLETO message, got: %s", issue.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected critical issue for legacy MCP format, got %d issues", len(issues))
	}
}

func TestValidateOpenCodeConfig_AgentSectionHasOVAVAgent(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
		"agent": map[string]any{
			"Platform Engineering": map[string]any{
				"description": "OVAV platform agent",
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Severity == "critical" && issue.Field == "agent.Platform Engineering" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected critical issue for OVAV agent in agent section, got %d issues", len(issues))
	}
}

func TestValidateOpenCodeConfig_MissingSchema(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.Field == "$schema" && issue.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected critical issue for missing $schema")
	}
}

func TestSyncOpenCodeConfig_LegacyFix(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
		"mcp": map[string]any{
			"ovav-budget": map[string]any{
				"command":     "python3",
				"args":        []any{"tools/mcp/ovav_mcp_server.py", "budget"},
				"description": "OVAV Budget tracking",
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	fixed, err := SyncOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fixed {
		t.Fatal("expected fix to be applied")
	}

	// Re-read and verify
	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("re-validation failed: %v", err)
	}

	criticals := 0
	for _, issue := range issues {
		if issue.Severity == "critical" {
			criticals++
			t.Errorf("post-fix critical: [%s] %s", issue.Field, issue.Message)
		}
	}
	if criticals > 0 {
		t.Fatalf("expected 0 critical issues after fix, got %d", criticals)
	}

	// Verify the fixed MCp entry structure
	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatalf("read fixed config: %v", err)
	}
	var fixedConfig map[string]any
	if err := json.Unmarshal(data, &fixedConfig); err != nil {
		t.Fatalf("parse fixed config: %v", err)
	}
	mcp := fixedConfig["mcp"].(map[string]any)
	server := mcp["ovav-budget"].(map[string]any)
	if server["type"] != "local" {
		t.Errorf("expected type=local, got %v", server["type"])
	}
	if _, ok := server["command"].([]any); !ok {
		t.Errorf("expected command to be array")
	}
	if server["enabled"] != true {
		t.Errorf("expected enabled=true")
	}
}

func TestSyncOpenCodeConfig_AlreadyValid(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
		"mcp": map[string]any{
			"ovav-budget": map[string]any{
				"type":    "local",
				"command": []any{"python3", "tools/mcp/ovav_mcp_server.py", "budget"},
				"enabled": true,
			},
		},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	fixed, err := SyncOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fixed {
		t.Error("expected no fix for already-valid config")
	}
}

func TestValidateOpenCodeConfig_NoMCP(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
	}
	writeJSON(t, filepath.Join(dir, "opencode.json"), config)

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, issue := range issues {
		if issue.Field != "$schema" && issue.Severity != "warning" {
			t.Errorf("unexpected non-warning issue with no MCP: [%s] %s", issue.Field, issue.Message)
		}
	}
}

func TestValidateOpenCodeConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ValidateOpenCodeConfig(dir)
	if err == nil {
		t.Fatal("expected error for missing opencode.json")
	}
}

func TestValidateOpenCodeConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("write invalid JSON: %v", err)
	}

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("expected issues not error for invalid JSON, got: %v", err)
	}
	if len(issues) == 0 || issues[0].Severity != "critical" {
		t.Errorf("expected critical issue for invalid JSON")
	}
}

func TestSyncOpenCodeConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("write invalid JSON: %v", err)
	}

	_, err := SyncOpenCodeConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── Generation Tests ─────────────────────────────────────────────────────

func TestGenerateOpenCodeConfig_FullGeneration(t *testing.T) {
	dir := t.TempDir()

	// Create canonical source
	sourceDir := filepath.Join(dir, ".ovav", "source", "opencode")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	canonical := `version: "1.0"
schema: "https://opencode.ai/config.json"
runtime:
  model: "openai/gpt-5.6-luna"
  small_model: "minimax-coding-plan/MiniMax-M3"
  default_agent: "Platform Engineering"
  instructions:
    - "AGENTS.md"
  agent: {}
mcp:
  test-server:
    type: local
    command: ["python3", "test.py"]
    enabled: true
plugins:
  - ".opencode/plugins/test.js"
providers:
  openai:
    options:
      timeout: 120000
permissions:
  edit: allow
  bash:
    "*": allow
    "rm -rf *": deny
  external_directory:
    "/tmp/*": allow
references:
  spec:
    path: docs
    description: "Test reference"
compaction:
  auto: true
  preserve_recent_tokens: 2000
tool_output:
  max_bytes: 4096
user:
  username: "Test User"
`
	if err := os.WriteFile(filepath.Join(sourceDir, "config.yaml"), []byte(canonical), 0644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}

	if err := GenerateOpenCodeConfig(dir); err != nil {
		t.Fatalf("GenerateOpenCodeConfig: %v", err)
	}

	// Verify generated output
	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatalf("read generated: %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse generated: %v", err)
	}

	// Verify key sections
	if config["$schema"] != "https://opencode.ai/config.json" {
		t.Errorf("schema mismatch: %v", config["$schema"])
	}
	if config["model"] != "openai/gpt-5.6-luna" {
		t.Errorf("model mismatch")
	}
	if config["username"] != "Test User" {
		t.Errorf("username mismatch: %v", config["username"])
	}

	mcp := config["mcp"].(map[string]any)
	server := mcp["test-server"].(map[string]any)
	if server["type"] != "local" {
		t.Errorf("mcp type: %v", server["type"])
	}
	if server["enabled"] != true {
		t.Errorf("mcp enabled: %v", server["enabled"])
	}
}

func TestGenerateOpenCodeConfig_OverwriteIdempotent(t *testing.T) {
	dir := t.TempDir()

	sourceDir := filepath.Join(dir, ".ovav", "source", "opencode")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	canonical := `version: "1.0"
schema: "https://opencode.ai/config.json"
runtime:
  model: "openai/gpt-5.6-luna"
  small_model: "minimax-coding-plan/MiniMax-M3"
  default_agent: "Test"
  instructions: ["TEST.md"]
  agent: {}
user:
  username: "Test"
`
	if err := os.WriteFile(filepath.Join(sourceDir, "config.yaml"), []byte(canonical), 0644); err != nil {
		t.Fatalf("write canonical: %v", err)
	}

	// First generation
	if err := GenerateOpenCodeConfig(dir); err != nil {
		t.Fatalf("first gen: %v", err)
	}

	// Write garbage to opencode.json (simulate bad manual edit)
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("garbage"), 0644); err != nil {
		t.Fatalf("write garbage: %v", err)
	}

	// Second generation — should overwrite garbage
	if err := GenerateOpenCodeConfig(dir); err != nil {
		t.Fatalf("second gen (overwrite): %v", err)
	}

	// Verify it's valid JSON again
	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatalf("read after overwrite: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse after overwrite: %v", err)
	}
	if config["model"] != "openai/gpt-5.6-luna" {
		t.Errorf("model lost after overwrite: %v", config["model"])
	}
}

func TestGenerateOpenCodeConfig_MissingSource(t *testing.T) {
	dir := t.TempDir()
	err := GenerateOpenCodeConfig(dir)
	if err == nil {
		t.Fatal("expected error for missing canonical source")
	}
}

func TestGenerateOpenCodeConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, ".ovav", "source", "opencode")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "config.yaml"), []byte(": invalid yaml:"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := GenerateOpenCodeConfig(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestGenerateOpenCodeConfig_UsesSupportedSchemaSurface(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, ".ovav", "source", "opencode")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	canonical := `version: "1.0"
schema: "https://opencode.ai/config.json"
runtime:
  model: "openai/gpt-5.6-luna"
  small_model: "minimax-coding-plan/MiniMax-M3"
  default_agent: "Platform Engineering"
  instructions: ["AGENTS.md"]
  agent: {}
mcp:
  broken-server:
    enabled: false
plugins:
  - ".opencode/plugins/ovav-monitor.js"
permissions:
  edit: allow
  bash:
    "*": allow
    "git push*": deny
    "gh pr create*": allow
`
	if err := os.WriteFile(filepath.Join(sourceDir, "config.yaml"), []byte(canonical), 0o644); err != nil {
		t.Fatalf("write canonical config: %v", err)
	}

	if err := GenerateOpenCodeConfig(dir); err != nil {
		t.Fatalf("GenerateOpenCodeConfig() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse generated config: %v", err)
	}
	if _, exists := config["theme"]; exists {
		t.Error("generated config contains unsupported top-level theme")
	}
	if config["model"] != "openai/gpt-5.6-luna" {
		t.Errorf("model = %v, want openai/gpt-5.6-luna", config["model"])
	}
	if config["small_model"] != "minimax-coding-plan/MiniMax-M3" {
		t.Errorf("small_model = %v, want minimax-coding-plan/MiniMax-M3", config["small_model"])
	}
	server := config["mcp"].(map[string]any)["broken-server"].(map[string]any)
	if len(server) != 1 || server["enabled"] != false {
		t.Errorf("disabled MCP projection = %#v, want enabled-only false entry", server)
	}
	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("ValidateOpenCodeConfig() error = %v", err)
	}
	for _, issue := range issues {
		if issue.Severity == "critical" {
			t.Errorf("generated config critical issue: %s: %s", issue.Field, issue.Message)
		}
	}
}

func TestValidateOpenCodeConfig_RejectsUnsupportedTopLevelTheme(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "opencode.json"), map[string]any{
		"$schema":      "https://opencode.ai/config.json",
		"model":        "openai/gpt-5.6-luna",
		"instructions": []string{"AGENTS.md"},
		"theme": map[string]any{
			"name": "ovav",
			"path": ".opencode/themes/ovav-dark.json",
		},
	})

	issues, err := ValidateOpenCodeConfig(dir)
	if err != nil {
		t.Fatalf("ValidateOpenCodeConfig() error = %v", err)
	}
	for _, issue := range issues {
		if issue.Field == "theme" && issue.Severity == "critical" {
			return
		}
	}
	t.Fatal("expected critical issue for unsupported top-level theme")
}

func TestValidateOpenCodeConfig_RejectsUnavailableModels(t *testing.T) {
	tests := []struct {
		name  string
		field string
		model string
	}{
		{name: "unsupported primary", field: "model", model: "openai/retired-model"},
		{name: "unsupported small", field: "small_model", model: "minimax-coding-plan/retired-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			config := map[string]any{
				"$schema":      "https://opencode.ai/config.json",
				"model":        "openai/gpt-5.6-luna",
				"small_model":  "minimax-coding-plan/MiniMax-M3",
				"instructions": []string{"AGENTS.md"},
			}
			config[tt.field] = tt.model
			writeJSON(t, filepath.Join(dir, "opencode.json"), config)

			issues, err := ValidateOpenCodeConfig(dir)
			if err != nil {
				t.Fatalf("ValidateOpenCodeConfig() error = %v", err)
			}
			for _, issue := range issues {
				if issue.Field == tt.field && issue.Severity == "critical" {
					return
				}
			}
			t.Fatalf("expected critical issue for unavailable %s %q", tt.field, tt.model)
		})
	}
}

func TestCanonicalOpenCodeConfigMatchesMaterializedTarget(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	issues, err := ValidateOpenCodeConfig(repoRoot)
	if err != nil {
		t.Fatalf("validate materialized config: %v", err)
	}
	for _, issue := range issues {
		if issue.Severity == "critical" {
			t.Errorf("materialized config critical issue: %s: %s", issue.Field, issue.Message)
		}
	}

	tempRoot := t.TempDir()
	tempSourceDir := filepath.Join(tempRoot, ".ovav", "source", "opencode")
	if err := os.MkdirAll(tempSourceDir, 0o755); err != nil {
		t.Fatalf("create temp source directory: %v", err)
	}
	source, err := os.ReadFile(filepath.Join(repoRoot, ".ovav", "source", "opencode", "config.yaml"))
	if err != nil {
		t.Fatalf("read canonical source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempSourceDir, "config.yaml"), source, 0o644); err != nil {
		t.Fatalf("write temp canonical source: %v", err)
	}
	policyDir := filepath.Join(tempRoot, ".ovav", "policy")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("create temp policy directory: %v", err)
	}
	policy, err := os.ReadFile(filepath.Join(repoRoot, ".ovav", "policy", "permission_authority.json"))
	if err != nil {
		t.Fatalf("read canonical permission authority: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "permission_authority.json"), policy, 0o644); err != nil {
		t.Fatalf("write temp permission authority: %v", err)
	}
	if err := GenerateOpenCodeConfig(tempRoot); err != nil {
		t.Fatalf("generate temp config: %v", err)
	}

	generated, err := os.ReadFile(filepath.Join(tempRoot, "opencode.json"))
	if err != nil {
		t.Fatalf("read generated temp config: %v", err)
	}
	materialized, err := os.ReadFile(filepath.Join(repoRoot, "opencode.json"))
	if err != nil {
		t.Fatalf("read materialized config: %v", err)
	}
	var generatedConfig map[string]any
	if err := json.Unmarshal(generated, &generatedConfig); err != nil {
		t.Fatalf("parse generated temp config: %v", err)
	}
	var materializedConfig map[string]any
	if err := json.Unmarshal(materialized, &materializedConfig); err != nil {
		t.Fatalf("parse materialized config: %v", err)
	}
	if !reflect.DeepEqual(generatedConfig, materializedConfig) {
		t.Error("opencode.json differs from canonical generation")
	}
}

func TestEffectiveOpenCodeAgentPermissionsPreserveCanonicalProtectedDenies(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	authority := permissions.NewPermissionAuthority(repoRoot)
	host, err := authority.MaterializePermissionBlock()
	if err != nil {
		t.Fatal(err)
	}
	areas, leads, teams, err := LoadCanonicalAgents(filepath.Join(repoRoot, "go-runtime", "internal", "agents"))
	if err != nil {
		t.Fatal(err)
	}

	type projectedAgent struct {
		name       string
		permission *PermissionBlock
	}
	agents := make([]projectedAgent, 0, len(areas)+len(leads)+len(teams))
	for _, area := range areas {
		agents = append(agents, projectedAgent{area.ID, area.Permission})
	}
	for _, lead := range leads {
		agents = append(agents, projectedAgent{lead.ID, lead.Permission})
	}
	for _, team := range teams {
		agents = append(agents, projectedAgent{team.ID, team.Permission})
	}

	for _, agent := range agents {
		effective := mergeOpenCodeProtectedDenies(agent.permission, host)
		if effective == nil {
			continue // No frontmatter override: the top-level policy remains authoritative.
		}
		for pattern, decision := range host.Bash {
			if decision == "deny" && effective.Bash[pattern] != decision {
				t.Errorf("%s effective bash permission loses host deny %q", agent.name, pattern)
			}
		}
		for pattern, decision := range host.ExternalDirectory {
			if decision == "deny" && effective.ExternalDirectory[pattern] != decision {
				t.Errorf("%s effective external-directory permission loses host deny %q", agent.name, pattern)
			}
		}
	}
}
