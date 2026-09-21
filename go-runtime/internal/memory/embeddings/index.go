package embeddings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// SearchResult represents a search match with similarity score.
type SearchResult struct {
	ID       string  `json:"id"`
	Text     string  `json:"text"`
	Score    float64 `json:"score"` // 0.0 to 1.0
	Category string  `json:"category,omitempty"`
}

// Index manages a collection of embeddings for semantic search.
type Index struct {
	mu         sync.RWMutex
	embeddings map[string]*Embedding
	embedder   *Embedder
	indexPath  string
}

// IndexConfig holds index configuration.
type IndexConfig struct {
	Dimensions int    `json:"dimensions"`
	IndexPath  string `json:"index_path"`
}

// NewIndex creates a new embedding index.
func NewIndex(dimensions int, indexPath string) *Index {
	return &Index{
		embeddings: make(map[string]*Embedding),
		embedder:   NewEmbedder(dimensions),
		indexPath:  indexPath,
	}
}

// Load loads an index from disk.
func LoadIndex(indexPath string) (*Index, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewIndex(384, indexPath), nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	var index IndexFile
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse index: %w", err)
	}

	idx := &Index{
		embeddings: make(map[string]*Embedding),
		embedder:   NewEmbedder(index.Config.Dimensions),
		indexPath:  indexPath,
	}

	for _, e := range index.Embeddings {
		idx.embeddings[e.ID] = e
	}

	return idx, nil
}

// Save saves the index to disk.
func (idx *Index) Save() error {
	if idx.indexPath == "" {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	embeddings := make([]*Embedding, 0, len(idx.embeddings))
	for _, e := range idx.embeddings {
		embeddings = append(embeddings, e)
	}

	indexFile := IndexFile{
		Config: IndexConfig{
			Dimensions: idx.embedder.Dimensions(),
			IndexPath:  idx.indexPath,
		},
		Embeddings: embeddings,
	}

	data, err := json.MarshalIndent(indexFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(idx.indexPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return os.WriteFile(idx.indexPath, data, 0644)
}

// IndexFile represents the on-disk index structure.
type IndexFile struct {
	Config     IndexConfig  `json:"config"`
	Embeddings []*Embedding `json:"embeddings"`
}

// Add adds an embedding to the index.
func (idx *Index) Add(id, text string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	embedding := idx.embedder.Embed(id, text)
	idx.embeddings[id] = embedding
	return nil
}

// AddCard adds a memory card to the index.
func (idx *Index) AddCard(id, summary, tags string) error {
	// Combine summary and tags for richer embedding
	text := summary
	if tags != "" {
		text += " " + tags
	}
	return idx.Add(id, text)
}

// Search finds the top k most similar embeddings to the query.
func (idx *Index) Search(query string, k int) []SearchResult {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if k <= 0 {
		k = 5
	}

	// Generate query embedding
	queryEmbedding := idx.embedder.Embed("__query__", query)

	// Compute similarities
	type scored struct {
		id    string
		score float64
	}
	var scores []scored

	for id, emb := range idx.embeddings {
		score := CosineSimilarity(queryEmbedding.Vector, emb.Vector)
		if score > 0 { // Only include positive matches
			scores = append(scores, scored{id, score})
		}
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Take top k
	if len(scores) > k {
		scores = scores[:k]
	}

	// Build results
	results := make([]SearchResult, len(scores))
	for i, s := range scores {
		emb := idx.embeddings[s.id]
		results[i] = SearchResult{
			ID:    s.id,
			Text:  emb.Text,
			Score: s.score,
		}
	}

	return results
}

// Remove removes an embedding from the index.
func (idx *Index) Remove(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.embeddings, id)
}

// Size returns the number of embeddings in the index.
func (idx *Index) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.embeddings)
}

// Clear removes all embeddings from the index.
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.embeddings = make(map[string]*Embedding)
}

// Deduplicate finds and removes near-duplicate embeddings.
// Returns the number of duplicates removed.
func (idx *Index) Deduplicate(threshold float64) int {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if threshold <= 0 {
		threshold = 0.95 // Default: 95% similarity is duplicate
	}

	var removed int
	var kept []*Embedding

	for _, emb := range idx.embeddings {
		isDup := false
		for _, keptEmb := range kept {
			sim := CosineSimilarity(emb.Vector, keptEmb.Vector)
			if sim >= threshold {
				isDup = true
				break
			}
		}
		if isDup {
			removed++
		} else {
			kept = append(kept, emb)
		}
	}

	// Rebuild index
	idx.embeddings = make(map[string]*Embedding)
	for _, e := range kept {
		idx.embeddings[e.ID] = e
	}

	return removed
}

// GetEmbedding returns an embedding by ID.
func (idx *Index) GetEmbedding(id string) *Embedding {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.embeddings[id]
}

// SaveTo saves the index to a specific path.
func (idx *Index) SaveTo(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	embeddings := make([]*Embedding, 0, len(idx.embeddings))
	for _, e := range idx.embeddings {
		embeddings = append(embeddings, e)
	}

	indexFile := IndexFile{
		Config: IndexConfig{
			Dimensions: idx.embedder.Dimensions(),
			IndexPath:  path,
		},
		Embeddings: embeddings,
	}

	data, err := json.MarshalIndent(indexFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
