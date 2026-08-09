// Governor Cycle — Autonomous governance cycle orchestrator.
//
// Replaces tools/governor/governor_cycle.py (171 LOC Python).
// Flow: LOAD → EVALUATE → DECIDE → DELEGATE → PUBLISH → REPORT
//
// Stdlib only. Thread-safe.

package governor

import (
	"fmt"
	"time"
)

// ── Cycle result types ──────────────────────────────────────────────────────

// CycleResult holds the complete result of a governance cycle.
type CycleResult struct {
	CycleID          string           `json:"cycle_id"`
	ElapsedS         float64          `json:"elapsed_s"`
	Assessments      CycleAssessments `json:"assessments"`
	DecisionsCount   int              `json:"decisions_count"`
	TasksDispatched  int              `json:"tasks_dispatched"`
	PendingTasks     int              `json:"pending_tasks"`
	DelegationStatus map[string]int   `json:"delegation_status"`
	Decisions        []Decision       `json:"decisions"`
}

// CycleAssessments holds the system assessment snapshot from a cycle.
type CycleAssessments struct {
	Integrity   string `json:"integrity"`
	Health      string `json:"health"`
	SNVPatterns int    `json:"snv_patterns"`
	FeedEvents  int    `json:"feed_events"`
	GitChanges  int    `json:"git_changes"`
}

// ── Governor Cycle ──────────────────────────────────────────────────────────

// RunGovernorCycle executes the full autonomous governance cycle.
//
// Replaces Python run_governor_cycle().
// Uses the provided TaskQueue and SessionFeed for delegation and event publishing.
// If queue or feed is nil, those operations are skipped.
func RunGovernorCycle(queue *TaskQueue, feed *SessionFeed) CycleResult {
	t0 := time.Now()
	cycleID := t0.UTC().Format("20060102-150405")

	// 1. Publish cycle start
	if feed != nil {
		feed.PublishEvent(
			fmt.Sprintf("Governance cycle #%s started", cycleID),
			"milestone", "ovav", "info", "",
		)
	}

	// 2. EVALUATE + DECIDE — gather state and run decision engine
	state := GatherSystemState()
	decisions := Decide(state)

	// 3. DELEGATE — dispatch tasks to leads (skip OVAV-internal decisions)
	var delegated []Decision
	var ovavInternal []Decision
	for _, d := range decisions {
		if d.Lead == LeadOVAV {
			ovavInternal = append(ovavInternal, d)
		} else {
			delegated = append(delegated, d)
		}
	}

	if queue != nil && len(delegated) > 0 {
		queue.DispatchFromDecisions(delegated)

		if feed != nil {
			leadSummary := ""
			for i, d := range delegated {
				if i > 0 {
					leadSummary += ", "
				}
				leadSummary += fmt.Sprintf("%s(%s)", d.Lead, d.Action)
			}
			feed.PublishEvent(
				fmt.Sprintf("Delegated %d tasks to leads: %s", len(delegated), leadSummary),
				"decision", "ovav", "info", "",
			)
		}
	}

	// 4. OVAV-internal decisions — log only
	if feed != nil {
		for _, d := range ovavInternal {
			feed.PublishEvent(
				fmt.Sprintf("OVAV processes internally: %s -> %s", d.Action, d.Target),
				"decision", "ovav", "info", "",
			)
		}
	}

	// 5. Delegation status
	pendingTasks := 0
	delegationStatus := make(map[string]int)
	if queue != nil {
		counts := queue.CountByStatus()
		for status, count := range counts {
			delegationStatus[string(status)] = count
		}
		pendingTasks = counts[TaskPending]
	}

	// 6. Feed event count
	feedEvents := 0
	if feed != nil {
		status := feed.Status()
		feedEvents = status.Events
	}

	elapsed := time.Since(t0).Seconds()

	// 7. Build assessments
	assessments := CycleAssessments{
		Integrity:   state.IntegrityStatus,
		Health:      state.HealthStatus,
		SNVPatterns: 0, // SNV patterns not tracked in Go runtime yet
		FeedEvents:  feedEvents,
		GitChanges:  state.GitChanges,
	}

	// 8. Publish summary
	if feed != nil {
		summary := fmt.Sprintf(
			"Cycle #%s completed in %.1fs. Integrity: %s. Decisions: %d. Delegated: %d. Pending: %d.",
			cycleID, elapsed, assessments.Integrity, len(decisions), len(delegated), pendingTasks,
		)
		feed.PublishEvent(summary, "milestone", "ovav", "info", "")
	}

	return CycleResult{
		CycleID:          cycleID,
		ElapsedS:         roundTo(elapsed, 2),
		Assessments:      assessments,
		DecisionsCount:   len(decisions),
		TasksDispatched:  len(delegated),
		PendingTasks:     pendingTasks,
		DelegationStatus: delegationStatus,
		Decisions:        decisions,
	}
}
