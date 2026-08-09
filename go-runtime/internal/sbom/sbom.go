// Package sbom provides Software Bill of Materials generation and verification
// for OVAV supply chain security (F0.1 — Supply Chain Integrity).
//
// Generates SHA-256 hashes for all core project files, tracks Go module
// dependencies, and verifies integrity against a stored baseline.
// Common Criteria EAL7-guided: every dependency must be verifiable.
//
// Replaces tools/security/sbom.py (Python, 376 lines) with a Go-native,
// stdlib-only implementation.
package sbom

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	SchemaVersion    = "ovav.sbom.v2"
	HashAlgorithm    = "sha256"
	SBOMRegistry     = ".ovav/registry/sbom.json"
	RequirementsHash = "requirements.sha256"
)

// Core file globs scanned for SBOM tracking.
var coreFileGlobs = []string{
	"AGENTS.md",
	".ovav/**/*.json",
	".ovav/**/*.yaml",
	".ovav/**/*.yml",
	"go-runtime/**/*.go",
	"go-runtime/**/go.mod",
	"go-runtime/**/go.sum",
	"go-runtime/internal/runtimes/opencode/agents/**/*.md",
	"go-runtime/internal/agents/**/*.yaml",
	"clients/opencode/**/*.json",
	".opencode/skills/**/*.md",
	".opencode/**/*.json",
	".opencode/**/*.jsonc",
	".opencode/**/*.yaml",
	"docs-site/**/*.mdx",
	"docs-site/**/*.astro",
	"docs/**/*.md",
	"tools/cpanel/src/**/*.ts",
	"tools/cpanel/src/**/*.tsx",
}

// Directories excluded from SBOM tracking.
var sbomExcludeDirs = []string{
	".git",
	"node_modules",
	"__pycache__",
	".pytest_cache",
	".ovav/vault",
	".ovav/runtime",
	".ovav/cache",
	".ovav/lockdown",
	".ovav/context",
	".ovav/evaluation",
	".ovav/alerts",
	".ovav/quarantine",
	".ovav/integrity_backups",
	".ovav/sync",      // sync manifest is an operational staging artifact — changes on every sync, not a source artifact
	".ovav/governor",  // runtime-generated governance state — not committed to git, regenerated on startup
	".ovav/memory",    // runtime-generated memory/state files — not committed to git
	".ovav/integrity", // runtime-generated integrity state — not committed to git
	".ovav/security",  // runtime-generated security state — not committed to git
	".ovav/verify",    // runtime-generated verification state — not committed to git
	".ovav/prompt",    // runtime-generated prompt state — not committed to git
	".ovav/config",    // runtime-generated config state — not committed to git
	".opencode",       // operational harness files — not committed to git, managed by the system
	"dist",
	"vendor",
	".wrangler",
}

// Self-referential files excluded from tracking.
var sbomExcludeFiles = []string{
	".ovav/registry/sbom.json",
	".ovav/registry/sbom.yaml",
	".ovav/registry/core_hashes.yaml",
	".ovav/runtime/integrity_status.json",
	".ovav/registry/alignment_progression.yaml",        // runtime-generated registry, not committed to git
	".ovav/knowledge/feedback_report.json",             // runtime-generated knowledge, not committed to git
	"clients/opencode/node_modules/.package-lock.json", // harness artifact, not in git
	"docs/build/BUILD17_FINAL_REVIEW_AND_HANDOFF.md",   // generated doc artifact
}

// ── Data Structures ──────────────────────────────────────────────────────────

// SBOM represents a Software Bill of Materials.
type SBOM struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Project       string            `json:"project"`
	HashAlgorithm string            `json:"hash_algorithm"`
	Metadata      SBOMetadata       `json:"metadata"`
	Dependencies  SBOMDependencies  `json:"dependencies"`
	CoreFiles     map[string]string `json:"core_files"`
}

// SBOMetadata holds SBOM generation context.
type SBOMetadata struct {
	RootPath  string `json:"root_path"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version,omitempty"`
}

