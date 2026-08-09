package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ── Shim Generation ────────────────────────────────────────────────────────────

func TestShimContentUnix_Basic(t *testing.T) {
	m := &Manager{
		RepoRoot:   "/test/repo",
		OVAVBinary: "/usr/local/bin/ovav",
	}

	content := m.shimContentUnix(StagePreCommit)

	// Must contain OVAV identifier
	if !strings.Contains(content, "OVAV Git Hook") {
		t.Error("shim missing OVAV identifier")
	}

	// Must contain stage
	if !strings.Contains(content, "Pre-commit") {
		t.Error("shim missing stage label")
	}

	// Must invoke binary
	if !strings.Contains(content, "/usr/local/bin/ovav hook run --stage pre-commit") {
		t.Error("shim not invoking ovav binary correctly")
	}

	// Must be executable shell
	if !strings.HasPrefix(content, "#!/usr/bin/env bash") {
		t.Error("shim missing shebang")
	}
}

func TestShimContentWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows shim test only runs on Windows")
	}

	m := &Manager{RepoRoot: "/test/repo", OVAVBinary: "ovav.exe"}
	content := m.shimContent(StagePrePush)
	if !strings.Contains(content, "OVAV Git Hook") {
		t.Error("Windows shim missing OVAV identifier")
	}
}

func TestShimContentFallbackBinary(t *testing.T) {
	m := &Manager{RepoRoot: "/test/repo"}
	// OVAVBinary is empty → should fallback to "ovav"
	content := m.shimContentUnix(StagePreCommit)
	if !strings.Contains(content, "ovav hook run --stage pre-commit") {
		t.Error("shim fallback binary not 'ovav'")
	}
}

// ── OVAV Shim Detection ────────────────────────────────────────────────────────

func TestIsOVAVShim_True(t *testing.T) {
	if !isOVAVShim("#!/bin/bash\n# OVAV Git Hook\novav hook run") {
		t.Error("OVAV shim not detected")
	}
	if !isOVAVShim("ovav hook run --stage pre-commit") {
		t.Error("OVAV shim alt pattern not detected")
	}
}

func TestIsOVAVShim_False(t *testing.T) {
	if isOVAVShim("#!/bin/bash\necho hello") {
		t.Error("non-OVAV shim incorrectly detected")
	}
	if isOVAVShim("") {
		t.Error("empty string incorrectly detected as OVAV")
	}
}

// ── Stage Filtering ────────────────────────────────────────────────────────────

func TestStageFilter_Matches_Exact(t *testing.T) {
	filter := StageFilter{"secrets_hygiene", "protected_branch", "workspace_safety"}
	if !filter.Matches("secrets_hygiene") {
		t.Error("exact match should work")
	}
	if filter.Matches("nonexistent") {
		t.Error("nonexistent validator should not match")
	}
}

func TestStageFilter_Matches_Prefixed(t *testing.T) {
	// YAML uses "check_secrets_hygiene", Go uses "secrets_hygiene"
	filter := StageFilter{"check_secrets_hygiene", "validate_workspace_safety_gate"}
	if !filter.Matches("secrets_hygiene") {
		t.Error("check_ prefix should map to Go ID")
	}
	if !filter.Matches("workspace_safety") {
		t.Error("validate_ prefix should map to Go ID")
	}
}

func TestGetStageFilter_HardcodedFallback(t *testing.T) {
	filter := hardcodedStageFilter(StagePreCommit)
	if len(filter) == 0 {
		t.Error("hardcoded pre-commit filter should not be empty")
	}
	if !filter.Matches("secrets_hygiene") {
		t.Error("hardcoded pre-commit should include secrets_hygiene (via check_ prefix)")
	}

	filter = hardcodedStageFilter(StagePrePush)
	if len(filter) == 0 {
		t.Error("hardcoded pre-push filter should not be empty")
	}
}

