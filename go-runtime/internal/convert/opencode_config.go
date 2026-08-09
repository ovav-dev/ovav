package convert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── OpenCode Config Validation (opencode.json) ─────────────────────────────

// OpenCodeConfig mirrors the opencode.json structure for validation.
type OpenCodeConfig struct {
	Schema  string         `json:"$schema"`
	Model   string         `json:"model"`
	MCP     map[string]any `json:"mcp"`
	Agent   map[string]any `json:"agent"`
	Plugin  []any          `json:"plugin"`
	Compact any            `json:"compaction"`
}

// ConfigIssue represents a single validation finding.
type ConfigIssue struct {
	Field    string `json:"field"`
	Severity string `json:"severity"` // "critical", "warning", "info"
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// ValidateOpenCodeConfig validates opencode.json against opencode 1.17.8
// requirements. Returns issues grouped by severity.
func ValidateOpenCodeConfig(root string) (issues []ConfigIssue, err error) {
	path := filepath.Join(root, "opencode.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read opencode.json: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return []ConfigIssue{{
			Field: "$", Severity: "critical",
			Message: fmt.Sprintf("JSON inválido: %v", err),
		}}, nil
	}

	// 1. Schema check
	issues = append(issues, checkSchema(config)...)

	// 2. MCP format (CRITICAL — wrong format BREAKS opencode)
	issues = append(issues, checkMCPFormat(config)...)

	// 3. Agent section — OVAV agents must live in .md files
	issues = append(issues, checkAgentSection(config)...)

	// 4. Required keys
	issues = append(issues, checkRequiredConfigKeys(config)...)

	// 5. Plugin format
	issues = append(issues, checkPluginFormat(config)...)

	return issues, nil
}

// SyncOpenCodeConfig ensures opencode.json has correct MCP format.
// Returns true if fixes were applied.
func SyncOpenCodeConfig(root string) (fixed bool, err error) {
	path := filepath.Join(root, "opencode.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read opencode.json: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return false, fmt.Errorf("invalid JSON in opencode.json: %w", err)
	}

	modified := false

	// Fix MCP entries with legacy format (command string + args separate)
	if mcp, ok := config["mcp"].(map[string]any); ok {
		for _, raw := range mcp {
			server, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			// Detect legacy: "command" is a string (not array) and "args" exists
			cmd, cmdIsStr := server["command"].(string)
			args, hasArgs := server["args"]
			if cmdIsStr && hasArgs {
				// Convert to new format
				argsList, ok := args.([]any)
				if !ok {
					continue
				}
				newCmd := make([]any, 0, len(argsList)+1)
				newCmd = append(newCmd, cmd)
				newCmd = append(newCmd, argsList...)

				server["type"] = "local"
				server["command"] = newCmd
				server["enabled"] = true
				delete(server, "args")
				delete(server, "description")

				modified = true
			}
		}
	}

	if !modified {
		return false, nil
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal fixed config: %w", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		return false, fmt.Errorf("write opencode.json: %w", err)
	}
	return true, nil
}

// ── Individual Checks ───────────────────────────────────────────────────

func checkSchema(config map[string]any) []ConfigIssue {
	schema, _ := config["$schema"].(string)
	canonical := "https://opencode.ai/config.json"
	if schema == "" {
		return []ConfigIssue{{
			Field: "$schema", Severity: "critical",
			Message: "Falta $schema — opencode puede rechazar la configuración",
			Fix:     fmt.Sprintf(`"$schema": %q`, canonical),
		}}
	}
	if schema != canonical {
		return []ConfigIssue{{
			Field: "$schema", Severity: "warning",
			Message: fmt.Sprintf("Schema no canónico: %q (esperado: %s)", schema, canonical),
		}}
	}
	return nil
}

func checkMCPFormat(config map[string]any) []ConfigIssue {
	var issues []ConfigIssue
	mcp, ok := config["mcp"].(map[string]any)
	if !ok || len(mcp) == 0 {
		return nil // No MCP configured — OK
	}

	for name, raw := range mcp {
		server, ok := raw.(map[string]any)
		if !ok {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s", name), Severity: "critical",
				Message: "Valor MCP debe ser un objeto",
			})
			continue
		}

		// CRITICAL: Legacy format — command is string + args exists
		cmd, cmdIsStr := server["command"].(string)
		_, hasArgs := server["args"]
		if cmdIsStr && hasArgs {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s", name), Severity: "critical",
				Message: "FORMATO OBSOLETO — 'command' es string con 'args' separados. Esto ROMPE opencode 1.17.x.",
				Fix: fmt.Sprintf(
					`{"type":"local","command":["%s","<script>","<arg>"],"enabled":true}`,
					cmd,
				),
			})
			continue
		}

		// Validate current format
		stype, _ := server["type"].(string)
		if stype != "local" {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s.type", name), Severity: "warning",
				Message: fmt.Sprintf("type debe ser 'local' (es: %q)", stype),
			})
		}

		cmdArr, cmdIsArr := server["command"].([]any)
		if !cmdIsArr {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s.command", name), Severity: "critical",
				Message: "'command' debe ser array",
				Fix:     `"command": ["python3", "script.py", "arg"]`,
			})
		} else if len(cmdArr) == 0 {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s.command", name), Severity: "critical",
				Message: "'command' array vacío",
			})
		}

		if _, hasEnabled := server["enabled"]; !hasEnabled {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("mcp.%s.enabled", name), Severity: "warning",
				Message: "Falta 'enabled' — el servidor MCP podría no iniciar",
				Fix:     `"enabled": true`,
			})
		}
	}

	return issues
}

