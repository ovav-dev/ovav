package validators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RuntimeIntegrity verifies protected runtime files without creating a baseline.
type RuntimeIntegrity struct{ mode ValidationMode }

const IntegrityBaselineSchema = "ovav.runtime_integrity.v1"

func NewRuntimeIntegrity(modes ...ValidationMode) *RuntimeIntegrity {
	mode := ValidationDeveloper
	if len(modes) > 0 {
		mode = modes[0]
	}
	return &RuntimeIntegrity{mode: mode}
}

func (r *RuntimeIntegrity) ID() string   { return "runtime_integrity" }
func (r *RuntimeIntegrity) Name() string { return "Runtime Integrity" }
func (r *RuntimeIntegrity) Description() string {
	return "Verifies protected runtime files against an explicit hash baseline and git state"
}
func (r *RuntimeIntegrity) Weight() int { return 20 }

// Mode reports whether drift is being evaluated for developer feedback or as a gate.
func (r *RuntimeIntegrity) Mode() ValidationMode { return r.mode }

var coreFiles = []string{
	"AGENTS.md",
	"opencode.json",
	".ovav/policy/permission_authority.json",
	".ovav/plan/caps.yaml",
	"go-runtime/go.mod",
	"go-runtime/internal/validators/cmd/validate/main.go",
}

type IntegrityBaseline struct {
	Schema    string            `json:"schema"`
	Algorithm string            `json:"algorithm"`
	Files     map[string]string `json:"files"`
	JSON      []byte            `json:"-"`
}

func baselinePath(root string) string {
	return filepath.Join(root, ".ovav", "integrity_backups", "baseline.json")
}

func (r *RuntimeIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var failures, warnings []string
	if err := ctx.Err(); err != nil {
		failures = append(failures, "validation context unavailable")
	}

	baseline, baselineErr := loadIntegrityBaseline(root)
	if baselineErr != nil {
		issue := "integrity baseline missing or invalid; validation did not create one: " + baselineErr.Error()
		if r.mode == ValidationGate {
			failures = append(failures, issue)
		} else {
			warnings = append(warnings, issue)
		}
	}

	for _, rel := range coreFiles {
		path := filepath.Join(root, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("core file missing or unreadable: %s", rel))
			continue
		}
		status, tracked := gitPathStatus(root, rel)
		if !tracked {
			failures = append(failures, fmt.Sprintf("untracked core file: %s", rel))
		}
		if baselineErr == nil {
			expected, ok := baseline.Files[rel]
			if !ok {
				failures = append(failures, fmt.Sprintf("protected surface absent from baseline: %s", rel))
			} else if digest(data) != expected {
				issue := fmt.Sprintf("protected surface drift: %s", rel)
				if r.mode == ValidationGate {
					failures = append(failures, issue)
				} else {
					warnings = append(warnings, issue)
				}
			}
		}
		if status != "" && r.mode == ValidationDeveloper {
			warnings = append(warnings, fmt.Sprintf("scoped developer modification: %s (%s)", rel, status))
		}
	}

	sort.Strings(failures)
	sort.Strings(warnings)
	issues := append(append([]string(nil), failures...), warnings...)
	if len(failures) > 0 {
		return Result{ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(), Message: fmt.Sprintf("FAIL runtime integrity — %d failure(s), %d warning(s)", len(failures), len(warnings)), Issues: issues, Duration: time.Since(start)}
	}
	if len(warnings) > 0 {
		return Result{ID: r.ID(), Name: r.Name(), Status: "warn", Weight: r.Weight(), Message: fmt.Sprintf("WARN runtime integrity baseline/developer scope — %d warning(s)", len(warnings)), Issues: warnings, Duration: time.Since(start)}
	}
	return Result{ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(), Message: fmt.Sprintf("PASS runtime integrity — %d protected files match baseline", len(coreFiles)), Duration: time.Since(start)}
}

func loadIntegrityBaseline(root string) (IntegrityBaseline, error) {
	data, err := os.ReadFile(baselinePath(root))
	if err != nil {
		return IntegrityBaseline{}, err
	}
	var baseline IntegrityBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return IntegrityBaseline{}, err
	}
	if baseline.Schema != IntegrityBaselineSchema || (baseline.Algorithm != "" && baseline.Algorithm != "sha256") || len(baseline.Files) == 0 {
		return IntegrityBaseline{}, fmt.Errorf("unsupported or empty baseline")
	}
	for path, hash := range baseline.Files {
		if path == "" || !validDigest(hash) {
			return IntegrityBaseline{}, fmt.Errorf("invalid baseline entry")
		}
	}
	return baseline, nil
}

