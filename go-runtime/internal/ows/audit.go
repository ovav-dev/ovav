package ows

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ── SQLite Audit ─────────────────────────────────────────────────────────────────

// AuditDB wraps a SQLite database for OWS audit logging.
// Uses modernc.org/sqlite (pure Go, no CGO).
type AuditDB struct {
	db *sql.DB
}

// OpenAudit opens or creates the OWS audit database at the given OVAV root.
// The database is stored at .ovav/ows/audit.db.
func OpenAudit(ovavRoot string) (*AuditDB, error) {
	dbDir := filepath.Join(ovavRoot, ".ovav", "ows")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("ows: create audit dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "audit.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("ows: open audit db: %w", err)
	}

	audit := &AuditDB{db: db}
	if err := audit.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ows: migrate audit db: %w", err)
	}

	return audit, nil
}

// migrate creates the schema if it doesn't exist.
func (a *AuditDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp   TEXT NOT NULL,
		actor       TEXT NOT NULL,
		command     TEXT NOT NULL,
		target      TEXT NOT NULL,
		result      TEXT NOT NULL,
		metadata    TEXT,
		perf_ms     INTEGER
	);

	CREATE TABLE IF NOT EXISTS worktree_state (
		id          TEXT PRIMARY KEY,
		branch      TEXT NOT NULL,
		profile     TEXT NOT NULL,
		owner       TEXT NOT NULL,
		state       TEXT NOT NULL,
		policy_ver  INTEGER DEFAULT 1,
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL,
		locked      INTEGER DEFAULT 0,
		lock_reason TEXT
	);

	CREATE TABLE IF NOT EXISTS policy_versions (
		policy_id   TEXT NOT NULL,
		version     INTEGER NOT NULL DEFAULT 1,
		rule        TEXT NOT NULL,
		created_at  TEXT NOT NULL,
		author      TEXT NOT NULL,
		PRIMARY KEY (policy_id, version)
	);

	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_command ON audit_log(command);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor);
	CREATE INDEX IF NOT EXISTS idx_worktree_owner ON worktree_state(owner);
	CREATE INDEX IF NOT EXISTS idx_worktree_state ON worktree_state(state);
	`
	_, err := a.db.Exec(schema)
	return err
}

// ── Audit Log ────────────────────────────────────────────────────────────────────

// LogEntry represents a single audit record.
type LogEntry struct {
	Timestamp time.Time
	Actor     string
	Command   string
	Target    string
	Result    string
	Metadata  string // JSON
	PerfMs    int64
}

// Log records an audit entry.
func (a *AuditDB) Log(entry LogEntry) error {
	_, err := a.db.Exec(
		`INSERT INTO audit_log (timestamp, actor, command, target, result, metadata, perf_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp.Format(time.RFC3339),
		entry.Actor,
		entry.Command,
		entry.Target,
		entry.Result,
		entry.Metadata,
		entry.PerfMs,
	)
	return err
}

