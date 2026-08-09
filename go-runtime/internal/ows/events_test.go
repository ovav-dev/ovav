package ows

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// events_test.go — Tests for events.go (Event Bus & Multi-Agent Sync)
// ═══════════════════════════════════════════════════════════════════════════

// ── Test Helpers ─────────────────────────────────────────────────────────────

// setupEventBusDir creates a temp dir with .ovav/ows/ for SQLite.
func setupEventBusDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "ows"), 0755)
	return dir
}

// ═══════════════════════════════════════════════════════════════════════════
// NewEventBus Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestNewEventBus_CreatesDB(t *testing.T) {
	dir := setupEventBusDir(t)

	bus, err := NewEventBus(dir)
	if err != nil {
		t.Fatalf("NewEventBus: %v", err)
	}
	defer bus.Close()

	if bus == nil {
		t.Fatal("bus should not be nil")
	}
	if bus.db == nil {
		t.Error("SQLite DB should be initialized")
	}
	if bus.subscribers == nil {
		t.Error("subscribers map should be initialized")
	}

	// Verify DB file exists
	dbPath := filepath.Join(dir, ".ovav", "ows", "events.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("events.db should exist")
	}
}

func TestNewEventBus_NoOwavDir(t *testing.T) {
	// Without .ovav/ows/ dir, SQLite open may fail but bus should still work
	dir := t.TempDir()

	bus, err := NewEventBus(dir)
	if err != nil {
		t.Fatalf("NewEventBus should not error: %v", err)
	}
	defer bus.Close()

	if bus == nil {
		t.Fatal("bus should not be nil even without dir")
	}
	// Bus should operate without DB (in-memory only)
}

func TestNewEventBus_SubscribersMapInitialized(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	if len(bus.subscribers) != 0 {
		t.Errorf("new bus should have 0 subscribers, got %d", len(bus.subscribers))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Subscribe Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_Subscribe_SingleHandler(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	called := false
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		called = true
	})

	if len(bus.subscribers[EvtWorktreeCreated]) != 1 {
		t.Errorf("expected 1 subscriber, got %d", len(bus.subscribers[EvtWorktreeCreated]))
	}

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})
	if !called {
		t.Error("handler should have been called")
	}
}

func TestEventBus_Subscribe_MultipleHandlers(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	var count int32
	for i := 0; i < 3; i++ {
		bus.Subscribe(EvtConflictDetected, func(e BusEvent) {
			atomic.AddInt32(&count, 1)
		})
	}

	bus.EmitSync(BusEvent{Type: EvtConflictDetected, Source: "test"})

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 handlers called, got %d", count)
	}
}

func TestEventBus_Subscribe_DifferentEventTypes(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {})
	bus.Subscribe(EvtConflictDetected, func(e BusEvent) {})
	bus.Subscribe(EvtCleanupComplete, func(e BusEvent) {})

	if len(bus.subscribers) != 3 {
		t.Errorf("expected 3 event types subscribed, got %d", len(bus.subscribers))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Emit Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_Emit_PersistsToSQLite(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.Emit(BusEvent{
		Type:    EvtWorktreeCreated,
		Source:  "thavren",
		Payload: map[string]any{"branch": "feature/test", "profile": "feature"},
	})

	// Give async operations time to complete
	time.Sleep(50 * time.Millisecond)

	events, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Source != "thavren" {
		t.Errorf("Source = %q, want thavren", events[0].Source)
	}
}

func TestEventBus_Emit_NoSubscribers(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	// Should not panic with no subscribers
	bus.Emit(BusEvent{
		Type:   EvtWorktreeLocked,
		Source: "test",
	})

	time.Sleep(50 * time.Millisecond)

	// Event should still be persisted
	events, err := bus.QueryEvents(EvtWorktreeLocked, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 persisted event, got %d", len(events))
	}
}

func TestEventBus_Emit_WithPayload(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.Emit(BusEvent{
		Type:   EvtStateTransition,
		Source: "ows",
		Payload: map[string]any{
			"from":  "CREATED",
			"to":    "ACTIVE",
			"event": "WORK_STARTED",
		},
	})

	time.Sleep(50 * time.Millisecond)

	events, err := bus.QueryEvents(EvtStateTransition, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestEventBus_Emit_NilPayload(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	// Should not panic with nil payload
	bus.Emit(BusEvent{
		Type:    EvtOperationQueued,
		Source:  "test",
		Payload: nil,
	})

	time.Sleep(50 * time.Millisecond)

	events, err := bus.QueryEvents(EvtOperationQueued, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestEventBus_Emit_EmptyPayload(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.Emit(BusEvent{
		Type:    EvtOperationFlushed,
		Source:  "test",
		Payload: map[string]any{},
	})

	time.Sleep(50 * time.Millisecond)

	events, err := bus.QueryEvents(EvtOperationFlushed, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestEventBus_Emit_SetsTimestamp(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	before := time.Now().UTC()

	var received BusEvent
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		received = e
	})

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})

	if received.Timestamp.Before(before) {
		t.Error("timestamp should be set to current time")
	}
}

func TestEventBus_Emit_AsyncHandlers(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // simulate work
	})
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		defer wg.Done()
	})

	bus.Emit(BusEvent{Type: EvtWorktreeCreated, Source: "test"})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Both handlers completed
	case <-time.After(2 * time.Second):
		t.Fatal("async handlers timed out")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EmitSync Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_EmitSync_BlocksUntilComplete(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	handlerDone := make(chan struct{})
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		time.Sleep(50 * time.Millisecond)
		close(handlerDone)
	})

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})

	// Handler should be done because EmitSync blocks
	select {
	case <-handlerDone:
		// OK
	default:
		t.Error("EmitSync should block until handler completes")
	}
}

