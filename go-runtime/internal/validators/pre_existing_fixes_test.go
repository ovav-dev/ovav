package validators

// ── Fix #1: GitPush waiver recognition ────────────────────────────────────
// Bug: git_push.go looked for waiver at .ovav/governance/waivers/<branch>.yaml
// but the actual working waiver file lives at .ovav/runtime/protected_branch_waiver.yaml
// (matching protected_branch.go). Plus, CEO session should bypass the requirement
// (matching protected_branch.go behavior).
//
// These tests pin the correct contract.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitPush_ProtectedBranchWithRuntimeWaiver(t *testing.T) {
	dir := setupGitAtBranch(t, "develop")
	// Write the centralized waiver file at the canonical runtime path.
	waiverDir := filepath.Join(dir, ".ovav", "runtime")
	if err := os.MkdirAll(waiverDir, 0755); err != nil {
		t.Fatal(err)
	}
	waiverYAML := `waiver:
  branches:
    - develop
    - main
  reason: "CI/CD pipeline deployment — Thavren/Platform Engineering Lead authorization"
  authorized_by: CEO
  expires: "2099-01-01T00:00:00Z"
`
	if err := os.WriteFile(filepath.Join(waiverDir, "protected_branch_waiver.yaml"), []byte(waiverYAML), 0644); err != nil {
		t.Fatal(err)
	}

	v := NewGitPush()
	safe, msg := v.checkBranchSafety(dir)
	if !safe {
		t.Errorf("expected runtime waiver to satisfy gate for 'develop', got unsafe: %s", msg)
	}
}

func TestGitPush_ProtectedBranchWithoutWaiver(t *testing.T) {
	dir := setupGitAtBranch(t, "main")
	// No waiver file written — protected branch must be rejected.

	v := NewGitPush()
	safe, msg := v.checkBranchSafety(dir)
	if safe {
		t.Errorf("expected 'main' to require waiver, got safe with msg: %s", msg)
	}
	if !strings.Contains(msg, "no active waiver") {
		t.Errorf("expected 'no active waiver' message, got: %s", msg)
	}
}

func TestGitPush_NonProtectedBranch_NoWaiverRequired(t *testing.T) {
	dir := setupGitAtBranch(t, "feature/my-feature")

	v := NewGitPush()
	safe, msg := v.checkBranchSafety(dir)
	if !safe {
		t.Errorf("feature branch should not require waiver, got unsafe: %s", msg)
	}
}

// ── Fix #2: ConfigIntegrity version scheme separation ─────────────────────
// Bug: VERSION_MISMATCH compared VERSION (CLI version, e.g. 3.4.0) against the
// git tag (product version, e.g. v2.3.2). These are independent version
// streams:
//   * CLI/go-runtime: VERSION, go-runtime/cmd/cpanel/shared.go Version constant
//   * Product:        caps.yaml product.version, latest git tag, package.json
//
// Fix: cross-check VERSION against shared.go (CLI stream), and cross-check
// product.version (caps.yaml) against the latest git tag (product stream).

func TestConfigIntegrity_CLIAndProductVersionsIndependent(t *testing.T) {
	dir := t.TempDir()

	// Required configs
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte(`product:
  name: "OVAV"
  type: "test"
  version: "2.3.2"
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# OVAV\n"), 0644)

	// CLI version stream — 3.4.0 in VERSION, 3.4.0 in shared.go
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("3.4.0\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "go-runtime", "cmd", "cpanel"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "cmd", "cpanel", "shared.go"),
		[]byte("package cpanel\n\nconst Version = \"3.4.0\"\n"), 0644)

	// Product version stream — 2.3.2 in caps.yaml, but the git-tag lookup will
	// return either "" (no tags in temp dir) or whatever the host repo's
	// "latest tag" is. We make sure product.version matches git tag by
	// initializing a git repo with a tag matching product.version.
	initRepoWithTag(t, dir, "v2.3.2")

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected PASS with independent CLI (3.4.0) + product (2.3.2) versions, got %s: %v", result.Status, result.Issues)
	}
}

func TestConfigIntegrity_CLIVersionMismatchWithSharedGo(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte(`product:
  version: "2.3.2"
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# OVAV\n"), 0644)

	// CLI stream mismatch — VERSION 3.4.0 vs shared.go 3.3.0
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("3.4.0\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "go-runtime", "cmd", "cpanel"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "cmd", "cpanel", "shared.go"),
		[]byte("package cpanel\n\nconst Version = \"3.3.0\"\n"), 0644)

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)

	hasMismatch := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "VERSION_MISMATCH") && strings.Contains(issue, "shared.go") {
			hasMismatch = true
		}
	}
	if !hasMismatch {
		t.Errorf("expected CLI VERSION_MISMATCH against shared.go, got: %v", result.Issues)
	}
}

