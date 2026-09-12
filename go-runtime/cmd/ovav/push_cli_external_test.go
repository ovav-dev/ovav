package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernedPushAuditPathExternalIsOutsideConsumer(t *testing.T) {
	root := t.TempDir()
	initExternalAuditFixture(t, root)
	central := t.TempDir()
	configHome := t.TempDir()
	registry := filepath.Join(central, "consumers.yaml")
	contents := "schema: ovav.consumer_registry.v1\nconsumers:\n  - id: audit-fixture\n    root_path: " + root + "\n    state: registered\n"
	if err := os.WriteFile(registry, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OVAV_CONSUMER_REGISTRY", registry)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path, ok := governedPushAuditPath(root)
	if !ok {
		t.Fatal("external audit path was not resolved")
	}
	if strings.HasPrefix(path, root+string(filepath.Separator)) {
		t.Fatalf("audit path leaked into consumer checkout")
	}
	if !strings.Contains(filepath.ToSlash(path), "/ovav/consumer-runtime/audit-fixture/") {
		t.Fatalf("audit path is not central consumer runtime state")
	}
}

func initExternalAuditFixture(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@ovav.dev"},
		{"config", "user.name", "OVAV Test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "README.md"}, {"commit", "-q", "-m", "fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
}
