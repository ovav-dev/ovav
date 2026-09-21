package validators

import (
	"bytes"
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

	"github.com/ovav/ovav/internal/consumers"
)

// ExternalRegistration is a fail-closed guard. A repository cannot silently
// fall back to the OVAV monorepo profile or invent its own governance source.
type ExternalRegistration struct{}

func NewExternalRegistration() *ExternalRegistration { return &ExternalRegistration{} }
func (e *ExternalRegistration) ID() string           { return "external_registration" }
func (e *ExternalRegistration) Name() string         { return "External Consumer Registration" }
func (e *ExternalRegistration) Description() string {
	return "Requires an active central OVAV consumer registration"
}
func (e *ExternalRegistration) Weight() int { return 25 }
func (e *ExternalRegistration) Validate(_ context.Context, root string) Result {
	start := time.Now()
	profile := consumers.Resolve(root)
	if profile.Active() {
		return Result{ID: e.ID(), Name: e.Name(), Status: "pass", Weight: e.Weight(),
			Message: "PASS external consumer registration — central profile active", Duration: time.Since(start)}
	}
	message := "external consumer registration unavailable"
	if profile.Err != nil {
		message += ": " + profile.Err.Error()
	}
	return Result{ID: e.ID(), Name: e.Name(), Status: "fail", Weight: e.Weight(), Message: "FAIL " + message,
		Issues: []string{"Register the repository in the central OVAV consumer registry; no project-local fallback is accepted"}, Duration: time.Since(start)}
}

// ExternalRegistry returns the security profile for an independent project.
// OVAV-internal architecture, agent, harness, and workstation validators are
// intentionally not included: absence of OVAV internals is not a project
// security finding. The remaining validators are real project gates.
func ExternalRegistry(mode ValidationMode) *Registry {
	return ExternalRegistryForRoot("", mode)
}

func ExternalRegistryForRoot(root string, mode ValidationMode) *Registry {
	return NewRegistry(
		NewExternalRegistration(),
		NewExternalSecretsHygiene(),
		NewExfilPatterns(),
		newExternalSupplyChain(mode),
		NewProtectedBranch(),
		newExternalGitPush(),
		NewWorkspaceSafetyForRoot(root),
		newExternalRuntimeIntegrity(mode),
	)
}

// GovernedPushValidators returns the smallest profile-specific push set. Both
// profiles retain the same transport, branch, workspace, supply-chain, and
// integrity gates; only OVAV-internal wiring checks are profile-specific.
func GovernedPushValidators(root string) []Validator {
	if consumers.Resolve(root).External {
		return ExternalRegistryForRoot(root, ValidationGate).All()
	}
	return []Validator{
		NewProtectedBranch(),
		NewGitPush(),
		NewWorkspaceSafety(),
		NewSupplyChain(ValidationGate),
		NewRuntimeIntegrity(ValidationGate),
	}
}

// DefaultRegistryForRoot selects the OVAV-internal or registered external
// profile. This selection is based on filesystem identity and central config,
// never on a project-controlled .ovav file.
func DefaultRegistryForRoot(root string, modes ...ValidationMode) *Registry {
	mode := ValidationDeveloper
	if len(modes) > 0 {
		mode = modes[0]
	}
	if consumers.IsOVAVRoot(root) {
		return DefaultRegistry(mode)
	}
	return ExternalRegistryForRoot(root, mode)
}

type externalSBOM struct {
	Schema   string            `json:"schema"`
	Project  string            `json:"project"`
	RootPath string            `json:"root_path"`
	Commit   string            `json:"commit"`
	Files    map[string]string `json:"files"`
}

type externalIntegrityBaseline struct {
	Schema   string            `json:"schema"`
	Project  string            `json:"project"`
	RootPath string            `json:"root_path"`
	Commit   string            `json:"commit"`
	Files    map[string]string `json:"files"`
}

const (
	externalSBOMSchema      = "ovav.external_sbom.v1"
	externalIntegritySchema = "ovav.external_integrity.v1"
	externalSBOMFile        = "sbom.json"
	externalIntegrityFile   = "integrity.json"
)

func externalProfile(root string) (consumers.Profile, error) {
	profile := consumers.Resolve(root)
	if !profile.External {
		return profile, nil
	}
	if !profile.Active() {
		if profile.Err != nil {
			return profile, profile.Err
		}
		return profile, fmt.Errorf("external consumer is not registered")
	}
	return profile, nil
}

