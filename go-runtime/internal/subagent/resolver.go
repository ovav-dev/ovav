// Package subagent resolves OVAV subagent invocations.
//
// Reads .ovav/registry/subagent_catalog.yaml and provides:
//   - Exact match by id (team-elena-frontend)
//   - Alias match (frontend-elena → team-elena-frontend)
//   - Keyword fuzzy match (react → team-elena-frontend)
//   - Ambiguity detection (elena → lead-elena OR team-elena-frontend)
//
// Used by:
//   - cmd/resolve_subagent/main.go (CLI entry point)
//   - tools/validators/check_cross_area_coherence.py (validator, mirrors logic)
package subagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent is the runtime representation of a catalog entry.
type Agent struct {
	ID                string   `yaml:"id" json:"id"`
	Kind              string   `yaml:"kind" json:"kind"` // "lead" or "team"
	Name              string   `yaml:"name" json:"name"`
	Area              string   `yaml:"area" json:"area"`
	Lead              *string  `yaml:"lead" json:"lead,omitempty"`
	Function          string   `yaml:"function" json:"function"`
	Aliases           []string `yaml:"aliases" json:"aliases"`
	Keywords          []string `yaml:"keywords" json:"keywords"`
	DisambiguatesFrom []string `yaml:"disambiguates_from" json:"disambiguates_from"`
	Note              string   `yaml:"note" json:"note,omitempty"`
	FilePath          string   `yaml:"file_path" json:"file_path"`
}

// Catalog is the loaded subagent catalog.
type Catalog struct {
	Version         string                 `yaml:"catalog_version" json:"catalog_version"`
	GeneratedBy     string                 `yaml:"generated_by" json:"generated_by"`
	GeneratedAt     string                 `yaml:"generated_at" json:"generated_at"`
	Agents          []Agent                `yaml:"agents" json:"agents"`
	ResolutionRules ResolutionRules        `yaml:"resolution_rules" json:"resolution_rules"`
	Governance      map[string]interface{} `yaml:"governance" json:"governance"`
}

// ResolutionRules defines how alias/intent resolution works.
type ResolutionRules struct {
	StrictMode              bool                `yaml:"strict_mode" json:"strict_mode"`
	AmbiguityStrategy       string              `yaml:"ambiguity_strategy" json:"ambiguity_strategy"`
	AliasesResolved         map[string][]string `yaml:"aliases_resolved" json:"aliases_resolved"`
	DisambiguationQuestions []struct {
		Match    []string `yaml:"match" json:"match"`
		Question string   `yaml:"question" json:"question"`
	} `yaml:"disambiguation_questions" json:"disambiguation_questions"`
}

// Resolution is the result of a resolver query.
type Resolution struct {
	Input        string   `json:"input"`
	ExactMatches []Agent  `json:"exact_matches"` // id matches
	AliasMatches []Agent  `json:"alias_matches"` // alias matches
	Ambiguous    bool     `json:"ambiguous"`
	AmbiguousIDs []string `json:"ambiguous_ids,omitempty"`
	Suggestion   string   `json:"suggestion,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// LoadCatalog reads .ovav/registry/subagent_catalog.yaml.
func LoadCatalog(repoRoot string) (*Catalog, error) {
	catalogPath := filepath.Join(repoRoot, ".ovav", "registry", "subagent_catalog.yaml")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, fmt.Errorf("subagent: cannot read catalog at %s: %w", catalogPath, err)
	}

	var c Catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("subagent: invalid YAML: %w", err)
	}

	if c.Version == "" {
		return nil, fmt.Errorf("subagent: catalog missing version")
	}

	return &c, nil
}

// Resolve takes a string input (id, alias, or intent keyword) and returns the resolution.
func (c *Catalog) Resolve(input string) Resolution {
	input = strings.TrimSpace(strings.ToLower(input))
	res := Resolution{Input: input}

	if input == "" {
		res.Error = "empty input"
		return res
	}

	// 1. Exact id match
	var exactMatches []Agent
	for _, a := range c.Agents {
		if strings.EqualFold(a.ID, input) {
			exactMatches = append(exactMatches, a)
		}
	}
	res.ExactMatches = exactMatches

	// 2. Alias match (from catalog entries AND from resolution_rules.aliases_resolved)
	aliasMatches := make(map[string]Agent)
	for alias, ids := range c.ResolutionRules.AliasesResolved {
		if strings.EqualFold(alias, input) {
			for _, id := range ids {
				if a := c.findByID(id); a != nil {
					aliasMatches[id] = *a
				}
			}
		}
	}

	// Also check per-agent aliases
	for _, a := range c.Agents {
		for _, alias := range a.Aliases {
			if strings.EqualFold(alias, input) {
				aliasMatches[a.ID] = a
			}
		}
	}

	for _, a := range aliasMatches {
		res.AliasMatches = append(res.AliasMatches, a)
	}

	// 3. Determine if ambiguous
	allMatches := append([]Agent{}, exactMatches...)
	allMatches = append(allMatches, res.AliasMatches...)

	// Deduplicate
	seen := make(map[string]bool)
	unique := []Agent{}
	for _, a := range allMatches {
		if !seen[a.ID] {
			seen[a.ID] = true
			unique = append(unique, a)
		}
	}

	if len(unique) == 0 {
		res.Error = fmt.Sprintf("no agent matches '%s'", input)
		return res
	}

	if len(unique) > 1 {
		res.Ambiguous = true
		ids := make([]string, 0, len(unique))
		for _, a := range unique {
			ids = append(ids, a.ID)
		}
		res.AmbiguousIDs = ids
		res.Suggestion = c.disambiguationHint(ids)
		return res
	}

	return res
}

func (c *Catalog) findByID(id string) *Agent {
	for i, a := range c.Agents {
		if a.ID == id {
			return &c.Agents[i]
		}
	}
	return nil
}

func (c *Catalog) disambiguationHint(ids []string) string {
	for _, q := range c.ResolutionRules.DisambiguationQuestions {
		if sameSet(q.Match, ids) {
			return q.Question
		}
	}
	return fmt.Sprintf("'%s' matches multiple agents: %s. Please disambiguate.", strings.Join(ids, ", "), strings.Join(ids, ", "))
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool)
	for _, x := range a {
		seen[x] = true
	}
	for _, x := range b {
		if !seen[x] {
			return false
		}
	}
	return true
}

// MustGet returns the single agent matching id, or panics if not found / ambiguous.
func (c *Catalog) MustGet(id string) Agent {
	r := c.Resolve(id)
	if r.Error != "" {
		panic(fmt.Sprintf("subagent.MustGet: %s", r.Error))
	}
	if r.Ambiguous {
		panic(fmt.Sprintf("subagent.MustGet: ambiguous id '%s' → %v", id, r.AmbiguousIDs))
	}
	if len(r.ExactMatches) > 0 {
		return r.ExactMatches[0]
	}
	if len(r.AliasMatches) > 0 {
		return r.AliasMatches[0]
	}
	panic("subagent.MustGet: unreachable")
}
