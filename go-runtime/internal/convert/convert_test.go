package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAreas(t *testing.T) {
	areas, err := loadAreas("../../internal/agents/areas")
	if err != nil {
		t.Fatalf("loadAreas: %v", err)
	}
	if len(areas) < 9 {
		t.Fatalf("expected at least 9 areas, got %d", len(areas))
	}

	// Verify all 10 areas are present
	areaIDs := map[string]bool{}
	for _, a := range areas {
		areaIDs[a.ID] = true
		if a.Name == "" {
			t.Errorf("area %q has empty name", a.ID)
		}
		if a.Lead == "" {
			t.Errorf("area %q has empty lead", a.ID)
		}
		if len(a.Functions) < 5 {
			t.Errorf("area %q: expected at least 5 functions, got %d", a.ID, len(a.Functions))
		}
		if len(a.Limitations) < 5 {
			t.Errorf("area %q: expected at least 5 limitations, got %d", a.ID, len(a.Limitations))
		}
		if !strings.Contains(a.HardStop, "HARD STOP") {
			t.Errorf("area %q: hard stop should contain 'HARD STOP'", a.ID)
		}
	}

	expectedIDs := []string{
		"adversarial-intelligence", "commercial-growth", "devops-infrastructure",
		"digital-product", "education-career", "health-performance",
		"legal-compliance", "platform-engineering", "research-intelligence", "ux-design",
	}
	for _, id := range expectedIDs {
		if !areaIDs[id] {
			t.Errorf("missing expected area: %s", id)
		}
	}
}

func TestLoadLeads(t *testing.T) {
	leads, err := loadLeads("../../internal/agents/leads")
	if err != nil {
		t.Fatalf("loadLeads: %v", err)
	}
	if len(leads) < 9 {
		t.Fatalf("expected at least 9 leads, got %d", len(leads))
	}
	for _, l := range leads {
		if l.Name == "" {
			t.Errorf("lead %q has empty name", l.ID)
		}
		if l.DisplayName == "" {
			t.Errorf("lead %q has empty display_name", l.ID)
		}
		if l.Area == "" {
			t.Errorf("lead %q has empty area", l.ID)
		}
		if len(l.Functions) < 5 {
			t.Errorf("lead %q: expected at least 5 functions, got %d", l.ID, len(l.Functions))
		}
		if len(l.Squad) < 2 {
			t.Errorf("lead %q: expected at least 2 squad members, got %d", l.ID, len(l.Squad))
		}
	}
}

func TestLoadTeams(t *testing.T) {
	teams, err := loadTeams("../../internal/agents/teams")
	if err != nil {
		t.Fatalf("loadTeams: %v", err)
	}
	if len(teams) < 40 {
		t.Fatalf("expected at least 40 teams, got %d", len(teams))
	}
	for _, tm := range teams {
		if tm.Name == "" {
			t.Errorf("team %q has empty name", tm.ID)
		}
		if tm.Area == "" {
			t.Errorf("team %q has empty area", tm.ID)
		}
		if tm.Lead == "" {
			t.Errorf("team %q has empty lead", tm.ID)
		}
	}
}

func TestOpenCodeConverter_Area(t *testing.T) {
	areas, err := loadAreas("../../internal/agents/areas")
	if err != nil {
		t.Fatalf("loadAreas: %v", err)
	}
	if len(areas) == 0 {
		t.Fatal("no areas found")
	}

	var pe *Area
	for _, a := range areas {
		if a.ID == "platform-engineering" {
			pe = a
			break
		}
	}
	if pe == nil {
		t.Fatal("platform-engineering area not found")
	}

	conv := &OpenCodeConverter{}
	output, err := conv.ConvertArea(pe, nil)
	if err != nil {
		t.Fatalf("ConvertArea: %v", err)
	}

	md := string(output)
	checks := []string{
		"name: \"Platform Engineering\"",
		"◆",
		"mode: primary",
		"hidden: false",
		"## Funciones Autorizadas",
		"## Limitaciones Explícitas",
		"## Respuesta de Hard Stop",
		"HARD STOP",
		"## Referencias Canónicas",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}

	// Verify it does NOT contain lead-only markers
	if strings.Contains(md, "LO QUE SÍ HAGO") {
		t.Error("area output should use 'LO QUE SÍ HACE', not 'LO QUE SÍ HAGO'")
	}
}

