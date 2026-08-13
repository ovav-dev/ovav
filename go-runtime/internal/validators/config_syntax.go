package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigSyntax validates YAML and JSON syntax across all configuration surfaces.
// Prevents silent corruption of config files that would break parsers.
// Replaces: check_config_syntax.py
type ConfigSyntax struct{}

func NewConfigSyntax() *ConfigSyntax { return &ConfigSyntax{} }

func (c *ConfigSyntax) ID() string   { return "config_syntax" }
func (c *ConfigSyntax) Name() string { return "Config Syntax" }
func (c *ConfigSyntax) Description() string {
	return "Validates YAML and JSON syntax across all config surfaces"
}
func (c *ConfigSyntax) Weight() int { return 16 }

// Directories to scan for config files (relative to repo root).
var scanDirs = []string{
	".ovav",
	".opencode",
	"schemas",
}

// Exclusion patterns for paths that shouldn't be validated.
var excludePatterns = []string{
	"__pycache__",
	".git",
	"node_modules",
	".pytest_cache",
	"integrity_backups",
	"/artifacts/",
	// .ovav/worktrees/ contains feature-isolation branches that may include
	// in-progress or malformed config files (e.g. JS-style comments in JSON,
	// WIP schema drafts). Worktrees are an isolation mechanism, not source —
	// they must not trigger validator failures on the parent repo.
	".ovav/worktrees/",
}

// File extensions to validate.
var yamlExts = map[string]bool{".yaml": true, ".yml": true}
var jsonExts = map[string]bool{".json": true}

func (c *ConfigSyntax) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	total, failed := 0, 0

	for _, scanDir := range scanDirs {
		scanPath := filepath.Join(root, scanDir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		_ = filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			rel, _ := filepath.Rel(root, path)

			// Check exclusions
			for _, pattern := range excludePatterns {
				if strings.Contains(rel, pattern) {
					return nil
				}
			}

			ext := strings.ToLower(filepath.Ext(path))
			if !yamlExts[ext] && !jsonExts[ext] {
				return nil
			}

			total++
			if yamlExts[ext] {
				if errStr := validateYAML(path); errStr != "" {
					failed++
					issues = append(issues, fmt.Sprintf("SYNTAX_ERROR: %s — %s", rel, errStr))
				}
			} else if jsonExts[ext] {
				if errStr := validateJSON(path); errStr != "" {
					failed++
					issues = append(issues, fmt.Sprintf("SYNTAX_ERROR: %s — %s", rel, errStr))
				}
			}
			return nil
		})
	}

	if total > 0 {
		issues = append([]string{fmt.Sprintf("INFO: Validated %d config files (%d ok, %d failed)", total, total-failed, failed)}, issues...)
	}

	if failed > 0 {
		return Result{
			ID: c.ID(), Name: c.Name(), Status: "fail", Weight: c.Weight(),
			Message:  fmt.Sprintf("FAIL config syntax — %d syntax error(s)", failed),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: c.ID(), Name: c.Name(), Status: "pass", Weight: c.Weight(),
		Message:  fmt.Sprintf("PASS config syntax — %d files validated, 0 errors", total),
		Duration: time.Since(start),
	}
}

func validateYAML(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("read error: %v", err)
	}
	// Skip empty files
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		return ""
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		// Try to extract line info
		msg := err.Error()
		// yaml.v3 errors include line numbers in the message
		return fmt.Sprintf("YAML parse error: %s", msg)
	}
	return ""
}

func validateJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("read error: %v", err)
	}
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		return ""
	}

	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		// json.SyntaxError has Offset but not Line
		return fmt.Sprintf("JSON parse error: %v", err)
	}
	return ""
}

var _ Validator = (*ConfigSyntax)(nil)
