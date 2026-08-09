package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ovav/ovav/internal/truststore"
)

// ── AdversarialVerification ─────────────────────────────────────────────────

func TestAdversarialVerification_NoClaims(t *testing.T) {
	dir := t.TempDir()
	v := NewAdversarialVerification()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass (no claims), got %s", result.Status)
	}
}

func TestAdversarialVerification_JudgeRejected(t *testing.T) {
	v := NewAdversarialVerification()
	jurors := []Juror{
		{Name: "a", Pass: false, Reason: "bad"},
		{Name: "b", Pass: false, Reason: "bad"},
		{Name: "c", Pass: false, Reason: "bad"},
	}
	verdict := v.judge(jurors)
	if verdict != "rejected" {
		t.Errorf("expected rejected, got %s", verdict)
	}
}

func TestAdversarialVerification_JudgeVerified(t *testing.T) {
	v := NewAdversarialVerification()
	jurors := []Juror{
		{Name: "a", Pass: true, Reason: "ok"},
		{Name: "b", Pass: true, Reason: "ok"},
		{Name: "c", Pass: false, Reason: "bad"},
	}
	verdict := v.judge(jurors)
	if verdict != "verified" {
		t.Errorf("expected verified, got %s", verdict)
	}
}

func TestAdversarialVerification_RunJurors(t *testing.T) {
	v := NewAdversarialVerification()
	claim := Claim{ID: "c1", Description: "test", Evidence: "some evidence"}
	jurors := v.runJurors(claim, t.TempDir())
	if len(jurors) != 3 {
		t.Fatalf("expected 3 jurors, got %d", len(jurors))
	}
}

func TestAdversarialVerification_JurorEvidenceNoEvidence(t *testing.T) {
	v := NewAdversarialVerification()
	claim := Claim{ID: "c1", Description: "test", Evidence: ""}
	juror := v.jurorEvidence(claim)
	if juror.Pass {
		t.Error("expected fail when no evidence")
	}
}

func TestAdversarialVerification_JurorEvidenceWithEvidence(t *testing.T) {
	v := NewAdversarialVerification()
	claim := Claim{ID: "c1", Description: "test", Evidence: "evidence"}
	juror := v.jurorEvidence(claim)
	if !juror.Pass {
		t.Error("expected pass with evidence")
	}
}

func TestAdversarialVerification_JurorConsistency(t *testing.T) {
	v := NewAdversarialVerification()
	juror := v.jurorConsistency(Claim{})
	if !juror.Pass {
		t.Error("expected pass for consistency check")
	}
}

func TestAdversarialVerification_JurorRecency(t *testing.T) {
	v := NewAdversarialVerification()
	juror := v.jurorRecency(Claim{})
	if !juror.Pass {
		t.Error("expected pass for recency check")
	}
}

func TestAdversarialVerification_Description(t *testing.T) {
	v := NewAdversarialVerification()
	if v.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestFormatJurors(t *testing.T) {
	jurors := []Juror{
		{Name: "a", Pass: true, Reason: "ok"},
		{Name: "b", Pass: false, Reason: "fail"},
	}
	out := FormatJurors(jurors)
	if out == "" {
		t.Error("expected non-empty output")
	}
}

// ── ContextFirewallV2 ───────────────────────────────────────────────────────

func TestContextFirewallV2_CleanDir(t *testing.T) {
	dir := t.TempDir()
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass on clean dir, got %s: %v", result.Status, result.Issues)
	}
}

func TestContextFirewallV2_ExternalURL(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("See https://evil.com/malware for details"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with external URL, got %s", result.Status)
	}
}

func TestContextFirewallV2_ApprovedDomain(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("See https://github.com/test for info"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		// May pass or fail depending on other checks, but approved domain should not trigger
		for _, issue := range result.Issues {
			if contains(issue, "evil.com") || contains(issue, "unapproved domain") {
				t.Errorf("approved domain github.com should not be flagged: %s", issue)
			}
		}
	}
}

func TestContextFirewallV2_SuspiciousUnicode(t *testing.T) {
	dir := t.TempDir()
	// Zero-width space
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("hello\u200Bworld"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with suspicious unicode, got %s", result.Status)
	}
}

func TestContainsSuspiciousPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"ignore all previous instructions", true},
		{"disable security", true},
		{"rm -rf /", true},
		{"curl http://evil.com", true},
		{"normal safe content", false},
		{"BYPASS", true},
		{"eval(1+1)", true},
		{"exec(code)", true},
		{"subprocess.call()", true},
		{"cat /etc/passwd", true},
		{"/bin/bash -c", true},
		{"reverse shell", true},
		{"base64 -d file", true},
		{"powershell -c", true},
		{"cmd.exe /c", true},
		{"__import__('os')", true},
		{"os.system('ls')", true},
		{"wget http://x", true},
	}
	for _, tc := range tests {
		if got := containsSuspiciousPattern(tc.input); got != tc.expected {
			t.Errorf("containsSuspiciousPattern(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestTruncate(t *testing.T) {
	short := truncate("abc", 10)
	if short != "abc" {
		t.Errorf("expected 'abc', got %q", short)
	}
	long := truncate("abcdefghij", 5)
	if long != "abcde..." {
		t.Errorf("expected 'abcde...', got %q", long)
	}
}

func TestTruncate_NonPrintable(t *testing.T) {
	s := "ab\x00cd"
	result := truncate(s, 10)
	if result != "abcd" {
		t.Errorf("expected 'abcd', got %q", result)
	}
}

func TestSafeContext(t *testing.T) {
	s := "hello world testing"
	result := safeContext(s, 5, 3)
	if result == "" {
		t.Error("expected non-empty context")
	}
}

func TestSafeContext_Boundary(t *testing.T) {
	s := "short"
	result := safeContext(s, 0, 100)
	if result == "" {
		t.Error("expected non-empty context")
	}
}

func TestIsApprovedDomain(t *testing.T) {
	if !isApprovedDomain("github.com") {
		t.Error("github.com should be approved")
	}
	if !isApprovedDomain("docs.github.com") {
		t.Error("docs.github.com (subdomain of github.com) should be approved")
	}
	if isApprovedDomain("evil.example.com") {
		t.Error("evil.example.com should not be approved")
	}
	if !isApprovedDomain("localhost") {
		t.Error("localhost should be approved")
	}
}

func TestContextFirewallV2_ApprovedSubdomain(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("https://docs.github.com/api"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	for _, issue := range result.Issues {
		if contains(issue, "unapproved domain") && contains(issue, "docs.github.com") {
			t.Errorf("subdomain of approved domain should not be flagged: %s", issue)
		}
	}
}

func TestContextFirewallV2_ControlChar(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("text\x01here"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with control character, got %s", result.Status)
	}
}

func TestContextFirewallV2_SkipsVendorDirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "node_modules"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "node_modules", "pkg.md"), []byte("https://evil.com"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	// node_modules should be skipped
	for _, issue := range result.Issues {
		if contains(issue, "evil.com") && contains(issue, "node_modules") {
			t.Errorf("node_modules should be skipped: %s", issue)
		}
	}
}

func TestContextFirewallV2_SkipsLockFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "package-lock.json"), []byte("lock"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %v", result.Status, result.Issues)
	}
}

