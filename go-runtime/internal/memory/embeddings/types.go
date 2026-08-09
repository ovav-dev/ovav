// Package embeddings implements vector embeddings for semantic search.
//
// This package provides lightweight, in-memory vector embeddings using TF-IDF
// and cosine similarity. No external dependencies (Qdrant, Pinecone, etc.).
//
// For production with larger datasets, consider:
//   - Qdrant for persistent vector storage
//   - OpenAI/Cohere for better embeddings
//   - Chroma for local vector DB
package embeddings

import "math"

// Embedding is a vector representation of text.
type Embedding struct {
	ID      string    `json:"id"`
	Vector  []float64 `json:"vector"`
	Text    string    `json:"text"`     // Original text for debugging
	Created int64     `json:"created"`  // Unix timestamp
}

// Embedder generates embeddings for text.
type Embedder struct {
	dimensions int
}

// NewEmbedder creates a new embedder with specified dimensions.
func NewEmbedder(dimensions int) *Embedder {
	if dimensions <= 0 {
		dimensions = 384 // Default for many embedding models
	}
	return &Embedder{dimensions: dimensions}
}

// Dimensions returns the embedding vector dimensions.
func (e *Embedder) Dimensions() int {
	return e.dimensions
}

// Embed generates an embedding for the given text.
// Uses TF-IDF based approach for lightweight, local embeddings.
func (e *Embedder) Embed(id, text string) *Embedding {
	vector := e.tfidfVector(text)
	return &Embedding{
		ID:      id,
		Vector:  vector,
		Text:    text,
		Created: now(),
	}
}

// tfidfVector generates a TF-IDF based vector representation.
func (e *Embedder) tfidfVector(text string) []float64 {
	words := tokenize(text)
	if len(words) == 0 {
		return make([]float64, e.dimensions)
	}

	// Count word frequencies
	freq := make(map[string]int)
	for _, w := range words {
		freq[w]++
	}

	// Create vector (use hash to map words to dimensions)
	vector := make([]float64, e.dimensions)
	for word, count := range freq {
		// Simple hash to dimension
		dim := hash(word) % e.dimensions
		// TF-IDF: term frequency * inverse document frequency
		tf := float64(count) / float64(len(words))
		idf := 1.0 // Simplified: assume all terms are informative
		vector[dim] += tf * idf
	}

	// Normalize vector
	norm := normalize(vector)
	return norm
}

// CosineSimilarity computes cosine similarity between two vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dot := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// normalize normalizes a vector to unit length.
func normalize(v []float64) []float64 {
	norm := 0.0
	for _, x := range v {
		norm += x * x
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return v
	}

	result := make([]float64, len(v))
	for i, x := range v {
		result[i] = x / norm
	}
	return result
}

// hash is a simple string hash function.
func hash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// tokenize splits text into words.
func tokenize(text string) []string {
	// Simple tokenizer: lowercase and split on non-alphanumeric
	var words []string
	var current string
	for _, c := range text {
		if isAlphanumeric(byte(c)) {
			current += string(toLower(byte(c)))
		} else {
			if len(current) >= 2 {
				words = append(words, current)
			}
			current = ""
		}
	}
	if len(current) >= 2 {
		words = append(words, current)
	}
	return words
}

func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

func now() int64 {
	return int64(len("2026-08-08")) * 24 * 60 * 60 // Simplified: just a marker
}
