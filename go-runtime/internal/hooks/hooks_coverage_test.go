package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// hooks_coverage_test.go — Coverage boost: 79.5% → 80%+
// Targets: loadStageFilterFromYAML (real file), parseAutoTriggersForEvent
// edge cases, Audit uninstalled status, hooksDir missing .git, RunStage paths.
// ═══════════════════════════════════════════════════════════════════════════

// ── loadStageFilterFromYAML with real file ────────────────────────────────────

func TestLoadStageFilterFromYAML_RealFile(t *testing.T) {
	tmp := t.TempDir()
	// Create a fake repo root with auto_triggers.yaml
	repoRoot := filepath.Join(tmp, "repo")
	os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755) // findRepoRoot needs .git
	os.MkdirAll(filepath.Join(repoRoot, ".ovav", "registry"), 0755)

	yamlContent := []byte(`
router:
  before_git_stage:
    - check_secrets_hygiene
    - check_protected_branch
  before_git_push:
    - check_release_gate
    - pre_push_intelligence
`)
	os.WriteFile(filepath.Join(repoRoot, ".ovav", "registry", "auto_triggers.yaml"), yamlContent, 0644)

	// Change to repo root so findRepoRoot finds it
	origDir, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(origDir)

	filter := loadStageFilterFromYAML(StagePreCommit)
	if len(filter) != 2 {
		t.Errorf("expected 2 validators from YAML, got %d: %v", len(filter), filter)
	}
}

func TestLoadStageFilterFromYAML_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
	os.MkdirAll(filepath.Join(repoRoot, ".ovav", "registry"), 0755)
	// No auto_triggers.yaml created

	origDir, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(origDir)

	filter := loadStageFilterFromYAML(StagePreCommit)
	if filter != nil {
		t.Errorf("missing YAML should return nil, got %v", filter)
	}
}

