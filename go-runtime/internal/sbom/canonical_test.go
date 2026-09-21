package sbom

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyClassifiesObsoleteBaselineEntry(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"go-runtime/go.mod":     "module test\n",
		"go-runtime/go.sum":     "sum\n",
		"go-runtime/removed.go": "package removed\n",
	})
	baseline := generateAndSave(t, root)
	if _, ok := baseline.CoreFiles["go-runtime/removed.go"]; !ok {
		t.Fatal("precondition: deleted file missing from baseline")
	}
	runGit(t, root, "rm", "go-runtime/removed.go")
	runGit(t, root, "commit", "-m", "remove historical file")

	result, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, result.BaselineIssues, "OBSOLETE_BASELINE: go-runtime/removed.go")
	assertNotContains(t, result.WorktreeWarnings, "go-runtime/removed.go")
}

func TestVerifyClassifiesCurrentModifiedTrackedFileAsWorktreeDrift(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"go-runtime/go.mod": "module test\n",
		"go-runtime/go.sum": "sum\n",
	})
	generateAndSave(t, root)
	writeFile(t, root, "go-runtime/go.mod", "module feature\n")

	result, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	assertNotContains(t, result.BaselineIssues, "go-runtime/go.mod")
	assertContains(t, result.WorktreeWarnings, "WORKTREE_MODIFIED: go-runtime/go.mod")
}

func TestVerifyRejectsUnexpectedTrackedBinaryAndHashMismatch(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"go-runtime/go.mod": "module test\n",
		"go-runtime/go.sum": "sum\n",
	})
	baseline := generateAndSave(t, root)
	baseline.CoreFiles["go-runtime/go.mod"] = strings.Repeat("0", 64)
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "payload.exe", "malicious\n")
	runGit(t, root, "add", "payload.exe")
	runGit(t, root, "commit", "-m", "add unexpected binary")

	result, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, result.BaselineIssues, "HASH_MISMATCH: go-runtime/go.mod")
	assertContains(t, result.BaselineIssues, "UNEXPECTED_TRACKED: payload.exe")
}

func TestVerifyRejectsDependencyMetadataDrift(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"go-runtime/go.mod": "module test\n\nrequire (\n\texample.com/dep v1.0.0\n)\n",
		"go-runtime/go.sum": "sum\n",
	})
	baseline := generateAndSave(t, root)
	baseline.Dependencies.Go = nil
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}

	result, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, result.BaselineIssues, "DEPENDENCY_METADATA_MISMATCH")
}

func TestGenerateIsDeterministicAndUsesHEADWithExplicitExclusions(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"AGENTS.md":                               "canonical\n",
		"go-runtime/go.mod":                       "module test\n",
		"go-runtime/go.sum":                       "sum\n",
		".ovav/registry/sbom.json":                "historical self reference\n",
		".ovav/runtime/session.json":              "runtime\n",
		".ovav/artifacts/report.json":             "artifact\n",
		".ovav/history/old.json":                  "history\n",
		".ovav/worktrees/nested/artifact.go":      "package artifact\n",
		"go-runtime/build/generated-artifact.exe": "artifact\n",
	})
	writeFile(t, root, "AGENTS.md", "dirty worktree\n")

	first, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := first.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := second.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatal("generation for the same HEAD is not byte deterministic")
	}
	if first.Metadata.GitIdentity == "" || first.Metadata.GitIdentity == "unknown" {
		t.Fatalf("generation did not record HEAD: %+v", first.Metadata)
	}
	if len(first.Policy.ExcludedDirectories) == 0 || len(first.Policy.ExcludedFiles) == 0 {
		t.Fatalf("generation did not record explicit exclusions: %+v", first.Policy)
	}
	for _, excluded := range []string{
		SBOMRegistry,
		".ovav/runtime/session.json",
		".ovav/artifacts/report.json",
		".ovav/history/old.json",
		".ovav/worktrees/nested/artifact.go",
		"go-runtime/build/generated-artifact.exe",
	} {
		if _, ok := first.CoreFiles[excluded]; ok {
			t.Errorf("excluded file present in baseline: %s", excluded)
		}
	}
	headHash := hashBytes([]byte("canonical\n"))
	if got := first.CoreFiles["AGENTS.md"]; got != headHash {
		t.Fatalf("AGENTS.md hash = %s, want HEAD hash %s", got, headHash)
	}
}

func TestSavedBaselineSurvivesSBOMOnlyCommit(t *testing.T) {
	root := newGitRepo(t, map[string]string{
		"go-runtime/go.mod": "module test\n",
		"go-runtime/go.sum": "sum\n",
	})
	generateAndSave(t, root)
	runGit(t, root, "add", SBOMRegistry)
	runGit(t, root, "commit", "-m", "record sbom")

	result, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("SBOM-only commit invalidated baseline: %v", result.BaselineIssues)
	}
}

func newGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@ovav.dev")
	runGit(t, root, "config", "user.name", "OVAV Test")
	for path, content := range files {
		writeFile(t, root, path, content)
	}
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "-m", "baseline")
	return root
}

func generateAndSave(t *testing.T, root string) *SBOM {
	t.Helper()
	baseline, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := baseline.Save(root); err != nil {
		t.Fatal(err)
	}
	return baseline
}

func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, want) {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, values)
}

func assertNotContains(t *testing.T, values []string, unwanted string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, unwanted) {
			t.Fatalf("did not expect %q in %v", unwanted, values)
		}
	}
}
