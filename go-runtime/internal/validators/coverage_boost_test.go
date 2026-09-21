package validators

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── IntegrityBaselineFresh edge cases ──────────────────────────────────────

func TestIntegrityBaselineFresh_NonExistentRoot(t *testing.T) {
	v := NewIntegrityBaselineFresh(ValidationGate)
	res := v.Validate(context.Background(), "/nonexistent/path/should/not/exist")
	if res.Status != "fail" {
		t.Fatalf("expected fail for non-existent root in gate mode, got %s", res.Status)
	}
	if len(res.Issues) == 0 {
		t.Fatal("expected issue for non-existent baseline")
	}
}

func TestIntegrityBaselineFresh_PinnedRequiredAndPresent(t *testing.T) {
	v := NewIntegrityBaselineFresh(ValidationDeveloper).WithPinnedRequired()
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Both files present
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	pinned := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), baseline, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "pass" {
		t.Fatalf("expected pass with both files present, got %s", res.Status)
	}
}

func TestIntegrityBaselineFresh_PinnedRequiredGateMode(t *testing.T) {
	v := NewIntegrityBaselineFresh(ValidationGate).WithPinnedRequired()
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Only baseline, no pinned
	baseline := []byte(`{"schema":"ovav.runtime_integrity.v1","algorithm":"sha256","files":{}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), baseline, 0o644)

	res := v.Validate(context.Background(), root)
	// Pinned required + missing should WARN (not fail)
	if res.Status != "warn" {
		t.Fatalf("expected warn for missing pinned, got %s", res.Status)
	}
}

func TestIntegrityBaselineFresh_StaleAtBoundary(t *testing.T) {
	// Test exactly at the boundary (8 days old vs 7-day threshold)
	v := NewIntegrityBaselineFresh(ValidationDeveloper).WithMaxAge(7 * 24 * time.Hour)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	os.MkdirAll(baseDir, 0o755)

	baseline := []byte(`{}`)
	path := filepath.Join(baseDir, "baseline.json")
	os.WriteFile(path, baseline, 0o644)

	// 6 days old → pass
	sixDaysAgo := time.Now().Add(-6 * 24 * time.Hour)
	os.Chtimes(path, sixDaysAgo, sixDaysAgo)
	res := v.Validate(context.Background(), root)
	if res.Status != "pass" {
		t.Fatalf("6 days old should pass, got %s", res.Status)
	}

	// 8 days old → warn
	eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(path, eightDaysAgo, eightDaysAgo)
	res = v.Validate(context.Background(), root)
	if res.Status != "warn" {
		t.Fatalf("8 days old should warn, got %s", res.Status)
	}
}

// ── PinnedBaselineDrift edge cases ─────────────────────────────────────────

func TestPinnedBaselineDrift_CurrentUnreadable(t *testing.T) {
	v := NewPinnedBaselineDrift(ValidationGate)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Valid pinned but malformed current
	pinned := []byte(`{"schema":"v1","algorithm":"sha256","files":{"a":"b"}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), []byte("not json"), 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for malformed current, got %s", res.Status)
	}
}

func TestPinnedBaselineDrift_PinnedUnreadable(t *testing.T) {
	v := NewPinnedBaselineDrift(ValidationGate)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Valid current but malformed pinned
	current := []byte(`{"schema":"v1","algorithm":"sha256","files":{"a":"b"}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), []byte("garbage"), 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for malformed pinned, got %s", res.Status)
	}
}

func TestPinnedBaselineDrift_FileRemoved(t *testing.T) {
	v := NewPinnedBaselineDrift(ValidationGate)
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	os.MkdirAll(baseDir, 0o755)
	// Pinned has 3 files, current has only 2
	pinned := []byte(`{"schema":"v1","algorithm":"sha256","files":{"a":"x","b":"y","c":"z"}}`)
	current := []byte(`{"schema":"v1","algorithm":"sha256","files":{"a":"x","b":"y"}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for missing pinned file, got %s", res.Status)
	}
	if len(res.Issues) == 0 {
		t.Fatal("expected issue for missing surface")
	}
}

func TestPinnedBaselineDrift_ShortHash(t *testing.T) {
	v := NewPinnedBaselineDrift()
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	os.MkdirAll(baseDir, 0o755)
	// Use very short hash (less than 8 chars) to test truncation
	pinned := []byte(`{"schema":"v1","algorithm":"sha256","files":{"file":"abc"}}`)
	current := []byte(`{"schema":"v1","algorithm":"sha256","files":{"file":"xyz"}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail for hash mismatch, got %s", res.Status)
	}
}

func TestPinnedBaselineDrift_LongHash(t *testing.T) {
	v := NewPinnedBaselineDrift()
	root := t.TempDir()
	baseDir := filepath.Join(root, ".ovav", "integrity_backups")
	os.MkdirAll(baseDir, 0o755)
	longPinned := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	longCurrent := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	pinned := []byte(`{"schema":"v1","algorithm":"sha256","files":{"file":"` + longPinned + `"}}`)
	current := []byte(`{"schema":"v1","algorithm":"sha256","files":{"file":"` + longCurrent + `"}}`)
	os.WriteFile(filepath.Join(baseDir, "baseline.pinned.json"), pinned, 0o644)
	os.WriteFile(filepath.Join(baseDir, "baseline.json"), current, 0o644)

	res := v.Validate(context.Background(), root)
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
}

func TestLoadBaselineFile_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	_, err := loadBaselineFile(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadBaselineFile_NonExistent(t *testing.T) {
	_, err := loadBaselineFile("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestLoadBaselineFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "good.json")
	data := map[string]interface{}{
		"schema":    "v1",
		"algorithm": "sha256",
		"files":     map[string]interface{}{"x": "y"},
	}
	jsonData, _ := json.Marshal(data)
	os.WriteFile(path, jsonData, 0o644)

	baseline, err := loadBaselineFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if baseline.Files["x"] != "y" {
		t.Fatalf("expected x=y, got %v", baseline.Files["x"])
	}
}

func TestFinishPinnedResult_Empty(t *testing.T) {
	v := NewPinnedBaselineDrift()
	res := finishPinnedResult(v, time.Now(), nil, "pass")
	if res.Status != "pass" {
		t.Fatalf("expected pass, got %s", res.Status)
	}
	if len(res.Issues) != 0 {
		t.Fatal("expected no issues")
	}
}

func TestFinishPinnedResult_WithIssues(t *testing.T) {
	v := NewPinnedBaselineDrift()
	res := finishPinnedResult(v, time.Now(), []string{"test issue"}, "fail")
	if res.Status != "fail" {
		t.Fatalf("expected fail, got %s", res.Status)
	}
	if len(res.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(res.Issues))
	}
}