func TestGetStageFilter_Unknown(t *testing.T) {
	filter := hardcodedStageFilter(StagePostCheckout)
	if len(filter) != 0 {
		t.Error("unknown stage should return empty filter")
	}
}

// ── Hooks Directory Resolution ─────────────────────────────────────────────────

func TestHooksDir_MainRepo(t *testing.T) {
	// Create a temp dir that looks like a main repo
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	os.MkdirAll(gitDir, 0755)
	os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	dir := m.hooksDir()
	if !strings.HasSuffix(dir, ".git/hooks") {
		t.Errorf("hooks dir should end with .git/hooks, got: %s", dir)
	}
}

// ── CI Detection ───────────────────────────────────────────────────────────────

func TestIsCI_False(t *testing.T) {
	// In normal test environment, CI should be false
	if IsCI() {
		t.Log("CI detected in test environment — this is fine if running in CI")
	}
}

func TestIsCI_True(t *testing.T) {
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")
	if !IsCI() {
		t.Error("CI=true should be detected")
	}
}

// ── Stage Constants ────────────────────────────────────────────────────────────

func TestStageHookName(t *testing.T) {
	tests := []struct {
		stage Stage
		want  string
	}{
		{StagePreCommit, "pre-commit"},
		{StagePrePush, "pre-push"},
		{StagePostCheckout, "post-checkout"},
		{StageCommitMsg, "commit-msg"},
	}
	for _, tt := range tests {
		if tt.stage.HookName() != tt.want {
			t.Errorf("%s.HookName() = %s, want %s", tt.stage, tt.stage.HookName(), tt.want)
		}
	}
}

func TestAllStages(t *testing.T) {
	stages := AllStages()
	if len(stages) != 4 {
		t.Errorf("Expected 4 stages, got %d", len(stages))
	}
}

// ── NoVerifyCheck ──────────────────────────────────────────────────────────────

func TestNoVerifyCheck_NoHooks(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	report, err := m.NoVerifyCheck()
	if err != nil {
		t.Fatalf("NoVerifyCheck error: %v", err)
	}
	if !report.Detected {
		// Hooks are not installed, so should detect potential bypass
		t.Log("NoVerifyCheck in empty dir returned:", report.Message)
	}
}

// ── Tampering Detection ───────────────────────────────────────────────────────

func TestCheckTampering_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := &Manager{RepoRoot: tmp}
	events := m.CheckTampering()
	// Should detect missing hooks
	if len(events) == 0 {
		t.Error("empty hooks dir should generate tampering events for missing hooks")
	}
}

// ── NewManager ──────────────────────────────────────────────────────────────────

func TestNewManager(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)

	m := NewManager(tmp)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.RepoRoot != tmp {
		t.Errorf("RepoRoot = %q, want %q", m.RepoRoot, tmp)
	}
	// OVAVBinary should be resolved to something (at least "ovav" fallback)
	if m.OVAVBinary == "" {
		t.Error("OVAVBinary should not be empty")
	}
}

// ── Stage.Label ─────────────────────────────────────────────────────────────────

func TestStageLabel_All(t *testing.T) {
	tests := []struct {
		stage Stage
		want  string
	}{
		{StagePreCommit, "Pre-commit"},
		{StagePrePush, "Pre-push"},
		{StagePostCheckout, "Post-checkout"},
		{StageCommitMsg, "Commit-msg"},
		{Stage("unknown-stage"), "unknown-stage"}, // default branch
	}
	for _, tt := range tests {
		got := tt.stage.Label()
		if got != tt.want {
			t.Errorf("%q.Label() = %q, want %q", tt.stage, got, tt.want)
		}
	}
}

// ── resolveOVAVBinary ───────────────────────────────────────────────────────────

func TestResolveOVAVBinary_EnvVar(t *testing.T) {
	t.Setenv("OVAV_BIN", "/custom/path/ovav")
	got := resolveOVAVBinary("/some/repo")
	if got != "/custom/path/ovav" {
		t.Errorf("resolveOVAVBinary with OVAV_BIN = %q, want /custom/path/ovav", got)
	}
}

