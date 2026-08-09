package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ServiceAreaGovernance validates that all service area registries exist
// and contain required governance terms (fail_closed, deny_by_default, etc.).
// Replaces: check_service_area_governance.py
type ServiceAreaGovernance struct{}

func NewServiceAreaGovernance() *ServiceAreaGovernance { return &ServiceAreaGovernance{} }

func (s *ServiceAreaGovernance) ID() string   { return "service_area_governance" }
func (s *ServiceAreaGovernance) Name() string { return "Service Area Governance" }
func (s *ServiceAreaGovernance) Description() string {
	return "Validates service area registry files exist and contain required governance terms"
}
func (s *ServiceAreaGovernance) Weight() int { return 5 }

// Required service area files that must exist.
var requiredServiceAreaFiles = []string{
	".ovav/service_areas/registry.yaml",
	".ovav/service_areas/areas/platform_engineering.yaml",
	".ovav/service_areas/areas/research_intelligence.yaml",
	".ovav/service_areas/shared/context_firewall.yaml",
	".ovav/service_areas/shared/source_registry.yaml",
	".ovav/service_areas/shared/tool_access_policy.yaml",
	".ovav/service_areas/shared/delegation_policy.yaml",
	".ovav/service_areas/shared/delivery_contracts.yaml",
	".ovav/service_areas/shared/handoff_protocol.yaml",
	".ovav/service_areas/shared/observability_policy.yaml",
	".ovav/service_areas/shared/model_budget_policy.yaml",
	".ovav/service_areas/shared/session_capsule_policy.yaml",
}

// Governance terms that must appear in specific files.
var serviceAreaTermChecks = map[string][]string{
	".ovav/service_areas/registry.yaml": {
		"platform_engineering",
		"research_intelligence",
		"active_p0_count: 2",
	},
	".ovav/service_areas/areas/research_intelligence.yaml": {
		"denied_by_default",
		"repo_root",
		"conceptual_or_external_research",
	},
	".ovav/service_areas/shared/context_firewall.yaml": {
		"fail_closed: true",
		"repo_root_default: deny",
	},
	".ovav/service_areas/shared/source_registry.yaml": {
		"no_research_repo_root_default: true",
		"unknown_path: deny_or_requires_permission",
	},
	".ovav/service_areas/shared/tool_access_policy.yaml": {
		"fail_closed: true",
		"edit_repo_files",
		"decision: deny",
	},
	".ovav/service_areas/shared/handoff_protocol.yaml": {
		"raw_chat_history",
		"denied_by_default",
		"controlled document",
	},
	".ovav/service_areas/shared/observability_policy.yaml": {
		"trace_id",
		"research_no_repo_default",
		"tool_capability_boundary",
	},
}

func (s *ServiceAreaGovernance) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Check all required files exist
	for _, path := range requiredServiceAreaFiles {
		fullPath := filepath.Join(root, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("missing required service area file: %s", path))
		}
	}

	// 2. Check governance terms in specific files
	for path, terms := range serviceAreaTermChecks {
		fullPath := filepath.Join(root, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue // Already reported as missing above
		}
		body := strings.ToLower(string(data))
		for _, term := range terms {
			if !strings.Contains(body, strings.ToLower(term)) {
				issues = append(issues, fmt.Sprintf("%s missing required governance term: %s", path, term))
			}
		}
	}

	if len(issues) > 0 {
		return Result{
			ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL service area governance — %d issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	return Result{
		ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  "PASS service area governance — all files present with required terms",
		Duration: time.Since(start),
	}
}

var _ Validator = (*ServiceAreaGovernance)(nil)
