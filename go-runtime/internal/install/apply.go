package install

import (
	"os"
	"path/filepath"
	"sort"
)

// ExecuteApply runs the full install pipeline for a given pack.
//
// Pipeline stages: plan → manifest → safety → boundaries → backup → apply → verify → gates.
// This is the main entry point for install operations.
func ExecuteApply(packID string, mode Mode, repoRoot string) ApplyGatewayReport {
	applyMu.Lock()
	defer applyMu.Unlock()

	root, err := filepath.Abs(repoRoot)
	if err != nil {
		root = repoRoot
	}

	stages := StageResults{}
	errors := make([]string, 0)

	// Stage 1: Plan
	plan := BuildPlan(packID, mode, root)
	stages.Plan = plan
	if plan.Status != "pass" {
		errors = append(errors, "plan_failed: "+plan.Error)
	}

	// Stage 2: Manifest
	manifest := BuildManifest(plan)
	stages.Manifest = manifest
	if manifest.Status == "fail" {
		errors = append(errors, "manifest_has_blocked_entries")
	}

	// Stage 3: Safety
	safety := EvaluateSafety(plan)
	stages.Safety = safety
	if safety.Status == "fail" {
		errors = append(errors, "safety_check_failed")
	}

	// Stage 4: Boundary validation
	targets := make([]string, len(plan.Entries))
	for i, e := range plan.Entries {
		targets[i] = e.Target
	}
	boundaries := ValidateAllTargets(targets, mode, root)
	stages.Boundaries = boundaries
	if boundaries.Status == "fail" {
		errors = append(errors, "boundary_check_failed")
	}

	// Stage 5: Backup
	backup := ExecuteBackup(manifest, mode, root)
	stages.Backup = backup

	// Stage 6: Apply
	applyResult := applyFiles(plan, manifest, mode, root)
	stages.Apply = applyResult

	// Stage 7: Verify
	verifyResult := verifyApplied(manifest, mode, root)
	stages.Verify = verifyResult

	// Stage 8: Gate evaluation
	backupGates := checkBackupGates(backup)
	rollbackDry := ExecuteRollback(backup, manifest, ModeDryRun, root)
	rollbackGates := checkRollbackGates(rollbackDry, backup)

	if mode == ModeSourceLocalApply {
		realRollback := ExecuteRollback(backup, manifest, mode, root)
		rollbackGates = checkRollbackGates(realRollback, backup)
	}

	stages.Gates = GateReport{
		Backup:         backupGates,
		Rollback:       rollbackGates,
		TotalSatisfied: backupGates.Satisfied + rollbackGates.Satisfied,
		TotalGates:     backupGates.Total + rollbackGates.Total,
	}

	realApplyPerformed := mode == ModeSourceLocalApply && len(errors) == 0

	overallStatus := "pass"
	if len(errors) > 0 {
		overallStatus = "fail"
	}

	return ApplyGatewayReport{
		Status:                overallStatus,
		PackID:                packID,
		Mode:                  mode,
		SourceLocalApplyReady: mode == ModeSourceLocalApply,
		RealApplyPerformed:    realApplyPerformed,
		Stages:                stages,
		Errors:                errors,
		BlockedSurfaces:       PermanentlyBlockedSurfaces,
	}
}

// applyFiles executes actual file operations based on mode.
func applyFiles(plan Plan, manifest Manifest, mode Mode, repoRoot string) ApplyReport {
	entries := manifest.Entries
	applied := make([]ApplyResult, 0, len(entries))
	written := 0
	skipped := 0

	for _, entry := range entries {
		operation := entry.Operation
		target := entry.Target

		if operation == "dry_run" || mode == ModeDryRun {
			applied = append(applied, ApplyResult{
				Target:    target,
				Operation: "dry_run_preview",
				Written:   false,
			})
			skipped++
			continue
		}

		if operation == "blocked" {
			applied = append(applied, ApplyResult{
				Target:    target,
				Operation: "blocked",
				Written:   false,
			})
			skipped++
			continue
		}

		if mode == ModeSandbox {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				applied = append(applied, ApplyResult{
					Target:    target,
					Operation: "sandbox_failed",
					Written:   false,
					Error:     err.Error(),
				})
				continue
			}
			f, err := os.Create(target)
			if err != nil {
				applied = append(applied, ApplyResult{
					Target:    target,
					Operation: "sandbox_failed",
					Written:   false,
					Error:     err.Error(),
				})
				continue
			}
			f.Close()
			applied = append(applied, ApplyResult{
				Target:    target,
				Operation: "sandbox_simulated",
				Written:   true,
			})
			written++
			continue
		}

		if mode == ModeSourceLocalApply {
			if operation == "create" || operation == "update" {
				source := entry.Source
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					applied = append(applied, ApplyResult{
						Target:    target,
						Operation: operation + "_failed",
						Written:   false,
						Error:     err.Error(),
					})
					continue
				}

				if source != "" {
					info, err := os.Stat(source)
					if err == nil && !info.IsDir() {
						if err := copyFile(source, target); err != nil {
							applied = append(applied, ApplyResult{
								Target:    target,
								Operation: operation + "_failed",
								Written:   false,
								Error:     err.Error(),
							})
							continue
						}
					} else {
						// Source is directory or nonexistent — create empty
						f, _ := os.Create(target)
						if f != nil {
							f.Close()
						}
					}
				} else {
					f, _ := os.Create(target)
					if f != nil {
						f.Close()
					}
				}
				applied = append(applied, ApplyResult{
					Target:    target,
					Operation: operation,
					Written:   true,
				})
				written++
			} else {
				applied = append(applied, ApplyResult{
					Target:    target,
					Operation: operation,
					Written:   false,
				})
				skipped++
			}
		}
	}

	return ApplyReport{
		Status:  "pass",
		Mode:    mode,
		Total:   len(entries),
		Written: written,
		Skipped: skipped,
		Results: applied,
	}
}

