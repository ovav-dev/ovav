package ows

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// ── F9: Event Bus & Multi-Agent Sync ──────────────────────────────────────

// EventType categorizes system events emitted by OWS.
type EventType string

const (
	EvtWorktreeCreated     EventType = "WORKTREE_CREATED"
	EvtWorktreeLocked      EventType = "WORKTREE_LOCKED"
	EvtWorktreeUnlocked    EventType = "WORKTREE_UNLOCKED"
	EvtConflictDetected    EventType = "CONFLICT_DETECTED"
	EvtConflictResolved    EventType = "CONFLICT_RESOLVED"
	EvtIntegrationComplete EventType = "INTEGRATION_COMPLETE"
	EvtCleanupComplete     EventType = "CLEANUP_COMPLETE"
	EvtOperationQueued     EventType = "OPERATION_QUEUED"
	EvtOperationFlushed    EventType = "OPERATION_FLUSHED"
	EvtStateTransition     EventType = "STATE_TRANSITION"
)

// BusEvent represents a system event with metadata for pub/sub distribution.
type BusEvent struct {
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Source    string         `json:"source"` // agent or process name
	Payload   map[string]any `json:"payload"`
}

// EventHandler is a callback for event subscribers.
type EventHandler func(event BusEvent)

// EventBus provides pub/sub event distribution with SQLite-backed durability.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]EventHandler
	db          *sql.DB
}

// NewEventBus creates a new event bus with optional SQLite persistence.
func NewEventBus(repoRoot string) (*EventBus, error) {
	bus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
	}

	// Setup SQLite durability
	dbPath := filepath.Join(repoRoot, ".ovav", "ows", "events.db")
	db, err := sql.Open(DriverName, dbPath)
	if err != nil {
		return bus, nil // Non-fatal: operate without durability
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			source TEXT NOT NULL,
			payload TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_events_type_ts ON events(type, timestamp);
	`)
	if err != nil {
		db.Close()
		return bus, nil
	}

	bus.db = db
	return bus, nil
}

// Subscribe registers a handler for a specific event type.
func (b *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

// Emit publishes an event to all subscribers and persists to SQLite.
func (b *EventBus) Emit(event BusEvent) {
	event.Timestamp = time.Now().UTC()

	// Persist to SQLite
	if b.db != nil {
		payload := "{}"
		if event.Payload != nil && len(event.Payload) > 0 {
			if jsonBytes, err := json.Marshal(event.Payload); err == nil {
				payload = string(jsonBytes)
			}
		}
		_, _ = b.db.Exec(
			"INSERT INTO events (type, timestamp, source, payload) VALUES (?, ?, ?, ?)",
			string(event.Type), event.Timestamp.Format(time.RFC3339), event.Source, payload,
		)
	}

	// Notify subscribers
	b.mu.RLock()
	handlers := b.subscribers[event.Type]
	b.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event) // async notification
	}
}

// EmitSync publishes an event synchronously (blocks until all handlers complete).
func (b *EventBus) EmitSync(event BusEvent) {
	event.Timestamp = time.Now().UTC()

	// Persist
	if b.db != nil {
		_, _ = b.db.Exec(
			"INSERT INTO events (type, timestamp, source, payload) VALUES (?, ?, ?, ?)",
			string(event.Type), event.Timestamp.Format(time.RFC3339), event.Source, "{}",
		)
	}

	b.mu.RLock()
	handlers := b.subscribers[event.Type]
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

// QueryEvents returns persisted events filtered by type and time range.
func (b *EventBus) QueryEvents(eventType EventType, since time.Duration) ([]BusEvent, error) {
	if b.db == nil {
		return nil, fmt.Errorf("event bus has no SQLite backend")
	}

	cutoff := time.Now().UTC().Add(-since)
	rows, err := b.db.Query(
		"SELECT type, timestamp, source, payload FROM events WHERE type = ? AND timestamp >= ? ORDER BY timestamp DESC LIMIT 100",
		string(eventType), cutoff.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []BusEvent
	for rows.Next() {
		var e BusEvent
		var ts, payload string
		if err := rows.Scan(&e.Type, &ts, &e.Source, &payload); err != nil {
			continue
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		events = append(events, e)
	}
	return events, nil
}

// Close closes the event bus and its SQLite connection.
func (b *EventBus) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// ── Multi-Agent Coordination ──────────────────────────────────────────────

// AgentLock provides distributed locking for multi-agent worktree coordination.
type AgentLock struct {
	Worktree   string    `json:"worktree"`
	Owner      string    `json:"owner"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Reason     string    `json:"reason"`
}

// LockAcquired checks if a worktree is currently locked by any agent.
func (l *AgentLock) IsExpired() bool {
	return time.Now().UTC().After(l.ExpiresAt)
}
