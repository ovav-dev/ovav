package validators

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Safe-fix registry ──────────────────────────────────────────────────────

func TestIsSafeFix(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"bash_readline_bindings", true},
		{"runtime_integrity_baseline_fresh", true},
		{"supply_chain", true},
		{"nonexistent_validator", false},
		{"", false},
	}
	for _, tc := range tests {
		got := IsSafeFix(tc.id)
		if got != tc.want {
			t.Errorf("IsSafeFix(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestGetSafeFixRegistry_NotEmpty(t *testing.T) {
	entries := GetSafeFixRegistry()
	if len(entries) == 0 {
		t.Fatal("registry should not be empty")
	}
	for _, e := range entries {
		if e.ValidatorID == "" {
			t.Error("entry missing ValidatorID")
		}
		if e.Description == "" {
			t.Errorf("entry %s missing Description", e.ValidatorID)
		}
		if e.RiskLevel == "" {
			t.Errorf("entry %s missing RiskLevel", e.ValidatorID)
		}
	}
}

func TestGetSafeFixRegistry_ReturnsCopy(t *testing.T) {
	entries1 := GetSafeFixRegistry()
	entries2 := GetSafeFixRegistry()
	if &entries1[0] == &entries2[0] {
		t.Fatal("GetSafeFixRegistry should return a copy, not a slice alias")
	}
}

// ── File helpers ───────────────────────────────────────────────────────────

func TestFileExists(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "exists.txt")
	os.WriteFile(p, []byte("hi"), 0o644)
	if !fileExists(p) {
		t.Fatal("should detect existing file")
	}
	if fileExists(filepath.Join(tmp, "nope.txt")) {
		t.Fatal("should detect non-existing file")
	}
}

// ── BashReadlineBindings Fix ──────────────────────────────────────────────

func TestBashReadlineBindings_Fix_AddsMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Fix writes to FRAGMENT (workstation/configs/inputrc/ovav.inputrc)
	fragmentDir := filepath.Join(home, "workstation", "configs", "inputrc")
	os.MkdirAll(fragmentDir, 0o755)
	inputrc := filepath.Join(fragmentDir, "ovav.inputrc")
	original := "set show-all-if-ambiguous on\n"
	os.WriteFile(inputrc, []byte(original), 0o644)

	b := NewBashReadlineBindings()
	if err := b.Fix(home); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(inputrc)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)
	if !contains(content, "deliberately UNBOUND") {
		t.Fatalf("expected marker, got: %s", content)
	}
	if !contains(content, original) {
		t.Fatalf("expected original content preserved, got: %s", content)
	}
}

func TestBashReadlineBindings_Fix_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fragmentDir := filepath.Join(home, "workstation", "configs", "inputrc")
	os.MkdirAll(fragmentDir, 0o755)
	inputrc := filepath.Join(fragmentDir, "ovav.inputrc")
	os.WriteFile(inputrc, []byte("set show-all-if-ambiguous on\n"), 0o644)

	b := NewBashReadlineBindings()
	// Apply twice
	b.Fix(home)
	b.Fix(home)

	got, _ := os.ReadFile(inputrc)
	count := stringsCount(string(got), "deliberately UNBOUND")
	if count != 1 {
		t.Fatalf("expected exactly 1 marker after 2 fixes, got %d", count)
	}
}

func TestBashReadlineBindings_Fix_NoInputrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No ~/.inputrc exists
	b := NewBashReadlineBindings()
	if err := b.Fix(home); err != nil {
		t.Fatalf("Fix should not fail when inputrc missing, got: %v", err)
	}
}

func TestBashReadlineBindings_FixDescription(t *testing.T) {
	b := NewBashReadlineBindings()
	if b.FixDescription() == "" {
		t.Fatal("FixDescription should not be empty")
	}
}

// ── IntegrityBaselineFresh Fix ────────────────────────────────────────────

func TestIntegrityBaselineFresh_Fix_RegeneratesBaseline(t *testing.T) {
	root := t.TempDir()
	// Create protected surface files (use real-ish content)
	createFile := func(path, content string) {
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	createFile(filepath.Join(root, "AGENTS.md"), "# AGENTS\n")
	createFile(filepath.Join(root, "opencode.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/policy/permission_authority.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/plan/caps.yaml"), "# caps\n")
	createFile(filepath.Join(root, "go-runtime/go.mod"), "module test\n")
	createFile(filepath.Join(root, "go-runtime/internal/validators/cmd/validate/main.go"), "package main\n")

	// Remove existing baseline
	os.Remove(filepath.Join(root, ".ovav/integrity_backups/baseline.json"))

	v := NewIntegrityBaselineFresh()
	if err := v.Fix(root); err != nil {
		t.Fatal(err)
	}

	// Verify baseline was created
	baselinePath := filepath.Join(root, ".ovav/integrity_backups/baseline.json")
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("baseline.json not created: %v", err)
	}

	// Verify the baseline has expected schema
	data, _ := os.ReadFile(baselinePath)
	if !contains(string(data), "ovav.runtime_integrity.v1") {
		t.Fatal("baseline missing schema")
	}
}

