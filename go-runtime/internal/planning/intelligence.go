// Package planning provides intelligent real-time project planning.
package planning

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PlanIntelligence provides AI-powered planning optimization.
type PlanIntelligence struct {
	mu            sync.RWMutex
	tasks         []Task
	sprints       []Sprint
	metrics       PlanMetrics
	predictions   []Prediction
	historicalData []HistoricalRecord
}

// Task represents a work item in the plan.
type Task struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	Status         string    `json:"status"` // "todo", "in_progress", "review", "done"
	Priority       float64   `json:"priority"` // 0.0 to 1.0
	Complexity     float64   `json:"complexity"` // story points 1-13
	EstimatedHours float64   `json:"estimated_hours"`
	ActualHours    float64   `json:"actual_hours,omitempty"`
	Assignee       string    `json:"assignee,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	Dependencies   []string  `json:"dependencies,omitempty"`
	Blockers       []string  `json:"blockers,omitempty"`
}

// Sprint represents a time-boxed iteration.
type Sprint struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Goal        string    `json:"goal"`
	TaskIDs     []string  `json:"task_ids"`
	Velocity    float64   `json:"velocity"` // story points completed
	Capacity    float64   `json:"capacity"` // total story points planned
	Status      string    `json:"status"` // "planned", "active", "completed"
}

// PlanMetrics tracks planning performance.
type PlanMetrics struct {
	TotalTasks       int     `json:"total_tasks"`
	CompletedTasks   int     `json:"completed_tasks"`
	AvgCycleTime     float64 `json:"avg_cycle_time_hours"`
	Velocity         float64 `json:"velocity"` // story points per sprint
	BurndownRate     float64 `json:"burndown_rate"` // tasks per day
	Efficiency       float64 `json:"efficiency"` // 0.0 to 1.0
	QualityScore     float64 `json:"quality_score"` // 0.0 to 1.0
}

// Prediction represents an AI-generated planning prediction.
type Prediction struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	Timeframe   string  `json:"timeframe"`
	Impact      string  `json:"impact"`
	Action      string  `json:"action"`
}

// HistoricalRecord stores past sprint data for learning.
type HistoricalRecord struct {
	SprintID     string    `json:"sprint_id"`
	PlannedPoints float64  `json:"planned_points"`
	CompletedPoints float64 `json:"completed_points"`
	Duration     time.Duration `json:"duration"`
	TeamSize     int       `json:"team_size"`
	QualityScore float64   `json:"quality_score"`
}

// NewPlanIntelligence creates a new intelligent planning system.
func NewPlanIntelligence() *PlanIntelligence {
	return &PlanIntelligence{
		tasks:          make([]Task, 0),
		sprints:        make([]Sprint, 0),
		historicalData: make([]HistoricalRecord, 0),
	}
}

// AddTask adds a task to the plan with intelligent prioritization.
func (pi *PlanIntelligence) AddTask(task Task) string {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	task.ID = fmt.Sprintf("task-%d", len(pi.tasks)+1)
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	
	// Auto-calculate priority based on various factors
	task.Priority = pi.calculateSmartPriority(task)
	
	pi.tasks = append(pi.tasks, task)
	return task.ID
}

func (pi *PlanIntelligence) calculateSmartPriority(task Task) float64 {
	basePriority := 0.5
	
	// Boost for urgency (due date proximity)
	if task.DueDate != nil {
		daysUntilDue := task.DueDate.Sub(time.Now()).Hours() / 24.0
		if daysUntilDue < 3 {
			basePriority += 0.3
		} else if daysUntilDue < 7 {
			basePriority += 0.15
		}
	}
	
	// Boost for complexity (harder tasks should start earlier)
	if task.Complexity > 8 {
		basePriority += 0.1
	}
	
	// Boost for dependencies (unblock others first)
	if len(task.Dependencies) == 0 && pi.hasDependents(task.ID) {
		basePriority += 0.15
	}
	
	// Reduce if blocked
	if len(task.Blockers) > 0 {
		basePriority -= 0.2
	}
	
	return math.Min(1.0, math.Max(0.0, basePriority))
}

func (pi *PlanIntelligence) hasDependents(taskID string) bool {
	for _, t := range pi.tasks {
		for _, dep := range t.Dependencies {
			if dep == taskID {
				return true
			}
		}
	}
	return false
}

// UpdateTaskStatus updates a task's status and tracks metrics.
func (pi *PlanIntelligence) UpdateTaskStatus(taskID, newStatus string, actualHours float64) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	for i, task := range pi.tasks {
		if task.ID == taskID {
			oldStatus := task.Status
			pi.tasks[i].Status = newStatus
			
			now := time.Now()
			if newStatus == "in_progress" && oldStatus == "todo" {
				pi.tasks[i].StartedAt = &now
			}
			
			if newStatus == "done" {
				pi.tasks[i].CompletedAt = &now
				pi.tasks[i].ActualHours = actualHours
				
				// Record historical data for learning
				pi.recordCompletion(task, actualHours)
			}
			
			// Recalculate metrics
			pi.recalculateMetrics()
			return nil
		}
	}
	
	return fmt.Errorf("task %s not found", taskID)
}

func (pi *PlanIntelligence) recordCompletion(task Task, actualHours float64) {
	// This would integrate with sprint tracking
	// For now, just store basic completion data
}

func (pi *PlanIntelligence) recalculateMetrics() {
	totalTasks := len(pi.tasks)
	completedTasks := 0
	totalCycleTime := 0.0
	cycleTimeCount := 0
	
	for _, task := range pi.tasks {
		if task.Status == "done" {
			completedTasks++
			
			if task.StartedAt != nil && task.CompletedAt != nil {
				cycleTime := task.CompletedAt.Sub(*task.StartedAt).Hours()
				totalCycleTime += cycleTime
				cycleTimeCount++
			}
		}
	}
	
	pi.metrics.TotalTasks = totalTasks
	pi.metrics.CompletedTasks = completedTasks
	
	if cycleTimeCount > 0 {
		pi.metrics.AvgCycleTime = totalCycleTime / float64(cycleTimeCount)
	}
	
	// Calculate efficiency
	if totalTasks > 0 {
		pi.metrics.Efficiency = float64(completedTasks) / float64(totalTasks)
	}
}

// GeneratePredictions creates AI-powered planning predictions.
func (pi *PlanIntelligence) GeneratePredictions() []Prediction {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	var predictions []Prediction
	
	// Predict sprint completion
	activeSprint := pi.getActiveSprint()
	if activeSprint != nil {
		completionPrediction := pi.predictSprintCompletion(activeSprint)
		if completionPrediction != nil {
			predictions = append(predictions, *completionPrediction)
		}
	}
	
	// Predict bottlenecks
	bottleneckPred := pi.predictBottlenecks()
	if bottleneckPred != nil {
		predictions = append(predictions, *bottleneckPred)
	}
	
	// Predict resource needs
	resourcePred := pi.predictResourceNeeds()
	if resourcePred != nil {
		predictions = append(predictions, *resourcePred)
	}
	
	// Predict quality risks
	qualityPred := pi.predictQualityRisks()
	if qualityPred != nil {
		predictions = append(predictions, *qualityPred)
	}
	
	return predictions
}

func (pi *PlanIntelligence) getActiveSprint() *Sprint {
	for i, sprint := range pi.sprints {
		if sprint.Status == "active" {
			return &pi.sprints[i]
		}
	}
	return nil
}

func (pi *PlanIntelligence) predictSprintCompletion(sprint *Sprint) *Prediction {
	now := time.Now()
	timeRemaining := sprint.EndDate.Sub(now).Hours() / 24.0
	
	// Calculate current velocity
	var completedPoints float64
	for _, taskID := range sprint.TaskIDs {
		for _, task := range pi.tasks {
			if task.ID == taskID && task.Status == "done" {
				completedPoints += task.Complexity
			}
		}
	}
	
	daysElapsed := now.Sub(sprint.StartDate).Hours() / 24.0
	if daysElapsed > 0 {
		dailyVelocity := completedPoints / daysElapsed
		remainingPoints := sprint.Capacity - completedPoints
		
		if dailyVelocity > 0 {
			daysNeeded := remainingPoints / dailyVelocity
			
			if daysNeeded > timeRemaining {
				return &Prediction{
					Type:        "sprint_risk",
					Description: fmt.Sprintf("Sprint may not complete on time (%.1f days needed, %.1f remaining)", 
						daysNeeded, timeRemaining),
					Confidence:  math.Min(1.0, daysNeeded/timeRemaining*0.7),
					Timeframe:   fmt.Sprintf("%.0f days", timeRemaining),
					Impact:      fmt.Sprintf("%.0f story points at risk", remainingPoints),
					Action:      "Consider descoping lower priority items or adding resources",
				}
			}
		}
	}
	
	return nil
}

func (pi *PlanIntelligence) predictBottlenecks() *Prediction {
	// Find tasks with many dependents
	type taskDependent struct {
		id         string
		title      string
		dependents int
	}
	
	var tasksWithDeps []taskDependent
	for _, task := range pi.tasks {
		if task.Status != "done" {
			count := pi.countDependents(task.ID)
			if count > 0 {
				tasksWithDeps = append(tasksWithDeps, taskDependent{
					id:         task.ID,
					title:      task.Title,
					dependents: count,
				})
			}
		}
	}
	
	if len(tasksWithDeps) > 0 {
		sort.Slice(tasksWithDeps, func(i, j int) bool {
			return tasksWithDeps[i].dependents > tasksWithDeps[j].dependents
		})
		
		top := tasksWithDeps[0]
		if top.dependents >= 3 {
			return &Prediction{
				Type:        "bottleneck",
				Description: fmt.Sprintf("Task '%s' is blocking %d other tasks", top.title, top.dependents),
				Confidence:  0.85,
				Timeframe:   "current_sprint",
				Impact:      fmt.Sprintf("%d tasks blocked", top.dependents),
				Action:      fmt.Sprintf("Prioritize completion of task %s", top.id),
			}
		}
	}
	
	return nil
}

func (pi *PlanIntelligence) countDependents(taskID string) int {
	count := 0
	for _, task := range pi.tasks {
		for _, dep := range task.Dependencies {
			if dep == taskID {
				count++
			}
		}
	}
	return count
}

func (pi *PlanIntelligence) predictResourceNeeds() *Prediction {
	// Analyze workload distribution
	assigneeWorkload := make(map[string]float64)
	
	for _, task := range pi.tasks {
		if task.Status != "done" && task.Assignee != "" {
			assigneeWorkload[task.Assignee] += task.Complexity
		}
	}
	
	if len(assigneeWorkload) > 0 {
		maxWorkload := 0.0
		maxAssignee := ""
		totalWorkload := 0.0
		
		for assignee, workload := range assigneeWorkload {
			totalWorkload += workload
			if workload > maxWorkload {
				maxWorkload = workload
				maxAssignee = assignee
			}
		}
		
		avgWorkload := totalWorkload / float64(len(assigneeWorkload))
		
		if maxWorkload > avgWorkload*1.5 {
			return &Prediction{
				Type:        "resource_imbalance",
				Description: fmt.Sprintf("Workload imbalance detected: %s has %.0f points (avg: %.0f)", 
					maxAssignee, maxWorkload, avgWorkload),
				Confidence:  0.75,
				Timeframe:   "next_week",
				Impact:      "Risk of burnout and delays",
				Action:      "Redistribute tasks or add team members",
			}
		}
	}
	
	return nil
}

func (pi *PlanIntelligence) predictQualityRisks() *Prediction {
	// Check for rushed tasks (low estimated vs actual hours ratio)
	rushedTasks := 0
	totalCompleted := 0
	
	for _, task := range pi.tasks {
		if task.Status == "done" && task.EstimatedHours > 0 && task.ActualHours > 0 {
			totalCompleted++
			ratio := task.ActualHours / task.EstimatedHours
			if ratio > 2.0 { // Took more than 2x estimated
				rushedTasks++
			}
		}
	}
	
	if totalCompleted > 0 && float64(rushedTasks)/float64(totalCompleted) > 0.3 {
		return &Prediction{
			Type:        "quality_risk",
			Description: fmt.Sprintf("%d of %d completed tasks took >2x estimated time", 
				rushedTasks, totalCompleted),
			Confidence:  0.8,
			Timeframe:   "ongoing",
			Impact:      "Technical debt accumulation",
			Action:      "Review estimation process and add buffer time",
		}
	}
	
	return nil
}

// OptimizePlan suggests improvements to the current plan.
func (pi *PlanIntelligence) OptimizePlan() []OptimizationSuggestion {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	var suggestions []OptimizationSuggestion
	
	// Suggest reordering based on dependencies
	orderSuggestions := pi.suggestTaskOrdering()
	suggestions = append(suggestions, orderSuggestions...)
	
	// Suggest priority adjustments
	prioritySuggestions := pi.suggestPriorityAdjustments()
	suggestions = append(suggestions, prioritySuggestions...)
	
	// Suggest sprint improvements
	sprintSuggestions := pi.suggestSprintImprovements()
	suggestions = append(suggestions, sprintSuggestions...)
	
	return suggestions
}

// OptimizationSuggestion represents a recommended improvement.
type OptimizationSuggestion struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Effort      string  `json:"effort"` // "low", "medium", "high"
	Priority    float64 `json:"priority"`
}

func (pi *PlanIntelligence) suggestTaskOrdering() []OptimizationSuggestion {
	var suggestions []OptimizationSuggestion
	
	// Find tasks that should be done earlier due to dependencies
	for _, task := range pi.tasks {
		if task.Status == "todo" && len(task.Dependencies) == 0 {
			dependents := pi.countDependents(task.ID)
			if dependents >= 2 {
				suggestions = append(suggestions, OptimizationSuggestion{
					Type:        "reorder",
					Description: fmt.Sprintf("Move '%s' earlier - blocks %d tasks", task.Title, dependents),
					Impact:      fmt.Sprintf("Unblock %d dependent tasks", dependents),
					Effort:      "low",
					Priority:    0.8,
				})
			}
		}
	}
	
	return suggestions
}

func (pi *PlanIntelligence) suggestPriorityAdjustments() []OptimizationSuggestion {
	var suggestions []OptimizationSuggestion
	
	// Find high-complexity tasks with low priority
	for _, task := range pi.tasks {
		if task.Status == "todo" && task.Complexity > 8 && task.Priority < 0.5 {
			suggestions = append(suggestions, OptimizationSuggestion{
				Type:        "reprioritize",
				Description: fmt.Sprintf("Increase priority for complex task '%s'", task.Title),
				Impact:      "Reduce risk of late delivery on complex work",
				Effort:      "low",
				Priority:    0.7,
			})
		}
	}
	
	return suggestions
}

func (pi *PlanIntelligence) suggestSprintImprovements() []OptimizationSuggestion {
	var suggestions []OptimizationSuggestion
	
	for _, sprint := range pi.sprints {
		if sprint.Status == "planned" || sprint.Status == "active" {
			utilization := sprint.Velocity / sprint.Capacity
			
			if utilization < 0.6 {
				suggestions = append(suggestions, OptimizationSuggestion{
					Type:        "sprint_capacity",
					Description: fmt.Sprintf("Sprint '%s' underutilized (%.0f%%)", sprint.Name, utilization*100),
					Impact:      "Add more tasks to improve throughput",
					Effort:      "medium",
					Priority:    0.6,
				})
			} else if utilization > 0.95 {
				suggestions = append(suggestions, OptimizationSuggestion{
					Type:        "sprint_capacity",
					Description: fmt.Sprintf("Sprint '%s' overcommitted (%.0f%%)", sprint.Name, utilization*100),
					Impact:      "Risk of incomplete sprint goals",
					Effort:      "medium",
					Priority:    0.85,
				})
			}
		}
	}
	
	return suggestions
}

// GetDashboard returns a comprehensive planning dashboard.
func (pi *PlanIntelligence) GetDashboard() Dashboard {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	dashboard := Dashboard{
		Metrics:     pi.metrics,
		Predictions: pi.GeneratePredictions(),
		Suggestions: pi.OptimizePlan(),
	}
	
	// Count tasks by status
	statusCounts := make(map[string]int)
	for _, task := range pi.tasks {
		statusCounts[task.Status]++
	}
	dashboard.TasksByStatus = statusCounts
	
	return dashboard
}

// Dashboard provides a comprehensive view of the plan.
type Dashboard struct {
	Metrics       PlanMetrics           `json:"metrics"`
	TasksByStatus map[string]int        `json:"tasks_by_status"`
	Predictions   []Prediction          `json:"predictions"`
	Suggestions   []OptimizationSuggestion `json:"suggestions"`
}
