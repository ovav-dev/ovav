// Package tracker implements OVAV's token usage tracking with AI optimization.
package tracker

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// AutoOptimizer provides intelligent provider optimization.
type AutoOptimizer struct {
	tracker *Tracker
}

// NewAutoOptimizer creates a new auto-optimizer.
func NewAutoOptimizer(tracker *Tracker) *AutoOptimizer {
	return &AutoOptimizer{tracker: tracker}
}

// OptimizationRecommendation represents an AI-generated recommendation.
type OptimizationRecommendation struct {
	Type        string  `json:"type"` // "switch_provider", "change_model", "optimize_usage"
	ProviderID  string  `json:"provider_id"`
	Model       string  `json:"model,omitempty"`
	Savings     float64 `json:"savings_usd"`
	Reason      string  `json:"reason"`
	Priority    float64 `json:"priority"` // 0.0 to 1.0
	Action      string  `json:"action"`
}

// ProviderAnalysis contains detailed analysis of a provider.
type ProviderAnalysis struct {
	ProviderID       string            `json:"provider_id"`
	TotalCost        float64           `json:"total_cost_usd"`
	TotalTokens      int               `json:"total_tokens"`
	AverageCostPerK  float64           `json:"avg_cost_per_1k_tokens"`
	EfficiencyScore  float64           `json:"efficiency_score"` // 0.0 to 1.0
	TrendDirection   string            `json:"trend_direction"`  // "increasing", "decreasing", "stable"
	BestModel        string            `json:"best_model"`
	WorstModel       string            `json:"worst_model"`
	UsagePattern     UsagePattern      `json:"usage_pattern"`
}

// UsagePattern describes how a provider is being used.
type UsagePattern struct {
	PeakHour         int     `json:"peak_hour"`
	AverageDailyCost float64 `json:"avg_daily_cost"`
	CostVolatility   float64 `json:"cost_volatility"` // Standard deviation
	PrimaryUseCase   string  `json:"primary_use_case"`
}

// AnalyzeProvider performs deep analysis of a provider's performance.
func (ao *AutoOptimizer) AnalyzeProvider(providerID string) (*ProviderAnalysis, error) {
	since := time.Now().AddDate(0, 0, -30) // Last 30 days
	records, err := ao.tracker.GetUsageHistory(providerID, since)
	if err != nil {
		return nil, fmt.Errorf("get usage history: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no usage data for provider %s", providerID)
	}

	analysis := &ProviderAnalysis{
		ProviderID: providerID,
	}

	// Calculate totals
	modelCosts := make(map[string]float64)
	modelTokens := make(map[string]int)
	dailyCosts := make(map[string]float64)
	hourlyUsage := make(map[int]int)

	for _, r := range records {
		analysis.TotalCost += r.CostUSD
		analysis.TotalTokens += r.TotalTokens
		
		modelCosts[r.Model] += r.CostUSD
		modelTokens[r.Model] += r.TotalTokens
		
		dayKey := r.Timestamp.Format("2006-01-02")
		dailyCosts[dayKey] += r.CostUSD
		
		hourlyUsage[r.Timestamp.Hour()]++
	}

	// Average cost per 1K tokens
	if analysis.TotalTokens > 0 {
		analysis.AverageCostPerK = (analysis.TotalCost / float64(analysis.TotalTokens)) * 1000
	}

	// Find best and worst models
	var bestModel, worstModel string
	bestCost := math.MaxFloat64
	worstCost := 0.0
	
	for model, cost := range modelCosts {
		tokens := modelTokens[model]
		if tokens == 0 {
			continue
		}
		costPerK := (cost / float64(tokens)) * 1000
		
		if costPerK < bestCost {
			bestCost = costPerK
			bestModel = model
		}
		if costPerK > worstCost {
			worstCost = costPerK
			worstModel = model
		}
	}
	
	analysis.BestModel = bestModel
	analysis.WorstModel = worstModel

	// Calculate efficiency score (lower cost = higher efficiency)
	maxPossibleCost := float64(analysis.TotalTokens) * 0.05 // Assume $0.05 per 1K as max
	if maxPossibleCost > 0 {
		analysis.EfficiencyScore = math.Max(0, 1.0-(analysis.TotalCost/maxPossibleCost))
	}

	// Determine trend
	days := len(dailyCosts)
	if days >= 7 {
		var costs []float64
		for _, cost := range dailyCosts {
			costs = append(costs, cost)
		}
		sort.Float64s(costs)
		
		firstHalf := costs[:len(costs)/2]
		secondHalf := costs[len(costs)/2:]
		
		avgFirst := average(firstHalf)
		avgSecond := average(secondHalf)
		
		if avgSecond > avgFirst*1.2 {
			analysis.TrendDirection = "increasing"
		} else if avgSecond < avgFirst*0.8 {
			analysis.TrendDirection = "decreasing"
		} else {
			analysis.TrendDirection = "stable"
		}
	}

	// Usage pattern
	if len(hourlyUsage) > 0 {
		peakHour := 0
		peakCount := 0
		for hour, count := range hourlyUsage {
			if count > peakCount {
				peakCount = count
				peakHour = hour
			}
		}
		analysis.UsagePattern.PeakHour = peakHour
	}

	var dailyValues []float64
	for _, cost := range dailyCosts {
		dailyValues = append(dailyValues, cost)
	}
	analysis.UsagePattern.AverageDailyCost = average(dailyValues)
	analysis.UsagePattern.CostVolatility = stdDev(dailyValues)

	return analysis, nil
}

