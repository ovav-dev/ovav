package gitflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrePushScan_NoiseFiles(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "test@test.com")
	runGit(dir, "config", "user.name", "test")

	// Create a noise file (untracked)
	os.MkdirAll(filepath.Join(dir, "__pycache__"), 0755)
	os.WriteFile(filepath.Join(dir, "__pycache__", "test.pyc"), []byte("cache"), 0644)

	result, err := PrePushScan(dir)
	if err != nil {
		t.Fatalf("PrePushScan failed: %v", err)
	}

	if len(result.NoiseFindings) == 0 {
		t.Error("expected noise findings for __pycache__")
	}
	if result.Clean {
		t.Error("expected Clean=false due to noise file")
	}

	// Verify __pycache__/ was detected
	found := false
	for _, nf := range result.NoiseFindings {
		if filepath.Base(filepath.Dir(nf.File)) == "__pycache__" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected __pycache__ in noise findings, got: %+v", result.NoiseFindings)
	}
}

func TestPrePushScan_Secrets(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "test@test.com")
	runGit(dir, "config", "user.name", "test")

	// Need at least one commit for git diff HEAD to work
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	runGit(dir, "add", "README.md")
	runGit(dir, "commit", "-m", "initial")

	// Create a file with a fake secret pattern
	secretFile := filepath.Join(dir, "config.go")
	os.WriteFile(secretFile, []byte("const apiKey = \"sk-12345678901234567890\""), 0644)
	runGit(dir, "add", "config.go")

	result, err := PrePushScan(dir)
	if err != nil {
		t.Fatalf("PrePushScan failed: %v", err)
	}

	if result.BlockingIssues == 0 {
		t.Error("expected blocking issues for fake OpenAI key pattern")
	}
}

func TestPrePushScan_EmptyFiles(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "test@test.com")
	runGit(dir, "config", "user.name", "test")

	// Create an empty untracked file
	os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(""), 0644)

	result, err := PrePushScan(dir)
	if err != nil {
		t.Fatalf("PrePushScan failed: %v", err)
	}

	if len(result.EmptyFindings) == 0 {
		t.Error("expected empty file finding")
	}
}

func TestPrePushScan_Clean(t *testing.T) {
	dir := t.TempDir()
	runGit(dir, "init")
	runGit(dir, "config", "user.email", "test@test.com")
	runGit(dir, "config", "user.name", "test")

	result, err := PrePushScan(dir)
	if err != nil {
		t.Fatalf("PrePushScan failed: %v", err)
	}

	if !result.Clean {
		t.Errorf("expected clean repo, got %d issues", result.TotalIssues)
	}
	if result.TotalIssues != 0 {
		t.Errorf("expected 0 issues in empty repo, got %d", result.TotalIssues)
	}
}
