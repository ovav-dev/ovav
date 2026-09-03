package validators

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIntegrityBaselineFresh_MissingBaseline(t *testing.T) {
	v := NewIntegrityBaselineFresh()
	root := t.TempDir()
	res := v.Validate(context.Background(), root)
	if len(res.Issues) == 0 {
		t.Fatalf("expected issue for missing baseline, got: %+v", res)
	}
}

func TestIntegrityBaselineFresh_FreshBaseline(t *testing.T) {
	v := NewIntegrityBaselineFresh().WithMaxAge(7 * 24 * time.Hour)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a fresh baseline.json
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), baseline, 0o644); err != nil {
		t.Fatal(err)
	}
	res := v.Validate(context.Background(), root)
	if res.Status != "pass" {
		t.Fatalf("fresh baseline should pass, got: %+v", res)
	}
}

func TestIntegrityBaselineFresh_StaleBaseline_GateModeFails(t *testing.T) {
	v := NewIntegrityBaselineFresh(ValidationGate).WithMaxAge(1 * time.Hour)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a stale baseline (2 hours old via mtime)
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	path := filepath.Join(baseDir, "baseline.json")
	if err := os.WriteFile(path, baseline, 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("stale baseline in gate mode should FAIL, got: %+v", res)
	}
}

func TestIntegrityBaselineFresh_StaleBaseline_DevModeWarns(t *testing.T) {
	v := NewIntegrityBaselineFresh(ValidationDeveloper).WithMaxAge(1 * time.Hour)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	path := filepath.Join(baseDir, "baseline.json")
	if err := os.WriteFile(path, baseline, 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
	res := v.Validate(context.Background(), root)
	if res.Status != "warn" {
		t.Fatalf("stale baseline in dev mode should WARN, got: %+v", res)
	}
}

func TestIntegrityBaselineFresh_PinnedRequiredMissing(t *testing.T) {
	v := NewIntegrityBaselineFresh().WithPinnedRequired()
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	if err := os.WriteFile(filepath.Join(baseDir, "baseline.json"), baseline, 0o644); err != nil {
		t.Fatal(err)
	}
	res := v.Validate(context.Background(), root)
	if res.Status != "warn" {
		t.Fatalf("missing pinned baseline should WARN, got: %+v", res)
	}
	if len(res.Issues) == 0 {
		t.Fatal("expected issue about missing pinned baseline")
	}
}
