package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

// Authorized models — must use opencode-go provider exclusively.
var authorizedModels = map[string]bool{
	"opencode-go/deepseek-v4-pro":   true,
	"opencode-go/qwen3.7-max":       true,
	"opencode-go/minimax-m3":        true,
	"opencode-go/glm-5.1":           true,
	"opencode-go/qwen3.7-plus":      true,
	"opencode-go/glm-5":             true,
	"opencode-go/minimax-m2.7":      true,
	"opencode-go/kimi-k2.6":         true,
	"opencode-go/mimo-v2.5-pro":     true,
	"opencode-go/kimi-k2.5":         true,
	"opencode-go/minimax-m2.5":      true,
	"opencode-go/deepseek-v4-flash": true,
	"opencode-go/qwen3.6-plus":      true,
	"opencode-go/mimo-v2.5":         true,
}

// Forbidden model patterns.
var forbiddenPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"openai_model", regexp.MustCompile(`(?i)\bopenai/(?:gpt|o1|o3|o4)`)},
	{"anthropic_claude", regexp.MustCompile(`(?i)\bclaude\b`)},
	{"google_gemini", regexp.MustCompile(`(?i)\bgoogle\b|\bgemini\b`)},
	{"openrouter", regexp.MustCompile(`(?i)\bopenrouter\b`)},
	{"free_suffix", regexp.MustCompile(`(?i)\b\w+-free\b`)},
	{"openai_o1", regexp.MustCompile(`(?i)\bopenai/o1\b`)},
	{"opencodego_raw", regexp.MustCompile(`(?i)\bopencodego\b`)},
	{"gpt_variants", regexp.MustCompile(`(?i)\bgpt-5\.\d`)},
}

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

	// 1. Scan opencode.json for models
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		var config map[string]interface{}
		if json.Unmarshal(data, &config) == nil {
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
	}

	// 2. Scan agent markdown files for frontmatter models
	agentsDir := filepath.Join(root, ".opencode", "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			fullPath := filepath.Join(agentsDir, e.Name())
			model := m.parseFrontmatterModel(fullPath)
			if model != "" && !authorizedModels[model] {
				issues = append(issues, fmt.Sprintf("unauthorized model '%s' in %s frontmatter", model, e.Name()))
			}
		}
	}

	// 3. Scan model_body_ladder.yaml for authorized models
	ladderPath := filepath.Join(root, ".ovav", "service_areas", "platform_engineering", "model_body_ladder.yaml")
	if data, err := os.ReadFile(ladderPath); err == nil {
		var ladder map[string]interface{}
		if yaml.Unmarshal(data, &ladder) == nil {
			// Verify ladder structure — per-agent model assignments with primary/secondary/tertiary/fallback
			rootSection, ok := ladder["model_body_ladder"].(map[string]interface{})
			if !ok {
				issues = append(issues, "model_body_ladder.yaml missing 'model_body_ladder' root key")
			} else {
				// Verify at least one agent has model assignments
				hasAgent := false
				for _, v := range rootSection {
					if agentMap, ok := v.(map[string]interface{}); ok {
						if _, ok := agentMap["primary"]; ok {
							hasAgent = true
							break
						}
					}
				}
				if !hasAgent {
					issues = append(issues, "model_body_ladder.yaml has no agent model assignments with 'primary' key")
				}
			}
		}
	} else {
		issues = append(issues, "model_body_ladder.yaml not found — model authority source missing")
	}

	// 4. Scan for forbidden model references in config files
	for _, pat := range forbiddenPatterns {
		// Scan opencode.json
		if data, err := os.ReadFile(ocPath); err == nil {
			if pat.re.Match(data) {
				issues = append(issues, fmt.Sprintf("forbidden model reference '%s' in opencode.json", pat.name))
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

var _ Validator = (*ModelPolicy)(nil)
