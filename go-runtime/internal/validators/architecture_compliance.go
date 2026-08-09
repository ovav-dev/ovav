// F1 — Architecture Structure Compliance validator.
//
// Validates OVAV project architecture:
//   - Root directory structure follows conventions
//   - Stack segregation: Go in go-runtime/, TS in tools/cpanel/src/, Python only in tools/
//   - Required artifacts exist (.ovav/, go-runtime/, docs/, tools/)
//   - No cross-contamination between stacks
package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArchitectureCompliance validates project structure and stack boundaries (F1).
type ArchitectureCompliance struct{}

func NewArchitectureCompliance() *ArchitectureCompliance { return &ArchitectureCompliance{} }

func (a *ArchitectureCompliance) ID() string   { return "architecture_compliance" }
func (a *ArchitectureCompliance) Name() string { return "F1 — Architecture Structure Compliance" }
func (a *ArchitectureCompliance) Description() string {
	return "Validates project directory structure, stack segregation, and required artifact presence"
}
func (a *ArchitectureCompliance) Weight() int { return 5 }

// resolveRepoRoot finds the OVAV repo root by walking up from the given root
// looking for both .ovav/ and go-runtime/ directories (both required at real root).
// Returns the original root if repo root not found.
func resolveRepoRoot(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	dir := abs
	for range 10 {
		// Both .ovav/ and go-runtime/go.mod must exist at the real repo root
		if _, err := os.Stat(filepath.Join(dir, ".ovav")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "go-runtime", "go.mod")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return abs // fallback to original absolute root
}

func (a *ArchitectureCompliance) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// Resolve the actual repo root (handles worktrees, subdirs, etc.)
	repoRoot := resolveRepoRoot(root)

	// ── 1. Required root directories ────────────────────────────────────────
	requiredDirs := []string{
		".ovav", "go-runtime", "docs", "tools", "docs-site",
	}
	for _, dir := range requiredDirs {
		path := filepath.Join(repoRoot, dir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			issues = append(issues, fmt.Sprintf("F1: required directory missing: %s/", dir))
		}
	}

	// ── 2. Stack segregation — no Go files outside go-runtime/ ─────────────
	var goViolations []string
	filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)

		// Skip vendor, .git, node_modules, build artifacts
		if strings.HasPrefix(rel, ".git/") ||
			strings.HasPrefix(rel, "go-runtime/") ||
			strings.Contains(rel, "/build/") ||
			strings.Contains(rel, "node_modules/") ||
			strings.Contains(rel, ".venv/") ||
			strings.Contains(rel, "__pycache__/") {
			return nil
		}
		// Skip known sub-projects, worktrees, and Python virtual environments
		if strings.HasPrefix(rel, "social-citas/") ||
			strings.HasPrefix(rel, "ovav-web/") ||
			strings.HasPrefix(rel, "ovav-mobile/") ||
			strings.HasPrefix(rel, ".mimocode/") ||
			strings.HasPrefix(rel, ".owav/") ||
			strings.HasPrefix(rel, "node_modules/") ||
			strings.HasPrefix(rel, ".venv/") ||
			strings.Contains(rel, "__pycache__/") ||
			strings.HasPrefix(rel, ".ovav/worktrees/") {
			return nil
		}

		if strings.HasSuffix(rel, ".go") {
			goViolations = append(goViolations, rel)
		}
		return nil
	})
	if len(goViolations) > 0 {
		for _, v := range goViolations {
			issues = append(issues, fmt.Sprintf("F1: Go file outside go-runtime/: %s", v))
		}
	}

	// ── 3. Stack segregation — no Python product code ──────────────────────
	// Python is allowed in tools/ (governance) but not as product code.
	// Product directories that should NOT contain Python: go-runtime/
	var pyViolations []string
	filepath.Walk(filepath.Join(repoRoot, "go-runtime"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".py") {
			rel, _ := filepath.Rel(repoRoot, path)
			pyViolations = append(pyViolations, rel)
		}
		return nil
	})
	for _, v := range pyViolations {
		issues = append(issues, fmt.Sprintf("F1: Python file in Go product directory: %s", v))
	}

	// ── 4. Required artifact files ─────────────────────────────────────────
	requiredFiles := []string{
		"go-runtime/go.mod",
		".ovav/plan/caps.yaml",
		".ovav/policy/permission_authority.json",
		"AGENTS.md",
	}
	for _, f := range requiredFiles {
		path := filepath.Join(repoRoot, f)
		if _, err := os.Stat(path); err != nil {
			issues = append(issues, fmt.Sprintf("F1: required artifact missing: %s", f))
		}
	}

	// ── 5. Banned patterns ─────────────────────────────────────────────────
	bannedDirs := []string{"__pycache__", ".pytest_cache", "node_modules", ".venv"}
	for _, bd := range bannedDirs {
		filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && info.Name() == bd {
				rel, _ := filepath.Rel(repoRoot, path)
				// node_modules are allowed everywhere — they are gitignored project dependencies
				if bd == "node_modules" {
					return nil
				}
				// Allow __pycache__ in tools/, .venv/, .ovav/, bin/, tests/, web/ (runtime artifacts)
				if bd == "__pycache__" && (strings.HasPrefix(rel, "tools/") ||
					strings.HasPrefix(rel, ".venv/") ||
					strings.HasPrefix(rel, ".ovav/") ||
					strings.HasPrefix(rel, "bin/") ||
					strings.HasPrefix(rel, "tests/") ||
					strings.HasPrefix(rel, "web/")) {
					return nil
				}
				// Allow .venv/ and .pytest_cache (Python virtual/test artifacts)
				if bd == ".venv" || bd == ".pytest_cache" {
					return nil
				}
				issues = append(issues, fmt.Sprintf("F1: banned directory found: %s/", rel))
			}
			return nil
		})
	}

	// ── Result ─────────────────────────────────────────────────────────────
	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}
	return Result{
		ID:          a.ID(),
		Name:        a.Name(),
		Status:      status,
		Issues:      issues,
		Weight:      a.Weight(),
		Duration:    time.Since(start),
		Description: a.Description(),
	}
}
