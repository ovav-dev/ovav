package benchmark

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReportGenerator creates benchmark reports
type ReportGenerator struct {
	OutputDir string
}

// NewReportGenerator creates a generator that writes to outputDir
func NewReportGenerator(outputDir string) *ReportGenerator {
	return &ReportGenerator{OutputDir: outputDir}
}

// GenerateReport creates the full A/B comparison report
func (g *ReportGenerator) GenerateReport(governed, raw []RunResult, model, ovavVer, capsVer string) (*Report, error) {
	if len(governed) != len(raw) {
		return nil, fmt.Errorf("mismatched result counts: governed=%d, raw=%d", len(governed), len(raw))
	}

	report := &Report{
		GeneratedAt: time.Now(),
		OVAVVersion: ovavVer,
		CapsVersion: capsVer,
		Model:       model,
		TotalTasks:  len(governed),
		Results:     append(governed, raw...),
	}

	// Compute deltas
	for i := range governed {
		g := governed[i]
		r := raw[i]
		if g.TaskID != r.TaskID {
			return nil, fmt.Errorf("task ID mismatch at index %d: %s vs %s", i, g.TaskID, r.TaskID)
		}

		delta := ComparisonDelta{
			TaskID:                  g.TaskID,
			Model:                   model,
			HallucinationsDelta:     g.Hallucinations - r.Hallucinations,
			SecurityViolationsDelta: g.SecurityViolations - r.SecurityViolations,
			CodeQualityDelta:        g.CodeQualityScore - r.CodeQualityScore,
			TestPassDelta:           g.TestPassRate - r.TestPassRate,
		}

		// Token efficiency: % reduction (positive = OVAV uses fewer tokens)
		if r.TokensUsed > 0 {
			delta.TokenEfficiencyPct = float64(r.TokensUsed-g.TokensUsed) / float64(r.TokensUsed) * 100
		}

		// Latency delta
		if r.DurationMs > 0 {
			delta.LatencyDeltaPct = float64(g.DurationMs-r.DurationMs) / float64(r.DurationMs) * 100
		}

		// OVAV is better if it has fewer hallucinations, fewer security violations,
		// and better or equal code quality
		delta.OVAVBetter = (g.Hallucinations <= r.Hallucinations &&
			g.SecurityViolations <= r.SecurityViolations &&
			g.CodeQualityScore >= r.CodeQualityScore)

		report.Deltas = append(report.Deltas, delta)
	}

	// Compute summary
	summary := ReportSummary{}
	var totalHallucReduction, totalSecReduction, totalCodeImprove, totalTokenReduction, totalLatency float64
	ovavBetter, rawBetter, tie := 0, 0, 0

	for _, d := range report.Deltas {
		// Hallucination reduction as % of raw
		rawH := 0
		for _, r := range raw {
			if r.TaskID == d.TaskID {
				rawH = r.Hallucinations
				break
			}
		}
		if rawH > 0 {
			totalHallucReduction += float64(-d.HallucinationsDelta) / float64(rawH) * 100
		}

		rawS := 0
		for _, r := range raw {
			if r.TaskID == d.TaskID {
				rawS = r.SecurityViolations
				break
			}
		}
		if rawS > 0 {
			totalSecReduction += float64(-d.SecurityViolationsDelta) / float64(rawS) * 100
		}

		totalCodeImprove += d.CodeQualityDelta * 100
		totalTokenReduction += d.TokenEfficiencyPct
		totalLatency += d.LatencyDeltaPct

		if d.OVAVBetter {
			ovavBetter++
		} else {
			// Check if raw is strictly better
			isRawBetter := false
			for i := range report.Deltas {
				if report.Deltas[i].TaskID == d.TaskID {
					g := governed[i]
					r := raw[i]
					isRawBetter = (r.Hallucinations < g.Hallucinations ||
						r.SecurityViolations < g.SecurityViolations ||
						r.CodeQualityScore > g.CodeQualityScore)
					break
				}
			}
			if isRawBetter {
				rawBetter++
			} else {
				tie++
			}
		}
	}

	n := float64(len(report.Deltas))
	summary.AvgHallucinationReduction = totalHallucReduction / n
	summary.AvgSecurityViolationReduction = totalSecReduction / n
	summary.AvgCodeQualityImprovement = totalCodeImprove / n
	summary.AvgTokenReduction = totalTokenReduction / n
	summary.AvgLatencyChange = totalLatency / n
	summary.TotalTasksOVAVBetter = ovavBetter
	summary.TotalTasksRawBetter = rawBetter
	summary.TotalTasksTie = tie
	if n > 0 {
		summary.OVAVWinRate = float64(ovavBetter) / n * 100
	}

	report.Summary = summary
	return report, nil
}

// WriteReport serializes the report to JSON
func (g *ReportGenerator) WriteReport(report *Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}

	filename := fmt.Sprintf("benchmark_%s_%s.json",
		report.Model,
		report.GeneratedAt.Format("20060102-150405"))
	path := filepath.Join(g.OutputDir, filename)

	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}

	return path, nil
}

// HashOutput computes a SHA-256 hash for output dedup/drift detection
func HashOutput(output string) string {
	h := sha256.Sum256([]byte(output))
	return fmt.Sprintf("%x", h)
}