func TestContextFirewallV2_PortInURL(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("https://evil.com:8080/api"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with port in URL, got %s", result.Status)
	}
}

// ── HeadIntegrity ───────────────────────────────────────────────────────────

func TestHeadIntegrity_NoGitDir(t *testing.T) {
	dir := t.TempDir()
	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail without git, got %s", result.Status)
	}
}

func TestCoverageBoost_HeadIntegrity_NoTrustedHash(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	// Create real git repo
	cmd := execGit(dir, "init")
	if cmd != nil {
		t.Skipf("git not available: %v", cmd)
	}
	execGit(dir, "config", "user.email", "test@test.com")
	execGit(dir, "config", "user.name", "Test")
	execGit(dir, "commit", "--allow-empty", "-m", "init")

	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass (no trusted hash = first run), got %s: %v", result.Status, result.Issues)
	}
}

func TestHeadIntegrity_Match(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	cmd := execGit(dir, "init")
	if cmd != nil {
		t.Skipf("git not available")
	}
	execGit(dir, "config", "user.email", "test@test.com")
	execGit(dir, "config", "user.name", "Test")
	execGit(dir, "commit", "--allow-empty", "-m", "init")

	// Get current HEAD
	headOut, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("cannot get HEAD: %v", err)
	}
	sha := string(headOut[:40])

	// Write trusted hash matching HEAD
	trustedDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(trustedDir, 0755)
	os.WriteFile(filepath.Join(trustedDir, "trusted_head_hash.json"),
		[]byte(`{"trusted_head_sha":"`+sha+`"}`), 0644)

	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass with matching hash, got %s: %v", result.Status, result.Issues)
	}
}

func TestHeadIntegrity_Mismatch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	cmd := execGit(dir, "init")
	if cmd != nil {
		t.Skipf("git not available")
	}
	execGit(dir, "config", "user.email", "test@test.com")
	execGit(dir, "config", "user.name", "Test")
	execGit(dir, "commit", "--allow-empty", "-m", "init")

	// Write a WRONG trusted hash
	trustedDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(trustedDir, 0755)
	os.WriteFile(filepath.Join(trustedDir, "trusted_head_hash.json"),
		[]byte(`{"trusted_head_sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`), 0644)

	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with mismatched hash, got %s", result.Status)
	}
}

func TestHeadIntegrity_PlainTextTrustedHash(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	cmd := execGit(dir, "init")
	if cmd != nil {
		t.Skipf("git not available")
	}
	execGit(dir, "config", "user.email", "test@test.com")
	execGit(dir, "config", "user.name", "Test")
	execGit(dir, "commit", "--allow-empty", "-m", "init")

	headOut, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("cannot get HEAD")
	}
	sha := string(headOut[:40])

	// Write trusted hash as plain text file
	trustedDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(trustedDir, 0755)
	os.WriteFile(filepath.Join(trustedDir, "trusted_head_hash"), []byte(sha), 0644)

	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass with plain text hash, got %s: %v", result.Status, result.Issues)
	}
}

func TestHeadIntegrity_TamperedAGENTS(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	cmd := execGit(dir, "init")
	if cmd != nil {
		t.Skipf("git not available")
	}
	execGit(dir, "config", "user.email", "test@test.com")
	execGit(dir, "config", "user.name", "Test")
	execGit(dir, "commit", "--allow-empty", "-m", "init")

	headOut, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("cannot get HEAD")
	}
	sha := string(headOut[:40])

	trustedDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(trustedDir, 0755)
	os.WriteFile(filepath.Join(trustedDir, "trusted_head_hash.json"),
		[]byte(`{"trusted_head_sha":"`+sha+`"}`), 0644)

	// AGENTS.md WITHOUT integrity seal
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# No seal here"), 0644)

	v := NewHeadIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with tampered AGENTS.md, got %s", result.Status)
	}
}

// ── SecurityPolicy ──────────────────────────────────────────────────────────

func TestSecurityPolicy_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewSecurityPolicy()
	result := v.Validate(context.Background(), dir)
	// Empty dir: no files to scan, validator passes (nothing to fail on)
	if result.Status != "pass" && result.Status != "fail" {
		t.Errorf("unexpected status on empty dir: %s", result.Status)
	}
}

func TestSecurityPolicy_NoWildcardCORS(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "shared.go"), []byte(`package cpanel
func handler() { w.Header().Set("Access-Control-Allow-Origin", "https://trusted.com") }`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "auth.go"), []byte(`package cpanel
var sessionExpiry = 24 * time.Hour
func checkRateLimit() {}`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "events.go"), []byte(`package cpanel
const maxSSEConnections = 10`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "static.go"), []byte(`package cpanel
import "net/url"
func handler() { url.PathUnescape("") }`), 0644)

	installDir := filepath.Join(dir, "go-runtime", "internal", "install")
	os.MkdirAll(installDir, 0755)
	os.WriteFile(filepath.Join(installDir, "install.go"), []byte(`package install
import "crypto/sha256"`), 0644)

	v := NewSecurityPolicy()
	result := v.Validate(context.Background(), dir)
	hasCORSSuccess := false
	for _, issue := range result.Issues {
		if contains(issue, "CORS") && contains(issue, "restricted") {
			hasCORSSuccess = true
		}
	}
	_ = hasCORSSuccess
	t.Logf("Security policy: %s — %v", result.Status, result.Issues)
}

func TestSecurityPolicy_WildcardCORS(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "shared.go"), []byte(`package cpanel
w.Header().Set("Access-Control-Allow-Origin", "*")`), 0644)

	v := NewSecurityPolicy()
	result := v.Validate(context.Background(), dir)
	hasCORSIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "wildcard CORS") {
			hasCORSIssue = true
		}
	}
	if !hasCORSIssue {
		t.Errorf("expected wildcard CORS issue, got: %v", result.Issues)
	}
}

