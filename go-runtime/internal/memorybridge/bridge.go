// Package memorybridge provides cross-session memory persistence via SQLite.
//
// Top-tier 2026 innovation: each session gets a sqlite-backed memory bank
// that survives across sessions. Insights, decisions, errors, and notes
// are stored with full provenance (actor, timestamp, worktree, branch).
//
// Use cases:
//   - Long-running engineering: pick up where previous session left off
//   - Knowledge preservation: decisions survive across reinstalls
//   - Audit trail: every memory write has traceability
//
// Schema:
//
//	memories(id, ts, actor, kind, content, context, branch, worktree, tags)
//	links(from_id, to_id, relation)  -- for memory-graph linking
//
// Kind values: insight, decision, error, note, question, answer, todo
package memorybridge

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// MemoryKind classifies what kind of memory is being stored.
type MemoryKind string

const (
	KindInsight  MemoryKind = "insight"
	KindDecision MemoryKind = "decision"
	KindError    MemoryKind = "error"
	KindNote     MemoryKind = "note"
	KindQuestion MemoryKind = "question"
	KindAnswer   MemoryKind = "answer"
	KindTodo     MemoryKind = "todo"
)

// Memory represents a single memory entry.
type Memory struct {
	ID        string     `json:"id"`
	Timestamp time.Time  `json:"timestamp"`
	Actor     string     `json:"actor"`
	Kind      MemoryKind `json:"kind"`
	Content   string     `json:"content"`
	Context   string     `json:"context,omitempty"`
	Branch    string     `json:"branch,omitempty"`
	Worktree  string     `json:"worktree,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
}

// Bridge is the main memory bridge.
type Bridge struct {
	mu   sync.Mutex
	db   *sql.DB
	path string
}

// New opens or creates a memory bridge at the given path.
func New(path string) (*Bridge, error) {
	if path == "" {
		return nil, errors.New("memorybridge: path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("memorybridge: mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("memorybridge: open: %w", err)
	}
	b := &Bridge{db: db, path: path}
	if err := b.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("memorybridge: migrate: %w", err)
	}
	return b, nil
}

// migrate creates schema if not exists.
func (b *Bridge) migrate() error {
	_, err := b.db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			ts DATETIME NOT NULL,
			actor TEXT NOT NULL,
			kind TEXT NOT NULL,
			content TEXT NOT NULL,
			context TEXT,
			branch TEXT,
			worktree TEXT,
			tags TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_memories_kind ON memories(kind);
		CREATE INDEX IF NOT EXISTS idx_memories_actor ON memories(actor);
		CREATE INDEX IF NOT EXISTS idx_memories_ts ON memories(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_memories_branch ON memories(branch);
	`)
	return err
}

// Close releases the database.
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.db == nil {
		return nil
	}
	err := b.db.Close()
	b.db = nil
	return err
}

// Put stores a memory. ID is auto-generated from content hash if empty.
func (b *Bridge) Put(ctx context.Context, m Memory) (Memory, error) {
	return b.putValue(ctx, m)
}

func (b *Bridge) putValue(ctx context.Context, m Memory) (Memory, error) {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if m.ID == "" {
		m.ID = hashID(m.Actor, m.Kind, m.Content, m.Timestamp)
	}
	tagsJSON, _ := json.Marshal(m.Tags)

	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO memories (id, ts, actor, kind, content, context, branch, worktree, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Timestamp, string(m.Actor), string(m.Kind), m.Content, m.Context, m.Branch, m.Worktree, string(tagsJSON))
	if err != nil {
		return Memory{}, err
	}
	return m, nil
}

// Get retrieves a memory by ID.
func (b *Bridge) Get(ctx context.Context, id string) (*Memory, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	row := b.db.QueryRowContext(ctx,
		`SELECT id, ts, actor, kind, content, context, branch, worktree, tags FROM memories WHERE id = ?`,
		id)
	return scanMemory(row)
}

