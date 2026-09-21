// Package main — append-only audit log for every bridge operation.
//
// Stored as .ovav/registry/audit/sheets/YYYY-MM-DD.jsonl (one line
// per call). Each line has actor, op, before/after fingerprint,
// spreadsheet/tab, ts, and a stable hash chain entry.
//
// The audit log is plaintext JSONL — readable by the CEO without
// keys. Sensitive payloads stay in encrypted snapshots, not here.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AuditEvent is one line in the JSONL log.
type AuditEvent struct {
	Timestamp     string         `json:"ts"`
	Actor         string         `json:"actor"`
	Operation     string         `json:"op"`
	SpreadsheetID string         `json:"spreadsheet_id"`
	Tab           string         `json:"tab,omitempty"`
	Args          map[string]any `json:"args,omitempty"`
	Affected      int            `json:"affected"`
	Hash          string         `json:"hash"`
	PrevHash      string         `json:"prev_hash"`
}

var lastAuditHash = ""

// audit appends one event. actor is hardcoded for now but the field
// is plumbed so multi-agent setups can populate it later.
func audit(repoRoot, sid, tab, op string, args map[string]any, affected int) {
	root := filepath.Join(repoRoot, ".ovav", "registry", "audit", "sheets")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return
	}
	file := filepath.Join(root, time.Now().UTC().Format("2006-01-02")+".jsonl")
	ev := AuditEvent{
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Actor:         actor(),
		Operation:     op,
		SpreadsheetID: sid,
		Tab:           tab,
		Args:          args,
		Affected:      affected,
		PrevHash:      lastAuditHash,
	}
	body, _ := json.Marshal(ev)
	h := sha256.Sum256(append([]byte(lastAuditHash), body...))
	ev.Hash = fmt.Sprintf("%x", h[:])
	lastAuditHash = ev.Hash
	line, _ := json.Marshal(ev)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

func actor() string {
	if v := os.Getenv("OVAV_ACTOR"); v != "" {
		return v
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "ovav-sheets"
}
