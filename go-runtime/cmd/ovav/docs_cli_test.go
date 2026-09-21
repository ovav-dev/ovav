package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdDocs_Help(t *testing.T) {
	code := cmdDocs([]string{"help"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestCmdDocs_Unknown(t *testing.T) {
	code := cmdDocs([]string{"unknown"})
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestCmdDocs_NoArgs(t *testing.T) {
	// No args → shows help (returns 0)
	code := cmdDocs([]string{})
	if code != 0 {
		t.Fatalf("expected 0 (help), got %d", code)
	}
}

func TestCmdDocs_GenerateNoRepo(t *testing.T) {
	root := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runDocsGenerate([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (no OVAV repo), got %d", code)
	}
}

func TestCmdDocs_GenerateAll(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runDocsGenerate([]string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}

	// Verify files were created
	outDir := filepath.Join(root, "docs", "auto-generated")
	for _, name := range []string{"validators.md", "commands.md", "drift-targets.md", "auto-fix.md"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestCmdDocs_GenerateTarget(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runDocsGenerate([]string{"--target=validators"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}

	// Only validators.md should be created
	outDir := filepath.Join(root, "docs", "auto-generated")
	if _, err := os.Stat(filepath.Join(outDir, "validators.md")); err != nil {
		t.Error("validators.md not created")
	}
	if _, err := os.Stat(filepath.Join(outDir, "commands.md")); err == nil {
		t.Error("commands.md should not be created")
	}
}

func TestCmdDocs_GenerateDryRun(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	code := runDocsGenerate([]string{"--dry-run"})
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}

	// No files should be created in dry-run mode
	outDir := filepath.Join(root, "docs", "auto-generated")
	if _, err := os.Stat(outDir); err == nil {
		t.Error("outDir should not be created in dry-run mode")
	}
}

func TestCmdDocs_Check(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	// First generate, then check should pass
	runDocsGenerate([]string{})
	code := runDocsCheck([]string{})
	if code != 0 {
		t.Fatalf("expected 0 (docs current), got %d", code)
	}
}

func TestCmdDocs_CheckStale(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	// Generate, then manually corrupt one file
	runDocsGenerate([]string{})
	outDir := filepath.Join(root, "docs", "auto-generated")
	os.WriteFile(filepath.Join(outDir, "validators.md"), []byte("corrupted"), 0o644)

	code := runDocsCheck([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (stale), got %d", code)
	}
}

func TestCmdDocs_CheckMissing(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	old, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(old)

	// No docs generated yet
	code := runDocsCheck([]string{})
	if code != 1 {
		t.Fatalf("expected 1 (missing), got %d", code)
	}
}

func TestCategoryFromID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"agent_permission", "Agent Governance"},
		{"supply_chain", "Supply Chain"},
		{"context_firewall", "Context Economy"},
		{"bash_readline_bindings", "Workstation"},
		{"zero_trust", "Security"},
		{"loop_guard", "Orchestration"},
		{"random_validator", "General"},
	}
	for _, tc := range tests {
		got := categoryFromID(tc.id)
		if got != tc.want {
			t.Errorf("categoryFromID(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestGenerateValidatorsContent_Deterministic(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".ovav", "plan"), 0o755)
	os.WriteFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"),
		[]byte("# test\ncanonical: test\n"), 0o644)

	// Generate twice — must produce identical output
	first, err := generateValidatorsContent(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateValidatorsContent(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("validator doc generation is non-deterministic")
	}
}