// SBOMDependencies holds categorized dependencies.
type SBOMDependencies struct {
	Go     []DepEntry `json:"go"`
	Python []DepEntry `json:"python,omitempty"`
	System []DepEntry `json:"system"`
}

// DepEntry represents a single dependency.
type DepEntry struct {
	Name    string `json:"name"`
	Group   string `json:"group,omitempty"`
	Version string `json:"version"`
}

// VerifyResult holds the outcome of SBOM verification.
type VerifyResult struct {
	Valid      bool     `json:"valid"`
	TotalFiles int      `json:"total_files"`
	Mismatches []string `json:"mismatches,omitempty"`
}

// ── Core Functions ───────────────────────────────────────────────────────────

// Generate creates a new SBOM for the project at root.
func Generate(root string) (*SBOM, error) {
	sbom := &SBOM{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Project:       "OVAV",
		HashAlgorithm: HashAlgorithm,
		Metadata: SBOMetadata{
			RootPath:  root,
			GitCommit: gitCommit(root),
		},
		Dependencies: SBOMDependencies{
			Go:     discoverGoDeps(root),
			Python: discoverPythonDeps(root),
			System: discoverSystemDeps(),
		},
		CoreFiles: make(map[string]string),
	}

	// Hash core files
	for _, glob := range coreFileGlobs {
		matches, err := walkGlob(root, glob)
		if err != nil {
			continue
		}
		for _, fpath := range matches {
			// fpath from walkGlob is relative; make it absolute
			absPath := filepath.Join(root, fpath)
			info, err := os.Stat(absPath)
			if err != nil || info.IsDir() {
				continue
			}
			if isExcluded(fpath) {
				continue
			}
			h, err := fileHash(absPath)
			if err != nil {
				sbom.CoreFiles[fpath] = "ERROR:" + err.Error()
			} else {
				sbom.CoreFiles[fpath] = h
			}
		}
	}

	return sbom, nil
}

// Load reads an existing SBOM from the registry.
func Load(root string) (*SBOM, error) {
	path := filepath.Join(root, SBOMRegistry)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("SBOM load: %w", err)
	}
	var sbom SBOM
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, fmt.Errorf("SBOM parse: %w", err)
	}
	return &sbom, nil
}

// Save writes the SBOM to the registry.
func (s *SBOM) Save(root string) error {
	path := filepath.Join(root, SBOMRegistry)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("SBOM save: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("SBOM marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Verify checks current file hashes against the stored baseline.
func Verify(root string) (*VerifyResult, error) {
	baseline, err := Load(root)
	if err != nil {
		return nil, fmt.Errorf("SBOM baseline not found — run 'ovav sbom generate' first: %w", err)
	}

	result := &VerifyResult{Valid: true}
	var mismatches []string

	// Check each tracked file
	seen := make(map[string]bool)
	for rel, expectedHash := range baseline.CoreFiles {
		fpath := filepath.Join(root, rel)
		currentHash, err := fileHash(fpath)
		if err != nil {
			mismatches = append(mismatches, fmt.Sprintf("MISSING: %s", rel))
			continue
		}
		if currentHash != expectedHash {
			mismatches = append(mismatches, fmt.Sprintf("MODIFIED: %s", rel))
		}
		seen[rel] = true
	}

	// Check for untracked new files
	for _, glob := range coreFileGlobs {
		matches, _ := walkGlob(root, glob)
		for _, fpath := range matches {
			// fpath from walkGlob is relative; make it absolute
			absPath := filepath.Join(root, fpath)
			info, err := os.Stat(absPath)
			if err != nil || info.IsDir() {
				continue
			}
			if isExcluded(fpath) {
				continue
			}
			if !seen[fpath] {
				mismatches = append(mismatches, fmt.Sprintf("UNTRACKED: %s", fpath))
			}
		}
	}

	result.Mismatches = mismatches
	result.TotalFiles = len(baseline.CoreFiles)
	result.Valid = len(mismatches) == 0
	return result, nil
}

// ── Hashing ──────────────────────────────────────────────────────────────────

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ── Dependency Discovery ─────────────────────────────────────────────────────

func discoverGoDeps(root string) []DepEntry {
	var deps []DepEntry
	goModPath := filepath.Join(root, "go-runtime", "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return deps
	}
	lines := strings.Split(string(data), "\n")
	inRequire := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				deps = append(deps, DepEntry{
					Name:    parts[0],
					Version: parts[1],
					Group:   "go",
				})
			}
		}
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps
}

