package ows

import (
	"os"
	"path/filepath"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// stack_test.go — Tests for stack.go (stack detection engine)
// All functions in stack.go are at 0% coverage — these tests cover them.
// ═══════════════════════════════════════════════════════════════════════════

// ── ValidatorsFor ────────────────────────────────────────────────────────

func TestValidatorsFor_Go(t *testing.T) {
	v := ValidatorsFor(StackGo)
	if len(v) == 0 {
		t.Error("expected validators for Go")
	}
	found := map[string]bool{}
	for _, s := range v {
		found[s] = true
	}
	for _, want := range []string{"go_vet", "go_test", "gofmt"} {
		if !found[want] {
			t.Errorf("missing validator %q for Go", want)
		}
	}
}

func TestValidatorsFor_TypeScript(t *testing.T) {
	for _, st := range []StackType{StackTSReact, StackTSNode, StackTSVue} {
		v := ValidatorsFor(st)
		if len(v) == 0 {
			t.Errorf("expected validators for %s", st)
		}
	}
}

func TestValidatorsFor_Python(t *testing.T) {
	v := ValidatorsFor(StackPython)
	if len(v) == 0 {
		t.Error("expected validators for Python")
	}
}

func TestValidatorsFor_Rust(t *testing.T) {
	v := ValidatorsFor(StackRust)
	if len(v) == 0 {
		t.Error("expected validators for Rust")
	}
}

func TestValidatorsFor_Monorepo(t *testing.T) {
	v := ValidatorsFor(StackMonorepo)
	if len(v) == 0 {
		t.Error("expected validators for Monorepo")
	}
}

func TestValidatorsFor_Unknown(t *testing.T) {
	v := ValidatorsFor(StackUnknown)
	if v != nil {
		t.Errorf("expected nil for unknown, got %v", v)
	}
}

// ── DetectStacks ─────────────────────────────────────────────────────────

func TestDetectStacks_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	info := DetectStacks(dir)
	if info == nil {
		t.Fatal("DetectStacks returned nil")
	}
	if len(info.Stacks) != 1 {
		t.Fatalf("expected 1 stack (unknown), got %d", len(info.Stacks))
	}
	if info.Stacks[0].Type != StackUnknown {
		t.Errorf("expected StackUnknown, got %s", info.Stacks[0].Type)
	}
}

func TestDetectStacks_GoModule(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0644)

	info := DetectStacks(dir)
	if !info.HasGo() {
		t.Error("should detect Go stack")
	}
	dirs := info.GoDirs()
	if len(dirs) == 0 {
		t.Error("GoDirs should not be empty")
	}
}

func TestDetectStacks_PythonPyproject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackPython {
			found = true
		}
	}
	if !found {
		t.Error("should detect Python stack from pyproject.toml")
	}
}

func TestDetectStacks_PythonRequirements(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackPython {
			found = true
		}
	}
	if !found {
		t.Error("should detect Python stack from requirements.txt")
	}
}

func TestDetectStacks_RustCargo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackRust {
			found = true
		}
	}
	if !found {
		t.Error("should detect Rust stack from Cargo.toml")
	}
}

func TestDetectStacks_JavaScriptReact(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies": {"react": "^18.0.0"}}`), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackTSReact {
			found = true
		}
	}
	if !found {
		t.Error("should detect React stack from package.json")
	}
}

func TestDetectStacks_JavaScriptVue(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies": {"vue": "^3.0.0"}}`), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackTSVue {
			found = true
		}
	}
	if !found {
		t.Error("should detect Vue stack from package.json")
	}
}

func TestDetectStacks_JavaScriptNode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies": {"express": "^4.0.0"}}`), 0644)

	info := DetectStacks(dir)
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackTSNode {
			found = true
		}
	}
	if !found {
		t.Error("should detect Node stack from package.json")
	}
}

func TestDetectStacks_Monorepo_pnpm(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pnpm-workspace.yaml"), []byte("packages:\n  - 'packages/*'\n"), 0644)

	info := DetectStacks(dir)
	if !info.IsMonorepo {
		t.Error("should detect monorepo from pnpm-workspace.yaml")
	}
}

