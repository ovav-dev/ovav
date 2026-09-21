package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ovav/ovav/internal/sbom"
)

func TestSBOMGenerateDryRunAndCheckAreReadOnly(t *testing.T) {
	root := t.TempDir()
	runSBOMGit(t, root, "init", "-q")
	runSBOMGit(t, root, "config", "user.email", "test@ovav.dev")
	runSBOMGit(t, root, "config", "user.name", "OVAV Test")
	writeSBOMCommandFile(t, root, "go-runtime/go.mod", "module test\n")
	writeSBOMCommandFile(t, root, "go-runtime/go.sum", "sum\n")
	runSBOMGit(t, root, "add", "go-runtime/go.mod", "go-runtime/go.sum")
	runSBOMGit(t, root, "commit", "-q", "-m", "baseline")
	t.Chdir(root)

	if code := sbomGenerate([]string{"--dry-run"}); code != 0 {
		t.Fatalf("dry-run exit = %d, want 0", code)
	}
	registry := filepath.Join(root, ".ovav", "registry", "sbom.json")
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote registry: %v", err)
	}
	if code := sbomGenerate([]string{"--check"}); code != 1 {
		t.Fatalf("check exit = %d, want 1 for missing/outdated baseline", code)
	}
	if _, err := os.Stat(registry); !os.IsNotExist(err) {
		t.Fatalf("check wrote registry: %v", err)
	}
}

func TestSBOMVerifyGateFailsSensitiveCandidateWhileInspectWarns(t *testing.T) {
	root := t.TempDir()
	runSBOMGit(t, root, "init", "-q")
	runSBOMGit(t, root, "config", "user.email", "test@ovav.dev")
	runSBOMGit(t, root, "config", "user.name", "OVAV Test")
	writeSBOMCommandFile(t, root, "go-runtime/go.mod", "module test\n")
	writeSBOMCommandFile(t, root, "go-runtime/go.sum", "sum\n")
	runSBOMGit(t, root, "add", "go-runtime/go.mod", "go-runtime/go.sum")
	runSBOMGit(t, root, "commit", "-q", "-m", "baseline")
	baseline, err := sbom.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	writeSBOMCommandFile(t, root, "go-runtime/go.mod", "module changed\n")
	t.Chdir(root)

	if code := sbomVerify([]string{"--inspect"}); code != 0 {
		t.Fatalf("inspect exit = %d, want 0", code)
	}
	if code := sbomVerify([]string{"--gate"}); code != 1 {
		t.Fatalf("gate exit = %d, want 1", code)
	}
}

func runSBOMGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git failed: %v\n%s", err, output)
	}
}

func writeSBOMCommandFile(t *testing.T, root, path, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