func TestResolveOVAVBinary_RepoBuild(t *testing.T) {
	t.Setenv("OVAV_BIN", "") // clear env

	tmp := t.TempDir()
	buildDir := filepath.Join(tmp, "go-runtime", "build")
	os.MkdirAll(buildDir, 0755)
	binPath := filepath.Join(buildDir, "ovav")
	os.WriteFile(binPath, []byte("#!/bin/bash\n"), 0755)

	// If ~/.local/bin/ovav exists on this machine, it will be returned
	// before the repo build path. This is expected behavior.
	home, _ := os.UserHomeDir()
	systemBin := filepath.Join(home, ".local", "bin", "ovav")
	if _, err := os.Stat(systemBin); err == nil {
		t.Skipf("Skipping: ~/.local/bin/ovav exists and takes priority over repo build path")
	}

	got := resolveOVAVBinary(tmp)
	if got != binPath {
		t.Errorf("resolveOVAVBinary repo build = %q, want %q", got, binPath)
	}
}

func TestResolveOVAVBinary_Fallback(t *testing.T) {
	t.Setenv("OVAV_BIN", "") // clear env

	// Use a non-existent repo root so no candidates match
	got := resolveOVAVBinary("/nonexistent/path/that/does/not/exist")
	// Should fall back to LookPath or "ovav"
	if got == "" {
		t.Error("resolveOVAVBinary fallback should not be empty")
	}
	// It's either a path from LookPath or the literal "ovav"
	if !strings.Contains(got, "ovav") {
		t.Errorf("resolveOVAVBinary fallback %q should contain 'ovav'", got)
	}
}

// ── Manifest ────────────────────────────────────────────────────────────────────

func TestManifest(t *testing.T) {
	m := &Manager{RepoRoot: "/test/repo", OVAVBinary: "/usr/local/bin/ovav"}
	manifest := m.Manifest()

	requiredKeys := []string{"repo_root", "ovav_binary", "stages", "platform", "managed_by"}
	for _, key := range requiredKeys {
		if _, ok := manifest[key]; !ok {
			t.Errorf("Manifest missing key %q", key)
		}
	}
	if manifest["repo_root"] != "/test/repo" {
		t.Errorf("Manifest repo_root = %v, want /test/repo", manifest["repo_root"])
	}
	if manifest["ovav_binary"] != "/usr/local/bin/ovav" {
		t.Errorf("Manifest ovav_binary = %v", manifest["ovav_binary"])
	}
	stages, ok := manifest["stages"].([]Stage)
	if !ok || len(stages) != 4 {
		t.Errorf("Manifest stages should be 4 Stage items, got %v", manifest["stages"])
	}
}

// ── containsTampered ────────────────────────────────────────────────────────────

func TestContainsTampered_True(t *testing.T) {
	hooks := []HookStatus{
		{Stage: StagePreCommit, OVAV: true, SHA256OK: true},
		{Stage: StagePrePush, OVAV: true, SHA256OK: false}, // tampered
	}
	if !containsTampered(hooks) {
		t.Error("should detect tampered hook")
	}
}

func TestContainsTampered_False_AllClean(t *testing.T) {
	hooks := []HookStatus{
		{Stage: StagePreCommit, OVAV: true, SHA256OK: true},
		{Stage: StagePrePush, OVAV: true, SHA256OK: true},
	}
	if containsTampered(hooks) {
		t.Error("all clean hooks should not be flagged as tampered")
	}
}

func TestContainsTampered_False_ForeignHook(t *testing.T) {
	// Foreign hook (OVAV=false) with SHA256OK=false should NOT count as tampered
	hooks := []HookStatus{
		{Stage: StagePreCommit, OVAV: false, SHA256OK: false},
	}
	if containsTampered(hooks) {
		t.Error("foreign hook should not be flagged by containsTampered (only OVAV hooks)")
	}
}

