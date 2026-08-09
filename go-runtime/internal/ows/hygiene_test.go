package ows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── Helper: create a minimal git repo for testing ────────────────────────────

func hygieneTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitHygiene(dir, "init")
	runGitHygiene(dir, "config", "user.email", "test@ovav.dev")
	runGitHygiene(dir, "config", "user.name", "Test User")
	// Create .gitignore
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n.ovav/audit/\n"), 0644)
	// Initial commit
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGitHygiene(dir, "add", "README.md", ".gitignore")
	runGitHygiene(dir, "commit", "-m", "initial")
	return dir
}

func runGitHygiene(dir string, args ...string) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Run()
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestHygiene_CleanRepo(t *testing.T) {
	dir := hygieneTestRepo(t)
	r := WorkspaceHygieneScan(dir)

	if !r.Clean {
		t.Errorf("expected clean repo, got %d issues", r.TotalIssues)
	}
}

func TestHygiene_UntrackedFiles(t *testing.T) {
	dir := hygieneTestRepo(t)
	os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("hello"), 0644)

	r := WorkspaceHygieneScan(dir)

	if len(r.UntrackedFiles) == 0 {
		t.Error("expected untracked files")
	}
	found := false
	for _, f := range r.UntrackedFiles {
		if f == "newfile.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected newfile.txt in untracked, got: %v", r.UntrackedFiles)
	}
}

func TestHygiene_UnstagedModified(t *testing.T) {
	dir := hygieneTestRepo(t)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# modified"), 0644)

	r := WorkspaceHygieneScan(dir)

	if len(r.UnstagedModified) == 0 {
		t.Error("expected unstaged modified files")
	}
}

func TestHygiene_StaleLocks(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Create .ovav/locks/ with old lock file
	locksDir := filepath.Join(dir, ".ovav", "locks")
	os.MkdirAll(locksDir, 0755)
	lockPath := filepath.Join(locksDir, "old-worktree.lock")
	os.WriteFile(lockPath, []byte("owner:test\nexpiry:past"), 0644)

	// Set mtime to 25 hours ago
	oldTime := time.Now().Add(-25 * time.Hour)
	os.Chtimes(lockPath, oldTime, oldTime)

	r := WorkspaceHygieneScan(dir)

	if len(r.StaleLockFiles) == 0 {
		t.Error("expected stale lock detection")
	}
	for _, sl := range r.StaleLockFiles {
		if sl.Path == ".ovav/locks/old-worktree.lock" && sl.Expired {
			return // ✅
		}
	}
	t.Errorf("expected expired lock .ovav/locks/old-worktree.lock, got: %+v", r.StaleLockFiles)
}

func TestHygiene_BrokenSymlinks(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Create broken symlink
	os.Symlink("/nonexistent/path", filepath.Join(dir, "broken-link"))

	r := WorkspaceHygieneScan(dir)

	if len(r.BrokenSymlinks) == 0 {
		t.Error("expected broken symlink detection")
	}
}

func TestHygiene_DirtyAfterMerge(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Create a file that's untracked AND not in .gitignore
	os.WriteFile(filepath.Join(dir, "orphan-change.txt"), []byte("forgotten"), 0644)

	r := WorkspaceHygieneScan(dir)

	// The untracked file should appear in either DirtyAfterMerge or UntrackedFiles
	// DirtyAfterMerge = untracked/unstaged files not inside any active worktree
	totalDirty := len(r.DirtyAfterMerge) + len(r.UntrackedFiles)
	if totalDirty == 0 {
		t.Error("expected at least one dirty or untracked file")
	}

	// Specifically, orphan-change.txt should be found somewhere
	found := false
	for _, f := range r.DirtyAfterMerge {
		if f == "orphan-change.txt" {
			found = true
		}
	}
	for _, f := range r.UntrackedFiles {
		if f == "orphan-change.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("orphan-change.txt not found in dirty (%v) or untracked (%v)",
			r.DirtyAfterMerge, r.UntrackedFiles)
	}
}

func TestHygiene_GeneratedFileDrift(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Create a generated directory and add an unstaged file
	genDir := filepath.Join(dir, "runtimes", "opencode", "agents")
	os.MkdirAll(genDir, 0755)
	os.WriteFile(filepath.Join(genDir, "lead-thavren.md"), []byte("stale content"), 0644)
	// Stage and commit to establish baseline, then modify
	runGitHygiene(dir, "add", "runtimes/opencode/agents/lead-thavren.md")
	runGitHygiene(dir, "commit", "-m", "add generated file")
	// Now modify unstaged
	os.WriteFile(filepath.Join(genDir, "lead-thavren.md"), []byte("modified unstaged"), 0644)

	r := WorkspaceHygieneScan(dir)

	if len(r.GeneratedFileDrift) == 0 {
		t.Error("expected generated file drift detection")
	}
	for _, gd := range r.GeneratedFileDrift {
		if strings.Contains(gd.File, "lead-thavren.md") && strings.Contains(gd.Detail, "Regenerable") {
			return // ✅
		}
	}
	t.Errorf("expected drift for lead-thavren.md, got: %+v", r.GeneratedFileDrift)
}

