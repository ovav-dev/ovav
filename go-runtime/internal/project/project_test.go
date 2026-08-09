package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── filesEqual (existing tests, kept) ────────────────────────────────────────

func TestFilesEqual_BothExist_Same(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("hello"), 0644)
	os.WriteFile(b, []byte("hello"), 0644)

	if !filesEqual(a, b) {
		t.Error("filesEqual should return true for identical files")
	}
}

func TestFilesEqual_BothExist_Different(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("hello"), 0644)
	os.WriteFile(b, []byte("world"), 0644)

	if filesEqual(a, b) {
		t.Error("filesEqual should return false for different files")
	}
}

func TestFilesEqual_OneMissing(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	os.WriteFile(a, []byte("hello"), 0644)

	if filesEqual(a, filepath.Join(dir, "missing.txt")) {
		t.Error("filesEqual should return false when one file is missing")
	}
}

func TestFilesEqual_BothMissing(t *testing.T) {
	dir := t.TempDir()
	if filesEqual(filepath.Join(dir, "x"), filepath.Join(dir, "y")) {
		t.Error("filesEqual should return false when both files are missing")
	}
}

// ── jsTemplate ───────────────────────────────────────────────────────────────

func TestJSTemplate(t *testing.T) {
	got := jsTemplate("hello ${x}")
	want := "`hello ${x}`"
	if got != want {
		t.Errorf("jsTemplate() = %q, want %q", got, want)
	}
}

// ── buildAlertSwitchJS ───────────────────────────────────────────────────────

func TestBuildAlertSwitchJS_NoAlert(t *testing.T) {
	alerts := map[string]alertEntry{}
	got := buildAlertSwitchJS("agent_switch", alerts)
	if !strings.Contains(got, "no config") {
		t.Errorf("expected 'no config' comment, got %q", got)
	}
}

func TestBuildAlertSwitchJS_WithAgentAlert(t *testing.T) {
	alerts := map[string]alertEntry{
		"agent_switch": {Toast: "Switched from {from} to {to}", DurationMs: 5000},
	}
	got := buildAlertSwitchJS("agent_switch", alerts)
	if !strings.Contains(got, "toast(") {
		t.Error("expected toast() call in output")
	}
	if !strings.Contains(got, "5000") {
		t.Error("expected duration 5000 in output")
	}
	if !strings.Contains(got, "state.currentAgent") {
		t.Error("expected state.currentAgent substitution for agent alert")
	}
}

func TestBuildAlertSwitchJS_WithModelAlert(t *testing.T) {
	alerts := map[string]alertEntry{
		"model_switch": {Toast: "Model {from} → {to}", DurationMs: 3000},
	}
	got := buildAlertSwitchJS("model_switch", alerts)
	if !strings.Contains(got, "state.currentModel") {
		t.Error("expected state.currentModel substitution for model alert")
	}
}

func TestBuildAlertSwitchJS_DefaultDuration(t *testing.T) {
	alerts := map[string]alertEntry{
		"agent_switch": {Toast: "switch", DurationMs: 0},
	}
	got := buildAlertSwitchJS("agent_switch", alerts)
	if !strings.Contains(got, "3000") {
		t.Error("expected default duration 3000 when DurationMs <= 0")
	}
}

// ── copyFile ─────────────────────────────────────────────────────────────────

func TestCopyFile_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("copy me")
	os.WriteFile(src, content, 0644)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, _ := os.ReadFile(dst)
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.sh")
	dst := filepath.Join(dir, "dst.sh")
	os.WriteFile(src, []byte("#!/bin/sh"), 0755)

	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	info, _ := os.Stat(dst)
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions: got %o, want %o", info.Mode().Perm(), 0755)
	}
}

func TestCopyFile_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "nope"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Error("expected error when source is missing")
	}
}

// ── copyDir ──────────────────────────────────────────────────────────────────