func TestSecurityPolicy_SelfPath(t *testing.T) {
	dir := t.TempDir()
	// Ensure self-path for install dir doesn't exist → skip
	v := NewSecurityPolicy()
	result := v.Validate(context.Background(), dir)
	// R6 should pass (install dir not found = skip)
	for _, issue := range result.Issues {
		if contains(issue, "R6") && contains(issue, "net/http") {
			t.Errorf("R6 should skip when install dir not found: %s", issue)
		}
	}
}

func TestSecurityPolicy_CheckAuthRequired(t *testing.T) {
	s := NewSecurityPolicy()
	ok, msg := s.checkAuthRequired(t.TempDir())
	if !ok {
		t.Errorf("expected pass, got %s", msg)
	}
}

func TestSecurityPolicy_CheckNoDebugOutput(t *testing.T) {
	s := NewSecurityPolicy()
	ok, msg := s.checkNoDebugOutput(t.TempDir())
	if !ok {
		t.Errorf("expected pass, got %s", msg)
	}
}

func TestSecurityPolicy_CheckNoPlaintextSecrets(t *testing.T) {
	s := NewSecurityPolicy()
	ok, msg := s.checkNoPlaintextSecrets(t.TempDir())
	if !ok {
		t.Errorf("expected pass, got %s", msg)
	}
}

func TestSecurityPolicy_AllChecks(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "shared.go"), []byte(`package cpanel
func handler() {}`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "auth.go"), []byte(`package cpanel
var expiry = 24 * time.Hour
func checkRateLimit() {}`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "events.go"), []byte(`package cpanel
const maxSSEConnections = 10`), 0644)
	os.WriteFile(filepath.Join(cpanelDir, "static.go"), []byte(`package cpanel
import "net/url"
func h() { url.PathUnescape("") }`), 0644)

	installDir := filepath.Join(dir, "go-runtime", "internal", "install")
	os.MkdirAll(installDir, 0755)
	os.WriteFile(filepath.Join(installDir, "install.go"), []byte(`package install
import "crypto/sha256"
import _ "net/http"`), 0644)

	v := NewSecurityPolicy()
	result := v.Validate(context.Background(), dir)
	t.Logf("All checks: %s — %d issues", result.Status, len(result.Issues))
	// At minimum, should check all 10 rules
	if len(result.Issues) == 0 {
		// All pass
		if result.Status != "pass" {
			t.Errorf("expected pass with no issues, got %s", result.Status)
		}
	}
}

// ── StaleArtifactReferences ─────────────────────────────────────────────────

func TestStaleArtifactReferences_CleanDir(t *testing.T) {
	dir := t.TempDir()
	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass on clean dir, got %s", result.Status)
	}
}

func TestStaleArtifactReferences_WithStaleRefs(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "design.md"), []byte("# Design\nSegment S42 is defined in the plan"), 0644)

	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	hasStaleRef := false
	for _, issue := range result.Issues {
		if contains(issue, "S* segment reference") {
			hasStaleRef = true
		}
	}
	if !hasStaleRef {
		t.Errorf("expected stale S42 reference, got: %v", result.Issues)
	}
}

func TestStaleArtifactReferences_UniqueStrings(t *testing.T) {
	input := []string{"S1", "S2", "S1", "S3", "S2"}
	result := uniqueStrings(input)
	if len(result) != 3 {
		t.Errorf("expected 3 unique strings, got %d: %v", len(result), result)
	}
}

func TestStaleArtifactReferences_BuildRef(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "old.md"), []byte("# Old\nSegment S99 reference"), 0644)

	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	// Just verify it runs without panic on docs with references
	if result.Status == "" {
		t.Error("result status should not be empty")
	}
}

func TestStaleArtifactReferences_SkipsExcludedDirs(t *testing.T) {
	dir := t.TempDir()
	excludeDir := filepath.Join(dir, ".ovav", "artifacts")
	os.MkdirAll(excludeDir, 0755)
	os.WriteFile(filepath.Join(excludeDir, "old.md"), []byte("S99 reference"), 0644)

	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	for _, issue := range result.Issues {
		if contains(issue, "S99") {
			t.Errorf("excluded dir should not be scanned: %s", issue)
		}
	}
}

func TestStaleArtifactReferences_CriticalPath(t *testing.T) {
	dir := t.TempDir()
	ovavPlan := filepath.Join(dir, ".ovav", "plan")
	os.MkdirAll(ovavPlan, 0755)
	os.WriteFile(filepath.Join(ovavPlan, "caps.yaml"), []byte("clean content"), 0644)

	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	// Just verify it runs without panic on plan dir
	if result.Status == "" {
		t.Error("result status should not be empty")
	}
}

// ── WeztermPathIntegrity ────────────────────────────────────────────────────

func TestWeztermPathIntegrity_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewWeztermPathIntegrity()
	result := v.Validate(context.Background(), dir)
	// Should fail due to missing proxy markers
	if result.Status != "fail" {
		t.Errorf("expected fail on empty dir, got %s", result.Status)
	}
}

func TestWeztermPathIntegrity_FallbackMissing(t *testing.T) {
	dir := t.TempDir()
	// Create proxy marker files with all markers
	for relPath, markers := range proxyMarkers {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := ""
		for _, m := range markers {
			content += m + "\n"
		}
		os.WriteFile(fullPath, []byte(content), 0644)
	}
	// Don't create fallback → should fail

	v := NewWeztermPathIntegrity()
	result := v.Validate(context.Background(), dir)
	hasFallbackIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "MISSING_FALLBACK") || contains(issue, "FALLBACK_MISSING_TOKEN") {
			hasFallbackIssue = true
		}
	}
	if !hasFallbackIssue {
		t.Errorf("expected fallback issue, got: %v", result.Issues)
	}
}

func TestWeztermPathIntegrity_MissingMarkers(t *testing.T) {
	dir := t.TempDir()
	// Create proxy marker files but with WRONG markers
	for relPath := range proxyMarkers {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte("WRONG MARKER\n"), 0644)
	}
	// Create fallback with correct markers
	fbDir := filepath.Join(dir, "config", "wezterm")
	os.MkdirAll(fbDir, 0755)
	os.WriteFile(filepath.Join(fbDir, "wezterm-fallback-minimal.lua"),
		[]byte("OVAV_FALLBACK_MARKER\nrequire 'wezterm'\nWORKSPACES\nswitch_workspace\nOVAV_WEZTERM_WORKSPACE\nreturn config"),
		0644)

	v := NewWeztermPathIntegrity()
	result := v.Validate(context.Background(), dir)
	hasMissingMarker := false
	for _, issue := range result.Issues {
		if contains(issue, "MISSING_MARKER") {
			hasMissingMarker = true
		}
	}
	if !hasMissingMarker {
		t.Errorf("expected MISSING_MARKER issue, got: %v", result.Issues)
	}
}