// OVAV agent names that must NOT appear in opencode.json's agent section.
// Canonical rule: agent definitions live exclusively in .opencode/agents/*.md.
var ovavAgentNames = []string{
	"platform-engineering", "research-intelligence", "ux-design",
	"digital-product", "commercial-growth", "health-performance",
	"education-career", "devops-infrastructure", "adversarial-intelligence",
	"legal-compliance", "brand-positioning",
	"thavren", "eidren", "elena", "dante", "sofia", "renata",
	"valeria", "uriel", "kenji-tanaka", "camila", "ines",
}

func checkAgentSection(config map[string]any) []ConfigIssue {
	agent, ok := config["agent"].(map[string]any)
	if !ok || len(agent) == 0 {
		return nil // Empty — OK
	}

	lower := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(s, " ", "-"), "_", "-"))
	}

	var issues []ConfigIssue
	for name := range agent {
		normalized := lower(name)
		for _, ovav := range ovavAgentNames {
			if normalized == ovav || strings.Contains(normalized, ovav) {
				issues = append(issues, ConfigIssue{
					Field: fmt.Sprintf("agent.%s", name), Severity: "critical",
					Message: fmt.Sprintf(
						"Agente OVAV '%s' en opencode.json. Debe estar en .opencode/agents/*.md (generado por convert engine).",
						name,
					),
					Fix: fmt.Sprintf("Eliminar '%s' de la sección 'agent' en opencode.json", name),
				})
				break
			}
		}
	}
	return issues
}

func checkRequiredConfigKeys(config map[string]any) []ConfigIssue {
	required := []string{"$schema", "model", "instructions"}
	var issues []ConfigIssue
	for _, key := range required {
		if _, ok := config[key]; !ok {
			issues = append(issues, ConfigIssue{
				Field: key, Severity: "warning",
				Message: fmt.Sprintf("Campo '%s' recomendado pero ausente", key),
			})
		}
	}
	return issues
}

func checkPluginFormat(config map[string]any) []ConfigIssue {
	plugins, ok := config["plugin"]
	if !ok {
		return nil
	}
	arr, ok := plugins.([]any)
	if !ok {
		return []ConfigIssue{{
			Field: "plugin", Severity: "warning",
			Message: "'plugin' debe ser un array de strings",
		}}
	}
	var issues []ConfigIssue
	for i, p := range arr {
		s, ok := p.(string)
		if !ok {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("plugin[%d]", i), Severity: "warning",
				Message: "Cada plugin debe ser un string (path al archivo .js)",
			})
			continue
		}
		// Warn on npm-scoped plugins (should be local)
		if strings.HasPrefix(s, "@") {
			issues = append(issues, ConfigIssue{
				Field: fmt.Sprintf("plugin[%d]", i), Severity: "info",
				Message: fmt.Sprintf("Plugin npm '%s' — considerar versión local .opencode/plugins/", s),
			})
		}
	}
	return issues
}

// ── OpenCode Config Generation (Canonical YAML → opencode.json) ────────────

// CanonicalOpenCodeConfig mirrors .ovav/source/opencode/config.yaml.
// This is the OVAV-native canonical source for opencode.json generation.
type CanonicalOpenCodeConfig struct {
	Version     string                        `yaml:"version"`
	Schema      string                        `yaml:"schema"`
	Runtime     canonicalRuntime              `yaml:"runtime"`
	MCP         map[string]canonicalMCPServer `yaml:"mcp"`
	Plugins     []string                      `yaml:"plugins"`
	Providers   map[string]canonicalProvider  `yaml:"providers"`
	Permissions canonicalPermissions          `yaml:"permissions"`
	References  map[string]canonicalReference `yaml:"references"`
	Compaction  map[string]any                `yaml:"compaction"`
	ToolOutput  map[string]int                `yaml:"tool_output"`
	User        *canonicalUser                `yaml:"user,omitempty"`
}