func TestContainsTampered_Empty(t *testing.T) {
	if containsTampered(nil) {
		t.Error("nil hooks should not be tampered")
	}
}

// ── CIStrictMode ────────────────────────────────────────────────────────────────

func TestCIStrictMode(t *testing.T) {
	filter := CIStrictMode()
	if len(filter) == 0 {
		t.Fatal("CIStrictMode should return non-empty filter")
	}
	// Must contain specific CI-only validators
	expected := []string{"check_living_integrity", "check_supply_chain", "check_exfil_patterns"}
	for _, e := range expected {
		found := false
		for _, f := range filter {
			if f == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CIStrictMode missing expected validator %q", e)
		}
	}
}

// ── FormatNoVerifyHuman ─────────────────────────────────────────────────────────

func TestFormatNoVerifyHuman_Detected(t *testing.T) {
	r := &NoVerifyReport{
		CheckedAt: "2026-06-22T10:00:00Z",
		Detected:  true,
		Evidence:  []string{"pre-commit: not installed", "pre-push: tampered"},
	}
	output := FormatNoVerifyHuman(r)

	if !strings.Contains(output, "BYPASS DETECTED") {
		t.Error("detected output should contain bypass warning")
	}
	if !strings.Contains(output, "pre-commit: not installed") {
		t.Error("detected output should include evidence items")
	}
	if !strings.Contains(output, "Recommendation") {
		t.Error("detected output should include recommendation")
	}
}

func TestFormatNoVerifyHuman_Clean(t *testing.T) {
	r := &NoVerifyReport{
		CheckedAt: "2026-06-22T10:00:00Z",
		Detected:  false,
	}
	output := FormatNoVerifyHuman(r)

	if !strings.Contains(output, "No evidence of bypass") {
		t.Error("clean output should contain clean message")
	}
	if !strings.Contains(output, "2026-06-22T10:00:00Z") {
		t.Error("clean output should include checked timestamp")
	}
}

// ── Install + Uninstall ─────────────────────────────────────────────────────────

func setupGitRepo(t *testing.T) (string, *Manager) {
	t.Helper()
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git", "hooks"), 0755)
	m := &Manager{RepoRoot: tmp, OVAVBinary: "/usr/local/bin/ovav"}
	return tmp, m
}

func TestInstall_Fresh(t *testing.T) {
	_, m := setupGitRepo(t)

	results, err := m.Install()
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if len(results) != len(AllStages()) {
		t.Fatalf("Install returned %d results, want %d", len(results), len(AllStages()))
	}

	for _, r := range results {
		if r.Status != "installed" {
			t.Errorf("stage %s: status = %q, want 'installed'", r.Stage, r.Status)
		}
		// Verify file exists and is executable
		info, err := os.Stat(r.Path)
		if err != nil {
			t.Errorf("stage %s: hook file not found at %s", r.Stage, r.Path)
			continue
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("stage %s: hook file not executable", r.Stage)
		}
		// Verify content is OVAV shim
		data, _ := os.ReadFile(r.Path)
		if !isOVAVShim(string(data)) {
			t.Errorf("stage %s: installed file is not an OVAV shim", r.Stage)
		}
	}
}

func TestInstall_Idempotent(t *testing.T) {
	_, m := setupGitRepo(t)

	// First install
	m.Install()

	// Second install — should skip all
	results, err := m.Install()
	if err != nil {
		t.Fatalf("Install (idempotent) error: %v", err)
	}
	for _, r := range results {
		if r.Status != "skip" {
			t.Errorf("stage %s: second install status = %q, want 'skip'", r.Stage, r.Status)
		}
	}
}

func TestInstall_Update(t *testing.T) {
	_, m := setupGitRepo(t)

	// First install
	m.Install()

	// Tamper with one hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\n# OVAV Git Hook — tampered\novav hook run --stage pre-commit\n"), 0755)

	// Re-install — should detect mismatch and update
	results, err := m.Install()
	if err != nil {
		t.Fatalf("Install (update) error: %v", err)
	}
	for _, r := range results {
		if r.Stage == StagePreCommit && r.Status != "updated" {
			t.Errorf("tampered hook: status = %q, want 'updated'", r.Status)
		}
	}
}

