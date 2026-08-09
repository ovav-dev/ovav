package ows

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ── F3: Conflict Prediction Tests ──────────────────────────────────────────

func setupConflictTestRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	// Init git repo
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@ovav.dev")
	runGitCmd(t, dir, "config", "user.name", "OWS Test")

	// Create base commit (git init default branch → rename to main)
	writeFile(t, dir, "shared.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	runGitCmd(t, dir, "add", "shared.go")
	runGitCmd(t, dir, "commit", "-m", "initial")
	runGitCmd(t, dir, "branch", "-m", "main") // rename default branch → main

	// Create develop branch
	runGitCmd(t, dir, "checkout", "-b", "develop")

	// Modify shared.go on develop (simulate ongoing work)
	writeFile(t, dir, "shared.go", "package main\n\nfunc main() {\n\tprintln(\"hello from develop\")\n}\n")
	runGitCmd(t, dir, "add", "shared.go")
	runGitCmd(t, dir, "commit", "-m", "develop change in shared.go")

	// Create feature branch from before the develop change
	runGitCmd(t, dir, "checkout", "-b", "task/test-feature", "main")
	writeFile(t, dir, "shared.go", "package main\n\nfunc main() {\n\tprintln(\"hello from feature\")\n}\n")
	runGitCmd(t, dir, "add", "shared.go")
	runGitCmd(t, dir, "commit", "-m", "feature change in shared.go")

	// Create a non-conflicting file on feature branch
	writeFile(t, dir, "new_file.go", "package main\n\nfunc newFunc() {}\n")
	runGitCmd(t, dir, "add", "new_file.go")
	runGitCmd(t, dir, "commit", "-m", "add new_file.go")

	// Go back to develop
	runGitCmd(t, dir, "checkout", "develop")

	cleanup := func() {
		// temp dir auto-cleans
	}
	return dir, cleanup
}

func TestPredictConflicts_BothModifySameFile(t *testing.T) {
	repo, cleanup := setupConflictTestRepo(t)
	defer cleanup()

	matrix, err := PredictConflicts(repo, "task/test-feature", "develop")
	if err != nil {
		t.Fatalf("PredictConflicts: %v", err)
	}

	if matrix.TotalFiles < 1 {
		t.Errorf("expected at least 1 file, got %d", matrix.TotalFiles)
	}

	// shared.go should be a conflict (modified on both sides with overlap)
	foundShared := false
	for _, f := range matrix.Files {
		if f.FilePath == "shared.go" {
			foundShared = true
			if f.Status != "conflict" {
				t.Errorf("shared.go should be conflict, got %s", f.Status)
			}
			if f.CanAutoMerge {
				t.Error("shared.go should NOT be auto-mergeable")
			}
		}
	}
	if !foundShared {
		t.Error("shared.go not found in conflict matrix")
	}

	// new_file.go should be safe (only on feature side)
	foundNew := false
	for _, f := range matrix.Files {
		if f.FilePath == "new_file.go" {
			foundNew = true
			if f.Status != "source_only" {
				t.Errorf("new_file.go should be source_only, got %s", f.Status)
			}
			if !f.CanAutoMerge {
				t.Error("new_file.go should be auto-mergeable")
			}
		}
	}
	if !foundNew {
		t.Error("new_file.go not found in conflict matrix")
	}

	if matrix.ConflictFiles == 0 {
		t.Error("expected at least 1 conflict")
	}

	t.Logf("Summary: %s", matrix.Summary())
}

func TestPredictConflicts_NoOverlap(t *testing.T) {
	repo, cleanup := setupConflictTestRepo(t)
	defer cleanup()

	// Modify a DIFFERENT region of shared.go on feature
	runGitCmd(t, repo, "checkout", "task/test-feature")
	content := "package main\n\nfunc main() {\n\tprintln(\"hello from feature\")\n}\n\n// NEW: added at bottom\nfunc helper() bool {\n\treturn true\n}\n"
	writeFile(t, repo, "shared.go", content)
	runGitCmd(t, repo, "add", "shared.go")
	runGitCmd(t, repo, "commit", "-m", "add helper at bottom")

	// On develop, modify the TOP of shared.go
	runGitCmd(t, repo, "checkout", "develop")
	content2 := "// DEVELOP HEADER\npackage main\n\nfunc main() {\n\tprintln(\"hello from develop\")\n}\n"
	writeFile(t, repo, "shared.go", content2)
	runGitCmd(t, repo, "add", "shared.go")
	runGitCmd(t, repo, "commit", "-m", "add header comment on develop")

	matrix, err := PredictConflicts(repo, "task/test-feature", "develop")
	if err != nil {
		t.Fatalf("PredictConflicts: %v", err)
	}

	for _, f := range matrix.Files {
		if f.FilePath == "shared.go" {
			t.Logf("shared.go: status=%s overlaps=%d", f.Status, len(f.OverlapRanges))
			if f.Status == "conflict" {
				t.Logf("  source ranges: %+v", f.SourceRanges)
				t.Logf("  target ranges: %+v", f.TargetRanges)
				t.Logf("  overlap ranges: %+v", f.OverlapRanges)
			}
		}
	}
}

