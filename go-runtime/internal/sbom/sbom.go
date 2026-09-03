// Package sbom provides the canonical OVAV software bill of materials.
package sbom

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	SchemaVersion    = "ovav.sbom.v4"
	PolicyVersion    = "tracked-content-v1"
	HashAlgorithm    = "sha256"
	SBOMRegistry     = ".ovav/registry/sbom.json"
	RequirementsHash = "requirements.sha256"
)

var sbomExcludeDirs = []string{
	".git",
	".ovav/alerts",
	".ovav/artifacts",
	".ovav/cache",
	".ovav/context",
	".ovav/evaluation",
	".ovav/governor",
	".ovav/history",
	".ovav/integrity",
	".ovav/integrity_backups",
	".ovav/lockdown",
	".ovav/memory",
	".ovav/prompt",
	".ovav/quarantine",
	".ovav/runtime",
	".ovav/security",
	".ovav/sync",
	".ovav/vault",
	".ovav/verify",
	".ovav/worktrees",
	".pytest_cache",
	".venv",
	".wrangler",
	"__pycache__",
	"dist",
	"go-runtime/build",
	"artifacts",
	"node_modules",
	"vendor",
}

var sbomExcludeFiles = []string{
	SBOMRegistry,
	".ovav/registry/sbom.yaml",
	".ovav/registry/core_hashes.yaml",
}

type SBOM struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Project       string            `json:"project"`
	HashAlgorithm string            `json:"hash_algorithm"`
	Metadata      SBOMetadata       `json:"metadata"`
	Policy        SBOMPolicy        `json:"policy"`
	Dependencies  SBOMDependencies  `json:"dependencies"`
	CoreFiles     map[string]string `json:"core_files"`
}

type SBOMetadata struct {
	RootPath    string `json:"root_path,omitempty"`
	GitIdentity string `json:"git_identity"`
	GoVersion   string `json:"go_version,omitempty"`
}

type SBOMPolicy struct {
	Version             string   `json:"version"`
	Source              string   `json:"source"`
	ExcludedDirectories []string `json:"excluded_directories"`
	ExcludedFiles       []string `json:"excluded_files"`
}

type SBOMDependencies struct {
	Go     []DepEntry `json:"go"`
	Python []DepEntry `json:"python,omitempty"`
	System []DepEntry `json:"system"`
}

type DepEntry struct {
	Name    string `json:"name"`
	Group   string `json:"group,omitempty"`
	Version string `json:"version"`
}

type VerifyResult struct {
	Valid            bool     `json:"valid"`
	TotalFiles       int      `json:"total_files"`
	BaselineIssues   []string `json:"baseline_issues,omitempty"`
	WorktreeWarnings []string `json:"worktree_warnings,omitempty"`
	Mismatches       []string `json:"mismatches,omitempty"`
}

// Generate creates a byte-deterministic baseline from tracked content at HEAD.
// The identity excludes the SBOM itself, so committing a freshly saved baseline
// does not invalidate that baseline.
func Generate(root string) (*SBOM, error) {
	identity, err := trackedContentIdentity(root)
	if err != nil {
		return generateFilesystemFallback(root), nil
	}
	paths, err := trackedHEADFiles(root)
	if err != nil {
		return nil, err
	}
	baseline := &SBOM{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Unix(0, 0).UTC().Format(time.RFC3339),
		Project:       "OVAV",
		HashAlgorithm: HashAlgorithm,
		Metadata:      SBOMetadata{GitIdentity: identity},
		Policy: SBOMPolicy{
			Version:             PolicyVersion,
			Source:              "git HEAD tracked content identity excluding SBOM plus declared dependency metadata",
			ExcludedDirectories: append([]string(nil), sbomExcludeDirs...),
			ExcludedFiles:       append([]string(nil), sbomExcludeFiles...),
		},
		Dependencies: dependenciesAtHEAD(root),
		CoreFiles:    make(map[string]string, len(paths)),
	}
	for _, path := range paths {
		if isExcluded(path) {
			continue
		}
		data, err := gitBlob(root, path)
		if err != nil {
			return nil, fmt.Errorf("read HEAD file %s: %w", path, err)
		}
		baseline.CoreFiles[path] = hashBytes(data)
	}
	return baseline, nil
}

func Load(root string) (*SBOM, error) {
	data, err := os.ReadFile(filepath.Join(root, SBOMRegistry))
	if err != nil {
		return nil, fmt.Errorf("SBOM load: %w", err)
	}
	var baseline SBOM
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("SBOM parse: %w", err)
	}
	return &baseline, nil
}

func (s *SBOM) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("SBOM marshal: %w", err)
	}
	return append(data, '\n'), nil
}

func (s *SBOM) Save(root string) error {
	data, err := s.Marshal()
	if err != nil {
		return err
	}
	path := filepath.Join(root, SBOMRegistry)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("SBOM save: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("SBOM save: %w", err)
	}
	return nil
}

