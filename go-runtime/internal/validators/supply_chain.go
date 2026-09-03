package validators

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/sbom"
)

type ValidationMode string

const (
	ValidationDeveloper ValidationMode = "developer"
	ValidationGate      ValidationMode = "gate"
)

// SupplyChain verifies the HEAD-anchored SBOM without mutating it.
type SupplyChain struct{ mode ValidationMode }

func NewSupplyChain(modes ...ValidationMode) *SupplyChain {
	mode := ValidationDeveloper
	if len(modes) > 0 {
		mode = modes[0]
	}
	return &SupplyChain{mode: mode}
}

func (s *SupplyChain) ID() string   { return "supply_chain" }
func (s *SupplyChain) Name() string { return "Supply Chain Integrity" }
func (s *SupplyChain) Description() string {
	return "Verifies the canonical SBOM against git HEAD and reports worktree drift separately"
}
func (s *SupplyChain) Weight() int { return 20 }

// Mode reports whether drift is being evaluated for developer feedback or as a gate.
func (s *SupplyChain) Mode() ValidationMode { return s.mode }

func (s *SupplyChain) Validate(_ context.Context, root string) Result {
	start := time.Now()
	var failures, warnings []string
	goSum, err := headOrFilesystemFile(root, "go-runtime/go.sum")
	if os.IsNotExist(err) {
		failures = append(failures, "MISSING: go-runtime/go.sum — Go module checksums not found")
	} else if err != nil {
		failures = append(failures, fmt.Sprintf("ERROR: Cannot read go.sum: %v", err))
	} else if len(goSum) == 0 {
		failures = append(failures, "EMPTY: go-runtime/go.sum — no dependency hashes")
	}
	if _, err := headOrFilesystemFile(root, "go-runtime/go.mod"); os.IsNotExist(err) {
		failures = append(failures, "MISSING: go-runtime/go.mod — Go module definition not found")
	}

	if result, err := sbom.Verify(root); err != nil {
		failures = append(failures, "baseline_invalid: "+err.Error())
	} else {
		for _, issue := range result.BaselineIssues {
			failures = append(failures, "baseline_invalid: "+issue)
		}
		for _, warning := range result.WorktreeWarnings {
			issue := "working_tree_drift: " + warning
			if s.mode == ValidationGate && sensitiveCandidateDrift(warning) {
				failures = append(failures, issue)
			} else {
				warnings = append(warnings, issue)
			}
		}
	}

	for _, path := range trackedSuspiciousBinaries(root) {
		failures = append(failures, "SUSPICIOUS: tracked binary file in HEAD: "+path)
	}
	sort.Strings(failures)
	sort.Strings(warnings)
	issues := append(append([]string(nil), failures...), warnings...)
	if len(failures) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(), Message: fmt.Sprintf("FAIL supply chain integrity — %d baseline/security issue(s), %d worktree warning(s)", len(failures), len(warnings)), Issues: issues, Duration: time.Since(start)}
	}
	if len(warnings) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "warn", Weight: s.Weight(), Message: fmt.Sprintf("WARN supply chain integrity — %d working-tree drift item(s)", len(warnings)), Issues: warnings, Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(), Message: "PASS supply chain integrity", Duration: time.Since(start)}
}

func sensitiveCandidateDrift(issue string) bool {
	path := issue
	if separator := strings.Index(issue, ": "); separator >= 0 {
		path = issue[separator+2:]
	}
	path = strings.ToLower(filepath.ToSlash(path))
	if strings.HasPrefix(path, ".ovav/") || strings.HasPrefix(path, "go-runtime/") {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".dll", ".so", ".dylib":
		return true
	}
	return path == "opencode.json" || path == "requirements.txt" || path == "go.mod" || path == "go.sum" || strings.HasSuffix(path, "/go.mod") || strings.HasSuffix(path, "/go.sum")
}

// IsSensitiveCandidateDrift exposes the gate classification for CLI reporting.
func IsSensitiveCandidateDrift(issue string) bool { return sensitiveCandidateDrift(issue) }

func headOrFilesystemFile(root, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(path))
	cmd.Dir = root
	if data, err := cmd.Output(); err == nil {
		return data, nil
	}
	return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
}

func trackedSuspiciousBinaries(root string) []string {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return filesystemSuspiciousBinaries(root)
	}
	suspicious := map[string]bool{".exe": true, ".dll": true, ".so": true, ".dylib": true}
	var paths []string
	for _, path := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if suspicious[strings.ToLower(filepath.Ext(path))] {
			paths = append(paths, filepath.ToSlash(path))
		}
	}
	sort.Strings(paths)
	return paths
}

func filesystemSuspiciousBinaries(root string) []string {
	suspicious := map[string]bool{".exe": true, ".dll": true, ".so": true, ".dylib": true}
	var paths []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil && suspicious[strings.ToLower(filepath.Ext(path))] {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(paths)
	return paths
}

var _ Validator = (*SupplyChain)(nil)
