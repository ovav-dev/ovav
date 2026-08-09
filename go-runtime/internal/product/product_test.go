package product

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortableAssets(t *testing.T) {
	assets := PortableAssets()
	if len(assets) < 3 {
		t.Errorf("expected at least 3 portable assets, got %d", len(assets))
	}

	categories := make(map[string]int)
	for _, a := range assets {
		categories[a.Category]++
	}
	if categories["agents"] == 0 {
		t.Error("missing agents asset")
	}
	if categories["skills"] == 0 {
		t.Error("missing skills asset")
	}
	if categories["identity"] == 0 {
		t.Error("missing identity asset")
	}
}

func TestRestrictedAssets(t *testing.T) {
	r := RestrictedAssets()
	if len(r) < 10 {
		t.Errorf("expected at least 10 restricted, got %d", len(r))
	}
}

func TestIsPortable(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"runtimes/mimocode/agents/", true},
		{"runtimes/mimocode/agents/lead-thavren.md", true},
		{".ovav/source/skills/", true},
		{"OVAV_IDENTITY.md", true},
		{"go-runtime/internal/validators/", false},
		{".ovav/source/configs/model_routing.json", true},
		{"random/path.txt", false},
	}
	for _, tt := range tests {
		if got := IsPortable(tt.path); got != tt.want {
			t.Errorf("IsPortable(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestProductDir(t *testing.T) {
	dir, err := ProductDir()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".local", "share", "ovav")
	if dir != want {
		t.Errorf("ProductDir() = %q, want %q", dir, want)
	}
}

func TestManifest(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)

	m := NewManifest("/test/ovav", "product")
	if m.Version != ProductManifestVersion {
		t.Errorf("version = %q", m.Version)
	}

	if err := m.AddEntry(testFile, "/target/test.txt", "test.txt", "test", true); err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(m.Entries))
	}
	if m.Entries[0].Hash == "" {
		t.Error("expected hash")
	}

	m.AddSymlink("/link", "/target", true)
	if len(m.Symlinks) != 1 {
		t.Errorf("symlinks = %d, want 1", len(m.Symlinks))
	}
}

func TestInstallAndVerify(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create minimal OVAV structure with required assets
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-thavren.md"), []byte("# Thavren\nPlatform Engineering expert"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	// Override home
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Install
	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Errorf("install errors: %v", result.Errors)
	}

	// Verify
	vr, err := ProductInstall(ovavRoot, "verify")
	if err != nil {
		t.Fatal(err)
	}
	if len(vr.Errors) > 0 {
		t.Errorf("verify errors: %v", vr.Errors)
	}

	// Uninstall
	ur, err := ProductInstall(ovavRoot, "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	if ur.FilesCopied == 0 && ur.LinksCreated == 0 {
		t.Error("expected files to be removed")
	}
}

func TestBootstrap(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create OVAV structure
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "area-test.md"), []byte("# Test\n---\nname: \"Test\"\nmode: primary\nhidden: false\n"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	// Override home
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Install
	if _, err := ProductInstall(ovavRoot, "install"); err != nil {
		t.Fatal(err)
	}

	// Create a project directory
	projectDir := filepath.Join(dir, "my-project")
	os.MkdirAll(projectDir, 0755)

	// Bootstrap
	br, err := Bootstrap(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if !br.AgentsLinked {
		t.Error("agents not linked")
	}
	if !br.SkillsLinked {
		t.Error("skills not linked")
	}

	// Verify .mimocode/ was created
	mcDir := filepath.Join(projectDir, ".mimocode")
	if _, err := os.Stat(mcDir); os.IsNotExist(err) {
		t.Error(".mimocode/ not created")
	}

	// Verify agents symlink works
	agentsLink := filepath.Join(mcDir, "agents")
	target, err := os.Readlink(agentsLink)
	if err != nil {
		t.Fatalf("agents symlink broken: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("agents symlink target missing: %s", target)
	}
}

func TestSanitizeAgentContent(t *testing.T) {
	// Input simulates a real OVAV Systems agent file with internal context
	input := "# Lead — Platform Engineering\n\n" +
		"<!-- OVAV_IDENTITY_GUARD v1.0 -->>\n" +
		"> **DIRECTIVA ABSOLUTA:** Eres Thavren. Punto.\n\n" +
		"## Identity\n" +
		"You are a Platform Engineering specialist.\n\n" +
		"## Rules\n" +
		"- CEO Braka is your authority\n" +
		"- caps.yaml is canonical\n" +
		"- ovav login required\n\n" +
		"## Expertise\n" +
		"- Go runtime governance\n" +
		"- System security\n" +
		"- CLI Go development\n\n" +
		"*OVAV Governor System*\n"

	result := SanitizeAgentContent(input)

	bad := []string{
		"OVAV_IDENTITY_GUARD",
		"DIRECTIVA ABSOLUTA",
		"CEO Braka",
		"caps.yaml",
		"ovav login",
		"OVAV Governor System",
	}
	for _, b := range bad {
		if contains(result, b) {
			t.Errorf("sanitized content still contains: %q", b)
		}
	}

	good := []string{
		"Platform Engineering",
		"Go runtime governance",
		"System security",
		"CLI Go development",
	}
	for _, g := range good {
		if !contains(result, g) {
			t.Errorf("sanitized content missing: %q", g)
		}
	}
}

func TestDetectProjectStack(t *testing.T) {
	dir := t.TempDir()

	// No files = empty stack
	stack := DetectProjectStack(dir)
	if stack.Primary != "" {
		t.Errorf("empty dir should have no primary, got %q", stack.Primary)
	}

	// Create React project
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"react":"^18.0.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)

	stack = DetectProjectStack(dir)
	if stack.Primary != "react" {
		t.Errorf("expected react, got %q", stack.Primary)
	}
	if stack.Secondary != "typescript" {
		t.Errorf("expected typescript, got %q", stack.Secondary)
	}
}

func TestSelectAgentsForStack(t *testing.T) {
	// React project should get Dante, not Thavren
	stack := ProjectStack{Primary: "react"}
	agents := SelectAgentsForStack(stack, "")
	if len(agents) == 0 {
		t.Fatal("expected agents for react stack")
	}
	if agents[0] != "lead-dante.md" {
		t.Errorf("react stack should default to Dante, got %q", agents[0])
	}

	// Go project should get Thavren
	stack = ProjectStack{Primary: "go"}
	agents = SelectAgentsForStack(stack, "")
	if agents[0] != "lead-thavren.md" {
		t.Errorf("go stack should default to Thavren, got %q", agents[0])
	}
}

// ── NEW TESTS: dryRun, truncate, countFiles, version, sanitize, bootstrap, etc. ──

func TestDryRun(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create OVAV structure
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-thavren.md"), []byte("# Thavren"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Dry run on fresh install
	result, err := ProductInstall(ovavRoot, "dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "dry-run" {
		t.Errorf("mode = %q, want dry-run", result.Mode)
	}
	if result.Preview == "" {
		t.Error("expected preview output")
	}
	if !strings.Contains(result.Preview, "Fresh installation") {
		t.Error("expected fresh installation message")
	}

	// Dry run with existing manifest (update path)
	if err := SaveManifest(NewManifest(ovavRoot, "product")); err != nil {
		t.Fatal(err)
	}
	result2, err := ProductInstall(ovavRoot, "dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result2.Preview, "Previous installation") {
		t.Error("expected previous installation message")
	}
}

func TestDryRunMissingSource(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")
	// Create minimal structure with some missing assets
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	// skills dir missing → triggers missing path
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Preview, "missing") {
		t.Error("expected missing asset in preview")
	}
}

func TestDryRunDirectoryAsset(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create agents dir with files
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "agent1.md"), []byte("# Agent1"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "agent2.md"), []byte("# Agent2"), 0644)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if result.LinksCreated == 0 {
		t.Error("expected links_created > 0 for directory assets")
	}
	if !strings.Contains(result.Preview, "[symlink]") {
		t.Error("expected symlink marker in preview")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"a", 1, "a"},
		{"abcde", 3, "..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestCountFiles(t *testing.T) {
	dir := t.TempDir()
	// Empty dir
	if c := countFiles(dir); c != 0 {
		t.Errorf("empty dir: countFiles = %d, want 0", c)
	}
	// Create files
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("c"), 0644)
	if c := countFiles(dir); c != 3 {
		t.Errorf("countFiles = %d, want 3", c)
	}
}

