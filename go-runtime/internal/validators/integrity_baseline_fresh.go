package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IntegrityBaselineFresh checks that the runtime integrity baseline is recent
// enough to be trustworthy. A stale baseline means the validator may be
// comparing against outdated file hashes.
//
// Per ADR-006 (baseline versioning): the baseline should be regenerated
// whenever a protected surface changes. If the baseline is older than the
// threshold (default 7 days), this validator warns so the operator knows
// to refresh.
//
// Two-file model:
//   - baseline.json (current operational)
//   - baseline.pinned.json (last CEO-approved)
//
// Both are tracked in git per the new .gitignore exception.
type IntegrityBaselineFresh struct {
	mode           ValidationMode
	maxAge         time.Duration
	pinnedRequired bool
}

// NewIntegrityBaselineFresh constructs a freshness validator with sensible defaults.
func NewIntegrityBaselineFresh(modes ...ValidationMode) *IntegrityBaselineFresh {
	mode := ValidationDeveloper
	if len(modes) > 0 {
		mode = modes[0]
	}
	return &IntegrityBaselineFresh{
		mode:           mode,
		maxAge:         7 * 24 * time.Hour, // 7 days
		pinnedRequired: false,              // can flip to true once pinning workflow exists
	}
}

// WithMaxAge overrides the staleness threshold (testing only).
func (i *IntegrityBaselineFresh) WithMaxAge(d time.Duration) *IntegrityBaselineFresh {
	i.maxAge = d
	return i
}

// WithPinnedRequired enables the pinned-baseline existence check.
func (i *IntegrityBaselineFresh) WithPinnedRequired() *IntegrityBaselineFresh {
	i.pinnedRequired = true
	return i
}

func (i *IntegrityBaselineFresh) ID() string                  { return "integrity_baseline_fresh" }
func (i *IntegrityBaselineFresh) Name() string                { return "Integrity Baseline Freshness" }
func (i *IntegrityBaselineFresh) Description() string {
	return "Verifies runtime integrity baseline is recent and (optionally) pinned"
}
func (i *IntegrityBaselineFresh) Weight() int { return 10 }

// Validate runs the freshness check.
//
// Status semantics:
//   - baseline missing        → FAIL (governance failure)
//   - baseline > maxAge old    → WARN in dev mode, FAIL in gate mode
//   - pinned missing + required → WARN (not yet adopted)
func (i *IntegrityBaselineFresh) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	status := "pass"

	if err := ctx.Err(); err != nil {
		issues = append(issues, "validation context unavailable")
		status = "fail"
	}

	baselineFile := filepath.Join(root, ".ovav", "integrity_backups", "baseline.json")
	pinnedFile := filepath.Join(root, ".ovav", "integrity_backups", "baseline.pinned.json")

	// Check baseline.json exists
	info, err := os.Stat(baselineFile)
	if err != nil {
		issue := fmt.Sprintf("integrity baseline missing: %s — regenerate via 'ovav integrity baseline --write' (in feature worktree)", baselineFile)
		issues = append(issues, issue)
		if i.mode == ValidationGate {
			status = "fail"
		} else if status != "fail" {
			status = "warn"
		}
	} else {
		// Check age
		age := time.Since(info.ModTime())
		if age > i.maxAge {
			issue := fmt.Sprintf("integrity baseline stale: %s (age %s > max %s) — regenerate", baselineFile, age.Truncate(time.Hour), i.maxAge)
			issues = append(issues, issue)
			if i.mode == ValidationGate {
				status = "fail"
			} else if status != "fail" {
				status = "warn"
			}
		}

		// Check pinned exists if required
		if i.pinnedRequired {
			if _, err := os.Stat(pinnedFile); err != nil {
				issues = append(issues, fmt.Sprintf("pinned baseline missing: %s — create via 'ovav integrity pin'", pinnedFile))
				if status == "pass" {
					status = "warn"
				}
			}
		}
	}

	return Result{
		ID:          i.ID(),
		Name:        i.Name(),
		Status:      status,
		Weight:      i.Weight(),
		Issues:      issues,
		Description: fmt.Sprintf("max age: %s, pinned required: %v, mode: %s", i.maxAge, i.pinnedRequired, i.mode),
		Duration:    time.Since(start),
	}
}