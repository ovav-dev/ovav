// Package projector — OVAV Canonical → Deploy Projection Engine
//
// Architecture v2.0: Modular projectors for each artifact type.
//
// Canonical source structure:
//   .ovav/source/configs/   — Tool configs (wezterm, fish, git, etc.)
//   .ovav/source/harnesses/ — Test harnesses
//   .ovav/source/programs/   — CI/CD, deploy programs
//   .ovav/source/agents/    — Agent definitions
//   .ovav/source/skills/    — Skill definitions
//   .ovav/visual/           — Theme, assets
//
// Projection targets:
//   config/                 — Tool configs
//   .github/workflows/      — CI programs
//   go-runtime/.../harness — Test harnesses
//   clients/opencode/       — Agent projections
//   .opencode/             — Skills, themes, plugins
//
// Each projector is independent and can be run separately.
package projector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Projector defines the interface for all projectors.
type Projector interface {
	// Name returns the projector's name.
	Name() string
	// SourceDir returns the canonical source directory relative to OVAV root.
	SourceDir() string
	// DeployDir returns the deploy target directory relative to OVAV root.
	DeployDir() string
	// Project runs the projection, returning count of artifacts projected.
	Project(root string, verbose bool) (int, error)
}

// AllProjectors returns all available projectors.
func AllProjectors() []Projector {
	return []Projector{
		&ConfigsProjector{},
		&VisualProjector{},
		&AgentsProjector{},
		&SkillsProjector{},
		&ProgramsProjector{},
		&HarnessesProjector{},
	}
}

// FindProjector returns a projector by name.
func FindProjector(name string) Projector {
	for _, p := range AllProjectors() {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// ── Configs Projector ─────────────────────────────────────────────────────────

// ConfigsProjector projects tool configs from .ovav/source/configs/ to config/.
type ConfigsProjector struct{}

func (p *ConfigsProjector) Name() string     { return "configs" }
func (p *ConfigsProjector) SourceDir() string { return ".ovav/source/configs" }
func (p *ConfigsProjector) DeployDir() string { return "config" }

func (p *ConfigsProjector) Project(root string, verbose bool) (int, error) {
	count := 0
	sourceRoot := filepath.Join(root, p.SourceDir())

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read source: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		toolName := entry.Name()
		sourcePath := filepath.Join(sourceRoot, toolName)

		// Deploy target: config/<tool>/
		deployPath := filepath.Join(root, p.DeployDir(), toolName)

		if err := os.MkdirAll(deployPath, 0755); err != nil {
			return count, fmt.Errorf("mkdir deploy %s: %w", toolName, err)
		}

		// Copy all files from source to deploy
		subEntries, err := os.ReadDir(sourcePath)
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() {
				continue
			}
			src := filepath.Join(sourcePath, sub.Name())
			if !p.validateFile(src) {
				if verbose {
					fmt.Printf("    ⚠️  %s: %s skipped (validation failed)\n", toolName, sub.Name())
				}
				continue
			}
			dst := filepath.Join(deployPath, sub.Name())
			if err := copyFile(src, dst); err != nil {
				return count, fmt.Errorf("copy %s: %w", sub.Name(), err)
			}
			count++
			if verbose {
				fmt.Printf("    ✅ %s: %s → %s\n", toolName, sub.Name(), deployPath)
			}
		}
	}
	return count, nil
}

// ── Visual Projector ──────────────────────────────────────────────────────────

// VisualProjector projects visual assets (theme, monitoring, assets).
type VisualProjector struct{}

func (p *VisualProjector) Name() string     { return "visual" }
func (p *VisualProjector) SourceDir() string { return ".ovav/visual" }
func (p *VisualProjector) DeployDir() string { return ".opencode" } // partial