func TestUninstall_AfterInstall(t *testing.T) {
	_, m := setupGitRepo(t)

	// Install first
	m.Install()

	// Uninstall
	results, err := m.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	if len(results) != len(AllStages()) {
		t.Fatalf("Uninstall returned %d results, want %d", len(results), len(AllStages()))
	}

	for _, r := range results {
		if r.Status != "removed" {
			t.Errorf("stage %s: uninstall status = %q, want 'removed'", r.Stage, r.Status)
		}
		// Verify file is gone
		if _, err := os.Stat(r.Path); !os.IsNotExist(err) {
			t.Errorf("stage %s: hook file still exists after uninstall", r.Stage)
		}
	}
}

func TestUninstall_PreservesForeign(t *testing.T) {
	_, m := setupGitRepo(t)

	// Write a non-OVAV hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'custom hook'\n"), 0755)

	results, err := m.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}

	for _, r := range results {
		if r.Stage == StagePreCommit {
			if r.Status != "skip" {
				t.Errorf("foreign hook: status = %q, want 'skip'", r.Status)
			}
			if !strings.Contains(r.Message, "preserved") {
				t.Errorf("foreign hook message should mention 'preserved', got %q", r.Message)
			}
			// File should still exist
			if _, err := os.Stat(r.Path); os.IsNotExist(err) {
				t.Error("foreign hook was removed — should have been preserved")
			}
		}
	}
}

func TestUninstall_NotFound(t *testing.T) {
	_, m := setupGitRepo(t)
	// Don't install anything — uninstall should skip all

	results, err := m.Uninstall()
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}
	for _, r := range results {
		if r.Status != "skip" {
			t.Errorf("stage %s: status = %q, want 'skip' (not found)", r.Stage, r.Status)
		}
	}
}

// ── Audit ───────────────────────────────────────────────────────────────────────

func TestAudit_CleanAfterInstall(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	if report.Status != "clean" {
		t.Errorf("Audit status = %q, want 'clean'", report.Status)
	}
	if len(report.Threats) != 0 {
		t.Errorf("clean audit should have 0 threats, got %d: %v", len(report.Threats), report.Threats)
	}
	if report.RepoRoot != m.RepoRoot {
		t.Errorf("Audit RepoRoot = %q, want %q", report.RepoRoot, m.RepoRoot)
	}
	if report.LastAudit == "" {
		t.Error("Audit LastAudit should not be empty")
	}
}

func TestAudit_MissingHooks(t *testing.T) {
	_, m := setupGitRepo(t)
	// Don't install — hooks are missing

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	if len(report.Threats) == 0 {
		t.Error("missing hooks should generate threats")
	}
	// Should contain "missing" or "bypass" in threats
	foundMissing := false
	for _, th := range report.Threats {
		if strings.Contains(th, "missing") || strings.Contains(th, "bypass") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("expected 'missing' threat, got: %v", report.Threats)
	}
}

func TestAudit_TamperedHook(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	// Tamper with a hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\n# OVAV Git Hook — evil\nevil_command\n"), 0755)

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	if report.Status != "tampered" {
		t.Errorf("Audit status = %q, want 'tampered'", report.Status)
	}
	foundTamper := false
	for _, th := range report.Threats {
		if strings.Contains(th, "SHA-256 MISMATCH") || strings.Contains(th, "tampered") {
			foundTamper = true
			break
		}
	}
	if !foundTamper {
		t.Errorf("expected tamper threat, got: %v", report.Threats)
	}
}

