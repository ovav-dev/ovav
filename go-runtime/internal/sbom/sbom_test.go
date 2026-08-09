package sbom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	// Create minimal Go structure
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash content\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	sbom, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if sbom.SchemaVersion != SchemaVersion {
		t.Errorf("expected schema %s, got %s", SchemaVersion, sbom.SchemaVersion)
	}
	if sbom.Project != "OVAV" {
		t.Errorf("expected project OVAV, got %s", sbom.Project)
	}
	if sbom.HashAlgorithm != "sha256" {
		t.Errorf("expected sha256, got %s", sbom.HashAlgorithm)
	}
	if len(sbom.CoreFiles) == 0 {
		t.Error("expected at least 1 core file")
	}
	t.Logf("Generated SBOM with %d core files, %d Go deps, %d Python deps",
		len(sbom.CoreFiles), len(sbom.Dependencies.Go), len(sbom.Dependencies.Python))
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	// Generate and save
	sbom, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := sbom.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, SBOMRegistry)); os.IsNotExist(err) {
		t.Fatal("SBOM file not written")
	}

	// Load and compare
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.SchemaVersion != sbom.SchemaVersion {
		t.Errorf("schema mismatch: %s vs %s", loaded.SchemaVersion, sbom.SchemaVersion)
	}
	if len(loaded.CoreFiles) != len(sbom.CoreFiles) {
		t.Errorf("core files count mismatch: %d vs %d", len(loaded.CoreFiles), len(sbom.CoreFiles))
	}
}

func TestVerify_Clean(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	sbom, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := sbom.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	result, err := Verify(dir)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid SBOM, got mismatches: %v", result.Mismatches)
	}
}

func TestVerify_Modified(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Test\n"), 0644)

	sbom, err := Generate(dir)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if err := sbom.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Modify a tracked file
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Modified\n"), 0644)

	result, err := Verify(dir)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid after modification")
	}
	t.Logf("Mismatches after modification: %v", result.Mismatches)
}

func TestVerify_MissingBaseline(t *testing.T) {
	dir := t.TempDir()
	_, err := Verify(dir)
	if err == nil {
		t.Error("expected error when baseline missing")
	}
}

func TestDiscoverGoDeps(t *testing.T) {
	dir := t.TempDir()
	goModContent := `module github.com/ovav/ovav

go 1.24

require (
	github.com/charmbracelet/bubbletea v1.3.4
	github.com/charmbracelet/lipgloss v1.2.1
	golang.org/x/crypto v0.42.0
)
`
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte(goModContent), 0644)

	deps := discoverGoDeps(dir)
	if len(deps) != 3 {
		t.Errorf("expected 3 Go deps, got %d", len(deps))
	}
	for _, d := range deps {
		t.Logf("  %s %s", d.Name, d.Version)
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		path     string
		excluded bool
	}{
		{".git/HEAD", true},
		{"node_modules/foo/index.js", true},
		{"__pycache__/module.pyc", true},
		{".ovav/vault/master.key", true},
		{".ovav/runtime/session.json", true},
		{".ovav/registry/sbom.json", true},
		{"go-runtime/internal/sbom/sbom.go", false},
		{"AGENTS.md", false},
		{"runtimes/opencode/agents/lead-thavren.md", false},
	}
	for _, tt := range tests {
		if got := isExcluded(tt.path); got != tt.excluded {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.path, got, tt.excluded)
		}
	}
}

func TestComputeRequirementsHash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "go-runtime"), 0755)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.mod"), []byte("module test\n"), 0644)
	os.WriteFile(filepath.Join(dir, "go-runtime", "go.sum"), []byte("hash\n"), 0644)
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("pyyaml==6.0\n"), 0644)

	hash := ComputeRequirementsHash(dir)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	// Same input should produce same hash
	hash2 := ComputeRequirementsHash(dir)
	if hash != hash2 {
		t.Error("hash should be deterministic")
	}
	t.Logf("Requirements hash: %s", hash[:16])
}
