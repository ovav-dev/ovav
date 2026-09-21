// Tests for gates.go, envdetect.go, freshsmoke.go — migrated from tools/cli/*.py.
package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Execution Gateway ────────────────────────────────────────────────────────

func TestExecutionGateReport_DryRun(t *testing.T) {
	r := ExecutionGateReport("setup", "dry_run", false, false)
	if !r["allowed"].(bool) {
		t.Error("dry_run should be allowed")
	}
	if r["action"] != "setup" {
		t.Errorf("action = %v, want setup", r["action"])
	}
	if r["mode"] != "dry_run" {
		t.Errorf("mode = %v, want dry_run", r["mode"])
	}
}

func TestExecutionGateReport_ApplyBlocked(t *testing.T) {
	r := ExecutionGateReport("sync", "apply", false, false)
	if r["allowed"].(bool) {
		t.Error("apply without consent should be blocked")
	}
	reqs := r["requires"].([]string)
	if len(reqs) != 2 {
		t.Errorf("requires = %v, want [consent, accept_risk]", reqs)
	}
}

func TestExecutionGateReport_ApplyAllowed(t *testing.T) {
	r := ExecutionGateReport("security", "apply", true, true)
	if !r["allowed"].(bool) {
		t.Error("apply with consent+risk should be allowed")
	}
}

func TestExecutionGateReport_InvalidMode(t *testing.T) {
	r := ExecutionGateReport("setup", "invalid", false, false)
	if r["allowed"].(bool) {
		t.Error("invalid mode should be blocked")
	}
}

func TestValidGatewayActions(t *testing.T) {
	actions := ValidGatewayActions()
	if len(actions) != 5 {
		t.Errorf("got %d actions, want 5", len(actions))
	}
}

// ── Surface Manager ──────────────────────────────────────────────────────────

func TestSurfacesCheck(t *testing.T) {
	// Create a temp dir with some surfaces
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".opencode"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# test"), 0644)

	result := SurfacesCheck(tmp)
	surfaces, ok := result["surfaces"].([]map[string]interface{})
	if !ok {
		t.Fatal("surfaces not found or wrong type")
	}
	if len(surfaces) == 0 {
		t.Error("expected surfaces entries")
	}

	// Check that .opencode is found
	found := false
	for _, s := range surfaces {
		if s["path"] == ".opencode/" {
			found = true
			if s["status"] != "ok" {
				t.Errorf(".opencode status = %v, want ok", s["status"])
			}
		}
	}
	if !found {
		t.Error(".opencode/ not found in surfaces")
	}
}

func TestSurfacesCheckRequiredStatus(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, root string)
		wantPassed bool
	}{
		{
			name:       "missing required surfaces",
			setup:      func(t *testing.T, root string) {},
			wantPassed: false,
		},
		{
			name: "all required surfaces",
			setup: func(t *testing.T, root string) {
				t.Helper()
				for _, surface := range DefaultManagedSurfaces() {
					if !surface.Required {
						continue
					}
					path := filepath.Join(root, surface.Path)
					if filepath.Ext(path) == "" {
						if err := os.MkdirAll(path, 0755); err != nil {
							t.Fatal(err)
						}
						continue
					}
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
						t.Fatal(err)
					}
				}
			},
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)
			result := SurfacesCheck(root)
			if got, ok := result["passed"].(bool); !ok || got != tt.wantPassed {
				t.Errorf("passed = %#v, want %v", result["passed"], tt.wantPassed)
			}
		})
	}
}

func TestSurfacesRepairPlan(t *testing.T) {
	tmp := t.TempDir()
	result := SurfacesRepairPlan(tmp)
	missing, ok := result["total_missing"].(int)
	if !ok {
		t.Fatal("total_missing not found or wrong type")
	}
	if missing == 0 {
		t.Error("expected missing surfaces in empty dir")
	}
}

// ── Export Gate ──────────────────────────────────────────────────────────────

func TestExportGateCheck_Clean(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# Test"), 0644)

	result := ExportGateCheck(tmp)
	if result["schema_version"] != "ovav.public_export_gate.v1" {
		t.Errorf("schema = %v", result["schema_version"])
	}
	secrets, ok := result["secrets_found"].([]map[string]interface{})
	if !ok {
		t.Fatal("secrets_found wrong type")
	}
	if len(secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(secrets))
	}
}