func TestCopyDir_Success(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")

	// Create src structure: src/a.txt + src/sub/b.txt
	os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("A"), 0644)
	os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("B"), 0644)

	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	// Verify files
	got, _ := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	if string(got) != "A" {
		t.Errorf("a.txt: got %q, want %q", got, "A")
	}
	got, _ = os.ReadFile(filepath.Join(dstDir, "sub", "b.txt"))
	if string(got) != "B" {
		t.Errorf("sub/b.txt: got %q, want %q", got, "B")
	}
}

func TestCopyDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "empty_src")
	dstDir := filepath.Join(dir, "empty_dst")
	os.MkdirAll(srcDir, 0755)

	err := copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyDir on empty dir: %v", err)
	}
}

// ── generateOpenCodeTheme ────────────────────────────────────────────────────

func TestGenerateOpenCodeTheme_Structure(t *testing.T) {
	theme := &themeRaw{
		Schema:  "theme-v1",
		Name:    "OVAV",
		Version: "1.0.0",
		Brand: map[string]string{
			"thavren":     "#00CED1",
			"eidren":      "#2ECC71",
			"ovav_core":   "#3498DB",
			"ovav_accent": "#E74C3C",
		},
		Semantic: map[string]string{
			"success": "#27AE60",
			"error":   "#E74C3C",
			"warning": "#F39C12",
			"info":    "#3498DB",
		},
		Surfaces: map[string]map[string]string{
			"dark": {
				"bg_root":        "#1E1E2E",
				"bg_panel":       "#252535",
				"bg_element":     "#2D2D3D",
				"border":         "#3D3D4D",
				"text_primary":   "#E0E0E0",
				"text_secondary": "#A0A0A0",
				"text_muted":     "#666666",
			},
		},
		Syntax: map[string]string{
			"keyword":  "#C678DD",
			"string":   "#98C379",
			"comment":  "#5C6370",
			"function": "#61AFEF",
			"type":     "#E5C07B",
		},
		Diff: map[string]string{
			"added":   "#98C379",
			"removed": "#E06C75",
			"context": "#ABB2BF",
		},
	}

	result := generateOpenCodeTheme(theme, "dark")

	// Verify top-level keys
	if result["$schema"] != "https://opencode.ai/theme.json" {
		t.Errorf("$schema: got %v", result["$schema"])
	}
	if result["name"] != "OVAV Dark" {
		t.Errorf("name: got %v", result["name"])
	}

	// Verify defs exist
	defs, ok := result["defs"].(map[string]string)
	if !ok {
		t.Fatal("defs should be map[string]string")
	}
	if defs["ovav_teal"] != "#00CED1" {
		t.Errorf("ovav_teal: got %q", defs["ovav_teal"])
	}
	if defs["ovav_bg"] != "#1E1E2E" {
		t.Errorf("ovav_bg: got %q", defs["ovav_bg"])
	}

	// Verify theme section
	themeSection, ok := result["theme"].(map[string]string)
	if !ok {
		t.Fatal("theme should be map[string]string")
	}
	if themeSection["primary"] != "ovav_teal" {
		t.Errorf("primary: got %q", themeSection["primary"])
	}
}

// ── generateOpenCodePlugin ───────────────────────────────────────────────────

func TestGenerateOpenCodePlugin_Structure(t *testing.T) {
	mon := &monitoringRaw{
		Schema:  "monitoring-v1",
		Version: "1.0.0",
		Alerts: map[string]alertEntry{
			"agent_switch": {Toast: "Agent: {from} → {to}", Severity: "info", DurationMs: 4000},
			"model_switch": {Toast: "Model: {from} → {to}", Severity: "info", DurationMs: 3000},
		},
	}

	result := generateOpenCodePlugin(mon)

	// Verify it's valid-ish JS structure
	checks := []string{
		"OVAVMonitor",
		"export const",
		"event:",
		"tool:",
		"ovav_monitor:",
		"session.status",
		"session.idle",
		"session.created",
		"checkBudget",
		"toast(",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("plugin JS missing: %q", check)
		}
	}
}