// Verify compares the saved baseline to HEAD, then independently reports worktree drift.
func Verify(root string) (*VerifyResult, error) {
	baseline, err := Load(root)
	if err != nil {
		return nil, fmt.Errorf("SBOM baseline not found — run 'ovav sbom generate' first: %w", err)
	}
	headPaths, err := trackedHEADFiles(root)
	if err != nil {
		return verifyFilesystemFallback(root, baseline), nil
	}
	headSet := make(map[string]bool, len(headPaths))
	result := &VerifyResult{Valid: true, TotalFiles: len(baseline.CoreFiles)}
	headIdentity, identityErr := trackedContentIdentity(root)
	if baseline.SchemaVersion != SchemaVersion {
		result.BaselineIssues = append(result.BaselineIssues, "SCHEMA_MISMATCH: expected "+SchemaVersion)
	}
	if baseline.HashAlgorithm != HashAlgorithm {
		result.BaselineIssues = append(result.BaselineIssues, "HASH_ALGORITHM_MISMATCH: expected "+HashAlgorithm)
	}
	if identityErr != nil || baseline.Metadata.GitIdentity != headIdentity {
		result.BaselineIssues = append(result.BaselineIssues, "CONTENT_IDENTITY_MISMATCH: baseline "+baseline.Metadata.GitIdentity+" current "+headIdentity)
	}
	expectedPolicy := SBOMPolicy{
		Version:             PolicyVersion,
		Source:              "git HEAD tracked content identity excluding SBOM plus declared dependency metadata",
		ExcludedDirectories: append([]string(nil), sbomExcludeDirs...),
		ExcludedFiles:       append([]string(nil), sbomExcludeFiles...),
	}
	if !reflect.DeepEqual(baseline.Policy, expectedPolicy) {
		result.BaselineIssues = append(result.BaselineIssues, "POLICY_MISMATCH: exclusions or policy version differ")
	}
	if !reflect.DeepEqual(baseline.Dependencies, dependenciesAtHEAD(root)) {
		result.BaselineIssues = append(result.BaselineIssues, "DEPENDENCY_METADATA_MISMATCH: declarations differ from HEAD")
	}
	for _, path := range headPaths {
		if !isExcluded(path) {
			headSet[path] = true
		}
	}
	for path, expected := range baseline.CoreFiles {
		if isExcluded(path) || !headSet[path] {
			result.BaselineIssues = append(result.BaselineIssues, "OBSOLETE_BASELINE: "+path)
			continue
		}
		data, err := gitBlob(root, path)
		if err != nil || hashBytes(data) != expected {
			result.BaselineIssues = append(result.BaselineIssues, "HASH_MISMATCH: "+path)
		}
	}
	for _, path := range sortedKeys(headSet) {
		if _, ok := baseline.CoreFiles[path]; !ok {
			result.BaselineIssues = append(result.BaselineIssues, "UNEXPECTED_TRACKED: "+path)
		}
	}
	status, err := gitRawOutput(root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return verifyFilesystemFallback(root, baseline), nil
	}
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 4 {
			continue
		}
		path := normalizeStatusPath(line[3:])
		if path == "" || isExcluded(path) {
			continue
		}
		kind := "WORKTREE_MODIFIED"
		if strings.HasPrefix(line, "??") {
			kind = "WORKTREE_UNTRACKED"
		} else if line[0] == 'D' || line[1] == 'D' {
			kind = "WORKTREE_DELETED"
		}
		result.WorktreeWarnings = append(result.WorktreeWarnings, kind+": "+path)
	}
	sort.Strings(result.BaselineIssues)
	sort.Strings(result.WorktreeWarnings)
	result.Mismatches = append(append([]string(nil), result.BaselineIssues...), result.WorktreeWarnings...)
	result.Valid = len(result.BaselineIssues) == 0
	return result, nil
}

func dependenciesAtHEAD(root string) SBOMDependencies {
	return SBOMDependencies{
		Go:     discoverGoDeps(root),
		Python: discoverPythonDeps(root),
		System: discoverSystemDeps(),
	}
}

func discoverGoDeps(root string) []DepEntry {
	data, err := gitBlob(root, "go-runtime/go.mod")
	if err != nil {
		data, _ = os.ReadFile(filepath.Join(root, "go-runtime", "go.mod"))
	}
	var deps []DepEntry
	inRequire := false
	for _, line := range strings.Split(string(data), "\n") {
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
				deps = append(deps, DepEntry{Name: parts[0], Version: parts[1], Group: "go"})
			}
		}
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps
}

func discoverPythonDeps(root string) []DepEntry {
	data, err := gitBlob(root, "requirements.txt")
	if err != nil {
		data, _ = os.ReadFile(filepath.Join(root, "requirements.txt"))
	}
	var deps []DepEntry
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		parts := strings.SplitN(trimmed, "==", 2)
		version := "unknown"
		if len(parts) == 2 {
			version = strings.TrimSpace(parts[1])
		}
		deps = append(deps, DepEntry{Name: strings.TrimSpace(parts[0]), Version: version, Group: "python"})
	}
	return deps
}

func discoverSystemDeps() []DepEntry {
	return []DepEntry{{Name: "go", Group: "system", Version: "stdlib-only"}}
}