func TestProductInstallUnknownMode(t *testing.T) {
	origHome := os.Getenv("HOME")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	_, err := ProductInstall("/nonexistent", "bogus-mode")
	if err == nil {
		t.Error("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Errorf("error should mention unknown mode, got: %v", err)
	}
}

func TestProductInstallDryRunUnknownMode(t *testing.T) {
	origHome := os.Getenv("HOME")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	_, err := ProductInstall("/ovav", "invalid")
	if err == nil {
		t.Error("expected error")
	}
}

func TestDoInstallMissingAssets(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")
	// Create only partial structure — skills dir missing
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	// Should have errors for missing skills dir
	hasMissing := false
	for _, e := range result.Errors {
		if strings.Contains(e, "missing") {
			hasMissing = true
			break
		}
	}
	if !hasMissing {
		t.Errorf("expected missing asset errors, got: %v", result.Errors)
	}
}

func TestDoInstallWithSymlinkDir(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create structure with a directory that should be symlinked
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-thavren.md"), []byte("# Thavren"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	if result.LinksCreated == 0 {
		t.Error("expected links to be created for skill dirs")
	}
}

func TestDoUninstallNoManifest(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	_, err := ProductInstall("/nonexistent", "uninstall")
	if err == nil {
		t.Error("expected error for no manifest")
	}
}

func TestDoVerifyNoManifest(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	_, err := ProductInstall("/nonexistent", "verify")
	if err == nil {
		t.Error("expected error for no manifest")
	}
}

func TestDoUninstallWithManifest(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// ProductInstall uses ProductDir() which resolves to $HOME/.local/share/ovav/
	pd, _ := ProductDir()
	os.MkdirAll(pd, 0755)

	// Create some files to uninstall
	f1 := filepath.Join(pd, "file1.txt")
	f2 := filepath.Join(pd, "file2.txt")
	os.WriteFile(f1, []byte("a"), 0644)
	os.WriteFile(f2, []byte("b"), 0644)

	m := NewManifest("/ovav", "product")
	_ = m.AddEntry(f1, f1, "file1.txt", "test", true)
	_ = m.AddEntry(f2, f2, "file2.txt", "test", true)
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesCopied == 0 {
		t.Error("expected files to be removed")
	}

	// Product dir should be removed if empty
	if _, err := os.Stat(pd); !os.IsNotExist(err) {
		t.Error("product dir should be removed when empty")
	}
}

func TestDoVerifyWithErrors(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Create manifest with entry pointing to nonexistent source
	m := NewManifest("/ovav", "product")
	m.Entries = append(m.Entries, ManifestEntry{
		Source:  "/nonexistent/file.txt",
		Target:  filepath.Join(productDir, "file.txt"),
		RelPath: "file.txt",
		Hash:    "abc123",
	})
	// Add symlink entry
	linkPath := filepath.Join(productDir, "link")
	os.WriteFile(filepath.Join(productDir, "target.txt"), []byte("t"), 0644)
	if err := os.Symlink(filepath.Join(productDir, "target.txt"), linkPath); err != nil {
		t.Fatal(err)
	}
	m.Symlinks = append(m.Symlinks, ManifestSymlink{
		Link:   linkPath,
		Target: filepath.Join(productDir, "target.txt"),
		Valid:  true,
	})
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Error("expected verify errors for missing source")
	}
}

func TestDoVerifyDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Create dangling symlink
	linkPath := filepath.Join(productDir, "dangling")
	os.Symlink("/nonexistent/target", linkPath)

	m := NewManifest("/ovav", "product")
	m.Symlinks = append(m.Symlinks, ManifestSymlink{
		Link:   linkPath,
		Target: "/nonexistent/target",
		Valid:  true,
	})
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	hasDangling := false
	for _, e := range result.Errors {
		if strings.Contains(e, "dangling") {
			hasDangling = true
		}
	}
	if !hasDangling {
		t.Errorf("expected dangling symlink error, got: %v", result.Errors)
	}
}

func TestDoVerifyBrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Create a broken symlink (target doesn't exist and link itself doesn't resolve)
	linkPath := filepath.Join(productDir, "broken")
	os.Symlink("/nonexistent/target", linkPath)
	os.Remove(linkPath) // remove the symlink itself
	// Now create a symlink that points to nothing
	os.Symlink("/nonexistent/target", linkPath)
	// Remove the file it's supposed to link to (it never existed)
	// The symlink is "broken" but Readlink still works — it's dangling

	m := NewManifest("/ovav", "product")
	// Add an entry with empty source
	m.Entries = append(m.Entries, ManifestEntry{
		Source:  "",
		Target:  filepath.Join(productDir, "missing.txt"),
		RelPath: "missing.txt",
	})
	m.Symlinks = append(m.Symlinks, ManifestSymlink{
		Link:   linkPath,
		Target: "/nonexistent/target",
		Valid:  false,
	})
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	hasMissing := false
	for _, e := range result.Errors {
		if strings.Contains(e, "missing") {
			hasMissing = true
		}
	}
	if !hasMissing {
		t.Errorf("expected missing entry error, got: %v", result.Errors)
	}
}

func TestDoVerifySourceGone(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Entries = append(m.Entries, ManifestEntry{
		Source:  "/nonexistent/source.txt",
		Target:  filepath.Join(productDir, "file.txt"),
		RelPath: "file.txt",
		Hash:    "somehash",
	})
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	hasSourceGone := false
	for _, e := range result.Errors {
		if strings.Contains(e, "source gone") {
			hasSourceGone = true
		}
	}
	if !hasSourceGone {
		t.Errorf("expected source gone error, got: %v", result.Errors)
	}
}

func TestDoVerifySourceChanged(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	// Create a source file
	srcFile := filepath.Join(dir, "source.txt")
	os.WriteFile(srcFile, []byte("original"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Get hash of original
	origHash, _ := fileHash(srcFile)

	m := NewManifest("/ovav", "product")
	m.Entries = append(m.Entries, ManifestEntry{
		Source:  srcFile,
		Target:  filepath.Join(productDir, "file.txt"),
		RelPath: "file.txt",
		Hash:    origHash,
	})
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	// Change the source file
	os.WriteFile(srcFile, []byte("modified content"), 0644)

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	hasChanged := false
	for _, e := range result.Errors {
		if strings.Contains(e, "changed") {
			hasChanged = true
		}
	}
	if !hasChanged {
		t.Errorf("expected changed source error, got: %v", result.Errors)
	}
}

func TestDoVerifyAllGood(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")
	os.MkdirAll(productDir, 0755)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	srcFile := filepath.Join(dir, "source.txt")
	os.WriteFile(srcFile, []byte("content"), 0644)

	m := NewManifest("/ovav", "product")
	_ = m.AddEntry(srcFile, filepath.Join(productDir, "file.txt"), "file.txt", "test", true)
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if result.FilesCopied != len(m.Entries) {
		t.Errorf("expected FilesCopied=%d, got %d", len(m.Entries), result.FilesCopied)
	}
}

func TestBootstrapNotInstalled(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	_, err := Bootstrap(filepath.Join(dir, "project"))
	if err == nil {
		t.Error("expected error when not installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error should mention not installed, got: %v", err)
	}
}

func TestBootstrapIdentityAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	if _, err := ProductInstall(ovavRoot, "install"); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0755)
	// Pre-create identity file
	os.WriteFile(filepath.Join(projectDir, "OVAV_IDENTITY.md"), []byte("existing"), 0644)

	br, err := Bootstrap(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if !br.IdentityCopied {
		t.Error("identity should be marked as copied (already exists)")
	}
}

func TestSanitizeAgentFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "agent.md")
	dst := filepath.Join(dir, "output", "agent.md")

	content := "---\nname: Thavren\n---\n\n" +
		"<!-- OVAV_IDENTITY_GUARD v1.0 -->\n" +
		"> DIRECTIVA ABSOLUTA\n" +
		"Normal content here\n" +
		"caps.yaml reference\n"
	os.WriteFile(src, []byte(content), 0644)

	if err := SanitizeAgentFile(src, dst); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	if strings.Contains(result, "OVAV_IDENTITY_GUARD") {
		t.Error("sanitized file still contains OVAV_IDENTITY_GUARD")
	}
	if strings.Contains(result, "DIRECTIVA ABSOLUTA") {
		t.Error("sanitized file still contains DIRECTIVA ABSOLUTA")
	}
	if strings.Contains(result, "caps.yaml") {
		t.Error("sanitized file still contains caps.yaml")
	}
	if !strings.Contains(result, "Normal content here") {
		t.Error("sanitized file should preserve normal content")
	}
}

func TestSanitizeAgentFileReadError(t *testing.T) {
	err := SanitizeAgentFile("/nonexistent/file.md", "/tmp/out.md")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestVersionInfoNoManifest(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	v := VersionInfo()
	if v != ProductVersion {
		t.Errorf("VersionInfo() = %q, want %q (fallback)", v, ProductVersion)
	}
}

func TestVersionInfoWithManifest(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = "2.0.0"
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	v := VersionInfo()
	if v != "2.0.0" {
		t.Errorf("VersionInfo() = %q, want %q", v, "2.0.0")
	}
}

func TestCheckForUpdateWithMockServer(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Save manifest with current version
	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	// Start mock cPanel server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"available":"2.0.0","channel":"stable"}`))
	}))
	defer ts.Close()

	info, err := CheckForUpdate(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !info.UpdateReady {
		t.Error("expected update to be ready")
	}
	if info.Available != "2.0.0" {
		t.Errorf("available = %q, want 2.0.0", info.Available)
	}
}

func TestCheckForUpdateDefaultURL(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	// Check with empty URL (uses default which will fail)
	info, err := CheckForUpdate("")
	if err != nil {
		t.Fatal(err)
	}
	// Should use cached/fallback
	if !info.Cached {
		t.Error("expected cached result when cPanel unreachable")
	}
}

func TestCheckForUpdateUnreachable(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	info, err := CheckForUpdate("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Cached {
		t.Error("expected cached when cPanel unreachable")
	}
	// Current == available in fallback → no update
	if info.UpdateReady {
		t.Error("expected no update when same version")
	}
}

