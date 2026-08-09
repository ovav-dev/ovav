// Session Feed — JSONL event stream for OVAV session consciousness.
//
// Replaces tools/governor/session_feed.py (288 LOC Python).
// Each significant event is published to a JSONL file so that OVAV
// observes session activity in real time without subagents.
//
// Thread-safe. Stdlib only.

package governor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── Event types ─────────────────────────────────────────────────────────────

// FeedEvent represents a single event in the session feed.
type FeedEvent struct {
	Timestamp string `json:"ts"`
	Kind      string `json:"kind"`
	Source    string `json:"source"`
	Summary   string `json:"summary"`
	Severity  string `json:"severity"`
	Detail    string `json:"detail,omitempty"`
}

// FeedStatus holds the current state of the session feed.
type FeedStatus struct {
	Exists  bool           `json:"exists"`
	Events  int            `json:"events"`
	SizeKB  float64        `json:"size_kb"`
	ByKind  map[string]int `json:"by_kind,omitempty"`
	Session string         `json:"session"`
	Error   string         `json:"error,omitempty"`
}

// ── Constants ───────────────────────────────────────────────────────────────

const (
	feedMaxSizeMB = 5
	feedMaxEvents = 500
	feedFileName  = "session_feed.jsonl"
)

// ── SessionFeed ─────────────────────────────────────────────────────────────

// SessionFeed manages the JSONL event feed.
// Thread-safe for concurrent publish/read.
type SessionFeed struct {
	mu      sync.Mutex
	feedDir string // directory containing the feed file
}

// NewSessionFeed creates a feed rooted at the given .ovav/runtime directory.
// If feedDir is empty, defaults to ".ovav/runtime".
func NewSessionFeed(feedDir string) *SessionFeed {
	if feedDir == "" {
		feedDir = filepath.Join(".ovav", "runtime")
	}
	return &SessionFeed{feedDir: feedDir}
}

func (f *SessionFeed) feedPath() string {
	return filepath.Join(f.feedDir, feedFileName)
}

// PublishEvent writes an event to the session feed.
// Replaces Python publish_event().
func (f *SessionFeed) PublishEvent(summary, kind, source, severity, detail string) FeedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	if kind == "" {
		kind = "event"
	}
	if source == "" {
		source = "thavren"
	}
	if severity == "" {
		severity = "info"
	}

	event := FeedEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Kind:      kind,
		Source:    source,
		Summary:   summary,
		Severity:  severity,
		Detail:    detail,
	}

	// Ensure directory exists
	_ = os.MkdirAll(f.feedDir, 0o755)

	// Append JSONL line
	data, err := json.Marshal(event)
	if err != nil {
		return event
	}

	file, err := os.OpenFile(f.feedPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return event
	}
	defer file.Close()

	_, _ = file.Write(append(data, '\n'))

	// Rotate if needed (best-effort, errors ignored)
	f.maybeRotate()

	return event
}

// ReadFeed reads events from the feed, optionally filtered.
// Replaces Python read_feed().
func (f *SessionFeed) ReadFeed(sinceMinutes int, limit int, kinds []string) []FeedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.feedPath())
	if err != nil {
		return nil
	}

	var cutoff time.Time
	if sinceMinutes > 0 {
		cutoff = time.Now().UTC().Add(-time.Duration(sinceMinutes) * time.Minute)
	}

	kindSet := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}

	var events []FeedEvent
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var ev FeedEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		// Filter by time
		if !cutoff.IsZero() {
			ts, err := time.Parse(time.RFC3339, ev.Timestamp)
			if err != nil || ts.Before(cutoff) {
				continue
			}
		}

		// Filter by kind
		if len(kindSet) > 0 && !kindSet[ev.Kind] {
			continue
		}

		events = append(events, ev)
	}

	// Return last N
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}

	return events
}

// Status returns the current state of the session feed.
// Replaces Python feed_status().
func (f *SessionFeed) Status() FeedStatus {
	f.mu.Lock()
	defer f.mu.Unlock()

	info, err := os.Stat(f.feedPath())
	if err != nil {
		return FeedStatus{Exists: false, Events: 0, SizeKB: 0, Session: "inactive"}
	}

	data, err := os.ReadFile(f.feedPath())
	if err != nil {
		return FeedStatus{Exists: true, Session: "error", Error: err.Error()}
	}

	byKind := make(map[string]int)
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev FeedEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		byKind[ev.Kind]++
		count++
	}

	return FeedStatus{
		Exists:  true,
		Events:  count,
		SizeKB:  roundTo(float64(info.Size())/1024, 1),
		ByKind:  byKind,
		Session: "active",
	}
}

// Clear removes the feed file. Called at session start.
// Replaces Python clear_feed().
func (f *SessionFeed) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	_ = os.Remove(f.feedPath())
}

// Archive moves the current feed to an archive file and clears the active feed.
// Replaces Python archive_feed().
func (f *SessionFeed) Archive() {
	events := f.ReadFeed(0, 0, nil) // all events
	if len(events) == 0 {
		return
	}

	archivePath := filepath.Join(f.feedDir, "session_feed_archive.jsonl")
	_ = os.MkdirAll(f.feedDir, 0o755)

	file, err := os.OpenFile(archivePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	// Write archive marker
	marker := FeedEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Kind:      "archive_marker",
		Source:    "system",
		Summary:   fmt.Sprintf("--- Session archived: %d events ---", len(events)),
		Severity:  "info",
	}
	if data, err := json.Marshal(marker); err == nil {
		_, _ = file.Write(append(data, '\n'))
	}

	for _, ev := range events {
		if data, err := json.Marshal(ev); err == nil {
			_, _ = file.Write(append(data, '\n'))
		}
	}

	f.Clear()
}

// ── Internal ────────────────────────────────────────────────────────────────

// maybeRotate trims the feed if it exceeds size limits.
// Must be called with f.mu held.
func (f *SessionFeed) maybeRotate() {
	info, err := os.Stat(f.feedPath())
	if err != nil {
		return
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)
	if sizeMB < feedMaxSizeMB {
		return
	}

	data, err := os.ReadFile(f.feedPath())
	if err != nil {
		return
	}

	var allEvents []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			allEvents = append(allEvents, line)
		}
	}

	if len(allEvents) > feedMaxEvents {
		kept := allEvents[len(allEvents)-feedMaxEvents:]
		content := strings.Join(kept, "\n") + "\n"
		_ = os.WriteFile(f.feedPath(), []byte(content), 0o644)
	}
}
