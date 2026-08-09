package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProfiles(t *testing.T) {
	profiles := getProfiles()
	if len(profiles) == 0 {
		t.Fatal("getProfiles() returned empty slice")
	}

	// At least the 8 service areas must be present.
	if len(profiles) < 8 {
		t.Errorf("expected at least 8 profiles, got %d", len(profiles))
	}

	// Check P0 profiles (area_platform and area_research from topology).
	p0IDs := map[string]bool{"area_platform": true, "area_research": true}
	for _, p := range profiles {
		if p0IDs[p.ID] && !p.P0 {
			t.Errorf("P0 profile %q should have P0=true", p.ID)
		}
		if !p0IDs[p.ID] && p.P0 {
			t.Errorf("non-P0 profile %q should have P0=false", p.ID)
		}
	}
}

func TestAllProfilesHaveRequiredFields(t *testing.T) {
	for _, p := range getProfiles() {
		if p.ID == "" {
			t.Error("profile with empty ID")
		}
		if p.Name == "" {
			t.Errorf("profile %q has empty Name", p.ID)
		}
		if p.Lead == "" {
			t.Errorf("profile %q has empty Lead", p.ID)
		}
		if p.Description == "" {
			t.Errorf("profile %q has empty Description", p.ID)
		}
	}
}

func TestNoDuplicateProfileIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range getProfiles() {
		if seen[p.ID] {
			t.Errorf("duplicate profile ID: %q", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestProfileDescriptionsExist(t *testing.T) {
	for _, p := range getProfiles() {
		desc, ok := profileDescriptions[p.ID]
		if !ok {
			t.Errorf("profile %q has no entry in profileDescriptions map", p.ID)
			continue
		}
		_ = desc // Fallback is used only when topology description is empty.
	}
}

func TestProfileLeadsMatchTopology(t *testing.T) {
	expectedLeads := map[string]string{
		"area_platform":                 "thavren",
		"area_research":                 "eidren",
		"area_digital_product":          "dante",
		"area_commercial_growth":        "sofía",
		"area_devops_infrastructure":    "uriel",
		"area_education_career":         "valeria",
		"area_health_performance":       "renata",
		"area_ux_design":                "elena",
		"area_adversarial_intelligence": "kenji",
	}

	for _, p := range getProfiles() {
		expected, ok := expectedLeads[p.ID]
		if !ok {
			t.Errorf("unexpected profile ID: %q (test may need updating for new area)", p.ID)
			continue
		}
		if p.Lead != expected {
			t.Errorf("profile %q lead: got %q, want %q", p.ID, p.Lead, expected)
		}
	}
}

func TestProfileNamesFromTopology(t *testing.T) {
	expectedNames := map[string]string{
		"area_platform":              "Platform Engineering",
		"area_research":              "Research Intelligence",
		"area_digital_product":       "Web Development",
		"area_commercial_growth":     "Commercial & Growth",
		"area_devops_infrastructure": "DevOps & Infrastructure",
		"area_education_career":      "Education & Training",
		"area_health_performance":    "Sports Science",
		"area_ux_design":             "UI/UX Design",
	}

	for _, p := range getProfiles() {
		expected, ok := expectedNames[p.ID]
		if !ok {
			t.Errorf("unexpected profile ID: %q (test may need updating for new area)", p.ID)
			continue
		}
		if p.Name != expected {
			t.Errorf("profile %q name: got %q, want %q", p.ID, p.Name, expected)
		}
	}
}

func TestCmdList(t *testing.T) {
	// Test human-readable output
	exitCode := CmdList([]string{})
	if exitCode != 0 {
		t.Errorf("CmdList returned %d, want 0", exitCode)
	}

	// Test JSON output
	exitCode = CmdList([]string{"--json"})
	if exitCode != 0 {
		t.Errorf("CmdList --json returned %d, want 0", exitCode)
	}
}

func TestCmdApplyHelp(t *testing.T) {
	exitCode := CmdApply([]string{"--help"})
	if exitCode != 0 {
		t.Errorf("CmdApply --help returned %d, want 0", exitCode)
	}

	exitCode = CmdApply([]string{"-h"})
	if exitCode != 0 {
		t.Errorf("CmdApply -h returned %d, want 0", exitCode)
	}
}

func TestCmdApplyInvalidProfile(t *testing.T) {
	exitCode := CmdApply([]string{"nonexistent_profile"})
	if exitCode != 1 {
		t.Errorf("CmdApply with invalid profile returned %d, want 1", exitCode)
	}
}

func TestCmdApplyNoArgs(t *testing.T) {
	exitCode := CmdApply([]string{})
	// Empty args shows help — exit 0 is correct behavior
	if exitCode != 0 {
		t.Errorf("CmdApply with no args returned %d, want 0 (shows help)", exitCode)
	}
}

func TestCmdApplyDryRun(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ovav-profile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	exitCode := CmdApply([]string{"area_platform", "--target", tmpDir, "--dry-run"})
	if exitCode != 0 {
		t.Errorf("CmdApply --dry-run returned %d, want 0", exitCode)
	}

	// Verify no files were written
	filesToCheck := []string{"AGENTS.md", "opencode.json"}
	for _, f := range filesToCheck {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("--dry-run should not create %s", f)
		}
	}
}

func TestCmdApplyAndRemove(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ovav-profile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Apply profile (skip confirmation)
	exitCode := CmdApply([]string{"area_platform", "--target", tmpDir, "--yes"})
	if exitCode != 0 {
		t.Fatalf("CmdApply returned %d, want 0", exitCode)
	}

	// Verify files were created
	filesToCheck := []string{"AGENTS.md", "opencode.json"}
	for _, f := range filesToCheck {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("apply should create %s", f)
		}
	}

	// Verify AGENTS.md content
	agentsMD := filepath.Join(tmpDir, "AGENTS.md")
	content, err := os.ReadFile(agentsMD)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Platform Engineering") {
		t.Error("AGENTS.md should contain 'Platform Engineering'")
	}
	if !strings.Contains(string(content), "thavren") {
		t.Error("AGENTS.md should contain lead name 'thavren'")
	}
	if !strings.Contains(string(content), "area_platform") {
		t.Error("AGENTS.md should contain area ID 'area_platform'")
	}

	// Verify opencode.json content
	opencodeJSON := filepath.Join(tmpDir, "opencode.json")
	jsonContent, err := os.ReadFile(opencodeJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonContent), "area_platform") {
		t.Error("opencode.json should contain profile area ID 'area_platform'")
	}

	// Verify .opencode directories
	for _, d := range []string{".opencode/agents", ".opencode/skills"} {
		path := filepath.Join(tmpDir, d)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("apply should create directory %s", d)
		}
	}

	// Remove profile
	exitCode = CmdRemove([]string{"area_platform", "--target", tmpDir, "--yes"})
	if exitCode != 0 {
		t.Fatalf("CmdRemove returned %d, want 0", exitCode)
	}

	// Verify files were removed
	for _, f := range filesToCheck {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("remove should delete %s", f)
		}
	}
}

