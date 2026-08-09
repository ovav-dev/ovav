package gitflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupWorktreeRepo creates a test repo with worktrees for testing.
func setupWorktreeRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	run("init")
	run("config", "user.email", "test@ovav.dev")
	run("config", "user.name", "Test User")

	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644)
	run("add", "README.md")
	run("commit", "-m", "initial commit")

	// Rename current branch to main so we can safely create 'develop'
	run("branch", "-m", "main")

	run("checkout", "-b", "develop")
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.0.0"), 0644)
	run("add", "VERSION")
	run("commit", "-m", "docs: add VERSION")

	// Create a worktree
	wtPath := filepath.Join(dir, ".ovav", "worktrees", "feature-x")
	run("worktree", "add", wtPath, "-b", "feature/test-wt")

	return dir
}

// ── parseWorktreeList ──────────────────────────────────────────────────────

func TestParseWorktreeList_Basic(t *testing.T) {
	input := `worktree /path/to/main
HEAD abc1234
branch refs/heads/develop

worktree /path/to/feature
HEAD def5678
branch refs/heads/feature/login

`
	entries := parseWorktreeList(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Worktree != "/path/to/main" {
		t.Errorf("entry[0].Worktree = %q, want /path/to/main", entries[0].Worktree)
	}
	if entries[0].Branch != "refs/heads/develop" {
		t.Errorf("entry[0].Branch = %q, want refs/heads/develop", entries[0].Branch)
	}
	if entries[1].Worktree != "/path/to/feature" {
		t.Errorf("entry[1].Worktree = %q, want /path/to/feature", entries[1].Worktree)
	}
}

func TestParseWorktreeList_NoTrailingBlank(t *testing.T) {
	input := `worktree /a
HEAD aaa
branch refs/heads/feat/a`
	entries := parseWorktreeList(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Worktree != "/a" {
		t.Errorf("got %q, want /a", entries[0].Worktree)
	}
}

func TestParseWorktreeList_Empty(t *testing.T) {
	entries := parseWorktreeList("")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseWorktreeList_IgnoredFields(t *testing.T) {
	input := `worktree /path
HEAD abc1234
branch refs/heads/feat-x
bare
detached
prunable

`
	entries := parseWorktreeList(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].HEAD != "abc1234" {
		t.Errorf("HEAD = %q, want abc1234", entries[0].HEAD)
	}
}

func TestParseWorktreeList_MultipleEntries(t *testing.T) {
	input := `worktree /a
HEAD 111
branch refs/heads/feat/alpha

worktree /b
HEAD 222
branch refs/heads/hotfix/bug

worktree /c
HEAD 333
branch refs/heads/release/v1
`
	entries := parseWorktreeList(input)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[2].Worktree != "/c" {
		t.Errorf("entry[2].Worktree = %q, want /c", entries[2].Worktree)
	}
}

// ── looksLikeWorktreePath ─────────────────────────────────────────────────

func TestLooksLikeWorktreePath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/home/user/work", true},
		{"./relative", true},
		{"../parent", true},
		{".ovav/worktrees/feat", true},
		{"feature/sprint-1", false},
		{"sprint-1.7", false},
		{"", false},
		{"../", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeWorktreePath(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeWorktreePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ── entryToResolved ───────────────────────────────────────────────────────

func TestEntryToResolved(t *testing.T) {
	e := porcelainEntry{
		Worktree: "/worktrees/feat-x",
		HEAD:     "abc123",
		Branch:   "refs/heads/feature/sprint-1",
	}

	result := entryToResolved(e, "/main/repo")
	if result.WorktreePath != "/worktrees/feat-x" {
		t.Errorf("WorktreePath = %q", result.WorktreePath)
	}
	if result.Branch != "feature/sprint-1" {
		t.Errorf("Branch = %q, want feature/sprint-1", result.Branch)
	}
	if result.MainRepoRoot != "/main/repo" {
		t.Errorf("MainRepoRoot = %q", result.MainRepoRoot)
	}
	if !result.IsWorktree {
		t.Error("IsWorktree should be true for non-main path")
	}
}

func TestEntryToResolved_MainRepo(t *testing.T) {
	e := porcelainEntry{
		Worktree: "/main/repo",
		HEAD:     "abc123",
		Branch:   "refs/heads/develop",
	}
	result := entryToResolved(e, "/main/repo")
	if result.IsWorktree {
		t.Error("IsWorktree should be false when worktree == mainRepoRoot")
	}
}

// ── resolveFromBranch ─────────────────────────────────────────────────────

func TestResolveFromBranch_ExactShort(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt", Branch: "refs/heads/feature/x"},
	}
	result, err := resolveFromBranch(entries, "feature/x", "/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "feature/x" {
		t.Errorf("Branch = %q, want feature/x", result.Branch)
	}
}

func TestResolveFromBranch_ExactFullRef(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt", Branch: "refs/heads/feature/y"},
	}
	result, err := resolveFromBranch(entries, "refs/heads/feature/y", "/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "feature/y" {
		t.Errorf("Branch = %q", result.Branch)
	}
}

func TestResolveFromBranch_SuffixMatch(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt", Branch: "refs/heads/feature/sprint-1.7"},
	}
	result, err := resolveFromBranch(entries, "sprint-1.7", "/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "feature/sprint-1.7" {
		t.Errorf("Branch = %q", result.Branch)
	}
}

func TestResolveFromBranch_NotFound(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt", Branch: "refs/heads/feature/x"},
	}
	_, err := resolveFromBranch(entries, "nonexistent", "/main")
	if err == nil {
		t.Fatal("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "no worktree found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── matchBranch ───────────────────────────────────────────────────────────

func TestMatchBranch_Found(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt1", Branch: "refs/heads/feature/a"},
		{Worktree: "/wt2", Branch: "refs/heads/hotfix/b"},
	}
	result, err := matchBranch(entries, "hotfix/b", "/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WorktreePath != "/wt2" {
		t.Errorf("WorktreePath = %q, want /wt2", result.WorktreePath)
	}
}

func TestMatchBranch_NotFound(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/wt1", Branch: "refs/heads/feature/a"},
	}
	_, err := matchBranch(entries, "nope", "/main")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── resolveFromPath ──────────────────────────────────────────────────────

func TestResolveFromPath_Found(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/worktrees/feat", Branch: "refs/heads/feature/login"},
	}
	result, err := resolveFromPath(entries, "/worktrees/feat", "/main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Branch != "feature/login" {
		t.Errorf("Branch = %q", result.Branch)
	}
}

func TestResolveFromPath_NotFound(t *testing.T) {
	entries := []porcelainEntry{
		{Worktree: "/worktrees/feat", Branch: "refs/heads/feature/x"},
	}
	_, err := resolveFromPath(entries, "/nonexistent", "/main")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "not a registered worktree") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── ResolveWorktree integration (with real git worktree) ─────────────────

func TestResolveWorktree_ByBranchHint(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	result, err := ResolveWorktree("feature/test-wt")
	if err != nil {
		t.Fatalf("ResolveWorktree failed: %v", err)
	}
	if result.Branch != "feature/test-wt" {
		t.Errorf("Branch = %q, want feature/test-wt", result.Branch)
	}
	if result.MainRepoRoot == "" {
		t.Error("MainRepoRoot should not be empty")
	}
}

func TestResolveWorktree_ByPath(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	wtPath := filepath.Join(dir, ".ovav", "worktrees", "feature-x")
	result, err := ResolveWorktree(wtPath)
	if err != nil {
		t.Fatalf("ResolveWorktree failed: %v", err)
	}
	if result.Branch != "feature/test-wt" {
		t.Errorf("Branch = %q, want feature/test-wt", result.Branch)
	}
}

func TestResolveWorktree_NotFound(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	_, err := ResolveWorktree("nonexistent/branch")
	if err == nil {
		t.Fatal("expected error for non-existent branch")
	}
}

func TestResolveWorktree_EmptyHint(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	_, _ = ResolveWorktree("")
	// Just ensure no panic — actual result depends on CWD state
}

// ── findMainRepoRoot ─────────────────────────────────────────────────────

func TestFindMainRepoRoot_FromMain(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	root, err := findMainRepoRoot(dir)
	if err != nil {
		t.Fatalf("findMainRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestFindMainRepoRoot_FromWorktree(t *testing.T) {
	dir := setupWorktreeRepo(t)
	wtPath := filepath.Join(dir, ".ovav", "worktrees", "feature-x")

	root, err := findMainRepoRoot(wtPath)
	if err != nil {
		t.Fatalf("findMainRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root for worktree")
	}
}

func TestFindMainRepoRoot_InvalidDir(t *testing.T) {
	_, err := findMainRepoRoot("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

// ── gitToplevel ──────────────────────────────────────────────────────────

func TestGitToplevel_Valid(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Run()

	result, err := gitToplevel(dir)
	if err != nil {
		t.Fatalf("gitToplevel: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty toplevel")
	}
}

func TestGitToplevel_InvalidDir(t *testing.T) {
	// gitToplevel falls back to dir on error
	result, err := gitToplevel("/nonexistent")
	// Should not error — falls back to dir
	_ = err
	if result != "/nonexistent" {
		t.Errorf("expected fallback to /nonexistent, got %q", result)
	}
}

// ── listWorktreeEntries (integration) ────────────────────────────────────

func TestListWorktreeEntries(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	entries, err := listWorktreeEntries()
	if err != nil {
		t.Fatalf("listWorktreeEntries: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("expected at least 2 worktree entries, got %d", len(entries))
	}
}

// ── findMainRepoRootFromEntries (integration) ────────────────────────────

func TestFindMainRepoRootFromEntries(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	entries, err := listWorktreeEntries()
	if err != nil {
		t.Fatalf("listWorktreeEntries: %v", err)
	}
	root := findMainRepoRootFromEntries(entries)
	if root == "" {
		t.Error("expected non-empty root from entries")
	}
	_ = dir
}

func TestFindMainRepoRootFromEntries_Empty(t *testing.T) {
	root := findMainRepoRootFromEntries(nil)
	if root != "" {
		t.Errorf("expected empty root for nil entries, got %q", root)
	}
}

// ── resolveFromHEAD (integration) ────────────────────────────────────────

func TestResolveFromHEAD_MainBranch(t *testing.T) {
	dir := setupWorktreeRepo(t)
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	entries, err := listWorktreeEntries()
	if err != nil {
		t.Fatalf("listWorktreeEntries: %v", err)
	}
	result, err := resolveFromHEAD(entries, dir)
	// If CWD is the main repo on develop, this should match
	if err != nil && !strings.Contains(err.Error(), "no worktree found") {
		t.Fatalf("resolveFromHEAD: %v", err)
	}
	if result != nil && result.Branch != "develop" {
		t.Errorf("expected develop branch, got %q", result.Branch)
	}
}

// ── detectCommitTypeFromFiles: cover missing cases ───────────────────────

func TestDetectCommitTypeFromFiles_Validators(t *testing.T) {
	repo, _ := setupTestRepo(t)
	os.MkdirAll(filepath.Join(repo, "validators"), 0755)
	os.WriteFile(filepath.Join(repo, "validators", "auth.go"), []byte("package v"), 0644)
	runGitCmd(t, repo, "add", "validators/auth.go")

	result := detectCommitTypeFromFiles(repo)
	if result != "feat(validators):" {
		t.Errorf("got %q, want feat(validators):", result)
	}
}

func TestDetectCommitTypeFromFiles_Install(t *testing.T) {
	repo, _ := setupTestRepo(t)
	os.WriteFile(filepath.Join(repo, "install.sh"), []byte("install"), 0644)
	runGitCmd(t, repo, "add", "install.sh")

	result := detectCommitTypeFromFiles(repo)
	if result != "feat(install):" {
		t.Errorf("got %q, want feat(install):", result)
	}
}

func TestDetectCommitTypeFromFiles_Git(t *testing.T) {
	repo, _ := setupTestRepo(t)
	os.WriteFile(filepath.Join(repo, "git_helper.go"), []byte("package main"), 0644)
	runGitCmd(t, repo, "add", "git_helper.go")

	result := detectCommitTypeFromFiles(repo)
	if result != "feat(git):" {
		t.Errorf("got %q, want feat(git):", result)
	}
}
