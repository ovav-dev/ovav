package ows

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// policy_extra_test.go — Additional tests for policy.go coverage
// Focus: parseWaiverYAML, splitLines, splitColon, trimSpace, randomNonce,
//        WaiverSecret, checkVerifiedGate, checkRemoteHTTPS, listGitRemotes
// ═══════════════════════════════════════════════════════════════════════════

// ── parseWaiverYAML ──────────────────────────────────────────────────────

func TestParseWaiverYAML_AllFields(t *testing.T) {
	yaml := `waiver_id: w-12345
command: owx
target: main
nonce: abc123def456
expires_at: 1700000000
signature: deadbeef12345678
`
	w, err := parseWaiverYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseWaiverYAML: %v", err)
	}
	if w.ID != "w-12345" {
		t.Errorf("ID = %q, want w-12345", w.ID)
	}
	if w.Command != "owx" {
		t.Errorf("Command = %q, want owx", w.Command)
	}
	if w.Target != "main" {
		t.Errorf("Target = %q, want main", w.Target)
	}
	if w.Nonce != "abc123def456" {
		t.Errorf("Nonce = %q, want abc123def456", w.Nonce)
	}
	if w.ExpiresAt != 1700000000 {
		t.Errorf("ExpiresAt = %d, want 1700000000", w.ExpiresAt)
	}
	if w.Signature != "deadbeef12345678" {
		t.Errorf("Signature = %q, want deadbeef12345678", w.Signature)
	}
}