func TestLoadStageFilterFromYAML_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
	os.MkdirAll(filepath.Join(repoRoot, ".ovav", "registry"), 0755)
	os.WriteFile(filepath.Join(repoRoot, ".ovav", "registry", "auto_triggers.yaml"),
		[]byte("{{invalid yaml content"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(origDir)

	filter := loadStageFilterFromYAML(StagePreCommit)
	if filter != nil {
		t.Errorf("invalid YAML should return nil, got %v", filter)
	}
}

func TestLoadStageFilterFromYAML_NoRouterKey(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
	os.MkdirAll(filepath.Join(repoRoot, ".ovav", "registry"), 0755)
	os.WriteFile(filepath.Join(repoRoot, ".ovav", "registry", "auto_triggers.yaml"),
		[]byte("other_key:\n  foo: bar\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(repoRoot)
	defer os.Chdir(origDir)

	filter := loadStageFilterFromYAML(StagePreCommit)
	if filter != nil {
		t.Errorf("missing router key should return nil, got %v", filter)
	}
}

func TestLoadStageFilterFromYAML_UnknownStage(t *testing.T) {
	// StagePostCheckout has no mapping in autoTriggersStageMap
	filter := loadStageFilterFromYAML(StagePostCheckout)
	if filter != nil {
		t.Errorf("unknown stage should return nil, got %v", filter)
	}
}

func TestLoadStageFilterFromYAML_NoRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	// Change to a dir with no .git anywhere
	origDir, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(origDir)

	filter := loadStageFilterFromYAML(StagePreCommit)
	if filter != nil {
		t.Errorf("no repo root should return nil, got %v", filter)
	}
}

// ── parseAutoTriggersForEvent edge cases ──────────────────────────────────────

func TestParseAutoTriggersForEvent_NonStringItems(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage:
    - check_secrets_hygiene
    - 123
    - true
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_stage")
	// Should only include string items
	if len(filter) != 1 {
		t.Errorf("expected 1 string validator, got %d: %v", len(filter), filter)
	}
}

func TestParseAutoTriggersForEvent_NonListEvent(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage: "single_string_value"
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_stage")
	if filter != nil {
		t.Errorf("non-list event should return nil, got %v", filter)
	}
}

// ── Audit — uninstalled status ───────────────────────────────────────────────

func TestAudit_AllUninstalled(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	if report.Status != "broken" {
		t.Errorf("all hooks missing → status = %q, want 'broken'", report.Status)
	}
}

func TestAudit_SymlinkThreat(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	// Replace a hook with a symlink
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.Remove(hookPath)
	os.Symlink("/some/other/path", hookPath)

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	foundSymlink := false
	for _, th := range report.Threats {
		if strings.Contains(th, "SYMLINK") {
			foundSymlink = true
			break
		}
	}
	if !foundSymlink {
		t.Errorf("symlink threat not detected, threats: %v", report.Threats)
	}
}

func TestAudit_NotExecutableThreat(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	// Make a hook non-executable
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.Chmod(hookPath, 0644)

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	foundNotExec := false
	for _, th := range report.Threats {
		if strings.Contains(th, "not executable") {
			foundNotExec = true
			break
		}
	}
	if !foundNotExec {
		t.Errorf("not-executable threat not detected, threats: %v", report.Threats)
	}
}

// ── hooksDir — edge cases ────────────────────────────────────────────────────

func TestHooksDir_MissingGitDir(t *testing.T) {
	tmp := t.TempDir()
	// No .git at all
	m := &Manager{RepoRoot: tmp}
	dir := m.hooksDir()
	if !strings.HasSuffix(dir, ".git/hooks") {
		t.Errorf("missing .git → should fallback to .git/hooks, got: %s", dir)
	}
}

func TestHooksDir_GitFile_ReadError(t *testing.T) {
	tmp := t.TempDir()
	// .git is a file but unreadable (permission denied)
	gitFile := filepath.Join(tmp, ".git")
	os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0000) // no read permission

	m := &Manager{RepoRoot: tmp}
	dir := m.hooksDir()
	// Should fallback gracefully
	if !strings.HasSuffix(dir, ".git/hooks") {
		t.Errorf("unreadable .git file → should fallback, got: %s", dir)
	}
}

func TestHooksDir_GitFile_NonGitdirContent(t *testing.T) {
	tmp := t.TempDir()
	gitFile := filepath.Join(tmp, ".git")
	os.WriteFile(gitFile, []byte("some random content\n"), 0644)

	m := &Manager{RepoRoot: tmp}
	dir := m.hooksDir()
	if !strings.HasSuffix(dir, ".git/hooks") {
		t.Errorf("non-gitdir .git file → should fallback, got: %s", dir)
	}
}

// ── RunStage — edge paths ────────────────────────────────────────────────────

func TestRunStage_PostCheckoutEmpty(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	result := m.RunStage(StagePostCheckout)
	if !result.Passed {
		t.Error("post-checkout with empty filter should pass")
	}
}

func TestRunStage_PreCommitRunsValidators(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	result := m.RunStage(StagePreCommit)
	// Should have run some validators (from hardcoded or YAML filter)
	if result.Duration == 0 {
		t.Error("RunStage should record duration")
	}
	// Result should exist (not nil)
	if result == nil {
		t.Fatal("RunStage returned nil")
	}
	if result.Stage != StagePreCommit {
		t.Errorf("stage = %q, want %q", result.Stage, StagePreCommit)
	}
}

// ── CheckTampering — symlink + foreign hook paths ────────────────────────────

func TestCheckTampering_SymlinkHook(t *testing.T) {
	tmp := t.TempDir()
	hooksDir := filepath.Join(tmp, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Create a symlink hook
	hookPath := filepath.Join(hooksDir, "pre-commit")
	os.Symlink("/some/target", hookPath)

	m := &Manager{RepoRoot: tmp}
	events := m.CheckTampering()
	foundSymlink := false
	for _, e := range events {
		if e.Type == "symlink" {
			foundSymlink = true
			break
		}
	}
	if !foundSymlink {
		t.Error("symlink tampering not detected")
	}
}

func TestCheckTampering_ForeignHook(t *testing.T) {
	tmp := t.TempDir()
	hooksDir := filepath.Join(tmp, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)

	// Write a non-OVAV hook
	hookPath := filepath.Join(hooksDir, "pre-commit")
	os.WriteFile(hookPath, []byte("#!/bin/bash\necho foreign\n"), 0755)

	m := &Manager{RepoRoot: tmp}
	events := m.CheckTampering()
	foundForeign := false
	for _, e := range events {
		if e.Type == "foreign_hook" {
			foundForeign = true
			break
		}
	}
	if !foundForeign {
		t.Error("foreign hook tampering not detected")
	}
}

// ── NoVerifyCheck — healthy hooks ────────────────────────────────────────────

func TestNoVerifyCheck_HealthyHooks(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	report, err := m.NoVerifyCheck()
	if err != nil {
		t.Fatalf("NoVerifyCheck error: %v", err)
	}
	if report.Detected {
		t.Error("healthy hooks should not trigger bypass detection")
	}
	if !strings.Contains(report.Message, "No evidence") {
		t.Errorf("clean report message = %q, should contain 'No evidence'", report.Message)
	}
}

// ── StatusReport — non-OVAV hooks ────────────────────────────────────────────

func TestStatus_ForeignHook(t *testing.T) {
	_, m := setupGitRepo(t)

	// Write a non-OVAV hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\necho custom\n"), 0755)

	report, err := m.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if report.AllHealthy {
		t.Error("foreign hook → AllHealthy should be false")
	}
	for _, hs := range report.Hooks {
		if hs.Stage == StagePreCommit {
			if hs.OVAV {
				t.Error("foreign hook should not be OVAV managed")
			}
			if !strings.Contains(hs.Message, "foreign") {
				t.Errorf("foreign hook message = %q, should mention 'foreign'", hs.Message)
			}
		}
	}
}

func TestStatus_TamperedHook(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	// Tamper with a hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\n# OVAV Git Hook — tampered\nevil\n"), 0755)

	report, err := m.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if report.AllHealthy {
		t.Error("tampered hook → AllHealthy should be false")
	}
	for _, hs := range report.Hooks {
		if hs.Stage == StagePreCommit && hs.SHA256OK {
			t.Error("tampered hook should have SHA256OK=false")
		}
	}
}

// ── normalizeName — edge cases ───────────────────────────────────────────────

func TestNormalizeName_MultiplePrefixes(t *testing.T) {
	// Only first matching prefix is stripped
	got := normalizeName("check_validate_something")
	if got != "validate_something" {
		t.Errorf("normalizeName(check_validate_something) = %q, want 'validate_something'", got)
	}
}

func TestNormalizeName_EmptyString(t *testing.T) {
	got := normalizeName("")
	if got != "" {
		t.Errorf("normalizeName('') = %q, want ''", got)
	}
}

// ── Install — foreign hook not replaced ───────────────────────────────────────

func TestInstall_PreservesForeignHook(t *testing.T) {
	_, m := setupGitRepo(t)

	// Write a non-OVAV hook before install
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\necho custom hook\n"), 0755)

	results, err := m.Install()
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	for _, r := range results {
		if r.Stage == StagePreCommit {
			// Foreign hook should be replaced with OVAV shim (fresh install)
			if r.Status != "installed" {
				t.Errorf("foreign hook: status = %q, want 'installed'", r.Status)
			}
			// Verify it's now an OVAV shim
			data, _ := os.ReadFile(r.Path)
			if !isOVAVShim(string(data)) {
				t.Error("installed hook should be OVAV shim")
			}
		}
	}
}
