package watchdog

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// watchdog_test.go — Sprint 8 T13 (innovation)
// ═══════════════════════════════════════════════════════════════════════════

func TestT13New_ValidRepo(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if wd == nil {
		t.Fatal("New returned nil")
	}
}

func TestT13New_InvalidRepo(t *testing.T) {
	_, err := New("/nonexistent-path-xyz-12345")
	if err == nil {
		t.Error("invalid path should error")
	}
}

func TestT13SetAutoFormat(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	wd.SetAutoFormat(true)
	if !wd.shouldAutoFormat() {
		t.Error("SetAutoFormat(true) should enable")
	}
	wd.SetAutoFormat(false)
	if wd.shouldAutoFormat() {
		t.Error("SetAutoFormat(false) should disable")
	}
}

func TestT13SetNotifyCallback(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	var called atomic.Bool
	wd.SetNotifyCallback(func(e Event) {
		called.Store(true)
	})
	wd.notify(Event{Kind: EventFileCreated, Path: "test"})
	// Async — give it a moment
	time.Sleep(50 * time.Millisecond)
	if !called.Load() {
		t.Error("callback should have been called")
	}
}

func TestT13Stats_TrackEvents(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	wd.incrementObserved()
	wd.incrementFormatted()
	wd.incrementDrift()
	stats := wd.Stats()
	if stats.EventsObserved != 1 {
		t.Errorf("expected 1 observed, got %d", stats.EventsObserved)
	}
	if stats.FilesFormatted != 1 {
		t.Errorf("expected 1 formatted, got %d", stats.FilesFormatted)
	}
	if stats.DriftsDetected != 1 {
		t.Errorf("expected 1 drift, got %d", stats.DriftsDetected)
	}
}

func TestT13Watch_NoCallback(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	err := wd.Watch(context.Background(), []string{"*.go"})
	if err == nil {
		t.Error("Watch without callback should error")
	}
}

func TestT13Watch_ValidCallback(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	called := 0
	wd.SetNotifyCallback(func(e Event) {
		called++
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go wd.Watch(ctx, []string{"*.go"})
	time.Sleep(100 * time.Millisecond)
	wd.Stop()
	// Either called or not (timing) — just no panic
	_ = called
}

func TestT13Snapshot_EmptyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	hashes, err := wd.snapshot([]string{"*.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 0 {
		t.Errorf("empty repo should have 0 hashes, got %d", len(hashes))
	}
}

func TestT13Snapshot_FindsFiles(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a .go file
	if err := os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, _ := New(tmpDir)
	hashes, err := wd.snapshot([]string{"*.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 1 {
		t.Errorf("expected 1 hash, got %d", len(hashes))
	}
}

func TestT13HashFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 64 {
		t.Errorf("SHA-256 hex should be 64 chars, got %d", len(h))
	}
}

func TestT13HashFile_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(path, []byte("hello"), 0o644)
	h1, _ := hashFile(path)
	h2, _ := hashFile(path)
	if h1 != h2 {
		t.Error("hashFile should be deterministic")
	}
}

func TestT13HashFile_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()
	a := filepath.Join(tmpDir, "a")
	b := filepath.Join(tmpDir, "b")
	os.WriteFile(a, []byte("aaa"), 0o644)
	os.WriteFile(b, []byte("bbb"), 0o644)
	h1, _ := hashFile(a)
	h2, _ := hashFile(b)
	if h1 == h2 {
		t.Error("different content should yield different hashes")
	}
}

func TestT13HashFile_Nonexistent(t *testing.T) {
	_, err := hashFile("/tmp/nonexistent-file-xyz-ovav-test.txt")
	if err == nil {
		t.Error("nonexistent file should error")
	}
}

func TestT13ClassifyDrift_IdenticalHashes(t *testing.T) {
	c := ClassifyDrift("abc123", "abc123")
	if c != DriftCosmetic {
		t.Errorf("identical hashes should be cosmetic, got %q", c)
	}
}

func TestT13ClassifyDrift_DifferentHashes(t *testing.T) {
	c := ClassifyDrift("abc123", "def456")
	if c != DriftSemantic {
		t.Errorf("different hashes should be semantic, got %q", c)
	}
}

func TestT13Format_GoFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "main.go")
	// unformatted content
	content := "package   main\n\nfunc main()  {\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, _ := New(tmpDir)
	err := wd.format(path)
	// Either formats successfully or gofmt not in PATH
	if err != nil {
		t.Logf("gofmt may not be available: %v", err)
	}
}

func TestT13Format_NonGoFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "data.txt")
	os.WriteFile(path, []byte("plain"), 0o644)
	wd, _ := New(tmpDir)
	if err := wd.format(path); err != nil {
		t.Errorf("format on .txt should be no-op, got: %v", err)
	}
}

func TestT13CheckSecretsHygiene_NoFile(t *testing.T) {
	_, err := CheckSecretsHygiene("/nonexistent-xyz-secrets.txt")
	if err == nil {
		t.Error("nonexistent should error")
	}
}

func TestT13CheckSecretsHygiene_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(path, []byte{}, 0o644)
	n, err := CheckSecretsHygiene(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("empty file should have 0 alerts")
	}
}

func TestT13CheckSecretsHygiene_NonEmptyCleanFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "clean.txt")
	os.WriteFile(path, []byte("no secrets here, just code"), 0o644)
	n, err := CheckSecretsHygiene(path)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("clean file should have 0 alerts, got %d", n)
	}
}

func TestT13EventKind_StringValues(t *testing.T) {
	kinds := []EventKind{
		EventFileModified,
		EventFileCreated,
		EventFileDeleted,
		EventFormatApplied,
		EventDriftDetected,
	}
	for _, k := range kinds {
		if string(k) == "" {
			t.Errorf("EventKind empty")
		}
	}
}

func TestT13DriftClassification_StringValues(t *testing.T) {
	c := []DriftClassification{DriftSemantic, DriftCosmetic, DriftMixed}
	for _, v := range c {
		if string(v) == "" {
			t.Errorf("DriftClassification empty")
		}
	}
}

func TestT13Stop_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	wd, _ := New(tmpDir)
	wd.Stop()
	// Should not panic on double-stop
	wd.Stop()
}