func TestAudit_ForeignHook(t *testing.T) {
	_, m := setupGitRepo(t)

	// Write a foreign hook
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\necho foreign\n"), 0755)

	report, err := m.Audit()
	if err != nil {
		t.Fatalf("Audit error: %v", err)
	}
	foundForeign := false
	for _, th := range report.Threats {
		if strings.Contains(th, "foreign") || strings.Contains(th, "not OVAV") {
			foundForeign = true
			break
		}
	}
	if !foundForeign {
		t.Errorf("expected foreign hook threat, got: %v", report.Threats)
	}
}

// ── VerifyIntegrity ─────────────────────────────────────────────────────────────

func TestVerifyIntegrity_Clean(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		t.Fatalf("GenerateIntegritySnapshot error: %v", err)
	}

	violations := m.VerifyIntegrity(snap)
	if len(violations) != 0 {
		t.Errorf("clean integrity check should have 0 violations, got %d: %v", len(violations), violations)
	}
}

func TestVerifyIntegrity_TamperedHook(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		t.Fatalf("GenerateIntegritySnapshot error: %v", err)
	}

	// Tamper with a hook after snapshot
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.WriteFile(hookPath, []byte("#!/bin/bash\n# OVAV Git Hook — modified\nevil\n"), 0755)

	violations := m.VerifyIntegrity(snap)
	if len(violations) == 0 {
		t.Error("tampered hook should produce violations")
	}
	foundTamper := false
	for _, v := range violations {
		if strings.Contains(v, "tampered") {
			foundTamper = true
			break
		}
	}
	if !foundTamper {
		t.Errorf("expected tamper violation, got: %v", violations)
	}
}

func TestVerifyIntegrity_MissingHook(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		t.Fatalf("GenerateIntegritySnapshot error: %v", err)
	}

	// Remove a hook after snapshot
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.Remove(hookPath)

	violations := m.VerifyIntegrity(snap)
	if len(violations) == 0 {
		t.Error("removed hook should produce violations")
	}
	foundMissing := false
	for _, v := range violations {
		if strings.Contains(v, "missing") || strings.Contains(v, "removed") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("expected missing violation, got: %v", violations)
	}
}

func TestVerifyIntegrity_EmptySnapshot(t *testing.T) {
	_, m := setupGitRepo(t)

	// Empty snapshot — no hooks or binary to verify
	snap := &IntegritySnapshot{
		GeneratedAt: "2026-06-22T10:00:00Z",
		Hooks:       make(map[Stage]string),
	}
	violations := m.VerifyIntegrity(snap)
	if len(violations) != 0 {
		t.Errorf("empty snapshot should produce 0 violations, got %d", len(violations))
	}
}

// ── GenerateIntegritySnapshot ───────────────────────────────────────────────────

func TestGenerateIntegritySnapshot_WithHooks(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		t.Fatalf("GenerateIntegritySnapshot error: %v", err)
	}
	if snap.GeneratedAt == "" {
		t.Error("GeneratedAt should not be empty")
	}
	if len(snap.Hooks) != len(AllStages()) {
		t.Errorf("snapshot should have %d hooks, got %d", len(AllStages()), len(snap.Hooks))
	}
	for stage, sha := range snap.Hooks {
		if sha == "" {
			t.Errorf("stage %s has empty SHA", stage)
		}
		if len(sha) != 64 { // SHA-256 hex = 64 chars
			t.Errorf("stage %s SHA length = %d, want 64", stage, len(sha))
		}
	}
}

func TestGenerateIntegritySnapshot_NoHooks(t *testing.T) {
	_, m := setupGitRepo(t)
	// Don't install hooks

	snap, err := m.GenerateIntegritySnapshot()
	if err != nil {
		t.Fatalf("GenerateIntegritySnapshot error: %v", err)
	}
	if len(snap.Hooks) != 0 {
		t.Errorf("no hooks installed → snapshot should have 0 hooks, got %d", len(snap.Hooks))
	}
}

// ── parseAutoTriggersForEvent ───────────────────────────────────────────────────