func TestPredictConflicts_FileOnlyOnOneSide(t *testing.T) {
	repo, cleanup := setupConflictTestRepo(t)
	defer cleanup()

	// Add file only on develop
	runGitCmd(t, repo, "checkout", "develop")
	writeFile(t, repo, "develop_only.go", "package main\n")
	runGitCmd(t, repo, "add", "develop_only.go")
	runGitCmd(t, repo, "commit", "-m", "add develop_only.go")

	matrix, err := PredictConflicts(repo, "task/test-feature", "develop")
	if err != nil {
		t.Fatalf("PredictConflicts: %v", err)
	}

	for _, f := range matrix.Files {
		if f.FilePath == "develop_only.go" {
			if f.Status != "target_only" {
				t.Errorf("develop_only.go should be target_only, got %s", f.Status)
			}
			if !f.CanAutoMerge {
				t.Error("develop_only.go should be auto-mergeable")
			}
		}
	}
}

func TestPredictConflicts_Summary(t *testing.T) {
	repo, cleanup := setupConflictTestRepo(t)
	defer cleanup()

	matrix, err := PredictConflicts(repo, "task/test-feature", "develop")
	if err != nil {
		t.Fatalf("PredictConflicts: %v", err)
	}

	summary := matrix.Summary()
	if summary == "" {
		t.Error("summary should not be empty")
	}
	t.Logf("Summary: %s", summary)

	conflicts := matrix.Conflicts()
	if len(conflicts) != matrix.ConflictFiles {
		t.Errorf("Conflicts() count %d != ConflictFiles %d", len(conflicts), matrix.ConflictFiles)
	}
}

func TestPredictConflicts_InvalidRefs(t *testing.T) {
	repo, cleanup := setupConflictTestRepo(t)
	defer cleanup()

	_, err := PredictConflicts(repo, "nonexistent", "develop")
	if err == nil {
		t.Fatal("expected error for nonexistent ref")
	}
}

// ── Unit Tests: merge-tree parser ────────────────────────────────────────────

