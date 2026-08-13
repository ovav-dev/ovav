package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelPolicy validates that only authorized models are used and forbidden models are detected.
// Replaces: validate_model_policy.py
type ModelPolicy struct{}

func NewModelPolicy() *ModelPolicy { return &ModelPolicy{} }

func (m *ModelPolicy) ID() string   { return "model_policy" }
func (m *ModelPolicy) Name() string { return "Model Policy Validator" }
func (m *ModelPolicy) Description() string {
	return "Validates authorized model list and detects forbidden model references"
}
func (m *ModelPolicy) Weight() int { return 15 }

func (m *ModelPolicy) parseFrontmatterModel(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			return ""
		}
		if strings.HasPrefix(line, "model:") {
			model := strings.TrimPrefix(line, "model:")
			model = strings.TrimSpace(model)
			model = strings.Trim(model, `"'`)
			return model
		}
	}
	return ""
}

func (m *ModelPolicy) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	authorizedModels, err := loadAuthorizedModels(root)
	if err != nil {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message: "FAIL model policy — canonical model authority unavailable",
			Issues:  []string{err.Error()}, Duration: time.Since(start),
		}
	}

	// 1. Scan opencode.json for models
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			issues = append(issues, fmt.Sprintf("opencode.json is invalid JSON: %v", err))
		} else {
			// Check top-level model
			if model, ok := config["model"].(string); ok {
				if !authorizedModels[model] {
					issues = append(issues, fmt.Sprintf("unauthorized model '%s' at opencode.json:model", model))
				}
			}
			if model, ok := config["small_model"].(string); ok {
				if !authorizedModels[model] {
					issues = append(issues, fmt.Sprintf("unauthorized model '%s' at opencode.json:small_model", model))
				}
			}
			// Check agent models
			if agents, ok := config["agent"].(map[string]interface{}); ok {
				for name, ac := range agents {
					if acMap, ok := ac.(map[string]interface{}); ok {
						if model, ok := acMap["model"].(string); ok {
							if !authorizedModels[model] {
								issues = append(issues, fmt.Sprintf("unauthorized model '%s' at opencode.json:agent.%s.model", model, name))
							}
						}
					}
				}
			}
		}
	} else {
		issues = append(issues, fmt.Sprintf("cannot read opencode.json: %v", err))
	}

	// 2. Scan the generated OpenCode projection and its client-facing surface.
	agentDirs := []string{
		filepath.Join("go-runtime", "internal", "runtimes", "opencode", "agents"),
		filepath.Join(".opencode", "agents"),
	}
	for _, relativeDir := range agentDirs {
		entries, err := os.ReadDir(filepath.Join(root, relativeDir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			fullPath := filepath.Join(root, relativeDir, e.Name())
			model := m.parseFrontmatterModel(fullPath)
			if model != "" && !authorizedModels[model] {
				relativePath := filepath.ToSlash(filepath.Join(relativeDir, e.Name()))
				issues = append(issues, fmt.Sprintf("unauthorized model '%s' in %s frontmatter", model, relativePath))
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: m.ID(), Name: m.Name(), Status: "fail", Weight: m.Weight(),
			Message:  fmt.Sprintf("FAIL model policy — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: m.ID(), Name: m.Name(), Status: "pass", Weight: m.Weight(),
		Message:  "PASS model policy — all models authorized",
		Duration: time.Since(start),
	}
}

func loadAuthorizedModels(root string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(root, ".ovav", "policy", "permission_authority.json"))
	if err != nil {
		return nil, fmt.Errorf("cannot read canonical permission_authority.json: %w", err)
	}
	var authority struct {
		ModelGroups map[string]struct {
			Models []string `json:"models"`
		} `json:"model_groups"`
	}
	if err := json.Unmarshal(data, &authority); err != nil {
		return nil, fmt.Errorf("canonical permission_authority.json is invalid: %w", err)
	}
	models := make(map[string]bool)
	for _, group := range authority.ModelGroups {
		for _, model := range group.Models {
			if strings.TrimSpace(model) != "" {
				models[model] = true
			}
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("canonical permission_authority.json has no authorized model_groups models")
	}
	return models, nil
}

var _ Validator = (*ModelPolicy)(nil)