func TestOpenCodeConverter_GenerateAll(t *testing.T) {
	canonicalRoot := "../../internal/agents"
	outputRoot := t.TempDir()

	err := GenerateAll(canonicalRoot, TargetOpenCode, outputRoot)
	if err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}

	// Check output directory exists
	outputDir := filepath.Join(outputRoot, "go-runtime/internal/runtimes/opencode/agents")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("output dir not created: %s", outputDir)
	}

	// Verify all 10 area files were generated
	expectedAreas := []string{
		"area-adversarial-intelligence.md",
		"area-commercial-growth.md",
		"area-devops-infrastructure.md",
		"area-digital-product.md",
		"area-education-career.md",
		"area-health-performance.md",
		"area-legal-compliance.md",
		"area-platform-engineering.md",
		"area-research-intelligence.md",
		"area-ux-design.md",
	}
	for _, expected := range expectedAreas {
		areaFile := filepath.Join(outputDir, expected)
		data, err := os.ReadFile(areaFile)
		if err != nil {
			t.Errorf("area file not generated: %s: %v", expected, err)
			continue
		}
		content := string(data)
		if !strings.Contains(content, "hidden: false") {
			t.Errorf("%s: must have hidden: false", expected)
		}
		if !strings.Contains(content, "mode: primary") {
			t.Errorf("%s: must have mode: primary", expected)
		}
	}

	// OpenCode now generates full hierarchy (AreasOnly=false) for 60+ permission blocks (GAP-1 fix).
	leadFiles, _ := filepath.Glob(filepath.Join(outputDir, "lead-*.md"))
	teamFiles, _ := filepath.Glob(filepath.Join(outputDir, "team-*.md"))
	totalFiles := len(expectedAreas) + len(leadFiles) + len(teamFiles)
	if totalFiles < 60 {
		t.Errorf("opencode: expected 60+ files (areas+leads+teams), got %d", totalFiles)
	}
	if len(leadFiles) == 0 {
		t.Errorf("opencode: expected lead files (AreasOnly=false), got 0")
	}
	if len(teamFiles) == 0 {
		t.Errorf("opencode: expected team files (AreasOnly=false), got 0")
	}

	t.Logf("Generated %d area + %d lead + %d team files (AreasOnly=false)",
		len(expectedAreas), len(leadFiles), len(teamFiles))
}

func TestFileExtension(t *testing.T) {
	conv := &OpenCodeConverter{}
	if ext := conv.FileExtension(); ext != ".md" {
		t.Errorf("expected .md, got %q", ext)
	}
}

func TestOutputDir(t *testing.T) {
	conv := &OpenCodeConverter{}
	if dir := conv.OutputDir(); dir != "go-runtime/internal/runtimes/opencode/agents" {
		t.Errorf("expected 'go-runtime/internal/runtimes/opencode/agents', got %q", dir)
	}
}

func TestClaudeCodeConverter_Area(t *testing.T) {
	conv := &ClaudeCodeConverter{}
	if ext := conv.FileExtension(); ext != ".md" {
		t.Errorf("expected .md, got %q", ext)
	}
	if dir := conv.OutputDir(); dir != "go-runtime/internal/runtimes/claude-code/agents" {
		t.Errorf("expected 'go-runtime/internal/runtimes/claude-code/agents', got %q", dir)
	}

	areas, err := loadAreas("../../internal/agents/areas")
	if err != nil || len(areas) == 0 {
		t.Fatal("need areas to test")
	}
	var pe *Area
	for _, a := range areas {
		if a.ID == "platform-engineering" {
			pe = a
			break
		}
	}
	if pe == nil {
		t.Fatal("platform-engineering not found")
	}

	output, err := conv.ConvertArea(pe, nil)
	if err != nil {
		t.Fatalf("ConvertArea: %v", err)
	}
	md := string(output)
	if !strings.Contains(md, "name: \"platform-engineering\"") {
		t.Error("should contain area ID as name")
	}
	if !strings.Contains(md, "description:") {
		t.Error("should contain description")
	}
}

