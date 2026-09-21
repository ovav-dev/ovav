package memory

import (
	"fmt"
	"time"
)

// Governor orchestrates memory write and read pipelines.
// Write pipeline: classify → validate → write
// Read pipeline:  query → filter → pack
type Governor struct {
	ledger     *Ledger
	classifier *Classifier
	recall     *Recall
}

// NewGovernor creates a memory governor for OVAV Systems.
// allowSensitive must be true for Systems (allows internal/sensitive storage).
func NewGovernor(root string, allowSensitive bool) (*Governor, error) {
	ledger, err := LoadLedger(root)
	if err != nil {
		return nil, fmt.Errorf("memory governor: load ledger: %w", err)
	}

	return &Governor{
		ledger:     ledger,
		classifier: NewClassifier(allowSensitive),
		recall:     NewRecall(ledger),
	}, nil
}

// Ledger returns the underlying ledger.
func (g *Governor) Ledger() *Ledger {
	return g.ledger
}

// Recall returns the recall engine.
func (g *Governor) Recall() *Recall {
	return g.recall
}

// Write submits a card through the write pipeline: classify → validate → write.
// Returns error if the card is rejected at any stage.
func (g *Governor) Write(card Card) error {
	// Stage 1: Classify
	result := g.classifier.Classify(card)
	if !result.Allow {
		return fmt.Errorf("memory governor: write rejected by classifier: %s (tag=%s)",
			result.Reason, result.Tag)
	}

	// Stage 2: Validate
	if err := g.validate(card); err != nil {
		return fmt.Errorf("memory governor: validation failed: %w", err)
	}

	// Stage 3: Write
	card.LastConfirmed = time.Now().Format("2006-01-02")
	g.ledger.UpsertCard(card)

	return g.ledger.Save()
}

// validate performs basic sanity checks on a card.
func (g *Governor) validate(card Card) error {
	if card.ID == "" {
		return fmt.Errorf("card ID is required")
	}
	if card.Summary == "" {
		return fmt.Errorf("card summary is required for %s", card.ID)
	}
	if card.OperationalRule == "" {
		return fmt.Errorf("card operational_rule is required for %s", card.ID)
	}
	if card.Status == "" {
		return fmt.Errorf("card status is required for %s", card.ID)
	}
	return nil
}

// SessionPack generates a compact context pack for session start.
// Includes critical cards, recent active cards, and operational rules.
func (g *Governor) SessionPack() *ContextPack {
	pack := &ContextPack{
		Source:      "active_context_ledger",
		GeneratedAt: time.Now(),
	}

	// Critical cards first
	critical := g.recall.CriticalCards()
	pack.Cards = append(pack.Cards, critical...)

	// Recent active cards (top 8)
	recent := g.recall.RecentActive(8)
	for _, card := range recent {
		// Skip if already in critical
		if card.Priority == "CRITICAL" {
			continue
		}
		pack.Cards = append(pack.Cards, card)
	}

	// Collect operational rules
	for _, card := range pack.Cards {
		if card.OperationalRule != "" {
			pack.OperationalRules = append(pack.OperationalRules, card.OperationalRule)
		}
	}

	return pack
}

// QueryResult wraps recall results with operational rules.
func (g *Governor) QueryResult(query string, limit int) *ContextPack {
	results := g.recall.ByQuery(query, limit)
	pack := &ContextPack{
		Source:      "recall:" + query,
		GeneratedAt: time.Now(),
	}

	for _, r := range results {
		pack.Cards = append(pack.Cards, r.Card)
		if r.Card.OperationalRule != "" {
			pack.OperationalRules = append(pack.OperationalRules, r.Card.OperationalRule)
		}
	}

	return pack
}
