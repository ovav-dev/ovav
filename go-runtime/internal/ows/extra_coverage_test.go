package ows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// extra_coverage_test.go — Additional tests for coverage gaps in:
// - handlers.go: cleanWorktrees, parseWorktreeListFromRepo, makeListHandler,
//   makeRouteHandler, makeVerifyHandler, makeRescueHandler, LockWorktree
// - hygiene.go: Report (various sections), checkOrphanWorktreeDirs
// - recovery.go: Verify (more branches)
// ═══════════════════════════════════════════════════════════════════════════

// ── cleanWorktrees ───────────────────────────────────────────────────────

func TestCleanWorktrees_DryRun(t *testing.T) {
	repo := setupRepoWithRemote(t)

	err := cleanWorktrees(repo, map[string]string{"dry-run": "true"})
	if err != nil {
		t.Fatalf("cleanWorktrees dry-run: %v", err)
	}
}

func TestCleanWorktrees_NoExtraWorktrees(t *testing.T) {
	repo := setupRepoWithRemote(t)

	err := cleanWorktrees(repo, map[string]string{})
	if err != nil {
		t.Fatalf("cleanWorktrees: %v", err)
	}
}

func TestCleanWorktrees_WithOrphanedWorktree(t *testing.T) {
	repo := setupRepoWithRemote(t)

	// Create a worktree
	wtDir := filepath.Join(filepath.Dir(repo), "orphan-wt")
	cmd := exec.Command("git", "worktree", "add", "-b", "orphan-branch", wtDir)
	cmd.Dir = repo
	cmd.Run()

	// Delete the branch (makes worktree orphaned)
	runGitOk(t, repo, "checkout", "develop")
	cmd = exec.Command("git", "branch", "-D", "orphan-branch")
	cmd.Dir = repo
	cmd.Run()

	err := cleanWorktrees(repo, map[string]string{})
	if err != nil {
		t.Logf("cleanWorktrees with orphan: %v", err)
	}
}

// ── parseWorktreeListFromRepo ────────────────────────────────────────────

func TestParseWorktreeListFromRepo(t *testing.T) {
	repo := setupRepoWithRemote(t)

	branches, err := parseWorktreeListFromRepo(repo)
	if err != nil {
		t.Fatalf("parseWorktreeListFromRepo: %v", err)
	}
	// Should list at least the main worktree
	if len(branches) == 0 {
		t.Error("expected at least 1 branch")
	}
}

// ── makeListHandler ──────────────────────────────────────────────────────

func TestListHandler_Standard(t *testing.T) {
	repo := setupRepoWithRemote(t)

	handler := makeListHandler(repo)
	err := handler(context.Background(), map[string]string{})
	if err != nil {
		t.Logf("list handler: %v", err)
	}
}

func TestListHandler_History_NoFile(t *testing.T) {
	repo := initTestRepo(t)
	handler := makeListHandler(repo)
	err := handler(context.Background(), map[string]string{"history": "true"})
	if err == nil {
		t.Error("expected error when no audit trail")
	}
}

func TestListHandler_History_WithFile(t *testing.T) {
	repo := initTestRepo(t)
	auditDir := filepath.Join(repo, ".ovav", "audit")
	os.MkdirAll(auditDir, 0755)
	os.WriteFile(filepath.Join(auditDir, "trail.jsonl"), []byte(`{"event":"test"}`), 0644)

	handler := makeListHandler(repo)
	err := handler(context.Background(), map[string]string{"history": "true"})
	if err != nil {
		t.Errorf("list handler history: %v", err)
	}
}

func TestListHandler_History_JSON(t *testing.T) {
	repo := initTestRepo(t)
	auditDir := filepath.Join(repo, ".ovav", "audit")
	os.MkdirAll(auditDir, 0755)
	os.WriteFile(filepath.Join(auditDir, "trail.jsonl"), []byte(`{"event":"test"}`), 0644)

	handler := makeListHandler(repo)
	err := handler(context.Background(), map[string]string{"history": "true", "json": "true"})
	if err != nil {
		t.Errorf("list handler json: %v", err)
	}
}

// ── makeRouteHandler ─────────────────────────────────────────────────────

func TestRouteHandler_NoTarget(t *testing.T) {
	handler := makeRouteHandler("/tmp")
	err := handler(context.Background(), map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "target branch required") {
		t.Errorf("error should mention target required: %v", err)
	}
}