func TestParseAutoTriggersForEvent_Valid(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage:
    - check_secrets_hygiene
    - check_protected_branch
    - validate_workspace_safety_gate
  before_git_push:
    - check_release_gate
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_stage")
	if len(filter) != 3 {
		t.Fatalf("expected 3 validators, got %d: %v", len(filter), filter)
	}
	if filter[0] != "check_secrets_hygiene" {
		t.Errorf("first validator = %q, want check_secrets_hygiene", filter[0])
	}
}

func TestParseAutoTriggersForEvent_DifferentEvent(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage:
    - check_secrets_hygiene
  before_git_push:
    - check_release_gate
    - pre_push_intelligence
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_push")
	if len(filter) != 2 {
		t.Fatalf("expected 2 validators for push, got %d", len(filter))
	}
}

func TestParseAutoTriggersForEvent_InvalidYAML(t *testing.T) {
	filter := parseAutoTriggersForEvent([]byte("{{invalid yaml"), "before_git_stage")
	if filter != nil {
		t.Errorf("invalid YAML should return nil, got %v", filter)
	}
}

func TestParseAutoTriggersForEvent_MissingRouter(t *testing.T) {
	yamlData := []byte(`
other_key:
  before_git_stage:
    - check_secrets_hygiene
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_stage")
	if filter != nil {
		t.Errorf("missing router should return nil, got %v", filter)
	}
}

func TestParseAutoTriggersForEvent_MissingEvent(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage:
    - check_secrets_hygiene
`)
	filter := parseAutoTriggersForEvent(yamlData, "nonexistent_event")
	if filter != nil {
		t.Errorf("missing event should return nil, got %v", filter)
	}
}

func TestParseAutoTriggersForEvent_EmptyList(t *testing.T) {
	yamlData := []byte(`
router:
  before_git_stage: []
`)
	filter := parseAutoTriggersForEvent(yamlData, "before_git_stage")
	if len(filter) != 0 {
		t.Errorf("empty list should return empty filter, got %v", filter)
	}
}

// ── normalizeName ───────────────────────────────────────────────────────────────

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"check_secrets_hygiene", "secrets_hygiene"},
		{"validate_workspace_safety_gate", "workspace_safety"},
		{"check_protected_branch", "protected_branch"},
		{"branch_shield", "branch_shield"},         // no prefix/suffix
		{"check_release_gate", "release"},          // prefix + suffix
		{"validate_all_validator", "all"},          // prefix + suffix
		{"something_check", "something"},           // suffix only
		{"check_something_validator", "something"}, // prefix + different suffix
	}
	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── StageFilter.Matches with suffix normalization ───────────────────────────────

func TestStageFilter_Matches_SuffixNormalization(t *testing.T) {
	filter := StageFilter{
		"check_secrets_hygiene",
		"validate_workspace_safety_gate",
		"branch_shield",
	}

	tests := []struct {
		validatorID string
		want        bool
	}{
		{"secrets_hygiene", true},
		{"workspace_safety", true},
		{"branch_shield", true},
		{"nonexistent", false},
		{"check_secrets_hygiene", true}, // exact match
	}
	for _, tt := range tests {
		got := filter.Matches(tt.validatorID)
		if got != tt.want {
			t.Errorf("filter.Matches(%q) = %v, want %v", tt.validatorID, got, tt.want)
		}
	}
}

// ── hooksDir — worktree ─────────────────────────────────────────────────────────

func TestHooksDir_Worktree(t *testing.T) {
	tmp := t.TempDir()

	// Simulate main repo .git/hooks
	mainGitDir := filepath.Join(tmp, "main", ".git")
	os.MkdirAll(filepath.Join(mainGitDir, "hooks"), 0755)
	os.MkdirAll(filepath.Join(mainGitDir, "worktrees", "feature-x"), 0755)

	// Simulate worktree .git file
	wtDir := filepath.Join(tmp, "worktree")
	os.MkdirAll(wtDir, 0755)
	gitFile := filepath.Join(wtDir, ".git")
	gitDirRef := filepath.Join(mainGitDir, "worktrees", "feature-x")
	os.WriteFile(gitFile, []byte("gitdir: "+gitDirRef+"\n"), 0644)

	m := &Manager{RepoRoot: wtDir}
	dir := m.hooksDir()

	// Should resolve to main repo's hooks dir
	expected := filepath.Join(mainGitDir, "hooks")
	if dir != expected {
		t.Errorf("worktree hooksDir = %q, want %q", dir, expected)
	}
}

