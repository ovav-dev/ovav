package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// areaProfiles maps each area to its agent file and expected hard stop terms.
var areaProfiles = []struct {
	id        string
	name      string
	file      string
	hardStops []string // terms that MUST appear in the profile
}{
	{
		id:   "platform_engineering",
		name: "Platform Engineering",
		file: "area-platform-engineering.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO diseño UI/UX",
			"NO investigación de fuentes",
			"NO frontend React/TypeScript",
			"NO estrategia comercial",
			"NO contenido educativo",
			"NO nutrición",
			"NO DevOps",
			"NO testing adversarial",
			"NO contratos legales",
		},
	},
	{
		id:   "research_intelligence",
		name: "Research Intelligence",
		file: "area-research-intelligence.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO Platform Engineering",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO educación",
			"NO DevOps",
		},
	},
	{
		id:   "commercial_growth",
		name: "Commercial & Growth",
		file: "area-commercial-growth.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime",
			"NO investigación técnica",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO nutrición",
			"NO contenido educativo",
			"NO DevOps",
			"NO testing adversarial",
		},
	},
	{
		id:   "digital_product",
		name: "Digital Product",
		file: "area-digital-product.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime Go",
			"NO investigación",
			"NO diseño UI/UX desde cero",
			"NO estrategia comercial",
			"NO nutrición",
			"NO contenido educativo",
			"NO infraestructura",
			"NO testing adversarial",
		},
	},
	{
		id:   "education_career",
		name: "Education & Career",
		file: "area-education-career.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime",
			"NO investigación de mercado",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO DevOps",
			"NO testing adversarial",
		},
	},
	{
		id:   "health_performance",
		name: "Health & Performance",
		file: "area-health-performance.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO Platform Engineering",
			"NO Research Intelligence",
			"NO UX Design",
			"NO Digital Product",
			"NO Commercial",
			"NO Education",
			"NO DevOps",
			"NO Adversarial",
		},
	},
	{
		id:   "ux_design",
		name: "UX Design",
		file: "area-ux-design.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime",
			"NO investigación",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO contenido educativo",
			"NO DevOps",
		},
	},
	{
		id:   "devops_infrastructure",
		name: "DevOps & Infrastructure",
		file: "area-devops-infrastructure.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO Platform Engineering",
			"NO investigación",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO educación",
			"NO Adversarial",
		},
	},
	{
		id:   "adversarial_intelligence",
		name: "Adversarial Intelligence",
		file: "area-adversarial-intelligence.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime Go",
			"NO investigación de fuentes",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO contenido educativo",
			"NO infraestructura",
			"NO escribir código de producción",
			"NO arreglar vulnerabilidades",
		},
	},
	{
		id:   "legal_compliance",
		name: "Legal & Compliance",
		file: "area-legal-compliance.md",
		hardStops: []string{
			"HARD STOP", "Fuera de mi área",
			"NO runtime Go",
			"NO investigación",
			"NO diseño UI/UX",
			"NO desarrollo de producto",
			"NO estrategia comercial",
			"NO nutrición",
			"NO contenido educativo",
			"NO infraestructura",
			"NO Adversarial",
		},
	},
}

func (s *ServiceAreaRouter) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	profilesChecked := 0

	agentsDir := filepath.Join(root, "runtimes", "opencode", "agents")

	for _, profile := range areaProfiles {
		filePath := filepath.Join(agentsDir, profile.file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("MISSING: %s profile (%s)", profile.name, profile.file))
			continue
		}

		text := string(data)
		profileOK := true

		for _, term := range profile.hardStops {
			if !strings.Contains(text, term) {
				issues = append(issues, fmt.Sprintf("%s: missing hard stop term: %q", profile.name, term))
				profileOK = false
			}
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