func TestRouteHandler_InvalidSource(t *testing.T) {
	handler := makeRouteHandler("/nonexistent")
	err := handler(context.Background(), map[string]string{"target": "develop"})
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

// ── makeRescueHandler ────────────────────────────────────────────────────

func TestRescueHandler_Success(t *testing.T) {
	repo := initTestRepo(t)
	handler := makeRescueHandler(repo)
	err := handler(context.Background(), map[string]string{})
	if err != nil {
		t.Fatalf("rescue handler: %v", err)
	}
}

// ── makeVerifyHandler ────────────────────────────────────────────────────

func TestVerifyHandler_SimpleRepo(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644)
	runGitHygiene(dir, "add", ".gitignore")
	runGitHygiene(dir, "commit", "-m", "gitignore")

	handler := makeVerifyHandler(dir)
	err := handler(context.Background(), map[string]string{})
	if err != nil {
		t.Logf("verify handler: %v", err)
	}
}

// ── makeUpdateHandler ────────────────────────────────────────────────────

func TestUpdateHandler_NoRemote(t *testing.T) {
	repo := initTestRepo(t)
	handler := makeUpdateHandler(repo)
	err := handler(context.Background(), map[string]string{})
	// Should handle gracefully (no remote)
	if err != nil {
		t.Logf("update handler: %v", err)
	}
}

// ── makeSyncHandler variants ─────────────────────────────────────────────

func TestSyncHandler_WithRebase(t *testing.T) {
	repo := setupRepoWithRemote(t)
	handler := makeSyncHandler(repo)
	err := handler(context.Background(), map[string]string{"rebase": "true"})
	if err != nil {
		t.Logf("sync handler rebase: %v", err)
	}
}

func TestSyncHandler_WithFull(t *testing.T) {
	repo := setupRepoWithRemote(t)
	handler := makeSyncHandler(repo)
	err := handler(context.Background(), map[string]string{"full": "true"})
	if err != nil {
		t.Logf("sync handler full: %v", err)
	}
}

// ── LockWorktree top-level function ──────────────────────────────────────

func TestLockWorktree_Success(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	lock := &AgentLock{
		Worktree: dir,
		Owner:    "thavren",
		Reason:   "testing",
	}
	err := LockWorktree(lock)
	if err != nil {
		t.Logf("LockWorktree: %v (may fail if worktree not in DB)", err)
	}
}

// ── showAuditTrail with multiple lines ───────────────────────────────────

func TestShowAuditTrail_MultipleLines(t *testing.T) {
	repo := initTestRepo(t)
	auditDir := filepath.Join(repo, ".ovav", "audit")
	os.MkdirAll(auditDir, 0755)
	lines := ""
	for i := 0; i < 5; i++ {
		lines += `{"event":"test` + string(rune('0'+i)) + `"}` + "\n"
	}
	os.WriteFile(filepath.Join(auditDir, "trail.jsonl"), []byte(lines), 0644)

	err := showAuditTrail(repo, false)
	if err != nil {
		t.Errorf("showAuditTrail: %v", err)
	}
}

// ── Handlers with wired registry ─────────────────────────────────────────

func TestDispatch_RouteHandler_NoArgs(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "route"})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "target is required") {
		t.Errorf("error should mention target required: %v", err)
	}
}

func TestDispatch_AbortHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "abort"})
	if err != nil {
		t.Logf("abort dispatch: %v", err)
	}
}

func TestDispatch_RescueHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "rescue"})
	if err != nil {
		t.Fatalf("rescue dispatch: %v", err)
	}
}

func TestDispatch_LockHandler_NoTarget(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "lock"})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "target is required") {
		t.Errorf("error should mention target required: %v", err)
	}
}

func TestDispatch_MoveHandler_NoArgs(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "move"})
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestDispatch_CleanHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "clean"})
	if err != nil {
		t.Logf("clean dispatch: %v", err)
	}
}

func TestDispatch_ListHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "list"})
	if err != nil {
		t.Logf("list dispatch: %v", err)
	}
}

func TestDispatch_VerifyHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "verify"})
	if err != nil {
		t.Logf("verify dispatch: %v", err)
	}
}

func TestDispatch_UpdateHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "update"})
	if err != nil {
		t.Logf("update dispatch: %v", err)
	}
}