func discoverPythonDeps(root string) []DepEntry {
	var deps []DepEntry
	reqPath := filepath.Join(root, "requirements.txt")
	data, err := os.ReadFile(reqPath)
	if err != nil {
		return deps
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		parts := strings.Split(trimmed, "==")
		name := strings.TrimSpace(parts[0])
		version := "unknown"
		if len(parts) > 1 {
			version = strings.TrimSpace(parts[1])
		}
		deps = append(deps, DepEntry{
			Name:    name,
			Version: version,
			Group:   "python",
		})
	}
	return deps
}

func discoverSystemDeps() []DepEntry {
	return []DepEntry{
		{Name: "go", Group: "system", Version: "stdlib-only"},
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// walkGlob recursively matches files using ** for recursive descent.
// filepath.Glob doesn't support ** in Go stdlib, so we implement it here.
func walkGlob(root, pattern string) ([]string, error) {
	var matches []string
	pattern = filepath.FromSlash(pattern)
	
	// Handle ** by splitting and recursively walking
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")
		
		baseDir := root
		if prefix != "" {
			baseDir = filepath.Join(root, prefix)
		}
		
		filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			
			// Check if file matches suffix pattern
			if matched, _ := filepath.Match(suffix, filepath.Base(rel)); matched {
				matches = append(matches, rel)
			}
			return nil
		})
		return matches, nil
	}
	
	// Standard glob for non-** patterns
	return filepath.Glob(filepath.Join(root, pattern))
}

func isExcluded(relPath string) bool {
	for _, excl := range sbomExcludeDirs {
		if relPath == excl || strings.Contains(relPath, "/"+excl+"/") || strings.HasPrefix(relPath, excl+"/") {
			return true
		}
	}
	for _, excl := range sbomExcludeFiles {
		if relPath == excl {
			return true
		}
	}
	if strings.Contains(relPath, "/__pycache__/") || strings.Contains(relPath, "/.pytest_cache/") {
		return true
	}
	return false
}

func gitCommit(root string) string {
	headPath := filepath.Join(root, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "unknown"
	}
	ref := strings.TrimSpace(string(data))
	// If it's a ref, try to read the actual commit
	if strings.HasPrefix(ref, "ref: ") {
		refPath := strings.TrimPrefix(ref, "ref: ")
		commitData, err := os.ReadFile(filepath.Join(root, ".git", refPath))
		if err != nil {
			// Worktrees: .git is a file pointing to the main repo
			// Try reading .git as a file to get gitdir path
			if gitFileData, err2 := os.ReadFile(filepath.Join(root, ".git")); err2 == nil {
				gitFileContent := strings.TrimSpace(string(gitFileData))
				if strings.HasPrefix(gitFileContent, "gitdir: ") {
					gitDir := strings.TrimPrefix(gitFileContent, "gitdir: ")
					commitData, err = os.ReadFile(filepath.Join(gitDir, refPath))
				}
			}
			if err != nil {
				return "unknown"
			}
		}
		return strings.TrimSpace(string(commitData))
	}
	return ref
}

// ComputeRequirementsHash generates a combined hash of all dependency declarations.
func ComputeRequirementsHash(root string) string {
	h := sha256.New()
	// Go mod files
	for _, f := range []string{"go-runtime/go.mod", "go-runtime/go.sum"} {
		if data, err := os.ReadFile(filepath.Join(root, f)); err == nil {
			h.Write(data)
		}
	}
	// Python requirements
	if data, err := os.ReadFile(filepath.Join(root, "requirements.txt")); err == nil {
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}
