package validators

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalRegistryUsesCentralBaselinesAndNoOVAVSurface(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	runGitTest(t, root, "checkout", "-b", "feature/external")
	runGitTest(t, root, "remote", "add", "origin", "https://github.com/example/external.git")
	writeTestFile(t, root, "package.json", `{"name":"external"}`)
	writeTestFile(t, root, ".gitignore", ".env\n")
	runGitTest(t, root, "add", "package.json", ".gitignore")
	runGitTest(t, root, "commit", "-m", "external baseline fixture")

	central := t.TempDir()
	registryPath := filepath.Join(central, ".ovav", "registry", "consumers.yaml")
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := "schema: ovav.consumer_registry.v1\nconsumers:\n  - id: external-fixture\n    root_path: " + root + "\n    state: registered\n"
	if err := os.WriteFile(registryPath, []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OVAV_CONSUMER_REGISTRY", registryPath)

	selected := DefaultRegistryForRoot(root, ValidationGate)
	for _, validator := range selected.All() {
		if strings.Contains(validator.Description(), "OVAV monorepo") {
			t.Fatalf("internal validator selected for external project: %s", validator.ID())
		}
	}
	if got := len(selected.All()); got != 8 {
		t.Fatalf("external registry size = %d, want 8", got)
	}

	if err := WriteExternalBaselines(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".ovav", "registry", "consumers")); !os.IsNotExist(err) {
		t.Fatal("external baseline leaked into consumer repository")
	}
	if result := newExternalSupplyChain(ValidationGate).Validate(context.Background(), root); result.Status != "pass" {
		t.Fatalf("external supply chain status = %s: %v", result.Status, result.Issues)
	}
	if result := newExternalRuntimeIntegrity(ValidationGate).Validate(context.Background(), root); result.Status != "pass" {
		t.Fatalf("external runtime integrity status = %s: %v", result.Status, result.Issues)
	}
	if result := newExternalGitPush().Validate(context.Background(), root); result.Status != "pass" {
		t.Fatalf("external git push status = %s: %v", result.Status, result.Issues)
	}
	writeTestFile(t, root, ".env", `PASSWORD="fixture-secret"`)
	if result := NewSecretsHygiene().Validate(context.Background(), root); result.Status != "fail" {
		t.Fatalf("external secrets status = %s, want fail", result.Status)
	}
	if _, err := os.Stat(filepath.Join(root, ".ovav", "alerts")); !os.IsNotExist(err) {
		t.Fatal("external secret scan wrote alert state into consumer repository")
	}
}

func TestExternalBaselinesPreserveUnicodePaths(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	path := "docs/SPEC-REDISEÑO-LOGIN-DASHBOARD.md"
	contents := "# Login dashboard\n"
	writeTestFile(t, root, path, contents)
	runGitTest(t, root, "config", "core.quotePath", "true")
	runGitTest(t, root, "add", path)
	runGitTest(t, root, "commit", "-m", "unicode path baseline fixture")

	paths, commit, err := externalHeadFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, gotPath := range paths {
		if gotPath == path {
			found = true
			break
		}
	}
	if !found || commit == "" {
		t.Fatalf("HEAD listing lost exact Unicode path %q: paths=%v commit=%q", path, paths, commit)
	}

	baseline, err := planExternalSBOM(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := baseline.Files[path]; !ok || got == "" {
		t.Fatalf("SBOM missing exact Unicode path %q: %v", path, baseline.Files)
	}

	integrity, err := planExternalIntegrity(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := integrity.Files[path]; ok {
		t.Fatal("documentation path unexpectedly included in integrity security surface")
	}
	data, err := gitBlob(root, path)
	if err != nil {
		t.Fatalf("read Unicode HEAD path: %v", err)
	}
	if string(data) != contents {
		t.Fatalf("Unicode HEAD content = %q, want %q", data, contents)
	}
}
