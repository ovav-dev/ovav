// Package alerts implements OVAV's alert queue system.
// Alerts are emitted by monitors and stored in .ovav/runtime/alerts/
//
// Queue structure:
//
//	queue.jsonl      — pending alerts (awaiting processing)
//	auto_fixed.jsonl — alerts resolved automatically
//	acknowledged.jsonl — alerts resolved by human
//	archived.jsonl   — old alerts for audit trail
package alerts

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Alert represents a single system alert
type Alert struct {
	ID      string      `json:"id"`
	TS      time.Time   `json:"ts"`
	Level   AlertLevel  `json:"level"`  // CRIT | ERROR | WARN | INFO
	Source  string      `json:"source"` // Which monitor generated it
	Issue   string      `json:"issue"`  // Human-readable issue
	Files   []string    `json:"files,omitempty"`
	Runbook string      `json:"runbook,omitempty"` // Auto-fix runbook name
	Status  AlertStatus `json:"status"`            // pending | auto-fixed | acknowledged | archived
	// Resolution tracking
	ResolvedTS *time.Time `json:"resolved_ts,omitempty"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
	Resolution string     `json:"resolution,omitempty"`
}

// AlertLevel defines alert severity
type AlertLevel string

const (
	LevelCRIT  AlertLevel = "CRIT"
	LevelERROR AlertLevel = "ERROR"
	LevelWARN  AlertLevel = "WARN"
	LevelINFO  AlertLevel = "INFO"
)

// AlertStatus tracks resolution state
type AlertStatus string

const (
	StatusPending      AlertStatus = "pending"
	StatusAutoFixed    AlertStatus = "auto-fixed"
	StatusAcknowledged AlertStatus = "acknowledged"
	StatusArchived     AlertStatus = "archived"
)

// Queue manages the alert queue in .ovav/runtime/alerts/
type Queue struct {
	root string
	mu   sync.RWMutex
}

const (
	fileQueue        = "queue.jsonl"
	fileAutoFixed    = "auto_fixed.jsonl"
	fileAcknowledged = "acknowledged.jsonl"
	fileArchived     = "archived.jsonl"
)

// NewQueue creates a new alert queue
func NewQueue(root string) *Queue {
	q := &Queue{root: root}
	os.MkdirAll(filepath.Join(root, ".ovav", "runtime", "alerts"), 0755)
	return q
}

func (q *Queue) queuePath() string {
	return filepath.Join(q.root, ".ovav", "runtime", "alerts", fileQueue)
}

func (q *Queue) autoFixedPath() string {
	return filepath.Join(q.root, ".ovav", "runtime", "alerts", fileAutoFixed)
}

func (q *Queue) acknowledgedPath() string {
	return filepath.Join(q.root, ".ovav", "runtime", "alerts", fileAcknowledged)
}

func (q *Queue) archivedPath() string {
	return filepath.Join(q.root, ".ovav", "runtime", "alerts", fileArchived)
}

// NewAlert creates a new alert with auto-generated ID and timestamp
func NewAlert(level AlertLevel, source, issue string, runbook string) *Alert {
	id := generateAlertID()
	return &Alert{
		ID:      id,
		TS:      time.Now(),
		Level:   level,
		Source:  source,
		Issue:   issue,
		Files:   []string{},
		Runbook: runbook,
		Status:  StatusPending,
	}
}

// generateAlertID creates a unique alert ID: ALT-{timestamp}-{random}
func generateAlertID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("ALT-%d-%x", time.Now().Unix(), b)
}

// Enqueue adds an alert to the pending queue
func (q *Queue) Enqueue(a *Alert) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	f, err := os.OpenFile(q.queuePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open queue: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write alert: %w", err)
	}
	return nil
}

// Dequeue removes an alert from pending queue (by marking it moved to another file)
func (q *Queue) Dequeue(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Read all alerts, filter out the one with this ID
	alerts, err := q.readQueue(q.queuePath())
	if err != nil {
		return err
	}

	var remaining []*Alert
	for _, a := range alerts {
		if a.ID != id {
			remaining = append(remaining, a)
		}
	}

	// Rewrite queue without the removed alert
	return q.rewriteQueue(q.queuePath(), remaining)
}

// GetPending returns all pending alerts
func (q *Queue) GetPending() ([]*Alert, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.readQueue(q.queuePath())
}

// GetAutoFixed returns all auto-fixed alerts
func (q *Queue) GetAutoFixed() ([]*Alert, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.readQueue(q.autoFixedPath())
}

// GetAcknowledged returns all human-acknowledged alerts
func (q *Queue) GetAcknowledged() ([]*Alert, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.readQueue(q.acknowledgedPath())
}

// MarkAutoFixed moves an alert from pending to auto-fixed
func (q *Queue) MarkAutoFixed(id, action string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	alerts, err := q.readQueue(q.queuePath())
	if err != nil {
		return err
	}

	now := time.Now()
	var remaining []*Alert
	for _, a := range alerts {
		if a.ID == id {
			a.Status = StatusAutoFixed
			a.ResolvedTS = &now
			a.ResolvedBy = "OVAV-AGENTS"
			a.Resolution = action
			if err := q.appendToFile(q.autoFixedPath(), a); err != nil {
				return err
			}
		} else {
			remaining = append(remaining, a)
		}
	}

	return q.rewriteQueue(q.queuePath(), remaining)
}

// MarkAcknowledged moves an alert from pending to acknowledged
func (q *Queue) MarkAcknowledged(id, by, note string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	alerts, err := q.readQueue(q.queuePath())
	if err != nil {
		return err
	}

	now := time.Now()
	var remaining []*Alert
	for _, a := range alerts {
		if a.ID == id {
			a.Status = StatusAcknowledged
			a.ResolvedTS = &now
			a.ResolvedBy = by
			a.Resolution = note
			if err := q.appendToFile(q.acknowledgedPath(), a); err != nil {
				return err
			}
		} else {
			remaining = append(remaining, a)
		}
	}

	return q.rewriteQueue(q.queuePath(), remaining)
}

// Archive moves old resolved alerts to the archive
func (q *Queue) Archive(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check auto_fixed
	alerts, _ := q.readQueue(q.autoFixedPath())
	for _, a := range alerts {
		if a.ID == id {
			a.Status = StatusArchived
			return q.appendToFile(q.archivedPath(), a)
		}
	}

	// Check acknowledged
	alerts, _ = q.readQueue(q.acknowledgedPath())
	for _, a := range alerts {
		if a.ID == id {
			a.Status = StatusArchived
			return q.appendToFile(q.archivedPath(), a)
		}
	}

	return nil
}

// CountPending returns the number of pending alerts
func (q *Queue) CountPending() int {
	alerts, err := q.GetPending()
	if err != nil {
		return 0
	}
	return len(alerts)
}

// readQueue reads all alerts from a jsonl file
func (q *Queue) readQueue(path string) ([]*Alert, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []*Alert{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	var alerts []*Alert
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var a Alert
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			continue // Skip malformed lines
		}
		alerts = append(alerts, &a)
	}
	return alerts, nil
}

// rewriteQueue rewrites the entire queue file
func (q *Queue) rewriteQueue(path string, alerts []*Alert) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	for _, a := range alerts {
		data, err := json.Marshal(a)
		if err != nil {
			continue
		}
		f.Write(append(data, '\n'))
	}
	return nil
}

// appendToFile appends a single alert to a jsonl file
func (q *Queue) appendToFile(path string, a *Alert) error {
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}