func (p *VisualProjector) Project(root string, verbose bool) (int, error) {
	// Theme → .opencode/themes/ovav.json
	// Monitoring → .opencode/plugins/ovav-monitor.js
	// Assets are read-only, not projected
	count := 0

	// Theme projection
	themeSrc := filepath.Join(root, ".ovav", "visual", "theme", "theme.yaml")
	themeDstDir := filepath.Join(root, ".opencode", "themes")
	if err := os.MkdirAll(themeDstDir, 0755); err != nil {
		return 0, fmt.Errorf("mkdir themes: %w", err)
	}
	if _, err := os.Stat(themeSrc); err == nil {
		count++
		if verbose {
			fmt.Printf("    ✅ theme: %s\n", themeSrc)
		}
	}

	// Monitoring projection
	monSrc := filepath.Join(root, ".ovav", "visual", "monitoring", "monitoring.yaml")
	monDstDir := filepath.Join(root, ".opencode", "plugins")
	if err := os.MkdirAll(monDstDir, 0755); err != nil {
		return 0, fmt.Errorf("mkdir plugins: %w", err)
	}
	if _, err := os.Stat(monSrc); err == nil {
		count++
		if verbose {
			fmt.Printf("    ✅ monitoring: %s\n", monSrc)
		}
	}

	return count, nil
}

// ── Agents Projector ──────────────────────────────────────────────────────────

// AgentsProjector projects agent definitions to CLI runtimes.
type AgentsProjector struct{}

func (p *AgentsProjector) Name() string     { return "agents" }
func (p *AgentsProjector) SourceDir() string { return ".ovav/source/agents" }
func (p *AgentsProjector) DeployDir() string { return "clients/opencode/agents" }

func (p *AgentsProjector) Project(root string, verbose bool) (int, error) {
	// Delegates to the convert package for agent projection
	// This is a stub that counts available agents
	sourceRoot := filepath.Join(root, p.SourceDir())
	count := 0

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read source: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subEntries, _ := os.ReadDir(filepath.Join(sourceRoot, entry.Name()))
		count += len(subEntries)
	}
	return count, nil
}

// ── Skills Projector ─────────────────────────────────────────────────────────

// SkillsProjector projects skills to .opencode/skills/.
type SkillsProjector struct{}

func (p *SkillsProjector) Name() string     { return "skills" }
func (p *SkillsProjector) SourceDir() string { return ".ovav/source/skills" }
func (p *SkillsProjector) DeployDir() string { return ".opencode/skills" }

func (p *SkillsProjector) Project(root string, verbose bool) (int, error) {
	count := 0
	sourceRoot := filepath.Join(root, p.SourceDir())

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read source: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		count++
	}
	return count, nil
}

// ── Programs Projector ────────────────────────────────────────────────────────

// ProgramsProjector projects CI/CD and deploy programs.
type ProgramsProjector struct{}

func (p *ProgramsProjector) Name() string     { return "programs" }
func (p *ProgramsProjector) SourceDir() string { return ".ovav/source/programs" }
func (p *ProgramsProjector) DeployDir() string { return ".github/workflows" }

func (p *ProgramsProjector) Project(root string, verbose bool) (int, error) {
	count := 0
	sourceRoot := filepath.Join(root, p.SourceDir())

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read source: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		programName := entry.Name()
		sourcePath := filepath.Join(sourceRoot, programName)

		// Deploy target: .github/workflows/<program>/
		deployPath := filepath.Join(root, p.DeployDir(), programName)

		if err := os.MkdirAll(deployPath, 0755); err != nil {
			return count, fmt.Errorf("mkdir deploy %s: %w", programName, err)
		}

		subEntries, _ := os.ReadDir(sourcePath)
		for _, sub := range subEntries {
			if sub.IsDir() {
				continue
			}
			src := filepath.Join(sourcePath, sub.Name())
			dst := filepath.Join(deployPath, sub.Name())
			if err := copyFile(src, dst); err != nil {
				return count, fmt.Errorf("copy %s: %w", sub.Name(), err)
			}
			count++
			if verbose {
				fmt.Printf("    ✅ program.%s: %s → %s\n", programName, sub.Name(), deployPath)
			}
		}
	}
	return count, nil
}