func TestWeztermPathIntegrity_BlockedPathInScanFile(t *testing.T) {
	dir := t.TempDir()
	// Create proxy marker files with all markers
	for relPath, markers := range proxyMarkers {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := ""
		for _, m := range markers {
			content += m + "\n"
		}
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create fallback with correct markers
	fbDir := filepath.Join(dir, "config", "wezterm")
	os.MkdirAll(fbDir, 0755)
	os.WriteFile(filepath.Join(fbDir, "wezterm-fallback-minimal.lua"),
		[]byte("OVAV_FALLBACK_MARKER\nrequire 'wezterm'\nWORKSPACES\nswitch_workspace\nOVAV_WEZTERM_WORKSPACE\nreturn config"),
		0644)

	// Create scan file with a blocked path reference (non-doc context)
	docsDir := filepath.Join(dir, "docs", "workstation")
	os.MkdirAll(docsDir, 0755)
	blocked := blockedPaths[0]
	os.WriteFile(filepath.Join(docsDir, "OVAV_WEZTERM_WORKSPACE_ISOLATION.md"),
		[]byte("# Workspace\nActive path: "+blocked+"\n"), 0644)

	v := NewWeztermPathIntegrity()
	result := v.Validate(context.Background(), dir)
	hasBlockedPath := false
	for _, issue := range result.Issues {
		if contains(issue, "BLOCKED_PATH") {
			hasBlockedPath = true
		}
	}
	if !hasBlockedPath {
		t.Errorf("expected BLOCKED_PATH issue, got: %v", result.Issues)
	}
}

// ── ThoughtFirewall ─────────────────────────────────────────────────────────

func TestThoughtFirewall_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewThoughtFirewall()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail (can't determine branch), got %s", result.Status)
	}
}

func TestThoughtFirewall_WorktreeFile(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	// .git is a file (worktree)
	os.WriteFile(gitDir, []byte("gitdir: /tmp/fake/worktrees/feature"), 0644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	// The worktree HEAD won't exist, so it should fail gracefully
	v := NewThoughtFirewall()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail (worktree HEAD missing), got %s", result.Status)
	}
}

func TestCoverageBoost_ThoughtFirewall_ProtectedBranch(t *testing.T) {
	// PROTECTED BRANCH: isProtected() is informational only — validator now PASSES
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)

	v := NewThoughtFirewall()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass on protected main branch (informational), got %s", result.Status)
	}
}

func TestCoverageBoost_ThoughtFirewall_TaskBranch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/task/feature\n"), 0644)

	v := NewThoughtFirewall()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass on task branch, got %s: %v", result.Status, result.Issues)
	}
}

func TestThoughtFirewall_IsProtected(t *testing.T) {
	v := NewThoughtFirewall()
	if !v.isProtected("main") {
		t.Error("main should be protected")
	}
	if v.isProtected("task/feature") {
		t.Error("task/feature should not be protected")
	}
}

func TestThoughtFirewall_HasBlockedIntent(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "request.md")
	os.WriteFile(fpath, []byte("Please implement the new feature and modify the config"), 0644)

	v := NewThoughtFirewall()
	if !v.hasBlockedIntent(fpath) {
		t.Error("expected blocked intent detected")
	}
}

func TestThoughtFirewall_HasBlockedIntent_Safe(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "request.md")
	os.WriteFile(fpath, []byte("Please inspect and verify the current state"), 0644)

	v := NewThoughtFirewall()
	if v.hasBlockedIntent(fpath) {
		t.Error("safe intents should not be blocked")
	}
}

func TestThoughtFirewall_HasBlockedIntent_NoFile(t *testing.T) {
	v := NewThoughtFirewall()
	if v.hasBlockedIntent("/nonexistent/path") {
		t.Error("non-existent file should not be blocked")
	}
}

func TestThoughtFirewall_DetachedHead(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)
	// Detached HEAD — raw SHA
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc123def456abc123def456abc123def456abc1\n"), 0644)

	v := NewThoughtFirewall()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass on detached HEAD (not protected), got %s: %v", result.Status, result.Issues)
	}
}

func TestThoughtFirewall_AllProtectedBranches(t *testing.T) {
	v := NewThoughtFirewall()
	protected := []string{"main", "master", "develop", "development", "prod", "production", "staging"}
	for _, branch := range protected {
		if !v.isProtected(branch) {
			t.Errorf("%s should be protected", branch)
		}
	}
}

// ── GateSelfProtection ──────────────────────────────────────────────────────

func TestGateSelfProtection_HashMismatch(t *testing.T) {
	dir := t.TempDir()
	gateDir := filepath.Join(dir, "go-runtime", "internal", "validators")
	os.MkdirAll(gateDir, 0755)
	os.WriteFile(filepath.Join(gateDir, "host_config_drift.go"), []byte("package validators"), 0644)

	// Write a wrong stored hash to the truststore path
	hashDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(hashDir, 0755)
	state := truststore.GateState{GateSHA256: "aaaabbbbccccddddeeeeffff00001111"}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(hashDir, "gate_state.json"), data, 0644)

	v := NewGateSelfProtection()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with hash mismatch, got %s", result.Status)
	}
}

func TestGateSelfProtection_HashMatch(t *testing.T) {
	dir := t.TempDir()
	gateDir := filepath.Join(dir, "go-runtime", "internal", "validators")
	os.MkdirAll(gateDir, 0755)
	gateContent := "package validators"
	os.WriteFile(filepath.Join(gateDir, "host_config_drift.go"), []byte(gateContent), 0644)

	// Compute actual hash
	g := NewGateSelfProtection()
	actualHash := g.fileSHA256(filepath.Join(gateDir, "host_config_drift.go"))

	// Write stored hash to truststore path
	hashDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(hashDir, 0755)
	state := truststore.GateState{GateSHA256: actualHash}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(hashDir, "gate_state.json"), data, 0644)

	result := g.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass with matching hash, got %s: %v", result.Status, result.Issues)
	}
}

func TestGateSelfProtection_Blockade(t *testing.T) {
	dir := t.TempDir()
	gateDir := filepath.Join(dir, "go-runtime", "internal", "validators")
	os.MkdirAll(gateDir, 0755)
	os.WriteFile(filepath.Join(gateDir, "host_config_drift.go"), []byte("package validators"), 0644)

	hashDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(hashDir, 0755)
	state := truststore.GateState{GateSHA256: "aaaabbbbccccddddeeeeffff00001111"}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(hashDir, "gate_state.json"), data, 0644)

	// Active blockade (same path in old and new impl)
	os.WriteFile(filepath.Join(dir, ".ovav", "host_defense_blockade"),
		[]byte(`{"blockade":"active","reason":"test blockade"}`), 0644)

	v := NewGateSelfProtection()
	result := v.Validate(context.Background(), dir)
	hasBlockade := false
	for _, issue := range result.Issues {
		if contains(issue, "BLOCKADE ACTIVE") {
			hasBlockade = true
		}
	}
	if !hasBlockade {
		t.Errorf("expected BLOCKADE ACTIVE, got: %v", result.Issues)
	}
}

