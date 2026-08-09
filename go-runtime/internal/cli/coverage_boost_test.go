package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── envdetect.go: IsOVAVProject (0% → tested) ─────────────────────────────

func TestIsOVAVProject(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav"), 0755)

	if !IsOVAVProject(tmp) {
		t.Error("IsOVAVProject should be true for dir with .git + .ovav")
	}
}

func TestIsOVAVProject_FalseForDev(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav"), 0755)
	os.MkdirAll(filepath.Join(tmp, "tools", "harnesses"), 0755)

	if IsOVAVProject(tmp) {
		t.Error("IsOVAVProject should be false for ovav_dev environment")
	}
}

func TestIsOVAVProject_FalseForExternal(t *testing.T) {
	tmp := t.TempDir()
	if IsOVAVProject(tmp) {
		t.Error("IsOVAVProject should be false for external environment")
	}
}

// ── envdetect.go: DetectEnv edge cases ─────────────────────────────────────

func TestDetectEnv_EmptyPath(t *testing.T) {
	// Empty path should use cwd, which is a valid git repo
	result := DetectEnv("")
	if result.Env == EnvExternal {
		// Might be external if cwd is not a git repo — that's fine
		return
	}
	// If we're in a git repo, should detect properly
	if result.Root == "" {
		t.Error("expected non-empty root when running from cwd")
	}
}

func TestDetectEnv_NonexistentPath(t *testing.T) {
	result := DetectEnv("/nonexistent/path/xyz")
	if result.Env != EnvExternal {
		t.Errorf("env = %v, want external for nonexistent path", result.Env)
	}
}

// ── envdetect.go: EffectiveTier edge cases ──────────────────────────────────

func TestIsInternalTier_False(t *testing.T) {
	os.Unsetenv("OVAV_DEV")
	if IsInternalTier() {
		t.Error("IsInternalTier should be false when OVAV_DEV is unset")
	}
}

// ── freshsmoke.go: FreshCloneSmoke + JSON ───────────────────────────────────

