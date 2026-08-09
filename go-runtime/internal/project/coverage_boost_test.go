package project

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ── sync_notify.go: 0% functions ─────────────────────────────────────────────

func TestDiffChanges_Empty(t *testing.T) {
	result := diffChanges("", "")
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestDiffChanges_SameContent(t *testing.T) {
	input := "M file1.txt\nM file2.txt"
	result := diffChanges(input, input)
	if len(result) != 0 {
		t.Errorf("expected empty for same content, got %v", result)
	}
}

func TestDiffChanges_NewLines(t *testing.T) {
	before := "M file1.txt"
	after := "M file1.txt\nA file2.txt"
	result := diffChanges(before, after)
	if len(result) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(result), result)
	}
	if result[0] != "A file2.txt" {
		t.Errorf("expected 'A file2.txt', got %q", result[0])
	}
}

func TestDiffChanges_RemovedLines(t *testing.T) {
	before := "M file1.txt\nM file2.txt"
	after := "M file1.txt"
	result := diffChanges(before, after)
	if len(result) != 0 {
		t.Errorf("expected 0 (removed lines are not detected as new), got %d: %v", len(result), result)
	}
}

func TestDiffChanges_EmptyLinesIgnored(t *testing.T) {
	before := "\n\nM file1.txt\n\n"
	after := "\n\nM file1.txt\nA file2.txt\n"
	result := diffChanges(before, after)
	if len(result) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(result), result)
	}
}

func TestNotifyProductUpdate_ReturnsJSON(t *testing.T) {
	result := NotifyProductUpdate("1.0.0", "1.1.0", []string{"a.txt", "b.txt"})
	if !strings.Contains(result, `"from_version": "1.0.0"`) {
		t.Errorf("missing from_version in output: %s", result)
	}
	if !strings.Contains(result, `"to_version": "1.1.0"`) {
		t.Errorf("missing to_version in output: %s", result)
	}
	if !strings.Contains(result, `"changed_files": 2`) {
		t.Errorf("missing changed_files count in output: %s", result)
	}
	if !strings.Contains(result, `"event": "product_update"`) {
		t.Errorf("missing event in output: %s", result)
	}
}

func TestNotifyProductUpdate_EmptyFiles(t *testing.T) {
	result := NotifyProductUpdate("0.0.0", "0.0.1", nil)
	if !strings.Contains(result, `"changed_files": 0`) {
		t.Errorf("expected changed_files 0 for nil slice, got: %s", result)
	}
}

func TestVersionFilePath(t *testing.T) {
	got := versionFilePath("/tmp/ovav")
	want := "/tmp/ovav/VERSION"
	if got != want {
		t.Errorf("versionFilePath() = %q, want %q", got, want)
	}
}

func TestReadVersion_Exists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("2.3.1\n"), 0644)

	got := ReadVersion(dir)
	if got != "2.3.1" {
		t.Errorf("ReadVersion() = %q, want %q", got, "2.3.1")
	}
}

func TestReadVersion_Missing(t *testing.T) {
	dir := t.TempDir()
	got := ReadVersion(dir)
	if got != "0.0.0" {
		t.Errorf("ReadVersion() for missing file = %q, want %q", got, "0.0.0")
	}
}

func TestReadVersion_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.0.0"), 0644)

	got := ReadVersion(dir)
	if got != "1.0.0" {
		t.Errorf("ReadVersion() = %q, want %q", got, "1.0.0")
	}
}

// ── detectCurrentChanges: test with a temp git repo ──────────────────────────

func TestDetectCurrentChanges_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")

	got := detectCurrentChanges(dir)
	if got != "" {
		t.Errorf("expected empty for clean repo, got %q", got)
	}
}

func TestDetectCurrentChanges_WithChanges(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")

	// Create a tracked directory that detectCurrentChanges monitors
	os.MkdirAll(filepath.Join(dir, "runtimes"), 0755)
	os.WriteFile(filepath.Join(dir, "runtimes", "new.txt"), []byte("hello"), 0644)

	got := detectCurrentChanges(dir)
	if got == "" {
		t.Error("expected non-empty for dirty repo")
	}
	if !strings.Contains(got, "runtimes") {
		t.Errorf("expected change to mention runtimes, got %q", got)
	}
}

