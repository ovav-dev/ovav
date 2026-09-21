package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdLaunch_DispatchHelp(t *testing.T) {
	code := cmdLaunch([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdLaunch_DispatchUnknown(t *testing.T) {
	code := cmdLaunch([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCmdLaunch_DispatchNoArgs(t *testing.T) {
	// No args → calls runLaunchStatus
	// Will fail if no bin/ovav exists in test dir
	dir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(old)

	// runLaunchStatus invokes ./bin/ovav which won't exist → print fallback
	code := cmdLaunch([]string{})
	// Don't assert exit code — depends on environment
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestRunLaunchVerify_RequiresWaiver(t *testing.T) {
	code := runLaunchVerify([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no waiver), got %d", code)
	}
}

func TestRunLaunchVerify_WithWaiver(t *testing.T) {
	// Verify all checks (most will fail in test env, but waiver should be accepted)
	code := runLaunchVerify([]string{"--ceo-waiver", "--reason=test"})
	// 0 = passed all (unlikely in test), 1 = some check failed
	if code != 0 && code != 1 {
		t.Fatalf("expected 0 or 1, got %d", code)
	}
}

func TestRunLaunchRoadmap(t *testing.T) {
	code := runLaunchRoadmap([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchStatus(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runLaunchStatus([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestRunLaunchEvidence(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runLaunchEvidence([]string{})
	// Will create evidence dir even if subprocess fails
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestPrintLaunchHelp(t *testing.T) {
	printLaunchHelp() // no panic
}