func TestGenerateOpenCodePlugin_NoAlerts(t *testing.T) {
	mon := &monitoringRaw{
		Alerts: map[string]alertEntry{},
	}

	result := generateOpenCodePlugin(mon)
	if !strings.Contains(result, "OVAVMonitor") {
		t.Error("plugin should still generate with no alerts")
	}
}

// ── syncPluginRegistry ───────────────────────────────────────────────────────

func TestSyncPluginRegistry_AddsPlugin(t *testing.T) {
	dir := t.TempDir()

	// Create opencode.json with empty plugin array
	config := map[string]interface{}{
		"plugin": []interface{}{},
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), append(data, '\n'), 0644)

	// MiMo Code has built-in plugins — syncPluginRegistry no longer auto-adds
	count, err := syncPluginRegistry(dir)
	if err != nil {
		t.Fatalf("syncPluginRegistry: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 plugins auto-registered (built-in), got %d", count)
	}
}

func TestSyncPluginRegistry_FiltersNpmRefs(t *testing.T) {
	dir := t.TempDir()

	// Create opencode.json with npm references
	config := map[string]interface{}{
		"plugin": []interface{}{"@ovav/opencode-tui", "some-other-plugin"},
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), append(data, '\n'), 0644)

	// Create the local plugin file
	pluginDir := filepath.Join(dir, ".opencode", "plugins")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "ovav-monitor.js"), []byte("// plugin"), 0644)

	_, err := syncPluginRegistry(dir)
	if err != nil {
		t.Fatalf("syncPluginRegistry: %v", err)
	}

	updated, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	var result map[string]interface{}
	json.Unmarshal(updated, &result)
	plugins := result["plugin"].([]interface{})

	// @ovav/opencode-tui should be filtered out
	for _, p := range plugins {
		if s, ok := p.(string); ok && s == "@ovav/opencode-tui" {
			t.Error("@ovav/opencode-tui should have been filtered out")
		}
	}

	// some-other-plugin should remain
	found := false
	for _, p := range plugins {
		if s, ok := p.(string); ok && s == "some-other-plugin" {
			found = true
		}
	}
	if !found {
		t.Error("some-other-plugin should have been preserved")
	}
}

func TestSyncPluginRegistry_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	_, err := syncPluginRegistry(dir)
	if err == nil {
		t.Error("expected error when opencode.json doesn't exist")
	}
}

func TestSyncPluginRegistry_NoPluginField(t *testing.T) {
	dir := t.TempDir()

	// Create opencode.json without plugin field
	config := map[string]interface{}{"other": "value"}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), append(data, '\n'), 0644)

	// MiMo Code has built-in plugins — syncPluginRegistry no longer auto-adds
	count, err := syncPluginRegistry(dir)
	if err != nil {
		t.Fatalf("syncPluginRegistry: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 plugins auto-registered (built-in), got %d", count)
	}
}

// ── copyDir error paths ──────────────────────────────────────────────────────

func TestCopyDir_SourceNotExists(t *testing.T) {
	dir := t.TempDir()
	err := copyDir(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Error("expected error when source directory does not exist")
	}
}

// ── generateOpenCodeTheme edge cases ─────────────────────────────────────────

func TestGenerateOpenCodeTheme_NilMaps(t *testing.T) {
	theme := &themeRaw{}
	// Should not panic with nil maps — all lookups return ""
	result := generateOpenCodeTheme(theme, "dark")
	if result["name"] != "OVAV Dark" {
		t.Errorf("name: got %v", result["name"])
	}
	defs, ok := result["defs"].(map[string]string)
	if !ok {
		t.Fatal("defs should be map[string]string")
	}
	if defs["ovav_teal"] != "" {
		t.Errorf("ovav_teal should be empty with nil brand map, got %q", defs["ovav_teal"])
	}
	if defs["ovav_bg"] != "" {
		t.Errorf("ovav_bg should be empty with nil surfaces, got %q", defs["ovav_bg"])
	}
}