func TestCursorConverter_Area(t *testing.T) {
	conv := &CursorConverter{}
	if ext := conv.FileExtension(); ext != ".md" {
		t.Errorf("expected .md, got %q", ext)
	}
	if dir := conv.OutputDir(); dir != "runtimes/cursor" {
		t.Errorf("expected 'runtimes/cursor', got %q", dir)
	}

	areas, err := loadAreas("../../internal/agents/areas")
	if err != nil || len(areas) == 0 {
		t.Fatal("need areas to test")
	}
	var pe *Area
	for _, a := range areas {
		if a.ID == "platform-engineering" {
			pe = a
			break
		}
	}
	if pe == nil {
		t.Fatal("platform-engineering not found")
	}

	output, err := conv.ConvertArea(pe, nil)
	if err != nil {
		t.Fatalf("ConvertArea: %v", err)
	}
	md := string(output)
	if !strings.Contains(md, "# OVAV Area: Platform Engineering") {
		t.Error("should contain area title")
	}
	if !strings.Contains(md, "alwaysApply: false") {
		t.Error("should contain alwaysApply")
	}
}

func TestAvailableTargets(t *testing.T) {
	targets := AvailableTargets()
	if len(targets) < 4 {
		t.Errorf("expected at least 4 targets, got %d", len(targets))
	}
	names := map[Target]bool{}
	for _, target := range targets {
		names[target] = true
		_, err := GetConverter(target)
		if err != nil {
			t.Errorf("GetConverter(%q): %v", target, err)
		}
	}
	for _, expected := range []Target{TargetMimocode, TargetOpenCode, TargetClaude, TargetCursor} {
		if !names[expected] {
			t.Errorf("missing target: %s", expected)
		}
	}
}

// ── Claude Code: ConvertLead + ConvertTeam ─────────────────────────────────