func TestFreshCloneSmoke(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.WriteFile(filepath.Join(tmp, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	result := FreshCloneSmoke(tmp, false)
	if result.SchemaVersion != "ovav.fresh_clone_smoke.v1" {
		t.Errorf("schema = %v", result.SchemaVersion)
	}
	if len(result.Checks) == 0 {
		t.Error("expected at least one check")
	}
}

func TestFreshCloneSmokeJSON(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.WriteFile(filepath.Join(tmp, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	data, err := FreshCloneSmokeJSON(tmp, false)
	if err != nil {
		t.Fatalf("FreshCloneSmokeJSON error = %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
	if !strings.Contains(string(data), "schema_version") {
		t.Error("JSON should contain schema_version")
	}
}

func TestFreshCloneSmoke_InvalidRepo(t *testing.T) {
	result := FreshCloneSmoke("/nonexistent/path/xyz", false)
	if result.OverallOK {
		t.Error("OverallOK should be false for invalid repo")
	}
}

func TestFreshCloneSmoke_KeepClone(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	os.WriteFile(filepath.Join(tmp, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	result := FreshCloneSmoke(tmp, true)
	// Just verify it doesn't panic and produces valid result
	if result.SchemaVersion != "ovav.fresh_clone_smoke.v1" {
		t.Errorf("schema = %v", result.SchemaVersion)
	}
}

// ── gates.go: scanForSecrets (39.1%) ───────────────────────────────────────

func TestScanForSecrets_NoSecrets(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	os.WriteFile(filepath.Join(tmp, "tools", "safe.py"), []byte("print('hello')"), 0644)

	findings := scanForSecrets(tmp)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestScanForSecrets_WithSecret(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	os.WriteFile(filepath.Join(tmp, "tools", "bad.py"), []byte("api_key = 'sk-abc123'"), 0644)

	findings := scanForSecrets(tmp)
	if len(findings) == 0 {
		t.Error("expected to find secret pattern")
	}
}

func TestScanForSecrets_MissingDirs(t *testing.T) {
	tmp := t.TempDir()
	// No tools/, bin/, .ovav/registry/, docs/ dirs
	findings := scanForSecrets(tmp)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings with no scan dirs, got %d", len(findings))
	}
}

func TestScanForSecrets_NonPythonFiles(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	os.WriteFile(filepath.Join(tmp, "tools", "secret.txt"), []byte("sk-fakekey"), 0644)

	findings := scanForSecrets(tmp)
	// Only .py files are scanned
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for non-py files, got %d", len(findings))
	}
}

func TestScanForSecrets_AllDirs(t *testing.T) {
	tmp := t.TempDir()
	dirs := []string{"tools", "bin", ".ovav/registry", "docs"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(tmp, d), 0755)
		os.WriteFile(filepath.Join(tmp, d, "test.py"), []byte("clean"), 0644)
	}

	findings := scanForSecrets(tmp)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings in clean files, got %d", len(findings))
	}
}

func TestScanForSecrets_MultiplePatterns(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	content := "key = 'ghp_abcdef1234567890'\nsecret = 'AIzaSyDfake'"
	os.WriteFile(filepath.Join(tmp, "tools", "config.py"), []byte(content), 0644)

	findings := scanForSecrets(tmp)
	if len(findings) < 2 {
		t.Errorf("expected 2+ findings for multiple patterns, got %d", len(findings))
	}
}

// ── gates.go: checkCI (27.3%) ──────────────────────────────────────────────

func TestCheckCI_NoWorkflowsDir(t *testing.T) {
	tmp := t.TempDir()
	result := checkCI(tmp)
	if result["present"].(bool) {
		t.Error("should report no CI when .github/workflows/ is missing")
	}
	wfs := result["workflows"].([]string)
	if len(wfs) != 0 {
		t.Error("expected empty workflows list")
	}
}

func TestCheckCI_EmptyWorkflowsDir(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)

	result := checkCI(tmp)
	if result["present"].(bool) {
		t.Error("should report no CI when workflows dir is empty")
	}
}

func TestCheckCI_WithWorkflows(t *testing.T) {
	tmp := t.TempDir()
	wfDir := filepath.Join(tmp, ".github", "workflows")
	os.MkdirAll(wfDir, 0755)
	os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI"), 0644)
	os.WriteFile(filepath.Join(wfDir, "deploy.yaml"), []byte("name: Deploy"), 0644)
	os.WriteFile(filepath.Join(wfDir, "readme.txt"), []byte("not a workflow"), 0644)

	result := checkCI(tmp)
	if !result["present"].(bool) {
		t.Error("should report CI present")
	}
	wfs := result["workflows"].([]string)
	if len(wfs) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(wfs))
	}
}

func TestCheckCI_OnlyNonYamlFiles(t *testing.T) {
	tmp := t.TempDir()
	wfDir := filepath.Join(tmp, ".github", "workflows")
	os.MkdirAll(wfDir, 0755)
	os.WriteFile(filepath.Join(wfDir, "readme.txt"), []byte("not a workflow"), 0644)

	result := checkCI(tmp)
	if result["present"].(bool) {
		t.Error("should report no CI with only non-yaml files")
	}
}

// ── gates.go: checkUncommitted (50%) ────────────────────────────────────────

func TestCheckUncommitted_IncompleteRepo(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".git"), 0755)
	// git status fails on incomplete repo — returns false + "git unavailable"
	clean, files := checkUncommitted(tmp)
	if clean {
		t.Error("expected false when git fails on incomplete repo")
	}
	if len(files) != 1 || files[0] != "git unavailable" {
		t.Errorf("expected [git unavailable], got %v", files)
	}
}

// ── gates.go: checkVersionConsistency (53.3%) ──────────────────────────────

func TestCheckVersionConsistency_Found(t *testing.T) {
	tmp := t.TempDir()
	content := "# AGENTS.md\nVersion 1.2.3\nSome other line\n"
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(content), 0644)

	result := checkVersionConsistency(tmp)
	if _, ok := result["AGENTS.md"]; !ok {
		t.Error("expected version info from AGENTS.md")
	}
}

func TestCheckVersionConsistency_NoVersion(t *testing.T) {
	tmp := t.TempDir()
	content := "# AGENTS.md\nNo version info here\n"
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(content), 0644)

	result := checkVersionConsistency(tmp)
	// Should not find version
	for _, v := range result {
		if strings.Contains(v, "Version") {
			t.Errorf("should not find version in: %v", v)
		}
	}
}

