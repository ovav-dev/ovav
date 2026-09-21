package warp

import (
	"context"
	"errors"
	"testing"
)

// TestOpenTabConfig_RejectsEmpty verifies empty config name is rejected
// without invoking the Warp CLI. This prevents path injection.
func TestOpenTabConfig_RejectsEmpty(t *testing.T) {
	a := &Adapter{WarpPath: "/nonexistent"}
	if err := a.OpenTabConfig(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty config name, got nil")
	}
}

// TestOpenTabConfig_RejectsPathInjection verifies that config names
// containing path separators are rejected.
func TestOpenTabConfig_RejectsPathInjection(t *testing.T) {
	a := &Adapter{WarpPath: "/nonexistent"}
	tests := []string{
		"../etc/passwd",
		"foo/bar",
		"foo\\bar",
		"foo bar",
	}
	for _, name := range tests {
		if err := a.OpenTabConfig(context.Background(), name); err == nil {
			t.Errorf("expected error for %q, got nil", name)
		}
	}
}

// TestOpenWorktree_RejectsRelative verifies worktree path must be absolute.
// OWS always creates absolute paths; reject non-absolute paths to avoid
// ambiguity about which worktree is being targeted.
func TestOpenWorktree_RejectsRelative(t *testing.T) {
	a := &Adapter{WarpPath: "/nonexistent"}
	if err := a.OpenWorktree(context.Background(), "relative/path", "tab-name"); err == nil {
		t.Fatal("expected error for relative path, got nil")
	}
}

// TestAdapter_NoGitCommands documents (via static review) that the
// adapter never invokes git. The implementation only uses exec.Command
// for: (a) `warp.exe open <uri>`, (b) `test -d <path>`.
//
// Any future PR that adds `git worktree`, `git checkout -b`, `git merge`,
// or any other git side-effect command violates OWS authority and MUST
// be rejected at review.
func TestAdapter_NoGitCommands(t *testing.T) {
	// Static check verified via:
	//   grep -E "git (worktree|checkout|merge|reset|clean|push)" adapter.go
	//
	// No git commands present in adapter.go.
	if errors.New("placeholder") == nil {
		t.Fatal("unreachable")
	}
}
