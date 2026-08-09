package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentRuntimeEnforcement validates agent runtime enforcement module presence and structure.
// Replaces: check_agent_runtime_enforcement.py
type AgentRuntimeEnforcement struct{}

func NewAgentRuntimeEnforcement() *AgentRuntimeEnforcement { return &AgentRuntimeEnforcement{} }

func (a *AgentRuntimeEnforcement) ID() string   { return "agent_runtime_enforcement" }
func (a *AgentRuntimeEnforcement) Name() string { return "Agent Runtime Enforcement" }
func (a *AgentRuntimeEnforcement) Description() string {
	return "Validates agent runtime modules: service_area_router, context_gateway, tool_gateway, handoff_protocol, delegation_router, observability_engine"
}
func (a *AgentRuntimeEnforcement) Weight() int { return 8 }

// Required runtime modules and their key signatures.
var runtimeModules = map[string][]string{
	"tools/agent_runtime/service_area_router.py": {
		"route_request",
		"service_area",
		"internal_repo_access_denied_by_default",
	},
	"tools/agent_runtime/context_gateway.py": {
		"request_context",
		"research_no_repo_default",
		"decision",
	},
	"tools/agent_runtime/tool_gateway.py": {
		"request_tool",
		"decision",
		"requires_permission",
	},
	"tools/agent_runtime/handoff_protocol.py": {
		"create_handoff",
		"decision",
		"denied_context",
	},
	"tools/agent_runtime/delegation_router.py": {
		"decide_delegation",
		"delegation_mode",
		"critical_squad",
	},
	"tools/agent_runtime/observability_engine.py": {
		"trace_event",
		"trace_id",
		"trace_",
	},
}

func (a *AgentRuntimeEnforcement) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	found := 0
	missing := 0

	for relPath, signatures := range runtimeModules {
		fullPath := filepath.Join(root, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			missing++
			issues = append(issues, fmt.Sprintf("MISSING: %s — agent runtime module not found", relPath))
			continue
		}
		content := string(data)
		moduleOk := true
		for _, sig := range signatures {
			if !strings.Contains(content, sig) {
				moduleOk = false
				issues = append(issues, fmt.Sprintf("SIGNATURE: %s missing '%s'", relPath, sig))
			}
		}
		if moduleOk {
			found++
		}
	}

	if missing > 0 || len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  fmt.Sprintf("FAIL agent runtime enforcement — %d/%d modules OK, %d issue(s)", found, len(runtimeModules), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  fmt.Sprintf("PASS agent runtime enforcement — %d/%d modules verified", found, len(runtimeModules)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*AgentRuntimeEnforcement)(nil)
