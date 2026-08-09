package scheduler

import (
	"testing"
	"time"
)

func TestDefaultTargets(t *testing.T) {
	targets := DefaultTargets()

	if len(targets) != 5 {
		t.Errorf("Expected 5 default targets, got %d", len(targets))
	}

	// Check all targets are enabled
	for _, tt := range targets {
		if !tt.Enabled {
			t.Errorf("Target %s should be enabled by default", tt.ID)
		}
	}

	// Check frequencies
	freqs := make(map[string]int)
	for _, tt := range targets {
		freqs[tt.Frequency]++
	}

	if freqs["daily"] != 4 {
		t.Errorf("Expected 4 daily targets, got %d", freqs["daily"])
	}
	if freqs["weekly"] != 1 {
		t.Errorf("Expected 1 weekly target, got %d", freqs["weekly"])
	}
}

func TestScheduler_ShouldRun(t *testing.T) {
	targets := DefaultTargets()
	s := New(targets)

	// Never run target should run
	target := targets[0]
	target.LastRun = time.Time{}
	if !s.ShouldRun(&target) {
		t.Error("Never-run target should run")
	}

	// Recently run target should not run
	target.LastRun = time.Now()
	if s.ShouldRun(&target) {
		t.Error("Recently-run target should not run")
	}

	// Disabled target should not run
	target.Enabled = false
	if s.ShouldRun(&target) {
		t.Error("Disabled target should not run")
	}
}

func TestScheduler_CalcNextRun(t *testing.T) {
	targets := DefaultTargets()
	s := New(targets)

	tests := []struct {
		freq     string
		expected time.Duration
	}{
		{"daily", 24 * time.Hour},
		{"weekly", 7 * 24 * time.Hour},
		{"unknown", 24 * time.Hour},
	}

	for _, tt := range tests {
		target := &Target{
			Frequency: tt.freq,
			LastRun:   time.Now(),
		}
		next := s.CalcNextRun(target)
		expected := target.LastRun.Add(tt.expected)

		if !next.Equal(expected) {
			t.Errorf("For %s: expected %v, got %v", tt.freq, expected, next)
		}
	}
}

func TestValidateFrequency(t *testing.T) {
	valid := []string{"daily", "weekly", "hourly"}
	for _, f := range valid {
		if err := ValidateFrequency(f); err != nil {
			t.Errorf("ValidateFrequency(%s) failed: %v", f, err)
		}
	}

	invalid := []string{"monthly", "yearly", "invalid"}
	for _, f := range invalid {
		if err := ValidateFrequency(f); err == nil {
			t.Errorf("ValidateFrequency(%s) should fail", f)
		}
	}
}

func TestScheduler_NextScheduled(t *testing.T) {
	targets := DefaultTargets()
	s := New(targets)

	// All targets have same next run initially
	next := s.NextScheduled()
	if next.IsZero() {
		t.Error("NextScheduled should not be zero")
	}
}
