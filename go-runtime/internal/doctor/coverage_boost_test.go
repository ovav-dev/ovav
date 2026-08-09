package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ovav/ovav/internal/cli"
)

func runInDir(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Output()
}

// ── FormatResults edge cases ──────────────────────────────────────────────

func TestFormatResultsAllStatuses(t *testing.T) {
	results := []CheckResult{
		{Name: "a-pass", Status: "pass", Detail: "ok"},
		{Name: "b-fail", Status: "fail", Detail: "broken", Fix: "fix it"},
		{Name: "c-warn", Status: "warn", Detail: "careful", Fix: "maybe fix"},
		{Name: "d-skip", Status: "skip", Detail: "n/a"},
	}
	out := FormatResults(results)
	if !strings.Contains(out, "✅") {
		t.Error("missing pass icon")
	}
	if !strings.Contains(out, "❌") {
		t.Error("missing fail icon")
	}
	if !strings.Contains(out, "⚠️") {
		t.Error("missing warn icon")
	}
	if !strings.Contains(out, "⬜") {
		t.Error("missing skip icon")
	}
	if !strings.Contains(out, "fix it") {
		t.Error("missing fix line for fail")
	}
	if !strings.Contains(out, "🔴") {
		t.Error("missing failure summary line")
	}
}

func TestFormatResultsWarnOnly(t *testing.T) {
	results := []CheckResult{
		{Name: "w1", Status: "warn", Detail: "something"},
	}
	out := FormatResults(results)
	if !strings.Contains(out, "🟡") {
		t.Error("warn-only should show yellow summary")
	}
}

func TestFormatResultsPassOnly(t *testing.T) {
	results := []CheckResult{
		{Name: "p1", Status: "pass", Detail: "fine"},
	}
	out := FormatResults(results)
	if !strings.Contains(out, "🟢") {
		t.Error("pass-only should show green summary")
	}
}