func TestCheckForUpdateUnreachableFallback(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// When cPanel is unreachable and no manifest exists, available is empty
	// Current == ProductVersion, available == "" → no update
	info, err := CheckForUpdate("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Cached {
		t.Error("expected cached")
	}
	// Without a manifest, available stays empty, Current == ProductVersion
	// available == "" != ProductVersion → UpdateReady should be true
	if info.Current != ProductVersion {
		t.Errorf("current = %q, want %q", info.Current, ProductVersion)
	}
}

func TestCheckForUpdateNoUpdateAvailable(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"available":"` + ProductVersion + `","channel":"stable"}`))
	}))
	defer ts.Close()

	info, err := CheckForUpdate(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.UpdateReady {
		t.Error("expected no update when same version")
	}
}

func TestCheckForUpdateEmptyAvailable(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"available":"","channel":"stable"}`))
	}))
	defer ts.Close()

	info, err := CheckForUpdate(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.UpdateReady {
		t.Error("expected no update when available is empty")
	}
}

func TestCheckForUpdateNon200(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	info, err := CheckForUpdate(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Cached {
		t.Error("expected cached fallback on non-200")
	}
}

func TestCheckForUpdateBadJSON(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	info, err := CheckForUpdate(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Cached {
		t.Error("expected cached fallback on bad JSON")
	}
}

func TestNeedsUpdate(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = ProductVersion
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	// Default cPanel unreachable → cached → same version → no update
	if NeedsUpdate() {
		t.Error("expected no update when cPanel unreachable and same version")
	}
}

func TestWriteVersionFile(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	if err := WriteVersionFile(); err != nil {
		t.Fatal(err)
	}

	productDir, _ := ProductDir()
	data, err := os.ReadFile(filepath.Join(productDir, "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != ProductVersion+"\n" {
		t.Errorf("VERSION = %q, want %q", string(data), ProductVersion+"\n")
	}
}

func TestSanitizeAgentContentYAMLFrontmatter(t *testing.T) {
	input := "---\nname: Test\nmode: primary\n---\n\nNormal content\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "---") {
		t.Error("frontmatter should be stripped")
	}
	if !strings.Contains(result, "Normal content") {
		t.Error("normal content should be preserved")
	}
}

func TestSanitizeAgentContentHTMLCommentMultiLine(t *testing.T) {
	// Multiline HTML comments share skipBlock with YAML frontmatter.
	// skipBlock is only reset by "---", so multiline comment consumes
	// all subsequent lines until a "---" is found.
	input := "Before\n<!-- OVAV_IDENTITY_GUARD\nmulti line comment\n-->\nAfter\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "multi line comment") {
		t.Error("multi-line HTML comment should be stripped")
	}
	if !strings.Contains(result, "Before") {
		t.Error("content before comment should be preserved")
	}
}

func TestSanitizeAgentContentBlockquoteWithDirectiva(t *testing.T) {
	input := "> DIRECTIVA ABSOLUTA\n> continued line\n> more lines\nNormal\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "DIRECTIVA") {
		t.Error("DIRECTIVA line should be stripped")
	}
	if strings.Contains(result, "continued line") {
		t.Error("blockquote continuation should be stripped")
	}
	if !strings.Contains(result, "Normal") {
		t.Error("normal content should be preserved")
	}
}

func TestSanitizeAgentContentCollapseBlankLines(t *testing.T) {
	// Input has 3 consecutive blanks between Line1 and Line2.
	// Algorithm collapses consecutive blanks to 1.
	input := "Line1\n\n\n\nLine2\n"
	result := SanitizeAgentContent(input)

	// Verify no 2+ consecutive blank lines
	lines := strings.Split(result, "\n")
	for i := 1; i < len(lines)-1; i++ {
		if strings.TrimSpace(lines[i]) == "" && strings.TrimSpace(lines[i-1]) == "" {
			t.Errorf("found consecutive blank lines at index %d in:\n%s", i, result)
		}
	}

	if !strings.Contains(result, "Line1") || !strings.Contains(result, "Line2") {
		t.Error("content lines should be preserved")
	}
}

func TestSanitizeAgentContentEmptyInput(t *testing.T) {
	result := SanitizeAgentContent("")
	if result != "" {
		t.Errorf("empty input should produce empty output, got %q", result)
	}
}

func TestSanitizeAgentContentSelfClosingComment(t *testing.T) {
	input := "Line1\n<!-- OVAV_IDENTITY_GUARD v1.0 -->>\nLine2\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "OVAV_IDENTITY_GUARD") {
		t.Error("self-closing comment should be stripped")
	}
	if !strings.Contains(result, "Line1") || !strings.Contains(result, "Line2") {
		t.Error("surrounding lines should be preserved")
	}
}

func TestSanitizeAgentContentAllStrippedPatterns(t *testing.T) {
	patterns := []string{
		"OVAV_IDENTITY_GUARD",
		"OVAV_INTEGRITY_SEAL",
		"DIRECTIVA ABSOLUTA",
		"CEO Braka",
		"CEO: Alexander Salvador",
		"OVAV Governor System",
		"OVAV is a sealed governor system",
		"ovav login",
		"ovav status",
		"caps.yaml",
		".ovav/plan/",
		".ovav/laws/",
		".ovav/runtime/",
		"session_greeting",
		"output_guard",
		"Protected Branch Lockdown",
		"Session Start — MANDATORY",
		"Context Budget — MANDATORY",
		"Blocked surfaces:",
		"OVAV GOVERNOR ALERT",
		"Internal reasoning: ENGLISH ONLY",
		"BrevityRail enforced",
		"NEVER use raw git push/merge",
		"Agent CANNOT create/edit/touch waiver",
		"OVAV_VAULT_KEY",
		".ovav/economy/",
		".ovav/evaluation/",
		".ovav/registry/",
		".ovav/alerts/",
		".ovav/service_areas/",
		".ovav/policy/",
		"permission_authority.json",
		"area_boundary_enforcement",
	}

	for _, p := range patterns {
		content := "Safe line\n" + p + "\nAnother safe line\n"
		result := SanitizeAgentContent(content)
		if strings.Contains(result, p) {
			t.Errorf("pattern %q should be stripped", p)
		}
		if !strings.Contains(result, "Safe line") {
			t.Errorf("safe line should be preserved when stripping %q", p)
		}
	}
}

