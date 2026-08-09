package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExecuteRollback restores files from backup to their original locations.
//
// Only targets that were successfully backed up are eligible for rollback.
// Verifies integrity after restore using SHA-256 hash comparison.
func ExecuteRollback(backup BackupReport, manifest Manifest, mode Mode, repoRoot string) RollbackReport {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return RollbackReport{Status: "fail", Mode: mode, Timestamp: timestamp()}
	}

	ts := timestamp()

	if mode == ModeDryRun {
		backedUpResults := make([]BackupResult, 0)
		for _, r := range backup.Results {
			if r.Status == "backed_up" {
				backedUpResults = append(backedUpResults, r)
			}
		}
		targets := make([]string, len(backedUpResults))
		for i, r := range backedUpResults {
			targets[i] = r.Target
		}
		return RollbackReport{
			Status:            "pass",
			Mode:              mode,
			Timestamp:         ts,
			RollbackPerformed: false,
			DryRunPreview:     true,
			TargetsAvailable:  len(backedUpResults),
			Targets:           targets,
		}
	}

	rollbackResults := make([]RollbackResult, 0)
	restored := 0
	failed := 0

	// Build completeness map from manifest
	manifestTargets := make(map[string]ManifestEntry)
	for _, e := range manifest.Entries {
		if e.NeedsBackup || e.NeedsRollback {
			manifestTargets[e.Target] = e
		}
	}

	for _, entry := range backup.Results {
		if entry.Status != "backed_up" {
			continue
		}

		if entry.BackupPath == "" || entry.SourceHash == "" {
			rollbackResults = append(rollbackResults, RollbackResult{
				Target: entry.Target,
				Status: "skipped",
				Reason: "missing_backup_path_or_hash",
			})
			continue
		}

		result := rollbackTarget(entry.Target, entry.BackupPath, entry.SourceHash, root)
		rollbackResults = append(rollbackResults, result)
		if result.Status == "restored" {
			restored++
		} else {
			failed++
		}
	}

	// Completeness check: every manifest target that needed backup must have a rollback result
	completenessOK := true
	for t := range manifestTargets {
		found := false
		for _, r := range rollbackResults {
			if r.Target == t {
				found = true
				break
			}
		}
		if !found {
			completenessOK = false
			break
		}
	}

	status := "pass"
	if failed > 0 || !completenessOK {
		status = "fail"
	}

	return RollbackReport{
		Status:            status,
		Mode:              mode,
		Timestamp:         ts,
		RollbackPerformed: true,
		TotalTargets:      len(rollbackResults),
		Restored:          restored,
		Failed:            failed,
		CompletenessOK:    completenessOK,
		Results:           rollbackResults,
	}
}

// rollbackTarget restores a single file from backup to its original location.
func rollbackTarget(target string, backupPath string, expectedHash string, repoRoot string) RollbackResult {
	// Boundary check: target must be within REPO_ROOT
	if !isSafeTarget(target, ModeSourceLocalApply, repoRoot) {
		return RollbackResult{
			Target: target,
			Status: "blocked",
			Reason: "target outside safe boundaries — rollback cannot escalate",
		}
	}

	if _, err := os.Stat(backupPath); err != nil {
		return RollbackResult{
			Target: target,
			Status: "failed",
			Reason: "backup_file_missing",
		}
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return RollbackResult{
			Target: target,
			Status: "failed",
			Reason: fmt.Sprintf("mkdir_failed: %v", err),
		}
	}

	// Restore from backup
	if err := copyFile(backupPath, target); err != nil {
		return RollbackResult{
			Target: target,
			Status: "failed",
			Reason: fmt.Sprintf("restore_failed: %v", err),
		}
	}

	restoredHash := hashFile(target)
	verified := restoredHash == expectedHash

	status := "restored"
	if !verified {
		status = "verification_failed"
	}

	return RollbackResult{
		Target:       target,
		BackupPath:   backupPath,
		ExpectedHash: expectedHash,
		RestoredHash: restoredHash,
		Verified:     verified,
		Status:       status,
	}
}
