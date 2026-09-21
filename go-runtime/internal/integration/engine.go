// Package integration implements OVAV's native subsystem auto-integration engine.
//
// This package provides autonomous interconnection between all OVAV subsystems:
// - Memory ↔ Research ↔ Connect ↔ Test ↔ Plan ↔ Validators
//
// All subsystems auto-trigger based on context, events, and schedules.
// Designed for real-time operation in 1st plane (harness work) and 2nd plane (background).
package integration

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	autonomous "github.com/ovav/ovav/internal/autonomous/engine"
	"github.com/ovav/ovav/internal/connect/tracker"
	"github.com/ovav/ovav/internal/memory"
)

// Finding represents a research finding from autonomous research.
type Finding struct {
	ID          string
	Title       string
	Description string
	Source      string
	Severity    string
	Category    string
}

// Config holds the integration engine configuration.
type Config struct {
	// Background modes
	BackgroundEnabled bool
	ResearchInterval  time.Duration
	IndexInterval     time.Duration
	CostInterval      time.Duration
	TestInterval      time.Duration

	// Foreground integration
	ForegroundEnabled bool

	// Data directories
	RootDir string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BackgroundEnabled: true,
		ResearchInterval:  24 * time.Hour,
		IndexInterval:     1 * time.Hour,
		CostInterval:      1 * time.Hour,
		TestInterval:      12 * time.Hour,
		ForegroundEnabled: true,
		RootDir:           ".",
	}
}

// Engine orchestrates all OVAV subsystems with auto-integration.
type Engine struct {
	config *Config
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Subsystems
	memory   *memory.VectorStore
	research *autonomous.Engine
	connect  *tracker.Tracker

	// Event channels
	events      chan Event
	subscribers map[string][]EventHandler

	// Status
	running      bool
	lastIndex    time.Time
	lastResearch time.Time
}

// Event represents a system event that can trigger integrations.
type Event struct {
	Type      EventType
	Source    string
	Target    string
	Payload   interface{}
	Timestamp time.Time
}

// EventType classifies system events.
type EventType string

const (
	EventFileChanged   EventType = "file_changed"
	EventSessionStart  EventType = "session_start"
	EventSessionEnd    EventType = "session_end"
	EventAgentQuery    EventType = "agent_query"
	EventAPICall       EventType = "api_call"
	EventTaskCompleted EventType = "task_completed"
	EventValidationRun EventType = "validation_run"
	EventResearchDone  EventType = "research_done"
	EventCostThreshold EventType = "cost_threshold"
	EventMemoryIndexed EventType = "memory_indexed"
)

// EventHandler is called when an event occurs.
type EventHandler func(Event)

// New creates a new integration engine.
func New(cfg *Config) (*Engine, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	e := &Engine{
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
		events:      make(chan Event, 100),
		subscribers: make(map[string][]EventHandler),
	}

	// Initialize subsystems
	if err := e.initSubsystems(); err != nil {
		cancel()
		return nil, fmt.Errorf("init subsystems: %w", err)
	}

	return e, nil
}

// initSubsystems initializes all OVAV subsystems.
func (e *Engine) initSubsystems() error {
	// Memory vector store
	memDir := filepath.Join(e.config.RootDir, ".ovav", "memory", "vectors")
	vs, err := memory.NewVectorStore(memDir)
	if err != nil {
		log.Printf("Warning: memory vector store init failed: %v", err)
	} else {
		e.memory = vs
	}

	// Research engine
	researchCfg := autonomous.Config{
		DataDir: filepath.Join(e.config.RootDir, ".ovav", "intelligence"),
		Timeout: 30 * time.Second,
	}
	research, err := autonomous.New(researchCfg)
	if err != nil {
		log.Printf("Warning: research engine init failed: %v", err)
	} else {
		e.research = research
	}

	// Connect tracker
	connectDir := filepath.Join(e.config.RootDir, ".ovav", "connect")
	tk := tracker.New(connectDir)
	if err := tk.LoadProviders(); err != nil {
		log.Printf("Warning: connect tracker init failed: %v", err)
	} else {
		e.connect = tk
	}

	return nil
}

// Start begins the integration engine.
func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("already running")
	}
	e.running = true
	e.mu.Unlock()

	// Start event processor
	go e.processEvents()

	// Start background tasks
	if e.config.BackgroundEnabled {
		go e.backgroundLoop()
	}

	log.Println("OVAV Integration Engine started")
	return nil
}

// Stop halts the integration engine.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.cancel()
	e.running = false
	log.Println("OVAV Integration Engine stopped")
}

// Subscribe registers a handler for events.
func (e *Engine) Subscribe(target string, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers[target] = append(e.subscribers[target], handler)
}

// Emit sends an event to all subscribers.
func (e *Engine) Emit(event Event) {
	select {
	case e.events <- event:
	default:
		log.Printf("Warning: event channel full, dropping %s", event.Type)
	}
}

// processEvents handles incoming events.
func (e *Engine) processEvents() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case event := <-e.events:
			e.handleEvent(event)
		}
	}
}