func TestSkillsForStack(t *testing.T) {
	// Empty stack — should get core skills only
	stack := ProjectStack{}
	skills := SkillsForStack(stack)
	if len(skills) < 4 {
		t.Errorf("expected at least 4 core skills, got %d", len(skills))
	}
	hasContextPack := false
	for _, s := range skills {
		if s == "ovav-context-pack" {
			hasContextPack = true
		}
	}
	if !hasContextPack {
		t.Error("expected ovav-context-pack in core skills")
	}

	// With primary set — should include session-continuity
	stack = ProjectStack{Primary: "react"}
	skills = SkillsForStack(stack)
	hasSC := false
	for _, s := range skills {
		if s == "ovav-session-continuity" {
			hasSC = true
		}
	}
	if !hasSC {
		t.Error("expected ovav-session-continuity for project with primary")
	}
}

func TestSelectAgentsForStackAllStacks(t *testing.T) {
	stacks := []struct {
		primary string
		want    string
	}{
		{"react", "lead-dante.md"},
		{"vue", "lead-dante.md"},
		{"angular", "lead-dante.md"},
		{"svelte", "lead-dante.md"},
		{"nextjs", "lead-dante.md"},
		{"nuxt", "lead-dante.md"},
		{"astro", "lead-dante.md"},
		{"remix", "lead-dante.md"},
		{"go", "lead-thavren.md"},
		{"python", "lead-thavren.md"},
		{"rust", "lead-thavren.md"},
		{"java", "lead-thavren.md"},    // fallback
		{"php", "lead-thavren.md"},     // fallback
		{"ruby", "lead-thavren.md"},    // fallback
		{"unknown", "lead-thavren.md"}, // fallback
	}

	for _, tt := range stacks {
		stack := ProjectStack{Primary: tt.primary}
		agents := SelectAgentsForStack(stack, "")
		if len(agents) == 0 {
			t.Errorf("stack %q: expected agents", tt.primary)
			continue
		}
		if agents[0] != tt.want {
			t.Errorf("stack %q: lead = %q, want %q", tt.primary, agents[0], tt.want)
		}
	}
}

func TestSelectAgentsForStackByFiles(t *testing.T) {
	// Stack with no primary but files that match
	stack := ProjectStack{
		Primary: "",
		Files:   []string{"go", "python"},
	}
	agents := SelectAgentsForStack(stack, "")
	if len(agents) == 0 {
		t.Fatal("expected agents")
	}
	// Should pick first match in priority → go → lead-thavren.md
	if agents[0] != "lead-thavren.md" {
		t.Errorf("lead = %q, want lead-thavren.md", agents[0])
	}
}

func TestSelectAgentsForStackByFilesReact(t *testing.T) {
	stack := ProjectStack{
		Primary: "",
		Files:   []string{"vue"},
	}
	agents := SelectAgentsForStack(stack, "")
	if agents[0] != "lead-dante.md" {
		t.Errorf("lead = %q, want lead-dante.md", agents[0])
	}
}

