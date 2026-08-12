// Package agents provides OVAV agent runtime primitives.
package agents

// ContextType represents the class of context being requested.
type ContextType string

// ContextRequest captures a request for agent context.
type ContextRequest struct {
	AgentID      string
	ContextType  ContextType
	RequestedAt  int64
}

// DecisionOutcome represents the result of a context decision.
type DecisionOutcome string

const (
	DecisionGranted  DecisionOutcome = "granted"
	DecisionDenied   DecisionOutcome = "denied"
	DecisionDeferred DecisionOutcome = "deferred"
)

// requestContext returns context data for an agent.
// TODO: integrate with memory subsystem for live context.
func RequestContext(agentID string, contextType ContextType) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":     agentID,
		"context_type": contextType,
		"active":       false,
	}
}

// ResearchNoRepoDefault returns whether research agents have default repo access.
func ResearchNoRepoDefault(agentID string) bool {
	// Research agents do not receive repo context by default.
	return false
}

// DecideContext determines whether to grant or deny a context request.
func DecideContext(ctxReq ContextRequest) DecisionOutcome {
	// Default policy: grant.
	// TODO: wire into governor policy engine for conditional decisions.
	return DecisionGranted
}