// handleEvent dispatches event to appropriate handlers.
func (e *Engine) handleEvent(event Event) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	handlers := e.subscribers[event.Target]
	for _, h := range handlers {
		go h(event)
	}

	// Also notify wildcard subscribers
	for _, h := range e.subscribers["*"] {
		go h(event)
	}
}

// backgroundLoop runs periodic background tasks.
func (e *Engine) backgroundLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.runBackgroundTasks()
		}
	}
}

// runBackgroundTasks executes periodic subsystem tasks.
func (e *Engine) runBackgroundTasks() {
	now := time.Now()

	// Research check (daily)
	if e.research != nil && now.Sub(e.lastResearch) >= e.config.ResearchInterval {
		e.runResearchCycle()
		e.lastResearch = now
	}

	// Memory index (hourly)
	if e.memory != nil && now.Sub(e.lastIndex) >= e.config.IndexInterval {
		e.RunMemoryIndex()
		e.lastIndex = now
	}
}

// runResearchCycle executes autonomous research.
func (e *Engine) runResearchCycle() {
	if e.research == nil {
		return
	}

	log.Println("Running autonomous research cycle...")

	result, err := e.research.Run()
	if err != nil {
		log.Printf("Research cycle failed: %v", err)
		return
	}

	if result != nil && len(result.Findings) > 0 {
		// Index findings in memory
		for _, f := range result.Findings {
			e.IndexFinding(Finding{
				ID:          f.ID,
				Title:       f.Title,
				Description: f.Description,
				Source:      f.Source,
				Severity:    f.Severity,
				Category:    f.Category,
			})
		}

		// Emit event
		e.Emit(Event{
			Type:      EventResearchDone,
			Source:    "research",
			Target:    "memory",
			Payload:   result,
			Timestamp: now(),
		})
	}

	log.Printf("Research cycle complete: %d findings", len(result.Findings))
}

// runMemoryIndex rebuilds the memory index.
func (e *Engine) RunMemoryIndex() {
	if e.memory == nil {
		return
	}

	log.Println("Running memory index...")

	// Load ledger and rebuild
	ledgerPath := filepath.Join(e.config.RootDir, ".ovav", "memory", "ledger.yaml")
	ledger, err := memory.LoadLedger(ledgerPath)
	if err != nil {
		log.Printf("Memory index: ledger load failed: %v", err)
		return
	}

	if err := e.memory.RebuildIndex(ledger); err != nil {
		log.Printf("Memory rebuild failed: %v", err)
		return
	}

	e.Emit(Event{
		Type:      EventMemoryIndexed,
		Source:    "memory",
		Target:    "*",
		Payload:   map[string]int{"cards": len(ledger.Cards)},
		Timestamp: now(),
	})

	log.Printf("Memory index complete: %d cards", len(ledger.Cards))
}

// IndexFinding adds a research finding to memory.
func (e *Engine) IndexFinding(finding Finding) error {
	if e.memory == nil {
		return fmt.Errorf("memory not initialized")
	}

	card := &memory.Card{
		ID:       "research-" + finding.ID,
		Topic:    finding.Category,
		Summary:  finding.Title + ": " + finding.Description,
		Tags:     []string{"research", finding.Source, finding.Severity},
		Priority: finding.Severity,
	}

	return e.memory.IndexCard(card)
}

// SearchMemory performs semantic search with auto-indexing.
func (e *Engine) SearchMemory(query string, limit int) ([]memory.SearchResult, error) {
	if e.memory == nil {
		return nil, fmt.Errorf("memory not initialized")
	}

	return e.memory.Search(query, limit)
}

// RecordAPIUsage tracks API call tokens.
func (e *Engine) RecordAPIUsage(providerID, model string, inputTokens, outputTokens int) error {
	if e.connect == nil {
		return fmt.Errorf("connect not initialized")
	}

	record := &tracker.UsageRecord{
		ID:           fmt.Sprintf("%s-%d", providerID, time.Now().Unix()),
		ProviderID:   providerID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		CostUSD:      tracker.CalculateCost(model, inputTokens, outputTokens),
		Timestamp:    now(),
	}

	return e.connect.RecordUsage(record)
}

// TriggerTestRun starts a test cycle.
func (e *Engine) TriggerTestRun() {
	e.Emit(Event{
		Type:      EventValidationRun,
		Source:    "integration",
		Target:    "test",
		Payload:   map[string]interface{}{"trigger": "scheduled"},
		Timestamp: now(),
	})
}

// Status returns the integration engine status.
func (e *Engine) Status() Status {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var memStats memory.VectorStats
	if e.memory != nil {
		memStats = e.memory.Stats()
	}

	return Status{
		Running:       e.running,
		MemoryIndexed: memStats.TotalEmbeddings,
		LastResearch:  e.lastResearch,
		LastIndex:     e.lastIndex,
		Subscribers:   len(e.subscribers),
	}
}

// Status holds integration engine status.
type Status struct {
	Running       bool
	MemoryIndexed int
	LastResearch  time.Time
	LastIndex     time.Time
	Subscribers   int
}

// Helper for now()
func now() time.Time {
	return time.Now()
}