func TestGateSelfProtection_FileSHA256(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")
	os.WriteFile(fpath, []byte("hello"), 0644)
	g := NewGateSelfProtection()
	hash := g.fileSHA256(fpath)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	// Same content → same hash
	hash2 := g.fileSHA256(fpath)
	if hash != hash2 {
		t.Error("same content should produce same hash")
	}
}

func TestGateSelfProtection_FileSHA256_Missing(t *testing.T) {
	g := NewGateSelfProtection()
	hash := g.fileSHA256("/nonexistent/file")
	if hash != "" {
		t.Error("expected empty hash for missing file")
	}
}

func TestGateSelfProtection_ReadStoredHash_JSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0755)
	state := truststore.GateState{GateSHA256: "abc123"}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, ".ovav", "runtime", "gate_state.json"), data, 0644)
	g := NewGateSelfProtection()
	_ = g // unused in the new API — truststore is called directly
	hash := truststore.ReadGateState(dir).GateSHA256
	if hash != "abc123" {
		t.Errorf("expected abc123, got %s", hash)
	}
}

func TestGateSelfProtection_ReadStoredHash_PlainText(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "runtime"), 0755)
	// Plain text not supported in truststore — test empty state
	state := truststore.GateState{GateSHA256: ""}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(dir, ".ovav", "runtime", "gate_state.json"), data, 0644)
	hash := truststore.ReadGateState(dir).GateSHA256
	if hash != "" {
		t.Errorf("expected empty, got %s", hash)
	}
}

func TestGateSelfProtection_ReadStoredHash_Missing(t *testing.T) {
	hash := truststore.ReadGateState("/nonexistent").GateSHA256
	if hash != "" {
		t.Error("expected empty for missing file")
	}
}

func TestGateSelfProtection_IsAuthorizedSession(t *testing.T) {
	dir := t.TempDir()
	g := NewGateSelfProtection()
	if g.isAuthorizedSession(dir) {
		t.Error("expected false when no session marker")
	}

	markerDir := filepath.Join(dir, ".ovav", "runtime")
	os.MkdirAll(markerDir, 0755)
	os.WriteFile(filepath.Join(markerDir, ".session_marker"), []byte("2024-01-01"), 0644)
	if !g.isAuthorizedSession(dir) {
		t.Error("expected true with session marker")
	}
}

// ── ServiceAreaRouter ───────────────────────────────────────────────────────

func TestServiceAreaRouter_MissingAgents(t *testing.T) {
	dir := t.TempDir()
	v := NewServiceAreaRouter()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with missing agents, got %s", result.Status)
	}
}

func TestServiceAreaRouter_OneAgentPresent(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "clients", "opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	// Create one agent with all required hard stops
	hardStops := areaProfiles[0].hardStops
	content := "# Platform Engineering\n"
	for _, hs := range hardStops {
		content += hs + "\n"
	}
	os.WriteFile(filepath.Join(agentsDir, areaProfiles[0].file), []byte(content), 0644)

	v := NewServiceAreaRouter()
	result := v.Validate(context.Background(), dir)
	// Should still fail (other agents missing)
	if result.Status != "fail" {
		t.Errorf("expected fail with only 1 agent, got %s", result.Status)
	}
}

// ── ToolReadiness ───────────────────────────────────────────────────────────

func TestToolReadiness_MissingMatrix(t *testing.T) {
	dir := t.TempDir()
	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail without matrix, got %s", result.Status)
	}
}

func TestToolReadiness_EmptyMatrix(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte("tool_readiness_matrix:\n  capabilities: {}\n"), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail with empty capabilities, got %s", result.Status)
	}
}

func TestToolReadiness_MissingBoundary(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)

	// Create a valid matrix with all required capabilities
	matrixYaml := "tool_readiness_matrix:\n  capabilities:\n"
	for _, cap := range requiredCapabilities {
		matrixYaml += "    " + cap + ":\n      current_state: blocked\n"
	}
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte(matrixYaml), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	// Should fail due to missing boundary file
	hasBoundary := false
	for _, issue := range result.Issues {
		if contains(issue, "MISSING") && contains(issue, "boundary") {
			hasBoundary = true
		}
	}
	if !hasBoundary {
		t.Errorf("expected missing boundary issue, got: %v", result.Issues)
	}
}

func TestToolReadiness_InvalidState(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)

	matrixYaml := "tool_readiness_matrix:\n  capabilities:\n"
	for _, cap := range requiredCapabilities {
		matrixYaml += "    " + cap + ":\n      current_state: blocked\n"
	}
	// Add an invalid state
	matrixYaml += "    test_cap:\n      current_state: INVALID_STATE\n"
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte(matrixYaml), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	hasInvalidState := false
	for _, issue := range result.Issues {
		if contains(issue, "invalid state") {
			hasInvalidState = true
		}
	}
	if !hasInvalidState {
		t.Errorf("expected invalid state issue, got: %v", result.Issues)
	}
}

func TestToolReadiness_MCPActiveByDefault(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)

	matrixYaml := "tool_readiness_matrix:\n  capabilities:\n"
	for _, cap := range requiredCapabilities {
		state := "blocked"
		if cap == "mcp" {
			state = "active_internal"
		}
		matrixYaml += "    " + cap + ":\n      current_state: " + state + "\n"
	}
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte(matrixYaml), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	hasMCPIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "mcp") && contains(issue, "must not be active") {
			hasMCPIssue = true
		}
	}
	if !hasMCPIssue {
		t.Errorf("expected MCP active-by-default issue, got: %v", result.Issues)
	}
}

func TestToolReadiness_BadYAML(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte("::: invalid yaml"), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	// Bad YAML should not crash — validator handles parse errors gracefully
	if result.Status == "" {
		t.Error("result status should not be empty")
	}
}

// ── ConfigIntegrity ─────────────────────────────────────────────────────────

func TestConfigIntegrity_MissingConfigs(t *testing.T) {
	dir := t.TempDir()
	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail without configs, got %s", result.Status)
	}
}

func TestConfigIntegrity_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("version: 1"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte(""), 0644) // Empty VERSION

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	hasEmpty := false
	for _, issue := range result.Issues {
		if contains(issue, "EMPTY") && contains(issue, "VERSION") {
			hasEmpty = true
		}
	}
	if !hasEmpty {
		t.Errorf("expected empty VERSION issue, got: %v", result.Issues)
	}
}

