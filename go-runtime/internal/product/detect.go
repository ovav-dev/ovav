package product

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectStack represents the detected technology stack of a project.
type ProjectStack struct {
	Primary   string   // react, vue, angular, go, python, rust, etc.
	Secondary string   // typescript, javascript, etc.
	Files     []string // detected config files
}

// DetectProjectStack scans a directory for technology indicators.
func DetectProjectStack(projectDir string) ProjectStack {
	stack := ProjectStack{}
	indicators := map[string]func(string) bool{
		// JavaScript/TypeScript frameworks
		"react": func(d string) bool { return fileExists(d, "package.json") && fileContains(d, "package.json", "react") },
		"vue":   func(d string) bool { return fileExists(d, "package.json") && fileContains(d, "package.json", "vue") },
		"angular": func(d string) bool {
			return fileExists(d, "angular.json") || fileExists(d, "package.json") && fileContains(d, "package.json", "@angular")
		},
		"svelte": func(d string) bool { return fileExists(d, "svelte.config.js") || fileExists(d, "svelte.config.ts") },
		"nextjs": func(d string) bool {
			return fileExists(d, "next.config.js") || fileExists(d, "next.config.mjs") || fileExists(d, "next.config.ts")
		},
		"nuxt":  func(d string) bool { return fileExists(d, "nuxt.config.ts") || fileExists(d, "nuxt.config.js") },
		"astro": func(d string) bool { return fileExists(d, "astro.config.mjs") || fileExists(d, "astro.config.ts") },
		"remix": func(d string) bool { return fileExists(d, "remix.config.js") || fileExists(d, "app/root.tsx") },
		// Backend
		"go": func(d string) bool { return fileExists(d, "go.mod") },
		"python": func(d string) bool {
			return fileExists(d, "requirements.txt") || fileExists(d, "pyproject.toml") || fileExists(d, "Pipfile")
		},
		"rust": func(d string) bool { return fileExists(d, "Cargo.toml") },
		"java": func(d string) bool { return fileExists(d, "pom.xml") || fileExists(d, "build.gradle") },
		"php":  func(d string) bool { return fileExists(d, "composer.json") },
		"ruby": func(d string) bool { return fileExists(d, "Gemfile") },
		// Infra
		"terraform": func(d string) bool { return hasFiles(d, ".tf") },
		"kubernetes": func(d string) bool {
			return fileExists(d, "Dockerfile") || fileExists(d, "docker-compose.yml") || fileExists(d, "docker-compose.yaml")
		},
	}

	for name, check := range indicators {
		if check(projectDir) {
			stack.Files = append(stack.Files, name)
		}
	}

	// Determine primary
	if len(stack.Files) > 0 {
		stack.Primary = stack.Files[0]
	}

	// Detect TypeScript
	if fileExists(projectDir, "tsconfig.json") {
		stack.Secondary = "typescript"
	}

	return stack
}

// SelectAgentsForStack returns the agent filenames relevant to a project stack.
func SelectAgentsForStack(stack ProjectStack, allAgentDir string) []string {
	// Priority-ordered lead mapping (deterministic — no map iteration)
	type leadEntry struct {
		stack string
		agent string
	}
	leadPriority := []leadEntry{
		{"react", "lead-dante.md"},
		{"vue", "lead-dante.md"},
		{"angular", "lead-dante.md"},
		{"svelte", "lead-dante.md"},
		{"nextjs", "lead-dante.md"},
		{"nuxt", "lead-dante.md"},
		{"astro", "lead-dante.md"},
		{"remix", "lead-dante.md"},
		{"go", "lead-thavren.md"},
		{"python", "lead-thavren.md"},
		{"rust", "lead-thavren.md"},
	}

	// Select lead by first matching stack in priority order
	lead := "lead-thavren.md" // fallback
	for _, entry := range leadPriority {
		if entry.stack == stack.Primary {
			lead = entry.agent
			break
		}
		for _, detected := range stack.Files {
			if detected == entry.stack {
				lead = entry.agent
				break
			}
		}
		if lead != "lead-thavren.md" {
			break
		}
	}

	core := []string{lead}
	return core
}

// SkillsForStack returns minimal skill names relevant to a project.
func SkillsForStack(stack ProjectStack) []string {
	// Core skills always included
	core := []string{
		"ovav-context-pack",
		"ovav-identity-guard",
		"ovav-response-contract",
		"ovav-platform-session",
	}

	// Stack-specific skills
	if stack.Primary != "" {
		core = append(core, "ovav-session-continuity")
	}

	return core
}

// fileExists checks if a file exists in a directory.
func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// fileContains checks if a file contains a string (simple grep).
func fileContains(dir, name, search string) bool {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), search)
}

// hasFiles checks if any file with the given extension exists.
func hasFiles(dir, ext string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			return true
		}
	}
	return false
}
