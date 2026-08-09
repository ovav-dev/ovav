package loopguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRecordAndCheck_NoLoop: mixed args + mixed outputs → no detection.
// A real "no loop" pattern has VARYING args OR VARYING outputs, not
// 5 identical calls. 3 identical calls IS a loop (covered by next test).
func TestRecordAndCheck_NoLoop(t *testing.T) {
	tmp := t.TempDir()
	eventsFile := filepath.Join(tmp, "events.jsonl")
	incidentsFile := filepath.Join(tmp, "incidents.yaml")
	SetSessionEventsPath(eventsFile)
	incidentsPath = incidentsFile
	t.Cleanup(func() {
		eventsPath = ".ovav/runtime/session_events.jsonl"
		incidentsPath = ".ovav/registry/loop_incidents.yaml"
	})

	// 2 calls with VARIED args — below MinOccurrences=3 threshold.
	calls := []struct {
		tool, args, output string
	}{
		{"bash", "echo a", "out1"},
		{"read", "file_b.go", "out2"},
	}
	for i, c := range calls {
		_, rc := RecordAndCheck(c.tool, c.args, c.output, i)
		if rc != 0 {
			t.Fatalf("iter %d: unexpected loop detection, rc=%d", i, rc)
		}
	}
}

// TestRecordAndCheck_LoopDetected: 5 identical bash calls → loop.
func TestRecordAndCheck_LoopDetected(t *testing.T) {
	tmp := t.TempDir()
	eventsFile := filepath.Join(tmp, "events.jsonl")
	incidentsFile := filepath.Join(tmp, "incidents.yaml")
	SetSessionEventsPath(eventsFile)
	incidentsPath = incidentsFile
	t.Cleanup(func() {
		eventsPath = ".ovav/runtime/session_events.jsonl"
		incidentsPath = ".ovav/registry/loop_incidents.yaml"
	})

	detected := false
	for i := 0; i < 6; i++ {
		_, rc := RecordAndCheck("read", "go-runtime/cmd/cockpit/welcome.go", "package main... 129 lines", i)
		if rc == 1 {
			detected = true
			break
		}
	}
	if !detected {
		t.Fatal("expected loop detection after 6 identical read calls")
	}

	// Verify incident auto-logged.
	if _, err := os.Stat(incidentsFile); err != nil {
		t.Fatalf("incidents file not created: %v", err)
	}
	data, _ := os.ReadFile(incidentsFile)
	if !strings.Contains(string(data), "args_hash") {
		t.Errorf("incident file missing args_hash field: %s", data)
	}
}

// TestDetectFromLog_Empty: no events → no loop.
func TestDetectFromLog_Empty(t *testing.T) {
	tmp := t.TempDir()
	SetSessionEventsPath(filepath.Join(tmp, "empty.jsonl"))
	t.Cleanup(func() { eventsPath = ".ovav/runtime/session_events.jsonl" })

	found, det, err := DetectFromLog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Errorf("empty log should not trigger: %+v", det)
	}
}

// TestDetectFromLog_AfterSession: simulate session that ended in loop,
// verify DetectFromLog catches it at next boot.
func TestDetectFromLog_AfterSession(t *testing.T) {
	tmp := t.TempDir()
	eventsFile := filepath.Join(tmp, "events.jsonl")
	incidentsFile := filepath.Join(tmp, "incidents.yaml")
	SetSessionEventsPath(eventsFile)
	incidentsPath = incidentsFile
	t.Cleanup(func() {
		eventsPath = ".ovav/runtime/session_events.jsonl"
		incidentsPath = ".ovav/registry/loop_incidents.yaml"
	})

	// 4 identical read calls in 2 seconds.
	for i := 0; i < 4; i++ {
		_, _ = RecordAndCheck("read", "foo.go", "same content", i)
		time.Sleep(100 * time.Millisecond)
	}
	found, det, err := DetectFromLog()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !found {
		t.Errorf("expected DetectFromLog to find in-progress loop")
	}
	if det.Tool != "read" {
		t.Errorf("expected tool=read, got %q", det.Tool)
	}
}

// TestParseArgs_StableOrder: same kv in different order → same hash.
func TestParseArgs_StableOrder(t *testing.T) {
	a := ParseArgs("a", "1", "b", "2", "c", "3")
	b := ParseArgs("c", "3", "a", "1", "b", "2")
	if a != b {
		t.Errorf("ParseArgs not stable: %q vs %q", a, b)
	}
	h1 := ArgsHash("bash", a)
	h2 := ArgsHash("bash", b)
	if h1 != h2 {
		t.Errorf("hashes differ for same args: %s vs %s", h1, h2)
	}
}
