package ows

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ── Red Team: Compliance Gates ─────────────────────────────────────────────
// Verifies that all compliance gates actually block merges as expected.

// TestComplianceRequirements_StandardElevated verifies standard compliance elevation.
func TestComplianceRequirements_StandardElevated(t *testing.T) {
	reqs := RequirementsFor(ComplianceStandard)

	checks := map[string]bool{
		"SecretsSweep":    reqs.SecretsSweep,
		"ForbiddenFiles":  reqs.ForbiddenFiles,
		"ValidateAll":     reqs.ValidateAll,
		"HygieneRequired": reqs.HygieneRequired,
		"BlockOnWarning":  reqs.BlockOnWarning,
		"ConflictPred":    reqs.ConflictPred,
		"Owv":             reqs.Owv,
	}
	for name, val := range checks {
		if !val {
			t.Errorf("FAIL: standard → %s should be TRUE (elevated)", name)
		}
	}
	if reqs.ValidateMinPct < 0.80 {
		t.Errorf("FAIL: standard → ValidateMinPct should be ≥0.80, got %.2f", reqs.ValidateMinPct)
	}
	t.Log("✅ Standard compliance elevated correctly")
}

// TestComplianceRequirements_Strict verifies strict level.
func TestComplianceRequirements_Strict(t *testing.T) {
	reqs := RequirementsFor(ComplianceStrict)
	if reqs.ValidateMinPct < 0.90 {
		t.Errorf("FAIL: strict → ValidateMinPct should be ≥0.90, got %.2f", reqs.ValidateMinPct)
	}
	if !reqs.GPGSigned {
		t.Error("FAIL: strict → GPGSigned should be TRUE")
	}
	if !reqs.ReviewerReq {
		t.Error("FAIL: strict → ReviewerReq should be TRUE")
	}
	t.Log("✅ Strict compliance requirements verified")
}

// TestComplianceRequirements_Maximum verifies maximum level (100%).
func TestComplianceRequirements_Maximum(t *testing.T) {
	reqs := RequirementsFor(ComplianceMaximum)
	if reqs.ValidateMinPct != 1.0 {
		t.Errorf("FAIL: maximum → ValidateMinPct should be 1.0 (100%%), got %.2f", reqs.ValidateMinPct)
	}
	if !reqs.RedTeam {
		t.Error("FAIL: maximum → RedTeam should be TRUE")
	}
	t.Log("✅ Maximum compliance: 100% validators + Red Team")
}

// TestComplianceRequirements_Quick verifies quick level is minimal.
func TestComplianceRequirements_Quick(t *testing.T) {
	reqs := RequirementsFor(ComplianceQuick)
	if reqs.SecretsSweep {
		t.Error("FAIL: quick → SecretsSweep should be FALSE (minimal)")
	}
	if reqs.BlockOnWarning {
		t.Error("FAIL: quick → BlockOnWarning should be FALSE (permissive)")
	}
	if reqs.HygieneRequired {
		t.Error("FAIL: quick → HygieneRequired should be FALSE")
	}
	t.Log("✅ Quick compliance is correctly minimal")
}

// ── Red Team: Secrets Sweep ─────────────────────────────────────────────