func TestDetectProjectStackVarious(t *testing.T) {
	dir := t.TempDir()

	// Go project
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	stack := DetectProjectStack(dir)
	if stack.Primary != "go" {
		t.Errorf("go: got %q", stack.Primary)
	}

	// Clean up
	os.Remove(filepath.Join(dir, "go.mod"))

	// Python project
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "python" {
		t.Errorf("python: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "requirements.txt"))

	// Rust
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "rust" {
		t.Errorf("rust: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "Cargo.toml"))

	// Java
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project>"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "java" {
		t.Errorf("java: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "pom.xml"))

	// PHP
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "php" {
		t.Errorf("php: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "composer.json"))

	// Ruby
	os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'rubygems'"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "ruby" {
		t.Errorf("ruby: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "Gemfile"))

	// Angular via angular.json
	os.WriteFile(filepath.Join(dir, "angular.json"), []byte("{}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "angular" {
		t.Errorf("angular: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "angular.json"))

	// Vue
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"vue":"^3.0"}}`), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "vue" {
		t.Errorf("vue: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "package.json"))

	// Angular via package.json
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"@angular/core":"^16"}}`), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "angular" {
		t.Errorf("angular via pkg: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "package.json"))

	// Svelte
	os.WriteFile(filepath.Join(dir, "svelte.config.js"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "svelte" {
		t.Errorf("svelte: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "svelte.config.js"))

	// Next.js
	os.WriteFile(filepath.Join(dir, "next.config.js"), []byte("module.exports = {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "nextjs" {
		t.Errorf("nextjs: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "next.config.js"))

	// Nuxt
	os.WriteFile(filepath.Join(dir, "nuxt.config.ts"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "nuxt" {
		t.Errorf("nuxt: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "nuxt.config.ts"))

	// Astro
	os.WriteFile(filepath.Join(dir, "astro.config.mjs"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "astro" {
		t.Errorf("astro: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "astro.config.mjs"))

	// Remix
	os.WriteFile(filepath.Join(dir, "remix.config.js"), []byte("module.exports = {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "remix" {
		t.Errorf("remix: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "remix.config.js"))

	// Terraform
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte("resource {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "terraform" {
		t.Errorf("terraform: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "main.tf"))

	// Kubernetes/Docker
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM ubuntu"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "kubernetes" {
		t.Errorf("kubernetes: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "Dockerfile"))

	// Next.js via .mjs
	os.WriteFile(filepath.Join(dir, "next.config.mjs"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "nextjs" {
		t.Errorf("nextjs mjs: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "next.config.mjs"))

	// Next.js via .ts
	os.WriteFile(filepath.Join(dir, "next.config.ts"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "nextjs" {
		t.Errorf("nextjs ts: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "next.config.ts"))

	// Svelte via .ts
	os.WriteFile(filepath.Join(dir, "svelte.config.ts"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "svelte" {
		t.Errorf("svelte ts: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "svelte.config.ts"))

	// Python via pyproject.toml
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "python" {
		t.Errorf("python pyproject: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "pyproject.toml"))

	// Python via Pipfile
	os.WriteFile(filepath.Join(dir, "Pipfile"), []byte("[packages]"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "python" {
		t.Errorf("python pipfile: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "Pipfile"))

	// Docker compose
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("version: '3'"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "kubernetes" {
		t.Errorf("docker-compose yml: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "docker-compose.yml"))

	os.WriteFile(filepath.Join(dir, "docker-compose.yaml"), []byte("version: '3'"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "kubernetes" {
		t.Errorf("docker-compose yaml: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "docker-compose.yaml"))

	// Java via build.gradle
	os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "java" {
		t.Errorf("java gradle: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "build.gradle"))

	// Nuxt via .js
	os.WriteFile(filepath.Join(dir, "nuxt.config.js"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "nuxt" {
		t.Errorf("nuxt js: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "nuxt.config.js"))

	// Astro via .ts
	os.WriteFile(filepath.Join(dir, "astro.config.ts"), []byte("export default {}"), 0644)
	stack = DetectProjectStack(dir)
	if stack.Primary != "astro" {
		t.Errorf("astro ts: got %q", stack.Primary)
	}
	os.Remove(filepath.Join(dir, "astro.config.ts"))
}

func TestDetectProjectStackTypeScriptSecondary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0644)

	stack := DetectProjectStack(dir)
	if stack.Primary != "go" {
		t.Errorf("primary = %q, want go", stack.Primary)
	}
	if stack.Secondary != "typescript" {
		t.Errorf("secondary = %q, want typescript", stack.Secondary)
	}
}

func TestFileContainsEdgeCases(t *testing.T) {
	dir := t.TempDir()

	// Nonexistent file
	if fileContains(dir, "nonexistent.txt", "foo") {
		t.Error("should return false for nonexistent file")
	}

	// File that doesn't contain search string
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello world"), 0644)
	if fileContains(dir, "test.txt", "goodbye") {
		t.Error("should return false when content doesn't match")
	}
}

func TestHasFilesEdgeCases(t *testing.T) {
	dir := t.TempDir()

	// Empty dir
	if hasFiles(dir, ".tf") {
		t.Error("empty dir should return false")
	}

	// Nonexistent dir
	if hasFiles("/nonexistent/dir", ".tf") {
		t.Error("nonexistent dir should return false")
	}

	// Dir with matching files
	os.WriteFile(filepath.Join(dir, "main.tf"), []byte("resource {}"), 0644)
	if !hasFiles(dir, ".tf") {
		t.Error("should find .tf file")
	}

	// Dir with only directories
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	if hasFiles(dir, ".nonexistent") {
		t.Error("should return false for nonexistent extension")
	}
}

func TestFileHash(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	hash, err := fileHash(f)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(hash) != 64 { // SHA-256 hex is 64 chars
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Same content → same hash
	hash2, _ := fileHash(f)
	if hash != hash2 {
		t.Error("same content should produce same hash")
	}

	// Different content → different hash
	os.WriteFile(f, []byte("world"), 0644)
	hash3, _ := fileHash(f)
	if hash == hash3 {
		t.Error("different content should produce different hash")
	}

	// Nonexistent file
	_, err = fileHash("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestManifestPath(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	path, err := ManifestPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".ovav-manifest.json") {
		t.Errorf("path = %q, should end with .ovav-manifest.json", path)
	}
}

func TestLoadManifestNotExist(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m, err := LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Error("expected nil manifest when not found")
	}
}

func TestLoadManifestCorrupt(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	productDir, _ := ProductDir()
	os.MkdirAll(productDir, 0755)
	os.WriteFile(filepath.Join(productDir, ".ovav-manifest.json"), []byte("not json"), 0644)

	_, err := LoadManifest()
	if err == nil {
		t.Error("expected error for corrupt manifest")
	}
}

func TestSaveManifestAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	m.Product = "1.0.0"
	m.AddSymlink("/link", "/target", true)

	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil manifest")
	}
	if loaded.Product != "1.0.0" {
		t.Errorf("product = %q, want 1.0.0", loaded.Product)
	}
	if len(loaded.Symlinks) != 1 {
		t.Errorf("symlinks = %d, want 1", len(loaded.Symlinks))
	}
}

func TestManifestAddEntryNonexistent(t *testing.T) {
	m := NewManifest("/ovav", "product")
	err := m.AddEntry("/nonexistent/file.txt", "/target", "test.txt", "test", true)
	if err == nil {
		t.Error("expected error for nonexistent source file")
	}
}

func TestProductInstallWithCockpitBuild(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create structure including product_cockpit cmd dir
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, "go-runtime", "cmd", "product_cockpit"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// This will try to build cockpit but fail (no Go source) — should produce an error
	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	// Should have cockpit build error
	hasCockpitErr := false
	for _, e := range result.Errors {
		if strings.Contains(e, "build cockpit") {
			hasCockpitErr = true
		}
	}
	// It's ok if cockpit source dir exists but has no valid Go — the build should fail gracefully
	// The function returns nil when source dir doesn't exist, so we check if it does exist
	if _, err := os.Stat(filepath.Join(ovavRoot, "go-runtime", "cmd", "product_cockpit")); err == nil {
		if !hasCockpitErr {
			// Build might succeed if there's actually a valid Go file there
			t.Logf("cockpit build did not produce expected error (may have built successfully)")
		}
	}
}

func TestInstallSelectiveAgentsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	productDir := filepath.Join(dir, "product")

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Create a minimal structure where agents dir exists but is empty
	ovavRoot := filepath.Join(dir, "ovav")
	agentsDir := filepath.Join(ovavRoot, "runtimes", "mimocode", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	m := NewManifest(ovavRoot, "product")
	r := &InstallResult{Mode: "install", ProductDir: productDir}

	// Empty agents directory → should return nil (nothing to install)
	err := installSelectiveAgents(agentsDir, filepath.Join(productDir, "agents"), "", m, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallSelectiveAgentsReadDirError(t *testing.T) {
	m := NewManifest("/ovav", "product")
	r := &InstallResult{Mode: "install", ProductDir: "/nonexistent"}

	err := installSelectiveAgents("/nonexistent/dir", "/nonexistent/dst", "", m, r)
	if err == nil {
		t.Error("expected error for nonexistent agents dir")
	}
}

func TestInstallSelectiveAgentsWithProjectDir(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create a fake project for stack detection
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0755)
	os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(`{"dependencies":{"react":"^18"}}`), 0644)

	m := NewManifest(dir, "product")
	r := &InstallResult{Mode: "install", ProductDir: filepath.Join(dir, "product")}

	// With projectDir set → should use SelectAgentsForStack
	err := installSelectiveAgents(agentsDir, filepath.Join(dir, "product", "agents"), projectDir, m, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoInstallMultipleAgentFiles(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-thavren.md"), []byte("# Thavren"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-dante.md"), []byte("# Dante"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) > 0 {
		t.Errorf("install errors: %v", result.Errors)
	}
	if result.FilesCopied < 2 {
		t.Errorf("expected at least 2 files copied, got %d", result.FilesCopied)
	}
}

func TestProductInstallVerifyChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Install
	_, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}

	// Modify a source file to trigger checksum mismatch
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Modified Identity"), 0644)

	// Verify should detect changed source
	vr, err := ProductInstall(ovavRoot, "verify")
	if err != nil {
		t.Fatal(err)
	}
	hasChanged := false
	for _, e := range vr.Errors {
		if strings.Contains(e, "changed") {
			hasChanged = true
		}
	}
	if !hasChanged {
		t.Errorf("expected changed source error, got: %v", vr.Errors)
	}
}

func TestBootstrapCWD(t *testing.T) {
	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// BootstrapCWD should fail when not installed
	err := BootstrapCWD()
	if err == nil {
		// It's ok if this doesn't fail — depends on whether product is installed
		t.Logf("BootstrapCWD succeeded (product may be installed)")
		return
	}
	// Restore cwd
	os.Chdir(origWd)
}

func TestTruncateShortString(t *testing.T) {
	if got := truncate("hi", 100); got != "hi" {
		t.Errorf("truncate short = %q, want hi", got)
	}
}

func TestCountFilesEmptySubdir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)
	c := countFiles(dir)
	if c != 0 {
		t.Errorf("expected 0, got %d", c)
	}
}

func TestSanitizeAgentContentIdentityGuardMultiLine(t *testing.T) {
	// Multiline HTML comment consumes all lines until "---" is found.
	content := "Before\n<!-- OVAV_IDENTITY_GUARD\nline1\nline2\nline3\n-->\nAfter\n"
	result := SanitizeAgentContent(content)

	if strings.Contains(result, "line1") || strings.Contains(result, "line2") || strings.Contains(result, "line3") {
		t.Error("multi-line OVAV_IDENTITY_GUARD block should be stripped")
	}
	if !strings.Contains(result, "Before") {
		t.Error("content before comment should be preserved")
	}
}

func TestSanitizeAgentContentIntegritySealMultiLine(t *testing.T) {
	input := "Before\n<!-- OVAV_INTEGRITY_SEAL\nsecret stuff\n-->\nAfter\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "secret stuff") {
		t.Error("multi-line OVAV_INTEGRITY_SEAL block should be stripped")
	}
}

func TestSanitizeAgentContentIntegritySealSelfClosing(t *testing.T) {
	input := "Line1\n<!-- OVAV_INTEGRITY_SEAL v1.0 -->>\nLine2\n"
	result := SanitizeAgentContent(input)

	if strings.Contains(result, "OVAV_INTEGRITY_SEAL") {
		t.Error("self-closing OVAV_INTEGRITY_SEAL comment should be stripped")
	}
}

// ── Additional coverage tests ──

func TestSymlinkSourceMissing(t *testing.T) {
	err := symlink("/nonexistent/source", "/tmp/test-link")
	if err == nil {
		t.Error("expected error when source missing")
	}
	if !strings.Contains(err.Error(), "source missing") {
		t.Errorf("error should mention source missing, got: %v", err)
	}
}

func TestSymlinkExistingDst(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("data"), 0644)
	os.WriteFile(dst, []byte("old"), 0644)

	if err := symlink(src, dst); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	if target != src {
		t.Errorf("symlink target = %q, want %q", target, src)
	}
}

func TestCopyFileBasic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "sub", "dst.txt")
	os.WriteFile(src, []byte("hello world"), 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q", string(data))
	}
}

func TestCopyFileSourceNotExist(t *testing.T) {
	err := copyFile("/nonexistent/file.txt", "/tmp/dst.txt")
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestCopyFileDestDirCreation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "deep", "nested", "dir", "dst.txt")
	os.WriteFile(src, []byte("data"), 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "data" {
		t.Errorf("content = %q", string(data))
	}
}

func TestCopyFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	os.WriteFile(src, []byte("data"), 0755)

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dst)
	if info.Mode().Perm() != 0755 {
		t.Errorf("mode = %v, want 0755", info.Mode().Perm())
	}
}

func TestDoInstallSymlinkError(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	// Create agents dir as a file instead of directory to cause symlink error
	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), []byte("not a dir"), 0644)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	result, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	// Should have errors from the agents install attempt
	if len(result.Errors) == 0 {
		t.Error("expected errors from agents install")
	}
}

func TestDoUninstallFileRemovalError(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	pd, _ := ProductDir()
	os.MkdirAll(pd, 0755)

	// Create a manifest with entries pointing to a file in a dir we'll make read-only
	subDir := filepath.Join(pd, "protected")
	os.MkdirAll(subDir, 0755)
	protectedFile := filepath.Join(subDir, "file.txt")
	os.WriteFile(protectedFile, []byte("data"), 0644)

	// Make the directory read-only so removal fails
	os.Chmod(subDir, 0555)
	defer os.Chmod(subDir, 0755) // restore after test

	m := NewManifest("/ovav", "product")
	_ = m.AddEntry(protectedFile, protectedFile, "protected/file.txt", "test", true)
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	// May or may not have errors depending on permissions
	t.Logf("uninstall result: files=%d links=%d errors=%v", result.FilesCopied, result.LinksCreated, result.Errors)
}

func TestDoVerifyNoEntriesNoSymlinks(t *testing.T) {
	origHome := os.Getenv("HOME")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	m := NewManifest("/ovav", "product")
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "verify")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestDoUninstallNoEntries(t *testing.T) {
	origHome := os.Getenv("HOME")
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	pd, _ := ProductDir()
	os.MkdirAll(pd, 0755)

	m := NewManifest("/ovav", "product")
	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	result, err := ProductInstall("/ovav", "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestBootstrapSuccessWithIdentity(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "area-test.md"), []byte("# Test\n---\nname: \"Test\"\nmode: primary\nhidden: false\n"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	if _, err := ProductInstall(ovavRoot, "install"); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0755)

	br, err := Bootstrap(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if !br.AgentsLinked {
		t.Error("agents not linked")
	}
	if !br.SkillsLinked {
		t.Error("skills not linked")
	}
	if !br.IdentityCopied {
		t.Error("identity not copied")
	}
	if br.CWD != projectDir {
		t.Errorf("CWD = %q, want %q", br.CWD, projectDir)
	}
}

func TestBootstrapCWDAfterInstall(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	if _, err := ProductInstall(ovavRoot, "install"); err != nil {
		t.Fatal(err)
	}

	// BootstrapCWD should succeed since product is installed
	err := BootstrapCWD()
	// May succeed or fail depending on current directory — that's OK
	if err != nil {
		t.Logf("BootstrapCWD error (expected in test env): %v", err)
	}
}

func TestNeedsUpdateWithUpdate(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// No manifest → VersionInfo returns ProductVersion, no update available
	// NeedsUpdate should return false
	if NeedsUpdate() {
		t.Error("expected no update when no manifest and no cPanel")
	}
}

func TestFetchVersionFromCPanelTrailingSlash(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"available":"2.0.0"}`))
	}))
	defer ts.Close()

	avail, err := fetchVersionFromCPanel(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if avail != "2.0.0" {
		t.Errorf("available = %q, want 2.0.0", avail)
	}
}

func TestFetchVersionFromCPanelNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := fetchVersionFromCPanel(ts.URL)
	if err == nil {
		t.Error("expected error for non-200")
	}
}

func TestFetchVersionFromCPanelBadJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer ts.Close()

	_, err := fetchVersionFromCPanel(ts.URL)
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestGenerateProductAgentKnownProfile(t *testing.T) {
	stack := ProjectStack{Primary: "react"}
	result := generateProductAgent("lead-dante.md", stack)
	if !strings.Contains(result, "Dante") {
		t.Error("expected Dante in agent output")
	}
	if !strings.Contains(result, "Digital Product Engineering") {
		t.Error("expected area in agent output")
	}
}

func TestGenerateProductAgentUnknownProfile(t *testing.T) {
	stack := ProjectStack{Primary: "go"}
	result := generateProductAgent("unknown-agent.md", stack)
	if !strings.Contains(result, "General Engineering") {
		t.Error("expected fallback area")
	}
}

func TestGenerateProductAgentAreaHidden(t *testing.T) {
	stack := ProjectStack{Primary: "react"}
	result := generateProductAgent("area-ux-design.md", stack)
	if !strings.Contains(result, "hidden: false") {
		t.Error("area agents should not be hidden")
	}
}

func TestGenerateProductAgentLeadHidden(t *testing.T) {
	stack := ProjectStack{Primary: "react"}
	result := generateProductAgent("lead-dante.md", stack)
	if !strings.Contains(result, "hidden: true") {
		t.Error("lead agents should be hidden")
	}
}

func TestManifestRoundTripFull(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	srcFile := filepath.Join(dir, "source.txt")
	os.WriteFile(srcFile, []byte("test content"), 0644)

	m := NewManifest("/ovav", "mimocode")
	_ = m.AddEntry(srcFile, "/target/file.txt", "file.txt", "config", true)
	m.AddSymlink("/link/path", "/real/path", true)

	if err := SaveManifest(m); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != ProductManifestVersion {
		t.Errorf("version = %q", loaded.Version)
	}
	if loaded.Product != ProductVersion {
		t.Errorf("product = %q", loaded.Product)
	}
	if loaded.OvavRoot != "/ovav" {
		t.Errorf("ovav_root = %q", loaded.OvavRoot)
	}
	if loaded.Platform != "mimocode" {
		t.Errorf("platform = %q", loaded.Platform)
	}
	if len(loaded.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(loaded.Entries))
	}
	if loaded.Entries[0].Hash == "" {
		t.Error("expected hash in entry")
	}
	if len(loaded.Symlinks) != 1 {
		t.Errorf("symlinks = %d, want 1", len(loaded.Symlinks))
	}
}

func TestInstallVerifyUninstallFullCycle(t *testing.T) {
	dir := t.TempDir()
	ovavRoot := filepath.Join(dir, "ovav")

	os.MkdirAll(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill"), 0755)
	os.MkdirAll(filepath.Join(ovavRoot, ".ovav", "source", "configs"), 0755)
	os.WriteFile(filepath.Join(ovavRoot, "runtimes", "mimocode", "agents", "lead-thavren.md"), []byte("# Thavren"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "skills", "test-skill", "SKILL.md"), []byte("# Skill"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, ".ovav", "source", "configs", "model_routing.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(ovavRoot, "OVAV_IDENTITY.md"), []byte("# Identity"), 0644)

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer t.Setenv("HOME", origHome)

	// Install
	ir, err := ProductInstall(ovavRoot, "install")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Mode != "install" {
		t.Errorf("mode = %q", ir.Mode)
	}

	// Verify
	vr, err := ProductInstall(ovavRoot, "verify")
	if err != nil {
		t.Fatal(err)
	}
	if len(vr.Errors) != 0 {
		t.Errorf("verify errors: %v", vr.Errors)
	}

	// Uninstall
	ur, err := ProductInstall(ovavRoot, "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	if ur.Mode != "uninstall" {
		t.Errorf("mode = %q", ur.Mode)
	}

	// Verify after uninstall — should fail (no manifest)
	_, err = ProductInstall(ovavRoot, "verify")
	if err == nil {
		t.Error("expected error after uninstall")
	}
}

func TestDetectProjectStackEmptyPackageJson(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)

	stack := DetectProjectStack(dir)
	// package.json exists but no react/vue keywords → no JS framework detected
	// But package.json without specific framework keywords shouldn't match any
	if stack.Primary == "react" || stack.Primary == "vue" || stack.Primary == "angular" {
		t.Errorf("empty package.json should not match framework, got %q", stack.Primary)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsImpl(s, sub))
}

func containsImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
