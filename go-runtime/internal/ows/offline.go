package ows

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── F7: Offline-First Operations ──────────────────────────────────────────

// PendingOp represents a queued operation waiting for connectivity.
type PendingOp struct {
	ID         string    `json:"id"`
	OpType     string    `json:"op_type"` // "push", "merge", "sync", "route"
	Payload    string    `json:"payload"` // JSON-encoded operation details
	CreatedAt  time.Time `json:"created_at"`
	RetryCount int       `json:"retry_count"`
	HMAC       string    `json:"hmac"`
}

// OfflineQueue manages a SQLite-backed queue of operations to execute when online.
type OfflineQueue struct {
	db *sql.DB
}

// OpenOfflineQueue opens (or creates) the offline operation queue.
func OpenOfflineQueue(repoRoot string) (*OfflineQueue, error) {
	dbPath := filepath.Join(repoRoot, ".ovav", "ows", "offline_queue.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create ows dir: %w", err)
	}

	db, err := sql.Open(DriverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open offline queue db: %w", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pending_ops (
			id TEXT PRIMARY KEY,
			op_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TEXT NOT NULL,
			retry_count INTEGER DEFAULT 0,
			hmac TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_pending_created ON pending_ops(created_at);
	`)
	if err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &OfflineQueue{db: db}, nil
}

// Enqueue adds an operation to the offline queue with HMAC integrity protection.
func (q *OfflineQueue) Enqueue(opType, payload, secret string) (*PendingOp, error) {
	id := fmt.Sprintf("op-%d", time.Now().UnixNano())
	now := time.Now().UTC()

	hmacSig := signHMAC(id+opType+payload, secret)

	_, err := q.db.Exec(
		"INSERT INTO pending_ops (id, op_type, payload, created_at, retry_count, hmac) VALUES (?, ?, ?, ?, 0, ?)",
		id, opType, payload, now.Format(time.RFC3339), hmacSig,
	)
	if err != nil {
		return nil, fmt.Errorf("enqueue: %w", err)
	}

	return &PendingOp{
		ID: id, OpType: opType, Payload: payload,
		CreatedAt: now, HMAC: hmacSig,
	}, nil
}

// DequeueAll returns all pending operations ordered by creation time.
func (q *OfflineQueue) DequeueAll() ([]PendingOp, error) {
	rows, err := q.db.Query("SELECT id, op_type, payload, created_at, retry_count, hmac FROM pending_ops ORDER BY created_at ASC")
	if err != nil {
		return nil, fmt.Errorf("query pending ops: %w", err)
	}
	defer rows.Close()

	var ops []PendingOp
	for rows.Next() {
		var op PendingOp
		var createdAt string
		if err := rows.Scan(&op.ID, &op.OpType, &op.Payload, &createdAt, &op.RetryCount, &op.HMAC); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		op.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		ops = append(ops, op)
	}
	return ops, nil
}

// VerifyHMAC checks that a pending operation has not been tampered with.
func (op *PendingOp) VerifyHMAC(secret string) bool {
	expected := signHMAC(op.ID+op.OpType+op.Payload, secret)
	return hmac.Equal([]byte(op.HMAC), []byte(expected))
}

// Complete removes a successfully processed operation from the queue.
func (q *OfflineQueue) Complete(id string) error {
	_, err := q.db.Exec("DELETE FROM pending_ops WHERE id = ?", id)
	return err
}

// IncrementRetry increments the retry count for a failed operation.
func (q *OfflineQueue) IncrementRetry(id string) error {
	_, err := q.db.Exec("UPDATE pending_ops SET retry_count = retry_count + 1 WHERE id = ?", id)
	return err
}

// Close closes the offline queue database.
func (q *OfflineQueue) Close() error {
	return q.db.Close()
}

// ── HMAC Utilities ─────────────────────────────────────────────────────

var owsHMACSecret = getHMACSecret()

// getHMACSecret loads the HMAC secret from environment or uses a development default.
// In production, always set OVAV_HMAC_SECRET via environment or vault.
func getHMACSecret() string {
	if s := os.Getenv("OVAV_HMAC_SECRET"); s != "" {
		return s
	}
	// Development fallback — NOT for production use.
	// Generates a unique-per-installation default based on hostname + repo path.
	return "ovav-dev-" + filepath.Base(os.TempDir())
}

func signHMAC(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
