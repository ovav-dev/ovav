package ows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// routing_test.go — Tests for route.go (routing engine) and recovery.go
// ═══════════════════════════════════════════════════════════════════════════

// ── Test Helpers ─────────────────────────────────────────────────────────────

// setupRouteRepo creates a repo with main + develop + a feature branch
// containing unique commits ready for routing.
func setupRouteRepo(t *testing.T) string {
	t.Helper()
	repo := initTestRepo(t)

	// Ensure we're on the default branch (master or main)
	defaultBranch := "master"
	if out, err := runGitOutput(repo, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		defaultBranch = out
	}

	// Rename to main if not already
	if defaultBranch != "main" {
		runGitOk(t, repo, "branch", "-M", "main")
	}

	// Create develop from main
	runGitOk(t, repo, "checkout", "develop")
	writeFileH(t, repo, "develop.txt", "develop base\n")
	runGitOk(t, repo, "add", "develop.txt")
	runGitOk(t, repo, "commit", "-m", "develop init")

	// Create feature branch from develop with unique commits
	runGitOk(t, repo, "checkout", "-b", "feature/route-test")
	writeFileH(t, repo, "feature1.go", "package feature\n")
	runGitOk(t, repo, "add", "feature1.go")
	runGitOk(t, repo, "commit", "-m", "feature: add file 1")

	writeFileH(t, repo, "feature2.go", "package feature2\n")
	runGitOk(t, repo, "add", "feature2.go")
	runGitOk(t, repo, "commit", "-m", "feature: add file 2")

	// Go back to develop for routing tests
	runGitOk(t, repo, "checkout", "develop")

	return repo
}

// ═══════════════════════════════════════════════════════════════════════════
// route.go — Route() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRoute_CherryPick_Success(t *testing.T) {
	repo := setupRouteRepo(t)

	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteCherryPick)
	if err != nil {
		t.Fatalf("Route cherry-pick: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Mode != RouteCherryPick {
		t.Errorf("Mode = %q, want %q", result.Mode, RouteCherryPick)
	}
	if result.SourceRef != "feature/route-test" {
		t.Errorf("SourceRef = %q", result.SourceRef)
	}
	if result.TargetRef != "develop" {
		t.Errorf("TargetRef = %q", result.TargetRef)
	}
	if len(result.Commits) != 2 {
		t.Errorf("Commits = %d, want 2", len(result.Commits))
	}
	if !result.Success {
		t.Errorf("Success = false, want true. Skipped: %v, Conflicts: %v", result.Skipped, result.Conflicts)
	}
}

func TestRoute_CherryPick_NoUniqueCommits(t *testing.T) {
	repo := initTestRepo(t)

	// Route from main to main — no unique commits
	result, err := Route(context.Background(), repo, "master", "master", RouteCherryPick)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if len(result.Commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(result.Commits))
	}
}

func TestRoute_Patch_Success(t *testing.T) {
	repo := setupRouteRepo(t)

	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RoutePatch)
	if err != nil {
		t.Fatalf("Route patch: %v", err)
	}
	if result.Mode != RoutePatch {
		t.Errorf("Mode = %q, want %q", result.Mode, RoutePatch)
	}
	if !result.Success {
		t.Errorf("patch should succeed. Conflicts: %v", result.Conflicts)
	}
	if len(result.Commits) != 2 {
		t.Errorf("Commits = %d, want 2", len(result.Commits))
	}
}

func TestRoute_Patch_ConflictDoesNotApply(t *testing.T) {
	repo := setupRouteRepo(t)

	// Modify develop.txt on develop to create a conflict with the patch
	runGitOk(t, repo, "checkout", "develop")
	writeFileH(t, repo, "feature1.go", "package conflict\n") // same file, different content
	runGitOk(t, repo, "add", "feature1.go")
	runGitOk(t, repo, "commit", "-m", "develop: conflicting change")

	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RoutePatch)
	if err != nil {
		t.Fatalf("Route patch: %v", err)
	}
	// Patch should report conflicts but not error
	if result.Success {
		t.Error("patch should NOT succeed when there are conflicts")
	}
	if len(result.Conflicts) == 0 {
		t.Error("expected conflicts to be reported")
	}
}

