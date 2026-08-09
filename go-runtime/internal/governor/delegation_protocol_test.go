package governor

import (
	"testing"
)

func TestTaskQueue_Dispatch(t *testing.T) {
	q := NewTaskQueue()

	task := q.Dispatch("Fix integrity mesh", LeadThavren, PriorityCritical, DomainSystem, "integrity_mesh", "100% healthy", "decision_engine")

	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
	if task.Status != TaskPending {
		t.Errorf("status = %s, want pending", task.Status)
	}
	if task.Lead != LeadThavren {
		t.Errorf("lead = %s, want thavren", task.Lead)
	}
	if task.Priority != PriorityCritical {
		t.Errorf("priority = %s, want CRITICAL", task.Priority)
	}
	if q.Count() != 1 {
		t.Errorf("count = %d, want 1", q.Count())
	}
}

func TestTaskQueue_DispatchFromDecisions(t *testing.T) {
	q := NewTaskQueue()

	decisions := []Decision{
		{Priority: PriorityCritical, Action: ActionRepair, Target: "integrity", Lead: LeadThavren, Domain: DomainSystem, Reason: "broken", ExpectedOutcome: "fixed"},
		{Priority: PriorityHigh, Action: ActionSync, Target: "contract", Lead: LeadThavren, Domain: DomainSystem, Reason: "drift", ExpectedOutcome: "synced"},
	}

	tasks := q.DispatchFromDecisions(decisions)

	if len(tasks) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(tasks))
	}
	if q.Count() != 2 {
		t.Errorf("count = %d, want 2", q.Count())
	}
}

