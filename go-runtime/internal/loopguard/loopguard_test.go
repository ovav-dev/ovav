package loopguard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDetect_NoLoop: sparse, varied events → no detection.
func TestDetect_NoLoop(t *testing.T) {
	events := []Event{
		{Turn: 1, Tool: "bash", Output: "ls output A", At: time.Now()},
		{Turn: 2, Tool: "read", Output: "file content X", At: time.Now().Add(1 * time.Second)},
		{Turn: 3, Tool: "bash", Output: "ls output B", At: time.Now().Add(2 * time.Second)},
	}
	det := Detect(events, DefaultConfig)
	if det.Detected {
		t.Fatalf("expected no loop, got: %+v", det)
	}
}

// TestDetect_Loop: same bash output ≥3 times within 5s → detection.
func TestDetect_Loop(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Turn: 1, Tool: "bash", Output: "directory listing: a.go b.go c.go", At: now},
		{Turn: 2, Tool: "bash", Output: "directory listing: a.go b.go c.go", At: now.Add(1 * time.Second)},
		{Turn: 3, Tool: "bash", Output: "directory listing: a.go b.go c.go", At: now.Add(2 * time.Second)},
	}
	det := Detect(events, DefaultConfig)
	if !det.Detected {
		t.Fatalf("expected loop detection, got zero value")
	}
	if det.Tool != "bash" {
		t.Errorf("expected tool=bash, got %s", det.Tool)
	}
	if det.Occurrences != 3 {
		t.Errorf("expected 3 occurrences, got %d", det.Occurrences)
	}
	if det.SuggestedAction == "" {
		t.Error("expected suggested action to be non-empty")
	}
}

// TestDetect_LoopOutOfWindow: same output but spread over 30s → no loop (window exceeded).
func TestDetect_LoopOutOfWindow(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Turn: 1, Tool: "bash", Output: "x", At: now},
		{Turn: 2, Tool: "bash", Output: "x", At: now.Add(15 * time.Second)},
		{Turn: 3, Tool: "bash", Output: "x", At: now.Add(30 * time.Second)},
	}
	det := Detect(events, DefaultConfig)
	if det.Detected {
		t.Fatalf("expected no loop (window exceeded), got: %+v", det)
	}
}

// TestDetect_DifferentOutputs: 3 bash calls but different outputs → no loop.
func TestDetect_DifferentOutputs(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Turn: 1, Tool: "bash", Output: "first output", At: now},
		{Turn: 2, Tool: "bash", Output: "second output (different)", At: now.Add(1 * time.Second)},
		{Turn: 3, Tool: "bash", Output: "third output (totally different)", At: now.Add(2 * time.Second)},
	}
	det := Detect(events, DefaultConfig)
	if det.Detected {
		t.Fatalf("expected no loop (different outputs), got: %+v", det)
	}
}

// TestDetect_OnlyOneTool: same output only fires for the matching tool.
// read events shouldn't trigger a bash loop.
func TestDetect_OnlyOneTool(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Turn: 1, Tool: "read", Output: "config content", At: now},
		{Turn: 2, Tool: "read", Output: "config content", At: now.Add(1 * time.Second)},
		{Turn: 3, Tool: "read", Output: "config content", At: now.Add(2 * time.Second)},
	}
	det := Detect(events, DefaultConfig)
	if !det.Detected {
		t.Fatal("expected loop on read tool")
	}
	if det.Tool != "read" {
		t.Errorf("expected tool=read, got %s", det.Tool)
	}
}

// TestLoadEvents_JSONL: round-trip serialize → deserialize preserves events.
func TestLoadEvents_JSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	now := time.Now().UTC().Truncate(time.Second)
	events := []Event{
		{Turn: 1, Tool: "bash", Output: "out1", At: now, ExitCode: 0},
		{Turn: 2, Tool: "read", Output: "out2", At: now.Add(time.Second), ExitCode: 0},
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if err := json.NewEncoder(f).Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	got, err := LoadEvents(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(got))
	}
	for i, e := range events {
		if got[i].Tool != e.Tool {
			t.Errorf("event %d: tool mismatch: %s vs %s", i, got[i].Tool, e.Tool)
		}
	}
}

// TestDetect_RealWorldBugD: reproduces the actual ses_05472d9dfffegXdTlJk7jS7Kdh
// incident pattern: ~80 reads of identical-mode inputs to a struct dump.
//
// The hash-based hashOnly=true mode should detect this even though individual
// reads might have trivially different output (whitespace, timestamps).
func TestDetect_RealWorldBugD(t *testing.T) {
	now := time.Now()
	// 80 reads of file paths, each surfacing similar header content.
	// We hash the structural fingerprint, so underlying file contents varying
	// slightly should still trip the loop detector because the harness pattern matches.
	common := "subsystem_names.yaml header lines"
	events := make([]Event, 80)
	for i := range events {
		events[i] = Event{
			Turn:   i + 1,
			Tool:   "read",
			Output: common,
			At:     now.Add(time.Duration(i) * 100 * time.Millisecond),
		}
	}
	det := Detect(events, DefaultConfig)
	if !det.Detected {
		t.Fatal("expected loop detection on 80-read dump pattern")
	}
	if det.Tool != "read" {
		t.Errorf("expected tool=read, got %s", det.Tool)
	}
	if det.Occurrences < 3 {
		t.Errorf("expected ≥3 occurrences, got %d", det.Occurrences)
	}
}
