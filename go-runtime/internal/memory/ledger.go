package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultLedgerPath = ".ovav/registry/active_context_ledger.yaml"

// LoadLedger reads the active context ledger from disk.
// root is the OVAV repository root.
func LoadLedger(root string) (*Ledger, error) {
	path := filepath.Join(root, defaultLedgerPath)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newEmptyLedger(path), nil
		}
		return nil, fmt.Errorf("memory: read ledger: %w", err)
	}

	var raw struct {
		ActiveContextLedger Ledger `yaml:"active_context_ledger"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("memory: parse ledger: %w", err)
	}

	ledger := &raw.ActiveContextLedger
	ledger.path = path
	ledger.loadedAt = time.Now()

	for i := range ledger.Cards {
		ledger.Cards[i].loadedAt = ledger.loadedAt
	}

	return ledger, nil
}

// SaveLedger writes the ledger back to disk atomically.
// Uses temp file + rename to prevent truncation on crash.
func (l *Ledger) Save() error {
	if l.path == "" {
		return fmt.Errorf("memory: ledger has no path — was it loaded from disk?")
	}

	l.LastUpdate = time.Now().Format(time.RFC3339)

	wrapper := struct {
		ActiveContextLedger Ledger `yaml:"active_context_ledger"`
	}{ActiveContextLedger: *l}

	data, err := yaml.Marshal(&wrapper)
	if err != nil {
		return fmt.Errorf("memory: marshal ledger: %w", err)
	}

	tmpPath := l.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("memory: write temp ledger: %w", err)
	}

	if err := os.Rename(tmpPath, l.path); err != nil {
		os.Remove(tmpPath) // best-effort cleanup
		return fmt.Errorf("memory: rename ledger: %w", err)
	}

	return nil
}

// ActiveCards returns cards with status "active" or "in_progress".
func (l *Ledger) ActiveCards() []Card {
	var active []Card
	for _, c := range l.Cards {
		if c.Status == StatusActive || c.Status == StatusInProgress {
			active = append(active, c)
		}
	}
	return active
}

// CardByID finds a card by its ID. Returns nil if not found.
func (l *Ledger) CardByID(id string) *Card {
	for i := range l.Cards {
		if l.Cards[i].ID == id {
			return &l.Cards[i]
		}
	}
	return nil
}

// UpsertCard adds a new card or updates an existing one by ID.
func (l *Ledger) UpsertCard(card Card) {
	card.loadedAt = time.Now()
	for i := range l.Cards {
		if l.Cards[i].ID == card.ID {
			l.Cards[i] = card
			return
		}
	}
	l.Cards = append(l.Cards, card)
}

// newEmptyLedger returns a minimal ledger for when the file doesn't exist.
func newEmptyLedger(path string) *Ledger {
	return &Ledger{
		Version:    3,
		Purpose:    "Operational memory for OVAV SYSTEMS.",
		Cards:      []Card{},
		LastUpdate: time.Now().Format(time.RFC3339),
		path:       path,
		loadedAt:   time.Now(),
	}
}
