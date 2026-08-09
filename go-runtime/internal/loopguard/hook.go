// Package loopguard — session-greeting hook integration.
//
// This file adds:
//   - RecordAndCheck(): idempotent, thread-safe counter for tool calls
//   - SessionEvents: append-only JSONL log of tool calls in current session
//   - DetectFromLog(): runs Detect() over the on-disk session log
//
// Wire-up path: session_greeting calls CheckOnStartup() at boot; agent
// runtime (or hook layer) calls RecordAndCheck() before each tool.
// Both paths are independent — the runtime hook is the authoritative gate;
// the session greeting check is the recovery path (if loop already in progress).
package loopguard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EventV2 is the wire format for SessionEvents JSONL. Distinct from Event
// in loopguard.go (which is in-memory only) to keep the persisted schema
// stable and additive.
type EventV2 struct {
	SessionID  string    `json:"session_id"`
	Turn       int       `json:"turn"`
	Tool       string    `json:"tool"`
	ArgsHash   string    `json:"args_hash"`
	OutputHash string    `json:"output_hash,omitempty"`
	At         time.Time `json:"at"`
}

var (
	eventsMu sync.Mutex
	// sessionID is set once per session by SetSessionID. If empty, the
	// RecordAndCheck path uses "unknown" (still useful — at least the count works).
	sessionID = ""
)

// SetSessionID initializes the session-scoped ID. Call from session_greeting
// or any host init code before any tool invocation.
func SetSessionID(id string) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	sessionID = id
}

// ArgsHash returns a short stable hash of the tool's argument string.
// Tools with the same name + same args produce the same hash, which is the
// trigger for loop detection.
func ArgsHash(tool, args string) string {
	h := sha256.Sum256([]byte(tool + "\x00" + args))
	return hex.EncodeToString(h[:8])
}

// OutputHash returns a short stable hash of tool output. Used to detect
// "same output" loops (different args, same result).
func OutputHash(output string) string {
	// Structural fingerprint — first 256 + last 256 chars + length.
	if len(output) > 512 {
		output = output[:256] + output[len(output)-256:]
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("len=%d|", len(output)) + output))
	return hex.EncodeToString(h[:8])
}

// DefaultEventsPath is where session events are persisted.
// SetSessionEventsPath can override at runtime (e.g. session_greeting injects
// a per-session file).
var eventsPath = ".ovav/runtime/session_events.jsonl"

func SetSessionEventsPath(p string) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	eventsPath = p
}

func appendEvent(ev EventV2) error {
	eventsMu.Lock()
	p := eventsPath
	sid := sessionID
	eventsMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir events dir: %w", err)
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open events: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if sid != "" {
		ev.SessionID = sid
	}
	return enc.Encode(ev)
}

// RecordAndCheck is the gate function. Call BEFORE every tool invocation.
// Returns the Detection (if any) and the loop result code:
//   - rc=0: not a loop, record event and continue
//   - rc=1: loop detected, return detection with strategy recommendation
//   - rc=2: tool dispatch error (file write failure)
//
// This function is the authoritative runtime gate; the session-greeting
// startup check is a fallback that runs DetectFromLog() at session boot.
func RecordAndCheck(tool, args, output string, turn int) (Detection, int) {
	ah := ArgsHash(tool, args)
	oh := OutputHash(output)
	now := time.Now().UTC()

	ev := EventV2{
		Turn:       turn,
		Tool:       tool,
		ArgsHash:   ah,
		OutputHash: oh,
		At:         now,
	}
	if err := appendEvent(ev); err != nil {
		return Detection{}, 2
	}

	// Sliding window: read recent events, build Event list, run Detect.
	events, err := LoadRecentEvents(50)
	if err != nil {
		return Detection{}, 2
	}
	// Convert EventV2 → Event (Detect signature).
	internalEvents := make([]Event, 0, len(events))
	for _, e := range events {
		internalEvents = append(internalEvents, Event{
			Turn:   e.Turn,
			Tool:   e.Tool,
			Output: e.OutputHash, // Detect uses Output field for hash; we pass hash
			At:     e.At,
		})
	}
	det := Detect(internalEvents, DefaultConfig)
	if det.Detected {
		// Auto-log incident.
		_ = appendIncident(det, tool, ah, turn)
		return det, 1
	}
	return Detection{}, 0
}

// LoadRecentEvents returns the last N events from the session log.
func LoadRecentEvents(n int) ([]EventV2, error) {
	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var all []EventV2
	for dec.More() {
		var e EventV2
		if err := dec.Decode(&e); err != nil {
			return nil, err
		}
		all = append(all, e)
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// DetectFromLog is the session-greeting startup check. Loads recent events
// and runs Detect(). Returns true if a loop is in progress.
func DetectFromLog() (bool, Detection, error) {
	events, err := LoadRecentEvents(50)
	if err != nil {
		return false, Detection{}, err
	}
	if len(events) == 0 {
		return false, Detection{}, nil
	}
	internal := make([]Event, 0, len(events))
	for _, e := range events {
		internal = append(internal, Event{
			Turn:   e.Turn,
			Tool:   e.Tool,
			Output: e.OutputHash,
			At:     e.At,
		})
	}
	det := Detect(internal, DefaultConfig)
	return det.Detected, det, nil
}

// incidentsPath is where auto-detected loop incidents are persisted.
var incidentsPath = ".ovav/registry/loop_incidents.yaml"

func appendIncident(det Detection, tool, argsHash string, turn int) error {
	f, err := os.OpenFile(incidentsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	_, _ = fmt.Fprintf(f, `
- timestamp: %q
  tool: %q
  args_hash: %q
  turn: %d
  occurrences: %d
  window_seconds: %.2f
  suggested_action: %q
`, ts, tool, argsHash, turn, det.Occurrences, det.WindowSeconds, det.SuggestedAction)
	return nil
}

// String formats a Detection for human/log output.
func (d Detection) String() string {
	if !d.Detected {
		return "no loop detected"
	}
	return fmt.Sprintf("LOOP: tool=%s hash=%s occurrences=%d window=%.2fs",
		d.Tool, d.OutputHash, d.Occurrences, d.WindowSeconds)
}

// ParseArgs builds a stable args representation from key=value pairs.
// Used by RecordAndCheck to produce deterministic hashes.
func ParseArgs(kv ...string) string {
	pairs := make([]string, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		pairs = append(pairs, kv[i]+"="+kv[i+1])
	}
	// Stable order for hashing.
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j-1] > pairs[j]; j-- {
			pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
		}
	}
	return strings.Join(pairs, "&")
}

// Helper to convert int → string (used in tests).
func itoa(i int) string { return strconv.Itoa(i) }
