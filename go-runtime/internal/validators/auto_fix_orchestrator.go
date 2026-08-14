package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AutoFixOrchestrator runs safe-fix validators with snapshot/rollback.
//
// Per ADR-011:
//   1. Snapshot files
//   2. Collect safe-fix candidates
//   3. Apply fixes
//   4. Verify no regression
//   5. Rollback on failure
type AutoFixOrchestrator struct {
	root         string
	dryRun       bool
	ceoWaiver    bool
	maxFixes     int
}

// NewAutoFixOrchestrator creates an orchestrator.
func NewAutoFixOrchestrator(root string) *AutoFixOrchestrator {
	return &AutoFixOrchestrator{
		root:     root,
		dryRun:   false,
		maxFixes: 10,
	}
}

// WithDryRun enables dry-run mode (no actual file changes).
func (a *AutoFixOrchestrator) WithDryRun() *AutoFixOrchestrator {
	a.dryRun = true
	return a
}

// WithCEOWaiver allows auto-fix to touch protected files.
func (a *AutoFixOrchestrator) WithCEOWaiver() *AutoFixOrchestrator {
	a.ceoWaiver = true
	return a
}

// Run executes the auto-fix pipeline.
// Returns a list of FixResult entries.
func (a *AutoFixOrchestrator) Run(ctx context.Context) ([]FixResult, error) {
	entries := GetSafeFixRegistry()
	if len(entries) == 0 {
		return nil, fmt.Errorf("safe-fix registry empty")
	}

	// Step 1: Snapshot
	snapDir, err := FixRegistrySnapshot(a.root, entries)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}

	results := []FixResult{}
	anyFailed := false

	// Step 2 + 3: Apply each fix
	fixCount := 0
	for _, entry := range entries {
		if fixCount >= a.maxFixes {
			break
		}

		start := time.Now()
		result := FixResult{
			ValidatorID: entry.ValidatorID,
			Description: entry.Description,
		}

		if a.dryRun {
			result.Outcome = "dry-run"
			result.DurationMs = time.Since(start).Milliseconds()
			results = append(results, result)
			continue
		}

		// Get the validator instance
		v := findValidator(entry.ValidatorID)
		if v == nil {
			result.Outcome = "skipped"
			result.Error = "validator not found"
			results = append(results, result)
			continue
		}

		// Check if implements Fixable
		fixable, ok := v.(Fixable)
		if !ok {
			result.Outcome = "skipped"
			result.Error = "validator does not implement Fixable"
			results = append(results, result)
			continue
		}

		// Apply fix
		if err := fixable.Fix(a.root); err != nil {
			result.Outcome = "failed"
			result.Error = err.Error()
			anyFailed = true
			result.DurationMs = time.Since(start).Milliseconds()
			results = append(results, result)
			continue
		}

		// Step 4: Verify (run validator again, check no new issues)
		newResult := v.Validate(ctx, a.root)
		// If validator now PASSES (no issues), fix succeeded
		if newResult.Status == "pass" {
			result.Outcome = "applied"
		} else if newResult.Status == "warn" {
			// Warnings are acceptable — fix resolved the fail but left an advisory
			result.Outcome = "applied"
		} else {
			// Failed → fix introduced regression
			result.Outcome = "rollback"
			result.Error = "fix introduced new issues: " + joinIssues(newResult.Issues)
			// Rollback
			if rbErr := FixRegistryRollback(snapDir); rbErr != nil {
				result.Error += "; rollback also failed: " + rbErr.Error()
			}
			anyFailed = true
		}
		result.DurationMs = time.Since(start).Milliseconds()
		results = append(results, result)
		fixCount++
	}

	// Step 5: Rollback if any failed (best-effort strategy: keep going)
	if anyFailed {
		// Log but don't rollback everything (best-effort)
		// Atomic strategy would rollback here
	}

	// Step 6: Log
	overallOutcome := "success"
	if a.dryRun {
		overallOutcome = "dry-run"
	}
	if anyFailed {
		overallOutcome = "partial"
	}
	log := FixResultLog{
		DeployID:   filepath.Base(snapDir),
		Operator:   os.Getenv("USER"),
		Results:    results,
		Outcome:    overallOutcome,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		DurationMs: 0, // computed from results
	}
	for _, r := range results {
		log.DurationMs += r.DurationMs
	}
	if err := AppendFixHistory(a.root, log); err != nil {
		// Non-fatal — continue
	}

	return results, nil
}

// findValidator returns the validator instance by ID.
// Currently uses the default registry; in production this should be wired
// to the actual registry used by `ovav validate`.
func findValidator(id string) Validator {
	switch id {
	case "bash_readline_bindings":
		return NewBashReadlineBindings()
	case "runtime_integrity_baseline_fresh":
		return NewIntegrityBaselineFresh(ValidationDeveloper)
	case "supply_chain":
		return NewSupplyChain(ValidationDeveloper)
	default:
		return nil
	}
}

func joinIssues(issues []string) string {
	out := ""
	for i, issue := range issues {
		if i > 0 {
			out += "; "
		}
		out += issue
	}
	return out
}