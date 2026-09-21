// Package healing provides autonomous self-healing and recovery mechanisms.
package healing

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SelfHealingEngine monitors system health and performs automatic repairs.
type SelfHealingEngine struct {
	mu                sync.RWMutex
	systems           map[string]*SystemHealth
	recoveryActions   []RecoveryAction
	healingHistory    []HealingEvent
	autoRepairEnabled bool
	checkInterval     time.Duration
	lastCheck         time.Time
}

// SystemHealth represents the health status of a subsystem.
type SystemHealth struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"` // "healthy", "degraded", "critical", "offline"
	Uptime       time.Duration `json:"uptime"`
	LastCheck    time.Time     `json:"last_check"`
	ErrorRate    float64       `json:"error_rate"` // 0.0 to 1.0
	ResponseTime time.Duration `json:"response_time"`
	MemoryUsage  float64       `json:"memory_usage"` // percentage
	CPUUsage     float64       `json:"cpu_usage"`    // percentage
	DiskUsage    float64       `json:"disk_usage"`   // percentage
	Issues       []Issue       `json:"issues"`
}

// Issue represents a detected problem.
type Issue struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"` // "info", "warning", "error", "critical"
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
	Category    string    `json:"category"`
	Context     string    `json:"context,omitempty"`
}

// RecoveryAction represents an automated repair action.
type RecoveryAction struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        string        `json:"type"` // "restart", "cleanup", "rebuild", "rollback", "scale"
	Target      string        `json:"target"`
	Status      string        `json:"status"` // "pending", "in_progress", "completed", "failed"
	ExecutedAt  *time.Time    `json:"executed_at,omitempty"`
	Result      string        `json:"result,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// HealingEvent records a healing operation.
type HealingEvent struct {
	Timestamp  time.Time     `json:"timestamp"`
	SystemName string        `json:"system_name"`
	IssueID    string        `json:"issue_id"`
	Action     string        `json:"action"`
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration"`
	Details    string        `json:"details"`
}

// NewSelfHealingEngine creates a new self-healing engine.
func NewSelfHealingEngine() *SelfHealingEngine {
	return &SelfHealingEngine{
		systems:           make(map[string]*SystemHealth),
		recoveryActions:   make([]RecoveryAction, 0),
		healingHistory:    make([]HealingEvent, 0),
		autoRepairEnabled: true,
		checkInterval:     5 * time.Minute,
		lastCheck:         time.Now(),
	}
}

// RegisterSystem adds a system to monitoring.
func (she *SelfHealingEngine) RegisterSystem(name string) {
	she.mu.Lock()
	defer she.mu.Unlock()

	she.systems[name] = &SystemHealth{
		Name:      name,
		Status:    "healthy",
		Uptime:    0,
		LastCheck: time.Now(),
		Issues:    make([]Issue, 0),
	}
}

// UpdateHealth updates the health metrics for a system.
func (she *SelfHealingEngine) UpdateHealth(name string, metrics HealthMetrics) {
	she.mu.Lock()
	defer she.mu.Unlock()

	sys, exists := she.systems[name]
	if !exists {
		return
	}

	sys.LastCheck = time.Now()
	sys.ErrorRate = metrics.ErrorRate
	sys.ResponseTime = metrics.ResponseTime
	sys.MemoryUsage = metrics.MemoryUsage
	sys.CPUUsage = metrics.CPUUsage
	sys.DiskUsage = metrics.DiskUsage

	// Detect issues based on thresholds
	she.detectIssues(sys)

	// Update status based on overall health
	she.updateSystemStatus(sys)
}

// HealthMetrics contains system health measurements.
type HealthMetrics struct {
	ErrorRate    float64
	ResponseTime time.Duration
	MemoryUsage  float64
	CPUUsage     float64
	DiskUsage    float64
}

func (she *SelfHealingEngine) detectIssues(sys *SystemHealth) {
	// Check error rate
	if sys.ErrorRate > 0.1 {
		severity := "warning"
		if sys.ErrorRate > 0.3 {
			severity = "error"
		}
		if sys.ErrorRate > 0.5 {
			severity = "critical"
		}

		sys.Issues = append(sys.Issues, Issue{
			ID:          fmt.Sprintf("issue-%d", len(sys.Issues)+1),
			Severity:    severity,
			Description: fmt.Sprintf("High error rate: %.1f%%", sys.ErrorRate*100),
			DetectedAt:  time.Now(),
			Category:    "performance",
		})
	}

	// Check memory usage
	if sys.MemoryUsage > 80 {
		severity := "warning"
		if sys.MemoryUsage > 90 {
			severity = "critical"
		}

		sys.Issues = append(sys.Issues, Issue{
			ID:          fmt.Sprintf("issue-%d", len(sys.Issues)+1),
			Severity:    severity,
			Description: fmt.Sprintf("High memory usage: %.0f%%", sys.MemoryUsage),
			DetectedAt:  time.Now(),
			Category:    "resources",
		})
	}

	// Check CPU usage
	if sys.CPUUsage > 85 {
		sys.Issues = append(sys.Issues, Issue{
			ID:          fmt.Sprintf("issue-%d", len(sys.Issues)+1),
			Severity:    "warning",
			Description: fmt.Sprintf("High CPU usage: %.0f%%", sys.CPUUsage),
			DetectedAt:  time.Now(),
			Category:    "resources",
		})
	}

	// Check disk usage
	if sys.DiskUsage > 85 {
		severity := "warning"
		if sys.DiskUsage > 95 {
			severity = "critical"
		}

		sys.Issues = append(sys.Issues, Issue{
			ID:          fmt.Sprintf("issue-%d", len(sys.Issues)+1),
			Severity:    severity,
			Description: fmt.Sprintf("High disk usage: %.0f%%", sys.DiskUsage),
			DetectedAt:  time.Now(),
			Category:    "storage",
		})
	}
}