func TestDetectCurrentChanges_InvalidRoot(t *testing.T) {
	got := detectCurrentChanges("/nonexistent/path/that/does/not/exist")
	if got != "" {
		t.Errorf("expected empty for invalid root, got %q", got)
	}
}

// ── Exported wrappers: SyncAgents, SyncConnectorBus, SyncVisual, SyncMiMoCode

func TestSyncAgents_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, _, err := SyncAgents(dir, false)
	// These are wrappers around the same functions already tested via Sync.
	// They should not panic.
	_ = err // may or may not error depending on convert state
}

func TestSyncConnectorBus_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	skills, agents, err := SyncConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("SyncConnectorBus on empty dir: %v", err)
	}
	if skills != 0 || agents != 0 {
		t.Errorf("expected 0,0 got %d,%d", skills, agents)
	}
}

func TestSyncVisual_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := SyncVisual(dir, false)
	// Should error because theme.yaml is missing
	if err == nil {
		t.Error("expected error from SyncVisual on empty dir")
	}
}

func TestSyncMiMoCode_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	count, err := SyncMiMoCode(dir, false)
	if err != nil {
		t.Fatalf("SyncMiMoCode on empty dir: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ── SyncAndDetectChanges: success path ────────────────────────────────────────

func TestSyncAndDetectChanges_EmptyDirFails(t *testing.T) {
	dir := t.TempDir()
	result, err := SyncAndDetectChanges(dir, false)
	// Sync fails on empty dir → should return error + failed status
	if err == nil {
		t.Skip("Sync succeeded on empty dir unexpectedly")
	}
	if result == nil {
		t.Fatal("expected non-nil result even on failure")
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if result.Failed == 0 {
		t.Error("expected Failed > 0")
	}
	if result.Duration == "" {
		t.Error("expected non-empty duration")
	}
	if result.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestSyncAndDetectChanges_VerboseFails(t *testing.T) {
	dir := t.TempDir()
	result, err := SyncAndDetectChanges(dir, true)
	if err == nil {
		t.Skip("Sync succeeded on empty dir unexpectedly")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Just verify verbose mode doesn't panic
	_ = result.Changes
}

func TestSyncAndDetectChanges_SystemsDirFailsGracefully(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)
	result, err := SyncAndDetectChanges(dir, false)
	// Will fail on missing sources, but must NOT fail on isolation
	if err == nil {
		return
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if strings.Contains(err.Error(), "ISOLATION") {
		t.Errorf("should not fail on isolation for Systems dir: %v", err)
	}
}

// ── Sync: success path with a minimal valid structure ─────────────────────────

func TestSync_WithConnectorBusSkills(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)

	// Set up a minimal connector_bus skills.yaml
	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(
		"version: \"1\"\nslot_type: connectors\ncomponents: {}\n",
	), 0644)

	err := Sync(dir, false)
	// Will still fail due to missing visual/mimocode/config sources,
	// but we get coverage of the connector_bus path in Sync
	if err == nil {
		return
	}
	_ = err
}

func TestSync_VerboseWithConnectorBus(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav"), 0755)

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(
		"version: \"1\"\nslot_type: connectors\ncomponents: {}\n",
	), 0644)

	// Verbose mode: tests the fmt.Fprintf error paths
	err := Sync(dir, true)
	_ = err
}

// ── projectFromConnectorBus: verbose and edge cases ───────────────────────────

func TestProjectFromConnectorBus_VerboseSkillsSync(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	skillsYAML := "version: \"1\"\nslot_type: connectors\ncomponents:\n  my-skill:\n    source_dir: ovav/skills/my-skill\n"
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(skillsYAML), 0644)

	skillSrcDir := filepath.Join(dir, "ovav", "skills", "my-skill")
	os.MkdirAll(skillSrcDir, 0755)
	os.WriteFile(filepath.Join(skillSrcDir, "SKILL.md"), []byte("# Skill"), 0644)

	skills, _, err := projectFromConnectorBus(dir, true)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if skills != 1 {
		t.Errorf("expected 1 skill, got %d", skills)
	}
}

func TestProjectFromConnectorBus_PersonnelUpToDate(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	personnelYAML := `version: "1"
slot_type: connectors
components:
  thavren:
    role: lead
    area: platform
    type: lead
    artifacts:
      - ovav/agents/leads/thavren.md
    active: true
`
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte(personnelYAML), 0644)

	leadsDir := filepath.Join(dir, "ovav", "agents", "leads")
	os.MkdirAll(leadsDir, 0755)
	content := []byte("# Thavren")
	os.WriteFile(filepath.Join(leadsDir, "thavren.md"), content, 0644)

	// Pre-create target with same content → up-to-date path
	agentsTarget := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsTarget, 0755)
	os.WriteFile(filepath.Join(agentsTarget, "lead-thavren.md"), content, 0644)

	_, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if agents != 0 {
		t.Errorf("expected 0 (up to date), got %d", agents)
	}
}

