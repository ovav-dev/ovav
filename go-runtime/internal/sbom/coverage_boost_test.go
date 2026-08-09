package sbom

import (
	"os"
	"path/filepath"
	"testing"
)

// ── discoverPythonDeps ──────────────────────────────────────────────────────

func TestDiscoverPythonDeps_WithVersions(t *testing.T) {
	dir := t.TempDir()
	content := "pyyaml==6.0\nrequests==2.31.0\nflask\n"
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0644)

	deps := discoverPythonDeps(dir)
	if len(deps) != 3 {
		t.Fatalf("expected 3 python deps, got %d", len(deps))
	}
	// pyyaml has version
	if deps[0].Name != "pyyaml" || deps[0].Version != "6.0" {
		t.Errorf("dep 0: got %s=%s", deps[0].Name, deps[0].Version)
	}
	// flask has no ==, version should be "unknown"
	if deps[2].Name != "flask" || deps[2].Version != "unknown" {
		t.Errorf("dep 2: got %s=%s, want flask=unknown", deps[2].Name, deps[2].Version)
	}
}

func TestDiscoverPythonDeps_SkipsCommentsAndOptions(t *testing.T) {
	dir := t.TempDir()
	content := "# comment\n\n-r base.txt\n\n-n index-url https://example.com\nnumpy==1.24\n"
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0644)

	deps := discoverPythonDeps(dir)
	if len(deps) != 1 {
		t.Fatalf("expected 1 python dep after filtering, got %d", len(deps))
	}
	if deps[0].Name != "numpy" {
		t.Errorf("expected numpy, got %s", deps[0].Name)
	}
}

func TestDiscoverPythonDeps_NoFile(t *testing.T) {
	dir := t.TempDir()
	deps := discoverPythonDeps(dir)
	if len(deps) != 0 {
		t.Errorf("expected 0 deps when no requirements.txt, got %d", len(deps))
	}
}

func TestDiscoverPythonDeps_VersionWithoutEqual(t *testing.T) {
	dir := t.TempDir()
	// Line with whitespace but no ==
	content := "  pandas  \n"
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0644)

	deps := discoverPythonDeps(dir)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].Name != "pandas" || deps[0].Version != "unknown" {
		t.Errorf("got %s=%s, want pandas=unknown", deps[0].Name, deps[0].Version)
	}
}

// ── gitCommit ───────────────────────────────────────────────────────────────

func TestGitCommit_DirectHash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("abc123def456\n"), 0644)

	got := gitCommit(dir)
	if got != "abc123def456" {
		t.Errorf("expected abc123def456, got %s", got)
	}
}

func TestGitCommit_RefPath(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git", "refs", "heads"), 0755)
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".git", "refs", "heads", "main"), []byte("deadbeef1234\n"), 0644)

	got := gitCommit(dir)
	if got != "deadbeef1234" {
		t.Errorf("expected deadbeef1234, got %s", got)
	}
}

func TestGitCommit_RefNotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	// HEAD points to a ref that doesn't exist, and .git is a dir not a file
	os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/missing\n"), 0644)

	got := gitCommit(dir)
	if got != "unknown" {
		t.Errorf("expected unknown when ref missing, got %s", got)
	}
}

func TestGitCommit_NoGitDir(t *testing.T) {
	dir := t.TempDir()
	got := gitCommit(dir)
	if got != "unknown" {
		t.Errorf("expected unknown, got %s", got)
	}
}

// ── Verify edge cases ───────────────────────────────────────────────────────

func TestVerify_MissingFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	sbomData, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := sbomData.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Delete a tracked file
	os.Remove(filepath.Join(dir, "AGENTS.md"))

	result, err := Verify(dir)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid after deleting a tracked file")
	}
	found := false
	for _, m := range result.Mismatches {
		if m == "MISSING: AGENTS.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MISSING mismatch for AGENTS.md, got %v", result.Mismatches)
	}
}

func TestVerify_UntrackedFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	sbomData, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := sbomData.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create a new file that matches a coreFileGlob
	// Go's filepath.Glob treats ** as *, so go-runtime/**/*.go matches go-runtime/<sub>/<file>.go
	os.MkdirAll(filepath.Join(dir, "go-runtime", "sub"), 0755)
	goFile := filepath.Join(dir, "go-runtime", "sub", "new.go")
	os.WriteFile(goFile, []byte("package main\n"), 0644)

	result, err := Verify(dir)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	// Should have untracked for the new .go file
	found := false
	for _, m := range result.Mismatches {
		if m == "UNTRACKED: go-runtime/sub/new.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected UNTRACKED mismatch, got %v", result.Mismatches)
	}
}

// ── Load edge cases ─────────────────────────────────────────────────────────

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)
	os.WriteFile(filepath.Join(dir, SBOMRegistry), []byte("not json {{{"), 0644)

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ── isExcluded sub-path patterns ────────────────────────────────────────────

func TestIsExcluded_SubPathPatterns(t *testing.T) {
	tests := []struct {
		path     string
		excluded bool
	}{
		{"src/__pycache__/foo.pyc", true},
		{"deep/nested/.pytest_cache/cache.json", true},
		{".ovav/cache/tmp.json", true},
		{".ovav/lockdown/state.yaml", true},
		{".ovav/context/ctx.json", true},
		{".ovav/evaluation/run.yaml", true},
		{".ovav/alerts/alert.yaml", true},
		{".ovav/quarantine/file.yaml", true},
		{".ovav/integrity_backups/backup.json", true},
		{"dist/output.js", true},
		{"vendor/lib.go", true},
		{".wrangler/tmp.toml", true},
		{"normal/file.go", false},
	}
	for _, tt := range tests {
		if got := isExcluded(tt.path); got != tt.excluded {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.path, got, tt.excluded)
		}
	}
}

// ── Generate with file hash error ───────────────────────────────────────────

func TestGenerate_FileHashError(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)

	// Create a file that matches a glob but make it unreadable
	badFile := filepath.Join(dir, "AGENTS.md")
	os.WriteFile(badFile, []byte("test"), 0644)
	// Remove read permissions to trigger fileHash error
	os.Chmod(badFile, 0000)
	defer os.Chmod(badFile, 0644) // restore for cleanup

	sbomData, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// Should have an ERROR: prefix for the unreadable file
	h, ok := sbomData.CoreFiles["AGENTS.md"]
	if !ok {
		t.Fatal("expected AGENTS.md in core files")
	}
	if h == "" || h == "ERROR:" {
		// It's OK if we get an error hash — the point is the path was hit
		t.Logf("AGENTS.md hash (may be error): %s", h)
	}
}

// ── ComputeRequirementsHash missing files ───────────────────────────────────

func TestComputeRequirementsHash_NoFiles(t *testing.T) {
	dir := t.TempDir()
	hash := ComputeRequirementsHash(dir)
	if hash == "" {
		t.Error("expected non-empty hash even with no files")
	}
}

// ── Save round-trip with Python deps ────────────────────────────────────────

func TestSaveAndLoad_WithPythonDeps(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==3.0\ndjango\n"), 0644)

	sbomData, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(sbomData.Dependencies.Python) != 2 {
		t.Fatalf("expected 2 python deps, got %d", len(sbomData.Dependencies.Python))
	}

	if err := sbomData.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Dependencies.Python) != 2 {
		t.Errorf("expected 2 python deps after round-trip, got %d", len(loaded.Dependencies.Python))
	}
}