func trackedHEADFiles(root string) ([]string, error) {
	output, err := gitOutput(root, "ls-tree", "-r", "--name-only", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("list HEAD files: %w", err)
	}
	var paths []string
	for _, path := range strings.Split(output, "\n") {
		path = filepath.ToSlash(strings.TrimSpace(path))
		if path != "" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func trackedContentIdentity(root string) (string, error) {
	paths, err := trackedHEADFiles(root)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, path := range paths {
		if isExcluded(path) {
			continue
		}
		data, err := gitBlob(root, path)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(hash, filepath.ToSlash(path))
		_, _ = hash.Write([]byte{0})
		_, _ = io.WriteString(hash, hashBytes(data))
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func gitBlob(root, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(path))
	cmd.Dir = root
	return cmd.Output()
}

func gitOutput(root string, args ...string) (string, error) {
	output, err := gitRawOutput(root, args...)
	return strings.TrimSpace(output), err
}

func gitRawOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	return string(output), err
}

func commitTime(root string) string {
	value, err := gitOutput(root, "show", "-s", "--format=%cI", "HEAD")
	if err != nil {
		return time.Unix(0, 0).UTC().Format(time.RFC3339)
	}
	return value
}

func normalizeStatusPath(path string) string {
	path = strings.TrimSpace(path)
	if arrow := strings.LastIndex(path, " -> "); arrow >= 0 {
		path = path[arrow+4:]
	}
	return filepath.ToSlash(strings.Trim(path, `"`))
}

func isExcluded(relPath string) bool {
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "./"))
	for _, excluded := range sbomExcludeDirs {
		if relPath == excluded || strings.HasPrefix(relPath, excluded+"/") || strings.Contains(relPath, "/"+excluded+"/") {
			return true
		}
	}
	for _, excluded := range sbomExcludeFiles {
		if relPath == excluded {
			return true
		}
	}
	return false
}

func generateFilesystemFallback(root string) *SBOM {
	baseline := &SBOM{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Unix(0, 0).UTC().Format(time.RFC3339),
		Project:       "OVAV",
		HashAlgorithm: HashAlgorithm,
		Metadata:      SBOMetadata{GitIdentity: "unknown"},
		Policy: SBOMPolicy{
			Version:             PolicyVersion,
			Source:              "filesystem fallback for non-git test fixtures",
			ExcludedDirectories: append([]string(nil), sbomExcludeDirs...),
			ExcludedFiles:       append([]string(nil), sbomExcludeFiles...),
		},
		Dependencies: dependenciesAtHEAD(root),
		CoreFiles:    map[string]string{},
	}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if isExcluded(rel) {
			return nil
		}
		hash, hashErr := fileHash(path)
		if hashErr != nil {
			baseline.CoreFiles[rel] = "ERROR:" + hashErr.Error()
		} else {
			baseline.CoreFiles[rel] = hash
		}
		return nil
	})
	return baseline
}

func verifyFilesystemFallback(root string, baseline *SBOM) *VerifyResult {
	current := generateFilesystemFallback(root)
	result := &VerifyResult{Valid: true, TotalFiles: len(baseline.CoreFiles)}
	for path, expected := range baseline.CoreFiles {
		actual, ok := current.CoreFiles[path]
		if !ok {
			result.BaselineIssues = append(result.BaselineIssues, "MISSING: "+path)
		} else if actual != expected {
			result.BaselineIssues = append(result.BaselineIssues, "MODIFIED: "+path)
		}
	}
	for path := range current.CoreFiles {
		if _, ok := baseline.CoreFiles[path]; !ok {
			result.BaselineIssues = append(result.BaselineIssues, "UNTRACKED: "+path)
		}
	}
	sort.Strings(result.BaselineIssues)
	result.Mismatches = append([]string(nil), result.BaselineIssues...)
	result.Valid = len(result.BaselineIssues) == 0
	return result
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func gitCommit(root string) string {
	commit, err := gitOutput(root, "rev-parse", "HEAD")
	if err == nil {
		return commit
	}
	headPath := filepath.Join(root, ".git", "HEAD")
	data, readErr := os.ReadFile(headPath)
	if readErr != nil {
		return "unknown"
	}
	ref := strings.TrimSpace(string(data))
	if !strings.HasPrefix(ref, "ref: ") {
		return ref
	}
	refData, refErr := os.ReadFile(filepath.Join(root, ".git", strings.TrimPrefix(ref, "ref: ")))
	if refErr != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(refData))
}

func ComputeRequirementsHash(root string) string {
	hash := sha256.New()
	for _, path := range []string{"go-runtime/go.mod", "go-runtime/go.sum", "requirements.txt"} {
		data, err := gitBlob(root, path)
		if err != nil {
			data, _ = os.ReadFile(filepath.Join(root, path))
		}
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// Equal reports whether two generated baselines have identical canonical bytes.
func Equal(left, right *SBOM) bool {
	leftData, leftErr := left.Marshal()
	rightData, rightErr := right.Marshal()
	return leftErr == nil && rightErr == nil && bytes.Equal(leftData, rightData)
}
