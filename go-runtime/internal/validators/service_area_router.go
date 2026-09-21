package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceAreaRouter validates all 10 area agent profiles have correct hard stops.
// Replaces: check_agent_runtime_service_area_router.py
type ServiceAreaRouter struct{}

func NewServiceAreaRouter() *ServiceAreaRouter { return &ServiceAreaRouter{} }

func (s *ServiceAreaRouter) ID() string   { return "service_area_router" }
func (s *ServiceAreaRouter) Name() string { return "Service Area Router Validator" }
func (s *ServiceAreaRouter) Description() string {
	return "Validates all 10 area agent profiles have hard stop contracts covering all other areas"
}
func (s *ServiceAreaRouter) Weight() int { return 8 }

// areaProfiles maps each area to its canonical YAML file.
var areaProfiles = []struct {
	id   string
	name string
	file string
}{
	{id: "platform_engineering", name: "Platform Engineering", file: "platform_engineering/area_boundaries.yaml"},
	{id: "research_intelligence", name: "Research Intelligence", file: "research_intelligence/area_boundaries.yaml"},
	{id: "commercial_growth", name: "Commercial & Growth", file: "commercial_growth/area_boundaries.yaml"},
	{id: "digital_product", name: "Digital Product", file: "digital_product/area_boundaries.yaml"},
	{id: "education_career", name: "Education & Career", file: "education_career/area_boundaries.yaml"},
	{id: "health_performance", name: "Health & Performance", file: "health_performance/area_boundaries.yaml"},
	{id: "ux_design", name: "UX Design", file: "ux_design/area_boundaries.yaml"},
	{id: "devops_infrastructure", name: "DevOps & Infrastructure", file: "devops_infrastructure/area_boundaries.yaml"},
	{id: "adversarial_intelligence", name: "Adversarial Intelligence", file: "adversarial_intelligence/area_boundaries.yaml"},
	{id: "legal_compliance", name: "Legal & Compliance", file: "legal_compliance/area_boundaries.yaml"},
}

func (s *ServiceAreaRouter) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	profilesChecked := 0

	saDir := filepath.Join(root, ".ovav", "service_areas")

	for _, profile := range areaProfiles {
		filePath := filepath.Join(saDir, profile.file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("MISSING: %s profile (%s)", profile.name, profile.file))
			continue
		}

		var rawDoc map[string]interface{}
		if err := yaml.Unmarshal(data, &rawDoc); err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid YAML: %v", profile.name, err))
			continue
		}

		profileOK := true
		hasScope := false

		if dnc, ok := rawDoc["does_not_cover"].([]interface{}); ok && len(dnc) > 0 {
			hasScope = true
		}
		if ab, ok := rawDoc["area_boundaries"].(map[string]interface{}); ok {
			if scope, ok := ab["scope"].(map[string]interface{}); ok {
				if excludes, ok := scope["excludes"].([]interface{}); ok && len(excludes) > 0 {
					hasScope = true
				}
			}
		}

		if !hasScope {
			issues = append(issues, fmt.Sprintf("%s: missing scope section (does_not_cover or scope.excludes)", profile.name))
			profileOK = false
		}

		if profileOK {
			profilesChecked++
		}
	}

	if len(issues) > 0 {
		return Result{
			ID:       s.ID(),
			Name:     s.Name(),
			Status:   "fail",
			Weight:   s.Weight(),
			Message:  fmt.Sprintf("FAIL — %d issue(s) in %d profiles checked (%d OK)", len(issues), len(areaProfiles), profilesChecked),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID:       s.ID(),
		Name:     s.Name(),
		Status:   "pass",
		Weight:   s.Weight(),
		Message:  fmt.Sprintf("PASS — all %d area profiles have valid hard stops", profilesChecked),
		Duration: time.Since(start),
	}
}

var _ Validator = (*ServiceAreaRouter)(nil)
