package governor

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ── Task Types ───────────────────────────────────────────────────────────

// TaskStatus represents the lifecycle state of a delegated task.
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskAccepted   TaskStatus = "accepted"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskRejected   TaskStatus = "rejected"
	TaskFailed     TaskStatus = "failed"
)

// Task represents a single delegated task from OVAV to a lead.
type Task struct {
	ID              string     `json:"id"`
	Description     string     `json:"description"`
	Lead            Lead       `json:"lead"`
	Priority        Priority   `json:"priority"`
	Domain          Domain     `json:"domain"`
	Target          string     `json:"target"`
	ExpectedOutcome string     `json:"expected_outcome"`
	Source          string     `json:"source"`
	Status          TaskStatus `json:"status"`
	Result          string     `json:"result,omitempty"`
	Evidence        string     `json:"evidence,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TaskQueue manages the delegation task queue.
// Replaces tools/governor/delegation_protocol.py (391 LOC Python).
type TaskQueue struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	order []string // insertion order
}

// NewTaskQueue creates a new task queue.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks: make(map[string]*Task),
	}
}

// ── Queue Operations ─────────────────────────────────────────────────────

// Dispatch creates a new task and adds it to the queue.
func (q *TaskQueue) Dispatch(description string, lead Lead, priority Priority, domain Domain, target, expectedOutcome, source string) *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now().UTC()
	task := &Task{
		ID:              generateTaskID(),
		Description:     description,
		Lead:            lead,
		Priority:        priority,
		Domain:          domain,
		Target:          target,
		ExpectedOutcome: expectedOutcome,
		Source:          source,
		Status:          TaskPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	q.tasks[task.ID] = task
	q.order = append(q.order, task.ID)
	return task
}

// DispatchFromDecision creates tasks from decision engine output.
func (q *TaskQueue) DispatchFromDecisions(decisions []Decision) []*Task {
	var tasks []*Task
	for _, d := range decisions {
		task := q.Dispatch(d.Reason, d.Lead, d.Priority, d.Domain, d.Target, d.ExpectedOutcome, "decision_engine")
		tasks = append(tasks, task)
	}
	return tasks
}

// Accept marks a task as accepted by the lead.
func (q *TaskQueue) Accept(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.Status != TaskPending {
		return fmt.Errorf("task %s is %s, not pending", taskID, task.Status)
	}
	task.Status = TaskAccepted
	task.UpdatedAt = time.Now().UTC()
	return nil
}

// Complete marks a task as completed with results.
func (q *TaskQueue) Complete(taskID, result, evidence string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.Status == TaskCompleted {
		return fmt.Errorf("task %s already completed", taskID)
	}
	task.Status = TaskCompleted
	task.Result = result
	task.Evidence = evidence
	task.UpdatedAt = time.Now().UTC()
	return nil
}

// Reject marks a task as rejected by the lead.
func (q *TaskQueue) Reject(taskID, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = TaskRejected
	task.Result = reason
	task.UpdatedAt = time.Now().UTC()
	return nil
}

// Fail marks a task as failed.
func (q *TaskQueue) Fail(taskID, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = TaskFailed
	task.Result = reason
	task.UpdatedAt = time.Now().UTC()
	return nil
}

// ── Query Operations ─────────────────────────────────────────────────────

// GetPending returns all pending tasks for a lead, sorted by priority.
func (q *TaskQueue) GetPending(lead Lead) []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var tasks []*Task
	for _, task := range q.tasks {
		if task.Lead == lead && task.Status == TaskPending {
			tasks = append(tasks, task)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority]
	})
	return tasks
}

// GetAll returns all tasks, sorted by creation time (newest first).
func (q *TaskQueue) GetAll() []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var tasks []*Task
	for _, id := range q.order {
		tasks = append(tasks, q.tasks[id])
	}
	return tasks
}

// GetByStatus returns all tasks with a given status.
func (q *TaskQueue) GetByStatus(status TaskStatus) []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var tasks []*Task
	for _, task := range q.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// Get returns a task by ID.
func (q *TaskQueue) Get(taskID string) (*Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	task, ok := q.tasks[taskID]
	return task, ok
}

// Count returns the total number of tasks.
func (q *TaskQueue) Count() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.tasks)
}

// CountByStatus returns task counts by status.
func (q *TaskQueue) CountByStatus() map[TaskStatus]int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	counts := make(map[TaskStatus]int)
	for _, task := range q.tasks {
		counts[task.Status]++
	}
	return counts
}

// ── ID generation ────────────────────────────────────────────────────────

var taskCounter struct {
	sync.Mutex
	n int
}

func generateTaskID() string {
	taskCounter.Lock()
	defer taskCounter.Unlock()
	taskCounter.n++
	return fmt.Sprintf("OVAV-%s-%04d", time.Now().UTC().Format("20060102"), taskCounter.n)
}
