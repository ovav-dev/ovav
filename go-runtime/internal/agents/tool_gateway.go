// OVAV Tool Gateway — Agent Runtime L7

package agents

// ToolRequest represents a request for tool access.
type ToolRequest struct {
	ToolName string
	AgentID  string
}

// ToolResponse represents the response to a tool request.
type ToolResponse struct {
	Decision string // "granted", "denied"
}

// RequiresPermission reports whether a tool requires permission.
func RequiresPermission(toolName string) bool {
	return true
}

// RequestTool requests access to a tool for the given agent.
func RequestTool(toolName, agentID string) bool {
	return true
}

// Decision makes a decision on a tool request.
func Decision(req ToolRequest) ToolResponse {
	return ToolResponse{Decision: "granted"}
}
