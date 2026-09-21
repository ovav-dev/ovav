package validators

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelPolicyRejectsUnauthorizedProjectedOpenCodeAgent(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		".ovav/policy/permission_authority.json": `{"model_groups":{"standard":{"models":["openai/gpt-5.6-luna","minimax-coding-plan/MiniMax-M3"]}}}`,
		"opencode.json":                          `{"model":"openai/gpt-5.6-luna"}`,
		"go-runtime/internal/runtimes/opencode/agents/team-retired.md": "---\nmodel: openai/retired-model\n---\n",
	})

	result := NewModelPolicy().Validate(context.Background(), root)
	if result.Status != "fail" {
		t.Fatalf("expected projected agent model drift to fail, got %s: %v", result.Status, result.Issues)
	}
	if len(result.Issues) != 1 || !strings.Contains(result.Issues[0], "go-runtime/internal/runtimes/opencode/agents/team-retired.md") {
		t.Fatalf("expected projected agent path in issue, got %v", result.Issues)
	}
}

func TestProjectedOpenCodeAgentModelsAreAuthorized(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	authorized, err := loadAuthorizedModels(root)
	if err != nil {
		t.Fatal(err)
	}

	agentsDir := filepath.Join(root, "go-runtime", "internal", "runtimes", "opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("read projected OpenCode agents: %v", err)
	}

	modelsChecked := 0
	policy := NewModelPolicy()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		model := policy.parseFrontmatterModel(filepath.Join(agentsDir, entry.Name()))
		if model == "" {
			continue
		}
		modelsChecked++
		if !authorized[model] {
			t.Errorf("%s uses unauthorized model %q", entry.Name(), model)
		}
	}
	if modelsChecked == 0 {
		t.Fatal("expected projected OpenCode agents with explicit models")
	}
}
