package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashBytes(t *testing.T) {
	got := hashBytes([]byte("hello world"))
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestAtomicWriteLive(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	content := []byte("hello world\n")
	if err := atomicWriteLive(live, content); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("expected %q, got %q", content, got)
	}
}

func TestAtomicWriteLive_Overwrite(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(live, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("new content here")
	if err := atomicWriteLive(live, newContent); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(live)
	if string(got) != string(newContent) {
		t.Fatalf("expected %q, got %q", newContent, got)
	}
}

func TestCreateSnapshot_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(live, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := createSnapshot(dir, "deploy-123", "test-target", live)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Existed {
		t.Fatal("expected Existed=true")
	}
	if snap.Hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if string(snap.Content) != "content" {
		t.Fatalf("content mismatch: %s", snap.Content)
	}
}

func TestCreateSnapshot_NonExistingFile(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "nope.txt")
	snap, err := createSnapshot(dir, "deploy-123", "test-target", live)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Existed {
		t.Fatal("expected Existed=false")
	}
	if snap.Hash != "" {
		t.Fatalf("expected empty hash, got %s", snap.Hash)
	}
}

func TestPersistAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	snap := DeploySnapshot{
		TargetID: "test",
		LivePath: "/some/live/path",
		Content:  []byte("snap content"),
		Hash:     "abc123",
		Existed:  true,
	}
	if err := persistSnapshot(dir, "deploy-1", snap); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSnapshot(dir, "deploy-1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TargetID != "test" || loaded.Hash != "abc123" {
		t.Fatalf("snapshot mismatch: %+v", loaded)
	}
	if string(loaded.Content) != "snap content" {
		t.Fatalf("content mismatch: %s", loaded.Content)
	}
}

func TestVerifyDeploy_Match(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	content := []byte("verify me")
	if err := atomicWriteLive(live, content); err != nil {
		t.Fatal(err)
	}
	if err := verifyDeploy(live, content); err != nil {
		t.Fatalf("expected verify to pass, got: %v", err)
	}
}

func TestVerifyDeploy_Mismatch(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	if err := os.WriteFile(live, []byte("actual content"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Expected content differs from what's on disk
	if err := verifyDeploy(live, []byte("expected content")); err == nil {
		t.Fatal("expected verify to fail on hash mismatch")
	}
}

func TestRollbackFromSnapshot(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	original := []byte("original content")
	if err := os.WriteFile(live, original, 0o644); err != nil {
		t.Fatal(err)
	}
	snap, err := createSnapshot(dir, "deploy-1", "test", live)
	if err != nil {
		t.Fatal(err)
	}
	// Modify live
	newContent := []byte("deployed content")
	if err := atomicWriteLive(live, newContent); err != nil {
		t.Fatal(err)
	}
	// Rollback
	if err := rollbackFromSnapshot(dir, "deploy-1", snap); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(live)
	if string(got) != string(original) {
		t.Fatalf("rollback failed: got %q, want %q", got, original)
	}
}

func TestRollbackFromSnapshot_NonExisted(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "live.txt")
	// Pretend we created the file via deploy
	if err := os.WriteFile(live, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	snap := DeploySnapshot{
		TargetID: "test",
		LivePath: live,
		Existed:  false, // file didn't exist before deploy
	}
	if err := rollbackFromSnapshot(dir, "deploy-1", snap); err != nil {
		t.Fatal(err)
	}
	// File should be removed
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatal("expected live file to be removed")
	}
}

func TestAppendAndReadDeployHistory(t *testing.T) {
	dir := t.TempDir()
	rec := DeployRecord{
		DeployID:   "deploy-1",
		Timestamp:  "2026-08-14T19:38:00Z",
		Operator:   "thavren",
		Status:     "success",
		DurationMs: 123,
	}
	if err := appendDeployHistory(dir, rec); err != nil {
		t.Fatal(err)
	}
	if err := appendDeployHistory(dir, rec); err != nil {
		t.Fatal(err)
	}
	records, err := readDeployHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	// Records are returned most-recent-first (but here same, so order is preserved)
	if records[0].DeployID != "deploy-1" {
		t.Fatalf("unexpected ID: %s", records[0].DeployID)
	}
}

func TestGenerateDeployID_Format(t *testing.T) {
	id := generateDeployID()
	if !strings.HasPrefix(id, "deploy-") {
		t.Fatalf("expected prefix 'deploy-', got %q", id)
	}
	if len(id) < 25 {
		t.Fatalf("expected ID length >= 25, got %q (len=%d)", id, len(id))
	}
}

func TestListSnapshots(t *testing.T) {
	dir := t.TempDir()
	// Create two snapshot dirs
	for _, id := range []string{"deploy-aaa", "deploy-bbb"} {
		if err := os.MkdirAll(filepath.Join(snapshotDir(dir), id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ids, err := listSnapshots(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(ids))
	}
}

func TestAppendAndReadCatalogEntry_JsonFormat(t *testing.T) {
	dir := t.TempDir()
	rec := DeployRecord{
		DeployID: "deploy-test-1",
		Operator: "thavren",
		Status:   "success",
	}
	if err := appendDeployHistory(dir, rec); err != nil {
		t.Fatal(err)
	}
	records, err := readDeployHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	// Should be valid JSON
	for _, r := range records {
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		var parsed DeployRecord
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("invalid JSON %q: %v", data, err)
		}
	}
}