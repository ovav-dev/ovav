// Package fde implements the Forward Deployed Engineer Brain system.
//
// Each OVAV lead has a "brain" stored in .ovav/service_areas/{area}/{lead}/
// consisting of SELF_MODEL, CRITERIA, EVOLUTION, OPERATING_LEVEL, and
// OVAV_RELATIONSHIP YAML files. This package loads and materializes
// those brains so each lead receives their identity, decision criteria,
// growth history, and cross-area relationships at session start.
package fde

// ── Brain Types ─────────────────────────────────────────────────────────────

// SelfModelCapacity is a single competency with proficiency score.
type SelfModelCapacity struct {
	Name        string  `yaml:"name"`
	Proficiency float64 `yaml:"proficiency"`
}

// SelfModelCompetency groups capacities by domain.
type SelfModelCompetency struct {
	Capacities []SelfModelCapacity `yaml:"capacities,omitempty"`
	// Alternative key used by Thavren's model.
	Items []struct {
		Name        string  `yaml:"name"`
		Proficiency float64 `yaml:"proficiency"`
	} `yaml:"items,omitempty"`
}

// SelfModelPartner is a cross-area collaboration partner.
type SelfModelPartner struct {
	Partner     string `yaml:"partner"`
	Type        string `yaml:"type"`
	Protocol    string `yaml:"protocol"`
	Description string `yaml:"description"`
}

// SelfModel represents SELF_MODEL.yaml (or {LEAD}_SELF_MODEL.yaml).
type SelfModel struct {
	Version       string                 `yaml:"version"`
	Operator      string                 `yaml:"operator"`
	ServiceArea   string                 `yaml:"service_area"`
	Competencies  map[string]interface{} `yaml:"core_competencies,omitempty"`
	Collaboration []SelfModelPartner     `yaml:"collaboration"`
	Knowledge     []string               `yaml:"knowledge_domains"`
	ModelPrefs    map[string]string      `yaml:"model_preferences"`
	BlindSpots    []SelfModelBlindSpot   `yaml:"blind_spots,omitempty"`
	Metrics       map[string]interface{} `yaml:"metrics,omitempty"`
	RealContribs  map[string]interface{} `yaml:"real_contributions,omitempty"`
	OVAVSystems   []string               `yaml:"ovav_systems_not_mine,omitempty"`
}

// SelfModelBlindSpot is a known limitation.
type SelfModelBlindSpot struct {
	Area        string `yaml:"area"`
	Description string `yaml:"description"`
	Severity    string `yaml:"severity"`
	Discovered  string `yaml:"discovered"`
	Mitigation  string `yaml:"mitigation"`
}

// Criterion is a single decision criterion.
// Supports both CRITERIA.yaml format (entries with criterion/id/confidence/domain)
// and simpler formats with name/rule.
type Criterion struct {
	ID         string  `yaml:"id"`
	Criterion  string  `yaml:"criterion"`
	Name       string  `yaml:"name,omitempty"`
	Rule       string  `yaml:"rule,omitempty"`
	Domain     string  `yaml:"domain"`
	Confidence float64 `yaml:"confidence"`
	Status     string  `yaml:"status"`
	FirstSeen  string  `yaml:"first_observed"`
}

// Criteria represents CRITERIA.yaml.
type Criteria struct {
	Version     string      `yaml:"version"`
	Operator    string      `yaml:"operator"`
	ServiceArea string      `yaml:"service_area"`
	Entries     []Criterion `yaml:"entries"`
	Items       []Criterion `yaml:"decision_criteria,omitempty"`
}

// All returns all criterion entries, preferring Entries over Items.
func (c *Criteria) All() []Criterion {
	if len(c.Entries) > 0 {
		return c.Entries
	}
	return c.Items
}

// Count returns the number of criteria available.
func (c *Criteria) Count() int {
	return len(c.All())
}

// EvolutionEntry is a single growth event.
type EvolutionEntry struct {
	Date         string `yaml:"date"`
	SessionFocus string `yaml:"session_focus"`
	Summary      string `yaml:"summary"`
	Event        string `yaml:"event,omitempty"`
	Description  string `yaml:"description,omitempty"`
}

// Evolution represents EVOLUTION.yaml.
type Evolution struct {
	Version     string           `yaml:"version"`
	Created     string           `yaml:"created"`
	Updated     string           `yaml:"updated"`
	Operator    string           `yaml:"operator,omitempty"`
	Sessions    []EvolutionEntry `yaml:"sessions"`
	History     []EvolutionEntry `yaml:"history,omitempty"`
	GrowthAreas []string         `yaml:"growth_areas,omitempty"`
	Lessons     []string         `yaml:"lessons_learned,omitempty"`
}

// AllSessions returns all evolution entries, preferring Sessions over History.
func (ev *Evolution) AllSessions() []EvolutionEntry {
	if len(ev.Sessions) > 0 {
		return ev.Sessions
	}
	return ev.History
}

// OperatingLevel represents OPERATING_LEVEL.yaml.
// Supports two formats:
//  1. foundational_law: {version, established, law: {statement}, ...} (Thavren)
//  2. operating_level: {version, operator, level, capabilities, ...} (Kenji, Camila)
type OperatingLevel struct {
	Version      string                 `yaml:"version"`
	Established  string                 `yaml:"established"`
	Operator     string                 `yaml:"operator,omitempty"`
	Level        string                 `yaml:"level,omitempty"`
	Declared     string                 `yaml:"declared,omitempty"`
	Description  string                 `yaml:"description,omitempty"`
	Law          map[string]interface{} `yaml:"law,omitempty"`
	Capabilities map[string]interface{} `yaml:"capabilities,omitempty"`
	Limitations  []string               `yaml:"limitations,omitempty"`
	Contracts    []string               `yaml:"contracts,omitempty"`
}

// Statement returns the law statement if available, otherwise description.
func (ol *OperatingLevel) Statement() string {
	if ol.Law != nil {
		if s, ok := ol.Law["statement"].(string); ok {
			return s
		}
	}
	return ol.Description
}

// RelationshipPoint is a single integration point.
type RelationshipPoint struct {
	Point       string   `yaml:"point"`
	Partner     string   `yaml:"partner,omitempty"`
	Partners    []string `yaml:"partners,omitempty"`
	Description string   `yaml:"description"`
}

// Relationship represents OVAV_RELATIONSHIP.yaml.
type Relationship struct {
	Version           string              `yaml:"version"`
	Established       string              `yaml:"established,omitempty"`
	Operator          string              `yaml:"operator,omitempty"`
	Type              string              `yaml:"relationship_type,omitempty"`
	PrimaryGovernor   string              `yaml:"primary_governor,omitempty"`
	IntegrationPoints []RelationshipPoint `yaml:"integration_points"`
	Constraints       []string            `yaml:"operational_constraints,omitempty"`
	Parties           []map[string]string `yaml:"parties,omitempty"`
	Principles        []string            `yaml:"principles,omitempty"`
}

// ── Brain Pack ──────────────────────────────────────────────────────────────

// BrainPack is the complete FDE brain loaded from service area YAMLs.
type BrainPack struct {
	Lead       string          `json:"lead"`
	Area       string          `json:"area"`
	SelfModel  *SelfModel      `json:"self_model"`
	Criteria   *Criteria       `json:"criteria"`
	Evolution  *Evolution      `json:"evolution"`
	OpLevel    *OperatingLevel `json:"operating_level"`
	Rel        *Relationship   `json:"relationship"`
	LoadedFrom string          `json:"loaded_from"`
}
