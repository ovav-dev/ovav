package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// backupRoot is the default backup directory within the repo.
var backupRoot = ".ovav/backups"

// ExecuteBackup executes backup for all targets in the manifest that need it.
//
// In dry-run mode: returns preview without copying files.
// In sandbox mode: backs up to sandbox directory within .ovav/artifacts/.
// In source-local-apply mode: real backup to .ovav/backups/ with SHA-256 verification.
func ExecuteBackup(manifest Manifest, mode Mode, repoRoot string) BackupReport {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return BackupReport{Status: "fail", Mode: mode, Timestamp: timestamp()}
	}

	ts := timestamp()

	if mode == ModeDryRun {
		targets := make([]string, 0)
		for _, e := range manifest.Entries {
			if e.NeedsBackup {
				targets = append(targets, e.Target)
			}
		}
		return BackupReport{
			Status:            "pass",
			Mode:              mode,
			Timestamp:         ts,
			BackupPerformed:   false,
			DryRunPreview:     true,
			TargetsIdentified: len(targets),
			Targets:           targets,
		}
	}

	// Determine backup directory
	var backupDir string
	if mode == ModeSandbox {
		backupDir = filepath.Join(root, ".ovav", "artifacts", "S86", "evidence", "sandbox", "backups", ts)
	} else {
		backupDir = filepath.Join(root, backupRoot, ts)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return BackupReport{
			Status:    "fail",
			Mode:      mode,
			BackupDir: backupDir,
		}
	}

	results := make([]BackupResult, 0)
	backedUp := 0
	failed := 0
	skipped := 0
	blocked := 0

	for _, entry := range manifest.Entries {
		if !entry.NeedsBackup {
			continue
		}

		target := entry.Target

		// Safety check: target must be within REPO_ROOT
		if !isSafeTarget(target, mode, root) {
			results = append(results, BackupResult{
				Target: target,
				Status: "blocked",
				Reason: "target outside safe boundaries",
			})
			blocked++
			continue
		}

		result := backupTarget(target, backupDir, root)
		results = append(results, result)

		switch result.Status {
		case "backed_up":
			backedUp++
		case "skipped":
			skipped++
		case "failed", "verification_failed":
			failed++
		case "blocked":
			blocked++
		}
	}

	status := "pass"
	if failed > 0 {
		status = "fail"
	}

	return BackupReport{
		Status:          status,
		Mode:            mode,
		Timestamp:       ts,
		BackupDir:       backupDir,
		BackupPerformed: true,
		TotalTargets:    len(results),
		BackedUp:        backedUp,
		Skipped:         skipped,
		Failed:          failed,
		Blocked:         blocked,
		Results:         results,
	}
}

// backupTarget backs up a single file target.
func backupTarget(target string, backupDir string, repoRoot string) BackupResult {
	// Compute relative path
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return BackupResult{Target: target, Status: "failed", Reason: "cannot_abs_target"}
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return BackupResult{Target: target, Status: "failed", Reason: "cannot_abs_root"}
	}

	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || strings.HasPrefix(rel, "..") {
		return BackupResult{
			Target: target,
			Status: "blocked",
			Reason: "target outside REPO_ROOT",
		}
	}

	// Check if target exists
	info, err := os.Stat(target)
	if err != nil {
		return BackupResult{
			Target: target,
			Status: "skipped",
			Reason: "target_does_not_exist",
		}
	}

	// Only backup regular files (directories are skipped)
	if info.IsDir() {
		return BackupResult{
			Target: target,
			Status: "skipped",
			Reason: "target_is_directory",
		}
	}

	sourceHash := hashFile(target)
	if sourceHash == "" {
		return BackupResult{
			Target: target,
			Status: "failed",
			Reason: "cannot_read_source",
		}
	}

	// Create backup path preserving relative directory structure
	backupPath := filepath.Join(backupDir, rel)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return BackupResult{
			Target: target,
			Status: "failed",
			Reason: fmt.Sprintf("mkdir_failed: %v", err),
		}
	}

	// Copy file
	if err := copyFile(target, backupPath); err != nil {
		return BackupResult{
			Target: target,
			Status: "failed",
			Reason: fmt.Sprintf("copy_failed: %v", err),
		}
	}

	backupHash := hashFile(backupPath)
	verified := sourceHash == backupHash

	status := "backed_up"
	if !verified {
		status = "verification_failed"
	}

	return BackupResult{
		Target:     target,
		BackupPath: backupPath,
		SourceHash: sourceHash,
		BackupHash: backupHash,
		Verified:   verified,
		Status:     status,
	}
}

// copyFile copies a file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}
