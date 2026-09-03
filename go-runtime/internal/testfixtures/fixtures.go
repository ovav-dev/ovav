// Package fixtures provides shared test fixtures for OVAV integration tests.
// This ensures all tests have consistent, complete environment simulation.
package fixtures

import (
	"os"
	"path/filepath"
)

// BuildCompleteServiceAreas creates a full .ovav/service_areas/ structure
// with all required areas, leads, and contracts for integration testing.
func BuildCompleteServiceAreas(root string) error {
	areas := []string{
		"platform_engineering",
		"research_intelligence",
		"commercial_growth",
		"digital_product",
		"devops_infrastructure",
		"ux_design",
		"legal_compliance",
		"education_career",
		"health_performance",
		"adversarial_intelligence",
	}

	for _, area := range areas {
		areaDir := filepath.Join(root, ".ovav", "service_areas", area)
		if err := os.MkdirAll(areaDir, 0755); err != nil {
			return err
		}

		// Create area_boundaries.yaml
		boundaries := BuildAreaBoundaries(area)
		if err := os.WriteFile(filepath.Join(areaDir, "area_boundaries.yaml"), []byte(boundaries), 0644); err != nil {
			return err
		}

		// Create lead_contract.yaml
		contract := BuildLeadContract(area)
		if err := os.WriteFile(filepath.Join(areaDir, "lead_contract.yaml"), []byte(contract), 0644); err != nil {
			return err
		}
	}

	return nil
}

// BuildAreaBoundaries creates a valid area_boundaries.yaml for a given area
func BuildAreaBoundaries(areaID string) string {
	return `area: ` + areaID + `
canonical_area_name: "` + areaID + `"
lead: thavren

covers:
  - ` + areaID + `_coverage:
      description: Coverage for ` + areaID + `
      examples: []

does_not_cover:
  - other_areas:
      description: Other areas not in scope
      route_to: "Other leads"

contracts:
  - visual_delivery_contract.yaml
  - safe_stop_contract.yaml
  - context_economy_contract.yaml
`
}

// BuildLeadContract creates a valid lead_contract.yaml for a given area
func BuildLeadContract(areaID string) string {
	return `lead_contract:
  version: "2.0.0"
  lead: thavren
  area: ` + areaID + `

  covers:
    - ` + areaID + `_domain:
        description: Coverage for ` + areaID + ` domain

  does_not_cover:
    - other_areas: Other areas not in scope

  authority: lead_only for micro/simple work
`
}

// BuildHarnessAgents creates minimal harness agent files for integration tests
func BuildHarnessAgents(root, harness string) error {
	agentsDir := filepath.Join(root, "go-runtime", "internal", "runtimes", harness, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	areas := []string{
		"platform_engineering",
		"research_intelligence",
		"commercial_growth",
		"digital_product",
		"devops_infrastructure",
		"ux_design",
		"legal_compliance",
		"education_career",
		"health_performance",
		"adversarial_intelligence",
	}

	leads := []string{
		"thavren",
		"eidren",
		"sofia",
		"dante",
		"uriel",
		"elena",
		"camila",
		"valeria",
		"renata",
		"kenji",
	}

	// Area files
	for _, area := range areas {
		content := `---\nname: "` + area + `"\nmode: primary\nhidden: false\n---\n\n# ` + area + `\n`
		if err := os.WriteFile(filepath.Join(agentsDir, "area-"+area+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	// Lead files
	for _, lead := range leads {
		content := `---\nname: "` + lead + `"\nmode: lead\nhidden: false\n---\n\n# ` + lead + `\n`
		if err := os.WriteFile(filepath.Join(agentsDir, "lead-"+lead+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