func TestCheckVersionConsistency_MissingFiles(t *testing.T) {
	tmp := t.TempDir()
	result := checkVersionConsistency(tmp)
	// No files exist, should return empty map
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestCheckVersionConsistency_LongLine(t *testing.T) {
	tmp := t.TempDir()
	longLine := "Version " + strings.Repeat("x", 100)
	content := longLine + "\n"
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte(content), 0644)

	result := checkVersionConsistency(tmp)
	if v, ok := result["AGENTS.md"]; ok {
		if len(v) > 80 {
			t.Errorf("line should be truncated to 80 chars, got %d", len(v))
		}
	}
}

// ── gates.go: helper functions ─────────────────────────────────────────────

func TestDetailOrClean_Zero(t *testing.T) {
	if detailOrClean(0, "secrets found") != "clean" {
		t.Error("expected 'clean' for count=0")
	}
}

func TestDetailOrClean_NonZero(t *testing.T) {
	got := detailOrClean(3, "secrets found")
	if got != "3 secrets found" {
		t.Errorf("got %q, want '3 secrets found'", got)
	}
}

func TestDetailListOrClean_Empty(t *testing.T) {
	got := detailListOrClean([]string{})
	if got != "clean" {
		t.Errorf("got %v, want 'clean'", got)
	}
}

func TestDetailListOrClean_WithItems(t *testing.T) {
	items := []string{".env", "secrets.yaml"}
	got := detailListOrClean(items)
	list, ok := got.([]string)
	if !ok || len(list) != 2 {
		t.Errorf("got %v, want 2-item list", got)
	}
}

func TestDetailOrList_Clean(t *testing.T) {
	got := detailOrList(true, "clean", []string{"file"})
	if got != "clean" {
		t.Errorf("got %v, want 'clean'", got)
	}
}

func TestDetailOrList_Dirty(t *testing.T) {
	files := []string{"file1.py", "file2.py"}
	got := detailOrList(false, "clean", files)
	list, ok := got.([]string)
	if !ok || len(list) != 2 {
		t.Errorf("got %v, want file list", got)
	}
}

func TestOrDefault_Empty(t *testing.T) {
	if orDefault("", "fallback") != "fallback" {
		t.Error("expected fallback for empty string")
	}
}

func TestOrDefault_NonEmpty(t *testing.T) {
	if orDefault("v1.0.0", "fallback") != "v1.0.0" {
		t.Error("expected value when non-empty")
	}
}

// ── gates.go: checkReadme edge cases ────────────────────────────────────────

func TestCheckReadme_TooShort(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("Hi"), 0644)

	result := checkReadme(tmp)
	issues := result["issues"].([]string)
	if len(issues) == 0 {
		t.Error("expected issues for short README")
	}
}

func TestCheckReadme_NoOvavMention(t *testing.T) {
	tmp := t.TempDir()
	content := "# My Project\n\n## Overview\nThis is a random project with enough content to pass the length check. It has sections and detail.\n"
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte(content), 0644)

	result := checkReadme(tmp)
	issues := result["issues"].([]string)
	found := false
	for _, i := range issues {
		if strings.Contains(i, "OVAV") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'does not mention OVAV' issue")
	}
}

func TestCheckReadme_NoSections(t *testing.T) {
	tmp := t.TempDir()
	content := "OVAV is a great project with no sections in the README at all. Just plain text. Lots of it.\n"
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte(content), 0644)

	result := checkReadme(tmp)
	issues := result["issues"].([]string)
	found := false
	for _, i := range issues {
		if strings.Contains(i, "sections") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'no sections' issue")
	}
}

func TestCheckReadme_Present(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# OVAV\n\n## Section\nContent\n"), 0644)

	result := checkReadme(tmp)
	if !result["present"].(bool) {
		t.Error("expected present=true")
	}
}

// ── gates.go: ExportGateCheck edge cases ────────────────────────────────────

