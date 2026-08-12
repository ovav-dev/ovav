package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Runbook is an auto-fix function that processes an alert
type Runbook func(ctx context.Context, a *Alert) error

// RunbookResult is the outcome of running a runbook
type RunbookResult struct {
	Success bool
	Action  string
	Error   error
}

// Dispatcher routes alerts to appropriate handlers based on level
type Dispatcher struct {
	queue    *Queue
	runners  map[string]Runbook
	mu       sync.RWMutex
}

// NewDispatcher creates a new alert dispatcher
func NewDispatcher(queue *Queue) *Dispatcher {
	return &Dispatcher{
		queue:   queue,
		runners: make(map[string]Runbook),
	}
}

// Queue returns the underlying alert queue
func (d *Dispatcher) Queue() *Queue {
	return d.queue
}

// RegisterRunbook registers a runbook for a specific source
func (d *Dispatcher) RegisterRunbook(source string, r Runbook) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.runners[source] = r
}

// Dispatch checks alert and routes:
//   - CRIT: No auto-fix, blocks, requires immediate human attention
//   - ERROR: Auto-fix via runbook if available, else alert for human
//   - WARN/INFO: Log only, no blocking action
func (d *Dispatcher) Dispatch(ctx context.Context, a *Alert) error {
	switch a.Level {
	case LevelCRIT:
		// CRIT alerts don't get auto-fixed — they block and require human action
		// Just enqueue for visibility
		return d.queue.Enqueue(a)

	case LevelERROR:
		// ERROR alerts try auto-fix first
		if a.Runbook != "" {
			runbook, ok := d.getRunner(a.Source)
			if ok && runbook != nil {
				result := d.executeRunbook(ctx, runbook, a)
				if result.Success {
					return d.queue.MarkAutoFixed(a.ID, result.Action)
				}
				// Runbook failed — keep as pending for human
				return d.queue.Enqueue(a)
			}
		}
		// No runbook available — enqueue for human
		return d.queue.Enqueue(a)

	case LevelWARN, LevelINFO:
		// WARN/INFO just get logged, not queued as blocking
		// They appear in the health report but don't interrupt work
		return d.queue.Enqueue(a)
	}

	return nil
}

// getRunner safely retrieves a runbook
func (d *Dispatcher) getRunner(source string) (Runbook, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	r, ok := d.runners[source]
	return r, ok
}

// executeRunbook runs a runbook with timeout
func (d *Dispatcher) executeRunbook(ctx context.Context, r Runbook, a *Alert) *RunbookResult {
	const runbookTimeout = 60 * time.Second

	type result struct {
		success bool
		action  string
		err     error
	}

	done := make(chan result, 1)
	go func() {
		err := r(ctx, a)
		done <- result{
			success: err == nil,
			action:  fmt.Sprintf("executed runbook: %s", a.Source),
			err:     err,
		}
	}()

	select {
	case <-ctx.Done():
		return &RunbookResult{Success: false, Action: "context cancelled"}
	case <-time.After(runbookTimeout):
		return &RunbookResult{Success: false, Action: "runbook timed out"}
	case res := <-done:
		return &RunbookResult{
			Success: res.success,
			Action:  res.action,
			Error:   res.err,
		}
	}
}

// DispatchAlerts processes a batch of alerts
func (d *Dispatcher) DispatchAlerts(ctx context.Context, alerts []*Alert) (processed int, failed int) {
	for _, a := range alerts {
		if err := d.Dispatch(ctx, a); err != nil {
			failed++
		} else {
			processed++
		}
	}
	return processed, failed
}
