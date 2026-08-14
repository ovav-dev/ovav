package validators

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPinnedBaselineDrift_NoPinnedYet(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	current := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"abc123"}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewPinnedBaselineDrift()
	res := v.Validate(context.Background(), root)
	if res.Status != "warn" {
		t.Fatalf("missing pinned should WARN, got: %+v", res)
	}
}

func TestPinnedBaselineDrift_Match(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	current := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"abc123"}}`)
	pinned := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"abc123"}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewPinnedBaselineDrift()
	res := v.Validate(context.Background(), root)
	if res.Status != "pass" {
		t.Fatalf("matching pinned should PASS, got: %+v", res)
	}
}

func TestPinnedBaselineDrift_MismatchFails(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	current := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"NEWHASH"}}`)
	pinned := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"OLDHASH"}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewPinnedBaselineDrift(ValidationGate)
	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("drift in gate mode should FAIL, got: %+v", res)
	}
	if len(res.Issues) == 0 {
		t.Fatal("expected drift failure")
	}
}

func TestPinnedBaselineDrift_NewSurfaceWarns(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	current := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"abc","NEW.md":"xyz"}}`)
	pinned := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{"AGENTS.md":"abc"}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644); err != nil {
		t.Fatal(err)
	}
	v := NewPinnedBaselineDrift()
	res := v.Validate(context.Background(), root)
	if res.Status != "warn" {
		t.Fatalf("new surface not pinned should WARN, got: %+v", res)
	}
}