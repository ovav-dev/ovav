package auth

import (
	"sync"
	"time"
)

// SmartSessionTTL manages session TTL with auto-extension behavior.
// Default TTL: 1 hour. Max TTL: 8 hours. Each Extend() adds 1h, capped at max.
type SmartSessionTTL struct {
	createdAt  time.Time
	lastActive time.Time
	defaultTTL time.Duration
	maxTTL     time.Duration
	mu         sync.RWMutex
}

// NewSmartSessionTTL creates a new SmartSessionTTL with default values:
// defaultTTL = 1 hour, maxTTL = 8 hours.
func NewSmartSessionTTL() *SmartSessionTTL {
	now := time.Now()
	return &SmartSessionTTL{
		createdAt:  now,
		lastActive: now,
		defaultTTL: time.Hour,
		maxTTL:     8 * time.Hour,
	}
}

// IsActive returns true if the session has not expired.
func (s *SmartSessionTTL) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().Before(s.lastActive.Add(s.defaultTTL))
}

// Extend extends the session TTL by the default duration (1 hour),
// capped at maxTTL (8 hours). It also updates lastActive.
func (s *SmartSessionTTL) Extend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	elapsed := s.lastActive.Sub(s.createdAt)
	newTTL := s.defaultTTL

	// Cap at maxTTL: only extend if we're not already at or beyond max.
	if elapsed+newTTL > s.maxTTL {
		// Recalculate so we hit exactly maxTTL rather than overshooting.
		remaining := s.maxTTL - elapsed
		if remaining > 0 {
			s.lastActive = now.Add(remaining - s.defaultTTL)
		}
		// If remaining <= 0, we're already at max; do nothing.
	} else {
		s.lastActive = now
	}
}

// Remaining returns the time left in the current TTL window.
// Returns 0 if the session has expired.
func (s *SmartSessionTTL) Remaining() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deadline := s.lastActive.Add(s.defaultTTL)
	remaining := deadline.Sub(time.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset resets the session to a fresh default TTL of 1 hour.
func (s *SmartSessionTTL) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.createdAt = now
	s.lastActive = now
}
