package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernedPushAuditPathExternalIsOutsideConsumer(t *testing.T) {
	root := t.TempDir()
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