// QueryLogs returns audit entries filtered by command and/or actor.
func (a *AuditDB) QueryLogs(command, actor string, limit int) ([]LogEntry, error) {
	query := "SELECT timestamp, actor, command, target, result, COALESCE(metadata,''), COALESCE(perf_ms,0) FROM audit_log WHERE 1=1"
	var args []interface{}

	if command != "" {
		query += " AND command = ?"
		args = append(args, command)
	}
	if actor != "" {
		query += " AND actor = ?"
		args = append(args, actor)
	}

	query += " ORDER BY timestamp DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		var ts string
		if err := rows.Scan(&ts, &e.Actor, &e.Command, &e.Target, &e.Result, &e.Metadata, &e.PerfMs); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ── Worktree State Persistence ───────────────────────────────────────────────────

// SaveWorktree persists a worktree record to the database.
func (a *AuditDB) SaveWorktree(wt WorktreeRecord) error {
	_, err := a.db.Exec(
		`INSERT OR REPLACE INTO worktree_state
		 (id, branch, profile, owner, state, policy_ver, created_at, updated_at, locked, lock_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wt.ID, wt.Branch, wt.Profile, wt.Owner,
		string(wt.State), wt.PolicyVer,
		wt.CreatedAt.Format(time.RFC3339),
		wt.UpdatedAt.Format(time.RFC3339),
		boolToInt(wt.Locked), wt.LockReason,
	)
	return err
}

// LoadWorktree loads a worktree record by ID.
func (a *AuditDB) LoadWorktree(id string) (*WorktreeRecord, error) {
	var wt WorktreeRecord
	var stateStr, createdAt, updatedAt string
	var locked int

	err := a.db.QueryRow(
		`SELECT id, branch, profile, owner, state, policy_ver, created_at, updated_at, locked, COALESCE(lock_reason,'')
		 FROM worktree_state WHERE id = ?`, id,
	).Scan(&wt.ID, &wt.Branch, &wt.Profile, &wt.Owner,
		&stateStr, &wt.PolicyVer, &createdAt, &updatedAt, &locked, &wt.LockReason)

	if err != nil {
		return nil, err
	}

	wt.State = State(stateStr)
	wt.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	wt.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	wt.Locked = locked == 1

	return &wt, nil
}

// ListWorktrees returns all worktrees, optionally filtered by state or owner.
func (a *AuditDB) ListWorktrees(state State, owner string) ([]WorktreeRecord, error) {
	query := "SELECT id, branch, profile, owner, state, policy_ver, created_at, updated_at, locked, COALESCE(lock_reason,'') FROM worktree_state WHERE 1=1"
	var args []interface{}

	if state != "" {
		query += " AND state = ?"
		args = append(args, string(state))
	}
	if owner != "" {
		query += " AND owner = ?"
		args = append(args, owner)
	}

	query += " ORDER BY updated_at DESC"

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wts []WorktreeRecord
	for rows.Next() {
		var wt WorktreeRecord
		var stateStr, createdAt, updatedAt string
		var locked int
		if err := rows.Scan(&wt.ID, &wt.Branch, &wt.Profile, &wt.Owner,
			&stateStr, &wt.PolicyVer, &createdAt, &updatedAt, &locked, &wt.LockReason); err != nil {
			return nil, err
		}
		wt.State = State(stateStr)
		wt.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		wt.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		wt.Locked = locked == 1
		wts = append(wts, wt)
	}
	return wts, rows.Err()
}

// PruneStale removes worktrees with state CLEANED older than the given duration.
func (a *AuditDB) PruneStale(maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge).Format(time.RFC3339)
	result, err := a.db.Exec(
		`DELETE FROM worktree_state WHERE state = ? AND updated_at < ?`,
		string(StateCleaned), cutoff,
	)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// ── Policy Versions ──────────────────────────────────────────────────────────────

// SavePolicy records a policy version.
func (a *AuditDB) SavePolicy(policyID string, version int, rule, author string) error {
	_, err := a.db.Exec(
		`INSERT OR REPLACE INTO policy_versions (policy_id, version, rule, created_at, author)
		 VALUES (?, ?, ?, ?, ?)`,
		policyID, version, rule, time.Now().Format(time.RFC3339), author,
	)
	return err
}

// GetPolicyVersion returns the latest version of a policy.
func (a *AuditDB) GetPolicyVersion(policyID string) (int, string, error) {
	var version int
	var rule string
	err := a.db.QueryRow(
		`SELECT version, rule FROM policy_versions WHERE policy_id = ? ORDER BY version DESC LIMIT 1`,
		policyID,
	).Scan(&version, &rule)
	if err != nil {
		return 0, "", err
	}
	return version, rule, nil
}

// ── Lifecycle ────────────────────────────────────────────────────────────────────

// Close closes the database connection.
func (a *AuditDB) Close() error {
	return a.db.Close()
}

// ── Helpers ──────────────────────────────────────────────────────────────────────

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
