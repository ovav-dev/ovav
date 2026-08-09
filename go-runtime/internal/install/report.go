package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteEvidence writes the gateway report to evidence files.
//
// Produces:
//   - Full gateway report (JSON)
//   - Human-readable summary (Markdown)
//   - Safety report (JSON)
//   - Gate satisfaction report (JSON)
func WriteEvidence(report ApplyGatewayReport, segment string, repoRoot string) (EvidenceReport, error) {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return EvidenceReport{}, fmt.Errorf("install: abs root: %w", err)
	}

	evidenceDir := filepath.Join(root, ".ovav", "artifacts", segment, "evidence")

	// Ensure evidence dir is within REPO_ROOT
	absEvidence, err := filepath.Abs(evidenceDir)
	if err != nil {
		return EvidenceReport{}, fmt.Errorf("install: abs evidence dir: %w", err)
	}
	if !strings.HasPrefix(absEvidence, root) {
		return EvidenceReport{}, fmt.Errorf("install: evidence dir %s is outside REPO_ROOT %s", absEvidence, root)
	}

	if err := os.MkdirAll(evidenceDir, 0755); err != nil {
		return EvidenceReport{}, fmt.Errorf("install: mkdir evidence: %w", err)
	}

	ts := timestamp()
	written := make([]string, 0)

	// 1. Full gateway report (JSON)
	gatewayPath := filepath.Join(evidenceDir, fmt.Sprintf("INSTALL_GATEWAY_REPORT_%s.json", ts))
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return EvidenceReport{}, fmt.Errorf("install: marshal gateway report: %w", err)
	}
	if err := os.WriteFile(gatewayPath, data, 0644); err != nil {
		return EvidenceReport{}, fmt.Errorf("install: write gateway report: %w", err)
	}
	written = append(written, gatewayPath)

	// 2. Human-readable summary (Markdown)
	summaryPath := filepath.Join(evidenceDir, fmt.Sprintf("INSTALL_GATEWAY_SUMMARY_%s.md", ts))
	summary := buildSummaryMD(report, segment)
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return EvidenceReport{}, fmt.Errorf("install: write summary: %w", err)
	}
	written = append(written, summaryPath)

	// 3. Safety report (JSON)
	safetyPath := filepath.Join(evidenceDir, fmt.Sprintf("INSTALL_SAFETY_REPORT_%s.json", ts))
	safetyData, _ := json.MarshalIndent(report.Stages.Safety, "", "  ")
	os.WriteFile(safetyPath, safetyData, 0644)
	written = append(written, safetyPath)

	// 4. Gate satisfaction report (JSON)
	gatesPath := filepath.Join(evidenceDir, fmt.Sprintf("INSTALL_GATES_REPORT_%s.json", ts))
	gatesData, _ := json.MarshalIndent(report.Stages.Gates, "", "  ")
	os.WriteFile(gatesPath, gatesData, 0644)
	written = append(written, gatesPath)

	return EvidenceReport{
		Status:       "pass",
		Segment:      segment,
		Timestamp:    ts,
		FilesWritten: len(written),
		Paths:        written,
	}, nil
}

// buildSummaryMD builds a human-readable Markdown summary.
func buildSummaryMD(report ApplyGatewayReport, segment string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s — Install Gateway Report\n\n", segment))
	b.WriteString(fmt.Sprintf("**Pack:** `%s`\n", report.PackID))
	b.WriteString(fmt.Sprintf("**Mode:** `%s`\n", report.Mode))
	b.WriteString(fmt.Sprintf("**Status:** `%s`\n\n", report.Status))

	b.WriteString("## Gate Satisfaction\n")
	gates := report.Stages.Gates
	b.WriteString(fmt.Sprintf("- Backup gates: %d/%d\n", gates.Backup.Satisfied, gates.Backup.Total))
	b.WriteString(fmt.Sprintf("- Rollback gates: %d/%d\n", gates.Rollback.Satisfied, gates.Rollback.Total))
	b.WriteString(fmt.Sprintf("- **Total gates satisfied: %d/%d**\n\n", gates.TotalSatisfied, gates.TotalGates))

	apply := report.Stages.Apply
	b.WriteString("## Apply Results\n")
	b.WriteString(fmt.Sprintf("- Files written: %d\n", apply.Written))
	b.WriteString(fmt.Sprintf("- Files skipped: %d\n\n", apply.Skipped))

	backup := report.Stages.Backup
	b.WriteString("## Backup\n")
	b.WriteString(fmt.Sprintf("- Backed up: %d\n", backup.BackedUp))
	b.WriteString(fmt.Sprintf("- Failed: %d\n", backup.Failed))
	b.WriteString(fmt.Sprintf("- Directory: `%s`\n\n", backup.BackupDir))

	if len(report.Errors) > 0 {
		b.WriteString("## Errors\n")
		for _, e := range report.Errors {
			b.WriteString(fmt.Sprintf("- %s\n", e))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Blocked Surfaces Preserved\n")
	b.WriteString(fmt.Sprintf("- %d surfaces blocked\n", len(report.BlockedSurfaces)))
	b.WriteString(fmt.Sprintf("- Source-local-apply ready: `%t`\n", report.SourceLocalApplyReady))
	b.WriteString(fmt.Sprintf("- Real apply performed: `%t`\n", report.RealApplyPerformed))

	return b.String()
}
