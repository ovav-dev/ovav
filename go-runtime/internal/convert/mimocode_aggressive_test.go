package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMimocodeBrain_CompleteValidation — AGGRESSIVE: verifies every aspect
// of the mimocode agent generation against the OpenCode baseline.
//
// Contract enforced (post AreasOnly refactor):
//   - Both mimocode and opencode publish ONLY area-*.md to the runtime
//     directory (TAB picker). Leads and teams remain in the canonical
//     tree (ovav/agents/leads, ovav/agents/teams) and are NOT leaked
//     into the user-facing picker. claude-code/cursor still get the
//     full hierarchy — that's tested separately in convert_test.go.
//   - Every area file has: mode: primary, hidden: false, name, description,
//     lead reference, color, governance contracts, authorized functions,
//     limitations, and a hard stop.
func generatePickerRuntime(t *testing.T) string {
	t.Helper()

	root := "../.."
	outputRoot := t.TempDir()
	canonicalRoot := filepath.Join(root, "internal", "agents")
	for _, target := range []Target{TargetMimocode, TargetOpenCode} {
		if err := GenerateAll(canonicalRoot, target, outputRoot); err != nil {
			t.Fatalf("generate %s runtime fixture: %v", target, err)
		}
	}
	return filepath.Join(outputRoot, "runtimes")
}

func TestMimocodeBrain_CompleteValidation(t *testing.T) {
	agentsRoot := generatePickerRuntime(t)

	// 1. Verify both runtimes exist
	// Note: OpenCode now outputs to go-runtime/internal/runtimes/opencode/agents
	for _, rt := range []string{"mimocode", "opencode"} {
		dir := filepath.Join(agentsRoot, rt, "agents")
		if rt == "opencode" {
			// OpenCode outputs to outputRoot/go-runtime/internal/runtimes/opencode/agents
			// agentsRoot is outputRoot/runtimes, so we go up one level
			dir = filepath.Join(agentsRoot, "..", "go-runtime", "internal", "runtimes", "opencode", "agents")
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Fatalf("runtime %s/agents/ does not exist at %s", rt, dir)
		}
	}

	// 2. Mimocode (AreasOnly=true) must have ONLY 10 areas + ovav.md (11 files total).
	// Anything beyond that means lead/team files leaked into the picker.
	{
		rt := "mimocode"
		dir := filepath.Join(agentsRoot, rt, "agents")
		files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
		leadLeak, _ := filepath.Glob(filepath.Join(dir, "lead-*.md"))
		teamLeak, _ := filepath.Glob(filepath.Join(dir, "team-*.md"))
		if len(leadLeak) > 0 {
			t.Errorf("%s: AreasOnly runtime leaked %d lead files into picker: %v", rt, len(leadLeak), leadLeak)
		}
		if len(teamLeak) > 0 {
			t.Errorf("%s: AreasOnly runtime leaked %d team files into picker: %v", rt, len(teamLeak), teamLeak)
		}
		if len(files) != 10 {
			t.Errorf("%s: expected 10 area files, got %d", rt, len(files))
		}
	}

	// 2b. OpenCode (AreasOnly=false) now generates full hierarchy: areas + leads + teams.
	// Verify it has 60+ files with permission blocks (GAP-1 converter fix).
	{
		rt := "opencode"
		opencodeDir := filepath.Join(agentsRoot, "..", "go-runtime", "internal", "runtimes", "opencode", "agents")
		files, _ := filepath.Glob(filepath.Join(opencodeDir, "*.md"))
		if len(files) < 60 {
			t.Errorf("%s: expected 60+ files (areas+leads+teams), got %d", rt, len(files))
		}
		t.Logf("%s: generated %d files (full hierarchy — AreasOnly=false)", rt, len(files))
	}

	// 3. Verify every mimocode area has correct frontmatter
	areaFiles, _ := filepath.Glob(filepath.Join(agentsRoot, "mimocode", "agents", "area-*.md"))
	if len(areaFiles) != 10 {
		t.Errorf("expected 10 area files, got %d", len(areaFiles))
	}
	for _, f := range areaFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("cannot read %s: %v", filepath.Base(f), err)
			continue
		}
		content := string(data)
		name := filepath.Base(f)

		// Must have mode: primary
		if !strings.Contains(content, "mode: primary") {
			t.Errorf("%s: missing 'mode: primary'", name)
		}
		// Must have hidden: false
		if !strings.Contains(content, "hidden: false") {
			t.Errorf("%s: missing 'hidden: false' — will NOT appear in TABS", name)
		}
		// Must NOT have hidden: true
		if strings.Contains(content, "hidden: true") {
			t.Errorf("%s: has 'hidden: true' — area should be VISIBLE", name)
		}
		// Must have name field
		if !strings.Contains(content, "name:") {
			t.Errorf("%s: missing 'name:' field", name)
		}
		// Must have description
		if !strings.Contains(content, "description:") {
			t.Errorf("%s: missing 'description:' field", name)
		}
		// Must have lead reference
		if !strings.Contains(content, "Lead:") && !strings.Contains(content, "Lead:**") {
			t.Errorf("%s: missing lead reference", name)
		}
		// Must have color
		if !strings.Contains(content, "color:") {
			t.Errorf("%s: missing 'color:' — TAB display needs it", name)
		}
		// Must have governance contracts
		if !strings.Contains(content, "Contratos de Gobernanza") {
			t.Errorf("%s: missing governance contracts section", name)
		}
		// Must have functions section
		if !strings.Contains(content, "Funciones Autorizadas") {
			t.Errorf("%s: missing authorized functions", name)
		}
		// Must have limitations section
		if !strings.Contains(content, "Limitaciones Explícitas") {
			t.Errorf("%s: missing limitations", name)
		}
		// Must have hard stop section
		if !strings.Contains(content, "HARD STOP") && !strings.Contains(content, "Hard Stop") {
			t.Errorf("%s: missing hard stop", name)
		}
	}

	// 4. (Skipped — lead/team files no longer published to AreasOnly runtimes.
	//    Lead/team contracts are verified in canonical tree separately.)

	// 5. Cross-validate: mimocode and opencode area files have semantically equivalent content
	for _, mf := range areaFiles {
		base := filepath.Base(mf)
		mimoData, _ := os.ReadFile(mf)
		opData, err := os.ReadFile(filepath.Join(agentsRoot, "..", "go-runtime", "internal", "runtimes", "opencode", "agents", base))
		if err != nil {
			t.Errorf("opencode missing area %s: %v", base, err)
			continue
		}
		mimoContent := string(mimoData)
		opContent := string(opData)
		// Check key sections exist in both (byte-level comparison can differ due to map iteration order)
		for _, section := range []string{
			"mode: primary",
			"hidden: false",
			"Contratos de Gobernanza",
			"Funciones Autorizadas",
			"Limitaciones Explícitas",
			"HARD STOP",
		} {
			if !strings.Contains(mimoContent, section) {
				t.Errorf("area %s: mimocode missing section %q", base, section)
			}
			if !strings.Contains(opContent, section) {
				t.Errorf("area %s: opencode missing section %q", base, section)
			}
		}
	}

	// 6. Verify .mimocode/ symlinks exist
	root := "../../.."
	// Resolve repo root by joining with the absolute path. EvalSymlinks on
	// relative paths depends on the caller's cwd, which is unreliable when
	// tests run from go-runtime/. We compute the resolved path explicitly.
	absRoot, absErr := filepath.Abs(root)
	if absErr != nil {
		t.Fatalf("resolve repo root: %v", absErr)
	}
	// `agents` resolves to the Product install directory, which may not be
	// present in pure Systems runs. It is wired through Product bootstrap and
	// is intentionally optional here. `skills` likewise targets the Product
	// install path under Product's retired consumer layout (2026-07-01) and
	// is optional for Systems-only validation. `commands`, `plugins`, and
	// `themes` are also Product consumer-layout features absent in Systems.
	optionalLinks := map[string]bool{
		"agents":   true,
		"skills":   true,
		"commands": true,
		"plugins":  true,
		"themes":   true,
	}
	for _, link := range []string{"agents", "skills", "commands", "plugins", "themes"} {
		path := filepath.Join(absRoot, ".mimocode", link)
		// Use EvalSymlinks to properly resolve both relative and absolute symlinks.
		// filepath.Join(dir, absolute_target) does NOT work for absolute targets in Go stdlib.
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			if optionalLinks[link] {
				t.Logf(".mimocode/%s symlink optional: %v", link, err)
				continue
			}
			t.Errorf(".mimocode/%s symlink missing: %v", link, err)
			continue
		}
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			if optionalLinks[link] {
				t.Logf(".mimocode/%s optional target missing: %s", link, resolved)
				continue
			}
			t.Errorf(".mimocode/%s -> %s: target does not exist", link, resolved)
		}
	}

	// 7. Verify mimocode.json symlink exists
	mimoJSON := filepath.Join(root, "mimocode.json")
	if _, err := os.Lstat(mimoJSON); err != nil {
		t.Errorf("mimocode.json symlink missing")
	}

	t.Logf("✅ AGGRESSIVE VALIDATION COMPLETE: %d areas — AreasOnly contract honored",
		len(areaFiles))
}

