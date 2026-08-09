package memory

import (
	"sort"
	"strings"
	"time"
)

// Recall queries the ledger for cards relevant to a topic or intent.
type Recall struct {
	ledger *Ledger
}

// NewRecall creates a recall engine for the given ledger.
func NewRecall(ledger *Ledger) *Recall {
	return &Recall{ledger: ledger}
}

// ByTags returns active cards matching any of the given tags.
// Relevance is weighted by tag match count and recency.
func (r *Recall) ByTags(tags []string) []RecallResult {
	if len(tags) == 0 {
		return nil
	}

	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	var results []RecallResult
	for _, card := range r.ledger.ActiveCards() {
		matches := 0
		for _, ct := range card.Tags {
			if tagSet[ct] {
				matches++
			}
		}
		if matches > 0 {
			relevance := float64(matches) / float64(len(tags)+1)
			results = append(results, RecallResult{
				Card:      card,
				Relevance: relevance,
				Reason:    "tag_match:" + strings.Join(card.Tags, ","),
			})
		}
	}

	// Sort by relevance descending, then by confirmed date descending
	sort.Slice(results, func(i, j int) bool {
		if results[i].Relevance != results[j].Relevance {
			return results[i].Relevance > results[j].Relevance
		}
		return results[i].Card.LastConfirmed > results[j].Card.LastConfirmed
	})

	return results
}

// ByQuery searches card summaries and operational rules for matching terms.
func (r *Recall) ByQuery(query string, limit int) []RecallResult {
	query = strings.ToLower(query)
	terms := strings.Fields(query)

	var results []RecallResult
	for _, card := range r.ledger.ActiveCards() {
		relevance := r.matchScore(card, terms)
		if relevance > 0 {
			results = append(results, RecallResult{
				Card:      card,
				Relevance: relevance,
				Reason:    "query_match",
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// RecentActive returns the most recently confirmed active cards, up to limit.
func (r *Recall) RecentActive(limit int) []Card {
	active := r.ledger.ActiveCards()

	sort.Slice(active, func(i, j int) bool {
		return active[i].LastConfirmed > active[j].LastConfirmed
	})

	if limit > 0 && len(active) > limit {
		active = active[:limit]
	}

	return active
}

// CriticalCards returns active cards with CRITICAL priority.
func (r *Recall) CriticalCards() []Card {
	var critical []Card
	for _, card := range r.ledger.ActiveCards() {
		if card.Priority == "CRITICAL" {
			critical = append(critical, card)
		}
	}
	return critical
}

// matchScore computes a relevance score for a card against search terms.
func (r *Recall) matchScore(card Card, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}

	summary := strings.ToLower(card.Summary)
	rule := strings.ToLower(card.OperationalRule)
	topic := strings.ToLower(card.Topic)
	id := strings.ToLower(card.ID)
	allTags := strings.ToLower(strings.Join(card.Tags, " "))

	matches := 0
	for _, term := range terms {
		// Higher weight for ID match
		if strings.Contains(id, term) {
			matches += 3
			continue
		}
		// Medium weight for tag match
		if strings.Contains(allTags, term) {
			matches += 2
			continue
		}
		// Base weight for summary/rule/topic match
		if strings.Contains(summary, term) || strings.Contains(rule, term) || strings.Contains(topic, term) {
			matches++
		}
	}

	if matches == 0 {
		return 0
	}

	// Normalize: max possible is len(terms)*3
	maxScore := float64(len(terms) * 3)
	score := float64(matches) / maxScore

	// Recency bonus: cards confirmed within 7 days get up to 0.2 bonus
	if card.LastConfirmed != "" {
		if confirmed, err := time.Parse("2006-01-02", card.LastConfirmed); err == nil {
			daysAgo := time.Since(confirmed).Hours() / 24
			if daysAgo < 7 {
				score += 0.2 * (1 - daysAgo/7)
			}
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}
