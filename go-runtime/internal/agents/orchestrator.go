// Package agents provides autonomous agent orchestration with AI intelligence.
package agents

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// AutonomousOrchestrator manages intelligent agent coordination and task delegation.
type AutonomousOrchestrator struct {
	mu              sync.RWMutex
	agents          map[string]*AgentProfile
	taskQueue       []Task
	activeTasks     map[string]*ActiveTask
	completedTasks  []Task
	performanceLog  []PerformanceMetric
	learningEnabled bool
}

// AgentProfile represents an AI agent's capabilities and state.
type AgentProfile struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Specialization string        `json:"specialization"`
	Capabilities   []string      `json:"capabilities"`
	CurrentLoad    float64       `json:"current_load"` // 0.0 to 1.0
	SuccessRate    float64       `json:"success_rate"`
	TasksCompleted int           `json:"tasks_completed"`
	AvgDuration    time.Duration `json:"avg_duration"`
	LastActive     time.Time     `json:"last_active"`
	Status         string        `json:"status"` // "idle", "busy", "offline"
}

// Task represents a work item for agents.
type Task struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Priority       float64           `json:"priority"`   // 0.0 to 1.0
	Complexity     float64           `json:"complexity"` // 0.0 to 1.0
	RequiredSkills []string          `json:"required_skills"`
	AssignedTo     string            `json:"assigned_to,omitempty"`
	Status         string            `json:"status"` // "pending", "in_progress", "completed", "failed"
	CreatedAt      time.Time         `json:"created_at"`
	StartedAt      *time.Time        `json:"started_at,omitempty"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ActiveTask tracks a currently running task.
type ActiveTask struct {
	TaskID       string    `json:"task_id"`
	AgentID      string    `json:"agent_id"`
	StartTime    time.Time `json:"start_time"`
	EstimatedEnd time.Time `json:"estimated_end"`
	Progress     float64   `json:"progress"` // 0.0 to 1.0
}

// PerformanceMetric records agent performance data.
type PerformanceMetric struct {
	Timestamp time.Time     `json:"timestamp"`
	AgentID   string        `json:"agent_id"`
	TaskID    string        `json:"task_id"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Quality   float64       `json:"quality"` // 0.0 to 1.0
}

// NewAutonomousOrchestrator creates a new orchestrator.
func NewAutonomousOrchestrator() *AutonomousOrchestrator {
	return &AutonomousOrchestrator{
		agents:          make(map[string]*AgentProfile),
		taskQueue:       make([]Task, 0),
		activeTasks:     make(map[string]*ActiveTask),
		completedTasks:  make([]Task, 0),
		performanceLog:  make([]PerformanceMetric, 0),
		learningEnabled: true,
	}
}

// RegisterAgent adds an agent to the orchestration pool.
func (ao *AutonomousOrchestrator) RegisterAgent(agent *AgentProfile) {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	agent.Status = "idle"
	agent.LastActive = time.Now()
	ao.agents[agent.ID] = agent
}

// SubmitTask adds a task to the queue for autonomous assignment.
func (ao *AutonomousOrchestrator) SubmitTask(task Task) string {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	task.ID = fmt.Sprintf("task-%d", len(ao.taskQueue)+len(ao.completedTasks)+1)
	task.Status = "pending"
	task.CreatedAt = time.Now()

	ao.taskQueue = append(ao.taskQueue, task)

	// Trigger immediate assignment if learning is enabled
	if ao.learningEnabled {
		go ao.autoAssign()
	}

	return task.ID
}

// autoAssign intelligently assigns pending tasks to best-fit agents.
func (ao *AutonomousOrchestrator) autoAssign() {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	if len(ao.taskQueue) == 0 {
		return
	}

	// Sort tasks by priority
	sort.Slice(ao.taskQueue, func(i, j int) bool {
		return ao.taskQueue[i].Priority > ao.taskQueue[j].Priority
	})

	// Assign highest priority task
	task := ao.taskQueue[0]
	bestAgent := ao.findBestAgent(task)

	if bestAgent != nil {
		ao.assignTask(&task, bestAgent)
		ao.taskQueue = ao.taskQueue[1:]
	}
}