func TestDetectStacks_Monorepo_lerna(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "lerna.json"), []byte(`{"packages": ["packages/*"]}`), 0644)

	info := DetectStacks(dir)
	if !info.IsMonorepo {
		t.Error("should detect monorepo from lerna.json")
	}
}

func TestDetectStacks_Monorepo_turbo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "turbo.json"), []byte(`{"packages": ["packages/*"]}`), 0644)

	info := DetectStacks(dir)
	if !info.IsMonorepo {
		t.Error("should detect monorepo from turbo.json")
	}
}

func TestDetectStacks_Monorepo_nx(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "nx.json"), []byte(`{"workspaceLayout": {}}`), 0644)

	info := DetectStacks(dir)
	if !info.IsMonorepo {
		t.Error("should detect monorepo from nx.json")
	}
}

func TestDetectStacks_NoPackageJson(t *testing.T) {
	dir := t.TempDir()
	// No package.json → should not crash
	info := DetectStacks(dir)
	if info == nil {
		t.Fatal("DetectStacks returned nil")
	}
}

// ── StackInfo Methods ────────────────────────────────────────────────────

func TestHasGo_NoGo(t *testing.T) {
	dir := t.TempDir()
	info := DetectStacks(dir)
	if info.HasGo() {
		t.Error("empty dir should not have Go")
	}
}

func TestHasGo_WithGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644)
	info := DetectStacks(dir)
	if !info.HasGo() {
		t.Error("should have Go")
	}
}

func TestGoDirs_NoGo(t *testing.T) {
	dir := t.TempDir()
	info := DetectStacks(dir)
	dirs := info.GoDirs()
	if len(dirs) != 0 {
		t.Errorf("expected 0 Go dirs, got %d", len(dirs))
	}
}

func TestGoDirs_WithGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644)
	info := DetectStacks(dir)
	dirs := info.GoDirs()
	if len(dirs) != 1 {
		t.Errorf("expected 1 Go dir, got %d", len(dirs))
	}
	if dirs[0] != "." {
		t.Errorf("Go dir = %q, want '.'", dirs[0])
	}
}

func TestPrimaryStack_Empty(t *testing.T) {
	info := &StackInfo{}
	ps := info.PrimaryStack()
	if ps.Type != StackUnknown {
		t.Errorf("expected StackUnknown for empty, got %s", ps.Type)
	}
	if ps.Dir != "." {
		t.Errorf("expected '.' dir, got %q", ps.Dir)
	}
}

func TestPrimaryStack_WithStacks(t *testing.T) {
	info := &StackInfo{
		Stacks: []DetectedStack{
			{Type: StackGo, Dir: "go-runtime"},
			{Type: StackPython, Dir: "."},
		},
	}
	ps := info.PrimaryStack()
	if ps.Type != StackGo {
		t.Errorf("expected Go as primary, got %s", ps.Type)
	}
	if ps.Dir != "go-runtime" {
		t.Errorf("expected 'go-runtime', got %q", ps.Dir)
	}
}

func TestSummary_Empty(t *testing.T) {
	info := &StackInfo{}
	s := info.Summary()
	if s != "no stacks detected" {
		t.Errorf("expected 'no stacks detected', got %q", s)
	}
}

func TestSummary_SingleStack(t *testing.T) {
	info := &StackInfo{
		Stacks: []DetectedStack{{Type: StackGo, Dir: "."}},
	}
	s := info.Summary()
	if s != "go" {
		t.Errorf("expected 'go', got %q", s)
	}
}

func TestSummary_MultipleStacks(t *testing.T) {
	info := &StackInfo{
		Stacks: []DetectedStack{
			{Type: StackGo, Dir: "."},
			{Type: StackPython, Dir: "scripts"},
		},
	}
	s := info.Summary()
	if s != "go, python @scripts" {
		t.Errorf("expected 'go, python @scripts', got %q", s)
	}
}