func TestExportGateCheck_WithSecret(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	os.WriteFile(filepath.Join(tmp, "tools", "bad.py"), []byte("api_key='sk-test123'"), 0644)
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# Test"), 0644)

	result := ExportGateCheck(tmp)
	if result["passed"].(bool) {
		t.Error("should fail when secrets found")
	}
}

func TestExportGateCheck_ForbiddenMultiple(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, ".env"), []byte("SECRET=1"), 0644)
	os.WriteFile(filepath.Join(tmp, "credentials.json"), []byte("{}"), 0644)

	result := ExportGateCheck(tmp)
	forbidden := result["forbidden_files"].([]string)
	if len(forbidden) != 2 {
		t.Errorf("expected 2 forbidden files, got %d", len(forbidden))
	}
}

// ── gates.go: ReleasePackageCheck edge cases ────────────────────────────────

func TestReleasePackageCheck_AllChecks(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("Version 2.0.0\n"), 0644)
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("Version 2.0.0\n"), 0644)

	result := ReleasePackageCheck(tmp)
	checks := result["checks"].([]map[string]interface{})
	if len(checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(checks))
	}
}

// ── gates.go: SurfacesRepairPlan edge cases ────────────────────────────────

func TestSurfacesRepairPlan_NoMissing(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".opencode"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav", "source", "skills"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".ovav", "policy"), 0755)
	os.WriteFile(filepath.Join(tmp, ".ovav", "policy", "permission_authority.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# agents"), 0644)
	os.MkdirAll(filepath.Join(tmp, "docs", "launch"), 0755)
	os.MkdirAll(filepath.Join(tmp, ".github", "workflows"), 0755)
	os.WriteFile(filepath.Join(tmp, ".github", "workflows", "ci.yml"), []byte("name: CI"), 0644)

	result := SurfacesRepairPlan(tmp)
	totalMissing := result["total_missing"].(int)
	if totalMissing != 0 {
		t.Errorf("expected 0 missing when all surfaces present, got %d", totalMissing)
	}
}

// ── gates.go: checkEssentials via ExportGateCheck ───────────────────────────

func TestExportGateCheck_NoEssentials(t *testing.T) {
	tmp := t.TempDir()
	// No README, no LICENSE
	result := ExportGateCheck(tmp)
	checks := result["checks"].([]map[string]interface{})
	for _, c := range checks {
		if c["name"] == "readme_present" && c["passed"].(bool) {
			t.Error("readme should not be present")
		}
		if c["name"] == "license_present" && c["passed"].(bool) {
			t.Error("license should not be present")
		}
	}
}

func TestExportGateCheck_WithLicense(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "LICENSE"), []byte("MIT License"), 0644)
	os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# Test"), 0644)

	result := ExportGateCheck(tmp)
	checks := result["checks"].([]map[string]interface{})
	for _, c := range checks {
		if c["name"] == "license_present" && !c["passed"].(bool) {
			t.Error("license should be present")
		}
	}
}

// ── gates.go: scanForSecrets file read error ────────────────────────────────

func TestScanForSecrets_FileReadError(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "tools"), 0755)
	// Create a symlink to nonexistent target to trigger read error
	os.Symlink("/nonexistent/target.py", filepath.Join(tmp, "tools", "broken.py"))

	findings := scanForSecrets(tmp)
	// Should handle gracefully — no crash
	if findings == nil {
		t.Error("expected non-nil findings slice")
	}
}

// ── gates.go: ExecutionGateReport requires edge case ───────────────────────

func TestExecutionGateReport_ApplyConsentOnly(t *testing.T) {
	r := ExecutionGateReport("sync", "apply", true, false)
	if r["allowed"].(bool) {
		t.Error("should be blocked without risk acceptance")
	}
	reqs := r["requires"].([]string)
	if len(reqs) != 1 || reqs[0] != "accept_risk" {
		t.Errorf("requires = %v, want [accept_risk]", reqs)
	}
}

func TestExecutionGateReport_ApplyRiskOnly(t *testing.T) {
	r := ExecutionGateReport("sync", "apply", false, true)
	if r["allowed"].(bool) {
		t.Error("should be blocked without consent")
	}
	reqs := r["requires"].([]string)
	if len(reqs) != 1 || reqs[0] != "consent" {
		t.Errorf("requires = %v, want [consent]", reqs)
	}
}
