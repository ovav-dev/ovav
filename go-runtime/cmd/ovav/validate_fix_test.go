package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdValidateFix_Help(t *testing.T) {
	code := cmdValidateFix([]string{"--help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdValidateFix_NoArgs(t *testing.T) {
	// No args + no repo → walks up to find OVAV root, fails if not found
	// Use a clean temp dir WITHOUT .ovav
	root := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdValidateFix([]string{})
	// Without OVAV repo, cliFindRepoRootSafe fails → returns 1
	if code != 1 {
		t.Fatalf("expected 1 (no repo), got %d", code)
	}
}

func TestCmdValidateFix_List(t *testing.T) {
	root := t.TempDir()
	// Setup minimal repo
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdValidateFix([]string{"--list"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdValidateFix_DryRun(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdValidateFix([]string{"--dry-run"})
	if code != 0 {
		t.Fatalf("expected 0 (dry-run always succeeds), got %d", code)
	}
}

func TestCmdValidateFix_StrategyFlags(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdValidateFix([]string{"--strategy=atomic", "--dry-run"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}

	code = cmdValidateFix([]string{"--strategy=best-effort", "--dry-run"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestListSafeFix_NoRepo(t *testing.T) {
	// Without OVAV repo, listSafeFix uses root directly
	dir := t.TempDir()
	code := listSafeFix(dir)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestPrintValidateFixHelp(t *testing.T) {
	printValidateFixHelp() // just verify no panic
}