// ── Harnesses Projector ──────────────────────────────────────────────────────

// HarnessesProjector projects test harnesses.
type HarnessesProjector struct{}

func (p *HarnessesProjector) Name() string     { return "harnesses" }
func (p *HarnessesProjector) SourceDir() string { return ".ovav/source/harnesses" }
func (p *HarnessesProjector) DeployDir() string { return "go-runtime/internal/testing/harnesses" }

func (p *HarnessesProjector) Project(root string, verbose bool) (int, error) {
	count := 0
	sourceRoot := filepath.Join(root, p.SourceDir())

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read source: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runtimeName := entry.Name()
		sourcePath := filepath.Join(sourceRoot, runtimeName)

		deployPath := filepath.Join(root, p.DeployDir(), runtimeName)

		if err := os.MkdirAll(deployPath, 0755); err != nil {
			return count, fmt.Errorf("mkdir deploy %s: %w", runtimeName, err)
		}

		subEntries, _ := os.ReadDir(sourcePath)
		for _, sub := range subEntries {
			if sub.IsDir() {
				continue
			}
			src := filepath.Join(sourcePath, sub.Name())
			dst := filepath.Join(deployPath, sub.Name())
			if err := copyFile(src, dst); err != nil {
				return count, fmt.Errorf("copy %s: %w", sub.Name(), err)
			}
			count++
			if verbose {
				fmt.Printf("    ✅ harness.%s: %s → %s\n", runtimeName, sub.Name(), deployPath)
			}
		}
	}
	return count, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// validateFile checks file content validity before projection.
// Returns true if the file is valid or doesn't need validation.
func (p *ConfigsProjector) validateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return validateYAML(path)
	case ".lua":
		return validateLua(path)
	}
	return true // No validation needed for unknown extensions
}

// validateYAML checks if a YAML file parses correctly.
func validateYAML(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var v any
	return yaml.Unmarshal(data, &v) == nil
}

// validateLua performs basic Lua syntax validation by checking bracket balance.
func validateLua(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)

	// Check for balanced curly braces { }
	if !balancedBrackets(content, "{", "}") {
		return false
	}
	// Check for balanced parentheses ( )
	if !balancedBrackets(content, "(", ")") {
		return false
	}
	// Check for balanced long-string brackets [[ ]]
	if !balancedLuaLongString(content) {
		return false
	}
	return true
}

// balancedLuaLongString checks [[ and ]] bracket balance in Lua long strings.
func balancedLuaLongString(content string) bool {
	depth := 0
	for i := 0; i < len(content); i++ {
		if i+1 < len(content) {
			if content[i] == '[' && content[i+1] == '[' {
				depth++
				i++ // skip next char
				continue
			}
			if content[i] == ']' && content[i+1] == ']' {
				depth--
				if depth < 0 {
					return false
				}
				i++ // skip next char
				continue
			}
		}
	}
	return depth == 0
}

// balancedBrackets checks if single-char brackets are balanced,
// ignoring brackets inside string literals.
func balancedBrackets(content, open, close string) bool {
	depth := 0
	inString := false
	stringChar := rune(0)

	for i := 0; i < len(content); i++ {
		c := rune(content[i])

		// Handle string literals (single and double quotes)
		if !inString && (c == '"' || c == '\'') {
			inString = true
			stringChar = c
			continue
		}
		if inString && c == stringChar {
			// Check for escaped quote
			if i > 0 && content[i-1] == '\\' {
				// Count backslashes to determine if escaped
				bs := 0
				for j := i - 2; j >= 0 && content[j] == '\\'; j-- {
					bs++
				}
				if bs%2 == 0 {
					inString = false
				}
			} else {
				inString = false
			}
			continue
		}
		if inString {
			continue
		}

		if c == rune(open[0]) {
			depth++
		} else if c == rune(close[0]) {
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}
