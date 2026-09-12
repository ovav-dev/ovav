package consumers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExternalConsumerByGitIdentity(t *testing.T) {
	root, worktree := setupConsumerRepo(t)
	other := setupGitRepo(t)
	escape := filepath.Join(root, ".ovav", "worktrees", "..", "..", "outside")
	if err := os.MkdirAll(escape, 0o755); err != nil {
		t.Fatal(err)
	}

	registryPath := filepath.Join(t.TempDir(), "consumers.yaml")
	registry := fmt.Sprintf("schema: %s\nconsumers:\n  - id: fixture\n    root_path: %s\n    state: registered\n", RegistrySchema, root)
	if err := os.WriteFile(registryPath, []byte(registry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OVAV_CONSUMER_REGISTRY", registryPath)

	tests := []struct {
		name       string
		path       string
		registered bool
	}{
		{name: "registered root path", path: root, registered: true},
		{name: "registered worktree path", path: worktree, registered: true},
		{name: "another repository", path: other, registered: false},
		{name: "path escape", path: escape, registered: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Resolve(tt.path)
			if profile.Registered != tt.registered {
				t.Fatalf("registered = %v, want %v (err=%v)", profile.Registered, tt.registered, profile.Err)
			}
			if !tt.registered {
				if profile.Active() {
					t.Fatal("unregistered path was active")
				}
				return
			}
			if got := profile.AuthorityRoot(); got != root {
				t.Fatalf("authority root = %q, want %q", got, root)
			}
			if profile.StateDir() == "" || strings.Contains(profile.StateDir(), filepath.Base(worktree)) {
				t.Fatalf("central state resolved from temporary worktree: %q", profile.StateDir())
			}
		})
	}
}

func setupConsumerRepo(t *testing.T) (string, string) {
	t.Helper()
	root := setupGitRepo(t)
	worktree := filepath.Join(root, ".ovav", "worktrees", "candidate")
	runConsumerGit(t, root, "worktree", "add", "-q", worktree, "-b", "feature/candidate")
	return root, worktree
}

func setupGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runConsumerGit(t, root, "init", "-q")
	runConsumerGit(t, root, "config", "user.email", "test@ovav.dev")
	runConsumerGit(t, root, "config", "user.name", "OVAV Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runConsumerGit(t, root, "add", "README.md")
	runConsumerGit(t, root, "commit", "-q", "-m", "fixture")
	return root
}

func runConsumerGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
