package install

import (
	"os"
	"path/filepath"
)

// BuildManifest builds a manifest from an install plan, classifying each entry
// as create, update, backup_required, dry_run, or blocked based on mode.
func BuildManifest(plan Plan) Manifest {
	entries := plan.Entries
	manifestEntries := make([]ManifestEntry, 0, len(entries))

	for _, entry := range entries {
		target := entry.Target
		risk := entry.TargetRisk
		if risk == "" {
			risk = classifyRisk(target)
		}
		writeEnabled := entry.WriteEnabled

		// Determine operation
		var operation string
		switch {
		case risk == "global-risk" || risk == "unknown-risk":
			operation = "blocked"
		case !writeEnabled:
			operation = "dry_run"
		case fileExists(target):
			operation = "update"
		default:
			operation = "create"
		}

		needsBackup := operation == "update"
		needsRollback := operation == "update" || operation == "create"

		manifestEntries = append(manifestEntries, ManifestEntry{
			Target:        target,
			TargetExists:  fileExists(target),
			TargetRisk:    risk,
			Operation:     operation,
			NeedsBackup:   needsBackup,
			NeedsRollback: needsRollback,
			WriteEnabled:  writeEnabled,
			Mode:          entry.Mode,
			Source:        entry.Source,
		})
	}

	// Summarize
	blocked := make([]ManifestEntry, 0)
	needsBackupCount := 0
	applyCount := 0
	for _, e := range manifestEntries {
		if e.Operation == "blocked" {
			blocked = append(blocked, e)
		}
		if e.NeedsBackup {
			needsBackupCount++
		}
		if e.Operation == "create" || e.Operation == "update" {
			applyCount++
		}
	}

	status := "pass"
	if len(blocked) > 0 {
		status = "fail"
	}

	return Manifest{
		Status:              status,
		PackID:              plan.PackID,
		Mode:                plan.Mode,
		TotalEntries:        len(manifestEntries),
		BlockedEntries:      len(blocked),
		ApplyEntries:        applyCount,
		BackupRequiredCount: needsBackupCount,
		DryRunOnly:          plan.Mode == ModeDryRun,
		Entries:             manifestEntries,
		BlockedDetails:      blocked,
	}
}

// fileExists checks if a path exists on disk.
func fileExists(path string) bool {
	// For directory targets, check as dir
	info, err := os.Stat(path)
	if err != nil {
		// Also try the path as a directory with a trailing separator
		info2, err2 := os.Stat(filepath.Clean(path))
		if err2 != nil {
			return false
		}
		return info2.IsDir()
	}
	_ = info
	return true
}