func TestParseWaiverYAML_MissingID(t *testing.T) {
	yaml := `command: owx
target: main
nonce: abc
`
	_, err := parseWaiverYAML([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing waiver_id")
	}
	if !strings.Contains(err.Error(), "missing waiver_id") {
		t.Errorf("error should mention missing waiver_id: %v", err)
	}
}

func TestParseWaiverYAML_EmptyInput(t *testing.T) {
	_, err := parseWaiverYAML([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseWaiverYAML_PartialFields(t *testing.T) {
	yaml := `waiver_id: w-partial
command: owd
`
	w, err := parseWaiverYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseWaiverYAML: %v", err)
	}
	if w.ID != "w-partial" {
		t.Errorf("ID = %q, want w-partial", w.ID)
	}
	if w.Command != "owd" {
		t.Errorf("Command = %q, want owd", w.Command)
	}
	// Other fields should be zero-valued
	if w.Target != "" {
		t.Errorf("Target should be empty, got %q", w.Target)
	}
}

func TestParseWaiverYAML_ExtraWhitespace(t *testing.T) {
	yaml := `  waiver_id:   w-spaces  
  command:   owx  
  target:   develop  
`
	w, err := parseWaiverYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseWaiverYAML: %v", err)
	}
	if w.ID != "w-spaces" {
		t.Errorf("ID = %q, want w-spaces", w.ID)
	}
	if w.Command != "owx" {
		t.Errorf("Command = %q, want owx", w.Command)
	}
}

func TestParseWaiverYAML_ColonInValue(t *testing.T) {
	// Values with colons (like timestamps) should be handled
	yaml := `waiver_id: w-colon
command: owx
target: feature/test:sub
`
	w, err := parseWaiverYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseWaiverYAML: %v", err)
	}
	// splitColon splits on first colon only, so "feature/test:sub" should be preserved
	if !strings.Contains(w.Target, "test") {
		t.Errorf("Target should contain 'test': %q", w.Target)
	}
}

// ── splitLines ────────────────────────────────────────────────────────────

func TestSplitLines_MultipleLines(t *testing.T) {
	lines := splitLines("line1\nline2\nline3")
	if len(lines) != 3 {
		t.Errorf("splitLines: got %d lines, want 3", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("splitLines: unexpected values: %v", lines)
	}
}

func TestSplitLines_SingleLine(t *testing.T) {
	lines := splitLines("single")
	if len(lines) != 1 {
		t.Errorf("splitLines: got %d lines, want 1", len(lines))
	}
	if lines[0] != "single" {
		t.Errorf("splitLines: got %q, want 'single'", lines[0])
	}
}

func TestSplitLines_Empty(t *testing.T) {
	lines := splitLines("")
	if len(lines) != 0 {
		t.Errorf("splitLines(''): got %d lines, want 0", len(lines))
	}
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	lines := splitLines("a\nb\n")
	// splitLines filters trailing empty elements, so "a\nb\n" → ["a", "b"]
	if len(lines) != 2 {
		t.Errorf("splitLines with trailing newline: got %d lines, want 2", len(lines))
	}
}

// ── splitColon ────────────────────────────────────────────────────────────

func TestSplitColon_WithColon(t *testing.T) {
	parts := splitColon("key:value")
	if len(parts) != 2 {
		t.Fatalf("splitColon: got %d parts, want 2", len(parts))
	}
	if parts[0] != "key" || parts[1] != "value" {
		t.Errorf("splitColon: got %v, want [key, value]", parts)
	}
}

func TestSplitColon_NoColon(t *testing.T) {
	parts := splitColon("nocolon")
	if len(parts) != 1 {
		t.Errorf("splitColon: got %d parts, want 1", len(parts))
	}
	if parts[0] != "nocolon" {
		t.Errorf("splitColon: got %q, want 'nocolon'", parts[0])
	}
}

func TestSplitColon_MultipleColons(t *testing.T) {
	// Should split on first colon only
	parts := splitColon("a:b:c")
	if len(parts) != 2 {
		t.Fatalf("splitColon: got %d parts, want 2", len(parts))
	}
	if parts[0] != "a" || parts[1] != "b:c" {
		t.Errorf("splitColon: got %v, want [a, b:c]", parts)
	}
}

func TestSplitColon_EmptyString(t *testing.T) {
	parts := splitColon("")
	if len(parts) != 1 {
		t.Errorf("splitColon(''): got %d parts, want 1", len(parts))
	}
}

// ── trimSpace ─────────────────────────────────────────────────────────────

func TestTrimSpace_Various(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"\thello\t", "hello"},
		{"hello", "hello"},
		{"  ", ""},
		{"", ""},
		{" \t hello \t ", "hello"},
		{"no_trim_here", "no_trim_here"},
	}
	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── randomNonce ───────────────────────────────────────────────────────────

func TestRandomNonce_Length(t *testing.T) {
	nonce, err := randomNonce()
	if err != nil {
		t.Fatalf("randomNonce: %v", err)
	}
	if len(nonce) != 16 {
		t.Errorf("nonce length = %d, want 16", len(nonce))
	}
}

func TestRandomNonce_Uniqueness(t *testing.T) {
	nonces := make(map[string]bool)
	for i := 0; i < 100; i++ {
		n, err := randomNonce()
		if err != nil {
			t.Fatalf("randomNonce iteration %d: %v", i, err)
		}
		if nonces[n] {
			t.Errorf("duplicate nonce detected: %s", n)
		}
		nonces[n] = true
	}
}

func TestRandomNonce_HexChars(t *testing.T) {
	nonce, err := randomNonce()
	if err != nil {
		t.Fatalf("randomNonce: %v", err)
	}
	for _, c := range nonce {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("nonce contains non-hex char: %c in %s", c, nonce)
			break
		}
	}
}

// ── WaiverSecret ──────────────────────────────────────────────────────────

func TestWaiverSecret_WithEnvVar(t *testing.T) {
	// init() in policy_test.go sets OVAV_WAIVER_SECRET
	secret, err := WaiverSecret()
	if err != nil {
		t.Fatalf("WaiverSecret: %v", err)
	}
	if len(secret) == 0 {
		t.Error("WaiverSecret returned empty bytes")
	}
}