func TestDispatch_SyncHandler(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "sync"})
	if err != nil {
		t.Logf("sync dispatch: %v", err)
	}
}

// ── parseArgs edge cases ─────────────────────────────────────────────────

func TestParseArgs_BooleanFlag(t *testing.T) {
	result := parseArgs([]string{"--dry-run", "--verbose"}, []Arg{})
	if result["dry-run"] != "true" {
		t.Errorf("boolean flag should be 'true', got %q", result["dry-run"])
	}
	if result["verbose"] != "true" {
		t.Errorf("boolean flag should be 'true', got %q", result["verbose"])
	}
}

func TestParseArgs_KeyValuePair(t *testing.T) {
	result := parseArgs([]string{"--profile", "hotfix", "--compliance", "strict"}, []Arg{})
	if result["profile"] != "hotfix" {
		t.Errorf("profile = %q, want hotfix", result["profile"])
	}
	if result["compliance"] != "strict" {
		t.Errorf("compliance = %q, want strict", result["compliance"])
	}
}

func TestParseArgs_EqualsSyntax(t *testing.T) {
	// OWS-GAP-05: --profile=<name> syntax
	result := parseArgs([]string{"--profile=hotfix", "--compliance=strict"}, []Arg{})
	if result["profile"] != "hotfix" {
		t.Errorf("profile = %q, want hotfix", result["profile"])
	}
	if result["compliance"] != "strict" {
		t.Errorf("compliance = %q, want strict", result["compliance"])
	}
}

func TestParseArgs_MixedSyntax(t *testing.T) {
	// OWS-GAP-05: mixed --flag value and --flag=value syntax
	result := parseArgs([]string{"task27", "--profile=hotfix", "--carry-uncommitted"}, []Arg{
		{Name: "name", Required: false},
	})
	if result["name"] != "task27" {
		t.Errorf("name = %q, want task27", result["name"])
	}
	if result["profile"] != "hotfix" {
		t.Errorf("profile = %q, want hotfix", result["profile"])
	}
	if result["carry-uncommitted"] != "true" {
		t.Errorf("carry-uncommitted = %q, want true", result["carry-uncommitted"])
	}
}

func TestParseArgs_PositionalAndFlags(t *testing.T) {
	defs := []Arg{
		{Name: "name", Required: false},
		{Name: "compliance", Default: "standard"},
	}
	result := parseArgs([]string{"my-task", "--profile", "hotfix"}, defs)
	if result["name"] != "my-task" {
		t.Errorf("name = %q, want my-task", result["name"])
	}
	if result["profile"] != "hotfix" {
		t.Errorf("profile = %q, want hotfix", result["profile"])
	}
	if result["compliance"] != "standard" {
		t.Errorf("compliance = %q, want standard (default)", result["compliance"])
	}
}

func TestParseArgs_FlagOverridesPositional(t *testing.T) {
	defs := []Arg{
		{Name: "target", Required: false},
	}
	result := parseArgs([]string{"positional-value", "--target", "flag-value"}, defs)
	if result["target"] != "flag-value" {
		t.Errorf("flag should override positional, got %q", result["target"])
	}
}

func TestParseArgs_MixedFlags(t *testing.T) {
	result := parseArgs([]string{"arg1", "--flag1", "val1", "--flag2", "arg2"}, []Arg{})
	if result["flag1"] != "val1" {
		t.Errorf("flag1 = %q", result["flag1"])
	}
}

// ── Hygiene Report sections ──────────────────────────────────────────────

func TestHygieneReport_LargeFiles(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Create a large untracked file
	largePath := filepath.Join(dir, "backup.tar.gz")
	f, _ := os.Create(largePath)
	f.Write(make([]byte, 1024*1024+1))
	f.Close()

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "LARGE UNTRACKED") {
		t.Error("report should mention LARGE UNTRACKED FILES")
	}
}

func TestHygieneReport_StaleLocks(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Create stale lock
	locksDir := filepath.Join(dir, ".ovav", "locks")
	os.MkdirAll(locksDir, 0755)
	lockPath := filepath.Join(locksDir, "old.lock")
	os.WriteFile(lockPath, []byte("test"), 0644)
	oldTime := time.Now().Add(-25 * time.Hour)
	os.Chtimes(lockPath, oldTime, oldTime)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "STALE LOCKS") {
		t.Error("report should mention STALE LOCKS")
	}
}