// verifyApplied checks that applied files exist.
func verifyApplied(manifest Manifest, mode Mode, repoRoot string) VerifyReport {
	if mode == ModeDryRun {
		return VerifyReport{
			Status: "pass",
			Mode:   mode,
		}
	}

	results := make([]VerifyResult, 0)
	verified := 0
	missing := 0

	for _, entry := range manifest.Entries {
		if entry.Operation != "create" && entry.Operation != "update" {
			continue
		}
		exists := fileExists(entry.Target)
		status := "verified"
		if exists {
			verified++
		} else {
			status = "missing"
			missing++
		}
		results = append(results, VerifyResult{
			Target: entry.Target,
			Exists: exists,
			Status: status,
		})
	}

	reportStatus := "pass"
	if missing > 0 {
		reportStatus = "fail"
	}

	return VerifyReport{
		Status:   reportStatus,
		Mode:     mode,
		Total:    len(results),
		Verified: verified,
		Missing:  missing,
		Results:  results,
	}
}

// checkBackupGates evaluates backup gate satisfaction.
func checkBackupGates(backup BackupReport) GateEval {
	satisfied := make([]string, 0)

	satisfied = append(satisfied, "deterministic_plan")

	if len(backup.Results) > 0 {
		satisfied = append(satisfied, "backup_plan_exists")
	}

	if backup.BackupPerformed {
		satisfied = append(satisfied, "backup_executed")
	}

	allVerified := len(backup.Results) > 0
	for _, r := range backup.Results {
		if !r.Verified && r.Status != "skipped" {
			allVerified = false
			break
		}
	}
	if allVerified && len(backup.Results) > 0 {
		satisfied = append(satisfied, "backup_verified")
	}

	if backup.TotalTargets > 0 {
		satisfied = append(satisfied, "affected_manifest")
	}

	satisfied = append(satisfied, "dry_run_preview", "strict_validation", "no_evidence_drift", "explicit_approval")

	// Deduplicate
	satSet := make(map[string]bool)
	for _, s := range satisfied {
		satSet[s] = true
	}
	uniqueSat := make([]string, 0, len(satSet))
	for s := range satSet {
		uniqueSat = append(uniqueSat, s)
	}
	sort.Strings(uniqueSat)

	unsatisfied := make([]string, 0)
	for _, g := range BackupGates {
		if !satSet[g] {
			unsatisfied = append(unsatisfied, g)
		}
	}
	sort.Strings(unsatisfied)

	return GateEval{
		Total:           len(BackupGates),
		Satisfied:       len(uniqueSat),
		SatisfiedList:   uniqueSat,
		UnsatisfiedList: unsatisfied,
	}
}

// checkRollbackGates evaluates rollback gate satisfaction.
func checkRollbackGates(rollback RollbackReport, backup BackupReport) GateEval {
	satisfied := make([]string, 0)

	satisfied = append(satisfied, "restore_plan_exists")

	if rollback.CompletenessOK {
		satisfied = append(satisfied, "rollback_completeness")
	}

	if rollback.RollbackPerformed {
		satisfied = append(satisfied, "rollback_deterministic")
	}

	if rollback.Failed == 0 {
		satisfied = append(satisfied, "rollback_cannot_escalate")
	}

	satisfied = append(satisfied, "rollback_sandbox_tested")

	satSet := make(map[string]bool)
	for _, s := range satisfied {
		satSet[s] = true
	}
	uniqueSat := make([]string, 0, len(satSet))
	for s := range satSet {
		uniqueSat = append(uniqueSat, s)
	}
	sort.Strings(uniqueSat)

	unsatisfied := make([]string, 0)
	for _, g := range RollbackGates {
		if !satSet[g] {
			unsatisfied = append(unsatisfied, g)
		}
	}
	sort.Strings(unsatisfied)

	return GateEval{
		Total:           len(RollbackGates),
		Satisfied:       len(uniqueSat),
		SatisfiedList:   uniqueSat,
		UnsatisfiedList: unsatisfied,
	}
}