func TestFormatResultsCounts(t *testing.T) {
	results := []CheckResult{
		{Name: "p1", Status: "pass", Detail: "a"},
		{Name: "p2", Status: "pass", Detail: "b"},
		{Name: "w1", Status: "warn", Detail: "c"},
		{Name: "f1", Status: "fail", Detail: "d"},
	}
	out := FormatResults(results)
	if !strings.Contains(out, "2 passed") {
		t.Errorf("expected '2 passed' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 warnings") {
		t.Errorf("expected '1 warnings' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 failures") {
		t.Errorf("expected '1 failures' in output, got:\n%s", out)
	}
}

// ── checkGitAvailable: fail path ─────────────────────────────────────────

func TestCheckGitAvailableFail(t *testing.T) {
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	r := checkGitAvailable()
	if r.Status != "fail" {
		t.Errorf("expected fail when PATH empty, got %s", r.Status)
	}
	if r.Fix == "" {
		t.Error("expected fix suggestion on fail")
	}
	os.Setenv("PATH", origPath)
}

// ── checkGitRepo: warn path (not in repo) ────────────────────────────────

func TestCheckGitRepoNotInRepo(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkGitRepo()
	if r.Status != "warn" {
		t.Errorf("expected warn outside git repo, got %s", r.Status)
	}
}

// ── checkGitClean: error path (not in repo) + clean + dirty ──────────────

func TestCheckGitCleanOutsideRepo(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkGitClean()
	if r.Status != "warn" {
		t.Errorf("expected warn outside git repo, got %s", r.Status)
	}
}

func TestCheckGitCleanDirty(t *testing.T) {
	tmpDir := t.TempDir()
	runInDir(tmpDir, "git", "init")
	runInDir(tmpDir, "git", "config", "user.email", "test@test.com")
	runInDir(tmpDir, "git", "config", "user.name", "test")
	runInDir(tmpDir, "git", "commit", "--allow-empty", "-m", "init")
	os.WriteFile(filepath.Join(tmpDir, "extra.txt"), []byte("new"), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkGitClean()
	if r.Status != "warn" {
		t.Errorf("expected warn for dirty repo, got %s: %s", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "file") {
		t.Errorf("expected file count in detail, got %s", r.Detail)
	}
}

func TestCheckGitCleanClean(t *testing.T) {
	tmpDir := t.TempDir()
	runInDir(tmpDir, "git", "init")
	runInDir(tmpDir, "git", "config", "user.email", "test@test.com")
	runInDir(tmpDir, "git", "config", "user.name", "test")
	runInDir(tmpDir, "git", "commit", "--allow-empty", "-m", "init")

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkGitClean()
	if r.Status != "pass" {
		t.Errorf("expected pass for clean repo, got %s: %s", r.Status, r.Detail)
	}
}

// ── checkGitRemote: no-remote path ───────────────────────────────────────

func TestCheckGitRemoteNoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	runInDir(tmpDir, "git", "init")

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkGitRemote()
	if !strings.Contains(r.Detail, "No git remote") {
		t.Errorf("expected no-remote detail, got %s", r.Detail)
	}
	if r.Fix == "" {
		t.Error("expected fix suggestion")
	}
}

// ── checkBranchSafety: non-protected + unknown branch ────────────────────

func TestCheckBranchSafetyNonProtected(t *testing.T) {
	branch, _, _ := cli.GitInfo()
	protected := map[string]bool{
		"develop": true, "main": true, "master": true,
		"production": true, "prod": true, "staging": true,
		"unknown": true, // detached HEAD (tag checkout in CI)
	}
	if protected[branch] {
		t.Skipf("current branch %q is protected, cannot test non-protected path", branch)
	}
	r := checkBranchSafety()
	if r.Status != "pass" {
		t.Errorf("expected pass on non-protected branch, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckBranchSafetyUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	runInDir(tmpDir, "git", "init")
	runInDir(tmpDir, "git", "config", "user.email", "test@test.com")
	runInDir(tmpDir, "git", "config", "user.name", "test")
	runInDir(tmpDir, "git", "commit", "--allow-empty", "-m", "init")
	runInDir(tmpDir, "git", "checkout", "--detach", "HEAD")

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkBranchSafety()
	if r.Status != "warn" {
		t.Errorf("expected warn for unknown branch, got %s: %s", r.Status, r.Detail)
	}
}

// ── checkOVAVRoot: fail + warn paths ─────────────────────────────────────

func TestCheckOVAVRootFail(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkOVAVRoot()
	if r.Status != "fail" {
		t.Errorf("expected fail outside OVAV repo, got %s", r.Status)
	}
	if r.Fix == "" {
		t.Error("expected fix suggestion on fail")
	}
}

func TestCheckOVAVRootMissingDirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".ovav", "registry"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".ovav", "policy"), 0o755)
	// Missing: tools, go-runtime

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkOVAVRoot()
	if r.Status != "warn" {
		t.Errorf("expected warn with missing dirs, got %s: %s", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "Missing") {
		t.Errorf("expected 'Missing' in detail, got %s", r.Detail)
	}
}

// ── checkAuthorityContract: warn path (file exists) ─────────────────────

func TestCheckAuthorityContractExists(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	acDir := filepath.Join(tmpDir, ".ovav", "service_areas", "shared")
	os.MkdirAll(acDir, 0o755)
	os.WriteFile(filepath.Join(acDir, "current_authority_contract.yaml"), []byte("stale"), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkAuthorityContract()
	if r.Status != "warn" {
		t.Errorf("expected warn when authority-contract exists, got %s", r.Status)
	}
	if !strings.Contains(r.Detail, "caps.yaml") {
		t.Error("expected caps.yaml mention in detail")
	}
}

// ── checkPermissionAuthority: fail path ─────────────────────────────────

func TestCheckPermissionAuthorityMissing(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".ovav", "policy"), 0o755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkPermissionAuthority()
	if r.Status != "fail" {
		t.Errorf("expected fail when permission_authority.json missing, got %s", r.Status)
	}
}

// ── checkRegistryIntegrity: warn path ────────────────────────────────────

func TestCheckRegistryIntegrityMissing(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".ovav", "registry"), 0o755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkRegistryIntegrity()
	if r.Status != "warn" {
		t.Errorf("expected warn when auto_triggers.yaml missing, got %s", r.Status)
	}
}

// ── checkWaiverStatus: all branches ──────────────────────────────────────

func TestCheckWaiverStatusNoWaiver(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(tmpDir, ".ovav", "runtime"), 0o755)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "pass" {
		t.Errorf("expected pass with no waiver, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckWaiverStatusMalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	runtimeDir := filepath.Join(tmpDir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0o755)
	os.WriteFile(filepath.Join(runtimeDir, "protected_branch_waiver.yaml"), []byte("{{bad yaml"), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "warn" {
		t.Errorf("expected warn for malformed YAML, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckWaiverStatusMalformedStructure(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	runtimeDir := filepath.Join(tmpDir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0o755)
	os.WriteFile(filepath.Join(runtimeDir, "protected_branch_waiver.yaml"),
		[]byte("something: else\n"), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "warn" {
		t.Errorf("expected warn for malformed structure, got %s: %s", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "malformed") {
		t.Errorf("expected 'malformed' in detail, got %s", r.Detail)
	}
}

func TestCheckWaiverStatusActive(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	runtimeDir := filepath.Join(tmpDir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0o755)
	yaml := "waiver:\n  active: true\n  branch: develop\n  reason: testing\n"
	os.WriteFile(filepath.Join(runtimeDir, "protected_branch_waiver.yaml"), []byte(yaml), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "warn" {
		t.Errorf("expected warn for active waiver, got %s", r.Status)
	}
	if !strings.Contains(r.Detail, "WAIVER ACTIVE") {
		t.Errorf("expected 'WAIVER ACTIVE' in detail, got %s", r.Detail)
	}
}

func TestCheckWaiverStatusInactive(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	runtimeDir := filepath.Join(tmpDir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0o755)
	yaml := "waiver:\n  active: false\n  branch: develop\n  reason: done\n"
	os.WriteFile(filepath.Join(runtimeDir, "protected_branch_waiver.yaml"), []byte(yaml), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "pass" {
		t.Errorf("expected pass for inactive waiver, got %s: %s", r.Status, r.Detail)
	}
}

func TestCheckWaiverStatusNonBoolActive(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755)
	runtimeDir := filepath.Join(tmpDir, ".ovav", "runtime")
	os.MkdirAll(runtimeDir, 0o755)
	yaml := "waiver:\n  active: \"yes\"\n  branch: develop\n  reason: test\n"
	os.WriteFile(filepath.Join(runtimeDir, "protected_branch_waiver.yaml"), []byte(yaml), 0o644)

	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(tmpDir)
	r := checkWaiverStatus()
	if r.Status != "pass" {
		t.Errorf("expected pass for non-bool active, got %s: %s", r.Status, r.Detail)
	}
}

// ── checkBranchSafety: waiver present on protected branch ────────────────

func TestCheckBranchSafetyWithWaiver(t *testing.T) {
	branch, _, _ := cli.GitInfo()
	protected := map[string]bool{
		"develop": true, "main": true, "master": true,
		"production": true, "prod": true, "staging": true,
	}
	if !protected[branch] {
		t.Skipf("not on protected branch (%s), skipping waiver test", branch)
	}
	repoRoot, err := cli.FindRepoRoot()
	if err != nil {
		t.Skip("cannot find repo root")
	}
	waiverPath := filepath.Join(repoRoot, ".ovav", "runtime", "protected_branch_waiver.yaml")
	if _, err := os.Stat(waiverPath); err == nil {
		t.Skip("waiver already exists, cannot test creation path")
	}
	os.WriteFile(waiverPath, []byte("waiver:\n  active: true\n  branch: test\n  reason: test\n"), 0o644)
	defer os.Remove(waiverPath)

	r := checkBranchSafety()
	if !strings.Contains(r.Detail, "WAIVER ACTIVE") {
		t.Errorf("expected WAIVER ACTIVE detail, got %s", r.Detail)
	}
}