func TestClaudeCodeConverter_ConvertLead(t *testing.T) {
	leads, err := loadLeads("../../internal/agents/leads")
	if err != nil || len(leads) == 0 {
		t.Fatal("need leads to test")
	}
	var lead *Lead
	for _, l := range leads {
		if l.ID == "thavren" {
			lead = l
			break
		}
	}
	if lead == nil {
		t.Fatal("lead thavren not found")
	}

	conv := &ClaudeCodeConverter{}
	output, err := conv.ConvertLead(lead)
	if err != nil {
		t.Fatalf("ConvertLead: %v", err)
	}
	md := string(output)

	checks := []string{
		"name:",          // Claude format uses name field
		"description:",   // Claude format uses description field
		"# Thavren",      // heading uses lead.Name
		"**Origin:**",    // Origin field is output
		"## Authorized Functions",
		"## Limitations",
		"## Hard Stop",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

func TestClaudeCodeConverter_ConvertTeam(t *testing.T) {
	teams, err := loadTeams("../../internal/agents/teams")
	if err != nil || len(teams) == 0 {
		t.Fatal("need teams to test")
	}
	var team *TeamMember
	for _, tm := range teams {
		if tm.ID == "clara" {
			team = tm
			break
		}
	}
	if team == nil {
		t.Fatal("team clara not found")
	}

	conv := &ClaudeCodeConverter{}
	output, err := conv.ConvertTeam(team)
	if err != nil {
		t.Fatalf("ConvertTeam: %v", err)
	}
	md := string(output)

	checks := []string{
		"name:",           // Claude format uses name field
		"description:",    // Claude format uses description field
		"# Clara",        // heading uses team.Name
		"**Country:**",   // Country field is output
		"**Reports to:**", // Reports to field is output
		"**Area:**",      // Area field is output
		"## Function",
		"## Actions",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

// ── Cursor: ConvertLead + ConvertTeam ──────────────────────────────────────

func TestCursorConverter_ConvertLead(t *testing.T) {
	leads, err := loadLeads("../../internal/agents/leads")
	if err != nil || len(leads) == 0 {
		t.Fatal("need leads to test")
	}
	var lead *Lead
	for _, l := range leads {
		if l.ID == "thavren" {
			lead = l
			break
		}
	}
	if lead == nil {
		t.Fatal("lead thavren not found")
	}

	conv := &CursorConverter{}
	output, err := conv.ConvertLead(lead)
	if err != nil {
		t.Fatalf("ConvertLead: %v", err)
	}
	md := string(output)

	checks := []string{
		`name: "thavren"`,        // Cursor uses name field with ID
		"description:",            // Cursor uses description field
		"readonly:",              // Cursor specific field
		"# Thavren",              // heading uses lead.Name
		"Origin:",                // Origin field is output
		"## Authorized Functions",
		"## Limitations",
		"## Hard Stop",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}

	// Cursor format should NOT contain hidden/mode fields (OpenCode-specific)
	if strings.Contains(md, "hidden:") {
		t.Error("Cursor format should not contain hidden field")
	}
	if strings.Contains(md, "mode:") {
		t.Error("Cursor format should not contain mode field")
	}
}

func TestCursorConverter_ConvertTeam(t *testing.T) {
	teams, err := loadTeams("../../internal/agents/teams")
	if err != nil || len(teams) == 0 {
		t.Fatal("need teams to test")
	}
	var team *TeamMember
	for _, tm := range teams {
		if tm.ID == "clara" {
			team = tm
			break
		}
	}
	if team == nil {
		t.Fatal("team clara not found")
	}

	conv := &CursorConverter{}
	output, err := conv.ConvertTeam(team)
	if err != nil {
		t.Fatalf("ConvertTeam: %v", err)
	}
	md := string(output)

	checks := []string{
		`name: "clara"`,         // Cursor uses name field with ID
		"description:",           // Cursor uses description field
		"readonly:",             // Cursor specific field
		"# Clara",               // heading uses team.Name
		"Country:",              // Country field is output
		"Reports to:",          // Reports to field is output
		"Area:",                // Area field is output
		"## Function",
		"## Actions",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}
}

// ── Error paths: loaders ───────────────────────────────────────────────────

func TestLoadAreas_NonexistentDir(t *testing.T) {
	areas, err := loadAreas("/tmp/ovav-test-nonexistent-areas-dir")
	if err != nil {
		t.Fatalf("non-existent dir should return nil error, got: %v", err)
	}
	if areas != nil {
		t.Errorf("non-existent dir should return nil areas, got %d", len(areas))
	}
}

func TestLoadLeads_NonexistentDir(t *testing.T) {
	leads, err := loadLeads("/tmp/ovav-test-nonexistent-leads-dir")
	if err != nil {
		t.Fatalf("non-existent dir should return nil error, got: %v", err)
	}
	if leads != nil {
		t.Errorf("non-existent dir should return nil leads, got %d", len(leads))
	}
}

func TestLoadTeams_NonexistentDir(t *testing.T) {
	teams, err := loadTeams("/tmp/ovav-test-nonexistent-teams-dir")
	if err != nil {
		t.Fatalf("non-existent dir should return nil error, got: %v", err)
	}
	if teams != nil {
		t.Errorf("non-existent dir should return nil teams, got %d", len(teams))
	}
}

func TestLoadAreas_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	badYAML := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badYAML, []byte("id: bad\n\tbroken: yaml: [["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadAreas(dir)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadLeads_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	badYAML := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badYAML, []byte("id: bad\n\tbroken: yaml: [["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadLeads(dir)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadTeams_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	badYAML := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badYAML, []byte("id: bad\n\tbroken: yaml: [["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadTeams(dir)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadCanonicalAgents_LoadError(t *testing.T) {
	// Create a temp dir with valid areas but invalid YAML in leads
	root := t.TempDir()

	areasDir := filepath.Join(root, "areas")
	if err := os.MkdirAll(areasDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(areasDir, "valid.yaml"),
		[]byte("id: test\nname: Test\nlead: someone\ndescription: desc\ncolor: '#000'\nfunctions:\n  - fn1\nlimitations:\n  - lim1\nhard_stop: stop\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create leads dir with broken YAML to trigger error
	leadsDir := filepath.Join(root, "leads")
	if err := os.MkdirAll(leadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(leadsDir, "broken.yaml"),
		[]byte("id: bad\n\tbroken: yaml: [["), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := LoadCanonicalAgents(root)
	if err == nil {
		t.Error("expected error for invalid YAML in leads, got nil")
	}
}

func TestLoadCanonicalAgents_TeamsError(t *testing.T) {
	// Valid areas and leads, but broken teams
	root := t.TempDir()

	for _, sub := range []string{"areas", "leads", "teams"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}

	validAgent := "id: test\nname: Test\nlead: someone\ndescription: desc\ncolor: '#000'\nfunctions:\n  - fn1\nlimitations:\n  - lim1\nhard_stop: stop\n"
	if err := os.WriteFile(filepath.Join(root, "areas", "valid.yaml"), []byte(validAgent), 0644); err != nil {
		t.Fatal(err)
	}

	validLead := "id: test\nname: Test\ndisplay_name: Test Lead\narea: testarea\norigin: test\nfunctions:\n  - fn1\nlimitations:\n  - lim1\nhard_stop: stop\nsquad:\n  - name: x\n    country: y\n    specialty: z\n"
	if err := os.WriteFile(filepath.Join(root, "leads", "valid.yaml"), []byte(validLead), 0644); err != nil {
		t.Fatal(err)
	}

	// Broken YAML in teams
	if err := os.WriteFile(filepath.Join(root, "teams", "broken.yaml"),
		[]byte("id: bad\n\tbroken: yaml: [["), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := LoadCanonicalAgents(root)
	if err == nil {
		t.Error("expected error for invalid YAML in teams, got nil")
	}
}

// ── GetConverter error path ────────────────────────────────────────────────

func TestGetConverter_UnknownTarget(t *testing.T) {
	_, err := GetConverter(Target("unknown-cli"))
	if err == nil {
		t.Error("expected error for unknown target, got nil")
	}
	if !strings.Contains(err.Error(), "no converter registered") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ── GenerateAll: Claude Code + Cursor + error paths ────────────────────────

func TestGenerateAll_ClaudeCode(t *testing.T) {
	canonicalRoot := "../../internal/agents"
	outputRoot := t.TempDir()

	err := GenerateAll(canonicalRoot, TargetClaude, outputRoot)
	if err != nil {
		t.Fatalf("GenerateAll(Claude): %v", err)
	}

	outputDir := filepath.Join(outputRoot, "go-runtime/internal/runtimes/claude-code/agents")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("output dir not created: %s", outputDir)
	}

	// Verify area files
	areaFiles, _ := filepath.Glob(filepath.Join(outputDir, "area-*.md"))
	if len(areaFiles) < 9 {
		t.Errorf("expected at least 9 area files, got %d", len(areaFiles))
	}

	// Verify lead files
	leadFiles, _ := filepath.Glob(filepath.Join(outputDir, "lead-*.md"))
	if len(leadFiles) < 9 {
		t.Errorf("expected at least 9 lead files, got %d", len(leadFiles))
	}

	// Verify team files
	teamFiles, _ := filepath.Glob(filepath.Join(outputDir, "team-*.md"))
	if len(teamFiles) < 40 {
		t.Errorf("expected at least 40 team files, got %d", len(teamFiles))
	}

	// Spot-check Claude-specific format conventions
	aeFile := filepath.Join(outputDir, "area-platform-engineering.md")
	data, err := os.ReadFile(aeFile)
	if err == nil {
		content := string(data)
		// Claude format uses name, description, color (not type:, hidden:, mode:)
		if !strings.Contains(content, "name:") {
			t.Error("Claude area should contain 'name:' field")
		}
		if !strings.Contains(content, "description:") {
			t.Error("Claude area should contain 'description:' field")
		}
		if strings.Contains(content, "type:") {
			t.Error("Claude format should not contain 'type:' field (OVAV extension)")
		}
	}

	// Check a lead file has name, description, color
	leadFile := filepath.Join(outputDir, "lead-thavren.md")
	data, err = os.ReadFile(leadFile)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, "name:") {
			t.Error("Claude lead should contain 'name:' field")
		}
		if !strings.Contains(content, "description:") {
			t.Error("Claude lead should contain 'description:' field")
		}
	}

	// Check a team file has name, description, model
	teamFile := filepath.Join(outputDir, "team-clara.md")
	data, err = os.ReadFile(teamFile)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, "name:") {
			t.Error("Claude team should contain 'name:' field")
		}
		if !strings.Contains(content, "description:") {
			t.Error("Claude team should contain 'description:' field")
		}
	}

	t.Logf("Claude: %d area + %d lead + %d team files",
		len(areaFiles), len(leadFiles), len(teamFiles))
}

func TestGenerateAll_Cursor(t *testing.T) {
	canonicalRoot := "../../internal/agents"
	outputRoot := t.TempDir()

	err := GenerateAll(canonicalRoot, TargetCursor, outputRoot)
	if err != nil {
		t.Fatalf("GenerateAll(Cursor): %v", err)
	}

	outputDir := filepath.Join(outputRoot, "runtimes/cursor")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("output dir not created: %s", outputDir)
	}

	// Verify area files (.md extension)
	areaFiles, _ := filepath.Glob(filepath.Join(outputDir, "area-*.md"))
	if len(areaFiles) < 9 {
		t.Errorf("expected at least 9 area .md files, got %d", len(areaFiles))
	}

	// Verify lead files
	leadFiles, _ := filepath.Glob(filepath.Join(outputDir, "lead-*.md"))
	if len(leadFiles) < 9 {
		t.Errorf("expected at least 9 lead .md files, got %d", len(leadFiles))
	}

	// Verify team files
	teamFiles, _ := filepath.Glob(filepath.Join(outputDir, "team-*.md"))
	if len(teamFiles) < 40 {
		t.Errorf("expected at least 40 team .md files, got %d", len(teamFiles))
	}

	// Spot-check Cursor-specific .md format
	aeFile := filepath.Join(outputDir, "area-platform-engineering.md")
	data, err := os.ReadFile(aeFile)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, "alwaysApply: false") {
			t.Error("Cursor .mdc should contain 'alwaysApply: false'")
		}
		if !strings.Contains(content, "# OVAV Area:") {
			t.Error("Cursor .mdc should contain '# OVAV Area:' header")
		}
		if strings.Contains(content, "mode:") || strings.Contains(content, "hidden:") {
			t.Error("Cursor .mdc should not contain OpenCode-specific fields")
		}
	}

	// Check team file .mdc specifics
	teamFile := filepath.Join(outputDir, "team-clara.mdc")
	data, err = os.ReadFile(teamFile)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, `description: "Team: Clara`) {
			t.Error("Cursor team should contain Team: description")
		}
		if !strings.Contains(content, "# Team: Clara") {
			t.Error("Cursor team should contain '# Team:' header")
		}
	}

	t.Logf("Cursor: %d area + %d lead + %d team files",
		len(areaFiles), len(leadFiles), len(teamFiles))
}

func TestGenerateAll_Mimocode(t *testing.T) {
	canonicalRoot := "../../internal/agents"
	outputRoot := t.TempDir()

	err := GenerateAll(canonicalRoot, TargetMimocode, outputRoot)
	if err != nil {
		t.Fatalf("GenerateAll(Mimocode): %v", err)
	}

	outputDir := filepath.Join(outputRoot, "runtimes/mimocode/agents")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatalf("output dir not created: %s", outputDir)
	}

	areaFiles, _ := filepath.Glob(filepath.Join(outputDir, "area-*.md"))
	if len(areaFiles) < 9 {
		t.Errorf("expected at least 9 area files, got %d", len(areaFiles))
	}

	// AreasOnly runtime: mimocode's TUI ignores `hidden: true`, so to keep
	// leads/teams out of the TAB picker we must not publish them at all.
	leadFiles, _ := filepath.Glob(filepath.Join(outputDir, "lead-*.md"))
	if len(leadFiles) != 0 {
		t.Errorf("mimocode is AreasOnly: expected 0 lead files, got %d", len(leadFiles))
	}
	teamFiles, _ := filepath.Glob(filepath.Join(outputDir, "team-*.md"))
	if len(teamFiles) != 0 {
		t.Errorf("mimocode is AreasOnly: expected 0 team files, got %d", len(teamFiles))
	}

	// Spot-check Mimocode-specific format (same as OpenCode: mode + hidden)
	aeFile := filepath.Join(outputDir, "area-platform-engineering.md")
	data, err := os.ReadFile(aeFile)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, "mode: primary") {
			t.Error("Mimocode area should contain 'mode: primary'")
		}
		if !strings.Contains(content, "hidden: false") {
			t.Error("Mimocode area should contain 'hidden: false'")
		}
	}

	t.Logf("Mimocode (AreasOnly): %d areas, %d leads, %d teams generated",
		len(areaFiles), len(leadFiles), len(teamFiles))
}

