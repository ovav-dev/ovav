// Package engine implements OVAV's autonomous research orchestration with AI intelligence.
package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ovav/ovav/internal/autonomous"
	"github.com/ovav/ovav/internal/autonomous/scheduler"
)

// IntelligenceLayer adds AI-powered autonomous capabilities to the research engine.
type IntelligenceLayer struct {
	engine *Engine
}

// NewIntelligenceLayer creates a new AI intelligence layer.
func NewIntelligenceLayer(engine *Engine) *IntelligenceLayer {
	return &IntelligenceLayer{engine: engine}
}

// PredictiveAnalysis analyzes findings and predicts future trends.
func (il *IntelligenceLayer) PredictiveAnalysis(findings []autonomous.Finding) ([]Prediction, error) {
	if len(findings) == 0 {
		return nil, fmt.Errorf("no findings to analyze")
	}

	// Group findings by category and severity
	categoryMap := make(map[string][]autonomous.Finding)
	for _, f := range findings {
		categoryMap[f.Category] = append(categoryMap[f.Category], f)
	}

	var predictions []Prediction
	for category, catFindings := range categoryMap {
		prediction := il.analyzeCategory(category, catFindings)
		predictions = append(predictions, prediction)
	}

	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Confidence > predictions[j].Confidence
	})

	return predictions, nil
}

// Prediction represents an AI-generated prediction about future trends.
type Prediction struct {
	Category   string    `json:"category"`
	Trend      string    `json:"trend"`
	Direction  string    `json:"direction"` // "increasing", "decreasing", "stable"
	Confidence float64   `json:"confidence"`
	Timeframe  string    `json:"timeframe"`
	Reasoning  string    `json:"reasoning"`
	Actions    []string  `json:"actions"`
}

func (il *IntelligenceLayer) analyzeCategory(category string, findings []autonomous.Finding) Prediction {
	severityScore := 0.0
	timestamps := make([]time.Time, 0, len(findings))

	for _, f := range findings {
		switch f.Severity {
		case "critical":
			severityScore += 1.0
		case "high":
			severityScore += 0.75
		case "medium":
			severityScore += 0.5
		case "low":
			severityScore += 0.25
		}
		timestamps = append(timestamps, f.Discovered)
	}

	avgSeverity := severityScore / float64(len(findings))

	// Analyze trend direction based on timestamps
	direction := "stable"
	if len(timestamps) > 1 {
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i].Before(timestamps[j])
		})
		
		recentCount := 0
		now := time.Now()
		lastWeek := now.AddDate(0, 0, -7)
		
		for _, t := range timestamps {
			if t.After(lastWeek) {
				recentCount++
			}
		}
		
		if recentCount > len(timestamps)/2 {
			direction = "increasing"
		} else if recentCount < len(timestamps)/4 {
			direction = "decreasing"
		}
	}

	confidence := math.Min(avgSeverity*0.8+float64(len(findings))*0.02, 1.0)

	// Generate reasoning
	reasoning := fmt.Sprintf("Based on %d findings in %s with average severity %.2f", 
		len(findings), category, avgSeverity)

	// Suggest actions
	actions := il.generateActions(category, direction, avgSeverity)

	return Prediction{
		Category:   category,
		Trend:      fmt.Sprintf("%s trend in %s", strings.ToTitle(direction[:1])+direction[1:], category),
		Direction:  direction,
		Confidence: confidence,
		Timeframe:  "next_30_days",
		Reasoning:  reasoning,
		Actions:    actions,
	}
}

func (il *IntelligenceLayer) generateActions(category, direction string, severity float64) []string {
	var actions []string
	
	if severity > 0.7 {
		actions = append(actions, fmt.Sprintf("Immediate review of %s required", category))
		actions = append(actions, "Escalate to senior team for analysis")
	}
	
	if direction == "increasing" {
		actions = append(actions, fmt.Sprintf("Set up enhanced monitoring for %s", category))
		actions = append(actions, "Increase research frequency for this area")
	} else if direction == "decreasing" {
		actions = append(actions, fmt.Sprintf("Verify improvements in %s are sustainable", category))
	}
	
	actions = append(actions, "Document findings in knowledge base")
	
	return actions
}

