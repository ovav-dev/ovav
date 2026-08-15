// OVAV Service Area Router — Agent Runtime L7

package agents

import "errors"

// ErrNoRoute indicates no service area could be found for the request.
var ErrNoRoute = errors.New("no service area route found")

// ErrUnknownAgent indicates the agent ID is not recognized.
var ErrUnknownAgent = errors.New("unknown agent ID")

// ServiceArea represents a service area domain.
type ServiceArea string

const (
	ServiceAreaPlatform   ServiceArea = "platform"
	ServiceAreaResearch   ServiceArea = "research"
	ServiceAreaUX         ServiceArea = "ux"
	ServiceAreaCommercial ServiceArea = "commercial"
	ServiceAreaEducation  ServiceArea = "education"
	ServiceAreaHealth     ServiceArea = "health"
	ServiceAreaDevOps     ServiceArea = "devops"
	ServiceAreaSecurity   ServiceArea = "security"
	ServiceAreaData       ServiceArea = "data"
	ServiceAreaLeadership ServiceArea = "leadership"
)

// RouteRequest routes a request to the appropriate service area.
// Returns the target service area or an error if routing fails.
func RouteRequest(request string) (ServiceArea, error) {
	if request == "" {
		return "", ErrNoRoute
	}

	// Simple keyword-based routing
	switch {
	case containsAny(request, "platform", "runtime", "governance", "repo", "opencode"):
		return ServiceAreaPlatform, nil
	case containsAny(request, "research", "evidence", "benchmark", "intelligence"):
		return ServiceAreaResearch, nil
	case containsAny(request, "ux", "ui", "design", "accessibility", "wcag"):
		return ServiceAreaUX, nil
	case containsAny(request, "commercial", "growth", "monetization", "business"):
		return ServiceAreaCommercial, nil
	case containsAny(request, "education", "learning", "curriculum", "career"):
		return ServiceAreaEducation, nil
	case containsAny(request, "health", "performance", "wellness", "nutrition"):
		return ServiceAreaHealth, nil
	case containsAny(request, "devops", "infrastructure", "deployment", "ci/cd"):
		return ServiceAreaDevOps, nil
	case containsAny(request, "security", "vault", "audit", "protection"):
		return ServiceAreaSecurity, nil
	case containsAny(request, "data", "analytics", "metrics", "warehouse"):
		return ServiceAreaData, nil
	case containsAny(request, "leadership", "strategy", "roadmap", "vision"):
		return ServiceAreaLeadership, nil
	default:
		return "", ErrNoRoute
	}
}

// ServiceAreaForAgent returns the service area for a given agent ID.
// Returns ErrUnknownAgent if the agent is not recognized.
func ServiceAreaForAgent(agentID string) (ServiceArea, error) {
	if agentID == "" {
		return "", ErrUnknownAgent
	}

	// Parse agent prefix (area-<name>, lead-<name>, team-<name>)
	switch {
	case len(agentID) >= 5 && agentID[:5] == "area-":
		name := agentID[5:]
		return agentIDToServiceArea(name)
	case len(agentID) >= 5 && agentID[:5] == "lead-":
		name := agentID[5:]
		return agentIDToServiceArea(name)
	case len(agentID) >= 6 && agentID[:6] == "team-":
		name := agentID[6:]
		return agentIDToServiceArea(name)
	default:
		return "", ErrUnknownAgent
	}
}

// InternalRepoAccessDeniedByDefault returns whether internal repo access
// is denied by default for the given agent. Returns true for all unknown agents.
func InternalRepoAccessDeniedByDefault(agentID string) bool {
	if agentID == "" {
		return true
	}
	_, err := ServiceAreaForAgent(agentID)
	return err != nil
}

// agentIDToServiceArea maps an agent name to its service area.
func agentIDToServiceArea(name string) (ServiceArea, error) {
	areaMap := map[string]ServiceArea{
		"thavren": ServiceAreaPlatform,
		"eidren":  ServiceAreaResearch,
		"elena":   ServiceAreaUX,
		"sofia":   ServiceAreaCommercial,
		"valeria": ServiceAreaEducation,
		"renata":  ServiceAreaHealth,
		"marco":   ServiceAreaDevOps,
		"clara":   ServiceAreaSecurity,
		"braka":   ServiceAreaLeadership,
		// Platform squad
		"lucas":  ServiceAreaPlatform,
		"andres": ServiceAreaPlatform,
	}

	area, ok := areaMap[name]
	if !ok {
		return "", ErrUnknownAgent
	}
	return area, nil
}

// containsAny checks if the target string contains any of the keywords.
func containsAny(target string, keywords ...string) bool {
	for _, kw := range keywords {
		if len(target) >= len(kw) {
			for i := 0; i <= len(target)-len(kw); i++ {
				if target[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}