func isRegisteredExternal(root string) bool {
	profile := consumers.Resolve(root)
	return profile.External && profile.Registered
}

func externalBaselinePath(root, name string) (string, error) {
	profile, err := externalProfile(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(profile.StateDir(), name)
	if profile.StateDir() == "" || filepath.Base(path) != name {
		return "", fmt.Errorf("central external baseline path unavailable")
	}
	return path, nil
}

func readExternalBaseline(root, name string, dst interface{}) error {
	path, err := externalBaselinePath(root, name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (s *SupplyChain) validateExternal(root string) Result {
	start := time.Now()
	var baseline externalSBOM
	if err := readExternalBaseline(root, externalSBOMFile, &baseline); err != nil {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: "FAIL external supply chain integrity — central SBOM missing or invalid",
			Issues:  []string{err.Error() + " — create it through the explicit OVAV consumer baseline flow"}, Duration: time.Since(start)}
	}
	current, err := planExternalSBOM(root)
	if err != nil {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(), Message: "FAIL external supply chain integrity — cannot inspect project HEAD", Issues: []string{err.Error()}, Duration: time.Since(start)}
	}
	issues := compareExternalFiles(baseline.Schema, externalSBOMSchema, baseline.Project, current.Project, baseline.RootPath, current.RootPath, baseline.Commit, current.Commit, baseline.Files, current.Files)
	if len(issues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(), Message: fmt.Sprintf("FAIL external supply chain integrity — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(), Message: fmt.Sprintf("PASS external supply chain integrity — %d tracked file(s) anchored to HEAD", len(current.Files)), Duration: time.Since(start)}
}

func (r *RuntimeIntegrity) validateExternal(root string) Result {
	start := time.Now()
	var baseline externalIntegrityBaseline
	if err := readExternalBaseline(root, externalIntegrityFile, &baseline); err != nil {
		return Result{ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(), Message: "FAIL external runtime integrity — central baseline missing or invalid", Issues: []string{err.Error() + " — create it through the explicit OVAV consumer baseline flow"}, Duration: time.Since(start)}
	}
	current, err := planExternalIntegrity(root)
	if err != nil {
		return Result{ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(), Message: "FAIL external runtime integrity — cannot inspect security surface", Issues: []string{err.Error()}, Duration: time.Since(start)}
	}
	issues := compareExternalFiles(baseline.Schema, externalIntegritySchema, baseline.Project, current.Project, baseline.RootPath, current.RootPath, baseline.Commit, current.Commit, baseline.Files, current.Files)
	if len(issues) > 0 {
		return Result{ID: r.ID(), Name: r.Name(), Status: "fail", Weight: r.Weight(), Message: fmt.Sprintf("FAIL external runtime integrity — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: r.ID(), Name: r.Name(), Status: "pass", Weight: r.Weight(), Message: fmt.Sprintf("PASS external runtime integrity — %d security file(s) match central baseline", len(current.Files)), Duration: time.Since(start)}
}

func compareExternalFiles(schema, expectedSchema, project, expectedProject, rootPath, expectedRootPath, commit, expectedCommit string, expected, actual map[string]string) []string {
	var issues []string
	if schema != expectedSchema {
		issues = append(issues, fmt.Sprintf("baseline schema mismatch: expected %s", expectedSchema))
	}
	if project != expectedProject {
		issues = append(issues, "baseline project identity mismatch")
	}
	if rootPath != expectedRootPath {
		issues = append(issues, "baseline repository identity mismatch")
	}
	if commit != expectedCommit {
		issues = append(issues, fmt.Sprintf("baseline commit mismatch: expected %s, got %s", commit, expectedCommit))
	}
	for path, hash := range expected {
		if actual[path] != hash {
			issues = append(issues, "baseline hash mismatch: "+path)
		}
	}
	for path := range actual {
		if _, ok := expected[path]; !ok {
			issues = append(issues, "new security surface not in baseline: "+path)
		}
	}
	sort.Strings(issues)
	return issues
}

func planExternalSBOM(root string) (externalSBOM, error) {
	paths, commit, err := externalHeadFiles(root)
	if err != nil {
		return externalSBOM{}, err
	}
	files := make(map[string]string, len(paths))
	for _, path := range paths {
		data, err := gitBlob(root, path)
		if err != nil {
			return externalSBOM{}, fmt.Errorf("read HEAD file %s: %w", path, err)
		}
		files[path] = externalDigest(data)
	}
	return externalSBOM{Schema: externalSBOMSchema, Project: externalProjectID(root), RootPath: canonicalExternalRoot(root), Commit: commit, Files: files}, nil
}

func planExternalIntegrity(root string) (externalIntegrityBaseline, error) {
	paths, commit, err := externalHeadFiles(root)
	if err != nil {
		return externalIntegrityBaseline{}, err
	}
	files := make(map[string]string)
	for _, path := range paths {
		if !isExternalSecuritySurface(path) {
			continue
		}
		data, err := gitBlob(root, path)
		if err != nil {
			return externalIntegrityBaseline{}, fmt.Errorf("read security surface %s: %w", path, err)
		}
		files[path] = externalDigest(data)
	}
	return externalIntegrityBaseline{Schema: externalIntegritySchema, Project: externalProjectID(root), RootPath: canonicalExternalRoot(root), Commit: commit, Files: files}, nil
}

func externalHeadFiles(root string) ([]string, string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "-z", "--name-only", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("list tracked HEAD files: %w", err)
	}
	commitCmd := exec.Command("git", "rev-parse", "HEAD")
	commitCmd.Dir = root
	commitOut, err := commitCmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("read project HEAD: %w", err)
	}
	var paths []string
	for _, rawPath := range bytes.Split(out, []byte{0}) {
		if len(rawPath) != 0 {
			paths = append(paths, filepath.ToSlash(string(rawPath)))
		}
	}
	sort.Strings(paths)
	return paths, strings.TrimSpace(string(commitOut)), nil
}

func isExternalSecuritySurface(path string) bool {
	path = filepath.ToSlash(path)
	base := filepath.Base(path)
	if strings.HasPrefix(path, ".github/workflows/") || strings.HasPrefix(path, ".gitlab/") {
		return true
	}
	if strings.HasPrefix(base, "Dockerfile") || base == "Makefile" || base == ".gitignore" || base == ".gitleaks.toml" {
		return true
	}
	for _, name := range []string{
		"package.json", "pnpm-lock.yaml", "package-lock.json", "yarn.lock", "bun.lockb",
		"go.mod", "go.sum", "requirements.txt", "requirements-dev.txt", "pyproject.toml",
		"poetry.lock", "Pipfile.lock", "Cargo.toml", "Cargo.lock", "composer.json", "composer.lock",
	} {
		if base == name {
			return true
		}
	}
	return false
}

func externalProjectID(root string) string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out))
	}
	return filepath.Base(canonicalExternalRoot(root))
}