func TestSummary_Monorepo(t *testing.T) {
	info := &StackInfo{
		Stacks:     []DetectedStack{{Type: StackGo, Dir: "."}},
		IsMonorepo: true,
	}
	s := info.Summary()
	if s != "go (monorepo)" {
		t.Errorf("expected 'go (monorepo)', got %q", s)
	}
}

// ── DetectProfileFromBranch ──────────────────────────────────────────────

func TestDetectProfileFromBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"feature/task27", "feature"},
		{"hotfix/critical-bug", "hotfix"},
		{"hotfix-critical", "hotfix"},
		{"release/v1.0", "release"},
		{"release-2.0", "release"},
		{"emergency-patch", "emergency"},
		{"enterprise/auth", "enterprise"},
		{"enterprise-auth", "enterprise"},
		{"spike/perf", "spike"},
		{"spike-perf", "spike"},
		{"research/ml", "research"},
		{"research-ml", "research"},
		{"docs/readme", "docs"},
		{"docs-readme", "docs"},
		{"random-branch", "feature"},
		{"", "feature"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := DetectProfileFromBranch(tt.branch)
			if got != tt.want {
				t.Errorf("DetectProfileFromBranch(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}

// ── LevelForProfile ──────────────────────────────────────────────────────

func TestLevelForProfile(t *testing.T) {
	tests := []struct {
		profile string
		want    VerificationLevel
	}{
		{"hotfix", VerifyStandard},
		{"release", VerifyStrict},
		{"enterprise", VerifyMaximum},
		{"emergency", VerifyStrict},
		{"spike", VerifyBasic},
		{"research", VerifyBasic},
		{"docs", VerifyBasic},
		{"feature", VerifyStandard},
		{"unknown", VerifyStandard},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			got := LevelForProfile(tt.profile)
			if got != tt.want {
				t.Errorf("LevelForProfile(%q) = %d, want %d", tt.profile, got, tt.want)
			}
		})
	}
}

// ── VerificationLevel Constants ──────────────────────────────────────────

func TestVerificationLevel_Constants(t *testing.T) {
	if VerifyBasic != 0 {
		t.Errorf("VerifyBasic = %d, want 0", VerifyBasic)
	}
	if VerifyStandard != 1 {
		t.Errorf("VerifyStandard = %d, want 1", VerifyStandard)
	}
	if VerifyStrict != 2 {
		t.Errorf("VerifyStrict = %d, want 2", VerifyStrict)
	}
	if VerifyMaximum != 3 {
		t.Errorf("VerifyMaximum = %d, want 3", VerifyMaximum)
	}
}

// ── StackType Constants ──────────────────────────────────────────────────

func TestStackType_Constants(t *testing.T) {
	types := map[StackType]string{
		StackGo:       "go",
		StackTSReact:  "ts:react",
		StackTSNode:   "ts:node",
		StackTSVue:    "ts:vue",
		StackPython:   "python",
		StackRust:     "rust",
		StackMonorepo: "monorepo",
		StackUnknown:  "unknown",
	}
	for st, expected := range types {
		if string(st) != expected {
			t.Errorf("StackType %v = %q, want %q", st, string(st), expected)
		}
	}
}

// ── parseGoModModule ─────────────────────────────────────────────────────

func TestParseGoModModule_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	os.WriteFile(path, []byte("module github.com/test/repo\n\ngo 1.21\n"), 0644)

	got := parseGoModModule(path)
	if got != "github.com/test/repo" {
		t.Errorf("parseGoModModule = %q, want github.com/test/repo", got)
	}
}

func TestParseGoModModule_NoModule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	os.WriteFile(path, []byte("go 1.21\n"), 0644)

	got := parseGoModModule(path)
	if got != "" {
		t.Errorf("parseGoModModule should return empty for no module line, got %q", got)
	}
}

func TestParseGoModModule_NonexistentFile(t *testing.T) {
	got := parseGoModModule("/nonexistent/go.mod")
	if got != "" {
		t.Errorf("parseGoModModule should return empty for nonexistent file, got %q", got)
	}
}

