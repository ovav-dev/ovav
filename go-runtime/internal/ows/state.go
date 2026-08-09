// [OVAV-FIX] Replace hardcoded credential with os.Getenv() from secure vault
// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
// [OVAV-FIX] Use parameterized queries (sql.Stmt) instead of string concatenation
package ows

import (
	"fmt"
	"strings"
	"time"
)

// ── Worktree State Machine ──────────────────────────────────────────────────────

// State represents a worktree lifecycle state.
type State string

const (
	StateCreated    State = "CREATED"
	StateActive     State = "ACTIVE"
	StateDirty      State = "DIRTY"
	StateVerified   State = "VERIFIED"
	StateIntegrated State = "INTEGRATED"
	StateCleaned    State = "CLEANED"
	StateLocked     State = "LOCKED"
	StateFailed     State = "FAILED"
	StateRescued    State = "RESCUED"
	StateStale      State = "STALE"
)

// Event represents a state transition trigger.
type Event string

const (
	EvWorkStarted         Event = "work_started"
	EvConflictDetected    Event = "conflict_detected"
	EvConflictResolved    Event = "conflict_resolved"
	EvUpdateSuccess       Event = "update_success"
	EvLockRequested       Event = "lock_requested"
	EvUnlockRequested     Event = "unlock_requested"
	EvVerificationPassed  Event = "verification_passed"
	EvVerificationFailed  Event = "verification_failed"
	EvIntegrationComplete Event = "integration_complete"
	EvCleanupComplete     Event = "cleanup_complete"
	EvRescueRequested     Event = "rescue_requested"
	EvStaleDetected       Event = "stale_detected"
)

// Transition defines a valid state transition with optional guard condition.
type Transition struct {
	From      State
	Event     Event
	To        State
	Condition string // human-readable condition
}

// TransitionTable is the canonical state machine definition.
// It can be exported as DOT/Graphviz for visualization.
var TransitionTable = []Transition{
	{StateCreated, EvWorkStarted, StateActive, "Worktree exists on disk"},
	{StateActive, EvConflictDetected, StateDirty, "Conflict detected in rebase/merge"},
	{StateActive, EvUpdateSuccess, StateActive, "Rebase completed cleanly"},
	{StateActive, EvLockRequested, StateLocked, "owlk executed by owner or lead"},
	{StateActive, EvVerificationPassed, StateVerified, "owv passes all checks"},
	{StateActive, EvVerificationFailed, StateFailed, "owv found issues"},
	{StateActive, EvStaleDetected, StateStale, "Worktree inactive >7 days"},
	{StateDirty, EvConflictResolved, StateActive, "Manual resolution + owu"},
	{StateVerified, EvIntegrationComplete, StateIntegrated, "Merge successful to develop"},
	{StateIntegrated, EvCleanupComplete, StateCleaned, "Worktree deleted + pruned"},
	{StateFailed, EvRescueRequested, StateRescued, "owr recovered state"},
	{StateLocked, EvUnlockRequested, StateActive, "Owner or lead unlocked"},
	{StateStale, EvUpdateSuccess, StateActive, "owu synced policies + rebase"},
}

// AllStates returns all valid states for validation.
func AllStates() []State {
	return []State{
		StateCreated, StateActive, StateDirty, StateVerified,
		StateIntegrated, StateCleaned, StateLocked, StateFailed,
		StateRescued, StateStale,
	}
}

// ValidTransition checks if a transition is valid according to the state machine.
func ValidTransition(from State, event Event) (State, bool) {
	for _, t := range TransitionTable {
		if t.From == from && t.Event == event {
			return t.To, true
		}
	}
	return "", false
}

// ── Worktree Record ──────────────────────────────────────────────────────────────

// WorktreeRecord represents a tracked worktree in the state machine.
type WorktreeRecord struct {
	ID            string    `json:"id"`          // worktree name (e.g., "task/feature-login")
	Branch        string    `json:"branch"`      // git branch name
	Profile       string    `json:"profile"`     // profile name
	Owner         string    `json:"owner"`       // user or agent who created it
	State         State     `json:"state"`       // current lifecycle state
	BaseBranch    string    `json:"base_branch"` // develop or main
	MergeTo       string    `json:"merge_to"`    // target for owd
	PolicyLevel   string    `json:"policy_level"`
	PolicyVer     int       `json:"policy_ver"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Locked        bool      `json:"locked"`
	LockReason    string    `json:"lock_reason,omitempty"`
	ModifiedFiles []string  `json:"modified_files,omitempty"` // for conflict prediction
}

// ── Transition Execution ─────────────────────────────────────────────────────────

// ExecuteTransition validates and executes a state transition on a worktree record.
// Returns an error if the transition is invalid.
func ExecuteTransition(wt *WorktreeRecord, event Event) error {
	next, ok := ValidTransition(wt.State, event)
	if !ok {
		return fmt.Errorf("invalid transition: %s → (%s) not allowed", wt.State, event)
	}
	wt.State = next
	wt.UpdatedAt = time.Now()
	return nil
}

// ── Visualization ────────────────────────────────────────────────────────────────

// DOTGraph returns the state machine as a Graphviz DOT representation.
// Usage: ovav worktree status --graph
func DOTGraph() string {
	var b strings.Builder
	b.WriteString("digraph OWS_StateMachine {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, style=rounded];\n\n")

	for _, t := range TransitionTable {
		fmt.Fprintf(&b, "  %s -> %s [label=\"%s\"];\n", t.From, t.To, t.Event)
	}

	b.WriteString("}\n")
	return b.String()
}

// ASCIIStateMatrix returns a human-readable transition matrix for CLI output.
func ASCIIStateMatrix() string {
	var b strings.Builder
	b.WriteString("State Transitions:\n")
	b.WriteString(strings.Repeat("-", 72) + "\n")

	for _, t := range TransitionTable {
		fmt.Fprintf(&b, "  %-12s ── %-22s ──► %-12s\n", t.From, t.Event, t.To)
	}

	return b.String()
}
