package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunChaosTest_AtomicWrite(t *testing.T) {
	scenario := chaosScenarios[0] // atomic_write_invariant
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "passed" {
		t.Fatalf("atomic_write_invariant failed: %s — %s", result.Outcome, result.Error)
	}
}

func TestRunChaosTest_SnapshotInvariant(t *testing.T) {
	scenario := chaosScenarios[1]
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "passed" {
		t.Fatalf("snapshot_invariant failed: %s — %s", result.Outcome, result.Error)
	}
}

func TestRunChaosTest_Rollback(t *testing.T) {
	scenario := chaosScenarios[2]
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "passed" {
		t.Fatalf("rollback_invariant failed: %s — %s", result.Outcome, result.Error)
	}
}

func TestRunChaosTest_Concurrency(t *testing.T) {
	scenario := chaosScenarios[3]
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "passed" {
		t.Fatalf("concurrency_invariant failed: %s — %s", result.Outcome, result.Error)
	}
}

func TestRunChaosTest_ContextCancel(t *testing.T) {
	scenario := chaosScenarios[4]
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "passed" {
		t.Fatalf("context_cancel_invariant failed: %s — %s", result.Outcome, result.Error)
	}
}

func TestRunChaosTest_InvariantViolation(t *testing.T) {
	// Custom scenario that violates invariant
	scenario := ChaosScenario{
		ID:          "test_violation",
		Description: "Force invariant violation",
		Setup:       func(root string) error { return nil },
		Restore:     func(root string) error { return nil },
		Verify: func(root string) error {
			return os.ErrInvalid // always fails
		},
	}
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "failed" {
		t.Fatalf("expected failed, got %s", result.Outcome)
	}
}

func TestRunChaosTest_SetupFails(t *testing.T) {
	scenario := ChaosScenario{
		ID:          "test_setup_fail",
		Description: "Setup fails",
		Setup: func(root string) error {
			return os.ErrPermission
		},
		Restore: func(root string) error { return nil },
		Verify:  func(root string) error { return nil },
	}
	tempDir, _ := os.MkdirTemp("", "chaos-test-*")
	defer os.RemoveAll(tempDir)

	result := RunChaosTest(scenario, tempDir)
	if result.Outcome != "skipped" {
		t.Fatalf("expected skipped, got %s", result.Outcome)
	}
}

func TestCmdDeployChaos_List(t *testing.T) {
	root := t.TempDir()
	// Setup minimal repo
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdDeployChaos([]string{"--list"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeployChaos_AllScenarios(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdDeployChaos([]string{})
	if code != 0 {
		t.Fatalf("expected 0 (all invariants pass), got %d", code)
	}
}

func TestCmdDeployChaos_SpecificScenario(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdDeployChaos([]string{"--scenario=atomic_write_invariant"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDeployChaos_NoRepo(t *testing.T) {
	root := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := cmdDeployChaos([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

func TestCmdDeployChaos_Help(t *testing.T) {
	code := cmdDeployChaos([]string{"--help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestChaosHistoryLog(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	// Run chaos → should log
	cmdDeployChaos([]string{})

	// Verify chaos_history.jsonl exists (allow small delay for file write)
	historyPath := filepath.Join(root, ".ovav", "registry", "chaos_history.jsonl")
	if _, err := os.Stat(historyPath); err != nil {
		// Print debug info
		entries, _ := os.ReadDir(filepath.Join(root, ".ovav", "registry"))
		t.Logf("Files in .ovav/registry: %v", entries)
		t.Fatalf("chaos history not written: %v", err)
	}
}