func TestDetectSectionType(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"changed in both", "both"},
		{"added in both", "both"},
		{"added in local", "source_only"},
		{"changed in local", "source_only"},
		{"added in remote", "target_only"},
		{"changed in remote", "target_only"},
		{"removed in remote", "removed"},
		{"removed in local", "removed"},
		{"removed in both", "removed"},
		{"@@ -1,5 +1,7 @@", ""},
		{"some random text", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := detectSectionType(tt.line)
		if got != tt.want {
			t.Errorf("detectSectionType(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"  base   100644 abc123 internal/auth/handler.go", "internal/auth/handler.go"},
		{"  our    100644 def456 path/to/file.go", "path/to/file.go"},
		{"  their  100644 ghi789 file with spaces.go", "file with spaces.go"},
		{"  base   100644 0000000000000000000000000000000000000000 new.go", "new.go"},
		{"short", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractPath(tt.line)
		if got != tt.want {
			t.Errorf("extractPath(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestParseHunkRange(t *testing.T) {
	tests := []struct {
		header string
		start  int
		end    int
		isNil  bool
	}{
		{"@@ -1,3 +1,5 @@", 1, 5, false},
		{"@@ -10,0 +20,10 @@", 20, 29, false},
		{"@@ -5 +7,1 @@", 7, 7, false},
		{"no plus here", 0, 0, true},
	}

	for _, tt := range tests {
		rng := parseHunkRange(tt.header)
		if tt.isNil {
			if rng != nil {
				t.Errorf("parseHunkRange(%q) should return nil", tt.header)
			}
			continue
		}
		if rng == nil {
			t.Errorf("parseHunkRange(%q) returned nil", tt.header)
			continue
		}
		if rng.Start != tt.start || rng.End != tt.end {
			t.Errorf("parseHunkRange(%q) = {%d,%d}, want {%d,%d}",
				tt.header, rng.Start, rng.End, tt.start, tt.end)
		}
	}
}

func TestParseMergeTreeOutput_Conflict(t *testing.T) {
	output := `abcdef1234567890

changed in both
  base   100644 aaa111 internal/auth/handler.go
  our    100644 bbb222 internal/auth/handler.go
  their  100644 ccc333 internal/auth/handler.go
@@ -10,5 +10,11 @@
 context
+<<<<<<< .our
+our code
+=======
+their code
+>>>>>>> .their
 more context
added in local
  base   100644 0000000000000000000000000000000000000000 new_file.go
  our    100644 ddd444 new_file.go
  their  100644 0000000000000000000000000000000000000000 new_file.go`

	preds := parseMergeTreeOutput(output)

	if len(preds) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(preds))
	}

	// First file: conflict
	if preds[0].FilePath != "internal/auth/handler.go" {
		t.Errorf("file[0] path = %q, want internal/auth/handler.go", preds[0].FilePath)
	}
	if preds[0].Status != "conflict" {
		t.Errorf("file[0] status = %q, want conflict", preds[0].Status)
	}
	if preds[0].CanAutoMerge {
		t.Error("file[0] should NOT be auto-mergeable")
	}

	// Second file: source_only
	if preds[1].FilePath != "new_file.go" {
		t.Errorf("file[1] path = %q, want new_file.go", preds[1].FilePath)
	}
	if preds[1].Status != "source_only" {
		t.Errorf("file[1] status = %q, want source_only", preds[1].Status)
	}
	if !preds[1].CanAutoMerge {
		t.Error("file[1] should be auto-mergeable")
	}
}

func TestParseMergeTreeOutput_SafeMerge(t *testing.T) {
	output := `abcdef1234567890

changed in both
  base   100644 aaa111 safe.go
  our    100644 bbb222 safe.go
  their  100644 ccc333 safe.go
@@ -1,5 +1,7 @@
 context
+merged addition
 more context`

	preds := parseMergeTreeOutput(output)

	if len(preds) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(preds))
	}
	if preds[0].Status != "safe" {
		t.Errorf("status = %q, want safe", preds[0].Status)
	}
	if !preds[0].CanAutoMerge {
		t.Error("should be auto-mergeable")
	}
}

func TestParseMergeTreeOutput_TargetOnly(t *testing.T) {
	output := `abcdef1234567890

added in remote
  base   100644 0000000000000000000000000000000000000000 remote_only.go
  our    100644 0000000000000000000000000000000000000000 remote_only.go
  their  100644 eee555 remote_only.go`

	preds := parseMergeTreeOutput(output)

	if len(preds) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(preds))
	}
	if preds[0].FilePath != "remote_only.go" {
		t.Errorf("path = %q, want remote_only.go", preds[0].FilePath)
	}
	if preds[0].Status != "target_only" {
		t.Errorf("status = %q, want target_only", preds[0].Status)
	}
	if !preds[0].CanAutoMerge {
		t.Error("should be auto-mergeable")
	}
}

func TestParseMergeTreeOutput_Empty(t *testing.T) {
	preds := parseMergeTreeOutput("")
	if len(preds) != 0 {
		t.Errorf("expected 0 predictions for empty output, got %d", len(preds))
	}

	preds2 := parseMergeTreeOutput("abcdef1234567890")
	if len(preds2) != 0 {
		t.Errorf("expected 0 predictions for OID-only output, got %d", len(preds2))
	}
}

func TestParseMergeTreeOutput_MultipleSections(t *testing.T) {
	output := `abcdef1234567890

changed in both
  base   100644 aaa111 conflict.go
  our    100644 bbb222 conflict.go
  their  100644 ccc333 conflict.go
@@ -5,3 +5,9 @@
 context
+<<<<<<< .our
+ours
+=======
+theirs
+>>>>>>> .their
changed in both
  base   100644 ddd444 safe.go
  our    100644 eee555 safe.go
  their  100644 fff666 safe.go
@@ -1,3 +1,5 @@
 context
+auto-merged line
added in local
  base   100644 0000000000000000000000000000000000000000 feature_only.go
  our    100644 ggg777 feature_only.go
  their  100644 0000000000000000000000000000000000000000 feature_only.go`

	preds := parseMergeTreeOutput(output)

	if len(preds) != 3 {
		t.Fatalf("expected 3 predictions, got %d", len(preds))
	}

	// conflict.go → conflict
	if preds[0].Status != "conflict" {
		t.Errorf("conflict.go status = %q, want conflict", preds[0].Status)
	}
	// safe.go → safe (changed in both but no markers)
	if preds[1].Status != "safe" {
		t.Errorf("safe.go status = %q, want safe", preds[1].Status)
	}
	// feature_only.go → source_only
	if preds[2].Status != "source_only" {
		t.Errorf("feature_only.go status = %q, want source_only", preds[2].Status)
	}
}

// ── Test Helpers ──────────────────────────────────────────────────────────

func runGitCmd(tb testing.TB, dir string, args ...string) {
	tb.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("git %s failed: %v\n%s", args, err, out)
	}
}

func writeFile(tb testing.TB, dir, name, content string) {
	tb.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		tb.Fatalf("write %s: %v", name, err)
	}
}