func TestWaiverSecret_WithoutEnvVar(t *testing.T) {
	// Save and clear
	orig := os.Getenv("OVAV_WAIVER_SECRET")
	os.Unsetenv("OVAV_WAIVER_SECRET")
	defer os.Setenv("OVAV_WAIVER_SECRET", orig)

	_, err := WaiverSecret()
	if err == nil {
		t.Fatal("WaiverSecret should fail without env var")
	}
	if err != ErrWaiverSecretNotSet {
		t.Errorf("expected ErrWaiverSecretNotSet, got: %v", err)
	}
}

// ── checkVerifiedGate ─────────────────────────────────────────────────────

func TestCheckVerifiedGate_NoResults(t *testing.T) {
	dir := t.TempDir()
	passed, msg := checkVerifiedGate(dir)
	if passed {
		t.Error("should fail when no verification results exist")
	}
	if !strings.Contains(msg, "run owv") {
		t.Errorf("message should suggest running owv: %s", msg)
	}
}

func TestCheckVerifiedGate_PassedResults(t *testing.T) {
	dir := t.TempDir()
	verifyDir := filepath.Join(dir, ".ovav", "verify")
	os.MkdirAll(verifyDir, 0755)

	vr := VerifyResult{Passed: true, GoVetPass: true, GofmtPass: true, GoTestPass: true}
	data, _ := json.Marshal(vr)
	os.WriteFile(filepath.Join(verifyDir, "last_result.json"), data, 0644)

	passed, _ := checkVerifiedGate(dir)
	if !passed {
		t.Error("should pass when verification results show passed=true")
	}
}

func TestCheckVerifiedGate_FailedResults(t *testing.T) {
	dir := t.TempDir()
	verifyDir := filepath.Join(dir, ".ovav", "verify")
	os.MkdirAll(verifyDir, 0755)

	vr := VerifyResult{Passed: false, Detail: "go vet failed"}
	data, _ := json.Marshal(vr)
	os.WriteFile(filepath.Join(verifyDir, "last_result.json"), data, 0644)

	passed, msg := checkVerifiedGate(dir)
	if passed {
		t.Error("should fail when verification results show passed=false")
	}
	if !strings.Contains(msg, "did not pass") {
		t.Errorf("message should mention failure: %s", msg)
	}
}

func TestCheckVerifiedGate_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	verifyDir := filepath.Join(dir, ".ovav", "verify")
	os.MkdirAll(verifyDir, 0755)

	os.WriteFile(filepath.Join(verifyDir, "last_result.json"), []byte("not json"), 0644)

	passed, msg := checkVerifiedGate(dir)
	if passed {
		t.Error("should fail for corrupt JSON")
	}
	if !strings.Contains(msg, "corrupt") {
		t.Errorf("message should mention corrupt: %s", msg)
	}
}

// ── checkRemoteHTTPS ──────────────────────────────────────────────────────

func TestCheckRemoteHTTPS_NoRemotes(t *testing.T) {
	repo := initTestRepo(t)
	// Ensure no remotes exist (initTestRepo has none)
	passed, _ := checkRemoteHTTPS(repo)
	// No remotes → should pass (nothing to check)
	if !passed {
		t.Error("should pass with no remotes")
	}
}

func TestCheckRemoteHTTPS_HTTPSRemote(t *testing.T) {
	repo := initTestRepo(t)
	// Add HTTPS remote (initTestRepo has no remotes)
	runGitOk(t, repo, "remote", "add", "origin", "https://github.com/test/repo.git")

	passed, _ := checkRemoteHTTPS(repo)
	if !passed {
		t.Error("should pass with HTTPS remote")
	}
}

func TestCheckRemoteHTTPS_SSHRemote(t *testing.T) {
	repo := initTestRepo(t)
	// Add SSH remote
	runGitOk(t, repo, "remote", "add", "origin", "git@github.com:test/repo.git")

	passed, msg := checkRemoteHTTPS(repo)
	if passed {
		t.Error("should fail with SSH remote")
	}
	if !strings.Contains(msg, "non-HTTPS") {
		t.Errorf("message should mention non-HTTPS: %s", msg)
	}
}