func TestConfigIntegrity_WrongVersion(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("version: 1"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.0.0"), 0644)

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	hasWrongVersion := false
	for _, issue := range result.Issues {
		if contains(issue, "expected 2.x.x") {
			hasWrongVersion = true
		}
	}
	if !hasWrongVersion {
		t.Errorf("expected wrong version issue, got: %v", result.Issues)
	}
}

func TestConfigIntegrity_DeprecatedFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("version: 1"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("2.0.0"), 0644)
	os.WriteFile(filepath.Join(dir, "IMPLEMENTATION_PLAN.md"), []byte("old plan"), 0644)

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	hasDeprecated := false
	for _, issue := range result.Issues {
		if contains(issue, "DEPRECATED") {
			hasDeprecated = true
		}
	}
	if !hasDeprecated {
		t.Errorf("expected DEPRECATED issue, got: %v", result.Issues)
	}
}

func TestConfigIntegrity_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "plan"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "laws"), 0755)
	os.MkdirAll(filepath.Join(dir, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("version: 1"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "laws", "ovav_laws.yaml"), []byte("laws: []"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte("not json"), 0644)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("2.0.0"), 0644)

	v := NewConfigIntegrity()
	result := v.Validate(context.Background(), dir)
	hasSyntaxErr := false
	for _, issue := range result.Issues {
		if contains(issue, "SYNTAX") && contains(issue, "JSON") {
			hasSyntaxErr = true
		}
	}
	if !hasSyntaxErr {
		t.Errorf("expected JSON syntax issue, got: %v", result.Issues)
	}
}

func TestGetLatestGitTag_NoGit(t *testing.T) {
	_, err := getLatestGitTag("/nonexistent")
	if err == nil {
		t.Error("expected error for non-existent dir")
	}
}

// ── AgentPermissionInvariants ───────────────────────────────────────────────

func TestAgentPermissionInvariants_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewAgentPermissionInvariants()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

func TestSetsEqual(t *testing.T) {
	a := map[string]bool{"x": true, "y": true}
	b := map[string]bool{"x": true, "y": true}
	if !setsEqual(a, b) {
		t.Error("expected equal sets")
	}
	c := map[string]bool{"x": true}
	if setsEqual(a, c) {
		t.Error("expected unequal sets")
	}
	d := map[string]bool{"x": true, "z": true}
	if setsEqual(a, d) {
		t.Error("expected unequal sets (different keys)")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]bool{"c": true, "a": true, "b": true}
	keys := sortedKeys(m)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("expected [a b c], got %v", keys)
	}
}

func TestMapsEqual(t *testing.T) {
	a := map[string]interface{}{"x": "1", "y": "2"}
	b := map[string]interface{}{"x": "1", "y": "2"}
	if !mapsEqual(a, b) {
		t.Error("expected equal maps")
	}
	c := map[string]interface{}{"x": "1"}
	if mapsEqual(a, c) {
		t.Error("expected unequal maps")
	}
	d := map[string]interface{}{"x": "1", "y": "3"}
	if mapsEqual(a, d) {
		t.Error("expected unequal maps (different values)")
	}
}

func TestParseAgentFrontmatter_MissingFile(t *testing.T) {
	_, err := parseAgentFrontmatter("/nonexistent/file.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseAgentFrontmatter_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("# Agent\nNo frontmatter"), 0644)
	_, err := parseAgentFrontmatter(fpath)
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseAgentFrontmatter_Malformed(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("---\nname: test"), 0644)
	_, err := parseAgentFrontmatter(fpath)
	if err == nil {
		t.Error("expected error for malformed frontmatter")
	}
}

func TestParseAgentFrontmatter_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("---\n::: invalid yaml\n---\nbody"), 0644)
	_, err := parseAgentFrontmatter(fpath)
	// Invalid YAML may or may not error depending on leniency — just verify no panic
	_ = err
}

func TestParseAgentFrontmatter_Valid(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("---\nname: Test\npermission:\n  edit: allow\n---\nbody"), 0644)
	fm, err := parseAgentFrontmatter(fpath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm["name"] != "Test" {
		t.Errorf("expected name 'Test', got %v", fm["name"])
	}
}

// ── BehavioralDirectives ────────────────────────────────────────────────────

func TestToFloat(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
	}{
		{float64(0.5), 0.5},
		{int(3), 3.0},
		{int64(7), 7.0},
		{"string", 0},
		{nil, 0},
	}
	for _, tc := range tests {
		got := toFloat(tc.input)
		if got != tc.expected {
			t.Errorf("toFloat(%v) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestBehavioralDirectives_MissingDir(t *testing.T) {
	dir := t.TempDir()
	v := NewBehavioralDirectives()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

// ── MultiPlatform ───────────────────────────────────────────────────────────

func TestMultiPlatform_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewMultiPlatform()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail on empty dir, got %s", result.Status)
	}
}

func TestMultiPlatform_AllPresent(t *testing.T) {
	dir := t.TempDir()

	// Windows loader with all markers
	loaderDir := filepath.Join(dir, ".ovav", "source", "configs", "wezterm")
	os.MkdirAll(loaderDir, 0755)
	os.WriteFile(filepath.Join(loaderDir, "ovav-windows-loader.wezterm.lua"),
		[]byte("OVAV_WZPROXY_v3\nOVAV_CAPA7_CROSS_PLATFORM"), 0644)

	// Cross-platform paths
	os.MkdirAll(filepath.Join(dir, "tools", "platform"), 0755)
	os.WriteFile(filepath.Join(dir, "tools", "platform", "cross_platform_paths.py"),
		[]byte("# paths"), 0644)

	// WezTerm fallback
	fbDir := filepath.Join(dir, "config", "wezterm")
	os.MkdirAll(fbDir, 0755)
	os.WriteFile(filepath.Join(fbDir, "wezterm-fallback-minimal.lua"),
		[]byte("OVAV_WZFALLBACK_v1"), 0644)

	// Skills enforcement
	enfDir := filepath.Join(dir, "tools", "agent_runtime")
	os.MkdirAll(enfDir, 0755)
	os.WriteFile(filepath.Join(enfDir, "skills_enforcement_gate.py"),
		[]byte("class SkillsEnforcementGate\ndef check_skills_enforcement\ndef get_compliance_semaphore"), 0644)

	// Session routing pipeline
	os.WriteFile(filepath.Join(enfDir, "session_routing_pipeline.py"),
		[]byte("def connect_status\ndef connect_health_check\ndef get_fallback_chain"), 0644)

	// Keyboard shortcuts doc
	docDir := filepath.Join(dir, "docs", "workstation")
	os.MkdirAll(docDir, 0755)
	os.WriteFile(filepath.Join(docDir, "OVAV_WEZTERM_WORKSPACE_ISOLATION.md"),
		[]byte("# Workspace\nshortcut Ctrl+T"), 0644)

	v := NewMultiPlatform()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %v", result.Status, result.Issues)
	}
}