// List returns memories matching a filter, newest first.
func (b *Bridge) List(ctx context.Context, filter ListFilter) ([]Memory, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	q := `SELECT id, ts, actor, kind, content, context, branch, worktree, tags FROM memories WHERE 1=1`
	args := []interface{}{}
	if filter.Kind != "" {
		q += ` AND kind = ?`
		args = append(args, string(filter.Kind))
	}
	if filter.Actor != "" {
		q += ` AND actor = ?`
		args = append(args, filter.Actor)
	}
	if filter.Branch != "" {
		q += ` AND branch = ?`
		args = append(args, filter.Branch)
	}
	q += ` ORDER BY ts DESC LIMIT ?`
	args = append(args, filter.Limit)

	rows, err := b.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mems []Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			continue
		}
		mems = append(mems, *m)
	}
	return mems, nil
}

// ListFilter queries memories.
type ListFilter struct {
	Kind   MemoryKind
	Actor  string
	Branch string
	Limit  int
}

// Search finds memories containing the query string (case-insensitive).
func (b *Bridge) Search(ctx context.Context, query string, limit int) ([]Memory, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	rows, err := b.db.QueryContext(ctx,
		`SELECT id, ts, actor, kind, content, context, branch, worktree, tags FROM memories
		 WHERE content LIKE ? OR tags LIKE ?
		 ORDER BY ts DESC LIMIT ?`,
		"%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mems []Memory
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			continue
		}
		mems = append(mems, *m)
	}
	return mems, nil
}

// Count returns total memory count, optionally filtered by kind.
func (b *Bridge) Count(ctx context.Context, kind MemoryKind) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var count int
	var err error
	if kind == "" {
		err = b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories`).Scan(&count)
	} else {
		err = b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memories WHERE kind = ?`, string(kind)).Scan(&count)
	}
	return count, err
}

// Export dumps all memories to JSON.
func (b *Bridge) Export(ctx context.Context) ([]byte, error) {
	mems, err := b.List(ctx, ListFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(mems, "", "  ")
}

// ── helpers ───────────────────────────────────────────────────────────────

// hashID generates a deterministic ID from content + metadata.
func hashID(actor string, kind MemoryKind, content string, ts time.Time) string {
	h := sha256.New()
	h.Write([]byte(actor))
	h.Write([]byte{0})
	h.Write([]byte(kind))
	h.Write([]byte{0})
	h.Write([]byte(content))
	h.Write([]byte{0})
	h.Write([]byte(ts.Format(time.RFC3339Nano)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// scanMemory reads one row from the memories table.
func scanMemory(row interface {
	Scan(dest ...interface{}) error
}) (*Memory, error) {
	var (
		id, actor, kind, content, context, branch, worktree, tagsJSON string
		ts                                                            time.Time
	)
	if err := row.Scan(&id, &ts, &actor, &kind, &content, &context, &branch, &worktree, &tagsJSON); err != nil {
		return nil, err
	}
	m := &Memory{
		ID:        id,
		Timestamp: ts,
		Actor:     actor,
		Kind:      MemoryKind(kind),
		Content:   content,
		Context:   context,
		Branch:    branch,
		Worktree:  worktree,
	}
	if tagsJSON != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err == nil {
			m.Tags = tags
		}
	}
	return m, nil
}

// ValidKind checks if a kind is in the allowlist.
func ValidKind(k MemoryKind) bool {
	switch k {
	case KindInsight, KindDecision, KindError, KindNote,
		KindQuestion, KindAnswer, KindTodo:
		return true
	}
	return false
}

// AllKinds returns all valid kinds.
func AllKinds() []MemoryKind {
	return []MemoryKind{
		KindInsight, KindDecision, KindError, KindNote,
		KindQuestion, KindAnswer, KindTodo,
	}
}

// FormatSummary humanizes a memory list.
func FormatSummary(mems []Memory) string {
	if len(mems) == 0 {
		return "no memories"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d memories: ", len(mems)))
	for _, m := range mems {
		b.WriteString(fmt.Sprintf("[%s] %s; ", m.Kind, m.Content))
	}
	return b.String()
}