func TestEventBus_EmitSync_PersistsToSQLite(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.EmitSync(BusEvent{
		Type:   EvtIntegrationComplete,
		Source: "test",
	})

	events, err := bus.QueryEvents(EvtIntegrationComplete, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestEventBus_EmitSync_NoSubscribers(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	// Should not panic
	bus.EmitSync(BusEvent{Type: EvtConflictResolved, Source: "test"})
}

func TestEventBus_EmitSync_MultipleHandlers(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	var count int32
	for i := 0; i < 5; i++ {
		bus.Subscribe(EvtCleanupComplete, func(e BusEvent) {
			atomic.AddInt32(&count, 1)
		})
	}

	bus.EmitSync(BusEvent{Type: EvtCleanupComplete, Source: "test"})

	if atomic.LoadInt32(&count) != 5 {
		t.Errorf("expected 5 handlers called, got %d", count)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// QueryEvents Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_QueryEvents_ByType(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	// Emit different event types
	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "a"})
	bus.EmitSync(BusEvent{Type: EvtWorktreeLocked, Source: "b"})
	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "c"})

	created, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(created) != 2 {
		t.Errorf("expected 2 WORKTREE_CREATED events, got %d", len(created))
	}

	locked, err := bus.QueryEvents(EvtWorktreeLocked, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(locked) != 1 {
		t.Errorf("expected 1 WORKTREE_LOCKED event, got %d", len(locked))
	}
}

func TestEventBus_QueryEvents_TimeFilter(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})

	// Query with very short window — should still find it (just emitted)
	events, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Second)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event in 1s window, got %d", len(events))
	}
}

func TestEventBus_QueryEvents_NoResults(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	events, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestEventBus_QueryEvents_NoDB(t *testing.T) {
	bus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		db:          nil,
	}

	_, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if err == nil {
		t.Fatal("expected error when no DB")
	}
}