func TestTaskQueue_Accept(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("test", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")

	err := q.Accept(task.ID)
	if err != nil {
		t.Fatalf("accept failed: %v", err)
	}

	retrieved, ok := q.Get(task.ID)
	if !ok {
		t.Fatal("task not found after accept")
	}
	if retrieved.Status != TaskAccepted {
		t.Errorf("status = %s, want accepted", retrieved.Status)
	}
}

func TestTaskQueue_AcceptWrongStatus(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("test", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")
	q.Accept(task.ID)

	// Second accept should fail
	err := q.Accept(task.ID)
	if err == nil {
		t.Error("second accept should fail")
	}
}

func TestTaskQueue_Complete(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("test", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")

	err := q.Complete(task.ID, "done", "evidence.log")
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	retrieved, _ := q.Get(task.ID)
	if retrieved.Status != TaskCompleted {
		t.Errorf("status = %s, want completed", retrieved.Status)
	}
	if retrieved.Result != "done" {
		t.Errorf("result = %s, want done", retrieved.Result)
	}
}

func TestTaskQueue_Reject(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("test", LeadThavren, PriorityLow, DomainSystem, "", "", "test")

	err := q.Reject(task.ID, "not needed")
	if err != nil {
		t.Fatalf("reject failed: %v", err)
	}

	retrieved, _ := q.Get(task.ID)
	if retrieved.Status != TaskRejected {
		t.Errorf("status = %s, want rejected", retrieved.Status)
	}
}

func TestTaskQueue_Fail(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("test", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")

	err := q.Fail(task.ID, "timeout")
	if err != nil {
		t.Fatalf("fail failed: %v", err)
	}

	retrieved, _ := q.Get(task.ID)
	if retrieved.Status != TaskFailed {
		t.Errorf("status = %s, want failed", retrieved.Status)
	}
}

func TestTaskQueue_GetPending(t *testing.T) {
	q := NewTaskQueue()

	// Dispatch tasks for different leads
	q.Dispatch("t1", LeadThavren, PriorityCritical, DomainSystem, "", "", "test")
	q.Dispatch("t2", LeadThavren, PriorityLow, DomainSystem, "", "", "test")
	q.Dispatch("t3", LeadOVAV, PriorityHigh, DomainSystem, "", "", "test")

	pending := q.GetPending(LeadThavren)
	if len(pending) != 2 {
		t.Fatalf("want 2 pending for thavren, got %d", len(pending))
	}

	// Should be sorted by priority: CRITICAL first
	if pending[0].Priority != PriorityCritical {
		t.Errorf("first pending should be CRITICAL, got %s", pending[0].Priority)
	}
}

func TestTaskQueue_GetPendingAfterAccept(t *testing.T) {
	q := NewTaskQueue()
	task := q.Dispatch("t1", LeadThavren, PriorityCritical, DomainSystem, "", "", "test")
	q.Accept(task.ID)

	pending := q.GetPending(LeadThavren)
	if len(pending) != 0 {
		t.Errorf("want 0 pending after accept, got %d", len(pending))
	}
}

func TestTaskQueue_CountByStatus(t *testing.T) {
	q := NewTaskQueue()

	t1 := q.Dispatch("t1", LeadThavren, PriorityCritical, DomainSystem, "", "", "test")
	t2 := q.Dispatch("t2", LeadThavren, PriorityHigh, DomainSystem, "", "", "test")
	q.Accept(t1.ID)
	q.Complete(t2.ID, "done", "")

	counts := q.CountByStatus()

	if counts[TaskPending] != 0 {
		t.Errorf("pending = %d, want 0", counts[TaskPending])
	}
	if counts[TaskAccepted] != 1 {
		t.Errorf("accepted = %d, want 1", counts[TaskAccepted])
	}
	if counts[TaskCompleted] != 1 {
		t.Errorf("completed = %d, want 1", counts[TaskCompleted])
	}
}

func TestTaskQueue_GetAll(t *testing.T) {
	q := NewTaskQueue()

	q.Dispatch("t1", LeadThavren, PriorityCritical, DomainSystem, "", "", "test")
	q.Dispatch("t2", LeadOVAV, PriorityHigh, DomainSystem, "", "", "test")

	all := q.GetAll()
	if len(all) != 2 {
		t.Fatalf("want 2 tasks, got %d", len(all))
	}
}

func TestTaskQueue_Get_NotFound(t *testing.T) {
	q := NewTaskQueue()

	_, ok := q.Get("nonexistent")
	if ok {
		t.Error("should not find nonexistent task")
	}
}

func TestTaskQueue_ConcurrentDispatch(t *testing.T) {
	q := NewTaskQueue()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			q.Dispatch("concurrent", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if q.Count() != 10 {
		t.Errorf("count = %d, want 10", q.Count())
	}
}

func TestTask_IDsAreUnique(t *testing.T) {
	q := NewTaskQueue()
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		task := q.Dispatch("test", LeadThavren, PriorityMedium, DomainSystem, "", "", "test")
		if ids[task.ID] {
			t.Errorf("duplicate ID: %s", task.ID)
		}
		ids[task.ID] = true
	}
}

func TestTaskQueue_GetByStatus(t *testing.T) {
	q := NewTaskQueue()

	// Dispatch tasks with different statuses
	t1 := q.Dispatch("task1", LeadThavren, PriorityHigh, DomainSystem, "", "", "test")
	t2 := q.Dispatch("task2", LeadThavren, PriorityHigh, DomainSystem, "", "", "test")
	t3 := q.Dispatch("task3", LeadThavren, PriorityHigh, DomainSystem, "", "", "test")
	t4 := q.Dispatch("task4", LeadThavren, PriorityHigh, DomainSystem, "", "", "test")

	// Accept t1, complete t2, reject t3 — t4 stays pending
	q.Accept(t1.ID)
	q.Complete(t2.ID, "done", "test evidence")
	q.Reject(t3.ID, "not needed")

	pending := q.GetByStatus(TaskPending)
	if len(pending) != 1 {
		t.Errorf("pending count = %d, want 1 (only t4 is pending)", len(pending))
	}
	for _, task := range pending {
		if task.Status != TaskPending {
			t.Errorf("status = %s, want TaskPending", task.Status)
		}
	}

	accepted := q.GetByStatus(TaskAccepted)
	if len(accepted) != 1 {
		t.Errorf("accepted count = %d, want 1", len(accepted))
	}
	if accepted[0].ID != t1.ID {
		t.Errorf("accepted[0].id = %s, want %s", accepted[0].ID, t1.ID)
	}

	completed := q.GetByStatus(TaskCompleted)
	if len(completed) != 1 {
		t.Errorf("completed count = %d, want 1", len(completed))
	}
	if completed[0].ID != t2.ID {
		t.Errorf("completed[0].id = %s, want %s", completed[0].ID, t2.ID)
	}

	rejected := q.GetByStatus(TaskRejected)
	if len(rejected) != 1 {
		t.Errorf("rejected count = %d, want 1", len(rejected))
	}
	if rejected[0].ID != t3.ID {
		t.Errorf("rejected[0].id = %s, want %s", rejected[0].ID, t3.ID)
	}

	// t4 is still pending
	pendingIDs := make(map[string]bool)
	for _, task := range pending {
		pendingIDs[task.ID] = true
	}
	if !pendingIDs[t4.ID] {
		t.Errorf("t4 should be in pending list")
	}
}
