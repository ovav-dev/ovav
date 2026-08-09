package governor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionFeed_PublishAndRead(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	ev := feed.PublishEvent("Test event", "decision", "thavren", "info", "")
	if ev.Kind != "decision" {
		t.Errorf("kind = %q, want decision", ev.Kind)
	}
	if ev.Source != "thavren" {
		t.Errorf("source = %q, want thavren", ev.Source)
	}
	if ev.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}

	// Read back
	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].Summary != "Test event" {
		t.Errorf("summary = %q, want 'Test event'", events[0].Summary)
	}
}

func TestSessionFeed_PublishMultiple(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("Event 1", "event", "thavren", "info", "")
	feed.PublishEvent("Event 2", "decision", "ovav", "info", "")
	feed.PublishEvent("Event 3", "alert", "ovav", "warn", "")

	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
}

func TestSessionFeed_FilterByKind(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("E1", "event", "thavren", "info", "")
	feed.PublishEvent("D1", "decision", "ovav", "info", "")
	feed.PublishEvent("E2", "event", "thavren", "info", "")

	events := feed.ReadFeed(0, 50, []string{"decision"})
	if len(events) != 1 {
		t.Fatalf("want 1 decision event, got %d", len(events))
	}
	if events[0].Summary != "D1" {
		t.Errorf("summary = %q, want D1", events[0].Summary)
	}
}

func TestSessionFeed_ReadLimit(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	for i := 0; i < 10; i++ {
		feed.PublishEvent("event", "event", "thavren", "info", "")
	}

	events := feed.ReadFeed(0, 3, nil)
	if len(events) != 3 {
		t.Fatalf("want 3 events with limit, got %d", len(events))
	}
}

func TestSessionFeed_Status(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	// Empty feed
	status := feed.Status()
	if status.Exists {
		t.Error("feed should not exist yet")
	}

	// After publishing
	feed.PublishEvent("E1", "event", "thavren", "info", "")
	feed.PublishEvent("D1", "decision", "ovav", "info", "")

	status = feed.Status()
	if !status.Exists {
		t.Error("feed should exist")
	}
	if status.Events != 2 {
		t.Errorf("events = %d, want 2", status.Events)
	}
	if status.Session != "active" {
		t.Errorf("session = %q, want active", status.Session)
	}
	if status.ByKind["event"] != 1 {
		t.Errorf("event count = %d, want 1", status.ByKind["event"])
	}
	if status.ByKind["decision"] != 1 {
		t.Errorf("decision count = %d, want 1", status.ByKind["decision"])
	}
}

func TestSessionFeed_Clear(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("E1", "event", "thavren", "info", "")
	feed.Clear()

	status := feed.Status()
	if status.Exists {
		t.Error("feed should not exist after clear")
	}
}

func TestSessionFeed_Archive(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("E1", "event", "thavren", "info", "")
	feed.PublishEvent("E2", "event", "thavren", "info", "")

	feed.Archive()

	// Active feed should be cleared
	status := feed.Status()
	if status.Exists {
		t.Error("active feed should be empty after archive")
	}

	// Archive should exist
	archivePath := filepath.Join(dir, "session_feed_archive.jsonl")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archive file should exist: %v", err)
	}
}

func TestSessionFeed_Detail(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("With detail", "event", "thavren", "info", "extended info here")

	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].Detail != "extended info here" {
		t.Errorf("detail = %q, want 'extended info here'", events[0].Detail)
	}
}

func TestSessionFeed_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	feed.PublishEvent("Defaults", "", "", "", "")

	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	if events[0].Kind != "event" {
		t.Errorf("kind = %q, want event", events[0].Kind)
	}
	if events[0].Source != "thavren" {
		t.Errorf("source = %q, want thavren", events[0].Source)
	}
	if events[0].Severity != "info" {
		t.Errorf("severity = %q, want info", events[0].Severity)
	}
}

func TestSessionFeed_EmptyRead(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 0 {
		t.Errorf("want 0 events from empty feed, got %d", len(events))
	}
}

func TestSessionFeed_ConcurrentPublish(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			feed.PublishEvent("concurrent", "event", "thavren", "info", "")
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	events := feed.ReadFeed(0, 50, nil)
	if len(events) != 10 {
		t.Errorf("want 10 events, got %d", len(events))
	}
}

// maybeRotate is hard to fully exercise without a >5MB file.
// These tests cover the early-return branches that don't need large files.

func TestSessionFeed_MaybeRotate_FileNotExist(t *testing.T) {
	// Create a feed with a non-existent path — maybeRotate returns early
	feed := &SessionFeed{feedDir: "/nonexistent/path/that/does/not/exist"}
	feed.mu.Lock()
	feed.maybeRotate() // Should return early without error
	feed.mu.Unlock()
	// Feed dir doesn't exist, but no error is returned (best-effort)
}

func TestSessionFeed_MaybeRotate_SmallFile(t *testing.T) {
	// Create a feed with a small file — size < feedMaxSizeMB, returns early
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	// Publish a few events — file will be small (< 5MB)
	for i := 0; i < 5; i++ {
		feed.PublishEvent("small event", "event", "thavren", "info", "")
	}

	// maybeRotate called internally by PublishEvent
	// Verify file exists and is small
	feedPath := filepath.Join(dir, "session_feed.jsonl")
	info, err := os.Stat(feedPath)
	if err != nil {
		t.Fatalf("feed file should exist: %v", err)
	}
	sizeMB := float64(info.Size()) / (1024 * 1024)
	if sizeMB >= 5 {
		t.Skip("file unexpectedly large, skipping")
	}
}
