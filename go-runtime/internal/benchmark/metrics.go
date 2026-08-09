package benchmark

import "time"

// RunResult holds the outcome of a single benchmark run
type RunResult struct {
	TaskID   string `json:"task_id"`
	Governed bool   `json:"governed"` // true = OVAV-governed, false = raw
	Model    string `json:"model"`

	// Primary metrics
	Hallucinations     int `json:"hallucinations"`      // Count of hallucinated facts/code
	SecurityViolations int `json:"security_violations"` // Count of security policy violations

	// Code quality (0.0–1.0)
	CodeQualityScore float64 `json:"code_quality_score"` // Compiles, tests pass, lint clean
	TestPassRate     float64 `json:"test_pass_rate"`     // % of generated tests that pass
	LintScore        float64 `json:"lint_score"`         // golangci-lint issues count normalized

	// Efficiency
	TokensUsed   int           `json:"tokens_used"`
	TokensInput  int           `json:"tokens_input"`
	TokensOutput int           `json:"tokens_output"`
	DurationMs   int64         `json:"duration_ms"`
	Duration     time.Duration `json:"-"`

	// Metadata
	OutputHash  string    `json:"output_hash"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	RetryCount  int       `json:"retry_count"`
	CompletedAt time.Time `json:"completed_at"`
}

// ComparisonDelta holds the A/B difference between governed and raw
type ComparisonDelta struct {
	TaskID string `json:"task_id"`
	Model  string `json:"model"`

	HallucinationsDelta     int     `json:"hallucinations_delta"`      // governed - raw (negative = improvement)
	SecurityViolationsDelta int     `json:"security_violations_delta"` // governed - raw
	CodeQualityDelta        float64 `json:"code_quality_delta"`        // governed - raw (positive = improvement)
	TestPassDelta           float64 `json:"test_pass_delta"`
	TokenEfficiencyPct      float64 `json:"token_efficiency_pct"` // % token reduction
	LatencyDeltaPct         float64 `json:"latency_delta_pct"`    // % duration change

	OVAVBetter bool `json:"ovav_better"` // true if OVAV strictly dominates
}

// Report aggregates all benchmark results
type Report struct {
	GeneratedAt time.Time `json:"generated_at"`
	OVAVVersion string    `json:"ovav_version"`
	CapsVersion string    `json:"caps_version"`
	Model       string    `json:"model"`
	TotalTasks  int       `json:"total_tasks"`

	Results []RunResult       `json:"results"`
	Deltas  []ComparisonDelta `json:"deltas"`

	Summary ReportSummary `json:"summary"`
}

// ReportSummary provides aggregate statistics
type ReportSummary struct {
	AvgHallucinationReduction     float64 `json:"avg_hallucination_reduction"` // average % reduction
	AvgSecurityViolationReduction float64 `json:"avg_security_violation_reduction"`
	AvgCodeQualityImprovement     float64 `json:"avg_code_quality_improvement"`
	AvgTokenReduction             float64 `json:"avg_token_reduction"`
	AvgLatencyChange              float64 `json:"avg_latency_change"`
	TotalTasksOVAVBetter          int     `json:"total_tasks_ovav_better"`
	TotalTasksRawBetter           int     `json:"total_tasks_raw_better"`
	TotalTasksTie                 int     `json:"total_tasks_tie"`
	OVAVWinRate                   float64 `json:"ovav_win_rate"`
}