// PlanIntegrityBaseline hashes the deterministic protected surface without writing it.
func PlanIntegrityBaseline(root string) (IntegrityBaseline, error) {
	baseline := IntegrityBaseline{
		Schema:    IntegrityBaselineSchema,
		Algorithm: "sha256",
		Files:     make(map[string]string, len(coreFiles)),
	}
	for _, rel := range coreFiles {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return IntegrityBaseline{}, fmt.Errorf("read protected surface %s: %w", rel, err)
		}
		if _, tracked := gitPathStatus(root, rel); !tracked {
			return IntegrityBaseline{}, fmt.Errorf("protected surface is not tracked: %s", rel)
		}
		baseline.Files[rel] = digest(data)
	}
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return IntegrityBaseline{}, fmt.Errorf("encode integrity baseline: %w", err)
	}
	baseline.JSON = append(data, '\n')
	return baseline, nil
}

// WriteIntegrityBaseline writes a planned baseline only from a safe feature candidate.
func WriteIntegrityBaseline(root string) (IntegrityBaseline, error) {
	gitInfo, err := os.Stat(filepath.Join(root, ".git"))
	if err != nil || !gitInfo.Mode().IsRegular() {
		return IntegrityBaseline{}, fmt.Errorf("integrity baseline write requires an isolated git worktree")
	}
	branch := getCurrentBranch(root)
	if !strings.HasPrefix(branch, "feature/") {
		return IntegrityBaseline{}, fmt.Errorf("integrity baseline write requires a feature branch, got %q", branch)
	}
	if err := requireSafeBaselineCandidate(root); err != nil {
		return IntegrityBaseline{}, err
	}
	baseline, err := PlanIntegrityBaseline(root)
	if err != nil {
		return IntegrityBaseline{}, err
	}
	path := baselinePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return IntegrityBaseline{}, fmt.Errorf("create integrity baseline directory: %w", err)
	}
	if err := os.WriteFile(path, baseline.JSON, 0o644); err != nil {
		return IntegrityBaseline{}, fmt.Errorf("write integrity baseline: %w", err)
	}
	return baseline, nil
}

func requireSafeBaselineCandidate(root string) error {
	unstaged := exec.Command("git", "diff", "--quiet", "--")
	unstaged.Dir = root
	if err := unstaged.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return fmt.Errorf("integrity baseline write rejects unstaged candidate changes; stage the exact candidate first")
		}
		return fmt.Errorf("check unstaged workspace safety: %w", err)
	}
	untracked := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untracked.Dir = root
	output, err := untracked.Output()
	if err != nil {
		return fmt.Errorf("check untracked workspace safety: %w", err)
	}
	if strings.TrimSpace(string(output)) != "" {
		return fmt.Errorf("integrity baseline write rejects untracked candidate changes; stage the exact candidate first")
	}
	staged := exec.Command("git", "diff", "--cached", "--quiet", "--")
	staged.Dir = root
	if err := staged.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return fmt.Errorf("check staged candidate safety: %w", err)
	}
	if !featureCandidateCommitted(root) {
		return fmt.Errorf("integrity baseline write requires an exact staged candidate or a governed candidate commit")
	}
	return nil
}

func featureCandidateCommitted(root string) bool {
	for _, ref := range []string{
		"refs/heads/develop",
		"refs/heads/main",
		"refs/heads/master",
		"refs/remotes/origin/develop",
		"refs/remotes/origin/main",
		"refs/remotes/origin/master",
	} {
		exists := exec.Command("git", "show-ref", "--verify", "--quiet", ref)
		exists.Dir = root
		if exists.Run() != nil {
			continue
		}
		count := exec.Command("git", "rev-list", "--count", ref+"..HEAD")
		count.Dir = root
		output, err := count.Output()
		return err == nil && strings.TrimSpace(string(output)) != "0"
	}
	return false
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func gitPathStatus(root, rel string) (string, bool) {
	trackedCmd := exec.Command("git", "cat-file", "-e", "HEAD:"+filepath.ToSlash(rel))
	trackedCmd.Dir = root
	tracked := trackedCmd.Run() == nil
	statusCmd := exec.Command("git", "status", "--porcelain=v1", "--", rel)
	statusCmd.Dir = root
	out, err := statusCmd.Output()
	if err != nil {
		return "git-status-error", tracked
	}
	return strings.TrimSpace(string(out)), tracked
}

var _ Validator = (*RuntimeIntegrity)(nil)