// findBestAgent uses AI-like scoring to find the optimal agent for a task.
func (ao *AutonomousOrchestrator) findBestAgent(task Task) *AgentProfile {
	var bestAgent *AgentProfile
	bestScore := -1.0

	for _, agent := range ao.agents {
		if agent.Status != "idle" || agent.CurrentLoad > 0.8 {
			continue
		}

		score := ao.calculateAgentTaskFit(agent, task)
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	return bestAgent
}

func (ao *AutonomousOrchestrator) calculateAgentTaskFit(agent *AgentProfile, task Task) float64 {
	score := 0.0

	// Skill match (40% weight)
	skillMatch := ao.calculateSkillMatch(agent.Capabilities, task.RequiredSkills)
	score += skillMatch * 0.4

	// Success rate (30% weight)
	score += agent.SuccessRate * 0.3

	// Load balancing (20% weight) - prefer less loaded agents
	loadScore := 1.0 - agent.CurrentLoad
	score += loadScore * 0.2

	// Specialization bonus (10% weight)
	if agent.Specialization != "" && contains(task.RequiredSkills, agent.Specialization) {
		score += 0.1
	}

	return score
}

func (ao *AutonomousOrchestrator) calculateSkillMatch(agentSkills, requiredSkills []string) float64 {
	if len(requiredSkills) == 0 {
		return 1.0
	}

	matchCount := 0
	for _, req := range requiredSkills {
		if contains(agentSkills, req) {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(requiredSkills))
}

func (ao *AutonomousOrchestrator) assignTask(task *Task, agent *AgentProfile) {
	now := time.Now()
	task.Status = "in_progress"
	task.AssignedTo = agent.ID
	task.StartedAt = &now

	agent.Status = "busy"
	agent.LastActive = now
	agent.CurrentLoad = math.Min(1.0, agent.CurrentLoad+task.Complexity*0.3)

	estimatedDuration := time.Duration(task.Complexity * float64(time.Hour))
	ao.activeTasks[task.ID] = &ActiveTask{
		TaskID:       task.ID,
		AgentID:      agent.ID,
		StartTime:    now,
		EstimatedEnd: now.Add(estimatedDuration),
		Progress:     0.0,
	}
}

// CompleteTask marks a task as completed and updates agent metrics.
func (ao *AutonomousOrchestrator) CompleteTask(taskID string, success bool, quality float64) error {
	ao.mu.Lock()
	defer ao.mu.Unlock()

	activeTask, exists := ao.activeTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found in active tasks", taskID)
	}

	now := time.Now()
	duration := now.Sub(activeTask.StartTime)

	// Find and update the task
	var taskIndex int
	var found bool
	for i, t := range ao.taskQueue {
		if t.ID == taskID {
			taskIndex = i
			found = true
			break
		}
	}

	if !found {
		// Check completed tasks
		for _, t := range ao.completedTasks {
			if t.ID == taskID {
				return fmt.Errorf("task %s already completed", taskID)
			}
		}
		return fmt.Errorf("task %s not found", taskID)
	}

	task := ao.taskQueue[taskIndex]
	task.Status = "completed"
	task.CompletedAt = &now

	// Update agent metrics
	if agent, exists := ao.agents[activeTask.AgentID]; exists {
		agent.TasksCompleted++
		agent.CurrentLoad = math.Max(0, agent.CurrentLoad-task.Complexity*0.3)
		if agent.CurrentLoad < 0.1 {
			agent.Status = "idle"
		}

		// Update success rate with exponential moving average
		oldWeight := 0.9
		newWeight := 0.1
		successValue := 0.0
		if success {
			successValue = 1.0
		}
		agent.SuccessRate = oldWeight*agent.SuccessRate + newWeight*successValue

		// Update average duration
		totalDuration := time.Duration(float64(agent.AvgDuration)*float64(agent.TasksCompleted-1) + float64(duration))
		agent.AvgDuration = totalDuration / time.Duration(agent.TasksCompleted)
	}

	// Log performance metric
	metric := PerformanceMetric{
		Timestamp: now,
		AgentID:   activeTask.AgentID,
		TaskID:    taskID,
		Duration:  duration,
		Success:   success,
		Quality:   quality,
	}
	ao.performanceLog = append(ao.performanceLog, metric)

	// Move to completed
	ao.completedTasks = append(ao.completedTasks, task)
	ao.taskQueue = append(ao.taskQueue[:taskIndex], ao.taskQueue[taskIndex+1:]...)
	delete(ao.activeTasks, taskID)

	// Try to assign next task
	if ao.learningEnabled && len(ao.taskQueue) > 0 {
		go ao.autoAssign()
	}

	return nil
}

// GetStatus returns the current orchestration status.
func (ao *AutonomousOrchestrator) GetStatus() OrchestrationStatus {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	status := OrchestrationStatus{
		TotalAgents:    len(ao.agents),
		IdleAgents:     0,
		BusyAgents:     0,
		PendingTasks:   len(ao.taskQueue),
		ActiveTasks:    len(ao.activeTasks),
		CompletedTasks: len(ao.completedTasks),
		SystemHealth:   ao.calculateSystemHealth(),
	}

	for _, agent := range ao.agents {
		if agent.Status == "idle" {
			status.IdleAgents++
		} else if agent.Status == "busy" {
			status.BusyAgents++
		}
	}

	return status
}

// OrchestrationStatus provides a snapshot of the system.
type OrchestrationStatus struct {
	TotalAgents    int     `json:"total_agents"`
	IdleAgents     int     `json:"idle_agents"`
	BusyAgents     int     `json:"busy_agents"`
	PendingTasks   int     `json:"pending_tasks"`
	ActiveTasks    int     `json:"active_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	SystemHealth   float64 `json:"system_health"` // 0.0 to 1.0
}

func (ao *AutonomousOrchestrator) calculateSystemHealth() float64 {
	if len(ao.agents) == 0 {
		return 0
	}

	// Health based on agent availability and success rates
	totalSuccessRate := 0.0
	count := 0
	for _, agent := range ao.agents {
		totalSuccessRate += agent.SuccessRate
		count++
	}

	avgSuccessRate := totalSuccessRate / float64(count)
	availability := float64(ao.GetStatus().IdleAgents) / float64(len(ao.agents))

	return (avgSuccessRate*0.7 + availability*0.3)
}

// GetRecommendations provides AI-generated optimization suggestions.
func (ao *AutonomousOrchestrator) GetRecommendations() []string {
	ao.mu.RLock()
	defer ao.mu.RUnlock()

	var recommendations []string
	status := ao.GetStatus()

	// Check for bottlenecks
	if status.PendingTasks > status.IdleAgents*2 && status.IdleAgents > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("High task backlog (%d pending). Consider adding more agents.", status.PendingTasks))
	}

	// Check for overloaded agents
	for _, agent := range ao.agents {
		if agent.CurrentLoad > 0.9 {
			recommendations = append(recommendations,
				fmt.Sprintf("Agent %s is heavily loaded (%.0f%%). Redistribute tasks.",
					agent.Name, agent.CurrentLoad*100))
		}
	}

	// Check for low success rates
	for _, agent := range ao.agents {
		if agent.SuccessRate < 0.6 && agent.TasksCompleted > 5 {
			recommendations = append(recommendations,
				fmt.Sprintf("Agent %s has low success rate (%.0f%%). Review training or reassign.",
					agent.Name, agent.SuccessRate*100))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System operating optimally. No changes needed.")
	}

	return recommendations
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
