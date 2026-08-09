package convert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateOpenCodeConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	config := map[string]any{
		"$schema":       "https://opencode.ai/config.json",
		"model":         "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
		"model":        "opencode-go/deepseek-v4-pro",
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
  model: "opencode-go/deepseek-v4-pro"
  small_model: "opencode-go/deepseek-v4-flash"
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
	if config["model"] != "opencode-go/deepseek-v4-pro" {
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
  model: "opencode-go/deepseek-v4-pro"
  small_model: "opencode-go/deepseek-v4-flash"
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
	if config["model"] != "opencode-go/deepseek-v4-pro" {
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
