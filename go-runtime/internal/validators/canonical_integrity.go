package validators

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CanonicalIntegrity validates canonical integrity across the codebase:
//  1. SHA256-duplicate files (same content, different paths) — catches copy-paste drift
//  2. Broken Python imports — import statements that don't resolve to real files
//
// Replaces: tools/validators/check_canonical_integrity.py
type CanonicalIntegrity struct{}

func NewCanonicalIntegrity() *CanonicalIntegrity { return &CanonicalIntegrity{} }

func (v *CanonicalIntegrity) ID() string   { return "canonical_integrity" }
func (v *CanonicalIntegrity) Name() string { return "Canonical Integrity" }
func (v *CanonicalIntegrity) Description() string {
	return "Detects SHA256-duplicate files and broken Python imports"
}
func (v *CanonicalIntegrity) Weight() int { return 8 }

// scanRoots are the directories scanned for integrity issues.
var scanRoots = []string{"tools", ".opencode"}

// excludeDirs are directory names to skip during scanning.
var excludeDirs = map[string]bool{
	"__pycache__": true, ".git": true, ".ovav": true,
	"node_modules": true, ".pytest_cache": true,
}

// excludeFiles are filenames to skip during scanning.
var excludeFiles = map[string]bool{
	"__init__.py": true, "conftest.py": true,
}

// minDupeSize is the minimum file size to consider for duplicate detection.
const minDupeSize = 50

// repoLocalPrefixes are import prefixes considered repo-local.
var repoLocalPrefixes = []string{
	"tools.", "registry.", ".ovav.", "ovav.", "schemas.", "config.", "docs.", "tests.",
}

var (
	reFromImport = regexp.MustCompile(`from\s+(\S+)\s+import`)
	reImport     = regexp.MustCompile(`import\s+(\S+)`)
)

func (v *CanonicalIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	issues = append(issues, v.findDuplicates(root)...)
	issues = append(issues, v.findBrokenImports(root)...)

	hasCritical := false
	for _, i := range issues {
		if strings.HasPrefix(i, "DUPLICATE:") || strings.HasPrefix(i, "BROKEN_IMPORT:") {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		return Result{
			ID:       v.ID(),
			Name:     v.Name(),
			Status:   "fail",
			Weight:   v.Weight(),
			Message:  fmt.Sprintf("FAIL canonical integrity — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID:       v.ID(),
		Name:     v.Name(),
		Status:   "pass",
		Weight:   v.Weight(),
		Message:  fmt.Sprintf("PASS canonical integrity — %d issue(s) found", len(issues)),
		Duration: time.Since(start),
	}
}

// findDuplicates scans scanRoots for .py and .go files, hashes them, and
// reports files with identical SHA256 content but different paths.
func (v *CanonicalIntegrity) findDuplicates(root string) []string {
	var issues []string
	hashToPaths := make(map[string][]string)

	for _, scanRoot := range scanRoots {
		scanPath := filepath.Join(root, scanRoot)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		_ = filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Skip excluded directories
			parts := strings.Split(filepath.ToSlash(path), "/")
			for _, part := range parts {
				if excludeDirs[part] {
					return nil
				}
			}

			// Only scan .py and .go files
			ext := filepath.Ext(path)
			if ext != ".py" && ext != ".go" {
				return nil
			}

			// Skip excluded filenames
			if excludeFiles[filepath.Base(path)] {
				return nil
			}

			// Skip files below minimum size
			if info.Size() < minDupeSize {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}

			hash := fmt.Sprintf("%x", sha256.Sum256(data))
			rel, _ := filepath.Rel(root, path)
			hashToPaths[hash] = append(hashToPaths[hash], rel)
			return nil
		})
	}

	for hash, paths := range hashToPaths {
		if len(paths) > 1 {
			issues = append(issues, fmt.Sprintf(
				"DUPLICATE: %d files share SHA256 %s...: %s",
				len(paths), hash[:16], strings.Join(paths, " | "),
			))
		}
	}

	return issues
}

// findBrokenImports scans Python files for import statements and verifies
// they resolve to real files on disk.
func (v *CanonicalIntegrity) findBrokenImports(root string) []string {
	var issues []string

	for _, scanRoot := range scanRoots {
		scanPath := filepath.Join(root, scanRoot)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		_ = filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			parts := strings.Split(filepath.ToSlash(path), "/")
			for _, part := range parts {
				if excludeDirs[part] {
					return nil
				}
			}

			if filepath.Ext(path) != ".py" {
				return nil
			}
			if excludeFiles[filepath.Base(path)] {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)
			lines := strings.Split(string(data), "\n")

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "#") {
					continue
				}

				// Check "from X import Y" pattern
				if matches := reFromImport.FindStringSubmatch(trimmed); len(matches) >= 2 {
					module := matches[1]
					// Skip relative imports
					if strings.HasPrefix(module, ".") {
						continue
					}
					if v.isRepoLocal(module) && !v.resolveImport(root, module) {
						issues = append(issues, fmt.Sprintf(
							"BROKEN_IMPORT: %s imports '%s' — target not found",
							relPath, module,
						))
					}
				}

				// Check "import X" pattern
				if matches := reImport.FindStringSubmatch(trimmed); len(matches) >= 2 {
					// Handle "import X, Y, Z" — split on comma
					for _, mod := range strings.Split(matches[1], ",") {
						mod = strings.TrimSpace(mod)
						if strings.HasPrefix(mod, ".") {
							continue
						}
						if v.isRepoLocal(mod) && !v.resolveImport(root, mod) {
							issues = append(issues, fmt.Sprintf(
								"BROKEN_IMPORT: %s imports '%s' — target not found",
								relPath, mod,
							))
						}
					}
				}
			}

			return nil
		})
	}

	return issues
}

// isRepoLocal checks if an import name matches a repo-local prefix.
func (v *CanonicalIntegrity) isRepoLocal(importName string) bool {
	for _, prefix := range repoLocalPrefixes {
		if strings.HasPrefix(importName, prefix) {
			return true
		}
	}
	return false
}

// resolveImport converts a dotted import path to a filesystem path and
// checks if it exists. Returns true if the module can be resolved.
func (v *CanonicalIntegrity) resolveImport(root, importPath string) bool {
	filePart := strings.ReplaceAll(importPath, ".", "/")
	candidates := []string{
		filepath.Join(root, filePart+".py"),
		filepath.Join(root, filePart, "__init__.py"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return true
		}
	}
	return false
}

var _ Validator = (*CanonicalIntegrity)(nil)