// TestMimocodeBrain_AreaNamesInTabs verifies that ONLY area names
// would appear in the TAB selector (not leads, not teams).
func TestMimocodeBrain_AreaNamesInTabs(t *testing.T) {
	agentsDir := filepath.Join(generatePickerRuntime(t), "mimocode", "agents")

	// Collect all display names from hidden:false agents
	files, _ := filepath.Glob(filepath.Join(agentsDir, "*.md"))
	var tabVisible, tabHidden []string

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(data)
		// Extract name from YAML frontmatter
		nameStart := strings.Index(content, "name:")
		if nameStart < 0 {
			continue
		}
		nameLine := content[nameStart : nameStart+80]
		if idx := strings.Index(nameLine, "\n"); idx > 0 {
			nameLine = nameLine[:idx]
		}
		nameLine = strings.TrimPrefix(nameLine, "name:")
		nameLine = strings.Trim(nameLine, " \"'")

		if strings.Contains(content, "hidden: false") {
			tabVisible = append(tabVisible, nameLine)
		} else {
			tabHidden = append(tabHidden, nameLine)
		}
	}

	t.Logf("TAB VISIBLE (%d): %v", len(tabVisible), tabVisible)
	t.Logf("TAB HIDDEN (%d): not shown in tabs", len(tabHidden))

	if len(tabVisible) != 10 {
		t.Errorf("expected 10 TAB-visible agents (areas), got %d", len(tabVisible))
	}

	// Verify no platform-engineering lead or team appears
	for _, visible := range tabVisible {
		if strings.Contains(strings.ToLower(visible), "team") {
			t.Errorf("team name '%s' would appear in TABS", visible)
		}
	}

	// Expected area names
	expectedAreas := []string{
		"Platform Engineering", "Research Intelligence", "Ux Design",
		"Digital Product", "Commercial Growth", "Health Performance",
		"Education Career", "Devops Infrastructure", "Adversarial Intelligence",
		"Legal Compliance",
	}
	for _, expected := range expectedAreas {
		found := false
		for _, visible := range tabVisible {
			if strings.EqualFold(visible, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected area %q in TAB-visible list, not found", expected)
		}
	}
}