func TestParseGoModModule_ModuleWithPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	os.WriteFile(path, []byte("module  my-module  \n"), 0644)

	got := parseGoModModule(path)
	if got != "my-module" {
		t.Errorf("parseGoModModule = %q, want my-module", got)
	}
}

// ── DetectStacks with nested Go modules ──────────────────────────────────

func TestDetectStacks_NestedGoModule(t *testing.T) {
	dir := t.TempDir()
	// Create a nested go.mod
	subDir := filepath.Join(dir, "packages", "app")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module nested\n\ngo 1.21\n"), 0644)

	info := DetectStacks(dir)
	goDirs := info.GoDirs()
	// Should find the nested go.mod
	found := false
	for _, d := range goDirs {
		if d == "packages/app" {
			found = true
		}
	}
	if !found {
		t.Errorf("should find nested go.mod in packages/app, got dirs: %v", goDirs)
	}
}

func TestDetectStacks_GoModWithJS(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies": {"react": "^18"}}`), 0644)

	info := DetectStacks(dir)
	if !info.HasGo() {
		t.Error("should detect Go")
	}
	found := false
	for _, s := range info.Stacks {
		if s.Type == StackTSReact {
			found = true
		}
	}
	if !found {
		t.Error("should also detect React")
	}
}

// ── DetectedStack Fields ─────────────────────────────────────────────────

func TestDetectedStack_Fields(t *testing.T) {
	s := DetectedStack{Type: StackGo, Dir: "go-runtime"}
	if s.Type != StackGo {
		t.Errorf("Type = %q", s.Type)
	}
	if s.Dir != "go-runtime" {
		t.Errorf("Dir = %q", s.Dir)
	}
}

func TestStackInfo_Fields(t *testing.T) {
	info := &StackInfo{
		Stacks:     []DetectedStack{{Type: StackGo, Dir: "."}},
		IsMonorepo: true,
		Root:       "/tmp",
	}
	if len(info.Stacks) != 1 {
		t.Errorf("Stacks len = %d", len(info.Stacks))
	}
	if !info.IsMonorepo {
		t.Error("IsMonorepo should be true")
	}
	if info.Root != "/tmp" {
		t.Errorf("Root = %q", info.Root)
	}
}

// ── DetectStacks with .ovav/worktrees dir in walk ───────────────────────

func TestDetectStacks_SkipsDotDirs(t *testing.T) {
	dir := t.TempDir()
	// Create a .hidden dir with go.mod — should be skipped
	hiddenDir := filepath.Join(dir, ".hidden")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "go.mod"), []byte("module hidden\n"), 0644)

	info := DetectStacks(dir)
	goDirs := info.GoDirs()
	for _, d := range goDirs {
		if d == ".hidden" {
			t.Error("should skip .hidden directories")
		}
	}
}

func TestDetectStacks_SkipsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, "go.mod"), []byte("module pkg\n"), 0644)

	info := DetectStacks(dir)
	goDirs := info.GoDirs()
	for _, d := range goDirs {
		if d == "node_modules/pkg" {
			t.Error("should skip node_modules directories")
		}
	}
}

func TestDetectStacks_SkipsVendor(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor", "lib")
	os.MkdirAll(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, "go.mod"), []byte("module lib\n"), 0644)

	info := DetectStacks(dir)
	goDirs := info.GoDirs()
	for _, d := range goDirs {
		if d == "vendor/lib" {
			t.Error("should skip vendor directories")
		}
	}
}

func TestDetectStacks_DeepNestedSkip(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod at depth 4 — should be skipped (count >= 3)
	deepDir := filepath.Join(dir, "a", "b", "c", "d")
	os.MkdirAll(deepDir, 0755)
	os.WriteFile(filepath.Join(deepDir, "go.mod"), []byte("module deep\n"), 0644)

	info := DetectStacks(dir)
	goDirs := info.GoDirs()
	for _, d := range goDirs {
		if d == "a/b/c/d" {
			t.Error("should skip directories deeper than 3 levels")
		}
	}
}