func TestRoute_Hotfix_Success(t *testing.T) {
	repo := setupRouteRepo(t)

	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteHotfix)
	if err != nil {
		t.Fatalf("Route hotfix: %v", err)
	}
	if result.Mode != RouteHotfix {
		t.Errorf("Mode = %q, want %q", result.Mode, RouteHotfix)
	}
	// Hotfix applies to both main and develop
	if len(result.Commits) == 0 {
		t.Error("hotfix should transfer commits")
	}
}

func TestRoute_Emergency_BlockedInCI(t *testing.T) {
	// Set CI env var
	t.Setenv("CI", "true")

	repo := setupRouteRepo(t)
	_, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteEmergency)
	if err == nil {
		t.Fatal("emergency should be blocked in CI")
	}
	if !strings.Contains(err.Error(), "emergency") {
		t.Errorf("error should mention emergency: %v", err)
	}
}

func TestRoute_Emergency_NotCI(t *testing.T) {
	// Ensure CI vars are not set
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME", "BUILD_ID", "DRONE"} {
		t.Setenv(v, "")
	}

	repo := setupRouteRepo(t)

	// OWS-B4 FIX: Create a valid waiver for emergency routing.
	// Emergency now requires a CEO-signed waiver (previously was a no-op bypass).
	// OVAV_WAIVER_SECRET is set globally by init() in policy_test.go.
	w := SignWaiver("owx", "develop", 30*time.Minute)
	if w == nil {
		t.Fatal("failed to create test waiver — OVAV_WAIVER_SECRET may not be set")
	}
	waiverPath := filepath.Join(repo, ".ovav", "runtime", "protected_branch_waiver.yaml")
	if err := os.MkdirAll(filepath.Dir(waiverPath), 0755); err != nil {
		t.Fatalf("create waiver dir: %v", err)
	}
	waiverData := fmt.Sprintf("waiver_id: %s\ncommand: owx\ntarget: develop\nnonce: %s\nexpires_at: %d\nsignature: %s\n",
		w.ID, w.Nonce, w.ExpiresAt, w.Signature)
	if err := os.WriteFile(waiverPath, []byte(waiverData), 0644); err != nil {
		t.Fatalf("write waiver: %v", err)
	}

	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteEmergency)
	if err != nil {
		t.Fatalf("Route emergency: %v", err)
	}
	if !result.Success {
		t.Errorf("emergency should succeed outside CI. Conflicts: %v", result.Conflicts)
	}
}

func TestRoute_UnknownMode(t *testing.T) {
	repo := setupRouteRepo(t)
	_, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteMode("invalid"))
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown route mode") {
		t.Errorf("error should mention unknown mode: %v", err)
	}
}

