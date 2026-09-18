// Package main — snapshot + rollback for the OVAV Sheets bridge.
//
// Every destructive operation captures the BEFORE state into
// .ovav/vault/snapshots/<spreadsheet>/<tab>/<timestamp>.json.gz
// (encrypted with the same vault key as credentials). `ovav-sheets
// rollback <id>` restores from a snapshot.
//
// Storage layout:
//
//	.ovav/vault/snapshots/
//	├── INDEX.jsonl                          (one line per snapshot)
//	└── <spreadsheetID>/
//	    └── <tab>/
//	        └── <timestamp>.json.gz          (encrypted value snapshot)
//
// Snapshots are append-only. Rollback copies the snapshot BACK to
// Sheets, never deletes the source.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/vault"
)

// SnapshotRecord is one entry in INDEX.jsonl.
type SnapshotRecord struct {
	ID            string    `json:"id"`
	SpreadsheetID string    `json:"spreadsheet_id"`
	Tab           string    `json:"tab"`
	Operation     string    `json:"operation"`
	Summary       string    `json:"summary"`
	CellCount     int       `json:"cell_count"`
	CreatedAt     time.Time `json:"created_at"`
	HashBefore    string    `json:"hash_before"`
	HashAfter     string    `json:"hash_after"`
	Path          string    `json:"path"`
}

// SnapshotPath returns the directory where snapshots for (sid, tab)
// live. repoRoot is typically the result of findRepoRoot().
func snapshotPath(repoRoot, sid, tab string) string {
	return filepath.Join(repoRoot, ".ovav", "vault", "snapshots",
		sanitize(sid), sanitize(tab))
}

func indexPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".ovav", "vault", "snapshots", "INDEX.jsonl")
}