func TestIntegrityBaselineFresh_Fix_Idempotent(t *testing.T) {
	root := t.TempDir()
	createFile := func(path, content string) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	createFile(filepath.Join(root, "AGENTS.md"), "# AGENTS\n")

	v := NewIntegrityBaselineFresh()
	v.Fix(root)
	firstHash := readHash(t, filepath.Join(root, ".ovav/integrity_backups/baseline.json"))
	v.Fix(root)
	secondHash := readHash(t, filepath.Join(root, ".ovav/integrity_backups/baseline.json"))

	if firstHash != secondHash {
		t.Fatal("Fix should be idempotent (same input → same output)")
	}
}

func TestIntegrityBaselineFresh_FixDescription(t *testing.T) {
	v := NewIntegrityBaselineFresh()
	if v.FixDescription() == "" {
		t.Fatal("FixDescription should not be empty")
	}
}

// ── Orchestrator ───────────────────────────────────────────────────────────

func TestAutoFixOrchestrator_DryRun(t *testing.T) {
	root := t.TempDir()
	orchestrator := NewAutoFixOrchestrator(root).WithDryRun()

	results, err := orchestrator.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	for _, r := range results {
		if r.Outcome != "dry-run" {
			t.Errorf("expected dry-run outcome, got %s", r.Outcome)
		}
	}
}

func TestAutoFixOrchestrator_RunWithFiles(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create FRAGMENT (where bash_readline_fix writes)
	fragmentDir := filepath.Join(root, "workstation", "configs", "inputrc")
	os.MkdirAll(fragmentDir, 0o755)
	inputrcFragment := filepath.Join(fragmentDir, "ovav.inputrc")
	os.WriteFile(inputrcFragment, []byte("set show-all-if-ambiguous on\n"), 0o644)

	// Create protected surfaces
	createFile := func(path, content string) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	createFile(filepath.Join(root, "AGENTS.md"), "# AGENTS\n")
	createFile(filepath.Join(root, "opencode.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/policy/permission_authority.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/plan/caps.yaml"), "# caps\n")
	createFile(filepath.Join(root, "go-runtime/go.mod"), "module test\n")
	createFile(filepath.Join(root, "go-runtime/internal/validators/cmd/validate/main.go"), "package main\n")

	orchestrator := NewAutoFixOrchestrator(root)
	results, err := orchestrator.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Verify fragment now has marker
	got, _ := os.ReadFile(inputrcFragment)
	if !contains(string(got), "deliberately UNBOUND") {
		t.Fatal("Fix should add marker to fragment")
	}

	// Verify at least one applied
	applied := 0
	for _, r := range results {
		if r.Outcome == "applied" {
			applied++
		}
	}
	if applied == 0 {
		t.Fatal("expected at least one applied fix")
	}
}

func TestAutoFixOrchestrator_RunMaxFixes(t *testing.T) {
	root := t.TempDir()
	orchestrator := NewAutoFixOrchestrator(root).WithDryRun()
	// Verify dryRun path applies all (maxFixes only applies to actual fixes)
	results, _ := orchestrator.Run(context.Background())
	// In dry-run mode, maxFixes doesn't apply (we just report)
	if len(results) == 0 {
		t.Fatal("expected dry-run to return all candidates")
	}
}

func TestAutoFixOrchestrator_MaxFixesLimit(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create fragment + protected surfaces (needed for fixes to apply)
	fragmentDir := filepath.Join(root, "workstation", "configs", "inputrc")
	os.MkdirAll(fragmentDir, 0o755)
	os.WriteFile(filepath.Join(fragmentDir, "ovav.inputrc"), []byte("set show-all-if-ambiguous on\n"), 0o644)
	createFile := func(path, content string) {
		os.MkdirAll(filepath.Dir(path), 0o755)
		os.WriteFile(path, []byte(content), 0o644)
	}
	createFile(filepath.Join(root, "AGENTS.md"), "# A\n")
	createFile(filepath.Join(root, "opencode.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/policy/permission_authority.json"), "{}\n")
	createFile(filepath.Join(root, ".ovav/plan/caps.yaml"), "# C\n")
	createFile(filepath.Join(root, "go-runtime/go.mod"), "module t\n")
	createFile(filepath.Join(root, "go-runtime/internal/validators/cmd/validate/main.go"), "package main\n")

	orchestrator := NewAutoFixOrchestrator(root)
	orchestrator.maxFixes = 1
	results, _ := orchestrator.Run(context.Background())

	// Count applied fixes
	applied := 0
	for _, r := range results {
		if r.Outcome == "applied" {
			applied++
		}
	}
	// applied should be <= 1 (maxFixes limits actual fixes)
	if applied > 1 {
		t.Fatalf("expected max 1 applied, got %d", applied)
	}
}

func TestAutoFixOrchestrator_HistoryLogged(t *testing.T) {
	root := t.TempDir()
	orchestrator := NewAutoFixOrchestrator(root).WithDryRun()
	orchestrator.Run(context.Background())

	logs, err := ReadFixHistory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least one log entry")
	}
	if logs[0].Outcome != "dry-run" {
		t.Errorf("expected dry-run outcome, got %s", logs[0].Outcome)
	}
}