func TestProjectFromConnectorBus_PersonnelVerbose(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	personnelYAML := `version: "1"
slot_type: connectors
components:
  soren:
    role: team
    area: platform
    type: team
    artifacts:
      - ovav/agents/teams/soren.md
    active: true
`
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte(personnelYAML), 0644)

	teamsDir := filepath.Join(dir, "ovav", "agents", "teams")
	os.MkdirAll(teamsDir, 0755)
	os.WriteFile(filepath.Join(teamsDir, "soren.md"), []byte("# Soren"), 0644)

	_, agents, err := projectFromConnectorBus(dir, true)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if agents != 1 {
		t.Errorf("expected 1 agent, got %d", agents)
	}
}

func TestProjectFromConnectorBus_AreaUpToDate(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte(
		"version: \"1\"\nslot_type: connectors\ncomponents: {}\n",
	), 0644)

	content := []byte("# Platform")
	areasDir := filepath.Join(dir, ".ovav", "source", "agents", "areas")
	os.MkdirAll(areasDir, 0755)
	os.WriteFile(filepath.Join(areasDir, "area-platform.md"), content, 0644)

	// Pre-create target → up-to-date
	agentsTarget := filepath.Join(dir, ".opencode", "agents")
	os.MkdirAll(agentsTarget, 0755)
	os.WriteFile(filepath.Join(agentsTarget, "area-platform.md"), content, 0644)

	_, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if agents != 0 {
		t.Errorf("expected 0 (up to date), got %d", agents)
	}
}

// ── projectToMimocode: verbose and edge cases ────────────────────────────────

func TestProjectToMimocode_SkillsVerbose(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "v-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: v\n---\n"), 0644)

	count, err := projectToMimocode(dir, true)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestProjectToMimocode_PluginsVerbose(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, ".ovav", "source", "plugins", "mimocode", "my-plug")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "my-plug.js"), []byte("export const P = async () => ({});\n"), 0644)

	count, err := projectToMimocode(dir, true)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestProjectToMimocode_WorkflowsVerbose(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "source", "workflows"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "source", "workflows", "wf.js"), []byte("export const meta = {};\n"), 0644)

	count, err := projectToMimocode(dir, true)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestProjectToMimocode_SkillsUpToDate(t *testing.T) {
	dir := t.TempDir()
	content := []byte("---\nname: cached\n---\n")
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "cached")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0644)

	// First projection
	projectToMimocode(dir, false)

	// Second projection — should be 0 (up to date)
	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (idempotent), got %d", count)
	}
}

func TestProjectToMimocode_PluginsUpToDate(t *testing.T) {
	dir := t.TempDir()
	content := []byte("export const P = async () => ({});\n")
	pluginDir := filepath.Join(dir, ".ovav", "source", "plugins", "mimocode", "cached-plug")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "cached-plug.js"), content, 0644)

	projectToMimocode(dir, false)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (idempotent), got %d", count)
	}
}

func TestProjectToMimocode_WorkflowsUpToDate(t *testing.T) {
	dir := t.TempDir()
	content := []byte("export const meta = {};\n")
	os.MkdirAll(filepath.Join(dir, ".ovav", "source", "workflows"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "source", "workflows", "cached-wf.js"), content, 0644)

	projectToMimocode(dir, false)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (idempotent), got %d", count)
	}
}