func canonicalExternalRoot(root string) string {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		return filepath.Clean(resolved)
	}
	abs, _ := filepath.Abs(root)
	return filepath.Clean(abs)
}

func externalDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func gitBlob(root, path string) ([]byte, error) {
	cmd := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(path))
	cmd.Dir = root
	return cmd.Output()
}

// PlanExternalBaselines is intentionally read-only and is used by the
// explicit consumer baseline command. It returns central-only artifacts.
func PlanExternalBaselines(root string) (sbom []byte, integrity []byte, err error) {
	profile, err := externalProfile(root)
	if err != nil {
		return nil, nil, err
	}
	if !profile.External {
		return nil, nil, fmt.Errorf("external baselines cannot be created for the OVAV monorepo")
	}
	s, err := planExternalSBOM(root)
	if err != nil {
		return nil, nil, err
	}
	i, err := planExternalIntegrity(root)
	if err != nil {
		return nil, nil, err
	}
	sbom, err = json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	integrity, err = json.MarshalIndent(i, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return append(sbom, '\n'), append(integrity, '\n'), nil
}

// WriteExternalBaselines writes only to OVAV's central consumer state.
// The caller must make the write explicit; validation never calls this.
func WriteExternalBaselines(root string) error {
	sbom, integrity, err := PlanExternalBaselines(root)
	if err != nil {
		return err
	}
	dir := ""
	if profile, resolveErr := externalProfile(root); resolveErr == nil {
		dir = profile.StateDir()
	}
	if dir == "" {
		return fmt.Errorf("central external baseline directory unavailable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, externalSBOMFile), sbom, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, externalIntegrityFile), integrity, 0o644)
}

var _ Validator = (*ExternalRegistration)(nil)