// TestSecretsSweep_DetectsAPIKey verifies the secrets scanner catches API keys in a real git repo.
func TestSecretsSweep_DetectsAPIKey(t *testing.T) {
	repoRoot := initTestRepo(t)

	// Create a file with a fake API key
	keyFile := filepath.Join(repoRoot, "config.go")
	os.WriteFile(keyFile, []byte(`package config
var API_KEY = "sk-12345678901234567890123456789012"
`), 0644)

	git(t, repoRoot, "checkout", "-b", "feature/test-secrets-key")
	git(t, repoRoot, "add", "config.go")
	git(t, repoRoot, "commit", "-m", "test: add config with secret")

	findings, err := scanSecretsInChanges(repoRoot, "feature/test-secrets-key")
	if err != nil {
		t.Fatalf("scanSecretsInChanges error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("FAIL: secrets scanner should detect OpenAI API key")
	} else {
		t.Logf("✅ Detected: %s line %d — %s", findings[0].File, findings[0].Line, findings[0].Detail)
	}
	git(t, repoRoot, "checkout", "main")
	git(t, repoRoot, "branch", "-D", "feature/test-secrets-key")
}

// TestSecretsSweep_DetectsGitHubToken verifies GitHub PAT detection.
func TestSecretsSweep_DetectsGitHubToken(t *testing.T) {
	repoRoot := initTestRepo(t)

	os.WriteFile(filepath.Join(repoRoot, "ci.yaml"), []byte(`token: ghp_123456789012345678901234567890123456
`), 0644)

	git(t, repoRoot, "checkout", "-b", "feature/test-gh-token")
	git(t, repoRoot, "add", "ci.yaml")
	git(t, repoRoot, "commit", "-m", "test: add GH token")

	findings, err := scanSecretsInChanges(repoRoot, "feature/test-gh-token")
	if err != nil {
		t.Fatalf("scanSecretsInChanges error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("FAIL: secrets scanner should detect GitHub PAT")
	} else {
		t.Logf("✅ Detected: %s", findings[0].Detail)
	}
	git(t, repoRoot, "checkout", "main")
	git(t, repoRoot, "branch", "-D", "feature/test-gh-token")
}

// TestSecretsSweep_SkipsComments verifies commented-out secrets are not flagged.
func TestSecretsSweep_SkipsComments(t *testing.T) {
	findings := matchSecretPatterns("test.go", 5, "// API_KEY = \"sk-12345678901234567890123456789012\"")
	if len(findings) > 0 {
		t.Error("FAIL: commented Go line should NOT be flagged as secret")
	}
	findings = matchSecretPatterns("test.py", 10, "# password = \"supersecret\"")
	if len(findings) > 0 {
		t.Error("FAIL: Python comment should NOT be flagged as secret")
	}
	t.Log("✅ Commented secrets correctly skipped")
}

// ── Red Team: Forbidden Files ───────────────────────────────────────────

// TestForbiddenFiles_BlocksEnvFile verifies .env files are blocked.
func TestForbiddenFiles_BlocksEnvFile(t *testing.T) {
	repoRoot := initTestRepo(t)

	os.WriteFile(filepath.Join(repoRoot, ".env"), []byte("SECRET=value\n"), 0644)
	git(t, repoRoot, "checkout", "-b", "feature/test-env")
	git(t, repoRoot, "add", ".env")
	git(t, repoRoot, "commit", "-m", "test: add .env")

	forbidden, err := scanForbiddenFiles(repoRoot, "feature/test-env")
	if err != nil {
		t.Fatalf("scanForbiddenFiles error: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("FAIL: .env file should be blocked as forbidden")
	} else {
		t.Logf("✅ Blocked: %s — %s", forbidden[0].Path, forbidden[0].Reason)
	}
	git(t, repoRoot, "checkout", "main")
	git(t, repoRoot, "branch", "-D", "feature/test-env")
}

// TestForbiddenFiles_BlocksPEMFile verifies .pem files are blocked.
func TestForbiddenFiles_BlocksPEMFile(t *testing.T) {
	repoRoot := initTestRepo(t)

	os.MkdirAll(filepath.Join(repoRoot, "ssl"), 0755)
	os.WriteFile(filepath.Join(repoRoot, "ssl/key.pem"), []byte("-----BEGIN RSA PRIVATE KEY-----\nMOCK\n-----END RSA PRIVATE KEY-----\n"), 0644)
	git(t, repoRoot, "checkout", "-b", "feature/test-pem")
	git(t, repoRoot, "add", "ssl/key.pem")
	git(t, repoRoot, "commit", "-m", "test: add PEM")

	forbidden, err := scanForbiddenFiles(repoRoot, "feature/test-pem")
	if err != nil {
		t.Fatalf("scanForbiddenFiles error: %v", err)
	}
	if len(forbidden) == 0 {
		t.Error("FAIL: .pem file should be blocked as forbidden")
	} else {
		t.Logf("✅ Blocked: %s — %s", forbidden[0].Path, forbidden[0].Reason)
	}
	git(t, repoRoot, "checkout", "main")
	git(t, repoRoot, "branch", "-D", "feature/test-pem")
}

// TestForbiddenFiles_AllowsNormalFiles verifies normal files pass.
func TestForbiddenFiles_AllowsNormalFiles(t *testing.T) {
	repoRoot := initTestRepo(t)

	os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0644)
	git(t, repoRoot, "checkout", "-b", "feature/test-normal")
	git(t, repoRoot, "add", "main.go")
	git(t, repoRoot, "commit", "-m", "test: add normal")

	forbidden, err := scanForbiddenFiles(repoRoot, "feature/test-normal")
	if err != nil {
		t.Fatalf("scanForbiddenFiles error: %v", err)
	}
	if len(forbidden) > 0 {
		t.Errorf("FAIL: normal .go file should not be blocked, got: %v", forbidden)
	} else {
		t.Log("✅ Normal files correctly allowed")
	}
	git(t, repoRoot, "checkout", "main")
	git(t, repoRoot, "branch", "-D", "feature/test-normal")
}

// ── Red Team: Compliance Level Differentiation ──────────────────────────

// TestComplianceLevels_Differentiation verifies each level is distinctly stricter.
func TestComplianceLevels_Differentiation(t *testing.T) {
	tests := []struct {
		level    ComplianceLevel
		minPct   float64
		gpg      bool
		secrets  bool
		reviewer bool
		redteam  bool
		block    bool
	}{
		{ComplianceQuick, 0.0, false, false, false, false, false},
		{ComplianceStandard, 0.85, false, true, false, false, true},
		{ComplianceStrict, 0.95, true, true, true, false, true},
		{ComplianceMaximum, 1.0, true, true, true, true, true},
	}

	for _, tc := range tests {
		reqs := RequirementsFor(tc.level)
		if reqs.ValidateMinPct != tc.minPct {
			t.Errorf("FAIL: %s → ValidateMinPct expected %.2f, got %.2f", tc.level, tc.minPct, reqs.ValidateMinPct)
		}
		if reqs.GPGSigned != tc.gpg {
			t.Errorf("FAIL: %s → GPGSigned expected %v, got %v", tc.level, tc.gpg, reqs.GPGSigned)
		}
		if reqs.SecretsSweep != tc.secrets {
			t.Errorf("FAIL: %s → SecretsSweep expected %v, got %v", tc.level, tc.secrets, reqs.SecretsSweep)
		}
		if reqs.ReviewerReq != tc.reviewer {
			t.Errorf("FAIL: %s → ReviewerReq expected %v, got %v", tc.level, tc.reviewer, reqs.ReviewerReq)
		}
		if reqs.RedTeam != tc.redteam {
			t.Errorf("FAIL: %s → RedTeam expected %v, got %v", tc.level, tc.redteam, reqs.RedTeam)
		}
		if reqs.BlockOnWarning != tc.block {
			t.Errorf("FAIL: %s → BlockOnWarning expected %v, got %v", tc.level, tc.block, reqs.BlockOnWarning)
		}
	}
	t.Log("✅ All 4 compliance levels correctly differentiated")
}

// ── Red Team: SealedStatus ──────────────────────────────────────────────

func TestSealedStatus_DisplaysCorrectly(t *testing.T) {
	standard := SealedStatus(ComplianceStandard)
	for _, want := range []string{"secrets", "validate", "forbidden", "hygiene"} {
		if !strings.Contains(standard, want) {
			t.Errorf("FAIL: standard status should include %q, got: %s", want, standard)
		}
	}

	quick := SealedStatus(ComplianceQuick)
	if strings.Contains(quick, "secrets") || strings.Contains(quick, "validate") {
		t.Errorf("FAIL: quick status should be minimal, got: %s", quick)
	}

	maximum := SealedStatus(ComplianceMaximum)
	if !strings.Contains(maximum, "100%") {
		t.Errorf("FAIL: maximum status should show 100%%, got: %s", maximum)
	}
	if !strings.Contains(maximum, "red-team") {
		t.Errorf("FAIL: maximum status should include red-team, got: %s", maximum)
	}

	t.Logf("Compliance status strings:")
	t.Logf("  Quick:    %s", quick)
	t.Logf("  Standard: %s", standard)
	t.Logf("  Strict:   %s", SealedStatus(ComplianceStrict))
	t.Logf("  Maximum:  %s", maximum)
}

// ── Red Team: Match Secret Patterns (unit-level) ────────────────────────

func TestMatchSecretPatterns_Various(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		line   string
		should bool
	}{
		{"GitHub PAT", "ci.yaml", "  token: ghp_123456789012345678901234567890123456", true},
		{"AWS Key ID", "aws.go", "var key = \"AKIA1234567890ABCDE\"", true},
		{"Stripe Live", "pay.go", "const key = \"sk_live_REDACTED_REDACTED_REDACTED1234\"", true},
		{"Private Key Block", "cert.pem", "-----BEGIN RSA PRIVATE KEY-----", true},
		{"Go comment (skip)", "test.go", "// API_KEY = \"sk-12345678901234567890123456789012\"", false},
		{"Python comment (skip)", "test.py", "# password = \"supersecret\"", false},
		{"Normal code", "main.go", "var name = \"hello\"", false},
		{"Short value", "config.go", "var key = \"test\"", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := matchSecretPatterns(tc.file, 1, tc.line)
			detected := len(findings) > 0
			if detected != tc.should {
				if tc.should {
					t.Errorf("FAIL: should detect secret in: %s", tc.line)
				} else {
					t.Errorf("FAIL: should NOT flag: %s", tc.line)
				}
			} else {
				if detected {
					t.Logf("✅ Correctly detected: %s", findings[0].Detail)
				} else {
					t.Logf("✅ Correctly skipped")
				}
			}
		})
	}
}

// ── Helpers ─────────────────────────────────────────────────────────

// initTestRepo creates a temporary git repo with main + develop branches.
func initTestRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoRoot, 0755)

	runGitOk(t, repoRoot, "init")
	runGitOk(t, repoRoot, "config", "user.email", "test@ovav.dev")
	runGitOk(t, repoRoot, "config", "user.name", "Test Bot")

	os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Test\n"), 0644)
	runGitOk(t, repoRoot, "add", "README.md")
	runGitOk(t, repoRoot, "commit", "-m", "initial commit")
	// Rename the current branch to 'master' so tests assuming master work correctly
	runGitOk(t, repoRoot, "branch", "-m", "master")
	runGitOk(t, repoRoot, "branch", "develop")

	return repoRoot
}

func git(t *testing.T, repoRoot string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
}

func runGitOk(t *testing.T, repoRoot string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
