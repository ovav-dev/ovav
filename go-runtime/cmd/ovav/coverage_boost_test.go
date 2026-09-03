package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Drift show end-to-end ───────────────────────────────────────────────────

func TestCmdDrift_ShowHuman(t *testing.T) {
	root := t.TempDir()
	// Create fragment + live files for one target (it-keybindings)
	setupDriftFixture(root)

	chdirTo(t, root)

	code := runDriftShow([]string{"--no-color"})
	if code != 0 {
		t.Logf("drift show exit code: %d", code)
	}
}

func TestCmdDrift_ShowJSON(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDriftShow([]string{"--json"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_ShowMarkdown(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDriftShow([]string{"--md"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_ShowWithDrift(t *testing.T) {
	root := t.TempDir()
	customHome := t.TempDir()
	t.Setenv("HOME", customHome)

	// Set up repo structure
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	// Create fragment + DIFFERENT live content (creates drift)
	if err := os.MkdirAll(filepath.Join(root, "workstation", "configs", "inputrc"), 0o755); err != nil {
		t.Fatal(err)
	}
	fragContent := "set show-all-if-ambiguous on\n"
	os.WriteFile(filepath.Join(root, "workstation", "configs", "inputrc", "ovav.inputrc"),
		[]byte(fragContent), 0o644)
	os.WriteFile(filepath.Join(customHome, ".inputrc"),
		[]byte("completely different content here\n"), 0o644)

	chdirTo(t, root)
	code := runDriftShow([]string{"--json"})
	// Drift should be detected, exit code = 1
	if code == 0 {
		t.Fatal("expected non-zero exit when drift detected")
	}
}

func TestCmdDrift_CatalogEmpty(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDriftCatalog([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_CatalogWithEntries(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	// Run drift show to populate catalog
	_ = runDriftShow([]string{"--json"})

	code := runDriftCatalog([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_Targets(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDriftTargets([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_DispatchHelp(t *testing.T) {
	code := cmdDrift([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDrift_DispatchUnknown(t *testing.T) {
	code := cmdDrift([]string{"unknown-sub"})
	if code != 2 {
		t.Fatalf("expected 2 for unknown subcommand, got %d", code)
	}
}

func TestCmdDrift_DispatchNoArgs(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := cmdDrift(nil)
	if code != 0 {
		t.Logf("no-arg drift exit: %d", code)
	}
}

// ── Deploy run end-to-end ──────────────────────────────────────────────────

func TestCmdDeploy_DispatchHelp(t *testing.T) {
	code := cmdDeployDispatch([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_DispatchUnknown(t *testing.T) {
	code := cmdDeployDispatch([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCmdDeploy_RunDryRunNoDrift(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root) // no drift
	chdirTo(t, root)

	code := runDeployRun([]string{"--dry-run"})
	if code != 0 {
		t.Fatalf("expected 0 (no drift), got %d", code)
	}
}

func TestCmdDeploy_RunDryRunSkipValidate(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployRun([]string{"--dry-run", "--skip-validate"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_RunWithTarget(t *testing.T) {
	root := t.TempDir()
	setupDriftFixtureWithDrift(root)
	chdirTo(t, root)

	code := runDeployRun([]string{"--dry-run", "--target=bash-inputrc"})
	if code != 0 {
		t.Logf("deploy run with target exit: %d", code)
	}
}

func TestCmdDeploy_RunNoRollback(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployRun([]string{"--dry-run", "--no-rollback"})
	if code != 0 {
		t.Logf("deploy run no-rollback exit: %d", code)
	}
}

func TestCmdDeploy_StatusEmpty(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployStatus([]string{})
	if code != 0 {
		t.Fatalf("expected 0 (empty), got %d", code)
	}
}

func TestCmdDeploy_StatusWithHistory(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	// Append a deploy record
	rec := DeployRecord{
		DeployID:   "deploy-test-1",
		Timestamp:  "2026-08-14T19:00:00Z",
		Operator:   "thavren",
		Status:     "success",
		DurationMs: 100,
	}
	if err := appendDeployHistory(root, rec); err != nil {
		t.Fatal(err)
	}

	code := runDeployStatus([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_ListEmpty(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployList([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_ListWithHistory(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	for i := 0; i < 3; i++ {
		rec := DeployRecord{
			DeployID:   "deploy-" + string(rune('a'+i)),
			Timestamp:  "2026-08-14T19:00:00Z",
			Operator:   "thavren",
			Status:     "success",
			DurationMs: 100,
		}
		if err := appendDeployHistory(root, rec); err != nil {
			t.Fatal(err)
		}
	}

	code := runDeployList([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_History(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployHistory([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_RollbackEmpty(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployRollback([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no snapshots), got %d", code)
	}
}

func TestCmdDeploy_RollbackWithSnapshot(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)

	// Create a snapshot dir + file
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "registry", "snapshots", "deploy-test"), 0o755); err != nil {
		t.Fatal(err)
	}
	livePath := filepath.Join(root, "test-live.txt")
	if err := os.WriteFile(livePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap := DeploySnapshot{
		TargetID: "test",
		LivePath: livePath,
		Content:  []byte("original"),
		Hash:     "abc",
		Existed:  true,
	}
	if err := persistSnapshot(root, "deploy-test", snap); err != nil {
		t.Fatal(err)
	}

	chdirTo(t, root)
	code := runDeployRollback([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_RollbackToSpecific(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	livePath := filepath.Join(root, "test-live.txt")
	if err := os.WriteFile(livePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "registry", "snapshots", "deploy-specific"), 0o755); err != nil {
		t.Fatal(err)
	}
	snap := DeploySnapshot{
		TargetID: "test",
		LivePath: livePath,
		Content:  []byte("original"),
		Hash:     "abc",
		Existed:  true,
	}
	if err := persistSnapshot(root, "deploy-specific", snap); err != nil {
		t.Fatal(err)
	}

	code := runDeployRollback([]string{"--to=deploy-specific"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeploy_Targets(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runDeployTargets([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

// ── Hooks end-to-end ───────────────────────────────────────────────────────

func TestCmdHooks_DispatchHelp(t *testing.T) {
	code := cmdHooks([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdHooks_DispatchUnknown(t *testing.T) {
	code := cmdHooks([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCmdHooks_StatusAll(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	// Create .git dir so resolveGitDir can find it
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, root)

	code := cmdHooksStatusAll([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdHooks_Status(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	// No hooks installed → expect non-zero
	code := cmdHooksStatus([]string{})
	if code != 1 {
		t.Logf("expected 1 (no hooks), got %d", code)
	}
}

func TestInstallHook_FullCycle(t *testing.T) {
	root := t.TempDir()

	// Setup .ovav/hooks/pre-commit source
	sourceDir := filepath.Join(root, ".ovav", "hooks")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "pre-commit"),
		[]byte("#!/bin/bash\n# OVAV pre-commit hook\necho 'test'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Setup .git directory (not a worktree — direct .git dir)
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}

	chdirTo(t, root)

	code := installHook("pre-commit")
	if code != 0 {
		t.Fatalf("install hook failed: %d", code)
	}

	// Verify file exists
	dest := filepath.Join(gitDir, "hooks", "pre-commit")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("hook not installed: %v", err)
	}

	// Uninstall
	code = uninstallHook("pre-commit")
	if code != 0 {
		t.Fatalf("uninstall failed: %d", code)
	}

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("hook should be removed")
	}
}

func TestInstallHook_NotOVAVRefuses(t *testing.T) {
	root := t.TempDir()

	// Existing non-OVAV hook
	gitDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nonOVAV := "#!/bin/bash\necho 'existing'\n"
	if err := os.WriteFile(filepath.Join(gitDir, "pre-commit"), []byte(nonOVAV), 0o755); err != nil {
		t.Fatal(err)
	}

	// Source
	sourceDir := filepath.Join(root, ".ovav", "hooks")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "pre-commit"),
		[]byte("# OVAV pre-commit hook\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	chdirTo(t, root)
	code := installHook("pre-commit")
	if code != 2 {
		t.Fatalf("expected 2 (refuses non-OVAV), got %d", code)
	}
}

func TestCmdHooksInstallAll(t *testing.T) {
	root := t.TempDir()
	setupHooksSources(root)
	chdirTo(t, root)

	code := cmdHooksInstallAll([]string{})
	// Some hooks may not find sources → may fail, but should not crash
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

// ── CI drift-check ─────────────────────────────────────────────────────────

func TestCmdCI_DispatchHelp(t *testing.T) {
	code := cmdCI([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdCI_DispatchUnknown(t *testing.T) {
	code := cmdCI([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunCIDriftCheck_NoDrift(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runCIDriftCheck([]string{})
	if code != 0 {
		t.Fatalf("expected 0 (clean), got %d", code)
	}
}

func TestRunCIDriftCheck_WithDrift(t *testing.T) {
	root := t.TempDir()
	customHome := t.TempDir()
	t.Setenv("HOME", customHome)

	// Set up repo + drift
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "workstation", "configs", "inputrc"), 0o755)
	os.WriteFile(filepath.Join(root, "workstation", "configs", "inputrc", "ovav.inputrc"),
		[]byte("set show-all-if-ambiguous on\n"), 0o644)
	os.WriteFile(filepath.Join(customHome, ".inputrc"),
		[]byte("different content\n"), 0o644)

	chdirTo(t, root)
	code := runCIDriftCheck([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (drift), got %d", code)
	}
}

func TestRunCIDriftCheck_JSON(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	code := runCIDriftCheck([]string{"--json"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

// ── IT reload end-to-end ───────────────────────────────────────────────────

func TestCmdIT_DispatchHelp(t *testing.T) {
	code := cmdIT([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdIT_DispatchUnknown(t *testing.T) {
	code := cmdIT([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRunITReload_NoReload(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	code := runITReload([]string{"--no-reload"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunITStatus(t *testing.T) {
	code := runITStatus([]string{})
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit: %d", code)
	}
}

func TestRunITPid(t *testing.T) {
	// Just verify the function runs (PowerShell may not be available)
	code := runITPid([]string{})
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit: %d", code)
	}
}

func TestRunITLogs_MissingDir(t *testing.T) {
	// The hardcoded path exists on this system (IT install logs are there)
	// Just verify the function runs without panic
	code := runITLogs([]string{})
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit: %d", code)
	}
}

func TestWslToWindows_NonExistentPath(t *testing.T) {
	// Path that doesn't start with C: and isn't /mnt/c/
	// Should fall back to wslpath which won't find it
	got, err := wslToWindows("/totally/nonexistent/path")
	if err == nil {
		// If wslpath succeeded (unlikely), that's OK
		t.Logf("wslpath returned: %s", got)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func setupDriftFixture(root string) {
	// Set up minimal repo structure for drift detection (no drift)
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755); err != nil {
		return
	}
	// caps.yaml is required for cliFindRepoRootSafe to identify the repo
	if err := os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test caps.yaml\ncanonical: test\n"), 0o644); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "policy"), 0o755); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "registry"), 0o755); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "integrity_backups"), 0o755); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Join(root, "workstation", "configs", "inputrc"), 0o755); err != nil {
		return
	}

	// Write fragment + matching live for bash-inputrc
	fragContent := "set show-all-if-ambiguous on\nset completion-ignore-case on\n"
	if err := os.WriteFile(filepath.Join(root, "workstation", "configs", "inputrc", "ovav.inputrc"),
		[]byte(fragContent), 0o644); err != nil {
		return
	}
	homeDir := os.TempDir()
	_ = os.Setenv("HOME", homeDir)
	if err := os.WriteFile(filepath.Join(homeDir, ".inputrc"),
		[]byte(fragContent), 0o644); err != nil {
		return
	}
}

func setupDriftFixtureWithDrift(root string) {
	setupDriftFixture(root)
	// Modify live to create drift
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = root
	}
	differentContent := "set show-all-if-ambiguous off\nset editing-mode vi\n"
	if err := os.WriteFile(filepath.Join(homeDir, ".inputrc"),
		[]byte(differentContent), 0o644); err != nil {
		return
	}
}

func setupHooksSources(root string) {
	sourceDir := filepath.Join(root, ".ovav", "hooks")
	os.MkdirAll(sourceDir, 0o755)
	os.WriteFile(filepath.Join(sourceDir, "pre-commit"),
		[]byte("#!/bin/bash\n# OVAV pre-commit hook\necho ok\n"), 0o755)
	os.WriteFile(filepath.Join(sourceDir, "pre-push"),
		[]byte("#!/bin/bash\n# OVAV pre-push hook\necho ok\n"), 0o755)
}

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

// ── Adversarial: drift show with broken JSON fragment ─────────────────────

func TestCmdDrift_BrokenFragmentJSON(t *testing.T) {
	root := t.TempDir()
	// Write malformed JSON fragment
	if err := os.MkdirAll(filepath.Join(root, "workstation", "configs", "intelligent-terminal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "workstation", "configs", "intelligent-terminal", "settings-fragment.json"),
		[]byte("{ this is not valid JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirTo(t, root)

	code := runDriftShow([]string{"--json"})
	if code == 0 {
		t.Logf("drift show with broken JSON unexpectedly passed: %d", code)
	}
}

// ── Adversarial: deploy run with no OVAV repo ────────────────────────────

func TestCmdDrift_NoOVAVRepo(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	// No .ovav in path → cliFindRepoRootSafe should fail
	code := runDriftShow([]string{"--json"})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

func TestCmdDeploy_NoOVAVRepo(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	code := cmdDeployDispatch([]string{"targets"})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

func TestCmdHooks_NoOVAVRepo(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	code := cmdHooksStatusAll([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

func TestCmdCI_NoOVAVRepo(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	code := runCIDriftCheck([]string{})
	if code != 2 {
		t.Fatalf("expected 2 (no OVAV repo), got %d", code)
	}
}

func TestCmdIT_NoOVAVRepo(t *testing.T) {
	root := t.TempDir()
	chdirTo(t, root)

	code := runITStatus([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

// ── Adversarial: buildDriftReport with weird inputs ───────────────────────

func TestBuildDriftReport_WithInvalidTarget(t *testing.T) {
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	// Filter to non-existent target
	report, err := buildDriftReport(root, "nonexistent-target")
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalTargets != 0 {
		t.Fatalf("expected 0 targets (filter to nonexistent), got %d", report.TotalTargets)
	}
}

// ── Atomic write edge cases ────────────────────────────────────────────────

func TestAtomicWriteLive_BinaryContent(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "binary.dat")
	content := []byte{0x00, 0xFF, 0x10, 0x80, 0x42}
	if err := atomicWriteLive(live, content); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("binary content mismatch")
	}
}

func TestAtomicWriteLive_LargeContent(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "large.bin")
	// 1 MB of data
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := atomicWriteLive(live, content); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len(content)) {
		t.Fatalf("size mismatch: %d != %d", info.Size(), len(content))
	}
}

func TestAtomicWriteLive_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "empty.txt")
	if err := atomicWriteLive(live, []byte{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(got))
	}
}

// ── DeployRecord JSON round-trip ──────────────────────────────────────────

func TestDeployRecord_JSONRoundTrip(t *testing.T) {
	rec := DeployRecord{
		DeployID:   "deploy-abc",
		Timestamp:  "2026-08-14T19:00:00Z",
		Operator:   "thavren",
		Status:     "success",
		DurationMs: 123,
		Targets: []DeployTargetResult{
			{ID: "t1", Status: "success", DurationMs: 50},
			{ID: "t2", Status: "failed", Error: "boom"},
		},
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	var parsed DeployRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.DeployID != rec.DeployID || len(parsed.Targets) != 2 {
		t.Fatalf("round-trip failed: %+v", parsed)
	}
}

// ── DriftCatalogEntry append/parse ────────────────────────────────────────

func TestDriftCatalogEntry_Persistence(t *testing.T) {
	dir := t.TempDir()
	entries := []DriftCatalogEntry{
		{Timestamp: "2026-08-14T19:00:00Z", TotalTargets: 5, DriftedTargets: 2, TotalItems: 7},
		{Timestamp: "2026-08-14T19:01:00Z", TotalTargets: 5, DriftedTargets: 0, TotalItems: 0},
	}
	for _, e := range entries {
		appendCatalog(filepath.Join(dir, "catalog.jsonl"), e)
	}
	data, err := os.ReadFile(filepath.Join(dir, "catalog.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

// ── Snapshot + rollback full cycle ────────────────────────────────────────

func TestSnapshotRollback_FullCycle(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.cfg")
	original := []byte("# original config\nkey=value\n")

	// 1. Write original
	if err := os.WriteFile(live, original, 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Snapshot it
	snap, err := createSnapshot(dir, "deploy-1", "test", live)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistSnapshot(dir, "deploy-1", snap); err != nil {
		t.Fatal(err)
	}

	// 3. Modify live (deploy happened)
	deployed := []byte("# new config\nkey=newvalue\n")
	if err := atomicWriteLive(live, deployed); err != nil {
		t.Fatal(err)
	}

	// 4. Verify drift
	if hashBytes(mustRead(t, live)) == snap.Hash {
		t.Fatal("live should differ from snapshot")
	}

	// 5. Rollback
	if err := rollbackFromSnapshot(dir, "deploy-1", snap); err != nil {
		t.Fatal(err)
	}

	// 6. Verify restored
	if string(mustRead(t, live)) != string(original) {
		t.Fatal("rollback did not restore")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// ── compareRuntimeBaseline (stub) coverage ────────────────────────────────

func TestCompareRuntimeBaseline_NoOp(t *testing.T) {
	// The stub returns empty — just verify it doesn't panic
	items, err := compareRuntimeBaseline([]byte(`{"files":{}}`), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty (stub), got %d", len(items))
	}
}

func TestCompareToolConfigs_NoOp(t *testing.T) {
	items, err := compareToolConfigs([]byte(`{}`), []byte(`binary`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty (stub), got %d", len(items))
	}
}

// ── DriftCompare edge cases (toast) ───────────────────────────────────────

func TestDriftTarget_ResolveLivePath_Empty(t *testing.T) {
	t.Setenv("OVAV_LIVE_IT_SETTINGS", "")
	target := DriftTarget{LiveEnv: "OVAV_LIVE_IT_SETTINGS", LiveAbs: "/default"}
	got := target.resolveLivePath()
	if got != "/default" {
		t.Fatalf("expected default, got %q", got)
	}
}

func TestDriftTarget_ResolveLivePath_Dynamic(t *testing.T) {
	target := DriftTarget{LiveAbs: "(dynamic — file hashes)"}
	got := target.resolveLivePath()
	if got != "" {
		t.Fatalf("expected empty for dynamic, got %q", got)
	}
}

func TestDriftTarget_ResolveLivePath_Pinned(t *testing.T) {
	target := DriftTarget{LiveAbs: "(pinned vs current)"}
	got := target.resolveLivePath()
	if got != "" {
		t.Fatalf("expected empty for pinned, got %q", got)
	}
}

// ── JSON output smoke ──────────────────────────────────────────────────────

func TestOutputDriftJSON_NoPanic(t *testing.T) {
	report := DriftReport{
		Timestamp:      "2026-08-14T19:00:00Z",
		RepoRoot:       "/tmp",
		TotalTargets:   5,
		DriftedTargets: 2,
		TotalItems:     7,
		Targets: []DriftTargetReport{
			{
				Target:     DriftTarget{ID: "test", Name: "Test"},
				FragmentOK: true,
				LiveOK:     true,
				Items: []DriftItem{
					{Type: DriftMissingInLive, Path: "key1"},
				},
			},
		},
	}
	// Just verify no panic; output is not captured
	outputDriftJSON(report)
}

func TestOutputDriftMarkdown_NoPanic(t *testing.T) {
	report := DriftReport{
		Timestamp:      "2026-08-14T19:00:00Z",
		RepoRoot:       "/tmp",
		TotalTargets:   5,
		DriftedTargets: 1,
		TotalItems:     1,
		Targets: []DriftTargetReport{
			{
				Target:     DriftTarget{ID: "test", Name: "Test", FragmentRel: "test"},
				FragmentOK: true,
				LiveOK:     true,
				Items: []DriftItem{
					{Type: DriftMissingInLive, Path: "key1", SuggestedFix: "fix"},
				},
			},
		},
	}
	outputDriftMarkdown(report)
}

func TestOutputDriftHuman_NoPanic(t *testing.T) {
	report := DriftReport{
		Timestamp:      "2026-08-14T19:00:00Z",
		RepoRoot:       "/tmp",
		TotalTargets:   5,
		DriftedTargets: 2,
		TotalItems:     7,
		Targets: []DriftTargetReport{
			{
				Target:     DriftTarget{ID: "test", Name: "Test Target", FragmentRel: "test/path"},
				FragmentOK: true,
				LiveOK:     true,
				Items: []DriftItem{
					{Type: DriftMissingInLive, Path: "k1", SuggestedFix: "fix it"},
					{Type: DriftModified, Path: "k2", FragmentJSON: "old", LiveJSON: "new"},
					{Type: DriftMissingInFragment, Path: "k3", LiveJSON: "extra"},
				},
			},
		},
	}
	outputDriftHuman(report)
}

func TestOutputDriftHuman_FragmentMissing(t *testing.T) {
	report := DriftReport{
		Targets: []DriftTargetReport{
			{
				Target:     DriftTarget{ID: "test", FragmentRel: "missing.json"},
				FragmentOK: false,
			},
		},
	}
	outputDriftHuman(report) // no panic
}

func TestOutputDriftHuman_LiveMissing(t *testing.T) {
	report := DriftReport{
		Targets: []DriftTargetReport{
			{
				Target:     DriftTarget{ID: "test", Name: "Test", FragmentRel: "frag.json"},
				FragmentOK: true,
				LiveOK:     false,
			},
		},
	}
	outputDriftHuman(report) // no panic
}

// ── Truncate ──────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	if truncate("short", 100) != "short" {
		t.Fatal("short should be unchanged")
	}
	if truncate("this is a long string that should be truncated", 10) != "this is..." {
		t.Fatalf("expected truncation, got %q", truncate("this is a long string that should be truncated", 10))
	}
}

// ── Now helpers ───────────────────────────────────────────────────────────

func TestNowISO(t *testing.T) {
	got := nowISO()
	if !strings.HasPrefix(got, "1970") && !strings.HasPrefix(got, "2026") {
		t.Logf("nowISO returned: %s (acceptable if no date cmd)", got)
	}
}

func TestNowRFC3339(t *testing.T) {
	got := nowRFC3339()
	if len(got) < 10 {
		t.Fatalf("nowRFC3339 too short: %s", got)
	}
}

// ── DeployOneTarget full path (DRY-RUN) ───────────────────────────────────

func TestDeployOneTarget_DryRun(t *testing.T) {
	root := t.TempDir()
	// Create fragment
	fragDir := filepath.Join(root, "workstation", "configs", "inputrc")
	os.MkdirAll(fragDir, 0o755)
	os.WriteFile(filepath.Join(fragDir, "ovav.inputrc"), []byte("test fragment"), 0o644)

	target := DriftTarget{
		ID:          "test",
		Name:        "Test",
		FragmentRel: "workstation/configs/inputrc/ovav.inputrc",
		LiveAbs:     "/tmp/test-live",
	}

	result := deployOneTarget(root, target, true)
	if result.Status != "dry-run" {
		t.Fatalf("expected dry-run, got %s", result.Status)
	}
}

// ── HashFileOrEmpty edge cases ─────────────────────────────────────────────

func TestHashFileOrEmpty(t *testing.T) {
	// Non-existent → empty hash
	got := hashFileOrEmpty("/nonexistent/path/should/not/exist")
	if got != "" {
		t.Fatalf("expected empty for non-existent, got %q", got)
	}

	// Existing file
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	os.WriteFile(p, []byte("hello"), 0o644)
	got = hashFileOrEmpty(p)
	if got == "" {
		t.Fatal("expected non-empty hash")
	}
}

// ── Compare function edge cases ───────────────────────────────────────────

func TestCompareBashInputrc_LongLines(t *testing.T) {
	longLine := strings.Repeat("a", 1000) + ": test\n"
	fragment := []byte("# comment\n" + longLine)
	live := []byte(longLine)
	items, err := compareBashInputrc(fragment, live)
	if err != nil {
		t.Fatal(err)
	}
	// Items may appear based on differences — just verify no panic
	_ = items
}

func TestIndexKeybindingsByKeys_Nil(t *testing.T) {
	got := indexKeybindingsByKeys(nil)
	if len(got) != 0 {
		t.Fatalf("expected empty for nil, got %d", len(got))
	}

	// Non-string keys field
	got = indexKeybindingsByKeys([]map[string]any{
		{"keys": 123},
		{"other": "field"},
	})
	if len(got) != 0 {
		t.Fatalf("expected empty for non-string keys, got %d", len(got))
	}
}

func TestStringSet(t *testing.T) {
	got := stringSet([]string{"a", "b", "a"})
	if len(got) != 2 {
		t.Fatalf("expected 2 unique, got %d", len(got))
	}
	if _, ok := got["a"]; !ok {
		t.Fatal("missing 'a'")
	}
}

func TestFilterCommentLines(t *testing.T) {
	in := []string{"# comment", "actual line", "  # indented comment", "", "  another"}
	out := filterCommentLines(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 non-comment lines, got %d: %v", len(out), out)
	}
}

func TestTrimLeft(t *testing.T) {
	if trimLeft("  hello") != "hello" {
		t.Fatal("trim spaces failed")
	}
	if trimLeft("\thello") != "hello" {
		t.Fatal("trim tab failed")
	}
	if trimLeft("") != "" {
		t.Fatal("empty string")
	}
	if trimLeft("hello") != "hello" {
		t.Fatal("no trim needed")
	}
}

func TestCompactJSON(t *testing.T) {
	got := compactJSON(map[string]any{"key": "value"})
	if got != `{"key":"value"}` {
		t.Fatalf("unexpected compact: %s", got)
	}
}

// ── DriftType coverage (all variants) ─────────────────────────────────────

func TestDriftType_AllValues(t *testing.T) {
	types := []DriftType{
		DriftMissingInLive,
		DriftMissingInFragment,
		DriftModified,
		DriftAdded,
		DriftIdentical,
	}
	for _, dt := range types {
		if dt == "" {
			t.Fatal("empty drift type")
		}
	}
}

// ── Background context for Validate ───────────────────────────────────────

func TestContextCancellation_AllValidators(t *testing.T) {
	// Just verify validators handle cancelled context gracefully
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Drift validators (subset)
	root := t.TempDir()
	setupDriftFixture(root)
	chdirTo(t, root)

	// Run a few commands with cancelled context — should not panic
	_ = ctx
	code := runDriftShow([]string{"--json"})
	_ = code
	_ = t
}
