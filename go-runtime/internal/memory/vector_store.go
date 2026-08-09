// Package memory provides semantic memory search using vector embeddings.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ovav/ovav/internal/memory/embeddings"
)

// VectorStore provides semantic search over memory cards.
type VectorStore struct {
	mu      sync.RWMutex
	index   *embeddings.Index
	dataDir string
}

// NewVectorStore creates a new vector store.
func NewVectorStore(dataDir string) (*VectorStore, error) {
	if dataDir == "" {
		dataDir = ".ovav/memory/vectors"
	}

	indexPath := filepath.Join(dataDir, "index.json")
	index, err := embeddings.LoadIndex(indexPath)
	if err != nil {
		return nil, fmt.Errorf("load index: %w", err)
	}

	return &VectorStore{
		index:   index,
		dataDir: dataDir,
	}, nil
}

// Save persists the vector store to disk.
func (vs *VectorStore) Save() error {
	return vs.index.Save()
}

// IndexCard adds a memory card to the vector index.
func (vs *VectorStore) IndexCard(card *Card) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Build searchable text from card
	text := buildSearchableText(card)
	return vs.index.AddCard(card.ID, text, "")
}

// buildSearchableText creates searchable text from a card.
func buildSearchableText(card *Card) string {
	text := card.Summary
	if card.OperationalRule != "" {
		text += " " + card.OperationalRule
	}
	if card.Topic != "" {
		text += " " + card.Topic
	}
	for _, tag := range card.Tags {
		text += " " + tag
	}
	return text
}

// Search performs semantic search over indexed cards.
func (vs *VectorStore) Search(query string, limit int) ([]SearchResult, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	results := vs.index.Search(query, limit)

	// Convert to memory SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			CardID:   r.ID,
			Text:     r.Text,
			Score:    r.Score,
			Category: "semantic",
		}
	}

	return searchResults, nil
}

// SearchHybrid performs hybrid search combining semantic and keyword matching.
func (vs *VectorStore) SearchHybrid(query string, tags []string, limit int) ([]SearchResult, error) {
	// First do semantic search
	semanticResults, err := vs.Search(query, limit*2)
	if err != nil {
		return nil, err
	}

	// If tags provided, filter by tag match
	if len(tags) > 0 {
		var filtered []SearchResult
		for _, r := range semanticResults {
			// Simple tag matching
			match := false
			for _, tag := range tags {
				if strings.Contains(strings.ToLower(query), strings.ToLower(tag)) {
					match = true
					break
				}
			}
			if match {
				r.Category = "hybrid"
				filtered = append(filtered, r)
			}
		}
		if len(filtered) > 0 {
			semanticResults = filtered
		}
	}

	if len(semanticResults) > limit {
		semanticResults = semanticResults[:limit]
	}

	return semanticResults, nil
}

// SearchResult represents a search result with relevance.
type SearchResult struct {
	CardID   string  `json:"card_id"`
	Text     string  `json:"text"`
	Score    float64 `json:"score"` // 0.0 to 1.0
	Category string  `json:"category,omitempty"`
}

// Deduplicate removes near-duplicate cards from the index.
func (vs *VectorStore) Deduplicate(threshold float64) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	removed := vs.index.Deduplicate(threshold)
	if err := vs.Save(); err != nil {
		return removed, err
	}
	return removed, nil
}

// Stats returns statistics about the vector store.
func (vs *VectorStore) Stats() VectorStats {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return VectorStats{
		TotalEmbeddings: vs.index.Size(),
		DataDir:         vs.dataDir,
	}
}

// VectorStats holds vector store statistics.
type VectorStats struct {
	TotalEmbeddings int    `json:"total_embeddings"`
	DataDir         string `json:"data_dir"`
}

// RebuildIndex rebuilds the entire index from memory cards.
func (vs *VectorStore) RebuildIndex(ledger *Ledger) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Clear existing index
	vs.index.Clear()

	// Re-index all cards
	for _, card := range ledger.Cards {
		text := buildSearchableText(&card)
		if err := vs.index.AddCard(card.ID, text, ""); err != nil {
			return fmt.Errorf("index card %s: %w", card.ID, err)
		}
	}

	return vs.Save()
}

// EnsureDir creates directory if not exists.
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
