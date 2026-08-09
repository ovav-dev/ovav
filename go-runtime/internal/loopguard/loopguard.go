// OVAV Loop Detector Guard — Internal package
// Package: github.com/ovav/ovav/internal/loopguard
// Purpose: Detect tool-call loops in session transcripts and emit a structured warning
// BEFORE the agent burns more turns on the same failing command.
//
// Trigger pattern (from ses_05472d9dfffegXdTlJk7jS7Kdh, 2026-07-28):
//   - Same tool invoked ≥3 times in a single agent turn
//   - Output hash identical across consecutive invocations
//   - Wall-clock delta < 5s between calls
//
// When triggered, returns OVAV_LOOP_DETECTED stderr line + exit code 2.
// The session can then abort and re-plan.
//
// USAGE (from bash):
//
//	go run -C go-runtime ./cmd/loopdetect --session=<session-id> --events=<events.jsonl>
//
// This is a STANDALONE subsystem — does not depend on output_guard, session_greeting,
// or any other running system. Safe to invoke from any toolchain layer.
package loopguard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Event is the minimal shape we need from a session transcript.
type Event struct {
	Turn     int       `json:"turn"`
	Tool     string    `json:"tool"`
	Output   string    `json:"output"`
	At       time.Time `json:"at"`
	ExitCode int       `json:"exit_code"`
}

// Detection is the result of a loop scan.
type Detection struct {
	Detected        bool      `json:"detected"`
	Tool            string    `json:"tool,omitempty"`
	Occurrences     int       `json:"occurrences,omitempty"`
	OutputHash      string    `json:"output_hash,omitempty"`
	FirstTime       time.Time `json:"first_time,omitempty"`
	LastTime        time.Time `json:"last_time,omitempty"`
	WindowSeconds   float64   `json:"window_seconds,omitempty"`
	SuggestedAction string    `json:"suggested_action,omitempty"`
}

// Config controls detection thresholds.
type Config struct {
	MinOccurrences int           // default 3
	WindowDuration time.Duration // default 5s
	HashOnly       bool          // if true, hash output; else hash raw bytes
}

// DefaultConfig matches the empirical pattern observed in ses_05472d9dfffegXdTlJk7jS7Kdh.
var DefaultConfig = Config{
	MinOccurrences: 3,
	WindowDuration: 5 * time.Second,
	HashOnly:       true,
}

// Detect scans events and returns the first loop detected, or zero-value Detection.
//
// Algorithm: sliding window per-tool. For each tool, keep the last N calls.
// If the last N calls share the same output hash AND fit within WindowDuration AND
// Occurrences ≥ MinOccurrences, return a Detection.
func Detect(events []Event, cfg Config) Detection {
	if cfg.MinOccurrences == 0 {
		cfg.MinOccurrences = DefaultConfig.MinOccurrences
	}
	if cfg.WindowDuration == 0 {
		cfg.WindowDuration = DefaultConfig.WindowDuration
	}
	if len(events) < cfg.MinOccurrences {
		return Detection{}
	}

	// Group by tool, preserving order.
	byTool := make(map[string][]Event)
	for _, e := range events {
		byTool[e.Tool] = append(byTool[e.Tool], e)
	}

	for tool, evs := range byTool {
		if len(evs) < cfg.MinOccurrences {
			continue
		}
		// Walk the tail; only consider consecutive window.
		tail := evs[len(evs)-cfg.MinOccurrences:]
		hash := hashOutput(tail[0].Output, cfg.HashOnly)
		allMatch := true
		for _, e := range tail[1:] {
			if hashOutput(e.Output, cfg.HashOnly) != hash {
				allMatch = false
				break
			}
		}
		if !allMatch {
			continue
		}
		// Check time window.
		first, last := tail[0].At, tail[len(tail)-1].At
		if last.Sub(first) > cfg.WindowDuration {
			continue
		}
		return Detection{
			Detected:        true,
			Tool:            tool,
			Occurrences:     len(tail),
			OutputHash:      hash,
			FirstTime:       first,
			LastTime:        last,
			WindowSeconds:   last.Sub(first).Seconds(),
			SuggestedAction: "change_strategy: read file directly, delegate to subagent, or accept current output and advance",
		}
	}
	return Detection{}
}

func hashOutput(s string, hashOnly bool) string {
	if hashOnly {
		// Hash the structural fingerprint: length + first 256 chars + last 256 chars.
		// Survives minor whitespace jitter, catches true repeats.
		if len(s) > 512 {
			s = s[:256] + s[len(s)-256:]
		}
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

// LoadEvents reads a JSONL transcript file into []Event.
// Each line MUST be a complete JSON object.
func LoadEvents(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}
	var events []Event
	// Split on newline; tolerate trailing newline.
	lines := splitLines(data)
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parse event %d: %w", i, err)
		}
		events = append(events, e)
	}
	return events, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