// TestGenerateAll_OpenCodeForceAll verifies that --levels=all override still
// publishes lead/team files for opencode (e.g. for debug/internal builds).
func TestGenerateAll_OpenCodeForceAll(t *testing.T) {
	canonicalRoot := "../../internal/agents"
	outputRoot := t.TempDir()

	err := GenerateAllWithFilter(canonicalRoot, TargetOpenCode, outputRoot, "all")
	if err != nil {
		t.Fatalf("GenerateAllWithFilter(all): %v", err)
	}

	outputDir := filepath.Join(outputRoot, "go-runtime/internal/runtimes/opencode/agents")

	leadFiles, _ := filepath.Glob(filepath.Join(outputDir, "lead-*.md"))
	if len(leadFiles) < 9 {
		t.Errorf("with --levels=all expected ≥9 lead files, got %d", len(leadFiles))
	}
	teamFiles, _ := filepath.Glob(filepath.Join(outputDir, "team-*.md"))
	if len(teamFiles) < 40 {
		t.Errorf("with --levels=all expected ≥40 team files, got %d", len(teamFiles))
	}
}

func TestGenerateAll_UnknownTarget(t *testing.T) {
	err := GenerateAll("../../internal/agents", Target("nonexistent"), t.TempDir())
	if err == nil {
		t.Error("expected error for unknown target, got nil")
	}
}