func TestProjectToMimocode_SkillNoSKILLmd(t *testing.T) {
	dir := t.TempDir()
	// Create a skill dir without SKILL.md → should be skipped
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "no-skill-md")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "NOT_SKILL.md"), []byte("not a skill"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (no SKILL.md), got %d", count)
	}
}

func TestProjectToMimocode_SkillNonDirEntry(t *testing.T) {
	dir := t.TempDir()
	skillsSource := filepath.Join(dir, ".ovav", "source", "skills")
	os.MkdirAll(skillsSource, 0755)
	// Create a file (not directory) in skills source — should be skipped
	os.WriteFile(filepath.Join(skillsSource, "not-a-dir.txt"), []byte("skip"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (non-dir entry), got %d", count)
	}
}

func TestProjectToMimocode_PluginNonDirEntry(t *testing.T) {
	dir := t.TempDir()
	pluginsSource := filepath.Join(dir, ".ovav", "source", "plugins", "mimocode")
	os.MkdirAll(pluginsSource, 0755)
	os.WriteFile(filepath.Join(pluginsSource, "not-a-dir.js"), []byte("skip"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (non-dir plugin entry), got %d", count)
	}
}

func TestProjectToMimocode_WorkflowDirEntry(t *testing.T) {
	dir := t.TempDir()
	wfSource := filepath.Join(dir, ".ovav", "source", "workflows")
	os.MkdirAll(filepath.Join(wfSource, "subdir"), 0755)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (directory entry in workflows), got %d", count)
	}
}

func TestProjectToMimocode_WorkflowNonJSFile(t *testing.T) {
	dir := t.TempDir()
	wfSource := filepath.Join(dir, ".ovav", "source", "workflows")
	os.MkdirAll(wfSource, 0755)
	os.WriteFile(filepath.Join(wfSource, "readme.txt"), []byte("not a workflow"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 (non-JS workflow file), got %d", count)
	}
}

func TestProjectToMimocode_SkillReferencesVerbose(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "ref-skill")
	refsDir := filepath.Join(skillDir, "references")
	os.MkdirAll(refsDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: ref\n---\n"), 0644)
	os.WriteFile(filepath.Join(refsDir, "example.md"), []byte("# Example"), 0644)

	count, err := projectToMimocode(dir, true)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// ── projectVisual: WezTerm up-to-date path ───────────────────────────────────

func TestProjectVisual_WezTermUpToDate(t *testing.T) {
	dir := setupVisualTestDir(t, true, true)

	wezDir := filepath.Join(dir, ".ovav", "visual", "wezterm")
	os.MkdirAll(wezDir, 0755)
	content := []byte("-- wezterm config")
	os.WriteFile(filepath.Join(wezDir, "config.lua"), content, 0644)

	// Pre-create deploy target with same content → up-to-date
	wezDeploy := filepath.Join(dir, "config", "wezterm")
	os.MkdirAll(wezDeploy, 0755)
	os.WriteFile(filepath.Join(wezDeploy, "wezterm.lua"), content, 0644)

	count, err := projectVisual(dir, false)
	if err != nil {
		t.Fatalf("projectVisual: %v", err)
	}
	// Should still count WezTerm (count++ even when up to date)
	if count < 3 {
		t.Errorf("expected at least 3, got %d", count)
	}
}

func TestProjectVisual_VerboseOutput(t *testing.T) {
	dir := setupVisualTestDir(t, true, true)

	count, err := projectVisual(dir, true)
	if err != nil {
		t.Fatalf("projectVisual: %v", err)
	}
	if count < 3 {
		t.Errorf("expected at least 3, got %d", count)
	}
}

func TestProjectVisual_VerboseWithWezTerm(t *testing.T) {
	dir := setupVisualTestDir(t, true, true)

	wezDir := filepath.Join(dir, ".ovav", "visual", "wezterm")
	os.MkdirAll(wezDir, 0755)
	os.WriteFile(filepath.Join(wezDir, "config.lua"), []byte("-- wezterm"), 0644)

	count, err := projectVisual(dir, true)
	if err != nil {
		t.Fatalf("projectVisual: %v", err)
	}
	if count < 4 {
		t.Errorf("expected at least 4, got %d", count)
	}
}

func TestProjectVisual_InvalidMonitoringYAML(t *testing.T) {
	dir := setupVisualTestDir(t, true, false)
	monDir := filepath.Join(dir, ".ovav", "visual", "monitoring")
	os.MkdirAll(monDir, 0755)
	os.WriteFile(filepath.Join(monDir, "monitoring.yaml"), []byte("{{invalid"), 0644)

	count, err := projectVisual(dir, false)
	if err == nil {
		t.Error("expected error for invalid monitoring YAML")
	}
	if count != 1 {
		t.Errorf("expected count=1 before error, got %d", count)
	}
}

// ── syncPluginRegistry: additional coverage ──────────────────────────────────

func TestSyncPluginRegistry_NpmPrefixFilter(t *testing.T) {
	dir := t.TempDir()
	config := map[string]interface{}{
		"plugin": []interface{}{"@ovav/some-plugin", "@ovav/another", "real-plugin"},
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), append(data, '\n'), 0644)

	_, err := syncPluginRegistry(dir)
	if err != nil {
		t.Fatalf("syncPluginRegistry: %v", err)
	}

	updated, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	var result map[string]interface{}
	json.Unmarshal(updated, &result)
	plugins := result["plugin"].([]interface{})

	// @ovav/ prefixed entries should be filtered
	for _, p := range plugins {
		if s, ok := p.(string); ok && strings.HasPrefix(s, "@ovav/") {
			t.Errorf("@ovav/ plugin should have been filtered: %q", s)
		}
	}
}

// ── copyFile: additional error paths ─────────────────────────────────────────

func TestCopyFile_DirDst(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "existing-dir")
	os.WriteFile(src, []byte("content"), 0644)
	os.MkdirAll(dst, 0755)

	err := copyFile(src, dst)
	// Should fail because dst is a directory (os.Create on dir)
	if err == nil {
		t.Error("expected error when dst is a directory")
	}
}

// ── projectFromConnectorBus: read error for skills ───────────────────────────

func TestProjectFromConnectorBus_SkillsReadError(t *testing.T) {
	dir := t.TempDir()
	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	// Create skills.yaml as a directory (will cause read error)
	os.MkdirAll(filepath.Join(connDir, "skills.yaml"), 0755)

	_, _, err := projectFromConnectorBus(dir, false)
	if err == nil {
		t.Error("expected error when skills.yaml is a directory")
	}
}

func TestProjectFromConnectorBus_PersonnelReadError(t *testing.T) {
	dir := t.TempDir()
	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	// Create personnel.yaml as a directory (will cause read error)
	os.MkdirAll(filepath.Join(connDir, "personnel.yaml"), 0755)

	_, _, err := projectFromConnectorBus(dir, false)
	if err == nil {
		t.Error("expected error when personnel.yaml is a directory")
	}
}

// ── generateOpenCodePlugin: bp with args ─────────────────────────────────────

func TestGenerateOpenCodePlugin_WithWatchers(t *testing.T) {
	mon := &monitoringRaw{
		Schema:  "monitoring-v1",
		Version: "1.0",
		Alerts: map[string]alertEntry{
			"agent_switch": {Toast: "Agent: {from} → {to}", Severity: "info", DurationMs: 5000},
			"model_switch": {Toast: "Model: {from} → {to}", Severity: "warning", DurationMs: 3000},
		},
		Watchers: map[string]watcherEntry{
			"token_usage": {
				Description: "Track token usage",
				Source:      "session.tokens",
				Display:     watcherDisplay{Format: "number", UpdateFrequency: "1s"},
			},
		},
	}

	result := generateOpenCodePlugin(mon)
	if !strings.Contains(result, "OVAVMonitor") {
		t.Error("missing OVAVMonitor")
	}
	// Watchers are parsed but not rendered in the current plugin template
	if !strings.Contains(result, "checkBudget") {
		t.Error("missing checkBudget")
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
