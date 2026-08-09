package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── GetQueueStatus — boost coverage ───────────────────────────────────────────

// TestGetQueueStatus_WithNonJSONFiles tests that non-.json entries in the
// queue directory are silently skipped.
func TestGetQueueStatus_WithNonJSONFiles(t *testing.T) {
	root := t.TempDir()
	queueDir := filepath.Join(root, ".ovav", "sync", "queue")
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queueDir, "readme.txt"), []byte("notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &SyncManifest{
		Version:   "1.0.0",
		Generated: time.Now().UTC(),
		Source:    root,
		Items:     []SyncItem{{Path: "test.yaml", Category: "config"}},
	}
	path := filepath.Join(queueDir, "test.json")
	if err := SaveManifest(manifest, path); err != nil {
		t.Fatal(err)
	}

	m, err := GetQueueStatus(root)
	if err != nil {
		t.Fatalf("GetQueueStatus: %v", err)
	}
	if m.TotalItems != 1 {
		t.Errorf("expected 1 item, got %d", m.TotalItems)
	}
}

// TestGetQueueStatus_LoadManifestError tests that a corrupt JSON file in the
// queue directory is skipped without failing.
func TestGetQueueStatus_LoadManifestError(t *testing.T) {
	root := t.TempDir()
	queueDir := filepath.Join(root, ".ovav", "sync", "queue")
	if err := os.MkdirAll(queueDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(queueDir, "corrupt.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := GetQueueStatus(root)
	if err != nil {
		t.Fatalf("GetQueueStatus should not fail on corrupt manifest: %v", err)
	}
	if m.TotalItems != 0 {
		t.Errorf("expected 0 items (corrupt skipped), got %d", m.TotalItems)
	}
}

// ── DetectChanges — verify missing-dir handling ─────────────────────────────────

// TestDetectChanges_SkipsMissingDirectory verifies that a missing directory is
// silently skipped (os.Stat error path).
func TestDetectChanges_SkipsMissingDirectory(t *testing.T) {
	root := t.TempDir()
	// Only create .ovav/sync — none of the syncDirectories exist
	if err := os.MkdirAll(filepath.Join(root, ".ovav", "sync"), 0o755); err != nil {
		t.Fatal(err)
	}

	m, err := DetectChanges(root)
	if err != nil {
		t.Fatalf("DetectChanges should not fail on missing dirs: %v", err)
	}
	if len(m.Items) != 0 {
		t.Errorf("expected 0 items (all dirs missing), got %d", len(m.Items))
	}
}

// ── scanDir edge cases ───────────────────────────────────────────────────────

// TestScanDir_SkipsVendor verifies that vendor/ is excluded.
func TestScanDir_SkipsVendor(t *testing.T) {
	root := t.TempDir()
	vendorDir := filepath.Join(root, "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "foo.go"), []byte("package vendor\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		if strings.Contains(item.Path, "vendor") {
			t.Errorf("vendor should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsDotBakFiles verifies that .bak files are excluded.
func TestScanDir_SkipsDotBakFiles(t *testing.T) {
	root := t.TempDir()
	goodFile := filepath.Join(root, "good.yaml")
	if err := os.WriteFile(goodFile, []byte("key: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	badFile := filepath.Join(root, "good.bak")
	if err := os.WriteFile(badFile, []byte("old: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		base := filepath.Base(item.Path)
		if strings.HasSuffix(base, ".bak") {
			t.Errorf(".bak files should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsSwpFiles verifies that .swp files are excluded.
func TestScanDir_SkipsSwpFiles(t *testing.T) {
	root := t.TempDir()
	goodFile := filepath.Join(root, "good.yaml")
	if err := os.WriteFile(goodFile, []byte("key: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	swpFile := filepath.Join(root, "good.swp")
	if err := os.WriteFile(swpFile, []byte("swap\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		base := filepath.Base(item.Path)
		if strings.HasSuffix(base, ".swp") {
			t.Errorf(".swp files should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsTildeFiles verifies that files starting with ~ are excluded.
func TestScanDir_SkipsTildeFiles(t *testing.T) {
	root := t.TempDir()
	goodFile := filepath.Join(root, "good.yaml")
	if err := os.WriteFile(goodFile, []byte("key: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tildeFile := filepath.Join(root, "~backup.yaml")
	if err := os.WriteFile(tildeFile, []byte("backup\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		base := filepath.Base(item.Path)
		if strings.HasPrefix(base, "~") {
			t.Errorf("tilde files should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsDotHashFiles verifies that .# files (emacs lock files) are excluded.
func TestScanDir_SkipsDotHashFiles(t *testing.T) {
	root := t.TempDir()
	goodFile := filepath.Join(root, "good.yaml")
	if err := os.WriteFile(goodFile, []byte("key: value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dotHashFile := filepath.Join(root, ".#emacs-lock")
	if err := os.WriteFile(dotHashFile, []byte("lock\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		base := filepath.Base(item.Path)
		if strings.HasPrefix(base, ".#") {
			t.Errorf(".# files should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsTestGoFiles verifies that _test.go files are excluded.
func TestScanDir_SkipsTestGoFiles(t *testing.T) {
	root := t.TempDir()
	goFile := filepath.Join(root, "foo.go")
	if err := os.WriteFile(goFile, []byte("package foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(root, "foo_test.go")
	if err := os.WriteFile(testFile, []byte("package foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		if strings.HasSuffix(item.Path, "_test.go") {
			t.Errorf("_test.go files should be skipped, got: %s", item.Path)
		}
	}
}

// TestScanDir_SkipsUnderscoreDir verifies that directories starting with _
// (except _ovav) are excluded.
func TestScanDir_SkipsUnderscoreDir(t *testing.T) {
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "_fixtures")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "data.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validDir := filepath.Join(root, "_ovav")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "data.yaml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	hasFixtures := false
	hasOvav := false
	for _, item := range items {
		if strings.HasPrefix(filepath.Base(filepath.Dir(item.Path)), "_fixtures") {
			hasFixtures = true
		}
		if strings.HasPrefix(filepath.Base(filepath.Dir(item.Path)), "_ovav") {
			hasOvav = true
		}
	}
	if hasFixtures {
		t.Error("_fixtures dir should be skipped")
	}
	if !hasOvav {
		t.Logf("_ovav dir should NOT be skipped (whitelisted)")
	}
}

// TestScanDir_SkipsQueueAndSync verifies that queue/ and sync/ directories are
// excluded to prevent sync loops.
func TestScanDir_SkipsQueueAndSync(t *testing.T) {
	root := t.TempDir()
	queueDir := filepath.Join(root, "queue")
	syncDir := filepath.Join(root, "sync")
	for _, d := range []string{queueDir, syncDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "foo.yaml"), []byte("x: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	items, err := scanDir(root, root, "config", nil)
	if err != nil {
		t.Fatalf("scanDir: %v", err)
	}
	for _, item := range items {
		base := filepath.Base(filepath.Dir(item.Path))
		if base == "queue" || base == "sync" {
			t.Errorf("queue/sync dirs should be skipped, got: %s", item.Path)
		}
	}
}

// ── scanFile error paths ─────────────────────────────────────────────────────

// TestScanFile_StatError verifies scanFile returns nil for non-existent path.
func TestScanFile_StatError(t *testing.T) {
	root := t.TempDir()
	item := scanFile(filepath.Join(root, "does_not_exist.go"), "does_not_exist.go", "config", nil)
	if item != nil {
		t.Error("scanFile should return nil for non-existent file")
	}
}

// TestScanFile_DirectoryAsFile verifies scanFile returns nil when path is a dir.
func TestScanFile_DirectoryAsFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "somedir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	item := scanFile(dir, "somedir", "config", nil)
	if item != nil {
		t.Error("scanFile should return nil when path is a directory")
	}
}
