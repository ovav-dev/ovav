// Package memory provides live learning capabilities for autonomous AI.
package memory

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// LiveLearningEngine enables real-time autonomous learning from interactions.
type LiveLearningEngine struct {
	mu               sync.RWMutex
	vectorStore      *VectorStore
	learningRate     float64
	forgettingFactor float64
	patterns         map[string]*Pattern
	interactions     []Interaction
	lastOptimization time.Time
}

// Pattern represents a learned behavioral pattern.
type Pattern struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Frequency   int       `json:"frequency"`
	LastSeen    time.Time `json:"last_seen"`
	Context     string    `json:"context"`
	Action      string    `json:"action"`
	SuccessRate float64   `json:"success_rate"`
	Confidence  float64   `json:"confidence"`
}

// Interaction represents a user-system interaction for learning.
type Interaction struct {
	Timestamp time.Time `json:"timestamp"`
	Query     string    `json:"query"`
	Result    string    `json:"result"`
	Feedback  float64   `json:"feedback"` // -1.0 to 1.0
	Context   string    `json:"context"`
}

// NewLiveLearningEngine creates a new live learning engine.
func NewLiveLearningEngine(vs *VectorStore) *LiveLearningEngine {
	return &LiveLearningEngine{
		vectorStore:      vs,
		learningRate:     0.1,
		forgettingFactor: 0.95,
		patterns:         make(map[string]*Pattern),
		interactions:     make([]Interaction, 0),
		lastOptimization: time.Now(),
	}
}

// RecordInteraction records an interaction for learning.
func (lle *LiveLearningEngine) RecordInteraction(query, result string, feedback float64, context string) {
	lle.mu.Lock()
	defer lle.mu.Unlock()

	interaction := Interaction{
		Timestamp: time.Now(),
		Query:     query,
		Result:    result,
		Feedback:  math.Max(-1.0, math.Min(1.0, feedback)),
		Context:   context,
	}

	lle.interactions = append(lle.interactions, interaction)

	// Keep only last 1000 interactions for efficiency
	if len(lle.interactions) > 1000 {
		lle.interactions = lle.interactions[len(lle.interactions)-1000:]
	}

	// Detect and update patterns
	lle.detectPatterns(interaction)
}

// detectPatterns identifies recurring patterns in interactions.
func (lle *LiveLearningEngine) detectPatterns(interaction Interaction) {
	contextKey := extractContextKey(interaction.Context)

	if pattern, exists := lle.patterns[contextKey]; exists {
		pattern.Frequency++
		pattern.LastSeen = interaction.Timestamp

		// Update success rate with exponential moving average
		oldWeight := 1.0 - lle.learningRate
		newWeight := lle.learningRate
		pattern.SuccessRate = oldWeight*pattern.SuccessRate + newWeight*math.Max(0, interaction.Feedback)

		// Update confidence based on frequency and consistency
		pattern.Confidence = math.Min(1.0, float64(pattern.Frequency)/10.0*pattern.SuccessRate)
	} else {
		lle.patterns[contextKey] = &Pattern{
			ID:          fmt.Sprintf("pattern-%d", len(lle.patterns)+1),
			Name:        contextKey,
			Frequency:   1,
			LastSeen:    interaction.Timestamp,
			Context:     interaction.Context,
			Action:      interaction.Result,
			SuccessRate: math.Max(0, interaction.Feedback),
			Confidence:  0.1,
		}
	}
}

// extractContextKey extracts a normalized key from context.
func extractContextKey(context string) string {
	// Simple normalization - could be enhanced with NLP
	key := context
	if len(key) > 50 {
		key = key[:50]
	}
	return key
}

// PredictAction predicts the best action based on learned patterns.
func (lle *LiveLearningEngine) PredictAction(query, context string) *Prediction {
	lle.mu.RLock()
	defer lle.mu.RUnlock()

	contextKey := extractContextKey(context)

	var bestPattern *Pattern
	bestScore := -1.0

	for _, pattern := range lle.patterns {
		score := lle.calculatePatternScore(pattern, contextKey)
		if score > bestScore {
			bestScore = score
			bestPattern = pattern
		}
	}

	if bestPattern == nil || bestScore < 0.3 {
		return nil
	}

	return &Prediction{
		PatternID:  bestPattern.ID,
		Action:     bestPattern.Action,
		Confidence: bestPattern.Confidence,
		Reasoning: fmt.Sprintf("Based on %d similar interactions with %.0f%% success rate",
			bestPattern.Frequency, bestPattern.SuccessRate*100),
	}
}

