package ows

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

// ScopeConfig represents the verify scope configuration.
type ScopeConfig struct {
	ExcludePatterns []string `yaml:"exclude_patterns"`
	ScopeMode       bool     `yaml:"scope_mode"`
	QuickMode       bool     `yaml:"quick_mode"`
}

// LoadScopeConfig loads scope.yaml from the OWS package directory.
// Returns default config if file doesn't exist.
func LoadScopeConfig() *ScopeConfig {
	defaultCfg := &ScopeConfig{
		ScopeMode: true,
		ExcludePatterns: []string{
			"social-citas/**",
			"ovav-web/**",
			"ovav-mobile/**",
			".ovav/worktrees/**",
			"**/.git/**",
			"**/node_modules/**",
			"**/vendor/**",
			"**/__pycache__/**",
			"**/dist/**",
			"**/build/**",
			"**/*.pb.go",
		},
	}

	data, err := os.ReadFile(filepath.Join(os.Getenv("OVAV_ROOT"), "go-runtime", "internal", "ows", "scope.yaml"))
	if err != nil {
		return defaultCfg
	}

	var cfg ScopeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultCfg
	}
	return &cfg
}

// IsExcluded returns true if path matches any exclude pattern.
func (sc *ScopeConfig) IsExcluded(path string) bool {
	for _, pattern := range sc.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Also check if pattern contains wildcards that could match subpaths
		if strings.Contains(pattern, "**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			prefix = strings.TrimSuffix(prefix, "/**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// FilterFiles removes excluded files from a list of changed files.
func (sc *ScopeConfig) FilterFiles(files []string) []string {
	var filtered []string
	for _, f := range files {
		if !sc.IsExcluded(f) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// AffectedPackages returns the Go package paths that contain or import changed .go files.
// This is the key to scoped verification: only test/vet the packages that matter.
func AffectedPackages(repoRoot, goDir string, changedFiles []string) []string {
	if len(changedFiles) == 0 {
		return nil
	}

	// Find all .go files in the module
	goRoot := filepath.Join(repoRoot, goDir)
	packageMap := make(map[string]bool) // package path -> bool

	err := filepath.WalkDir(goRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		dir := filepath.Dir(path)
		rel, _ := filepath.Rel(goRoot, dir)
		pkgPath := rel
		if pkgPath == "." || pkgPath == "" {
			pkgPath = repoRoot
		}
		if !strings.HasPrefix(pkgPath, repoRoot) {
			pkgPath = filepath.Join(repoRoot, pkgPath)
		}
		packageMap[pkgPath] = true
		return nil
	})
	if err != nil {
		return nil
	}

	// Build set of affected packages from changed files
	affected := make(map[string]bool)
	for _, f := range changedFiles {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		dir := filepath.Dir(filepath.Join(repoRoot, f))
		// Find the package for this file
		for pkg := range packageMap {
			if strings.HasPrefix(dir, pkg) || dir == pkg {
				affected[pkg] = true
			}
		}
	}

	var result []string
	for pkg := range affected {
		result = append(result, pkg)
	}
	return result
}

// ScopedGoDirs returns only the Go directories that have changed files.
func (sc *ScopeConfig) ScopedGoDirs(repoRoot string, goDirs []string, changedFiles []string) []string {
	if !sc.ScopeMode || len(changedFiles) == 0 {
		return goDirs
	}

	filtered := sc.FilterFiles(changedFiles)
	if len(filtered) == 0 {
		return goDirs // fallback to all if filter removes everything
	}

	var scoped []string
	for _, dir := range goDirs {
		hasChanged := false
		for _, f := range filtered {
			if strings.HasPrefix(f, dir) {
				hasChanged = true
				break
			}
		}
		if hasChanged {
			scoped = append(scoped, dir)
		}
	}

	if len(scoped) == 0 {
		return goDirs // fallback to all if nothing matched
	}
	return scoped
}