func TestConfigIntegrity_ProductVersionMismatchWithTag(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	// product.version says 2.3.2
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte(`product:
  version: "2.3.2"
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# OVAV\n"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("3.4.0\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "go-runtime", "cmd", "cpanel"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "cmd", "cpanel", "shared.go"),
		[]byte("package cpanel\n\nconst Version = \"3.4.0\"\n"), 0644)

	// Git tag disagrees — v2.3.1 vs product.version 2.3.2
	initRepoWithTag(t, dir, "v2.3.1")

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)

	hasProductMismatch := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "PRODUCT_VERSION_MISMATCH") {
			hasProductMismatch = true
		}
	}
	if !hasProductMismatch {
		t.Errorf("expected PRODUCT_VERSION_MISMATCH between caps.yaml and git tag, got: %v", result.Issues)
	}
}

// ── Fix #3: PluginSecurity recognizes gh CLI credential management ───────
// Bug: plugin_security.go flagged HTTPS remotes as needing a credential.helper
// even when `gh` CLI is providing authentication via ~/.config/gh/hosts.yml.
// This is a false positive in modern dev environments.

func TestPluginSecurity_HTTPSWithGhCLI(t *testing.T) {
	// Redirect $HOME to a tempdir so we control ~/.config/gh/hosts.yml.
	dir := t.TempDir()
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	ghDir := filepath.Join(fakeHome, ".config", "gh")
	if err := os.MkdirAll(ghDir, 0755); err != nil {
		t.Fatal(err)
	}
	ghHosts := `github.com:
    oauth_token: gho_FAKE_TOKEN_FOR_TESTING
    user: test-user
`
	if err := os.WriteFile(filepath.Join(ghDir, "hosts.yml"), []byte(ghHosts), 0600); err != nil {
		t.Fatal(err)
	}

	// HTTPS remote with NO credential.helper configured in .git/config
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	gitConfig := `[remote "origin"]
	url = https://github.com/ovav-dev/ovav.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(gitConfig), 0644); err != nil {
		t.Fatal(err)
	}

	v := NewPluginSecurity()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "credential helper") {
			t.Errorf("HTTPS + gh CLI auth should not flag credential helper, got: %s", issue)
		}
	}
}

func TestPluginSecurity_HTTPSWithoutAnyCredential(t *testing.T) {
	// Empty HOME — no gh CLI auth available.
	dir := t.TempDir()
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	// Make sure ~/.config/gh does NOT exist
	os.RemoveAll(filepath.Join(fakeHome, ".config"))

	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	gitConfig := `[remote "origin"]
	url = https://github.com/ovav-dev/ovav.git
	fetch = +refs/heads/*:refs/remotes/origin/*
`
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte(gitConfig), 0644); err != nil {
		t.Fatal(err)
	}

	v := NewPluginSecurity()
	result := v.Validate(context.Background(), dir)

	hasCredIssue := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "credential helper") {
			hasCredIssue = true
		}
	}
	if !hasCredIssue {
		t.Errorf("HTTPS without any credential mechanism should be flagged, got: %v", result.Issues)
	}
}

// ── Fix #4: ConfigSyntax excludes worktree paths ─────────────────────────
// Bug: .ovav/worktrees/<name>/ tracked files (which may be in-progress branches
// with malformed config files like JS-style comments in JSON) were scanned and
// caused false-positive JSON parse failures. Worktrees are isolation mechanism,
// not source. Add .ovav/worktrees/ to excludePatterns.

func TestConfigSyntax_SkipsWorktreePaths(t *testing.T) {
	dir := t.TempDir()

	// Place a broken JSON file inside .ovav/worktrees/<name>/ — must be ignored.
	wtDir := filepath.Join(dir, ".ovav", "worktrees", "feature-broken")
	if err := os.MkdirAll(wtDir, 0755); err != nil {
		t.Fatal(err)
	}
	// File with leading JS-style /** comment — invalid JSON.
	broken := `/**
 * JS-style comment header — invalid JSON
 */
{
  "name": "broken"
`
	if err := os.WriteFile(filepath.Join(wtDir, "extension.json"), []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}

	v := NewConfigSyntax()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected PASS — worktree paths should be excluded, got %s: %v", result.Status, result.Issues)
	}
}

// ── Test helpers ──────────────────────────────────────────────────────────

// initRepoWithTag initializes a git repo at dir and creates a single tag.
// This isolates the git-tag lookup from the host repo's tags.
func initRepoWithTag(t *testing.T, dir, tag string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"-c", "user.email=test@test", "-c", "user.name=test", "commit", "--allow-empty", "-m", "init"},
		{"tag", tag},
	} {
		cmd := newGitCmd(dir, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
}

// newGitCmd builds a git command rooted at dir, inheriting the test env so
// HOME/GIT_AUTHOR_*/GIT_COMMITTER_* stay under the test's control.
func newGitCmd(dir string, args ...string) *exec.Cmd {
	c := exec.Command("git", args...)
	c.Dir = dir
	return c
}

// setupGitAtBranch creates a real git repo at t.TempDir() with HEAD pointing
// at the given branch. checkBranchSafety uses `git branch --show-current`,
// which requires a real git repo.
func setupGitAtBranch(t *testing.T, branch string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--initial-branch=" + branch},
		{"-c", "user.email=test@test", "-c", "user.name=test", "commit", "--allow-empty", "-m", "init"},
	} {
		cmd := newGitCmd(dir, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	return dir
}
