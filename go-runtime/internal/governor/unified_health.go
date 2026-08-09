// Package governor provides OVAV system governance tools in Go.
//
// Replaces Python tools/governor/*.py. Zero Python dependencies.
// Stack: Go stdlib only.
package governor

import (
	"sort"
	"time"
)

// ── Types ─────────────────────────────────────────────────────────────────

// HealthStatus represents the health state of a subsystem.
type HealthStatus string

const (
	StatusHealthy  HealthStatus = "healthy"
	StatusDegraded HealthStatus = "degraded"
	StatusCritical HealthStatus = "critical"
	StatusError    HealthStatus = "error"
	StatusUnknown  HealthStatus = "unknown"
)

// SubsystemReport holds health data from one subsystem.
type SubsystemReport struct {
	Name    string                 `json:"name"`
	Score   float64                `json:"score"` // 0-100
	Status  HealthStatus           `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// UnifiedResult holds the composite health result.
type UnifiedResult struct {
	CompositeScore float64                    `json:"composite_score"`
	Overall        HealthStatus               `json:"overall"`
	Subsystems     map[string]SubsystemReport `json:"subsystems"`
	RuleEnforced   string                     `json:"rule_enforced"`
	GeneratedAt    time.Time                  `json:"generated_at"`
}

// ── Subsystem weights ─────────────────────────────────────────────────────

type subsystemWeight struct {
	Name   string
	Weight float64
}

// defaultWeights defines the contribution of each subsystem to the composite.
// Integrity Mesh carries the most weight because it's the hard gate.
var defaultWeights = []subsystemWeight{
	{Name: "integrity_mesh", Weight: 0.40},
	{Name: "self_diagnosis", Weight: 0.35},
	{Name: "pain_scorer", Weight: 0.25},
}

// ── Composite computation ─────────────────────────────────────────────────

// ComputeUnifiedHealth aggregates subsystem reports into a single health score.
//
// Rules (matching Python unified_health.py):
//   - Weighted average: Integrity Mesh 40%, Self-Diagnosis 35%, PainScorer 25%
//   - HARD RULE: if Integrity Mesh is DEGRADED/FAILING, overall CANNOT be healthy
//   - If Self-Diagnosis is CRITICAL, overall is CRITICAL
//   - PainScorer DEGRADED → overall DEGRADED
func ComputeUnifiedHealth(reports ...SubsystemReport) UnifiedResult {
	result := UnifiedResult{
		Subsystems:  make(map[string]SubsystemReport),
		GeneratedAt: time.Now().UTC(),
	}

	// Index reports by name
	for _, r := range reports {
		result.Subsystems[r.Name] = r
	}

	// Build weight map
	weightMap := make(map[string]float64)
	for _, w := range defaultWeights {
		weightMap[w.Name] = w.Weight
	}

	// Compute weighted score
	var weightedScore, totalWeight float64
	activeCount := 0
	for _, r := range reports {
		if r.Score > 0 && r.Status != StatusError {
			w := weightMap[r.Name]
			if w == 0 {
				w = 0.33 // fallback for unknown subsystems
			}
			weightedScore += r.Score * w
			totalWeight += w
			activeCount++
		}
	}

	if totalWeight > 0 {
		result.CompositeScore = roundTo(weightedScore/totalWeight, 1)
	}

	// ── HARD RULE enforcement ──────────────────────────────────────────
	im := result.Subsystems["integrity_mesh"]
	sd := result.Subsystems["self_diagnosis"]
	ps := result.Subsystems["pain_scorer"]

	imStatus := im.Status
	if imStatus == "" {
		imStatus = StatusUnknown
	}
	sdStatus := sd.Status
	if sdStatus == "" {
		sdStatus = StatusUnknown
	}

	switch {
	case activeCount == 0:
		result.Overall = StatusUnknown
		result.RuleEnforced = "No active subsystems"

	case imStatus == StatusDegraded || imStatus == StatusCritical ||
		imStatus == StatusError || imStatus == StatusUnknown:
		result.Overall = StatusDegraded
		result.RuleEnforced = "Integrity Mesh DEGRADED → overall cannot be HEALTHY"

	case sdStatus == StatusCritical || sdStatus == StatusError:
		result.Overall = StatusCritical
		result.RuleEnforced = "Self-Diagnosis CRITICAL → overall cannot be HEALTHY"

	case sdStatus == StatusDegraded || ps.Status == StatusDegraded:
		result.Overall = StatusDegraded
		if ps.Status == StatusDegraded {
			result.RuleEnforced = "PainScorer DEGRADED → overall DEGRADED"
		} else {
			result.RuleEnforced = "Self-Diagnosis DEGRADED → overall DEGRADED"
		}

	default:
		result.Overall = StatusHealthy
		result.RuleEnforced = "All subsystems healthy"
	}

	return result
}

// ── Subsystem: Integrity Mesh (Go-native) ─────────────────────────────────

// IntegrityMeshHealth creates a subsystem report from living integrity results.
// passCount, failCount, totalCount come from internal/validators/living_integrity.
func IntegrityMeshHealth(passCount, failCount, totalCount int) SubsystemReport {
	r := SubsystemReport{
		Name: "integrity_mesh",
		Details: map[string]interface{}{
			"passed": passCount,
			"failed": failCount,
			"total":  totalCount,
		},
	}

	if totalCount == 0 {
		r.Score = 0
		r.Status = StatusUnknown
		return r
	}

	r.Score = roundTo(float64(passCount)/float64(totalCount)*100, 1)

	switch {
	case failCount == 0:
		r.Status = StatusHealthy
	case r.Score >= 80:
		r.Status = StatusDegraded
	default:
		r.Status = StatusCritical
	}

	return r
}

// ── Subsystem: Self-Diagnosis ─────────────────────────────────────────────

// SelfDiagnosisHealth creates a subsystem report from validator pipeline results.
func SelfDiagnosisHealth(okChecks, warnChecks, critChecks int) SubsystemReport {
	total := okChecks + warnChecks + critChecks
	r := SubsystemReport{
		Name: "self_diagnosis",
		Details: map[string]interface{}{
			"ok":       okChecks,
			"warnings": warnChecks,
			"critical": critChecks,
			"total":    total,
		},
	}

	if total == 0 {
		r.Score = 0
		r.Status = StatusUnknown
		return r
	}

	// Score: each OK contributes full weight, warnings half, criticals zero
	maxScore := float64(total)
	actualScore := float64(okChecks) + float64(warnChecks)*0.5
	r.Score = roundTo(actualScore/maxScore*100, 1)

	switch {
	case critChecks > 0:
		r.Status = StatusCritical
	case warnChecks > 0:
		r.Status = StatusDegraded
	default:
		r.Status = StatusHealthy
	}

	return r
}

// ── Subsystem: Pain Scorer (simplified Go-native) ────────────────────────

// PainScorerHealth creates a subsystem report from operational pain metrics.
// Replaces the legacy SNV pain_scorer.py neural system.
func PainScorerHealth(avgPain, maxPain float64, totalEvents int, escalationDetected bool) SubsystemReport {
	r := SubsystemReport{
		Name: "pain_scorer",
		Details: map[string]interface{}{
			"avg_pain":            avgPain,
			"max_pain":            maxPain,
			"total_events":        totalEvents,
			"escalation_detected": escalationDetected,
		},
	}

	if totalEvents == 0 {
		r.Score = 100
		r.Status = StatusHealthy
		return r
	}

	// Score: 100 - pain level. Lower pain = higher health.
	// Pain is 0-100 scale where 0 = no pain, 100 = extreme.
	score := 100.0 - avgPain
	if score < 0 {
		score = 0
	}
	r.Score = roundTo(score, 1)

	switch {
	case escalationDetected || maxPain >= 80:
		r.Status = StatusCritical
	case avgPain >= 50 || maxPain >= 50:
		r.Status = StatusDegraded
	default:
		r.Status = StatusHealthy
	}

	return r
}

// ── Helpers ───────────────────────────────────────────────────────────────

func roundTo(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}

// SortSubsystems returns subsystem names in deterministic order.
func SortSubsystems(result UnifiedResult) []string {
	names := make([]string, 0, len(result.Subsystems))
	for name := range result.Subsystems {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
