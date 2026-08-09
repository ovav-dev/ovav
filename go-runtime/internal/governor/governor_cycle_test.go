package governor

import (
	"testing"
)

func TestRunGovernorCycle_HealthySystem(t *testing.T) {
	dir := t.TempDir()
	queue := NewTaskQueue()
	feed := NewSessionFeed(dir)

	result := RunGovernorCycle(queue, feed)

	if result.CycleID == "" {
		t.Error("cycle_id should not be empty")
	}
	if result.ElapsedS < 0 {
		t.Errorf("elapsed should be >= 0, got %.2f", result.ElapsedS)
	}
	if result.Assessments.Integrity == "" {
		t.Error("integrity assessment should not be empty")
	}

	// Verify events were published
	status := feed.Status()
	if status.Events < 2 {
		t.Errorf("expected at least 2 feed events (start + summary), got %d", status.Events)
	}
}

func TestRunGovernorCycle_NilQueue(t *testing.T) {
	dir := t.TempDir()
	feed := NewSessionFeed(dir)

	// Should not panic with nil queue.
	// Note: TasksDispatched reflects decisions generated, not tasks enqueued.
	// Decisions are computed independently of the queue. With nil queue,
	// tasks are generated but not dispatched — PendingTasks should be 0.
	result := RunGovernorCycle(nil, feed)

	if result.CycleID == "" {
		t.Error("cycle_id should not be empty")
	}
	if result.PendingTasks != 0 {
		t.Errorf("pending_tasks should be 0 with nil queue (no queue to dispatch to), got %d", result.PendingTasks)
	}
}

func TestRunGovernorCycle_NilFeed(t *testing.T) {
	queue := NewTaskQueue()

	// Should not panic with nil feed
	result := RunGovernorCycle(queue, nil)

	if result.CycleID == "" {
		t.Error("cycle_id should not be empty")
	}
}

func TestRunGovernorCycle_NilBoth(t *testing.T) {
	// Should not panic with both nil
	result := RunGovernorCycle(nil, nil)

	if result.CycleID == "" {
		t.Error("cycle_id should not be empty")
	}
}

func TestRunGovernorCycle_Assessments(t *testing.T) {
	dir := t.TempDir()
	queue := NewTaskQueue()
	feed := NewSessionFeed(dir)

	result := RunGovernorCycle(queue, feed)

	// Assessments should have non-empty values
	if result.Assessments.Integrity == "" {
		t.Error("integrity should not be empty")
	}
	if result.Assessments.Health == "" {
		t.Error("health should not be empty")
	}
}

func TestRunGovernorCycle_DelegationStatus(t *testing.T) {
	dir := t.TempDir()
	queue := NewTaskQueue()
	feed := NewSessionFeed(dir)

	// Pre-populate queue with a task
	queue.Dispatch("pre-existing", LeadThavren, PriorityHigh, DomainSystem, "test", "done", "test")

	result := RunGovernorCycle(queue, feed)

	// Should report the pre-existing pending task
	if result.PendingTasks < 1 {
		t.Errorf("pending_tasks should be >= 1, got %d", result.PendingTasks)
	}
}
