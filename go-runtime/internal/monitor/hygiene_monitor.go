package monitor

import (
	"context"
	"time"

	"github.com/ovav/ovav/internal/monitor/alerts"
	"github.com/ovav/ovav/internal/ows"
)

// HygieneMonitor wraps OWS hygiene checks as a Monitor.
// Runs the full OWS hygiene scan and emits alerts for any findings.
// This replaces: workspace_safety, merge_readiness, stale_locks validators.
type HygieneMonitor struct {
	repoRoot string
	interval time.Duration
}

// NewHygieneMonitor creates a new hygiene monitor
func NewHygieneMonitor(repoRoot string) *HygieneMonitor {
	return &HygieneMonitor{
		repoRoot: repoRoot,
		interval: 1 * time.Minute,
	}
}

func (m *HygieneMonitor) Name() string            { return "hygiene" }
func (m *HygieneMonitor) Interval() time.Duration { return m.interval }

// Run executes the OWS hygiene scan and converts findings to alerts
func (m *HygieneMonitor) Run(ctx context.Context) ([]*alerts.Alert, error) {
	result := ows.WorkspaceHygieneScan(m.repoRoot)

	// No issues = no alerts
	if result.Clean {
		return nil, nil
	}

	var alertList []*alerts.Alert

	// Generated file drift → ERROR (auto-fixable)
	if len(result.GeneratedFileDrift) > 0 {
		files := make([]string, len(result.GeneratedFileDrift))
		for i, gd := range result.GeneratedFileDrift {
			files[i] = gd.File
		}
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelERROR, "hygiene", "generated file drift detected", "fix_generated_drift").ID,
			TS:      time.Now(),
			Level:   alerts.LevelERROR,
			Source:  "hygiene",
			Issue:   "generated files out of sync with canonical source",
			Files:   files,
			Runbook: "fix_generated_drift",
			Status:  alerts.StatusPending,
		})
	}

	// Large untracked files → CRIT (blocking, no auto-fix)
	if len(result.LargeUntrackedFiles) > 0 {
		files := make([]string, len(result.LargeUntrackedFiles))
		for i, lf := range result.LargeUntrackedFiles {
			files[i] = lf.Path
		}
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelCRIT, "hygiene", "large untracked files detected", "").ID,
			TS:      time.Now(),
			Level:   alerts.LevelCRIT,
			Source:  "hygiene",
			Issue:   "large binary files detected in working tree",
			Files:   files,
			Runbook: "", // No auto-fix for binaries
			Status:  alerts.StatusPending,
		})
	}

	// Dirty after merge → WARN (not blocking)
	if len(result.DirtyAfterMerge) > 0 {
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelWARN, "hygiene", "dirty files outside worktrees", "").ID,
			TS:      time.Now(),
			Level:   alerts.LevelWARN,
			Source:  "hygiene",
			Issue:   "files modified outside active worktrees",
			Files:   result.DirtyAfterMerge,
			Runbook: "",
			Status:  alerts.StatusPending,
		})
	}

	// Broken symlinks → WARN
	if len(result.BrokenSymlinks) > 0 {
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelWARN, "hygiene", "broken symlinks", "").ID,
			TS:      time.Now(),
			Level:   alerts.LevelWARN,
			Source:  "hygiene",
			Issue:   "broken symbolic links detected",
			Files:   result.BrokenSymlinks,
			Runbook: "",
			Status:  alerts.StatusPending,
		})
	}

	// Stale locks → ERROR (auto-fixable)
	if len(result.StaleLockFiles) > 0 {
		files := make([]string, len(result.StaleLockFiles))
		for i, sl := range result.StaleLockFiles {
			files[i] = sl.Path
		}
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelERROR, "hygiene", "stale locks detected", "fix_stale_locks").ID,
			TS:      time.Now(),
			Level:   alerts.LevelERROR,
			Source:  "hygiene",
			Issue:   "expired locks (>24h TTL)",
			Files:   files,
			Runbook: "fix_stale_locks",
			Status:  alerts.StatusPending,
		})
	}

	// Git config warnings → WARN
	if len(result.GitConfigWarnings) > 0 {
		for _, w := range result.GitConfigWarnings {
			alertList = append(alertList, &alerts.Alert{
				ID:      alerts.NewAlert(alerts.LevelWARN, "hygiene", "git config issue: "+w, "").ID,
				TS:      time.Now(),
				Level:   alerts.LevelWARN,
				Source:  "hygiene",
				Issue:   w,
				Files:   nil,
				Runbook: "",
				Status:  alerts.StatusPending,
			})
		}
	}

	// Audit trail warning → WARN
	if result.AuditTrailWarning != "" {
		alertList = append(alertList, &alerts.Alert{
			ID:      alerts.NewAlert(alerts.LevelWARN, "hygiene", "audit trail issue", "").ID,
			TS:      time.Now(),
			Level:   alerts.LevelWARN,
			Source:  "hygiene",
			Issue:   result.AuditTrailWarning,
			Files:   nil,
			Runbook: "",
			Status:  alerts.StatusPending,
		})
	}

	return alertList, nil
}
