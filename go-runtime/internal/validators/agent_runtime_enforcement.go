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

type runtimeComponent struct {
	name       string
	path       string
	signatures []string
}

// runtimeComponents describes current Go-native runtime truth.
var runtimeComponents = []runtimeComponent{
	{name: "Service Area Router", path: "go-runtime/internal/agents/service_area.go", signatures: []string{"RouteRequest", "ServiceAreaForAgent", "InternalRepoAccessDeniedByDefault"}},
	{name: "Context Gateway", path: "go-runtime/internal/agents/context.go", signatures: []string{"RequestContext", "ResearchNoRepoDefault", "DecideContext"}},
	{name: "Context Firewall", path: "go-runtime/internal/validators/context_firewall_v2.go", signatures: []string{"type ContextFirewallV2", "Validate", "containsSuspiciousPattern"}},
	{name: "Tool Gateway", path: "go-runtime/internal/agents/tool_gateway.go", signatures: []string{"RequestTool", "RequiresPermission", "Decision"}},
	{name: "Handoff", path: "go-runtime/internal/agents/handoff.go", signatures: []string{"CreateHandoff", "EvaluateHandoff", "DeniedContext"}},
	{name: "Delegation Router", path: "go-runtime/internal/agents/delegation.go", signatures: []string{"DecideDelegation", "DelegationModeForSquad", "CriticalSquad"}},
	{name: "Observability", path: "go-runtime/internal/agents/observability.go", signatures: []string{
		"type TraceID string", "type TraceEvent struct", "type TraceSink interface",
		"NewTraceEvent", "Validate() error", "MarshalJSON", "NewFileTraceSink",
		"NewMemoryTraceSink", "RouteRequestWithTrace",
	}},
}

func (a *AgentRuntimeEnforcement) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	found := 0

	for _, component := range runtimeComponents {
		fullPath := filepath.Join(root, component.path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("MISSING: %s Go implementation not found: %s", component.name, component.path))
			continue
		}
		content := string(data)
		moduleOk := true
		for _, sig := range component.signatures {
			if !strings.Contains(content, sig) {
				moduleOk = false
				issues = append(issues, fmt.Sprintf("SIGNATURE: %s missing %q", component.path, sig))
			}
		}
		if moduleOk {
			found++
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  fmt.Sprintf("FAIL agent runtime enforcement — %d/%d Go components OK, %d issue(s)", found, len(runtimeComponents), len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  fmt.Sprintf("PASS agent runtime enforcement — %d/%d Go components verified", found, len(runtimeComponents)),
		Duration: time.Since(start),
	}
}

func fileContainsAll(root, rel string, signatures ...string) bool {
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return false
	}
	content := string(data)
	for _, signature := range signatures {
		if !strings.Contains(content, signature) {
			return false
		}
	}
	return true
}

var _ Validator = (*AgentRuntimeEnforcement)(nil)
