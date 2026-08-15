package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PinnedBaselineDrift compares the current baseline against the pinned
// baseline (last CEO-approved). If any protected surface differs, the
// validator fails — CEO approval is required to update the pinned baseline.
//
// Per ADR-006: pinned baseline represents the "last known good" state.
// Drift = unapproved surface change. Two scenarios:
//
//  1. Current baseline != pinned → surface changed since last CEO approval
//  2. Pinned baseline missing → no CEO approval has been recorded yet
//
// This validator is the "drift firewall" for runtime integrity.
type PinnedBaselineDrift struct {
	mode ValidationMode
}

func NewPinnedBaselineDrift(modes ...ValidationMode) *PinnedBaselineDrift {
	mode := ValidationDeveloper
	if len(modes) > 0 {
		mode = modes[0]
	}
	return &PinnedBaselineDrift{mode: mode}
}

func (p *PinnedBaselineDrift) ID() string   { return "pinned_baseline_drift" }
func (p *PinnedBaselineDrift) Name() string { return "Pinned Baseline Drift" }
func (p *PinnedBaselineDrift) Description() string {
	return "Compares current runtime baseline against last CEO-approved pinned baseline"
}
func (p *PinnedBaselineDrift) Weight() int { return 15 }

// Validate performs the comparison.
//
// Drift categories:
//   - missing in pinned: surface newly protected (no CEO approval yet) → WARN
//   - missing in current: surface removed since pinning → WARN
//   - hash mismatch: surface modified since pinning → FAIL
func (p *PinnedBaselineDrift) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	status := "pass"

	if err := ctx.Err(); err != nil {
		issues = append(issues, "validation context unavailable")
		status = "fail"
	}

	baselineFile := filepath.Join(root, ".ovav", "integrity_backups", "baseline.json")
	pinnedFile := filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json")

	current, err := loadBaselineFile(baselineFile)
	if err != nil {
		issues = append(issues, fmt.Sprintf("current baseline unreadable: %v", err))
		return finishPinnedResult(p, start, issues, "fail")
	}

	pinned, pinnedErr := loadBaselineFile(pinnedFile)
	if pinnedErr != nil {
		if os.IsNotExist(pinnedErr) {
			issues = append(issues, "no pinned baseline yet — create via 'ovav integrity pin' (CEO approval required)")
			return finishPinnedResult(p, start, issues, "warn")
		}
		issues = append(issues, fmt.Sprintf("pinned baseline unreadable: %v", pinnedErr))
		return finishPinnedResult(p, start, issues, "fail")
	}

	// Compare: every pinned file must exist in current with same hash
	for rel, expectedHash := range pinned.Files {
		actualHash, ok := current.Files[rel]
		if !ok {
			issues = append(issues, fmt.Sprintf("pinned surface missing in current: %s", rel))
			status = "fail"
			continue
		}
		if actualHash != expectedHash {
			expShort := expectedHash
			if len(expShort) > 8 {
				expShort = expShort[:8]
			}
			actShort := actualHash
			if len(actShort) > 8 {
				actShort = actShort[:8]
			}
			issues = append(issues, fmt.Sprintf("pinned surface drift: %s (expected %s, got %s)", rel, expShort, actShort))
			status = "fail"
		}
	}

	// New surfaces in current not in pinned → WARN (not yet approved)
	for rel := range current.Files {
		if _, ok := pinned.Files[rel]; !ok {
			issues = append(issues, fmt.Sprintf("new surface not yet pinned: %s", rel))
			if status == "pass" {
				status = "warn"
			}
		}
	}

	return finishPinnedResult(p, start, issues, status)
}

// loadBaselineFile reads a baseline JSON file.
func loadBaselineFile(path string) (IntegrityBaseline, error) {
	var baseline IntegrityBaseline
	data, err := os.ReadFile(path)
	if err != nil {
		return baseline, err
	}
	if err := json.Unmarshal(data, &baseline); err != nil {
		return baseline, fmt.Errorf("parse: %w", err)
	}
	return baseline, nil
}

// finishPinnedResult builds the Result struct.
func finishPinnedResult(p *PinnedBaselineDrift, start time.Time, issues []string, status string) Result {
	return Result{
		ID:          p.ID(),
		Name:        p.Name(),
		Status:      status,
		Weight:      p.Weight(),
		Issues:      issues,
		Description: fmt.Sprintf("mode: %s", p.mode),
		Duration:    time.Since(start),
	}
}