func TestExportGateCheck_ForbiddenFile(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, ".env"), []byte("SECRET=1"), 0644)

	result := ExportGateCheck(tmp)
	forbidden, ok := result["forbidden_files"].([]string)
	if !ok {
		t.Fatal("forbidden_files wrong type")
	}
	if len(forbidden) == 0 {
		t.Error("expected .env in forbidden files")
	}
}

func TestScanForSecrets_IgnoresRegexLiteralsAndComments(t *testing.T) {
	tmp := t.TempDir()
	toolsDir := filepath.Join(tmp, "tools", "web")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join([]string{
		`OPENAI_PATTERN = re.compile(r"sk-[A-Za-z0-9]{32,}")`,
		`GITHUB_PATTERN = re.compile(r"ghp_[A-Za-z0-9]{36}")`,
		`# example token prefix: ghp_`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(toolsDir, "search_gateway.py"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if findings := scanForSecrets(tmp); len(findings) != 0 {
		t.Fatalf("regex literals and comments produced %d finding(s): %#v", len(findings), findings)
	}
}

func TestScanForSecrets_DetectsStructuredSecretWithoutExposingIt(t *testing.T) {
	tmp := t.TempDir()
	toolsDir := filepath.Join(tmp, "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatal(err)
	}
	secret := "ghp_" + strings.Repeat("A7", 18)
	if err := os.WriteFile(filepath.Join(toolsDir, "config.py"), []byte(`token = "`+secret+`"`), 0644); err != nil {
		t.Fatal(err)
	}

	findings := scanForSecrets(tmp)
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	encoded, err := json.Marshal(findings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatal("secret finding exposed matching credential text")
	}
}

// ── Repo Presentation Gate ───────────────────────────────────────────────────

func TestRepoPresentationGate_NoReadme(t *testing.T) {
	tmp := t.TempDir()
	result := RepoPresentationGate(tmp)
	if result["passed"].(bool) {
		t.Error("should fail without README")
	}
}

func TestRepoPresentationGate_GoodReadme(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# OVAV\n\n## Overview\nThis is the OVAV system for AI workstation governance.\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# agents"), 0644)

	result := RepoPresentationGate(tmp)
	checks, ok := result["checks"].([]map[string]interface{})
	if !ok {
		t.Fatal("checks wrong type")
	}
	// readme_present should pass
	for _, c := range checks {
		if c["name"] == "readme_present" && !c["passed"].(bool) {
			t.Error("readme_present should pass")
		}
	}
}

// ── Release Package ──────────────────────────────────────────────────────────

func TestReleasePackageCheck(t *testing.T) {
	tmp := t.TempDir()
	result := ReleasePackageCheck(tmp)
	if result["schema_version"] != "ovav.release_package.v1" {
		t.Errorf("schema = %v", result["schema_version"])
	}
	// Should have checks
	checks, ok := result["checks"].([]map[string]interface{})
	if !ok || len(checks) == 0 {
		t.Error("expected checks")
	}
}

func TestReleasePackageCheck_DirtyWorktreeFails(t *testing.T) {
	root := initCandidateTestRepo(t)
	mustWriteTestFile(t, root, "VERSION", "3.4.0\n")

	result := ReleasePackageCheck(root)
	if result["ready"].(bool) {
		t.Fatal("dirty worktree reported release ready")
	}
	checks := result["checks"].([]map[string]interface{})
	for _, check := range checks {
		if check["name"] == "no_uncommitted_changes" {
			if check["passed"].(bool) {
				t.Fatal("dirty-worktree check passed")
			}
			return
		}
	}
	t.Fatal("dirty-worktree check missing")
}

func TestCheckVersionConsistency(t *testing.T) {
	tests := []struct {
		name       string
		root       string
		goRuntime  string
		packageVer string
		productVer string
		wantPass   bool
		wantIssue  string
	}{
		{name: "match", root: "3.4.0", goRuntime: "3.4.0", packageVer: "2.3.2", productVer: "2.3.2", wantPass: true},
		{name: "mismatch", root: "3.4.0", goRuntime: "3.3.0", packageVer: "2.3.2", productVer: "2.3.2", wantIssue: "version mismatch"},
		{name: "empty", root: "", goRuntime: "3.4.0", packageVer: "2.3.2", productVer: "2.3.2", wantIssue: "VERSION is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			mustWriteTestFile(t, root, "VERSION", tt.root+"\n")
			mustWriteTestFile(t, root, "go-runtime/VERSION", tt.goRuntime+"\n")
			mustWriteTestFile(t, root, "package.json", `{"name":"ovav","version":"`+tt.packageVer+`"}`)
			mustWriteTestFile(t, root, ".ovav/plan/caps.yaml", "product:\n  version: \""+tt.productVer+"\"\n")

			result := checkVersionConsistency(root)
			if result.Passed() != tt.wantPass {
				t.Fatalf("Passed() = %v, want %v; issues: %v", result.Passed(), tt.wantPass, result.Issues)
			}
			if tt.wantIssue != "" && !strings.Contains(strings.Join(result.Issues, "\n"), tt.wantIssue) {
				t.Fatalf("issues = %v, want substring %q", result.Issues, tt.wantIssue)
			}
		})
	}
}

func mustWriteTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// ── Environment Detector ─────────────────────────────────────────────────────

func TestDetectEnv_External(t *testing.T) {
	tmp := t.TempDir()
	result := DetectEnv(tmp)
	if result.Env != EnvExternal {
		t.Errorf("env = %v, want external", result.Env)
	}
}

func TestDetectEnv_OVAVDev(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(tmp, "tools", "harnesses"), 0755)

	result := DetectEnv(tmp)
	if result.Env != EnvOVAVDev {
		t.Errorf("env = %v, want ovav_dev", result.Env)
	}
	if !result.HasOVAV {
		t.Error("has_ovav should be true")
	}
	if !result.HasDevTools {
		t.Error("has_dev_tools should be true")
	}
}

func TestDetectEnv_OVAVProject(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav"), 0755)

	result := DetectEnv(tmp)
	if result.Env != EnvOVAVProject {
		t.Errorf("env = %v, want ovav_project", result.Env)
	}
}

func TestIsOVAVDev(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(tmp, "tools", "harnesses"), 0755)

	if !IsOVAVDev(tmp) {
		t.Error("IsOVAVDev should be true")
	}
}