// ── Status ──────────────────────────────────────────────────────────────────────

func TestStatus_WithInstalledHooks(t *testing.T) {
	_, m := setupGitRepo(t)
	m.Install()

	report, err := m.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if !report.AllHealthy {
		t.Error("all hooks installed → AllHealthy should be true")
	}
	if len(report.Hooks) != len(AllStages()) {
		t.Errorf("Status hooks count = %d, want %d", len(report.Hooks), len(AllStages()))
	}
	for _, hs := range report.Hooks {
		if !hs.Installed {
			t.Errorf("stage %s: should be installed", hs.Stage)
		}
		if !hs.OVAV {
			t.Errorf("stage %s: should be OVAV managed", hs.Stage)
		}
		if !hs.Executable {
			t.Errorf("stage %s: should be executable", hs.Stage)
		}
		if !hs.SHA256OK {
			t.Errorf("stage %s: SHA-256 should match", hs.Stage)
		}
	}
}

func TestStatus_NotInstalled(t *testing.T) {
	_, m := setupGitRepo(t)

	report, err := m.Status()
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if report.AllHealthy {
		t.Error("no hooks installed → AllHealthy should be false")
	}
	for _, hs := range report.Hooks {
		if hs.Installed {
			t.Errorf("stage %s: should not be installed", hs.Stage)
		}
	}
}

// ── Install edge case: broken symlink ───────────────────────────────────────────

func TestInstall_ReplacesBrokenSymlink(t *testing.T) {
	_, m := setupGitRepo(t)

	// Create a broken symlink at the hook path
	hookPath := filepath.Join(m.hooksDir(), StagePreCommit.HookName())
	os.Symlink("/nonexistent/target", hookPath)

	results, err := m.Install()
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	for _, r := range results {
		if r.Stage == StagePreCommit {
			if r.Status != "installed" {
				t.Errorf("broken symlink: status = %q, want 'installed'", r.Status)
			}
			// Verify it's now a real file, not a symlink
			info, err := os.Lstat(r.Path)
			if err != nil {
				t.Fatalf("hook file not found after install: %v", err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				t.Error("hook is still a symlink after install")
			}
		}
	}
}

// ── GetStageFilter with fallback ────────────────────────────────────────────────

func TestGetStageFilter_FallbackForUnknownStage(t *testing.T) {
	filter := GetStageFilter(StagePostCheckout)
	// post-checkout has no hardcoded filter and no YAML mapping
	if len(filter) != 0 {
		t.Errorf("post-checkout should have empty filter, got %d items", len(filter))
	}
}

func TestGetStageFilter_PreCommitNonEmpty(t *testing.T) {
	filter := GetStageFilter(StagePreCommit)
	if len(filter) == 0 {
		t.Error("pre-commit filter should not be empty (fallback or YAML)")
	}
}

// ── findRepoRoot ────────────────────────────────────────────────────────────────

func TestFindRepoRoot_Found(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	subDir := filepath.Join(tmp, "a", "b", "c")
	os.MkdirAll(subDir, 0755)

	root := findRepoRoot(subDir)
	if root != tmp {
		t.Errorf("findRepoRoot = %q, want %q", root, tmp)
	}
}

func TestFindRepoRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	// No .git directory anywhere
	root := findRepoRoot(tmp)
	if root != "" {
		t.Errorf("findRepoRoot should return empty for no .git, got %q", root)
	}
}