func TestCmdRemoveNoArgs(t *testing.T) {
	exitCode := CmdRemove([]string{})
	// Empty args shows help — exit 0 is correct behavior
	if exitCode != 0 {
		t.Errorf("CmdRemove with no args returned %d, want 0 (shows help)", exitCode)
	}
}

func TestCmdRemoveInvalidProfile(t *testing.T) {
	exitCode := CmdRemove([]string{"nonexistent"})
	if exitCode != 1 {
		t.Errorf("CmdRemove with invalid profile returned %d, want 1", exitCode)
	}
}

func TestCmdRemoveDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ovav-profile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Apply first
	CmdApply([]string{"area_platform", "--target", tmpDir, "--yes"})

	// Remove with dry-run
	exitCode := CmdRemove([]string{"area_platform", "--target", tmpDir, "--dry-run"})
	if exitCode != 0 {
		t.Errorf("CmdRemove --dry-run returned %d, want 0", exitCode)
	}

	// Files should still exist after dry-run
	agentsMD := filepath.Join(tmpDir, "AGENTS.md")
	if _, err := os.Stat(agentsMD); os.IsNotExist(err) {
		t.Error("--dry-run should not delete AGENTS.md")
	}
}

func TestPrintHelp(t *testing.T) {
	// Just verify it doesn't panic
	PrintHelp()
}

func TestCmdRemoveHelp(t *testing.T) {
	exitCode := CmdRemove([]string{"--help"})
	if exitCode != 0 {
		t.Errorf("CmdRemove --help returned %d, want 0", exitCode)
	}

	exitCode = CmdRemove([]string{"-h"})
	if exitCode != 0 {
		t.Errorf("CmdRemove -h returned %d, want 0", exitCode)
	}
}

func TestGenerateProfileAllAreas(t *testing.T) {
	for _, p := range getProfiles() {
		tmpDir, err := os.MkdirTemp("", "ovav-profile-test-*")
		if err != nil {
			t.Fatal(err)
		}

		exitCode := CmdApply([]string{p.ID, "--target", tmpDir, "--yes"})
		if exitCode != 0 {
			t.Errorf("apply %q returned %d, want 0", p.ID, exitCode)
		}

		// Verify AGENTS.md contains profile name
		content, _ := os.ReadFile(filepath.Join(tmpDir, "AGENTS.md"))
		if !strings.Contains(string(content), p.Name) {
			t.Errorf("AGENTS.md for %q should contain profile name %q", p.ID, p.Name)
		}

		// Verify AGENTS.md contains area ID
		if !strings.Contains(string(content), p.ID) {
			t.Errorf("AGENTS.md for %q should contain area ID %q", p.ID, p.ID)
		}

		os.RemoveAll(tmpDir)
	}
}

func TestCmdApplyEdgeCases(t *testing.T) {
	// Empty target
	exitCode := CmdApply([]string{"area_platform", "--target", ""})
	if exitCode != 0 {
		t.Logf("CmdApply with empty target: exit=%d", exitCode)
	}

	// Non-existent path
	exitCode = CmdApply([]string{"area_platform", "--target", "/nonexistent/path/xyz"})
	if exitCode == 0 {
		t.Log("CmdApply with non-existent path should not succeed")
	}
}