// ── loaders: ReadDir on a file (not a directory) ──────────────────────────

func TestLoadAreas_ReadDirOnFile(t *testing.T) {
	dir := t.TempDir()
	aFile := filepath.Join(dir, "notadir")
	if err := os.WriteFile(aFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	// Pass a file path as if it were a directory — ReadDir should fail
	_, err := loadAreas(aFile)
	if err == nil {
		t.Error("expected error when ReadDir on a file path, got nil")
	}
}

func TestLoadLeads_ReadDirOnFile(t *testing.T) {
	dir := t.TempDir()
	aFile := filepath.Join(dir, "notadir")
	if err := os.WriteFile(aFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadLeads(aFile)
	if err == nil {
		t.Error("expected error when ReadDir on a file path, got nil")
	}
}

func TestLoadTeams_ReadDirOnFile(t *testing.T) {
	dir := t.TempDir()
	aFile := filepath.Join(dir, "notadir")
	if err := os.WriteFile(aFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadTeams(aFile)
	if err == nil {
		t.Error("expected error when ReadDir on a file path, got nil")
	}
}

// ── Edge case: loaders skip non-YAML and directories ───────────────────────

// ── loaders: unreadable file triggers ReadFile error ───────────────────────

func TestLoadAreas_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "secret.yaml")
	if err := os.WriteFile(badFile, []byte("id: x\n"), 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(badFile, 0644) // cleanup so TempDir can be removed
	_, err := loadAreas(dir)
	if err == nil {
		t.Error("expected error for unreadable YAML file, got nil")
	}
}

// ── Edge case: loaders skip non-YAML and directories ───────────────────────

func TestLoadAreas_SkipsNonYAML(t *testing.T) {
	dir := t.TempDir()

	// Create a valid YAML file
	validYAML := filepath.Join(dir, "valid.yaml")
	if err := os.WriteFile(validYAML, []byte("id: test\nname: Test\nlead: someone\ndescription: desc\ncolor: '#000'\nfunctions:\n  - fn1\nlimitations:\n  - lim1\nhard_stop: stop\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-YAML file that should be skipped
	nonYAML := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(nonYAML, []byte("not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory that should be skipped
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	areas, err := loadAreas(dir)
	if err != nil {
		t.Fatalf("loadAreas: %v", err)
	}
	if len(areas) != 1 {
		t.Errorf("expected 1 area, got %d", len(areas))
	}
	if areas[0].ID != "test" {
		t.Errorf("expected area id 'test', got %q", areas[0].ID)
	}
}

// ── GAP-3: CRITERIA injection ─────────────────────────────────────────────────

func TestLoadCanonicalAgents_CriteriaLoaded(t *testing.T) {
	// LoadCanonicalAgents should populate lead.Criteria from .ovav/service_areas/<area>/<lead>/CRITERIA.yaml
	// This test verifies that for at least one lead, CRITERIA is non-empty.
	// The CRITERIA path resolution is: repoRoot/.ovav/service_areas/<area>/<lead>/CRITERIA.yaml
	areas, leads, _, err := LoadCanonicalAgents("../../internal/agents")
	if err != nil {
		t.Fatalf("LoadCanonicalAgents: %v", err)
	}
	if len(areas) == 0 {
		t.Fatal("expected at least one area")
	}
	if len(leads) == 0 {
		t.Fatal("expected at least one lead")
	}

	// Count leads with CRITERIA loaded
	withCriteria := 0
	withoutCriteria := []string{}
	for _, lead := range leads {
		if lead.Criteria != "" {
			withCriteria++
		} else {
			withoutCriteria = append(withoutCriteria, lead.ID)
		}
	}

	// At least 8 leads should have CRITERIA (thavren, elena, eidren, sofia, uriel, renata, camila, valeria, kenji, dante)
	if withCriteria < 8 {
		t.Errorf("expected at least 8 leads with CRITERIA loaded, got %d; missing: %v", withCriteria, withoutCriteria)
	}
}

func TestOpenCodeConverter_Lead_CriteriaInjected(t *testing.T) {
	// Verify that OpenCodeConverter.ConvertLead injects lead.Criteria into output.
	conv := &OpenCodeConverter{}
	lead := &Lead{
		AgentBase:   AgentBase{ID: "test-lead", Name: "Test Lead", Description: "test"},
		DisplayName: "Test Lead",
		Origin:      "test origin",
		Authority:   "test authority",
		Functions:   []string{"function 1"},
		Limitations: []string{"limitation 1"},
		HardStop:    "HARD STOP",
		Squad:       []SquadMember{},
		Criteria:    "## Test Criteria\n\nThis is a test criteria section.",
	}

	output, err := conv.ConvertLead(lead)
	if err != nil {
		t.Fatalf("ConvertLead: %v", err)
	}

	if !strings.Contains(string(output), "## Decision Criteria") {
		t.Error("ConvertLead output should contain '## Decision Criteria' section")
	}
	if !strings.Contains(string(output), "This is a test criteria section") {
		t.Error("ConvertLead output should contain the CRITERIA content")
	}
}

func TestMimocodeConverter_Lead_CriteriaInjected(t *testing.T) {
	// Verify that MimocodeConverter.ConvertLead injects lead.Criteria into output.
	// GAP-3 fix: MimocodeConverter was missing CRITERIA injection.
	conv := &MimocodeConverter{}
	lead := &Lead{
		AgentBase:   AgentBase{ID: "test-lead", Name: "Test Lead", Description: "test"},
		DisplayName: "Test Lead",
		Origin:      "test origin",
		Authority:   "test authority",
		Functions:   []string{"function 1"},
		Limitations: []string{"limitation 1"},
		HardStop:    "HARD STOP",
		Squad:       []SquadMember{},
		Criteria:    "## Test Criteria\n\nThis is a test criteria section for MimocodeConverter.",
	}

	output, err := conv.ConvertLead(lead)
	if err != nil {
		t.Fatalf("ConvertLead: %v", err)
	}

	if !strings.Contains(string(output), "## Decision Criteria") {
		t.Error("MimocodeConverter.ConvertLead output should contain '## Decision Criteria' section")
	}
	if !strings.Contains(string(output), "This is a test criteria section for MimocodeConverter") {
		t.Error("MimocodeConverter.ConvertLead output should contain the CRITERIA content")
	}
}
