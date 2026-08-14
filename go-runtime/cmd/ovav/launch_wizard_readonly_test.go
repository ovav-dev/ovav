package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for the read-only wizard mode.

func TestRunLaunchWizardReadOnly_StatusMode(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runLaunchWizardReadOnly(root, "status")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchWizardReadOnly_InfoMode(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runLaunchWizardReadOnly(root, "info")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchWizardReadOnly_NoOVAVRepo(t *testing.T) {
	// Without /root/.ovav/plan/caps.yaml, should fail
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Will fail at cliFindRepoRootSafe
	// Just verify it doesn't panic
	_ = runLaunchWizardReadOnly(dir, "info")
}

func TestRunLaunchWizardReadOnly_PinnedBaseline(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)
	// Pre-pin baseline
	os.MkdirAll(filepath.Join(root, ".ovav", "integrity_backups"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json"),
		[]byte(`{"files":{}}`), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	// Now pinned should pass
	code := runLaunchWizardReadOnly(root, "status")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchWizard_HelpFlag(t *testing.T) {
	code := runLaunchWizard([]string{"--help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchWizard_NoArgs(t *testing.T) {
	// Will try to use ./bin/ovav which doesn't exist in test env
	// Should not panic, may return 1
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	code := runLaunchWizard([]string{})
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestCmdLaunch_RoutesFlagsToWizard(t *testing.T) {
	// --status should go through runLaunchWizard → runLaunchWizardReadOnly
	// (not return "unknown subcommand")
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// Should NOT return 2 (which is "unknown subcommand")
	code := cmdLaunch([]string{"--status"})
	if code == 2 {
		t.Fatal("--status should not be treated as unknown subcommand")
	}
}