// CorrelateFindings finds relationships between different findings.
func (il *IntelligenceLayer) CorrelateFindings(findings []autonomous.Finding) []Correlation {
	var correlations []Correlation
	
	for i := 0; i < len(findings); i++ {
		for j := i + 1; j < len(findings); j++ {
			corr := il.findCorrelation(findings[i], findings[j])
			if corr != nil {
				correlations = append(correlations, *corr)
			}
		}
	}
	
	sort.Slice(correlations, func(i, j int) bool {
		return correlations[i].Strength > correlations[j].Strength
	})
	
	return correlations
}

// Correlation represents a relationship between two findings.
type Correlation struct {
	Finding1 string  `json:"finding1_id"`
	Finding2 string  `json:"finding2_id"`
	Strength float64 `json:"strength"`
	Type     string  `json:"type"` // "causal", "temporal", "semantic"
	Reason   string  `json:"reason"`
}

func (il *IntelligenceLayer) findCorrelation(f1, f2 autonomous.Finding) *Correlation {
	strength := 0.0
	corrType := ""
	reason := ""

	// Check temporal correlation
	timeDiff := f1.Discovered.Sub(f2.Discovered)
	if timeDiff.Hours() < 24 {
		strength += 0.3
		corrType = "temporal"
		reason = "Close temporal proximity"
	}

	// Check semantic correlation (same category or related keywords)
	if f1.Category == f2.Category {
		strength += 0.4
		if corrType == "" {
			corrType = "semantic"
		}
		reason += fmt.Sprintf(", Same category: %s", f1.Category)
	}

	// Check severity correlation
	if f1.Severity == f2.Severity && f1.Severity != "" {
		strength += 0.2
		reason += fmt.Sprintf(", Same severity level: %s", f1.Severity)
	}

	if strength > 0.3 {
		return &Correlation{
			Finding1: f1.ID,
			Finding2: f2.ID,
			Strength: strength,
			Type:     corrType,
			Reason:   reason,
		}
	}

	return nil
}

// PrioritizeTargets uses AI to prioritize research targets based on importance.
func (il *IntelligenceLayer) PrioritizeTargets(targets []scheduler.Target, findings []autonomous.Finding) []PrioritizedTarget {
	targetMap := make(map[string]*scheduler.Target)
	for i := range targets {
		targetMap[targets[i].ID] = &targets[i]
	}

	findingCount := make(map[string]int)
	for _, f := range findings {
		findingCount[f.Source]++
	}

	var prioritized []PrioritizedTarget
	for _, target := range targets {
		score := il.calculateTargetScore(&target, findingCount[target.ID])
		prioritized = append(prioritized, PrioritizedTarget{
			Target: target,
			Score:  score,
			Rank:   0,
		})
	}

	sort.Slice(prioritized, func(i, j int) bool {
		return prioritized[i].Score > prioritized[j].Score
	})

	for i := range prioritized {
		prioritized[i].Rank = i + 1
	}

	return prioritized
}

// PrioritizedTarget represents a research target with priority score.
type PrioritizedTarget struct {
	Target scheduler.Target `json:"target"`
	Score  float64          `json:"score"`
	Rank   int              `json:"rank"`
}

func (il *IntelligenceLayer) calculateTargetScore(target *scheduler.Target, findingCount int) float64 {
	score := 0.0

	// Base score from frequency
	switch target.Frequency {
	case "hourly":
		score += 1.0
	case "daily":
		score += 0.8
	case "weekly":
		score += 0.6
	default:
		score += 0.4
	}

	// Boost score based on findings count
	if findingCount > 10 {
		score += 0.5
	} else if findingCount > 5 {
		score += 0.3
	} else if findingCount > 0 {
		score += 0.1
	}

	// Enable status matters
	if target.Enabled {
		score += 0.2
	}

	return score
}
