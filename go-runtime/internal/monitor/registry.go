package monitor

import (
	"context"
	"time"

	"github.com/ovav/ovav/internal/monitor/alerts"
)

// Monitor is the base interface for all system monitors.
// Each monitor runs on a defined interval and emits alerts
// when it detects issues.
type Monitor interface {
	// Name returns the monitor's identifier
	Name() string

	// Interval returns how often this monitor should run
	Interval() time.Duration

	// Run executes the monitor check and returns any alerts
	Run(ctx context.Context) ([]*alerts.Alert, error)
}

// MonitorRegistry holds all registered monitors and their dispatcher
type MonitorRegistry struct {
	monitors   []Monitor
	dispatcher *alerts.Dispatcher
}

// NewRegistry creates a new monitor registry
func NewRegistry(d *alerts.Dispatcher) *MonitorRegistry {
	return &MonitorRegistry{
		monitors:   []Monitor{},
		dispatcher: d,
	}
}

// Register adds a monitor to the registry
func (r *MonitorRegistry) Register(m Monitor) {
	r.monitors = append(r.monitors, m)
}

// GetMonitors returns all registered monitors
func (r *MonitorRegistry) GetMonitors() []Monitor {
	return r.monitors
}

// RunAll executes all registered monitors once
func (r *MonitorRegistry) RunAll(ctx context.Context) ([]*alerts.Alert, error) {
	var allAlerts []*alerts.Alert

	for _, m := range r.monitors {
		monitorAlerts, err := m.Run(ctx)
		if err != nil {
			// Monitor failed — emit error alert but continue
			allAlerts = append(allAlerts, alerts.NewAlert(
				alerts.LevelERROR,
				m.Name(),
				"monitor run failed: "+err.Error(),
				"", // no auto-fix for monitor failures
			))
			continue
		}

		// Dispatch each alert through the dispatcher
		for _, a := range monitorAlerts {
			if err := r.dispatcher.Dispatch(ctx, a); err != nil {
				// Dispatch failed — still include alert
				allAlerts = append(allAlerts, a)
			}
		}
	}

	return allAlerts, nil
}

// RunLoop runs all monitors on their configured intervals continuously
// This is meant to run as a background goroutine
func (r *MonitorRegistry) RunLoop(ctx context.Context) {
	type monitorNext struct {
		m       Monitor
		nextRun time.Time
	}

	var pending []monitorNext
	for _, m := range r.monitors {
		pending = append(pending, monitorNext{
			m:       m,
			nextRun: time.Now().Add(m.Interval()),
		})
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for i, mn := range pending {
				if now.After(mn.nextRun) || now.Equal(mn.nextRun) {
					go func(m Monitor) {
						m.Run(ctx)
					}(mn.m)
					pending[i].nextRun = now.Add(mn.m.Interval())
				}
			}
		}
	}
}

// HealthReport is the summary of all monitor runs
type HealthReport struct {
	TS         time.Time
	Monitors   int
	TotalAlerts int
	CRIT       int
	ERROR      int
	WARN       int
	INFO       int
	AutoFixed  int
	Pending    int
	Duration   time.Duration
	MonitorStatus map[string]string // monitor name -> status
}

// GenerateHealthReport creates a summary report
func (r *MonitorRegistry) GenerateHealthReport(ctx context.Context) *HealthReport {
	report := &HealthReport{
		TS:             time.Now(),
		Monitors:       len(r.monitors),
		MonitorStatus:  make(map[string]string),
	}

	pending, _ := r.dispatcher.Queue().GetPending()
	report.Pending = len(pending)

	autoFixed, _ := r.dispatcher.Queue().GetAutoFixed()
	report.AutoFixed = len(autoFixed)

	for _, a := range pending {
		report.TotalAlerts++
		switch a.Level {
		case alerts.LevelCRIT:
			report.CRIT++
		case alerts.LevelERROR:
			report.ERROR++
		case alerts.LevelWARN:
			report.WARN++
		case alerts.LevelINFO:
			report.INFO++
		}
	}

	for _, m := range r.monitors {
		report.MonitorStatus[m.Name()] = "ok"
	}

	return report
}

// Queue returns the underlying alert queue
func (r *MonitorRegistry) Queue() *alerts.Queue {
	return r.dispatcher.Queue()
}