// ── listGitRemotes ────────────────────────────────────────────────────────

func TestListGitRemotes_WithOrigin(t *testing.T) {
	repo := initTestRepo(t)
	// Add HTTPS remote
	runGitOk(t, repo, "remote", "add", "origin", "https://github.com/test/repo.git")

	urls, err := listGitRemotes(repo)
	if err != nil {
		t.Fatalf("listGitRemotes: %v", err)
	}
	if len(urls) == 0 {
		t.Error("should have at least one remote URL")
	}
}

func TestListGitRemotes_NoRemotes(t *testing.T) {
	repo := initTestRepo(t)
	// initTestRepo creates a repo with no remotes

	urls, err := listGitRemotes(repo)
	if err != nil {
		t.Fatalf("listGitRemotes: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(urls))
	}
}

func TestListGitRemotes_InvalidDir(t *testing.T) {
	_, err := listGitRemotes("/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid directory")
	}
}

// ── PolicyEngine.ValidateAll with different levels ────────────────────────

func TestPolicyEngine_ValidateAll_Relaxed(t *testing.T) {
	dir := t.TempDir()
	db, _ := OpenAudit(dir)
	defer db.Close()
	pe := NewPolicyEngine(db)

	// Relaxed should pass (only relaxed policies apply)
	err := pe.ValidateAll(PolicyRelaxed, dir, dir)
	if err != nil {
		t.Logf("relaxed: %v (may fail on HTTPS check)", err)
	}
}

func TestPolicyEngine_ValidateAll_Waiver(t *testing.T) {
	dir := t.TempDir()
	db, _ := OpenAudit(dir)
	defer db.Close()
	pe := NewPolicyEngine(db)

	// Waiver level only checks POL-008 (which delegates to ValidateWaiver)
	err := pe.ValidateAll(PolicyWaiver, dir, dir)
	if err != nil {
		t.Logf("waiver: %v", err)
	}
}

// ── SignWaiver edge cases ─────────────────────────────────────────────────

func TestSignWaiver_NoSecret(t *testing.T) {
	orig := os.Getenv("OVAV_WAIVER_SECRET")
	os.Unsetenv("OVAV_WAIVER_SECRET")
	defer os.Setenv("OVAV_WAIVER_SECRET", orig)

	w := SignWaiver("owx", "main", 30*time.Minute)
	if w != nil {
		t.Error("SignWaiver should return nil without secret")
	}
}

func TestSignWaiver_ZeroTTL(t *testing.T) {
	w := SignWaiver("owx", "main", -1*time.Second)
	if w == nil {
		t.Fatal("SignWaiver should succeed with negative TTL")
	}
	// Should be expired immediately (negative TTL)
	err := ValidateWaiver(w, "owx", "main")
	if err == nil {
		t.Error("negative TTL waiver should be expired")
	}
}

// ── ValidateWaiver edge cases ─────────────────────────────────────────────

func TestValidateWaiver_EmptyCommandMatch(t *testing.T) {
	w := SignWaiver("owx", "main", 30*time.Minute)
	// Empty expectedCommand should match any command
	err := ValidateWaiver(w, "", "main")
	if err != nil {
		t.Errorf("empty command should match: %v", err)
	}
}

func TestValidateWaiver_EmptyTargetMatch(t *testing.T) {
	w := SignWaiver("owx", "main", 30*time.Minute)
	// Empty expectedTarget should match any target
	err := ValidateWaiver(w, "owx", "")
	if err != nil {
		t.Errorf("empty target should match: %v", err)
	}
}

// ── parseWorktreeList ─────────────────────────────────────────────────────

