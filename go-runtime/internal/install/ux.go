package install

import (
	"fmt"
	"strings"
)

// PreviewPlan generates a human-readable preview of an install plan.
func PreviewPlan(plan Plan) string {
	var b strings.Builder

	b.WriteString("# Install Plan Preview\n\n")
	b.WriteString(fmt.Sprintf("Pack: `%s`\n", plan.PackID))
	b.WriteString(fmt.Sprintf("Mode: `%s`\n", plan.Mode))
	b.WriteString(fmt.Sprintf("Entries: %d\n\n", plan.EntryCount))

	for _, entry := range plan.Entries {
		write := "👁️"
		if entry.WriteEnabled {
			write = "✏️"
		}
		b.WriteString(fmt.Sprintf("- %s `%s` (%s)\n", write, entry.Target, entry.TargetRisk))
	}

	b.WriteString("\n")
	if plan.DryRunOnly {
		b.WriteString("> Dry-run mode — no files will be written.\n")
	} else if plan.SandboxOnly {
		b.WriteString("> Sandbox mode — simulated writes only.\n")
	} else if plan.RealApply {
		b.WriteString("> ⚠️ Source-local-apply mode — real writes to REPO_ROOT.\n")
	}

	return b.String()
}

// PreviewRisk generates a human-readable risk summary.
func PreviewRisk(safety SafetyReport) string {
	var b strings.Builder

	b.WriteString("# Risk Summary\n\n")
	b.WriteString(fmt.Sprintf("Overall safety: `%s`\n", safety.OverallSafety))
	b.WriteString(fmt.Sprintf("Blocked: %t\n", safety.HasBlocked))
	b.WriteString(fmt.Sprintf("Review required: %t\n", safety.HasReviewReq))
	b.WriteString(fmt.Sprintf("Apply allowed: %t\n\n", safety.RealApplyAllowed))

	for _, entry := range safety.Entries {
		icon := map[string]string{
			"allow":           "✅",
			"review_required": "⚠️",
			"blocked":         "🚫",
		}[entry.SafetyStatus]
		if icon == "" {
			icon = "❓"
		}
		b.WriteString(fmt.Sprintf("- %s `%s` → `%s`\n", icon, entry.Target, entry.SafetyStatus))
	}

	if len(safety.Issues) > 0 {
		b.WriteString(fmt.Sprintf("\n## Issues (%d)\n", len(safety.Issues)))
		for _, issue := range safety.Issues {
			b.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}

	return b.String()
}

// PreviewRollbackGuide generates a human-readable rollback guide.
func PreviewRollbackGuide(backup BackupReport) string {
	var b strings.Builder

	b.WriteString("# Rollback Guide\n\n")
	b.WriteString(fmt.Sprintf("Backup directory: `%s`\n", backup.BackupDir))
	b.WriteString(fmt.Sprintf("Backed up: %d\n", backup.BackedUp))
	b.WriteString(fmt.Sprintf("Failed: %d\n\n", backup.Failed))

	if len(backup.Results) == 0 {
		b.WriteString("No files were backed up.\n")
	} else {
		b.WriteString("## Backed up files\n")
		for _, r := range backup.Results {
			icon := "✅"
			if r.Status != "backed_up" {
				icon = "❌"
			}
			b.WriteString(fmt.Sprintf("- %s `%s` → `%s`\n", icon, r.Target, r.Status))
		}
	}

	b.WriteString("\n## Rollback procedure\n")
	b.WriteString("1. Verify backup integrity: check hashes match\n")
	b.WriteString("2. Run rollback with the backup report\n")
	b.WriteString("3. Verify restored files match expected state\n")

	return b.String()
}

// BuildUXPreview builds a complete UX preview.
func BuildUXPreview(plan Plan, safety SafetyReport, backup BackupReport) UXPreview {
	return UXPreview{
		Status:           "pass",
		Mode:             plan.Mode,
		PlanPreview:      PreviewPlan(plan),
		RiskPreview:      PreviewRisk(safety),
		RollbackGuide:    PreviewRollbackGuide(backup),
		RealApplyAllowed: safety.RealApplyAllowed,
		GatesRequired:    safety.RealApplyAllowed,
	}
}