// GenerateRecommendations generates optimization recommendations.
func (ao *AutoOptimizer) GenerateRecommendations() ([]OptimizationRecommendation, error) {
	providers := ao.tracker.ListProviders()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}

	var recommendations []OptimizationRecommendation

	// Analyze each provider
	analyses := make(map[string]*ProviderAnalysis)
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		
		analysis, err := ao.AnalyzeProvider(p.ID)
		if err != nil {
			continue
		}
		analyses[p.ID] = analysis
	}

	// Find cheapest provider for comparison
	var cheapestProvider string
	cheapestCostPerK := math.MaxFloat64
	for id, analysis := range analyses {
		if analysis.AverageCostPerK < cheapestCostPerK && analysis.AverageCostPerK > 0 {
			cheapestCostPerK = analysis.AverageCostPerK
			cheapestProvider = id
		}
	}

	// Generate recommendations
	for id, analysis := range analyses {
		// Recommendation 1: Switch to cheaper provider if significant difference
		if cheapestProvider != "" && id != cheapestProvider {
			costDiff := analysis.AverageCostPerK - cheapestCostPerK
			if costDiff > cheapestCostPerK*0.3 { // 30% more expensive
				potentialSavings := (costDiff / 1000) * float64(analysis.TotalTokens)
				recommendations = append(recommendations, OptimizationRecommendation{
					Type:       "switch_provider",
					ProviderID: id,
					Savings:    potentialSavings,
					Reason:     fmt.Sprintf("%.1f%% more expensive than %s", 
						(costDiff/cheapestCostPerK)*100, cheapestProvider),
					Priority:   math.Min(1.0, costDiff/cheapestCostPerK),
					Action:     fmt.Sprintf("Consider switching from %s to %s for cost savings", id, cheapestProvider),
				})
			}
		}

		// Recommendation 2: Optimize model usage
		if analysis.WorstModel != "" && analysis.BestModel != "" {
			worstCost := getCostForModel(analyses[id], analysis.WorstModel)
			bestCost := getCostForModel(analyses[id], analysis.BestModel)
			
			if worstCost > bestCost*1.5 { // 50% more expensive
				recommendations = append(recommendations, OptimizationRecommendation{
					Type:       "change_model",
					ProviderID: id,
					Model:      analysis.WorstModel,
					Savings:    worstCost - bestCost,
					Reason:     fmt.Sprintf("Model %s is significantly more expensive than %s", 
						analysis.WorstModel, analysis.BestModel),
					Priority:   0.8,
					Action:     fmt.Sprintf("Switch from %s to %s on %s", 
						analysis.WorstModel, analysis.BestModel, id),
				})
			}
		}

		// Recommendation 3: Address increasing cost trend
		if analysis.TrendDirection == "increasing" && analysis.UsagePattern.CostVolatility > 0 {
			recommendations = append(recommendations, OptimizationRecommendation{
				Type:       "optimize_usage",
				ProviderID: id,
				Savings:    analysis.TotalCost * 0.2, // Estimate 20% savings
				Reason:     "Costs are trending upward with high volatility",
				Priority:   0.7,
				Action:     "Review usage patterns and implement caching or batching",
			})
		}
	}

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	return recommendations, nil
}

func getCostForModel(analysis *ProviderAnalysis, model string) float64 {
	// This would need access to actual model costs
	// For now, return a placeholder
	return analysis.AverageCostPerK
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := average(values)
	sumSqDiff := 0.0
	for _, v := range values {
		diff := v - avg
		sumSqDiff += diff * diff
	}
	return math.Sqrt(sumSqDiff / float64(len(values)-1))
}

// PredictFutureCosts predicts future costs based on trends.
func (ao *AutoOptimizer) PredictFutureCosts(providerID string, days int) (float64, error) {
	since := time.Now().AddDate(0, 0, -30)
	records, err := ao.tracker.GetUsageHistory(providerID, since)
	if err != nil {
		return 0, fmt.Errorf("get usage history: %w", err)
	}

	if len(records) == 0 {
		return 0, fmt.Errorf("no usage data")
	}

	// Simple linear regression for prediction
	dailyCosts := make(map[string]float64)
	for _, r := range records {
		dayKey := r.Timestamp.Format("2006-01-02")
		dailyCosts[dayKey] += r.CostUSD
	}

	var costs []float64
	for _, cost := range dailyCosts {
		costs = append(costs, cost)
	}

	if len(costs) < 2 {
		return average(costs) * float64(days), nil
	}

	// Calculate trend
	avgCost := average(costs)
	trend := (costs[len(costs)-1] - costs[0]) / float64(len(costs))
	
	predictedDaily := avgCost + trend*float64(days)/2
	return predictedDaily * float64(days), nil
}