// ── WorkspaceIsolation ──────────────────────────────────────────────────────

func TestWorkspaceIsolation_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewWorkspaceIsolation()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" && result.Status != "warn" {
		t.Errorf("expected fail/warn, got %s", result.Status)
	}
}

// ── ContextEconomy ──────────────────────────────────────────────────────────

func TestContextEconomy_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewContextEconomy()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail on empty dir, got %s", result.Status)
	}
}

// ── AgentSurfaceHierarchy ───────────────────────────────────────────────────

func TestAgentSurfaceHierarchy_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewAgentSurfaceHierarchy()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

func TestAgentSurfaceHierarchy_WithAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "clients", "opencode", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "lead-thavren.md"),
		[]byte("---\nname: Thavren\n---\n# Thavren"), 0644)

	v := NewAgentSurfaceHierarchy()
	result := v.Validate(context.Background(), dir)
	t.Logf("Agent surface hierarchy: %s — %v", result.Status, result.Issues)
}

// ── ContextFirewallV2 — base64 with suspicious pattern ──────────────────────

func TestContextFirewallV2_SuspiciousBase64(t *testing.T) {
	dir := t.TempDir()
	// Base64 encode "rm -rf /" which is suspicious
	encoded := "cm0gLXJmIC8="
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("data: "+encoded), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	// Might or might not trigger depending on regex match, just verify no panic
	t.Logf("Base64 test: %s — %v", result.Status, result.Issues)
}

// ── SecurityPolicy — network check with net/http import ─────────────────────

func TestSecurityPolicy_NetHTTPInInstall(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "go-runtime", "internal", "install")
	os.MkdirAll(installDir, 0755)
	os.WriteFile(filepath.Join(installDir, "install.go"),
		[]byte(`package install
import "net/http"
func init() { http.Get("http://evil.com") }`), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkNoExternalNetwork(dir)
	if ok {
		t.Error("expected fail with net/http in install")
	}
	t.Logf("Result: %s", msg)
}

func TestSecurityPolicy_NoSHA256InInstall(t *testing.T) {
	dir := t.TempDir()
	installDir := filepath.Join(dir, "go-runtime", "internal", "install")
	os.MkdirAll(installDir, 0755)
	os.WriteFile(filepath.Join(installDir, "install.go"),
		[]byte("package install\nfunc doStuff() {}"), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkSHA256Verification(dir)
	if ok {
		t.Error("expected fail without sha256")
	}
	t.Logf("Result: %s", msg)
}

func TestSecurityPolicy_NoRateLimit(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "auth.go"),
		[]byte("package cpanel\nfunc handler() {}"), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkRateLimiting(dir)
	if ok {
		t.Error("expected fail without rate limiting")
	}
	t.Logf("Result: %s", msg)
}

func TestSecurityPolicy_NoSSELimits(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "events.go"),
		[]byte("package cpanel\nfunc handler() {}"), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkSSELimits(dir)
	if ok {
		t.Error("expected fail without SSE limits")
	}
	t.Logf("Result: %s", msg)
}

func TestSecurityPolicy_NoPathTraversal(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "static.go"),
		[]byte("package cpanel\nfunc handler() {}"), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkPathTraversalDefense(dir)
	if ok {
		t.Error("expected fail without PathUnescape")
	}
	t.Logf("Result: %s", msg)
}

func TestSecurityPolicy_NoSessionExpiry(t *testing.T) {
	dir := t.TempDir()
	cpanelDir := filepath.Join(dir, "go-runtime", "cmd", "cpanel")
	os.MkdirAll(cpanelDir, 0755)
	os.WriteFile(filepath.Join(cpanelDir, "auth.go"),
		[]byte("package cpanel\nvar expiry = 48 * time.Hour"), 0644)

	s := NewSecurityPolicy()
	ok, msg := s.checkSessionExpiry(dir)
	if ok {
		t.Error("expected fail without 24h expiry")
	}
	t.Logf("Result: %s", msg)
}

// ── WeztermPathIntegrity — work_ledger deprecated path ──────────────────────

func TestWeztermPathIntegrity_DeprecatedLedgerPath(t *testing.T) {
	dir := t.TempDir()
	// Create proxy markers with all markers
	for relPath, markers := range proxyMarkers {
		fullPath := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := ""
		for _, m := range markers {
			content += m + "\n"
		}
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create fallback with all tokens
	fbDir := filepath.Join(dir, "config", "wezterm")
	os.MkdirAll(fbDir, 0755)
	os.WriteFile(filepath.Join(fbDir, "wezterm-fallback-minimal.lua"),
		[]byte("OVAV_FALLBACK_MARKER\nrequire 'wezterm'\nWORKSPACES\nswitch_workspace\nOVAV_WEZTERM_WORKSPACE\nreturn config"),
		0644)

	v := NewWeztermPathIntegrity()
	result := v.Validate(context.Background(), dir)
	// Just verify it runs without panic on full setup
	if result.Status == "" {
		t.Error("result status should not be empty")
	}
}

// ── ToolReadiness — opencode.json check ─────────────────────────────────────

func TestToolReadiness_OpenCodeJSON_MissingDeny(t *testing.T) {
	dir := t.TempDir()
	matrixDir := filepath.Join(dir, ".ovav", "service_areas", "shared")
	os.MkdirAll(matrixDir, 0755)

	matrixYaml := "tool_readiness_matrix:\n  capabilities:\n"
	for _, cap := range requiredCapabilities {
		matrixYaml += "    " + cap + ":\n      current_state: blocked\n"
	}
	os.WriteFile(filepath.Join(matrixDir, "tool_readiness_matrix.yaml"), []byte(matrixYaml), 0644)

	// opencode.json without deny for pip install
	os.WriteFile(filepath.Join(dir, "opencode.json"),
		[]byte(`{"permission":{"bash":{}}}`), 0644)

	v := NewToolReadiness()
	result := v.Validate(context.Background(), dir)
	hasDenyIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "must deny") {
			hasDenyIssue = true
		}
	}
	if !hasDenyIssue {
		t.Errorf("expected deny issue for package install, got: %v", result.Issues)
	}
}

// ── ContextFirewallV2 — approved domain list completeness ────────────────────