// Prediction represents an AI prediction based on learned patterns.
type Prediction struct {
	PatternID  string  `json:"pattern_id"`
	Action     string  `json:"action"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

func (lle *LiveLearningEngine) calculatePatternScore(pattern *Pattern, contextKey string) float64 {
	// Base score from success rate
	score := pattern.SuccessRate * 0.5

	// Boost for recency
	hoursSince := time.Since(pattern.LastSeen).Hours()
	recencyBonus := math.Max(0, 1.0-hoursSince/168.0) // Decay over 1 week
	score += recencyBonus * 0.3

	// Boost for frequency
	freqBonus := math.Min(1.0, float64(pattern.Frequency)/20.0)
	score += freqBonus * 0.2

	return score
}

// GetInsights returns insights from learned patterns.
func (lle *LiveLearningEngine) GetInsights() []Insight {
	lle.mu.RLock()
	defer lle.mu.RUnlock()

	var insights []Insight

	for _, pattern := range lle.patterns {
		if pattern.Frequency >= 3 && pattern.Confidence > 0.5 {
			insights = append(insights, Insight{
				Type: "pattern",
				Description: fmt.Sprintf("Recurring pattern '%s' detected (%d occurrences, %.0f%% success)",
					pattern.Name, pattern.Frequency, pattern.SuccessRate*100),
				Recommendation: fmt.Sprintf("Consider automating: %s", pattern.Action),
				Priority:       pattern.Confidence,
			})
		}
	}

	// Sort by priority
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Priority > insights[j].Priority
	})

	return insights
}

// Insight represents a learned insight.
type Insight struct {
	Type           string  `json:"type"`
	Description    string  `json:"description"`
	Recommendation string  `json:"recommendation"`
	Priority       float64 `json:"priority"`
}

// Optimize performs periodic optimization of learned patterns.
func (lle *LiveLearningEngine) Optimize() error {
	lle.mu.Lock()
	defer lle.mu.Unlock()

	now := time.Now()

	// Apply forgetting factor to old patterns
	for id, pattern := range lle.patterns {
		daysSince := now.Sub(pattern.LastSeen).Hours() / 24.0
		if daysSince > 30 {
			pattern.Confidence *= math.Pow(lle.forgettingFactor, daysSince/30.0)
			if pattern.Confidence < 0.1 {
				delete(lle.patterns, id)
			}
		}
	}

	lle.lastOptimization = now
	return nil
}

// Stats returns statistics about the learning engine.
func (lle *LiveLearningEngine) Stats() LearningStats {
	lle.mu.RLock()
	defer lle.mu.RUnlock()

	return LearningStats{
		TotalInteractions:      len(lle.interactions),
		TotalPatterns:          len(lle.patterns),
		HighConfidencePatterns: lle.countHighConfidencePatterns(),
		LastOptimization:       lle.lastOptimization,
		AverageSuccessRate:     lle.calculateAverageSuccessRate(),
	}
}

// LearningStats holds learning engine statistics.
type LearningStats struct {
	TotalInteractions      int       `json:"total_interactions"`
	TotalPatterns          int       `json:"total_patterns"`
	HighConfidencePatterns int       `json:"high_confidence_patterns"`
	LastOptimization       time.Time `json:"last_optimization"`
	AverageSuccessRate     float64   `json:"average_success_rate"`
}

func (lle *LiveLearningEngine) countHighConfidencePatterns() int {
	count := 0
	for _, p := range lle.patterns {
		if p.Confidence > 0.7 {
			count++
		}
	}
	return count
}

func (lle *LiveLearningEngine) calculateAverageSuccessRate() float64 {
	if len(lle.patterns) == 0 {
		return 0
	}

	total := 0.0
	for _, p := range lle.patterns {
		total += p.SuccessRate
	}
	return total / float64(len(lle.patterns))
}
