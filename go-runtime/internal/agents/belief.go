// belief.go — OVAV Belief Manager L7
// Manages agent belief states and emergent deprecation.
package agents

import (
	"fmt"
	"sync"
	"time"
)

// BeliefState is the governed lifecycle class for a belief.
type BeliefState string

const (
	BeliefStateRevocable BeliefState = "revocable"
	BeliefStateEmergent  BeliefState = "emergent"
)

// BeliefManager manages belief states for the feedback loop.
type BeliefManager struct {
	mu         sync.RWMutex
	beliefs    map[string]any
	states     map[string]BeliefState
	createdAt  map[string]time.Time
	deprecated map[string]bool
}

// NewBeliefManager creates a new BeliefManager.
func NewBeliefManager() *BeliefManager {
	return &BeliefManager{
		beliefs:    make(map[string]any),
		states:     make(map[string]BeliefState),
		createdAt:  make(map[string]time.Time),
		deprecated: make(map[string]bool),
	}
}

// AddBelief adds or updates a belief entry.
func (bm *BeliefManager) AddBelief(key string, value any) {
	_ = bm.AddBeliefWithState(key, value, BeliefStateRevocable, time.Now())
}

// AddBeliefWithState adds a timestamped belief and fails closed on unknown states.
func (bm *BeliefManager) AddBeliefWithState(key string, value any, state BeliefState, createdAt time.Time) error {
	if state != BeliefStateRevocable && state != BeliefStateEmergent {
		return fmt.Errorf("belief state %q is not governed", state)
	}
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.beliefs[key] = value
	bm.states[key] = state
	bm.createdAt[key] = createdAt
	delete(bm.deprecated, key)
	return nil

}

// AddEmergentBeliefAt adds an emergent belief with an explicit creation time.
func (bm *BeliefManager) AddEmergentBeliefAt(key string, value any, createdAt time.Time) error {
	return bm.AddBeliefWithState(key, value, BeliefStateEmergent, createdAt)
}

// DeprecateBelief marks a belief as deprecated and removes it.
func (bm *BeliefManager) DeprecateBelief(key string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.deprecated[key] = true
	delete(bm.beliefs, key)
	delete(bm.states, key)
	delete(bm.createdAt, key)
}

// DeprecateStaleEmergent deprecates emergent beliefs older than maxAge.
func (bm *BeliefManager) DeprecateStaleEmergent(maxAge time.Duration) {
	bm.DeprecateStaleEmergentAt(maxAge, time.Now())

}

// DeprecateStaleEmergentAt is the deterministic form used by tests and schedulers.
func (bm *BeliefManager) DeprecateStaleEmergentAt(maxAge time.Duration, now time.Time) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	cutoff := now.Add(-maxAge)
	for key, state := range bm.states {
		if state != BeliefStateEmergent || bm.createdAt[key].After(cutoff) {
			continue
		}
		bm.deprecated[key] = true
		delete(bm.beliefs, key)
		delete(bm.states, key)
		delete(bm.createdAt, key)
	}
}

// Belief returns a belief value without exposing internal maps.
func (bm *BeliefManager) Belief(key string) (any, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	value, ok := bm.beliefs[key]
	return value, ok
}

// IsDeprecated reports whether a belief has been explicitly or automatically deprecated.
func (bm *BeliefManager) IsDeprecated(key string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.deprecated[key]
}