func TestEventBus_QueryEvents_OrderDesc(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "first"})
	time.Sleep(10 * time.Millisecond)
	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "second"})

	events, err := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	// DESC order: second should come first
	if events[0].Source != "second" {
		t.Errorf("first result should be 'second' (DESC order), got %q", events[0].Source)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Close Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_Close_WithDB(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Double close should error (DB already closed)
	err = bus.Close()
	if err == nil {
		t.Log("double close did not error (driver-dependent)")
	}
}

func TestEventBus_Close_NoDB(t *testing.T) {
	bus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		db:          nil,
	}

	err := bus.Close()
	if err != nil {
		t.Fatalf("Close without DB should not error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// AgentLock Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestAgentLock_IsExpired_Expired(t *testing.T) {
	lock := &AgentLock{
		Worktree:   "feature/test",
		Owner:      "thavren",
		AcquiredAt: time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:  time.Now().UTC().Add(-1 * time.Hour), // expired 1h ago
		Reason:     "code review",
	}

	if !lock.IsExpired() {
		t.Error("lock should be expired")
	}
}

func TestAgentLock_IsExpired_NotExpired(t *testing.T) {
	lock := &AgentLock{
		Worktree:   "feature/test",
		Owner:      "thavren",
		AcquiredAt: time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(1 * time.Hour), // expires in 1h
		Reason:     "code review",
	}

	if lock.IsExpired() {
		t.Error("lock should not be expired")
	}
}

func TestAgentLock_IsExpired_ExactlyNow(t *testing.T) {
	lock := &AgentLock{
		Worktree:   "feature/test",
		Owner:      "thavren",
		AcquiredAt: time.Now().UTC().Add(-1 * time.Hour),
		ExpiresAt:  time.Now().UTC().Add(-1 * time.Millisecond), // just expired
		Reason:     "test",
	}

	if !lock.IsExpired() {
		t.Error("lock just past expiry should be expired")
	}
}

func TestAgentLock_Fields(t *testing.T) {
	now := time.Now().UTC()
	lock := &AgentLock{
		Worktree:   "feature/my-task",
		Owner:      "soren",
		AcquiredAt: now,
		ExpiresAt:  now.Add(24 * time.Hour),
		Reason:     "implementing tests",
	}

	if lock.Worktree != "feature/my-task" {
		t.Errorf("Worktree = %q", lock.Worktree)
	}
	if lock.Owner != "soren" {
		t.Errorf("Owner = %q", lock.Owner)
	}
	if lock.Reason != "implementing tests" {
		t.Errorf("Reason = %q", lock.Reason)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EventType Constants Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestEventType_Constants(t *testing.T) {
	types := map[EventType]string{
		EvtWorktreeCreated:     "WORKTREE_CREATED",
		EvtWorktreeLocked:      "WORKTREE_LOCKED",
		EvtWorktreeUnlocked:    "WORKTREE_UNLOCKED",
		EvtConflictDetected:    "CONFLICT_DETECTED",
		EvtConflictResolved:    "CONFLICT_RESOLVED",
		EvtIntegrationComplete: "INTEGRATION_COMPLETE",
		EvtCleanupComplete:     "CLEANUP_COMPLETE",
		EvtOperationQueued:     "OPERATION_QUEUED",
		EvtOperationFlushed:    "OPERATION_FLUSHED",
		EvtStateTransition:     "STATE_TRANSITION",
	}

	for evt, expected := range types {
		if string(evt) != expected {
			t.Errorf("EventType %v = %q, want %q", evt, string(evt), expected)
		}
	}

	// Verify we have exactly 10 event types
	if len(types) != 10 {
		t.Errorf("expected 10 event types, got %d", len(types))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BusEvent Tests
// ═══════════════════════════════════════════════════════════════════════════

func TestBusEvent_Fields(t *testing.T) {
	now := time.Now().UTC()
	event := BusEvent{
		Type:      EvtWorktreeCreated,
		Timestamp: now,
		Source:    "thavren",
		Payload: map[string]any{
			"branch":  "feature/test",
			"profile": "feature",
			"count":   42,
		},
	}

	if event.Type != EvtWorktreeCreated {
		t.Errorf("Type = %q", event.Type)
	}
	if event.Source != "thavren" {
		t.Errorf("Source = %q", event.Source)
	}
	if event.Payload["branch"] != "feature/test" {
		t.Errorf("Payload branch = %v", event.Payload["branch"])
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Integration: Full Event Lifecycle
// ═══════════════════════════════════════════════════════════════════════════

func TestEventBus_FullLifecycle(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, err := NewEventBus(dir)
	if err != nil {
		t.Fatalf("NewEventBus: %v", err)
	}
	defer bus.Close()

	// Subscribe to multiple event types
	var createdCount, lockedCount int32
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		atomic.AddInt32(&createdCount, 1)
	})
	bus.Subscribe(EvtWorktreeLocked, func(e BusEvent) {
		atomic.AddInt32(&lockedCount, 1)
	})

	// Simulate worktree lifecycle
	bus.EmitSync(BusEvent{
		Type:    EvtWorktreeCreated,
		Source:  "ows",
		Payload: map[string]any{"branch": "feature/lifecycle"},
	})

	bus.EmitSync(BusEvent{
		Type:    EvtWorktreeLocked,
		Source:  "thavren",
		Payload: map[string]any{"branch": "feature/lifecycle", "reason": "review"},
	})

	bus.EmitSync(BusEvent{
		Type:   EvtWorktreeUnlocked,
		Source: "thavren",
	})

	bus.EmitSync(BusEvent{
		Type:   EvtIntegrationComplete,
		Source: "ows",
	})

	bus.EmitSync(BusEvent{
		Type:   EvtCleanupComplete,
		Source: "ows",
	})

	// Verify handler counts
	if atomic.LoadInt32(&createdCount) != 1 {
		t.Errorf("created handler called %d times, want 1", createdCount)
	}
	if atomic.LoadInt32(&lockedCount) != 1 {
		t.Errorf("locked handler called %d times, want 1", lockedCount)
	}

	// Verify persisted events
	allCreated, _ := bus.QueryEvents(EvtWorktreeCreated, 1*time.Minute)
	if len(allCreated) != 1 {
		t.Errorf("persisted created events = %d, want 1", len(allCreated))
	}

	allLocked, _ := bus.QueryEvents(EvtWorktreeLocked, 1*time.Minute)
	if len(allLocked) != 1 {
		t.Errorf("persisted locked events = %d, want 1", len(allLocked))
	}

	allIntegrated, _ := bus.QueryEvents(EvtIntegrationComplete, 1*time.Minute)
	if len(allIntegrated) != 1 {
		t.Errorf("persisted integrated events = %d, want 1", len(allIntegrated))
	}
}

func TestEventBus_ConcurrentEmit(t *testing.T) {
	dir := setupEventBusDir(t)
	bus, _ := NewEventBus(dir)
	defer bus.Close()

	var count int32
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		atomic.AddInt32(&count, 1)
	})

	// Emit concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "concurrent"})
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&count) != 10 {
		t.Errorf("expected 10 handler calls, got %d", count)
	}
}

func TestEventBus_EmitWithoutDB(t *testing.T) {
	// Bus without SQLite — should still notify subscribers
	bus := &EventBus{
		subscribers: make(map[EventType][]EventHandler),
		db:          nil,
	}

	called := false
	bus.Subscribe(EvtWorktreeCreated, func(e BusEvent) {
		called = true
	})

	bus.EmitSync(BusEvent{Type: EvtWorktreeCreated, Source: "test"})

	if !called {
		t.Error("handler should be called even without DB")
	}
}