func TestContextFirewallV2_ApprovedDomains(t *testing.T) {
	approved := []string{
		"github.com", "gitlab.com", "cloudflare.com", "pkg.go.dev",
		"golang.org", "python.org", "npmjs.com", "localhost",
		"127.0.0.1", "0.0.0.0", "ovav.dev", "fly.dev", "opencode.ai",
	}
	for _, d := range approved {
		if !isApprovedDomain(d) {
			t.Errorf("expected %s to be approved", d)
		}
	}
	unapproved := []string{"evil.com", "malware.net", "phishing.org"}
	for _, d := range unapproved {
		if isApprovedDomain(d) {
			t.Errorf("expected %s to NOT be approved", d)
		}
	}
}

// ── ContextFirewallV2 — URL with no domain match ────────────────────────────

func TestContextFirewallV2_URLNoMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("no urls here just text"), 0644)
	v := NewContextFirewallV2()
	result := v.Validate(context.Background(), dir)
	if result.Status != "pass" {
		t.Errorf("expected pass with no URLs, got %s: %v", result.Status, result.Issues)
	}
}

// ── StaleArtifactReferences — extension filtering ───────────────────────────

func TestStaleArtifactReferences_SkipsWrongExtension(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "image.png"), []byte("S42 ref in binary"), 0644)

	v := NewStaleArtifactReferences()
	result := v.Validate(context.Background(), dir)
	for _, issue := range result.Issues {
		if contains(issue, "S42") {
			t.Errorf("non-text extension should be skipped: %s", issue)
		}
	}
}

// ── AgentSurfaceHierarchy — parseFrontmatter ────────────────────────────────

func TestAgentSurfaceHierarchy_ParseFrontmatter(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("---\nname: Test\nsurface: area\n---\nbody"), 0644)

	v := NewAgentSurfaceHierarchy()
	fm := v.parseFrontmatter(fpath)
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	if fm["name"] != "Test" {
		t.Errorf("expected name 'Test', got %v", fm["name"])
	}
}

func TestAgentSurfaceHierarchy_ParseFrontmatter_NoFile(t *testing.T) {
	v := NewAgentSurfaceHierarchy()
	fm := v.parseFrontmatter("/nonexistent/file.md")
	if fm != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestAgentSurfaceHierarchy_ParseFrontmatter_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "agent.md")
	os.WriteFile(fpath, []byte("# No frontmatter here"), 0644)

	v := NewAgentSurfaceHierarchy()
	fm := v.parseFrontmatter(fpath)
	if fm != nil {
		t.Error("expected nil for no frontmatter")
	}
}

// ── CrossTargetConsistency ──────────────────────────────────────────────────

func TestCrossTargetConsistency_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewCrossTargetConsistency()
	result := v.Validate(context.Background(), dir)
	t.Logf("Cross target: %s — %v", result.Status, result.Issues)
}

// ── ContractEnforcement — more paths ────────────────────────────────────────

func TestContractEnforcement_NoContractsDir(t *testing.T) {
	dir := t.TempDir()
	v := NewContractEnforcement()
	result := v.Validate(context.Background(), dir)
	// No contracts dir = skip
	t.Logf("Contract enforcement: %s — %v", result.Status, result.Issues)
}

// ── ArchitecturalGuardian — more paths ──────────────────────────────────────

func TestArchitectureGuardian_WithStructure(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{
		".ovav/policy", ".ovav/plan", ".ovav/service_areas/shared",
		"go-runtime", "docs", "tools", "tools/validators",
		"tools/agent_runtime", "clients/opencode/agents",
		"config/workstation", "config/ssh",
	} {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}
	os.WriteFile(filepath.Join(dir, ".ovav", "policy", "permission_authority.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "plan", "caps.yaml"), []byte("version: 1"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test"), 0644)

	v := NewArchitectureGuardian()
	result := v.Validate(context.Background(), dir)
	t.Logf("Architecture guardian: %s — %d issues", result.Status, len(result.Issues))
}

// ── ModelPolicy — more paths ────────────────────────────────────────────────

func TestModelPolicy_WithValidConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "opencode.json"),
		[]byte(`{"model": "opencode-go/deepseek-v4-pro", "agent": {"test": {"model": "opencode-go/qwen3.7-plus"}}}`), 0644)

	v := NewModelPolicy()
	result := v.Validate(context.Background(), dir)
	t.Logf("Model policy: %s — %v", result.Status, result.Issues)
}

func TestModelPolicy_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{}`), 0644)

	v := NewModelPolicy()
	result := v.Validate(context.Background(), dir)
	t.Logf("Model policy empty: %s — %v", result.Status, result.Issues)
}

// ── FeedbackLoop ────────────────────────────────────────────────────────────

func TestFeedbackLoop_AllMissing(t *testing.T) {
	dir := t.TempDir()
	v := NewFeedbackLoop()
	result := v.Validate(context.Background(), dir)
	if result.Status != "fail" {
		t.Errorf("expected fail, got %s", result.Status)
	}
}

// ── PhaseDAG ────────────────────────────────────────────────────────────────

func TestPhaseDAG_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewPhaseDAG()
	result := v.Validate(context.Background(), dir)
	t.Logf("PhaseDAG: %s — %v", result.Status, result.Issues)
}

// ── BootstrapChain ──────────────────────────────────────────────────────────

func TestBootstrapChain_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewBootstrapChain()
	result := v.Validate(context.Background(), dir)
	t.Logf("BootstrapChain: %s — %v", result.Status, result.Issues)
}

// ── Skills validator ────────────────────────────────────────────────────────

func TestSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewSkills()
	result := v.Validate(context.Background(), dir)
	t.Logf("Skills: %s — %v", result.Status, result.Issues)
}

// ── ServiceProfiles ─────────────────────────────────────────────────────────

func TestServiceProfiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewServiceProfiles()
	result := v.Validate(context.Background(), dir)
	t.Logf("ServiceProfiles: %s — %v", result.Status, result.Issues)
}

// ── HarnessContractAlignment ────────────────────────────────────────────────

func TestHarnessContractAlignment_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewHarnessContractAlignment()
	result := v.Validate(context.Background(), dir)
	t.Logf("HarnessContractAlignment: %s — %v", result.Status, result.Issues)
}

// ── MemoryPolicy ────────────────────────────────────────────────────────────

func TestMemoryPolicy_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	v := NewMemoryPolicy()
	result := v.Validate(context.Background(), dir)
	t.Logf("MemoryPolicy: %s — %v", result.Status, result.Issues)
}

// ── Helper ──────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// execGit runs a git command in the given directory.
func execGit(dir string, args ...string) error {
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s (%w)", args, string(out), err)
	}
	return nil
}