func TestHygiene_LargeUntracked(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Create a "large" zip file (just over 1MB)
	largePath := filepath.Join(dir, "backup.zip")
	f, _ := os.Create(largePath)
	f.Write(make([]byte, 1024*1024+1)) // 1MB + 1 byte
	f.Close()

	r := WorkspaceHygieneScan(dir)

	if len(r.LargeUntrackedFiles) == 0 {
		t.Error("expected large untracked file detection")
	}
	if r.BlockingIssues == 0 {
		t.Error("large files should be blocking")
	}
}

func TestHygiene_GitConfig(t *testing.T) {
	dir := t.TempDir()
	runGitHygiene(dir, "init")
	// No user.email or user.name configured

	r := WorkspaceHygieneScan(dir)

	if len(r.GitConfigWarnings) == 0 {
		t.Error("expected git config warnings for missing email/name")
	}
}

func TestHygiene_AuditTrail(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Remove .ovav/audit/ from .gitignore
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0644)

	r := WorkspaceHygieneScan(dir)

	if r.AuditTrailWarning == "" {
		t.Error("expected audit trail warning when .ovav/audit/ not in .gitignore")
	}
}

func TestHygiene_Report(t *testing.T) {
	dir := hygieneTestRepo(t)
	os.WriteFile(filepath.Join(dir, "orphan.txt"), []byte("orphan"), 0644)

	r := WorkspaceHygieneScan(dir)
	report := r.Report()

	if !strings.Contains(report, "Workspace Hygiene") {
		t.Error("report should contain header")
	}
	if r.Clean {
		t.Error("should not be clean with untracked files")
	}
}

func TestHygiene_Summary(t *testing.T) {
	dir := hygieneTestRepo(t)
	os.WriteFile(filepath.Join(dir, "f1.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "f2.txt"), []byte("b"), 0644)

	r := WorkspaceHygieneScan(dir)

	if r.TotalIssues == 0 {
		t.Error("expected issues")
	}
	if r.BlockingIssues != 0 {
		t.Errorf("expected 0 blocking, got %d", r.BlockingIssues)
	}
}

// TestHygiene_RealWorldScenario simulates the exact issue that prompted this:
// after a merge, generated runtime files and plugins can be left unstaged/modified.
func TestHygiene_RealWorldScenario(t *testing.T) {
	dir := hygieneTestRepo(t)

	// Simulate generated file drift: runtimes/opencode/agents/ modified
	genDir := filepath.Join(dir, "runtimes", "opencode", "agents")
	os.MkdirAll(genDir, 0755)
	os.WriteFile(filepath.Join(genDir, "lead-eidren.md"), []byte("canonical content"), 0644)
	os.WriteFile(filepath.Join(genDir, "lead-thavren.md"), []byte("canonical content"), 0644)
	runGitHygiene(dir, "add", "runtimes/")
	runGitHygiene(dir, "commit", "-m", "add generated runtime")

	// Now simulate post-merge: modify these files without staging
	os.WriteFile(filepath.Join(genDir, "lead-eidren.md"), []byte("modified unstaged"), 0644)
	os.WriteFile(filepath.Join(genDir, "lead-thavren.md"), []byte("modified unstaged"), 0644)

	// Also simulate plugin file drift
	pluginDir := filepath.Join(dir, "clients", "opencode", "plugins")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "ovav-monitor.js"), []byte("old plugin"), 0644)
	runGitHygiene(dir, "add", "clients/")
	runGitHygiene(dir, "commit", "-m", "add plugin")
	os.WriteFile(filepath.Join(pluginDir, "ovav-monitor.js"), []byte("modified plugin unstaged"), 0644)

	r := WorkspaceHygieneScan(dir)

	// Should find unstaged modifications
	if len(r.UnstagedModified) < 2 {
		t.Errorf("expected at least 2 unstaged files, got %d: %v", len(r.UnstagedModified), r.UnstagedModified)
	}

	// Should find generated file drift
	if len(r.GeneratedFileDrift) == 0 {
		t.Error("expected generated file drift for modified runtime files")
	}

	// Verify the drift tells you it's regenerable
	hasRegenerable := false
	for _, gd := range r.GeneratedFileDrift {
		if strings.Contains(gd.Detail, "Regenerable") {
			hasRegenerable = true
		}
	}
	if !hasRegenerable {
		t.Error("drift findings should mention files are regenerable")
	}

	// Total issues should be non-zero
	if r.TotalIssues == 0 {
		t.Error("expected hygiene issues in real-world scenario")
	}
	t.Logf("Real-world scenario: %d issues, %d blocking", r.TotalIssues, r.BlockingIssues)
	t.Logf("Unstaged: %v", r.UnstagedModified)
	t.Logf("Drift: %+v", r.GeneratedFileDrift)
}
