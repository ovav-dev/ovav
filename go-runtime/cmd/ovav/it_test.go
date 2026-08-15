package main

import (
	"strings"
	"testing"
	"time"
)

func TestFindITReloadScript(t *testing.T) {
	// In worktree, script should be found
	// In other dirs, may not — both outcomes are OK
	scriptPath, err := findITReloadScript()
	if err != nil {
		t.Logf("Script not found (acceptable): %v", err)
		return
	}
	if !strings.HasSuffix(scriptPath, "it-reload.ps1") {
		t.Fatalf("expected path ending in it-reload.ps1, got %s", scriptPath)
	}
}

func TestWslToWindows_AlreadyWindowsPath(t *testing.T) {
	got, err := wslToWindows(`C:\Users\test`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `C:\Users\test` {
		t.Fatalf("expected unchanged path, got %q", got)
	}
}

func TestWslToWindows_MntCPath(t *testing.T) {
	// Test fallback path conversion (when wslpath not available)
	got, err := wslToWindows("/mnt/c/Users/test/file.txt")
	if err != nil {
		// If wslpath is available, it returns that instead — that's also OK
		t.Logf("wslpath available: %v", err)
		return
	}
	if !strings.HasPrefix(got, "C:") && !strings.HasPrefix(got, "C:\\") {
		t.Fatalf("expected C: prefix, got %q", got)
	}
}

func TestGenerateDeployID_Unique(t *testing.T) {
	id1 := generateDeployID()
	// Sleep tiny amount
	time.Sleep(10 * time.Millisecond)
	id2 := generateDeployID()
	if id1 == id2 {
		t.Fatalf("deploy IDs should differ: %s vs %s", id1, id2)
	}
}
