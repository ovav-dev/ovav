// Package scheduler implements OVAV's research scheduling system.
package scheduler

import (
	"fmt"
	"time"
)

// Target represents a research target.
type Target struct {
	ID        string
	Name      string
	URL       string
	Frequency string
	LastRun   time.Time
	NextRun   time.Time
	Enabled   bool
}

// Scheduler manages research target scheduling.
type Scheduler struct {
	targets []Target
}

// New creates a new Scheduler with the given targets.
func New(targets []Target) *Scheduler {
	return &Scheduler{targets: targets}
}

// ShouldRun checks if a target should run now based on its frequency.
func (s *Scheduler) ShouldRun(target *Target) bool {
	if !target.Enabled {
		return false
	}

	switch target.Frequency {
	case "daily":
		return time.Since(target.LastRun) >= 24*time.Hour
	case "weekly":
		return time.Since(target.LastRun) >= 7*24*time.Hour
	default:
		return time.Since(target.LastRun) >= 24*time.Hour
	}
}

// NextScheduled returns when the next research run is due.
func (s *Scheduler) NextScheduled() time.Time {
	var next time.Time

	for _, t := range s.targets {
		if !t.Enabled {
			continue
		}
		if next.IsZero() || t.NextRun.Before(next) {
			next = t.NextRun
		}
	}

	return next
}

// UpdateTarget updates a target after it's been run.
func (s *Scheduler) UpdateTarget(target *Target) {
	target.LastRun = time.Now()
	target.NextRun = s.CalcNextRun(target)
}

// CalcNextRun calculates the next run time for a target.
func (s *Scheduler) CalcNextRun(target *Target) time.Time {
	switch target.Frequency {
	case "daily":
		return target.LastRun.Add(24 * time.Hour)
	case "weekly":
		return target.LastRun.Add(7 * 24 * time.Hour)
	default:
		return target.LastRun.Add(24 * time.Hour)
	}
}

// DefaultTargets returns the default research targets for OVAV.
func DefaultTargets() []Target {
	now := time.Now()
	return []Target{
		{
			ID:        "openai",
			Name:      "OpenAI",
			URL:       "https://platform.openai.com/docs/changelog",
			Frequency: "daily",
			LastRun:   time.Time{},
			NextRun:   now,
			Enabled:   true,
		},
		{
			ID:        "anthropic",
			Name:      "Anthropic",
			URL:       "https://docs.anthropic.com/en/release-notes/overview",
			Frequency: "daily",
			LastRun:   time.Time{},
			NextRun:   now,
			Enabled:   true,
		},
		{
			ID:        "google-ai",
			Name:      "Google AI",
			URL:       "https://ai.google/discover/palm-api",
			Frequency: "daily",
			LastRun:   time.Time{},
			NextRun:   now,
			Enabled:   true,
		},
		{
			ID:        "openrouter",
			Name:      "OpenRouter",
			URL:       "https://openrouter.ai/docs",
			Frequency: "daily",
			LastRun:   time.Time{},
			NextRun:   now,
			Enabled:   true,
		},
		{
			ID:        "owasp",
			Name:      "OWASP",
			URL:       "https://owasp.org/",
			Frequency: "weekly",
			LastRun:   time.Time{},
			NextRun:   now,
			Enabled:   true,
		},
	}
}

// ValidateFrequency validates a frequency string.
func ValidateFrequency(freq string) error {
	switch freq {
	case "daily", "weekly", "hourly":
		return nil
	default:
		return fmt.Errorf("invalid frequency: %q (valid: daily, weekly, hourly)", freq)
	}
}
