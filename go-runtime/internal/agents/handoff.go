// OVAV Handoff Protocol — Agent Runtime L7

package agents

import (
	"errors"
	"time"
)

// Handoff represents a transfer of context between two agents.
type Handoff struct {
	From    string
	To      string
	Context map[string]any
}

// HandoffResult is the outcome of a handoff decision.
type HandoffResult struct {
	Status       string
	Handoff      *Handoff
	DeniedCtx    map[string]any
	DecidedAt    time.Time
}

// ErrHandoffDenied is returned when a handoff is rejected.
var ErrHandoffDenied = errors.New("handoff denied")

// CreateHandoff creates a new handoff between two agents.
func CreateHandoff(fromAgent, toAgent string, ctx map[string]any) *Handoff {
	return &Handoff{
		From:    fromAgent,
		To:      toAgent,
		Context: ctx,
	}
}

// EvaluateHandoff evaluates a handoff and returns the result.
// Returns ErrHandoffDenied if the handoff is rejected.
func EvaluateHandoff(h *Handoff) (*HandoffResult, error) {
	// TODO: implement actual approval logic with governance checks
	result := &HandoffResult{
		Status:    "approved",
		Handoff:   h,
		DeniedCtx: nil,
		DecidedAt: time.Now().UTC(),
	}
	return result, nil
}

// DeniedContext returns the denial context for a rejected handoff.
func DeniedContext(h *Handoff) map[string]any {
	return nil
}