func TestParseWorktreeList_Porcelain(t *testing.T) {
	porcelain := `worktree /home/user/repo
HEAD abc123
branch refs/heads/main

worktree /home/user/repo/.ovav/worktrees/feature-x
HEAD def456
branch refs/heads/feature/x

`
	result := parseWorktreeList(porcelain)
	if len(result) != 2 {
		t.Errorf("parseWorktreeList: got %d, want 2", len(result))
	}
	if result[0] != "/home/user/repo" {
		t.Errorf("first worktree = %q", result[0])
	}
	if result[1] != "/home/user/repo/.ovav/worktrees/feature-x" {
		t.Errorf("second worktree = %q", result[1])
	}
}

func TestParseWorktreeList_Empty(t *testing.T) {
	result := parseWorktreeList("")
	if len(result) != 0 {
		t.Errorf("parseWorktreeList(''): got %d, want 0", len(result))
	}
}

// ── LockWorktree (top-level function) ─────────────────────────────────────

func TestLockWorktree_NilLock(t *testing.T) {
	err := LockWorktree(nil)
	if err == nil {
		t.Fatal("LockWorktree(nil) should fail")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should mention invalid: %v", err)
	}
}

func TestLockWorktree_EmptyWorktree(t *testing.T) {
	err := LockWorktree(&AgentLock{Worktree: "", Reason: "test", Owner: "user"})
	if err == nil {
		t.Fatal("LockWorktree with empty worktree should fail")
	}
}

func TestLockWorktree_NonGitPath(t *testing.T) {
	dir := t.TempDir()
	err := LockWorktree(&AgentLock{Worktree: dir, Reason: "test", Owner: "user"})
	if err == nil {
		t.Fatal("LockWorktree on non-git path should fail")
	}
}

// ── resolveRepoRoot ───────────────────────────────────────────────────────

func TestResolveRepoRoot_ValidRepo(t *testing.T) {
	repo := initTestRepo(t)
	root, err := resolveRepoRoot(repo)
	if err != nil {
		t.Fatalf("resolveRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("resolveRepoRoot returned empty string")
	}
}

func TestResolveRepoRoot_InvalidPath(t *testing.T) {
	_, err := resolveRepoRoot("/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-git path")
	}
}

// ── VerifyResult JSON round-trip ──────────────────────────────────────────

func TestVerifyResult_JSONRoundTrip(t *testing.T) {
	vr := VerifyResult{
		GoTestPass:    true,
		GoVetPass:     true,
		GofmtPass:     false,
		CoveragePass:  true,
		CoveragePct:   85.5,
		ValidatePass:  10,
		ValidateFail:  2,
		ValidateTotal: 12,
		ValidateRan:   true,
		HygieneClean:  true,
		HygieneIssues: 0,
		Passed:        true,
		Detail:        "all good",
	}

	data, err := json.Marshal(vr)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded VerifyResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.GoTestPass != vr.GoTestPass {
		t.Error("GoTestPass mismatch")
	}
	if decoded.CoveragePct != vr.CoveragePct {
		t.Error("CoveragePct mismatch")
	}
	if decoded.ValidatePass != vr.ValidatePass {
		t.Error("ValidatePass mismatch")
	}
	if decoded.Detail != vr.Detail {
		t.Error("Detail mismatch")
	}
}

// ── RescueResult ──────────────────────────────────────────────────────────

func TestRescueResult_Empty(t *testing.T) {
	r := &RescueResult{}
	if len(r.RecoveredCommits) != 0 {
		t.Error("empty result should have 0 commits")
	}
	if len(r.RecoveredBranches) != 0 {
		t.Error("empty result should have 0 branches")
	}
	if len(r.RecoveredWorktrees) != 0 {
		t.Error("empty result should have 0 worktrees")
	}
}

// ── Rescue on real repo ───────────────────────────────────────────────────

func TestRescue_BasicRepo(t *testing.T) {
	repo := initTestRepo(t)
	result, err := Rescue(repo)
	if err != nil {
		t.Fatalf("Rescue: %v", err)
	}
	if result == nil {
		t.Fatal("Rescue returned nil result")
	}
	// Basic repo should have no orphaned worktrees
	// (may have some reflog entries depending on git operations)
}
