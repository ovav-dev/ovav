package memorybridge

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// bridge_test.go — Sprint 8 T13 (innovation)
// ═══════════════════════════════════════════════════════════════════════════

func setupBridge(t *testing.T) *Bridge {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "memories.db")
	b, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { b.Close() })
	return b
}

func TestT13New_ValidPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	b, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Close()
	if b.path != path {
		t.Errorf("path should match, got %q", b.path)
	}
}

func TestT13New_EmptyPath(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("empty path should error")
	}
}

func TestT13New_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.db")
	b1, _ := New(path)
	b1.Close()
	b2, err := New(path)
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	b2.Close()
}

func TestT13Put_BasicMemory(t *testing.T) {
	b := setupBridge(t)
	m := Memory{
		Actor:   "thavren",
		Kind:    KindInsight,
		Content: "test content",
	}
	if _, err := b.Put(context.Background(), m); err != nil {
		t.Fatalf("Put: %v", err)
	}
}

func TestT13Put_AutoID(t *testing.T) {
	b := setupBridge(t)
	m := Memory{
		Actor:   "thavren",
		Kind:    KindDecision,
		Content: "auto id test",
	}
	got, err := b.Put(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == "" {
		t.Error("ID should be auto-generated")
	}
}

func TestT13Put_AutoTimestamp(t *testing.T) {
	b := setupBridge(t)
	m := Memory{
		Actor:   "thavren",
		Kind:    KindNote,
		Content: "timestamp test",
	}
	before := time.Now()
	got, _ := b.Put(context.Background(), m)
	after := time.Now()
	if got.Timestamp.Before(before) || got.Timestamp.After(after) {
		t.Errorf("timestamp %v not in [%v, %v]", got.Timestamp, before, after)
	}
}

func TestT13Get_Existing(t *testing.T) {
	b := setupBridge(t)
	m := Memory{Actor: "thavren", Kind: KindInsight, Content: "findable"}
	put, _ := b.Put(context.Background(), m)
	got, err := b.Get(context.Background(), put.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "findable" {
		t.Errorf("content mismatch: %q", got.Content)
	}
}

func TestT13Get_Nonexistent(t *testing.T) {
	b := setupBridge(t)
	_, err := b.Get(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("nonexistent ID should error")
	}
}

func TestT13List_Empty(t *testing.T) {
	b := setupBridge(t)
	mems, err := b.List(context.Background(), ListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(mems) != 0 {
		t.Errorf("empty bridge should return 0, got %d", len(mems))
	}
}

func TestT13List_AllKind(t *testing.T) {
	b := setupBridge(t)
	for _, k := range AllKinds() {
		b.Put(context.Background(), Memory{Actor: "tester", Kind: k, Content: string(k)})
	}
	mems, _ := b.List(context.Background(), ListFilter{Limit: 100})
	if len(mems) < len(AllKinds()) {
		t.Errorf("expected %d memories, got %d", len(AllKinds()), len(mems))
	}
}

func TestT13List_FilterByKind(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindInsight, Content: "i1"})
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindError, Content: "e1"})
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindInsight, Content: "i2"})
	mems, _ := b.List(context.Background(), ListFilter{Kind: KindInsight, Limit: 10})
	if len(mems) != 2 {
		t.Errorf("expected 2 insights, got %d", len(mems))
	}
}

func TestT13List_FilterByActor(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "alice", Kind: KindNote, Content: "a1"})
	b.Put(context.Background(), Memory{Actor: "bob", Kind: KindNote, Content: "b1"})
	b.Put(context.Background(), Memory{Actor: "alice", Kind: KindNote, Content: "a2"})
	mems, _ := b.List(context.Background(), ListFilter{Actor: "alice", Limit: 10})
	if len(mems) != 2 {
		t.Errorf("expected 2 alice memories, got %d", len(mems))
	}
}

func TestT13Search_Found(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindNote, Content: "the quick brown fox"})
	got, err := b.Search(context.Background(), "fox", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Error("search should find match")
	}
}

func TestT13Search_NotFound(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindNote, Content: "hello world"})
	got, _ := b.Search(context.Background(), "zzzzz-no-match", 10)
	if len(got) != 0 {
		t.Errorf("no-match search should return 0, got %d", len(got))
	}
}

func TestT13Count_All(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindNote, Content: "n1"})
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindError, Content: "e1"})
	n, _ := b.Count(context.Background(), "")
	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
}

func TestT13Count_Filtered(t *testing.T) {
	b := setupBridge(t)
	for i := 0; i < 3; i++ {
		b.Put(context.Background(), Memory{Actor: "a", Kind: KindInsight, Content: "ins"})
	}
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindNote, Content: "n"})
	n, _ := b.Count(context.Background(), KindInsight)
	if n != 3 {
		t.Errorf("expected 3 insights, got %d", n)
	}
}

func TestT13Export_ValidJSON(t *testing.T) {
	b := setupBridge(t)
	b.Put(context.Background(), Memory{Actor: "a", Kind: KindNote, Content: "exp"})
	data, err := b.Export(context.Background())
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(data) == 0 {
		t.Error("export should produce non-empty JSON")
	}
}

func TestT13ValidKind_Accept(t *testing.T) {
	if !ValidKind(KindInsight) {
		t.Error("KindInsight should be valid")
	}
}

func TestT13ValidKind_Reject(t *testing.T) {
	if ValidKind(MemoryKind("bogus")) {
		t.Error("bogus kind should not be valid")
	}
}

func TestT13AllKinds_Count(t *testing.T) {
	kinds := AllKinds()
	if len(kinds) != 7 {
		t.Errorf("expected 7 kinds, got %d", len(kinds))
	}
}

func TestT13FormatSummary_Empty(t *testing.T) {
	s := FormatSummary(nil)
	if s == "" {
		t.Error("empty summary shouldn't be empty string")
	}
	if s != "no memories" {
		t.Errorf("expected 'no memories', got %q", s)
	}
}

func TestT13FormatSummary_Populated(t *testing.T) {
	mems := []Memory{
		{Actor: "a", Kind: KindInsight, Content: "test"},
	}
	s := FormatSummary(mems)
	if s == "" || s == "no memories" {
		t.Errorf("populated summary should not be empty: %q", s)
	}
}

func TestT13HashID_Deterministic(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	id1 := hashID("actor", KindInsight, "content", now)
	id2 := hashID("actor", KindInsight, "content", now)
	if id1 != id2 {
		t.Errorf("hashID should be deterministic: %s vs %s", id1, id2)
	}
}

func TestT13HashID_DifferentContent(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	id1 := hashID("actor", KindInsight, "content1", now)
	id2 := hashID("actor", KindInsight, "content2", now)
	if id1 == id2 {
		t.Error("hashID should differ for different content")
	}
}

func TestT13HashID_Format(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	id := hashID("actor", KindInsight, "content", now)
	if len(id) != 16 {
		t.Errorf("ID should be 16 chars, got %d", len(id))
	}
}

func TestT13Close_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	b, _ := New(path)
	if err := b.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	// Second close should not panic
	if err := b.Close(); err != nil {
		t.Logf("second Close returned: %v", err)
	}
}

func TestT13Put_WithTags(t *testing.T) {
	b := setupBridge(t)
	m := Memory{
		Actor:   "thavren",
		Kind:    KindInsight,
		Content: "tagged memory",
		Tags:    []string{"alpha", "beta"},
	}
	got, _ := b.Put(context.Background(), m)
	fetched, _ := b.Get(context.Background(), got.ID)
	if len(fetched.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(fetched.Tags))
	}
}
