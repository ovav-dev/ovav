package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ServiceProfiles validates the service_profiles.yaml, service_lanes.yaml, and squads.yaml registries.
// Replaces: validate_service_profiles.py
type ServiceProfiles struct{}

func NewServiceProfiles() *ServiceProfiles { return &ServiceProfiles{} }

func (s *ServiceProfiles) ID() string   { return "validate_service_profiles" }
func (s *ServiceProfiles) Name() string { return "Service Profiles Validator" }
func (s *ServiceProfiles) Description() string {
	return "Validates service_profiles.yaml against expected profiles, lanes, and squads"
}
func (s *ServiceProfiles) Weight() int { return 7 }

type expectedProfile struct {
	DisplayName    string
	LeadOperator   string
	Squad          string
	PolicyEnvelope string
	MemoryScope    string
	EvalSuite      string
}

var expectedProfiles = map[string]expectedProfile{
	"ovav_systems_architect": {
		DisplayName:    "OVAV-Systems Architect",
		LeadOperator:   "thavren",
		Squad:          "systems_architecture_squad",
		PolicyEnvelope: "systems_authoritative_guarded",
		MemoryScope:    "customer_project_profile_operator",
		EvalSuite:      "systems_architect_p0",
	},
	"ovav_research_analyst": {
		DisplayName:    "OVAV-Research Analyst",
		LeadOperator:   "eidren",
		Squad:          "research_intelligence_squad",
		PolicyEnvelope: "research_evidence_guarded",
		MemoryScope:    "customer_project_profile_operator",
		EvalSuite:      "research_analyst_p0",
	},
	"ovav_health_performance": {
		DisplayName:    "OVAV-Health & Performance Science",
		LeadOperator:   "renata",
		Squad:          "health_performance_squad",
		PolicyEnvelope: "health_performance_guarded",
		MemoryScope:    "customer_project_profile_operator",
		EvalSuite:      "health_performance_p0",
	},
}

func (s *ServiceProfiles) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	registryDir := filepath.Join(root, ".ovav", "registry")

	// Load service_profiles.yaml
	profilesPath := filepath.Join(registryDir, "service_profiles.yaml")
	profilesData, err := os.ReadFile(profilesPath)
	if err != nil {
		return Result{ID: s.ID(), Name: s.Name(), Status: "skip", Weight: s.Weight(),
			Message: fmt.Sprintf("SKIP — service_profiles.yaml not found: %v", err), Duration: time.Since(start)}
	}
	var profilesDoc struct {
		ServiceProfiles map[string]map[string]interface{} `yaml:"service_profiles"`
	}
	if err := yaml.Unmarshal(profilesData, &profilesDoc); err != nil {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: "FAIL — service_profiles.yaml parse error",
			Issues:  []string{fmt.Sprintf("YAML error: %v", err)}, Duration: time.Since(start)}
	}

	// Load service_lanes.yaml
	lanesPath := filepath.Join(registryDir, "service_lanes.yaml")
	lanesData, _ := os.ReadFile(lanesPath)
	var lanesDoc struct {
		Lanes map[string]map[string]interface{} `yaml:"lanes"`
	}
	yaml.Unmarshal(lanesData, &lanesDoc)

	// Load squads.yaml
	squadsPath := filepath.Join(registryDir, "squads.yaml")
	squadsData, _ := os.ReadFile(squadsPath)
	var squadsDoc struct {
		Squads map[string]map[string]interface{} `yaml:"squads"`
	}
	yaml.Unmarshal(squadsData, &squadsDoc)

	profiles := profilesDoc.ServiceProfiles

	// Check profile count
	if len(profiles) != len(expectedProfiles) {
		issues = append(issues, fmt.Sprintf("expected %d profiles, found %d", len(expectedProfiles), len(profiles)))
	}

	for profileName, expected := range expectedProfiles {
		profile, ok := profiles[profileName]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing profile entry: %s", profileName))
			continue
		}

		// Check display_name
		if dn, _ := profile["display_name"].(string); dn != expected.DisplayName {
			issues = append(issues, fmt.Sprintf("%s: display_name must be %q", profileName, expected.DisplayName))
		}
		// Check lead_operator
		if lo, _ := profile["lead_operator"].(string); lo != expected.LeadOperator {
			issues = append(issues, fmt.Sprintf("%s: lead_operator must be %q", profileName, expected.LeadOperator))
		}
		// Check lanes
		lanesRaw, _ := profile["lanes"].([]interface{})
		if len(lanesRaw) == 0 {
			issues = append(issues, fmt.Sprintf("%s: must declare lanes", profileName))
		} else {
			for _, lane := range lanesRaw {
				laneName, _ := lane.(string)
				laneEntry, lok := lanesDoc.Lanes[laneName]
				if !lok {
					issues = append(issues, fmt.Sprintf("%s: lane %q missing from service_lanes.yaml", profileName, laneName))
				} else if lp, _ := laneEntry["profile"].(string); lp != profileName {
					issues = append(issues, fmt.Sprintf("%s: lane %q must map back to %s", profileName, laneName, profileName))
				}
			}
		}
		// Check squad
		if sq, _ := profile["squad"].(string); sq != expected.Squad {
			issues = append(issues, fmt.Sprintf("%s: squad must be %q", profileName, expected.Squad))
		}
		sqEntry, sqOk := squadsDoc.Squads[expected.Squad]
		if !sqOk {
			issues = append(issues, fmt.Sprintf("%s: squad %q missing from squads.yaml", profileName, expected.Squad))
		} else if op, _ := sqEntry["owner_profile"].(string); op != profileName {
			issues = append(issues, fmt.Sprintf("%s: squad must be owned by %s", profileName, profileName))
		}
		// Check policy_envelope
		if pe, _ := profile["policy_envelope"].(string); pe != expected.PolicyEnvelope {
			issues = append(issues, fmt.Sprintf("%s: policy_envelope must be %q", profileName, expected.PolicyEnvelope))
		}
		// Check memory_scope
		if ms, _ := profile["memory_scope"].(string); ms != expected.MemoryScope {
			issues = append(issues, fmt.Sprintf("%s: memory_scope must be %q", profileName, expected.MemoryScope))
		}
		// Check eval_suite
		if es, _ := profile["eval_suite"].(string); es != expected.EvalSuite {
			issues = append(issues, fmt.Sprintf("%s: eval_suite must be %q", profileName, expected.EvalSuite))
		}
		// Check customer_visible
		if cv, _ := profile["customer_visible"].(bool); !cv {
			issues = append(issues, fmt.Sprintf("%s: must remain customer_visible", profileName))
		}
		// Check p0
		if p0, _ := profile["p0"].(bool); !p0 {
			issues = append(issues, fmt.Sprintf("%s: must remain p0", profileName))
		}
	}

	_ = strings.TrimSpace // used for future lint checks

	if len(issues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message: fmt.Sprintf("FAIL — %d issue(s)", len(issues)), Issues: issues, Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message: fmt.Sprintf("PASS — %d profiles validated", len(profiles)), Duration: time.Since(start)}
}

var _ Validator = (*ServiceProfiles)(nil)