// Capture takes a snapshot of the given range and returns the record.
// payload is opaque bytes the caller chose to persist (typically the
// raw JSON values). The hash chain links the BEFORE state to AFTER
// (set by the caller via record.HashAfter once the new state is known).
func (cl *Client) Capture(repoRoot, op, summary, tab string, payload []byte) (*SnapshotRecord, error) {
	ts := time.Now().UTC()
	idSum := sha256.Sum256([]byte(op + tab + ts.String()))
	id := fmt.Sprintf("%s-%s-%x", op, ts.Format("20060102T150405Z"), idSum[:3])
	id = sanitize(id)
	dir := snapshotPath(repoRoot, cl.spreadsheetID, tab)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("snapshot: mkdir: %w", err)
	}
	// Encrypt the payload via vault. Even though Sheets stores are
	// already in Google's cloud, we keep snapshots encrypted at rest
	// so a leaked vault directory is useless without vault.key.
	key, err := loadVaultKey(repoRoot)
	if err != nil {
		return nil, err
	}
	ct, err := vault.Encrypt(payload, key)
	if err != nil {
		return nil, fmt.Errorf("snapshot: encrypt: %w", err)
	}
	// gzip the ciphertext — saves 70%+ on large range dumps.
	var gzbuf bytes.Buffer
	gz := gzip.NewWriter(&gzbuf)
	if _, err := gz.Write(ct); err != nil {
		return nil, fmt.Errorf("snapshot: gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	rel := filepath.Join(sanitize(cl.spreadsheetID), sanitize(tab), id+".json.gz")
	abs := filepath.Join(repoRoot, ".ovav", "vault", "snapshots", rel)
	if err := os.WriteFile(abs, gzbuf.Bytes(), 0o600); err != nil {
		return nil, fmt.Errorf("snapshot: write: %w", err)
	}
	hashBefore := sha256.Sum256(payload)
	rec := &SnapshotRecord{
		ID:            id,
		SpreadsheetID: cl.spreadsheetID,
		Tab:           tab,
		Operation:     op,
		Summary:       summary,
		CellCount:     countCells(payload),
		CreatedAt:     ts,
		HashBefore:    fmt.Sprintf("%x", hashBefore[:]),
		Path:          rel,
	}
	if err := appendIndex(repoRoot, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// LoadSnapshot returns the decrypted payload bytes for a record.
func LoadSnapshot(repoRoot string, rec *SnapshotRecord) ([]byte, error) {
	abs := filepath.Join(repoRoot, ".ovav", "vault", "snapshots", rec.Path)
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("snapshot: read %s: %w", abs, err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("snapshot: gzip reader: %w", err)
	}
	ct, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("snapshot: read gz: %w", err)
	}
	key, err := loadVaultKey(repoRoot)
	if err != nil {
		return nil, err
	}
	return vault.Decrypt(ct, key)
}

// ListSnapshots returns every snapshot, newest first, optionally
// filtered by spreadsheet ID and tab.
func ListSnapshots(repoRoot, sid, tab string) ([]SnapshotRecord, error) {
	indexFile := indexPath(repoRoot)
	var recs []SnapshotRecord
	if data, err := os.ReadFile(indexFile); err == nil {
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
			if len(line) == 0 {
				continue
			}
			var r SnapshotRecord
			if err := json.Unmarshal(line, &r); err != nil {
				continue
			}
			if sid != "" && r.SpreadsheetID != sid {
				continue
			}
			if tab != "" && r.Tab != tab {
				continue
			}
			recs = append(recs, r)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt.After(recs[j].CreatedAt) })
	return recs, nil
}

// Rollback restores a snapshot by writing the captured values back
// to Sheets. The operation creates a NEW snapshot first (so the
// rollback itself is reversible).
func Rollback(repoRoot, id string) error {
	recs, err := ListSnapshots(repoRoot, "", "")
	if err != nil {
		return err
	}
	var rec *SnapshotRecord
	for i := range recs {
		if recs[i].ID == id {
			rec = &recs[i]
			break
		}
	}
	if rec == nil {
		return fmt.Errorf("snapshot %q not found", id)
	}
	payload, err := LoadSnapshot(repoRoot, rec)
	if err != nil {
		return err
	}
	var vr ValueRange
	if err := json.Unmarshal(payload, &vr); err != nil {
		return fmt.Errorf("snapshot: parse payload: %w", err)
	}
	store := NewCredStore(repoRoot)
	creds, err := store.Load()
	if err != nil {
		return err
	}
	cl := NewClient(creds, rec.SpreadsheetID)
	if err := assertAllowed(rec.SpreadsheetID); err != nil {
		return err
	}
	// Capture pre-rollback state.
	pre, _ := cl.ReadRange(vr.Range)
	preBytes, _ := json.Marshal(pre)
	_, _ = cl.Capture(repoRoot, "rollback-pre", "rollback to "+id, rec.Tab, preBytes)
	ur, err := cl.WriteValues(vr.Range, vr.Values, "USER_ENTERED")
	if err != nil {
		return err
	}
	// Capture post-rollback state, linking HashBefore->HashAfter.
	post, _ := cl.ReadRange(vr.Range)
	postBytes, _ := json.Marshal(post)
	postRec, err := cl.Capture(repoRoot, "rollback-post", "after rollback to "+id, rec.Tab, postBytes)
	if err != nil {
		return fmt.Errorf("snapshot: post-rollback capture: %w", err)
	}
	postRec.HashBefore = fmt.Sprintf("%x", sha256.Sum256(preBytes))
	if err := patchIndexHashAfter(repoRoot, postRec.ID, postRec.HashBefore, fmt.Sprintf("%x", sha256.Sum256(postBytes))); err != nil {
		return err
	}
	fmt.Printf("✅ rolled back %d cells at %s (snapshot %s)\n",
		ur.UpdatedCells, ur.UpdatedRange, id)
	return nil
}

func appendIndex(repoRoot string, rec *SnapshotRecord) error {
	if err := os.MkdirAll(filepath.Dir(indexPath(repoRoot)), 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(indexPath(repoRoot), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func patchIndexHashAfter(repoRoot, id, before, after string) error {
	data, err := os.ReadFile(indexPath(repoRoot))
	if err != nil {
		return err
	}
	out := bytes.Buffer{}
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var r SnapshotRecord
		if err := json.Unmarshal(line, &r); err == nil && r.ID == id {
			r.HashBefore = before
			r.HashAfter = after
			line, _ = json.Marshal(r)
		}
		out.Write(line)
		out.WriteByte('\n')
	}
	return os.WriteFile(indexPath(repoRoot), out.Bytes(), 0o600)
}

func countCells(payload []byte) int {
	var vr ValueRange
	if err := json.Unmarshal(payload, &vr); err != nil {
		return 0
	}
	n := 0
	for _, r := range vr.Values {
		n += len(r)
	}
	return n
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			out = append(out, c)
		case c == '-' || c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

// loadVaultKey duplicates credStore.loadKey for callers that don't
// have a credStore handy (snapshot code paths). Returns []byte.
func loadVaultKey(repoRoot string) ([]byte, error) {
	kp := filepath.Join(repoRoot, ".ovav", "vault", "vault.key")
	if k, err := os.ReadFile(kp); err == nil {
		if len(k) != vault.KeySize {
			return nil, fmt.Errorf("vault.key size %d, want %d", len(k), vault.KeySize)
		}
		return k, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	k, err := vault.GenerateKey()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(kp), 0o700); err != nil {
		return nil, err
	}
	return k, os.WriteFile(kp, k, 0o600)
}

// Helper used by Table.UpdateWhere to materialize snapshots.
var _ = strings.TrimSpace