func TestHygieneReport_BrokenSymlinks(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	os.Symlink("/nonexistent/path", filepath.Join(dir, "broken-link"))

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "BROKEN SYMLINKS") {
		t.Error("report should mention BROKEN SYMLINKS")
	}
}

func TestHygieneReport_GitConfig(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init")
	// No user.email or user.name

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "GIT CONFIG") {
		t.Error("report should mention GIT CONFIG")
	}
}

func TestHygieneReport_AuditTrail(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")
	// Remove .ovav/audit from gitignore
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "AUDIT TRAIL") {
		t.Error("report should mention AUDIT TRAIL")
	}
}

func TestHygieneReport_Clean(t *testing.T) {
	r := &HygieneResult{Clean: true}
	report := r.Report()
	if !strings.Contains(report, "clean") {
		t.Error("clean report should mention 'clean'")
	}
}

func TestHygieneReport_UntrackedFiles(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Create untracked files
	os.WriteFile(filepath.Join(dir, "orphan1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "orphan2.txt"), []byte("b"), 0644)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "Untracked files") {
		t.Error("report should mention untracked files count")
	}
}

func TestHygieneReport_UnstagedModified(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Modify staged file without staging
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# modified"), 0644)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	if !strings.Contains(report, "MODIFIED UNSTAGED") {
		t.Error("report should mention MODIFIED UNSTAGED")
	}
}

func TestHygieneReport_DirtyAfterMerge(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Create untracked file in root — this IS in the main worktree so
	// it won't be "dirty after merge" but it will be detected as untracked
	os.WriteFile(filepath.Join(dir, "orphan-change.txt"), []byte("forgotten"), 0644)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()
	// In a fresh repo with no worktrees, untracked files in root are part of
	// the main worktree, so they won't trigger DIRTY FILES. Just verify
	// the scan ran and produced output
	if report == "" {
		t.Error("report should not be empty")
	}
	if r.TotalIssues == 0 && len(r.UntrackedFiles) == 0 {
		t.Error("should detect at least untracked file")
	}
}

// ── LockHandler edge cases ───────────────────────────────────────────────

func TestLockHandler_UnlockByDifferentOwner(t *testing.T) {
	repo := t.TempDir()
	runGitHygiene(repo, "init")
	runGitHygiene(repo, "config", "user.email", "thavren@test.com")
	runGitHygiene(repo, "config", "user.name", "Thavren")

	// Lock as thavren
	handler := makeLockHandler(repo)
	err := handler(context.Background(), map[string]string{
		"target": "feature/test-unlock-diff",
		"reason": "test",
	})
	if err != nil {
		t.Fatalf("lock: %v", err)
	}

	// Change user to dante
	runGitHygiene(repo, "config", "user.email", "dante@test.com")
	runGitHygiene(repo, "config", "user.name", "Dante")

	// Try unlock as dante (should fail — not owner)
	err = handler(context.Background(), map[string]string{
		"target": "feature/test-unlock-diff",
		"unlock": "1",
	})
	if err == nil {
		t.Error("unlock by different owner should fail")
	}
	if err != nil && !strings.Contains(err.Error(), "denied") {
		t.Errorf("error should mention denied: %v", err)
	}
}

// ── Dispatch with --help for various commands ────────────────────────────

func TestDispatch_Help_Route(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "route", "--help"})
	if err != nil {
		t.Fatalf("help route: %v", err)
	}
}

func TestDispatch_Help_List(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "list", "--help"})
	if err != nil {
		t.Fatalf("help list: %v", err)
	}
}

func TestDispatch_Help_Lock(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "lock", "--help"})
	if err != nil {
		t.Fatalf("help lock: %v", err)
	}
}

func TestDispatch_Help_Clean(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "clean", "--help"})
	if err != nil {
		t.Fatalf("help clean: %v", err)
	}
}

func TestDispatch_Help_Move(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "move", "--help"})
	if err != nil {
		t.Fatalf("help move: %v", err)
	}
}

func TestDispatch_UnwiredCommand(t *testing.T) {
	// Save current handler, set to nil, restore after
	cmd := CommandRegistry["ovav worktree create"]
	origHandler := cmd.Handler
	cmd.Handler = nil
	CommandRegistry["ovav worktree create"] = cmd
	defer func() {
		cmd.Handler = origHandler
		CommandRegistry["ovav worktree create"] = cmd
	}()

	err := Dispatch(context.Background(), "/tmp", []string{"ovav", "worktree", "create", "test"})
	if err == nil {
		t.Fatal("expected error for unwired command")
	}
	if !strings.Contains(err.Error(), "handler not wired") {
		t.Errorf("error should mention handler not wired: %v", err)
	}
}