func TestGenerateOpenCodeTheme_EmptyDarkSurface(t *testing.T) {
	theme := &themeRaw{
		Brand:    map[string]string{"thavren": "#111"},
		Semantic: map[string]string{"success": "#222"},
		Surfaces: map[string]map[string]string{
			"dark": {}, // empty — all keys return ""
		},
		Syntax: map[string]string{"keyword": "#333"},
		Diff:   map[string]string{"added": "#444"},
	}
	result := generateOpenCodeTheme(theme, "dark")
	defs := result["defs"].(map[string]string)
	if defs["ovav_teal"] != "#111" {
		t.Errorf("ovav_teal: got %q", defs["ovav_teal"])
	}
	if defs["ovav_bg"] != "" {
		t.Errorf("ovav_bg should be empty with empty dark surface, got %q", defs["ovav_bg"])
	}
	if defs["ovav_syn_keyword"] != "#333" {
		t.Errorf("ovav_syn_keyword: got %q", defs["ovav_syn_keyword"])
	}
}

// ── projectFromConnectorBus ──────────────────────────────────────────────────

func TestProjectFromConnectorBus_NoFiles(t *testing.T) {
	dir := t.TempDir()
	skills, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skills != 0 || agents != 0 {
		t.Errorf("expected 0,0 got %d,%d", skills, agents)
	}
}

func TestProjectFromConnectorBus_SkillsSync(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	skillsYAML := "version: \"1\"\nslot_type: connectors\ncomponents:\n  my-skill:\n    source_dir: ovav/skills/my-skill\n    owner_profile: platform\n    risk_level: low\n"
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(skillsYAML), 0644)

	skillSrcDir := filepath.Join(dir, "ovav", "skills", "my-skill")
	os.MkdirAll(skillSrcDir, 0755)
	os.WriteFile(filepath.Join(skillSrcDir, "SKILL.md"), []byte("# My Skill\nContent"), 0644)

	skills, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if skills != 1 {
		t.Errorf("expected 1 skill synced, got %d", skills)
	}
	if agents != 0 {
		t.Errorf("expected 0 agents, got %d", agents)
	}

	target := filepath.Join(dir, ".opencode", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target skill not created: %v", err)
	}
	if string(data) != "# My Skill\nContent" {
		t.Errorf("content mismatch: %q", data)
	}
}

func TestProjectFromConnectorBus_SkillsWithReferences(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	skillsYAML := "version: \"1\"\nslot_type: connectors\ncomponents:\n  ref-skill:\n    source_dir: ovav/skills/ref-skill\n"
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(skillsYAML), 0644)

	skillSrcDir := filepath.Join(dir, "ovav", "skills", "ref-skill")
	os.MkdirAll(filepath.Join(skillSrcDir, "references"), 0755)
	os.WriteFile(filepath.Join(skillSrcDir, "SKILL.md"), []byte("# Ref Skill"), 0644)
	os.WriteFile(filepath.Join(skillSrcDir, "references", "ref1.md"), []byte("ref content"), 0644)

	skills, _, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if skills != 1 {
		t.Errorf("expected 1 skill, got %d", skills)
	}

	refTarget := filepath.Join(dir, ".opencode", "skills", "ref-skill", "references", "ref1.md")
	data, err := os.ReadFile(refTarget)
	if err != nil {
		t.Fatalf("reference not copied: %v", err)
	}
	if string(data) != "ref content" {
		t.Errorf("ref content: %q", data)
	}
}

func TestProjectFromConnectorBus_SkillEmptySourceDir(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	skillsYAML := "version: \"1\"\nslot_type: connectors\ncomponents:\n  empty:\n    source_dir: \"\"\n"
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(skillsYAML), 0644)

	skills, _, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skills != 0 {
		t.Errorf("expected 0 skills for empty source_dir, got %d", skills)
	}
}

func TestProjectFromConnectorBus_InvalidSkillsYAML(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte("{{invalid yaml"), 0644)

	_, _, err := projectFromConnectorBus(dir, false)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestProjectFromConnectorBus_InvalidPersonnelYAML(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte("{{invalid yaml"), 0644)

	_, _, err := projectFromConnectorBus(dir, false)
	if err == nil {
		t.Error("expected error for invalid personnel YAML")
	}
}

