// Package agents provides OVAV agent position tracking.
// Worktree operations are handled by the OWS (OVAV Worktree System).
package agents

import "errors"

// Run executes the position stub — OWS handles worktree operations.
// Returns nil as this is a no-op implementation.
func Run() error {
	// No-op: OWS handles worktree operations
	return nil
}

// ErrOWSHandled indicates worktree operations were handled by OWS.
var ErrOWSHandled = errors.New("worktree operations handled by OWS")
