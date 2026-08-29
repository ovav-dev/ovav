package ows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// recovery_test.go — Tests targeting recovery.go gaps
// Verify (8.1%), truncateOutput (0%), pruneStaleWorktrees (44%)
// ═══════════════════════════════════════════════════════════════════════════

// ── truncateOutput ───────────────────────────────────────────────────────

func TestTruncateOutput_Short(t *testing.T) {
	got := truncateOutput("hello", 100)
	if got != "hello" {
		t.Errorf("truncateOutput short = %q, want hello", got)
	}
}

func TestTruncateOutput_Exact(t *testing.T) {
	got := truncateOutput("12345", 5)
	if got != "12345" {
		t.Errorf("truncateOutput exact = %q, want 12345", got)
	}
}

func TestTruncateOutput_TooLong(t *testing.T) {
	got := truncateOutput("hello world this is long", 10)
	if got != "hello worl..." {
		t.Errorf("truncateOutput long = %q, want 'hello worl...'", got)
	}
}

func TestTruncateOutput_Empty(t *testing.T) {
	got := truncateOutput("", 10)
	if got != "" {
		t.Errorf("truncateOutput empty = %q, want empty", got)
	}
}

func TestTruncateOutput_Whitespace(t *testing.T) {
	got := truncateOutput("  hello  ", 100)
	if got != "hello" {
		t.Errorf("truncateOutput whitespace = %q, want 'hello'", got)
	}
}

func TestTruncateOutput_TrimmedTooLong(t *testing.T) {
	got := truncateOutput("  this is a very long trimmed string  ", 10)
	if len(got) > 13 { // 10 chars + "..."
		t.Errorf("truncateOutput should be at most 13 chars, got %d: %q", len(got), got)
	}
}

// ── Verify with real repo ────────────────────────────────────────────────

func TestVerify_SimpleRepo(t *testing.T) {
	dir := t.TempDir()
	// Init a minimal git repo
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644)
	runGitHygiene(dir, "add", ".gitignore")
	runGitHygiene(dir, "commit", "-m", "add gitignore")

	result, err := Verify(dir, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// Non-Go repo should pass Go checks as true (skipped)
	if !result.GoVetPass {
		t.Error("GoVetPass should be true for non-Go repo")
	}
	if !result.GofmtPass {
		t.Error("GofmtPass should be true for non-Go repo")
	}
	if !result.GoTestPass {
		t.Error("GoTestPass should be true for non-Go repo")
	}
}

func TestVerify_RepoWithHygieneIssues(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Create untracked file → hygiene issue
	os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("orphan"), 0644)

	result, err := Verify(dir, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.HygieneClean {
		t.Error("should not be clean with untracked files")
	}
	if result.HygieneIssues == 0 {
		t.Error("should have hygiene issues")
	}
	if result.HygieneBlocking != 0 {
		t.Fatalf("ordinary untracked file should be advisory, got %d blocking issues", result.HygieneBlocking)
	}
	if !result.Passed {
		t.Errorf("advisory hygiene warning must not fail verification: %s", result.Detail)
	}
}

func TestVerify_InvalidNodeManifestBlocks(t *testing.T) {
	dir := t.TempDir()
	writeOWSTestFile(t, filepath.Join(dir, "package.json"), "{")
	initOWSTestRepo(t, dir)

	result, err := Verify(dir, nil, true)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.StackFailures != 1 || result.Passed {
		t.Fatalf("invalid package manifest must block: %#v", result)
	}
}

func TestVerify_ConfiguredNodeFailureBlocks(t *testing.T) {
	dir := t.TempDir()
	binDir := t.TempDir()
	writeOWSExecutable(t, filepath.Join(binDir, "npx"), "#!/bin/sh\necho typecheck-failed >&2\nexit 9\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	writeOWSTestFile(t, filepath.Join(dir, "package.json"), `{}`)
	writeOWSTestFile(t, filepath.Join(dir, "tsconfig.json"), `{}`)
	initOWSTestRepo(t, dir)

	result, err := Verify(dir, nil, true)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.StackFailures != 1 || result.Passed {
		t.Fatalf("configured Node failure must block: %#v", result)
	}
	if !strings.Contains(result.Detail, "typecheck-failed") {
		t.Fatalf("missing Node failure detail: %s", result.Detail)
	}
}

func initOWSTestRepo(t *testing.T, dir string) {
	t.Helper()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	runGitHygiene(dir, "add", "package.json")
	if _, err := os.Stat(filepath.Join(dir, "tsconfig.json")); err == nil {
		runGitHygiene(dir, "add", "tsconfig.json")
	}
	runGitHygiene(dir, "commit", "-m", "init")
}

func TestVerify_InvalidRepoPath(t *testing.T) {
	result, err := Verify("/nonexistent/path/that/does/not/exist", nil)
	if err != nil {
		t.Fatalf("Verify should not error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Passed {
		t.Error("should not pass for invalid path")
	}
	if result.Detail == "" {
		t.Error("should have detail about the error")
	}
}

func TestVerify_PersistsResults(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	Verify(dir, nil)

	// Verify results were persisted
	resultPath := filepath.Join(dir, ".ovav", "verify", "last_result.json")
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Error("Verify should persist results to .ovav/verify/last_result.json")
	}
}

// ── pruneStaleWorktrees ──────────────────────────────────────────────────

func TestPruneStaleWorktrees_NoExtraWorktrees(t *testing.T) {
	repo := initTestRepo(t)
	count := pruneStaleWorktrees(repo, 0) // 0 duration = everything stale
	if count != 0 {
		t.Errorf("main worktree should be skipped, got %d", count)
	}
}

// ── Rescue edge cases ────────────────────────────────────────────────────

func TestRescue_NonexistentPath(t *testing.T) {
	result, err := Rescue("/nonexistent/path")
	// Should not panic
	if result != nil {
		t.Logf("result from nonexistent path: %+v", result)
	}
	_ = err // may error, that's fine
}

func TestSync_NonexistentPath(t *testing.T) {
	err := Sync("/nonexistent/path")
	if err == nil {
		t.Error("Sync on nonexistent path should error")
	}
}

// ── Verify with Go repo (OVAV-like) ──────────────────────────────────────

func TestVerify_GoRepo(t *testing.T) {
	t.Skip("Integration test: Verify() on real repo takes 4+ min — run manually with -run TestVerify_GoRepo")

	// Find the go-runtime directory
	goroot := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(goroot, "go.mod")); err != nil {
		t.Skip("go-runtime not found, skipping")
	}

	result, err := Verify(goroot, nil)
	if err != nil {
		t.Fatalf("Verify on go-runtime: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// Just log the results — we're testing that it runs without panicking
	t.Logf("Go vet: %v, gofmt: %v, go test: %v, hygiene clean: %v, passed: %v",
		result.GoVetPass, result.GofmtPass, result.GoTestPass, result.HygieneClean, result.Passed)
	t.Logf("Detail: %s", result.Detail)
}