func TestProjectFromConnectorBus_PersonnelSync(t *testing.T) {
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
    permissions: full
    active: true
  inactive-agent:
    role: team
    area: platform
    type: team
    artifacts:
      - ovav/agents/teams/inactive.md
    permissions: read
    active: false
`
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte(personnelYAML), 0644)

	leadsDir := filepath.Join(dir, "ovav", "agents", "leads")
	os.MkdirAll(leadsDir, 0755)
	os.WriteFile(filepath.Join(leadsDir, "thavren.md"), []byte("# Thavren"), 0644)

	skills, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if skills != 0 {
		t.Errorf("expected 0 skills, got %d", skills)
	}
	if agents != 1 {
		t.Errorf("expected 1 agent synced, got %d", agents)
	}

	target := filepath.Join(dir, ".opencode", "agents", "lead-thavren.md")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target agent not created: %v", err)
	}
	if string(data) != "# Thavren" {
		t.Errorf("content: %q", data)
	}
}

func TestProjectFromConnectorBus_PersonnelTeamArtifact(t *testing.T) {
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

	_, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if agents != 1 {
		t.Errorf("expected 1 agent, got %d", agents)
	}

	target := filepath.Join(dir, ".opencode", "agents", "team-soren.md")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("team agent not created: %v", err)
	}
}

func TestProjectFromConnectorBus_AreaFiles(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	// Empty personnel (needed to enter the personnel block and reach area logic)
	personnelYAML := "version: \"1\"\nslot_type: connectors\ncomponents: {}\n"
	os.WriteFile(filepath.Join(connDir, "personnel.yaml"), []byte(personnelYAML), 0644)

	areasDir := filepath.Join(dir, ".ovav", "source", "agents", "areas")
	os.MkdirAll(areasDir, 0755)
	os.WriteFile(filepath.Join(areasDir, "area-platform.md"), []byte("# Platform"), 0644)
	os.WriteFile(filepath.Join(areasDir, "not-area.md"), []byte("# Skip"), 0644)       // no area- prefix
	os.WriteFile(filepath.Join(areasDir, "area-security.txt"), []byte("# Skip"), 0644) // wrong ext

	_, agents, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("projectFromConnectorBus: %v", err)
	}
	if agents != 1 {
		t.Errorf("expected 1 area agent, got %d", agents)
	}

	target := filepath.Join(dir, ".opencode", "agents", "area-platform.md")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("area file not copied: %v", err)
	}
}

func TestProjectFromConnectorBus_SkillUpToDate(t *testing.T) {
	dir := t.TempDir()

	connDir := filepath.Join(dir, ".ovav", "connector_bus.legacy", "connectors")
	os.MkdirAll(connDir, 0755)

	skillsYAML := "version: \"1\"\nslot_type: connectors\ncomponents:\n  cached:\n    source_dir: ovav/skills/cached\n"
	os.WriteFile(filepath.Join(connDir, "skills.yaml"), []byte(skillsYAML), 0644)

	content := []byte("# Cached Skill")
	skillSrcDir := filepath.Join(dir, "ovav", "skills", "cached")
	os.MkdirAll(skillSrcDir, 0755)
	os.WriteFile(filepath.Join(skillSrcDir, "SKILL.md"), content, 0644)

	// Pre-create target with same content → filesEqual returns true → skip
	targetDir := filepath.Join(dir, ".opencode", "skills", "cached")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "SKILL.md"), content, 0644)

	skills, _, err := projectFromConnectorBus(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skills != 0 {
		t.Errorf("expected 0 skills (up to date), got %d", skills)
	}
}

// ── projectVisual ────────────────────────────────────────────────────────────

const testThemeYAML = `schema: theme-v1
name: OVAV
version: "1.0"
brand:
  thavren: "#00CED1"
  eidren: "#2ECC71"
  ovav_core: "#3498DB"
  ovav_accent: "#E74C3C"
semantic:
  success: "#27AE60"
  error: "#E74C3C"
  warning: "#F39C12"
  info: "#3498DB"
surfaces:
  dark:
    bg_root: "#1E1E2E"
    bg_panel: "#252535"
    bg_element: "#2D2D3D"
    border: "#3D3D4D"
    text_primary: "#E0E0E0"
    text_secondary: "#A0A0A0"
    text_muted: "#666666"
syntax:
  keyword: "#C678DD"
  string: "#98C379"
  comment: "#5C6370"
  function: "#61AFEF"
  type: "#E5C07B"
diff:
  added: "#98C379"
  removed: "#E06C75"
  context: "#ABB2BF"
`

const testMonitoringYAML = `schema: monitoring-v1
version: "1.0"
description: test monitoring
watchers: {}
alerts:
  agent_switch:
    trigger: agent change
    severity: info
    toast: "Agent: {from} to {to}"
    duration_ms: 4000
  model_switch:
    trigger: model change
    severity: info
    toast: "Model: {from} to {to}"
    duration_ms: 3000
`

func setupVisualTestDir(t *testing.T, theme, monitoring bool) string {
	t.Helper()
	dir := t.TempDir()
	if theme {
		themeDir := filepath.Join(dir, ".ovav", "visual", "theme")
		os.MkdirAll(themeDir, 0755)
		os.WriteFile(filepath.Join(themeDir, "theme.yaml"), []byte(testThemeYAML), 0644)
	}
	if monitoring {
		monDir := filepath.Join(dir, ".ovav", "visual", "monitoring")
		os.MkdirAll(monDir, 0755)
		os.WriteFile(filepath.Join(monDir, "monitoring.yaml"), []byte(testMonitoringYAML), 0644)
	}
	return dir
}

func TestProjectVisual_FullPipeline(t *testing.T) {
	dir := setupVisualTestDir(t, true, true)

	count, err := projectVisual(dir, false)
	if err != nil {
		t.Fatalf("projectVisual: %v", err)
	}
	if count < 3 {
		t.Errorf("expected at least 3 artifacts, got %d", count)
	}

	// Verify theme JSON (dark and light variants)
	themeJSONDarkPath := filepath.Join(dir, ".opencode", "themes", "ovav-dark.json")
	data, err := os.ReadFile(themeJSONDarkPath)
	if err != nil {
		t.Fatalf("ovav-dark.json not created: %v", err)
	}
	var themeMap map[string]interface{}
	if err := json.Unmarshal(data, &themeMap); err != nil {
		t.Fatalf("invalid theme JSON: %v", err)
	}
	if themeMap["name"] != "OVAV Dark" {
		t.Errorf("theme name: got %v", themeMap["name"])
	}

	// Verify light theme also exists
	themeJSONLightPath := filepath.Join(dir, ".opencode", "themes", "ovav-light.json")
	_, err = os.ReadFile(themeJSONLightPath)
	if err != nil {
		t.Fatalf("ovav-light.json not created: %v", err)
	}

	// Verify plugin JS
	pluginPath := filepath.Join(dir, ".opencode", "plugins", "ovav-monitor.js")
	pluginData, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("plugin JS not created: %v", err)
	}
	if !strings.Contains(string(pluginData), "OVAVMonitor") {
		t.Error("plugin JS missing OVAVMonitor")
	}

	// Verify tui.json
	tuiPath := filepath.Join(dir, "tui.json")
	if _, err := os.Stat(tuiPath); err != nil {
		t.Errorf("tui.json not created: %v", err)
	}
}

func TestProjectVisual_MissingTheme(t *testing.T) {
	dir := t.TempDir()
	_, err := projectVisual(dir, false)
	if err == nil {
		t.Error("expected error when theme.yaml is missing")
	}
}

func TestProjectVisual_MissingMonitoring(t *testing.T) {
	dir := setupVisualTestDir(t, true, false)

	count, err := projectVisual(dir, false)
	if err == nil {
		t.Error("expected error when monitoring.yaml is missing")
	}
	if count != 2 {
		t.Errorf("expected count=2 (theme dark+light before error), got %d", count)
	}
}

func TestProjectVisual_WithWezTerm(t *testing.T) {
	dir := setupVisualTestDir(t, true, true)

	// Create wezterm canonical config
	wezDir := filepath.Join(dir, ".ovav", "visual", "wezterm")
	os.MkdirAll(wezDir, 0755)
	os.WriteFile(filepath.Join(wezDir, "config.lua"), []byte("-- wezterm config"), 0644)

	count, err := projectVisual(dir, false)
	if err != nil {
		t.Fatalf("projectVisual: %v", err)
	}
	if count < 4 {
		t.Errorf("expected at least 4 artifacts (theme+plugin+tui+wezterm), got %d", count)
	}

	// Verify wezterm deploy target
	wezDeploy := filepath.Join(dir, "config", "wezterm", "wezterm.lua")
	data, err := os.ReadFile(wezDeploy)
	if err != nil {
		t.Fatalf("wezterm deploy not created: %v", err)
	}
	if string(data) != "-- wezterm config" {
		t.Errorf("wezterm content: %q", data)
	}
}

func TestProjectVisual_InvalidThemeYAML(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, ".ovav", "visual", "theme")
	os.MkdirAll(themeDir, 0755)
	os.WriteFile(filepath.Join(themeDir, "theme.yaml"), []byte("{{invalid"), 0644)

	_, err := projectVisual(dir, false)
	if err == nil {
		t.Error("expected error for invalid theme YAML")
	}
}

// ── Sync ─────────────────────────────────────────────────────────────────────

func TestSync_EmptyDirReturnsError(t *testing.T) {
	dir := t.TempDir()
	err := Sync(dir, false)
	if err == nil {
		t.Error("expected error from Sync on empty directory")
	}
	if !strings.Contains(err.Error(), "projector(s) failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSync_VerboseOutput(t *testing.T) {
	dir := t.TempDir()
	// Verbose mode should not panic even on failures
	err := Sync(dir, true)
	if err == nil {
		t.Error("expected error from Sync on empty directory")
	}
}

// ── syncPluginRegistry edge cases ────────────────────────────────────────────

func TestSyncPluginRegistry_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("{{invalid json"), 0644)

	_, err := syncPluginRegistry(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSyncPluginRegistry_PluginFieldNotArray(t *testing.T) {
	dir := t.TempDir()
	config := map[string]interface{}{"plugin": "not-an-array"}
	data, _ := json.Marshal(config)
	os.WriteFile(filepath.Join(dir, "opencode.json"), data, 0644)

	_, err := syncPluginRegistry(dir)
	if err == nil {
		t.Error("expected error when plugin field is not an array")
	}
}

func TestSyncPluginRegistry_NonStringPluginEntry(t *testing.T) {
	dir := t.TempDir()
	config := map[string]interface{}{
		"plugin": []interface{}{42, "valid-plugin"},
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(dir, "opencode.json"), append(data, '\n'), 0644)

	pluginDir := filepath.Join(dir, ".opencode", "plugins")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "ovav-monitor.js"), []byte("// plugin"), 0644)

	_, err := syncPluginRegistry(dir)
	if err != nil {
		t.Fatalf("syncPluginRegistry: %v", err)
	}

	// Verify non-string entry was preserved
	updated, _ := os.ReadFile(filepath.Join(dir, "opencode.json"))
	var result map[string]interface{}
	json.Unmarshal(updated, &result)
	plugins := result["plugin"].([]interface{})
	foundInt := false
	for _, p := range plugins {
		if _, ok := p.(float64); ok {
			foundInt = true
		}
	}
	if !foundInt {
		t.Error("non-string plugin entry should be preserved")
	}
}

// ── projectToMimocode tests ────────────────────────────────────────────────

func TestProjectToMimocode_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode on empty dir: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 artifacts, got %d", count)
	}
}

func TestProjectToMimocode_SkillsSync(t *testing.T) {
	dir := t.TempDir()

	// Create canonical skill source
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n# Test Skill\n"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 artifact, got %d", count)
	}

	// Verify target exists
	target := filepath.Join(dir, ".mimocode", "skills", "test-skill", "SKILL.md")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("skill was not projected to .mimocode/skills/")
	}
}

func TestProjectToMimocode_SkillsWithReferences(t *testing.T) {
	dir := t.TempDir()

	// Create canonical skill with references
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "test-skill")
	refsDir := filepath.Join(skillDir, "references")
	os.MkdirAll(refsDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644)
	os.WriteFile(filepath.Join(refsDir, "example.md"), []byte("# Example\n"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 artifact, got %d", count)
	}

	// Verify references were synced
	targetRefs := filepath.Join(dir, ".mimocode", "skills", "test-skill", "references", "example.md")
	if _, err := os.Stat(targetRefs); os.IsNotExist(err) {
		t.Error("references were not projected")
	}
}

func TestProjectToMimocode_PluginsSync(t *testing.T) {
	dir := t.TempDir()

	// Create canonical MiMo plugin source
	pluginDir := filepath.Join(dir, ".ovav", "source", "plugins", "mimocode", "my-plugin")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "my-plugin.js"), []byte("export const MyPlugin = async () => ({});\n"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 artifact, got %d", count)
	}

	// Verify target exists
	target := filepath.Join(dir, ".mimocode", "plugins", "my-plugin.js")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("plugin was not projected to .mimocode/plugins/")
	}
}

func TestProjectToMimocode_WorkflowsSync(t *testing.T) {
	dir := t.TempDir()

	// Create canonical workflow source
	os.MkdirAll(filepath.Join(dir, ".ovav", "source", "workflows"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "source", "workflows", "test-wf.js"), []byte("export const meta = { name: 'test' };\n"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 artifact, got %d", count)
	}

	// Verify target exists
	target := filepath.Join(dir, ".mimocode", "workflows", "test-wf.js")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error("workflow was not projected to .mimocode/workflows/")
	}
}

func TestProjectToMimocode_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Create canonical sources
	skillDir := filepath.Join(dir, ".ovav", "source", "skills", "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644)

	// First projection
	count1, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("first projection: %v", err)
	}
	if count1 != 1 {
		t.Errorf("first projection: expected 1, got %d", count1)
	}

	// Second projection — should be 0 (up to date)
	count2, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("second projection: %v", err)
	}
	if count2 != 0 {
		t.Errorf("second projection: expected 0 (idempotent), got %d", count2)
	}
}

func TestProjectToMimocode_FullPipeline(t *testing.T) {
	dir := t.TempDir()

	// Set up complete canonical structure
	// Skills
	for _, skill := range []string{"alpha", "beta"} {
		skillDir := filepath.Join(dir, ".ovav", "source", "skills", skill)
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: "+skill+"\n---\n"), 0644)
	}

	// Plugins
	pluginDir := filepath.Join(dir, ".ovav", "source", "plugins", "mimocode", "gov")
	os.MkdirAll(pluginDir, 0755)
	os.WriteFile(filepath.Join(pluginDir, "gov.js"), []byte("export const Gov = async () => ({});\n"), 0644)

	// Workflows
	os.MkdirAll(filepath.Join(dir, ".ovav", "source", "workflows"), 0755)
	os.WriteFile(filepath.Join(dir, ".ovav", "source", "workflows", "wf.js"), []byte("export const meta = { name: 'wf' };\n"), 0644)

	count, err := projectToMimocode(dir, false)
	if err != nil {
		t.Fatalf("projectToMimocode: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 artifacts (2 skills + 1 plugin + 1 workflow), got %d", count)
	}

	// Verify all targets
	targets := []string{
		filepath.Join(dir, ".mimocode", "skills", "alpha", "SKILL.md"),
		filepath.Join(dir, ".mimocode", "skills", "beta", "SKILL.md"),
		filepath.Join(dir, ".mimocode", "plugins", "gov.js"),
		filepath.Join(dir, ".mimocode", "workflows", "wf.js"),
	}
	for _, target := range targets {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			t.Errorf("missing projected file: %s", target)
		}
	}
}