func TestEffectiveTier(t *testing.T) {
	os.Setenv("OVAV_DEV", "1")
	defer os.Unsetenv("OVAV_DEV")
	if EffectiveTier() != "internal" {
		t.Errorf("tier = %v, want internal", EffectiveTier())
	}

	os.Unsetenv("OVAV_DEV")
	if EffectiveTier() != "public" {
		t.Errorf("tier = %v, want public", EffectiveTier())
	}
}

func TestIsInternalTier(t *testing.T) {
	os.Setenv("OVAV_DEV", "1")
	defer os.Unsetenv("OVAV_DEV")
	if !IsInternalTier() {
		t.Error("IsInternalTier should be true when OVAV_DEV=1")
	}
}

func TestGetRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	root := GetRepoRoot(tmp)
	if root != tmp {
		t.Errorf("root = %v, want %v", root, tmp)
	}
}

func TestGetRepoRoot_NotFound(t *testing.T) {
	root := GetRepoRoot("/nonexistent/path/that/does/not/exist")
	if root != "" {
		t.Errorf("root = %v, want empty", root)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func TestPathExists(t *testing.T) {
	tmp := t.TempDir()
	if !pathExists(tmp) {
		t.Error("pathExists should be true for temp dir")
	}
	if pathExists(filepath.Join(tmp, "nonexistent")) {
		t.Error("pathExists should be false for nonexistent")
	}
}

func TestHasDigit(t *testing.T) {
	if !hasDigit("v1.0.0") {
		t.Error("should find digit")
	}
	if hasDigit("no digits here") {
		t.Error("should not find digit")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 3) != "hel" {
		t.Error("truncate failed")
	}
	if truncate("hi", 10) != "hi" {
		t.Error("truncate should not pad")
	}
}