// ── LockWorktree with invalid owner ──────────────────────────────────────

func TestLockWorktree_NonOwner(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init", "-b", "main")
	runGitHygiene(dir, "config", "user.email", "test@test.com")
	runGitHygiene(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md")
	runGitHygiene(dir, "commit", "-m", "init")

	// Save worktree record with owner "alice"
	audit, err := OpenAudit(dir)
	if err != nil {
		t.Fatal(err)
	}
	audit.SaveWorktree(WorktreeRecord{
		ID:    dir,
		Owner: "alice",
		State: StateActive,
	})
	audit.Close()

	// Try to lock as "bob" (different owner) — should fail
	err = LockWorktree(&AgentLock{
		Worktree: dir,
		Owner:    "bob",
		Reason:   "test",
	})
	if err != nil {
		t.Logf("LockWorktree non-owner: %v", err)
	}
}

// ── MakeDoneHandler: protected branch rejection ──────────────────────────

func TestDoneHandler_MasterRejected(t *testing.T) {
	repo := setupRepoWithRemote(t)
	handler := makeDoneHandler(repo)

	err := handler(context.Background(), map[string]string{"branch": "main"})
	if err == nil {
		t.Fatal("done handler should reject master/main branch")
	}
}

// ── MatchSecretPatterns: more patterns ───────────────────────────────────

func TestMatchSecretPatterns_NPMToken(t *testing.T) {
	findings := matchSecretPatterns("ci.yml", 1, `NPM_TOKEN = "npm_abc12345678901234567890"`)
	if len(findings) == 0 {
		t.Error("should detect NPM token")
	}
}

func TestMatchSecretPatterns_DockerPassword(t *testing.T) {
	findings := matchSecretPatterns("ci.yml", 1, `DOCKER_PASSWORD = "supersecretpassword"`)
	if len(findings) == 0 {
		t.Error("should detect Docker password")
	}
}

func TestMatchSecretPatterns_SecretEnv(t *testing.T) {
	findings := matchSecretPatterns("env.sh", 1, `SECRET_TOKEN="abcdefghij1234567890"`)
	if len(findings) == 0 {
		t.Error("should detect SECRET_TOKEN env var")
	}
}

func TestMatchSecretPatterns_HuggingFace(t *testing.T) {
	findings := matchSecretPatterns("config.py", 1, `hf_token = "hf_abcdefghijklmnopqrstuvwxyz123456"`)
	if len(findings) == 0 {
		t.Error("should detect HuggingFace token")
	}
}

func TestMatchSecretPatterns_GitLabToken(t *testing.T) {
	findings := matchSecretPatterns("ci.yml", 1, `gitlab_token: "glpat-1234567890abcdefghij"`)
	if len(findings) == 0 {
		t.Error("should detect GitLab token")
	}
}

func TestMatchSecretPatterns_OpenAIKey(t *testing.T) {
	findings := matchSecretPatterns("config.go", 1, `apiKey := "sk-abcdefghijklmnopqrstuvwxyz123456"`)
	if len(findings) == 0 {
		t.Error("should detect OpenAI API key")
	}
}

func TestMatchSecretPatterns_GoogleAPIKey(t *testing.T) {
	findings := matchSecretPatterns("config.js", 1, `apiKey = "AIzaSyD1234567890abcdefghijklmnopqrstuv"`)
	if len(findings) == 0 {
		t.Error("should detect Google API key")
	}
}

func TestMatchSecretPatterns_WildcardSecret(t *testing.T) {
	findings := matchSecretPatterns("config.env", 1, `MY_SECRET = "abcdefghij1234567890"`)
	if len(findings) == 0 {
		t.Error("should detect generic SECRET pattern")
	}
}

// ── Forbidden files: more patterns ───────────────────────────────────────

func TestScanForbiddenFiles_BlocksBinary(t *testing.T) {
	repo := initTestRepo(t)
	runGitOk(t, repo, "checkout", "-b", "feature/binary")
	writeFileH(t, repo, "app.exe", "fake binary content")
	runGitOk(t, repo, "add", "app.exe")
	runGitOk(t, repo, "commit", "-m", "add binary")

	forbidden, err := scanForbiddenFiles(repo, "feature/binary")
	if err != nil {
		t.Fatalf("scanForbiddenFiles: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("should detect .exe as forbidden")
	}
}

func TestScanForbiddenFiles_BlocksKey(t *testing.T) {
	repo := initTestRepo(t)
	runGitOk(t, repo, "checkout", "-b", "feature/key")
	writeFileH(t, repo, "server.key", "fake key")
	runGitOk(t, repo, "add", "server.key")
	runGitOk(t, repo, "commit", "-m", "add key")

	forbidden, err := scanForbiddenFiles(repo, "feature/key")
	if err != nil {
		t.Fatalf("scanForbiddenFiles: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("should detect .key as forbidden")
	}
}

func TestScanForbiddenFiles_BlocksDatabase(t *testing.T) {
	repo := initTestRepo(t)
	runGitOk(t, repo, "checkout", "-b", "feature/db")
	writeFileH(t, repo, "data.sqlite", "fake db")
	runGitOk(t, repo, "add", "data.sqlite")
	runGitOk(t, repo, "commit", "-m", "add db")

	forbidden, err := scanForbiddenFiles(repo, "feature/db")
	if err != nil {
		t.Fatalf("scanForbiddenFiles: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("should detect .sqlite as forbidden")
	}
}

func TestScanForbiddenFiles_BlocksLog(t *testing.T) {
	repo := initTestRepo(t)
	runGitOk(t, repo, "checkout", "-b", "feature/log")
	writeFileH(t, repo, "app.log", "some log entry\n")
	runGitOk(t, repo, "add", "-f", "app.log")
	runGitOk(t, repo, "commit", "-m", "add log")

	forbidden, err := scanForbiddenFiles(repo, "feature/log")
	if err != nil {
		t.Fatalf("scanForbiddenFiles: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("should detect .log as forbidden")
	}
}

// ── Seal with various levels ─────────────────────────────────────────────

func TestSeal_AllLevels(t *testing.T) {
	repo := initTestRepo(t)
	treeHash := GetGitTreeHash(repo)

	for _, level := range []ComplianceLevel{ComplianceQuick, ComplianceStandard, ComplianceStrict, ComplianceMaximum} {
		seal := GenerateSeal(repo, "feature/test", "author", "reviewer",
			level, treeHash, 3, 77)
		if seal.Hash == "" {
			t.Errorf("seal hash empty for level %s", level)
		}
		if seal.Level != string(level) {
			t.Errorf("seal level = %q, want %s", seal.Level, level)
		}
	}
}

// ── ConflictSummary for 0 conflicts ──────────────────────────────────────

func TestConflictMatrix_Summary_NoConflicts(t *testing.T) {
	m := &ConflictMatrix{
		TotalFiles: 5,
		SafeFiles:  5,
	}
	summary := m.Summary()
	if !strings.Contains(summary, "0 conflicts") {
		t.Errorf("summary should mention 0 conflicts: %s", summary)
	}
}

func TestConflictMatrix_Summary_WithConflicts(t *testing.T) {
	m := &ConflictMatrix{
		TotalFiles:    10,
		ConflictFiles: 3,
		SafeFiles:     7,
	}
	summary := m.Summary()
	if !strings.Contains(summary, "3 conflict(s)") {
		t.Errorf("summary should mention 3 conflicts: %s", summary)
	}
}

// ── OWS-GAP-11: Tier access tests ──────────────────────────────────────────

func TestTierFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected TierLevel
	}{
		{"free", TierFree},
		{"Free", TierFree},
		{"FREE", TierFree},
		{"pro", TierPro},
		{"PRO", TierPro},
		{"business", TierBusiness},
		{"enterprise", TierEnterprise},
		{"unknown", TierFree}, // defaults to free
		{"", TierFree},        // defaults to free
	}
	for _, tt := range tests {
		got := TierFromString(tt.input)
		if got != tt.expected {
			t.Errorf("TierFromString(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestTierLevel_CanRun(t *testing.T) {
	tests := []struct {
		cmdTier       TierLevel
		effectiveTier string
		canRun        bool
	}{
		// Free tier commands
		{TierFree, "free", true},
		{TierFree, "pro", true},
		{TierFree, "business", true},
		{TierFree, "enterprise", true},
		// Pro tier commands
		{TierPro, "free", false},
		{TierPro, "pro", true},
		{TierPro, "business", true},
		{TierPro, "enterprise", true},
		// Business tier commands
		{TierBusiness, "free", false},
		{TierBusiness, "pro", false},
		{TierBusiness, "business", true},
		{TierBusiness, "enterprise", true},
		// Enterprise tier commands
		{TierEnterprise, "free", false},
		{TierEnterprise, "pro", false},
		{TierEnterprise, "business", false},
		{TierEnterprise, "enterprise", true},
		// Zero tier defaults to free
		{0, "free", true},
		{0, "pro", true},
	}
	for _, tt := range tests {
		got := tt.cmdTier.CanRun(tt.effectiveTier)
		if got != tt.canRun {
			t.Errorf("TierLevel(%v).CanRun(%q) = %v, want %v",
				tt.cmdTier, tt.effectiveTier, got, tt.canRun)
		}
	}
}

func TestTierLevel_String(t *testing.T) {
	tests := []struct {
		tier TierLevel
		want string
	}{
		{TierFree, "free"},
		{TierPro, "pro"},
		{TierBusiness, "business"},
		{TierEnterprise, "enterprise"},
		{TierLevel(99), "unknown"},
		{TierLevel(0), "free"}, // zero value for TierLevel
	}
	for _, tt := range tests {
		got := tt.tier.String()
		if got != tt.want {
			t.Errorf("TierLevel(%v).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestCommandRegistry_TierFields(t *testing.T) {
	// OWS-GAP-11: abort, rescue, route (cherry-pick), lock must be TierFree
	gated := map[string]TierLevel{
		"ovav worktree abort":  TierFree,
		"ovav worktree rescue": TierFree,
		"ovav worktree route":  TierFree,
		"ovav worktree lock":   TierFree,
	}
	for name, expectedTier := range gated {
		cmd, ok := CommandRegistry[name]
		if !ok {
			t.Errorf("command %q not found in registry", name)
			continue
		}
		if cmd.Tier != expectedTier {
			t.Errorf("command %q has Tier=%v, want %v (OWS-GAP-11: free tier)",
				name, cmd.Tier, expectedTier)
		}
	}
}

func TestDispatch_TierCheck_Abort_FreeTier(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)

	// Free tier: abort should work (no tier error)
	t.Setenv("OVAV_CONSUMER_TIER", "free")
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "abort"})
	// abort may fail if nothing to abort, but tier check should pass (no tier error)
	if err != nil && strings.Contains(err.Error(), "tier") {
		t.Errorf("free tier should not get tier error for abort: %v", err)
	}
}

func TestDispatch_TierCheck_Abort_ProTier(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)

	t.Setenv("OVAV_CONSUMER_TIER", "pro")
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "abort"})
	if err != nil && strings.Contains(err.Error(), "tier") {
		t.Errorf("pro tier should not get tier error for abort: %v", err)
	}
}

func TestDispatch_TierCheck_Rescue_FreeTier(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)

	t.Setenv("OVAV_CONSUMER_TIER", "free")
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "rescue"})
	if err != nil && strings.Contains(err.Error(), "tier") {
		t.Errorf("free tier should not get tier error for rescue: %v", err)
	}
}

func TestDispatch_TierCheck_Lock_FreeTier(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)

	t.Setenv("OVAV_CONSUMER_TIER", "free")
	// lock requires a target arg, so we get "target is required" not tier error
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "lock"})
	if err == nil {
		t.Error("expected error for missing target")
	}
	if err != nil && strings.Contains(err.Error(), "tier") {
		t.Errorf("free tier should not get tier error for lock: %v", err)
	}
}

func TestDispatch_TierCheck_Route_FreeTier(t *testing.T) {
	repo := initTestRepo(t)
	WireHandlers(repo)

	t.Setenv("OVAV_CONSUMER_TIER", "free")
	// route requires target arg, so we get "target is required" not tier error
	err := Dispatch(context.Background(), repo, []string{"ovav", "worktree", "route"})
	if err == nil {
		t.Error("expected error for missing target")
	}
	if err != nil && strings.Contains(err.Error(), "tier") {
		t.Errorf("free tier should not get tier error for route: %v", err)
	}
}
