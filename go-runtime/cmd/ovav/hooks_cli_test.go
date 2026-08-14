package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGitDir_DirectDotGit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveGitDir(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(root, ".git")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestResolveGitDir_WorktreePointer(t *testing.T) {
	root := t.TempDir()
	mainGit := t.TempDir()
	worktreesDir := filepath.Join(mainGit, "worktrees", "test-wt")
	if err := os.MkdirAll(worktreesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a .git file pointing to the worktree's gitdir
	gitfile := filepath.Join(root, ".git")
	content := "gitdir: " + worktreesDir + "\n"
	if err := os.WriteFile(gitfile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveGitDir(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != mainGit {
		t.Fatalf("expected %q, got %q", mainGit, got)
	}
}

func TestResolveGitDir_MalformedFile(t *testing.T) {
	root := t.TempDir()
	gitfile := filepath.Join(root, ".git")
	if err := os.WriteFile(gitfile, []byte("not gitdir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveGitDir(root); err == nil {
		t.Fatal("expected error for malformed .git file")
	}
}

func TestResolveGitDir_NotAGitRepo(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveGitDir(root); err == nil {
		t.Fatal("expected error for missing .git")
	}
}

func TestCopyFileIntegration(t *testing.T) {
	// Verifies that copyFile (defined in convert_cli.go) works for our use case
	src := t.TempDir() + "/src.txt"
	dst := t.TempDir() + "/dst.txt"
	content := []byte("hello world")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", content, got)
	}
}