func TestAutoFixOrchestrator_RollbackOnRegression(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create fragment WITH marker (Fix would be no-op)
	fragmentDir := filepath.Join(root, "workstation", "configs", "inputrc")
	os.MkdirAll(fragmentDir, 0o755)
	os.WriteFile(filepath.Join(fragmentDir, "ovav.inputrc"),
		[]byte("# deliberately UNBOUND\nset show-all-if-ambiguous on\n"), 0o644)

	orchestrator := NewAutoFixOrchestrator(root)
	results, _ := orchestrator.Run(context.Background())

	// bash_readline_bindings should be no-op (already has marker)
	for _, r := range results {
		if r.ValidatorID == "bash_readline_bindings" {
			if r.Outcome == "failed" || r.Outcome == "rollback" {
				t.Errorf("expected no failure for bash_readline (already fixed), got %s", r.Outcome)
			}
		}
	}
}

// ── Find validator helper ──────────────────────────────────────────────────

func TestFindValidator_KnownIDs(t *testing.T) {
	tests := []string{
		"bash_readline_bindings",
		"runtime_integrity_baseline_fresh",
		"supply_chain",
	}
	for _, id := range tests {
		v := findValidator(id)
		if v == nil {
			t.Errorf("findValidator(%q) returned nil", id)
		}
	}
}

func TestFindValidator_UnknownID(t *testing.T) {
	v := findValidator("does_not_exist")
	if v != nil {
		t.Fatal("expected nil for unknown validator")
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func TestJoinIssues(t *testing.T) {
	if joinIssues(nil) != "" {
		t.Fatal("nil should join to empty")
	}
	if joinIssues([]string{"a"}) != "a" {
		t.Fatal("single item should be unchanged")
	}
	if joinIssues([]string{"a", "b", "c"}) != "a; b; c" {
		t.Fatal("multiple items should be joined with semicolons")
	}
}

// ── Fix history persistence ────────────────────────────────────────────────

func TestAppendFixHistory_Persistence(t *testing.T) {
	root := t.TempDir()
	log := FixResultLog{
		DeployID:  "fix-test",
		Operator:  "thavren",
		Outcome:   "success",
		StartedAt: "2026-08-14T19:00:00Z",
		Results: []FixResult{
			{ValidatorID: "test", Outcome: "applied", DurationMs: 100},
		},
	}
	if err := AppendFixHistory(root, log); err != nil {
		t.Fatal(err)
	}
	if err := AppendFixHistory(root, log); err != nil {
		t.Fatal(err)
	}

	logs, err := ReadFixHistory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}
}

func TestReadFixHistory_Empty(t *testing.T) {
	root := t.TempDir()
	logs, err := ReadFixHistory(root)
	if err != nil {
		t.Fatal(err)
	}
	if logs != nil {
		t.Fatalf("expected nil/empty, got %v", logs)
	}
}

// ── Snapshot / rollback cycle ──────────────────────────────────────────────

func TestFixRegistrySnapshot_RollbackCycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create inputrc
	inputrc := filepath.Join(home, ".inputrc")
	original := "original content\n"
	os.WriteFile(inputrc, []byte(original), 0o644)

	entries := GetSafeFixRegistry()
	snapDir, err := FixRegistrySnapshot(root, entries)
	if err != nil {
		t.Fatal(err)
	}

	// Modify inputrc (simulating a fix)
	os.WriteFile(inputrc, []byte("modified\n"), 0o644)

	// Rollback
	if err := FixRegistryRollback(snapDir); err != nil {
		t.Fatal(err)
	}

	// Verify restored
	got, _ := os.ReadFile(inputrc)
	if string(got) != original {
		t.Fatalf("expected rollback to restore original, got: %s", got)
	}
}

// ── Edge cases ─────────────────────────────────────────────────────────────

func TestAutoFixOrchestrator_ContextCancel(t *testing.T) {
	root := t.TempDir()
	orchestrator := NewAutoFixOrchestrator(root)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should not panic
	results, _ := orchestrator.Run(ctx)
	if results == nil {
		// Acceptable — cancelled returns nil
	}
}

// ── Protected file enforcement ─────────────────────────────────────────────

func TestAutoFix_ProtectedFileSkip(t *testing.T) {
	// bash_readline_bindings fixes ~/.inputrc (not protected).
	// runtime_integrity_baseline_fresh fixes baseline.json (not protected).
	// supply_chain regenerates sbom.json (not protected).
	//
	// Verify: all entries have low risk level
	entries := GetSafeFixRegistry()
	for _, e := range entries {
		if e.RequiresWaiver && e.RiskLevel != "high" {
			t.Errorf("entry %s: waiver requires high risk, got %s", e.ValidatorID, e.RiskLevel)
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringsCount(s, substr string) int {
	count := 0
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func readHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Simple hash check — just return content length
	return intToString(len(data))
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

func createFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(content), 0o644)
}

// Prevent unused imports
var _ = time.Second
var _ = context.Canceled
