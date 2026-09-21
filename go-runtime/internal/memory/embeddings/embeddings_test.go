package embeddings

import (
	"testing"
)

func TestEmbedder_Dimensions(t *testing.T) {
	e := NewEmbedder(256)
	if e.Dimensions() != 256 {
		t.Errorf("Expected 256, got %d", e.Dimensions())
	}
}

func TestEmbedder_Embed(t *testing.T) {
	e := NewEmbedder(128)

	emb := e.Embed("test-id", "hello world")
	if emb.ID != "test-id" {
		t.Errorf("Expected id test-id, got %s", emb.ID)
	}
	if len(emb.Vector) != 128 {
		t.Errorf("Expected vector length 128, got %d", len(emb.Vector))
	}
	if emb.Text != "hello world" {
		t.Errorf("Expected text 'hello world', got %s", emb.Text)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "mismatched lengths",
			a:        []float64{1, 0},
			b:        []float64{1, 0, 0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCosineSimilarity_45Degree(t *testing.T) {
	// Cosine of 45 degrees = sqrt(2)/2 ≈ 0.707
	a := []float64{1, 1}
	b := []float64{1, 0}

	result := CosineSimilarity(a, b)
	expected := 0.707

	if result < expected-0.01 || result > expected+0.01 {
		t.Errorf("Expected ~%f, got %f", expected, result)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected int // minimum word count
	}{
		{"hello world", 2},
		{"One Two Three", 3},
		{"single", 1},
		{"", 0},
		{"a", 0}, // too short
		{"ab", 1},
	}

	for _, tt := range tests {
		result := tokenize(tt.input)
		if len(result) < tt.expected {
			t.Errorf("tokenize(%q): expected at least %d words, got %d", tt.input, tt.expected, len(result))
		}
	}
}

func TestIndex_Add(t *testing.T) {
	idx := NewIndex(128, "")

	err := idx.Add("test-1", "hello world")
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if idx.Size() != 1 {
		t.Errorf("Expected size 1, got %d", idx.Size())
	}
}

func TestIndex_Search(t *testing.T) {
	idx := NewIndex(128, "")

	// Add some cards
	idx.Add("card-1", "python programming language")
	idx.Add("card-2", "go programming language")
	idx.Add("card-3", "machine learning algorithms")

	// Search for similar
	results := idx.Search("python code", 2)

	if len(results) == 0 {
		t.Error("Expected results, got none")
	}

	// First result should be card-1 (python)
	if len(results) > 0 && results[0].ID != "card-1" {
		t.Errorf("Expected card-1 as first result, got %s", results[0].ID)
	}
}

func TestIndex_Deduplicate(t *testing.T) {
	idx := NewIndex(128, "")

	// Add similar cards
	idx.Add("card-1", "python programming language")
	idx.Add("card-2", "python programming language") // Duplicate
	idx.Add("card-3", "go programming language")     // Different

	initialSize := idx.Size()

	// Deduplicate
	removed := idx.Deduplicate(0.95)

	if removed != 1 {
		t.Errorf("Expected 1 duplicate removed, got %d", removed)
	}

	if idx.Size() != initialSize-1 {
		t.Errorf("Expected size %d, got %d", initialSize-1, idx.Size())
	}
}

func TestIndex_Clear(t *testing.T) {
	idx := NewIndex(128, "")

	idx.Add("card-1", "test")
	idx.Add("card-2", "test")

	if idx.Size() != 2 {
		t.Errorf("Expected size 2, got %d", idx.Size())
	}

	idx.Clear()

	if idx.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", idx.Size())
	}
}

func TestIndex_GetEmbedding(t *testing.T) {
	idx := NewIndex(128, "")

	idx.Add("card-1", "test content")

	emb := idx.GetEmbedding("card-1")
	if emb == nil {
		t.Error("Expected embedding, got nil")
	}

	if emb.ID != "card-1" {
		t.Errorf("Expected id card-1, got %s", emb.ID)
	}

	// Non-existent
	emb = idx.GetEmbedding("nonexistent")
	if emb != nil {
		t.Error("Expected nil for nonexistent id")
	}
}