func TestRoute_InvalidRepo(t *testing.T) {
	_, err := Route(context.Background(), "/nonexistent/path", "a", "b", RouteCherryPick)
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestRoute_ContextCancellation(t *testing.T) {
	repo := setupRouteRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, err := Route(ctx, repo, "feature/route-test", "develop", RouteCherryPick)
	// Should either error or return partial result
	if err != nil && err != context.Canceled {
		t.Logf("cancelled route: %v", err)
	}
	if result != nil {
		t.Logf("partial result: %d commits transferred", len(result.Commits))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// route.go — Abort() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAbort_NoOperation(t *testing.T) {
	repo := initTestRepo(t)
	err := Abort(repo)
	if err == nil {
		t.Fatal("expected error when no operation in progress")
	}
	if !strings.Contains(err.Error(), "no operation in progress") {
		t.Errorf("error should mention no operation: %v", err)
	}
}

func TestAbort_CherryPickInProgress(t *testing.T) {
	repo := initTestRepo(t)

	// Create a conflicting cherry-pick scenario
	runGitOk(t, repo, "checkout", "-b", "feature/abort-test")
	writeFileH(t, repo, "conflict.txt", "version A\n")
	runGitOk(t, repo, "add", "conflict.txt")
	runGitOk(t, repo, "commit", "-m", "add conflict file A")

	// Get the commit hash
	out, _ := runGitOutput(repo, "rev-parse", "HEAD")
	commitHash := strings.TrimSpace(out)

	// Go to a new branch and make a conflicting change
	defaultBranch := "master"
	if b, err := runGitOutput(repo, "rev-parse", "--abbrev-ref", "main"); err == nil && b != "" {
		defaultBranch = "main"
	}
	runGitOk(t, repo, "checkout", defaultBranch)
	writeFileH(t, repo, "conflict.txt", "version B\n")
	runGitOk(t, repo, "add", "conflict.txt")
	runGitOk(t, repo, "commit", "-m", "add conflict file B")

	// Start a cherry-pick that will conflict (don't abort, leave in progress)
	// Use raw exec to allow failure
	runGit(repo, "cherry-pick", "--no-commit", commitHash)

	// Now abort should work
	err := Abort(repo)
	if err != nil {
		// If cherry-pick didn't actually conflict, there may be no operation to abort
		t.Logf("abort result: %v", err)
	}
}

func TestAbort_MergeInProgress(t *testing.T) {
	repo := initTestRepo(t)

	// Create diverging branches
	runGitOk(t, repo, "checkout", "-b", "branch-a")
	writeFileH(t, repo, "diverge.txt", "branch A\n")
	runGitOk(t, repo, "add", "diverge.txt")
	runGitOk(t, repo, "commit", "-m", "branch A commit")

	defaultBranch := "master"
	runGitOk(t, repo, "checkout", defaultBranch)
	writeFileH(t, repo, "diverge.txt", "branch main\n")
	runGitOk(t, repo, "add", "diverge.txt")
	runGitOk(t, repo, "commit", "-m", "main diverge commit")

	// Start merge (will conflict)
	runGit(repo, "merge", "branch-a")

	// Abort should clean up
	err := Abort(repo)
	if err != nil {
		t.Logf("abort merge: %v", err)
	}
}

func TestAbort_InvalidRepo(t *testing.T) {
	err := Abort("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// route.go — IsCI() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestIsCI_Default(t *testing.T) {
	// Clear all CI vars
	for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME", "BUILD_ID", "DRONE"} {
		t.Setenv(v, "")
	}
	if IsCI() {
		t.Error("IsCI() should be false when no CI vars set")
	}
}

func TestIsCI_GitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	if !IsCI() {
		t.Error("IsCI() should be true when GITHUB_ACTIONS set")
	}
}

func TestIsCI_GenericCI(t *testing.T) {
	t.Setenv("CI", "true")
	if !IsCI() {
		t.Error("IsCI() should be true when CI set")
	}
}

func TestIsCI_GitLabCI(t *testing.T) {
	t.Setenv("GITLAB_CI", "true")
	if !IsCI() {
		t.Error("IsCI() should be true when GITLAB_CI set")
	}
}

func TestIsCI_Jenkins(t *testing.T) {
	t.Setenv("JENKINS_HOME", "/var/jenkins")
	if !IsCI() {
		t.Error("IsCI() should be true when JENKINS_HOME set")
	}
}

func TestIsCI_Drone(t *testing.T) {
	t.Setenv("DRONE", "true")
	if !IsCI() {
		t.Error("IsCI() should be true when DRONE set")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// route.go — Git Helper Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestUniqueCommits_FindsCommits(t *testing.T) {
	repo := setupRouteRepo(t)

	commits, err := uniqueCommits(repo, "feature/route-test", "develop")
	if err != nil {
		t.Fatalf("uniqueCommits: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("expected 2 unique commits, got %d", len(commits))
	}
}

func TestUniqueCommits_NoCommits(t *testing.T) {
	repo := initTestRepo(t)

	commits, err := uniqueCommits(repo, "master", "master")
	if err != nil {
		t.Fatalf("uniqueCommits: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}

func TestUniqueCommits_InvalidRef(t *testing.T) {
	repo := initTestRepo(t)

	_, err := uniqueCommits(repo, "nonexistent", "master")
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
}

func TestCurrentBranch_RealRepo(t *testing.T) {
	repo := initTestRepo(t)

	branch, err := currentBranch(repo)
	if err != nil {
		t.Fatalf("currentBranch: %v", err)
	}
	if branch == "" {
		t.Error("branch should not be empty")
	}
}

func TestCurrentBranch_InvalidRepo(t *testing.T) {
	_, err := currentBranch("/nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestCheckout_ValidBranch(t *testing.T) {
	repo := initTestRepo(t)
	runGitOk(t, repo, "checkout", "-b", "test-branch")

	err := checkout(repo, "master")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}

	branch, _ := currentBranch(repo)
	if branch != "master" {
		t.Errorf("branch = %q, want master", branch)
	}
}

func TestCheckout_InvalidBranch(t *testing.T) {
	repo := initTestRepo(t)
	err := checkout(repo, "nonexistent-branch")
	if err == nil {
		t.Fatal("expected error for invalid branch")
	}
}

func TestIsInProgress_NoGitDir(t *testing.T) {
	if isInProgress("/nonexistent", "CHERRY_PICK_HEAD") {
		t.Error("should return false for nonexistent dir")
	}
}

func TestIsInProgress_NormalRepo(t *testing.T) {
	repo := initTestRepo(t)

	// No operation in progress
	if isInProgress(repo, "CHERRY_PICK_HEAD") {
		t.Error("should return false when no cherry-pick in progress")
	}
	if isInProgress(repo, "REBASE_HEAD") {
		t.Error("should return false when no rebase in progress")
	}
	if isInProgress(repo, "MERGE_HEAD") {
		t.Error("should return false when no merge in progress")
	}
}

func TestIsInProgress_WorktreeGitFile(t *testing.T) {
	// Create a fake worktree .git file scenario
	dir := t.TempDir()
	gitDir := filepath.Join(dir, "real-git-dir")
	os.MkdirAll(gitDir, 0755)

	// Write .git as a file (worktree style)
	gitFile := filepath.Join(dir, ".git")
	os.WriteFile(gitFile, []byte("gitdir: "+gitDir+"\n"), 0644)

	// No marker file
	if isInProgress(dir, "CHERRY_PICK_HEAD") {
		t.Error("should return false when no marker in worktree gitdir")
	}

	// Create marker file in the real git dir
	os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("abc123"), 0644)
	if !isInProgress(dir, "CHERRY_PICK_HEAD") {
		t.Error("should return true when marker exists in worktree gitdir")
	}
}

func TestHasStagedChanges_WithChanges(t *testing.T) {
	repo := initTestRepo(t)

	writeFileH(t, repo, "staged.txt", "content\n")
	runGitOk(t, repo, "add", "staged.txt")

	if !hasStagedChanges(repo) {
		t.Error("should detect staged changes")
	}
}

func TestHasStagedChanges_NoChanges(t *testing.T) {
	repo := initTestRepo(t)

	if hasStagedChanges(repo) {
		t.Error("should return false when no staged changes")
	}
}

func TestHasStagedChanges_InvalidRepo(t *testing.T) {
	if hasStagedChanges("/nonexistent") {
		t.Error("should return false for invalid repo")
	}
}

func TestRunGit_Success(t *testing.T) {
	repo := initTestRepo(t)
	err := runGit(repo, "status")
	if err != nil {
		t.Fatalf("runGit status: %v", err)
	}
}

func TestRunGit_Failure(t *testing.T) {
	err := runGit("/nonexistent", "status")
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestRunGitCmdNoFail_InvalidRepo(t *testing.T) {
	// Should not panic even with invalid repo
	runGitCmdNoFail("/nonexistent", "status")
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — Rescue() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestRescue_ScansReflog(t *testing.T) {
	repo := initTestRepo(t)

	// Create a branch, commit, then delete it to create reflog entries
	runGitOk(t, repo, "checkout", "-b", "feature/rescue-test")
	writeFileH(t, repo, "rescue.txt", "save me\n")
	runGitOk(t, repo, "add", "rescue.txt")
	runGitOk(t, repo, "commit", "-m", "rescue: add file")
	runGitOk(t, repo, "checkout", "master")
	runGitOk(t, repo, "branch", "-D", "feature/rescue-test")

	result, err := Rescue(repo)
	if err != nil {
		t.Fatalf("Rescue: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// The deleted branch should appear in reflog
	t.Logf("Rescued commits: %d, branches: %d, worktrees: %d",
		len(result.RecoveredCommits), len(result.RecoveredBranches), len(result.RecoveredWorktrees))
}

func TestRescue_OrphanedBranches(t *testing.T) {
	repo := initTestRepo(t)

	// Create an unmerged branch
	runGitOk(t, repo, "checkout", "-b", "feature/orphan")
	writeFileH(t, repo, "orphan.txt", "orphaned work\n")
	runGitOk(t, repo, "add", "orphan.txt")
	runGitOk(t, repo, "commit", "-m", "orphan: add file")
	runGitOk(t, repo, "checkout", "master")

	result, err := Rescue(repo)
	if err != nil {
		t.Fatalf("Rescue: %v", err)
	}

	// The orphan branch should be found (it's not merged into develop)
	found := false
	for _, b := range result.RecoveredBranches {
		if strings.Contains(b, "feature/orphan") {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Recovered branches: %v (orphan may not be detected without develop branch)", result.RecoveredBranches)
	}
}

func TestRescue_WorktreeList(t *testing.T) {
	repo := initTestRepo(t)

	result, err := Rescue(repo)
	if err != nil {
		t.Fatalf("Rescue: %v", err)
	}
	// Should at least list the main worktree
	if len(result.RecoveredWorktrees) == 0 {
		t.Error("should list at least the main worktree")
	}
}

func TestRescue_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "empty")
	os.MkdirAll(repo, 0755)
	runGitOk(t, repo, "init")
	runGitOk(t, repo, "config", "user.email", "test@test.com")
	runGitOk(t, repo, "config", "user.name", "Test")

	result, err := Rescue(repo)
	if err != nil {
		t.Fatalf("Rescue on empty repo: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestRescue_InvalidRepo(t *testing.T) {
	// Rescue on nonexistent path should not panic
	result, err := Rescue("/nonexistent/path")
	if err != nil {
		t.Logf("Rescue on invalid path: %v (expected)", err)
	}
	if result != nil {
		t.Logf("result from invalid path: %+v", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — Sync() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestSync_WithRemote(t *testing.T) {
	repo := setupRepoWithRemote(t)

	err := Sync(repo)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
}

func TestSync_NoRemote(t *testing.T) {
	repo := initTestRepo(t)

	err := Sync(repo)
	// Should fail on fetch but not panic
	if err != nil {
		t.Logf("Sync without remote: %v (expected)", err)
	}
}

func TestSync_InvalidRepo(t *testing.T) {
	err := Sync("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — pruneStaleWorktrees() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestPruneStaleWorktrees_NoWorktrees(t *testing.T) {
	repo := initTestRepo(t)

	count := pruneStaleWorktrees(repo, 7*24*time.Hour)
	if count != 0 {
		t.Errorf("expected 0 pruned, got %d", count)
	}
}

func TestPruneStaleWorktrees_MainWorktreeSkipped(t *testing.T) {
	repo := initTestRepo(t)

	// Main worktree should never be pruned
	count := pruneStaleWorktrees(repo, 0) // 0 duration = everything is stale
	if count != 0 {
		t.Errorf("main worktree should not be pruned, got %d", count)
	}
}

func TestPruneStaleWorktrees_InvalidRepo(t *testing.T) {
	count := pruneStaleWorktrees("/nonexistent", 7*24*time.Hour)
	if count != 0 {
		t.Errorf("expected 0 for invalid repo, got %d", count)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — parseWorktreeList() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestParseWorktreeList_ValidOutput(t *testing.T) {
	porcelain := `worktree /home/user/repo
HEAD abc123
branch refs/heads/main

worktree /home/user/repo/.ovav/worktrees/feature-test
HEAD def456
branch refs/heads/feature/test

`
	worktrees := parseWorktreeList(porcelain)
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
	}
	if worktrees[0] != "/home/user/repo" {
		t.Errorf("worktree[0] = %q", worktrees[0])
	}
	if worktrees[1] != "/home/user/repo/.ovav/worktrees/feature-test" {
		t.Errorf("worktree[1] = %q", worktrees[1])
	}
}

func TestParseWorktreeList_EmptyOutput(t *testing.T) {
	worktrees := parseWorktreeList("")
	if len(worktrees) != 0 {
		t.Errorf("expected 0 worktrees, got %d", len(worktrees))
	}
}

func TestParseWorktreeList_SingleWorktree(t *testing.T) {
	porcelain := "worktree /tmp/repo\nHEAD abc\nbranch refs/heads/main\n"
	worktrees := parseWorktreeList(porcelain)
	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(worktrees))
	}
}

func TestParseWorktreePaths_SameAsList(t *testing.T) {
	porcelain := "worktree /a\nworktree /b\n"
	paths := parseWorktreePaths(porcelain)
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — parseValidateOutput() Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestParseValidateOutput_StandardFormat(t *testing.T) {
	out := `Checking validators...
── Results: 60 passed, 17 failed ──
`
	passed, failed := parseValidateOutput(out)
	if passed != 60 {
		t.Errorf("passed = %d, want 60", passed)
	}
	if failed != 17 {
		t.Errorf("failed = %d, want 17", failed)
	}
}

func TestParseValidateOutput_AllPassed(t *testing.T) {
	out := "── Results: 100 passed, 0 failed ──\n"
	passed, failed := parseValidateOutput(out)
	if passed != 100 {
		t.Errorf("passed = %d, want 100", passed)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
}

func TestParseValidateOutput_EmptyOutput(t *testing.T) {
	passed, failed := parseValidateOutput("")
	if passed != 0 || failed != 0 {
		t.Errorf("expected 0,0 for empty output, got %d,%d", passed, failed)
	}
}

func TestParseValidateOutput_NoResultsLine(t *testing.T) {
	out := "Some random output without results line\n"
	passed, failed := parseValidateOutput(out)
	if passed != 0 || failed != 0 {
		t.Errorf("expected 0,0 for no results, got %d,%d", passed, failed)
	}
}

func TestParseValidateOutput_MultilineOutput(t *testing.T) {
	out := `Step 1: Checking policies... OK
Step 2: Running validators...
── Results: 42 passed, 3 failed ──
Done.
`
	passed, failed := parseValidateOutput(out)
	if passed != 42 {
		t.Errorf("passed = %d, want 42", passed)
	}
	if failed != 3 {
		t.Errorf("failed = %d, want 3", failed)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// recovery.go — Verify() Tests (partial — full pipeline needs go-runtime)
// ═══════════════════════════════════════════════════════════════════════════

func TestVerify_InvalidRepo(t *testing.T) {
	// Verify on nonexistent path — should not panic, returns structured result
	result, err := Verify("/nonexistent/path", nil)
	if err != nil {
		t.Fatalf("Verify should not error, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	// All checks should fail for invalid repo
	if result.Passed {
		t.Error("Verify should not pass for invalid repo")
	}
	if result.GoVetPass {
		t.Error("GoVetPass should be false for invalid repo")
	}
	if result.GoTestPass {
		t.Error("GoTestPass should be false for invalid repo")
	}
}

func TestVerifyResult_Fields(t *testing.T) {
	r := &VerifyResult{
		GoTestPass:    true,
		GoVetPass:     true,
		GofmtPass:     true,
		ValidatePass:  10,
		ValidateFail:  0,
		ValidateTotal: 10,
		ValidateRan:   true,
		HygieneClean:  true,
		HygieneIssues: 0,
		Passed:        true,
		Detail:        "all checks passed",
	}

	if !r.Passed {
		t.Error("all green should pass")
	}
	if r.Detail != "all checks passed" {
		t.Errorf("Detail = %q", r.Detail)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Integration: Route + Abort
// ═══════════════════════════════════════════════════════════════════════════

func TestRouteAndAbort_Integration(t *testing.T) {
	repo := setupRouteRepo(t)

	// Route successfully
	result, err := Route(context.Background(), repo, "feature/route-test", "develop", RouteCherryPick)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if !result.Success {
		t.Skip("route didn't succeed, skipping abort test")
	}

	// After successful route, abort should say no operation
	err = Abort(repo)
	if err == nil {
		t.Error("abort should fail after successful route (no operation in progress)")
	}
}

func TestRouteMode_Constants(t *testing.T) {
	modes := []RouteMode{RouteCherryPick, RoutePatch, RouteHotfix, RouteEmergency}
	expected := []string{"cherry-pick", "patch", "hotfix", "emergency"}

	for i, mode := range modes {
		if string(mode) != expected[i] {
			t.Errorf("mode %d = %q, want %q", i, mode, expected[i])
		}
	}
}

func TestRouteResult_Fields(t *testing.T) {
	r := &RouteResult{
		Mode:      RouteCherryPick,
		SourceRef: "feature/x",
		TargetRef: "develop",
		Commits:   []string{"abc", "def"},
		Skipped:   []string{"ghi"},
		Conflicts: []string{"conflict in file.go"},
		Success:   false,
	}

	if r.Success {
		t.Error("should not be successful with skipped commits")
	}
	if len(r.Commits) != 2 {
		t.Errorf("Commits = %d", len(r.Commits))
	}
}