func (she *SelfHealingEngine) updateSystemStatus(sys *SystemHealth) {
	criticalCount := 0
	errorCount := 0
	warningCount := 0

	for _, issue := range sys.Issues {
		switch issue.Severity {
		case "critical":
			criticalCount++
		case "error":
			errorCount++
		case "warning":
			warningCount++
		}
	}

	if criticalCount > 0 {
		sys.Status = "critical"
	} else if errorCount > 0 {
		sys.Status = "degraded"
	} else if warningCount > 0 {
		sys.Status = "degraded"
	} else {
		sys.Status = "healthy"
	}
}

// RunDiagnostics performs a full diagnostic cycle.
func (she *SelfHealingEngine) RunDiagnostics() []DiagnosisResult {
	she.mu.Lock()
	defer she.mu.Unlock()

	var results []DiagnosisResult

	for name, sys := range she.systems {
		result := she.diagnoseSystem(name, sys)
		results = append(results, result)

		// Auto-repair if enabled and issues found
		if she.autoRepairEnabled && len(sys.Issues) > 0 {
			she.attemptAutoRepair(name, sys)
		}
	}

	she.lastCheck = time.Now()
	return results
}

// DiagnosisResult contains diagnostic findings.
type DiagnosisResult struct {
	SystemName      string   `json:"system_name"`
	Status          string   `json:"status"`
	Issues          []Issue  `json:"issues"`
	Recommendations []string `json:"recommendations"`
	HealthScore     float64  `json:"health_score"` // 0.0 to 1.0
}

func (she *SelfHealingEngine) diagnoseSystem(name string, sys *SystemHealth) DiagnosisResult {
	result := DiagnosisResult{
		SystemName: name,
		Status:     sys.Status,
		Issues:     make([]Issue, len(sys.Issues)),
	}
	copy(result.Issues, sys.Issues)

	// Generate recommendations
	result.Recommendations = she.generateRecommendations(sys)

	// Calculate health score
	result.HealthScore = she.calculateHealthScore(sys)

	return result
}