type canonicalRuntime struct {
	Model        string         `yaml:"model"`
	SmallModel   string         `yaml:"small_model"`
	DefaultAgent string         `yaml:"default_agent"`
	Instructions []string       `yaml:"instructions"`
	Agent        map[string]any `yaml:"agent"`
}

type canonicalMCPServer struct {
	Type    string   `yaml:"type"`
	Command []string `yaml:"command"`
	Enabled bool     `yaml:"enabled"`
}

type canonicalProvider struct {
	Options map[string]int `yaml:"options"`
}

type canonicalPermissions struct {
	Edit              string            `yaml:"edit"`
	Bash              map[string]string `yaml:"bash"`
	ExternalDirectory map[string]string `yaml:"external_directory"`
}

type canonicalReference struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Hidden      bool   `yaml:"hidden,omitempty"`
}

type canonicalUser struct {
	Username string `yaml:"username"`
}

// GenerateOpenCodeConfig reads the canonical OVAV YAML config and generates
// opencode.json. This OVERWRITES the existing opencode.json — OVAV is the
// authoritative source for CLI configuration.
//
// Canonical source: .ovav/source/opencode/config.yaml
// Target:           opencode.json
func GenerateOpenCodeConfig(root string) error {
	sourcePath := filepath.Join(root, ".ovav", "source", "opencode", "config.yaml")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read canonical config: %w", err)
	}

	var canonical CanonicalOpenCodeConfig
	if err := yaml.Unmarshal(data, &canonical); err != nil {
		return fmt.Errorf("parse canonical config: %w", err)
	}

	// Build opencode.json structure
	config := make(map[string]any)

	// Schema
	config["$schema"] = canonical.Schema

	// Runtime
	config["model"] = canonical.Runtime.Model
	config["small_model"] = canonical.Runtime.SmallModel
	config["default_agent"] = canonical.Runtime.DefaultAgent
	config["instructions"] = canonical.Runtime.Instructions
	config["agent"] = canonical.Runtime.Agent

	// User override (if present)
	if canonical.User != nil && canonical.User.Username != "" {
		config["username"] = canonical.User.Username
	}

	// MCP servers
	if len(canonical.MCP) > 0 {
		mcp := make(map[string]any)
		for name, server := range canonical.MCP {
			mcp[name] = map[string]any{
				"type":    server.Type,
				"command": server.Command,
				"enabled": server.Enabled,
			}
		}
		config["mcp"] = mcp
	}

	// Plugins
	if len(canonical.Plugins) > 0 {
		plugins := make([]any, len(canonical.Plugins))
		for i, p := range canonical.Plugins {
			plugins[i] = p
		}
		config["plugin"] = plugins
	}

	// Providers
	if len(canonical.Providers) > 0 {
		providers := make(map[string]any)
		for name, prov := range canonical.Providers {
			opts := make(map[string]any)
			for k, v := range prov.Options {
				opts[k] = v
			}
			providers[name] = map[string]any{"options": opts}
		}
		config["provider"] = providers
	}

	// Permissions
	if canonical.Permissions.Edit != "" || len(canonical.Permissions.Bash) > 0 {
		perm := make(map[string]any)
		perm["edit"] = canonical.Permissions.Edit

		if len(canonical.Permissions.Bash) > 0 {
			bash := make(map[string]any)
			for k, v := range canonical.Permissions.Bash {
				bash[k] = v
			}
			perm["bash"] = bash
		}
		if len(canonical.Permissions.ExternalDirectory) > 0 {
			ext := make(map[string]any)
			for k, v := range canonical.Permissions.ExternalDirectory {
				ext[k] = v
			}
			perm["external_directory"] = ext
		}
		config["permission"] = perm
	}

	// References
	if len(canonical.References) > 0 {
		refs := make(map[string]any)
		for name, ref := range canonical.References {
			r := map[string]any{
				"path":        ref.Path,
				"description": ref.Description,
			}
			if ref.Hidden {
				r["hidden"] = true
			}
			refs[name] = r
		}
		config["references"] = refs
	}

	// Compaction
	if len(canonical.Compaction) > 0 {
		config["compaction"] = canonical.Compaction
	}

	// Tool output
	if len(canonical.ToolOutput) > 0 {
		to := make(map[string]any)
		for k, v := range canonical.ToolOutput {
			to[k] = v
		}
		config["tool_output"] = to
	}

	// Write opencode.json (OVERWRITE — OVAV is authoritative)
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal generated config: %w", err)
	}

	targetPath := filepath.Join(root, "opencode.json")
	if err := os.WriteFile(targetPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("write opencode.json: %w", err)
	}

	return nil
}