func (she *SelfHealingEngine) generateRecommendations(sys *SystemHealth) []string {
	var recommendations []string

	for _, issue := range sys.Issues {
		switch issue.Category {
		case "performance":
			recommendations = append(recommendations,
				fmt.Sprintf("Investigate high error rate in %s - consider adding retry logic or circuit breakers", sys.Name))
		case "resources":
			if sys.MemoryUsage > 80 {
				recommendations = append(recommendations,
					fmt.Sprintf("Memory optimization needed for %s - review for leaks or increase allocation", sys.Name))
			}
			if sys.CPUUsage > 85 {
				recommendations = append(recommendations,
					fmt.Sprintf("CPU optimization needed for %s - consider horizontal scaling or query optimization", sys.Name))
			}
		case "storage":
			recommendations = append(recommendations,
				fmt.Sprintf("Disk cleanup needed for %s - archive old data or expand storage", sys.Name))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System operating normally")
	}

	return recommendations
}

func (she *SelfHealingEngine) calculateHealthScore(sys *SystemHealth) float64 {
	score := 1.0

	// Penalize for issues
	for _, issue := range sys.Issues {
		switch issue.Severity {
		case "critical":
			score -= 0.3
		case "error":
			score -= 0.2
		case "warning":
			score -= 0.1
		}
	}

	// Penalize for resource usage
	if sys.MemoryUsage > 90 {
		score -= 0.2
	} else if sys.MemoryUsage > 80 {
		score -= 0.1
	}

	if sys.CPUUsage > 90 {
		score -= 0.2
	} else if sys.CPUUsage > 80 {
		score -= 0.1
	}

	if sys.ErrorRate > 0.5 {
		score -= 0.3
	} else if sys.ErrorRate > 0.2 {
		score -= 0.15
	} else if sys.ErrorRate > 0.1 {
		score -= 0.05
	}

	return math.Max(0, score)
}

func (she *SelfHealingEngine) attemptAutoRepair(systemName string, sys *SystemHealth) {
	for _, issue := range sys.Issues {
		if issue.Severity == "critical" || issue.Severity == "error" {
			action := she.createRecoveryAction(systemName, issue)
			she.recoveryActions = append(she.recoveryActions, action)

			// Execute repair asynchronously
			go she.executeRepair(action, systemName, issue)
		}
	}
}

func (she *SelfHealingEngine) createRecoveryAction(systemName string, issue Issue) RecoveryAction {
	actionType := "investigate"

	switch issue.Category {
	case "resources":
		if strings.Contains(issue.Description, "memory") {
			actionType = "cleanup"
		} else if strings.Contains(issue.Description, "CPU") {
			actionType = "scale"
		}
	case "storage":
		actionType = "cleanup"
	case "performance":
		actionType = "restart"
	}

	return RecoveryAction{
		ID:          fmt.Sprintf("action-%d", len(she.recoveryActions)+1),
		Name:        fmt.Sprintf("Auto-repair: %s", issue.Description),
		Description: fmt.Sprintf("Automated response to %s issue", issue.Category),
		Type:        actionType,
		Target:      systemName,
		Status:      "pending",
	}
}

func (she *SelfHealingEngine) executeRepair(action RecoveryAction, systemName string, issue Issue) {
	startTime := time.Now()

	// Simulate repair actions (in real implementation, these would perform actual fixes)
	var success bool
	var result string

	switch action.Type {
	case "cleanup":
		success = true
		result = "Cleanup completed successfully - freed resources"
	case "restart":
		success = true
		result = "Service restarted with fresh state"
	case "scale":
		success = true
		result = "Additional resources allocated"
	case "rollback":
		success = true
		result = "Rolled back to previous stable version"
	default:
		success = false
		result = "Manual intervention required"
	}

	duration := time.Since(startTime)

	// Update action status
	she.mu.Lock()
	for i, a := range she.recoveryActions {
		if a.ID == action.ID {
			she.recoveryActions[i].Status = "completed"
			now := time.Now()
			she.recoveryActions[i].ExecutedAt = &now
			she.recoveryActions[i].Result = result
			she.recoveryActions[i].Duration = duration
			break
		}
	}

	// Log healing event
	event := HealingEvent{
		Timestamp:  startTime,
		SystemName: systemName,
		IssueID:    issue.ID,
		Action:     action.Name,
		Success:    success,
		Duration:   duration,
		Details:    result,
	}
	she.healingHistory = append(she.healingHistory, event)
	she.mu.Unlock()

	// Remove resolved issue if repair was successful
	if success {
		she.mu.Lock()
		if sys, exists := she.systems[systemName]; exists {
			var remaining []Issue
			for _, iss := range sys.Issues {
				if iss.ID != issue.ID {
					remaining = append(remaining, iss)
				}
			}
			sys.Issues = remaining
			she.updateSystemStatus(sys)
		}
		she.mu.Unlock()
	}
}

// GetHealingStats returns statistics about healing operations.
func (she *SelfHealingEngine) GetHealingStats() HealingStats {
	she.mu.RLock()
	defer she.mu.RUnlock()

	stats := HealingStats{
		TotalSystems:      len(she.systems),
		HealthySystems:    0,
		DegradedSystems:   0,
		CriticalSystems:   0,
		TotalRepairs:      len(she.healingHistory),
		SuccessfulRepairs: 0,
		AvgRepairTime:     0,
	}

	for _, sys := range she.systems {
		switch sys.Status {
		case "healthy":
			stats.HealthySystems++
		case "degraded":
			stats.DegradedSystems++
		case "critical":
			stats.CriticalSystems++
		}
	}

	totalDuration := time.Duration(0)
	for _, event := range she.healingHistory {
		if event.Success {
			stats.SuccessfulRepairs++
			totalDuration += event.Duration
		}
	}

	if stats.SuccessfulRepairs > 0 {
		stats.AvgRepairTime = totalDuration / time.Duration(stats.SuccessfulRepairs)
	}

	return stats
}

// HealingStats contains healing operation statistics.
type HealingStats struct {
	TotalSystems      int           `json:"total_systems"`
	HealthySystems    int           `json:"healthy_systems"`
	DegradedSystems   int           `json:"degraded_systems"`
	CriticalSystems   int           `json:"critical_systems"`
	TotalRepairs      int           `json:"total_repairs"`
	SuccessfulRepairs int           `json:"successful_repairs"`
	AvgRepairTime     time.Duration `json:"avg_repair_time"`
}

// CleanupTempFiles performs automated cleanup of temporary files.
func (she *SelfHealingEngine) CleanupTempFiles(pattern string, maxAge time.Duration) (int, error) {
	tempDir := os.TempDir()
	searchPattern := filepath.Join(tempDir, pattern)

	matches, err := filepath.Glob(searchPattern)
	if err != nil {
		return 0, fmt.Errorf("glob pattern: %w", err)
	}

	removed := 0
	cutoff := time.Now().Add(-maxAge)

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(match); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}
