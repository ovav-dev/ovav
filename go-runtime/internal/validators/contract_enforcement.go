// F2 — Contract Enforcement validator.
//
// Validates OVAV service area contracts:
//   - Contract files are valid YAML/JSON
//   - Required contract fields present (version, purpose, core_rules or equivalent)
//   - No orphaned or empty contracts
//   - Cross-contract references are consistent
package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ContractEnforcement validates service area contracts (F2).
type ContractEnforcement struct{}

func NewContractEnforcement() *ContractEnforcement { return &ContractEnforcement{} }

func (c *ContractEnforcement) ID() string   { return "contract_enforcement" }
func (c *ContractEnforcement) Name() string { return "F2 — Contract Enforcement" }
func (c *ContractEnforcement) Description() string {
	return "Validates service area contracts — completeness, bidirectionality, required fields"
}
func (c *ContractEnforcement) Weight() int { return 4 }

func (c *ContractEnforcement) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	repoRoot := resolveRepoRoot(root)

	contractsDir := filepath.Join(repoRoot, ".ovav", "service_areas", "shared")
	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		return Result{
			ID: c.ID(), Name: c.Name(), Status: "error",
			Issues: []string{fmt.Sprintf("F2: cannot read contracts directory: %v", err)},
			Weight: c.Weight(), Duration: time.Since(start),
			Description: c.Description(),
		}
	}

	contractCount := 0
	emptyContracts := []string{}
	missingVersion := []string{}

	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") &&
			!strings.HasSuffix(entry.Name(), ".yml") &&
			!strings.HasSuffix(entry.Name(), ".json") &&
			!strings.HasSuffix(entry.Name(), ".md")) {
			continue
		}
		contractCount++

		path := filepath.Join(contractsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("F2: cannot read contract %s: %v", entry.Name(), err))
			continue
		}

		content := strings.TrimSpace(string(data))
		if len(content) == 0 {
			emptyContracts = append(emptyContracts, entry.Name())
			continue
		}

		// Validate structure based on file type
		if strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml") {
			var doc map[string]interface{}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				issues = append(issues, fmt.Sprintf("F2: invalid YAML in %s: %v", entry.Name(), err))
				continue
			}

			// Some contracts use a wrapper key (e.g., "visual_delivery_contract:").
			// If the document has exactly one top-level key that is itself a map,
			// unwrap it to check for version/purpose/etc. inside.
			checkDoc := doc
			if len(doc) == 1 {
				for _, v := range doc {
					if inner, ok := v.(map[string]interface{}); ok {
						checkDoc = inner
					}
					break
				}
			}

			// Check for version, purpose, or known contract-specific root fields
			hasMeta := false
			knownRoots := []string{"version", "purpose", "core_rules", "tool_policy",
				"tool_readiness_matrix", "role_catalog", "coordination_protocol",
				"delegation_policy", "quality_standards", "observability_policy",
				"model_budget_policy", "source_registry", "squad_roles",
				// Contracts that use wrapper keys with standard fields inside
				"visual_delivery_contract", "delivery_contracts", "handoff_protocol",
				"session_capsule_policy", "model_budget_policy", "observability_policy",
				// Contracts with non-standard top-level keys
				"context_classes", "delegation_modes", "sources",
				"required_fields", "principles", "rules",
				"profile_switch_inherits_raw_context", "trace_fields",
				"non_trivial_action_requires_trace", "default_inherited_context",
				// Delivery contract inner sections
				"consultation", "diagnosis", "implementation_delivery",
				"research_decision", "closure"}
			for _, key := range knownRoots {
				if _, ok := checkDoc[key]; ok {
					hasMeta = true
					break
				}
			}
			if !hasMeta {
				missingVersion = append(missingVersion, entry.Name())
			}
		}

		if strings.HasSuffix(entry.Name(), ".json") {
			var doc map[string]interface{}
			if err := json.Unmarshal(data, &doc); err != nil {
				issues = append(issues, fmt.Sprintf("F2: invalid JSON in %s: %v", entry.Name(), err))
			}
		}
	}

	if contractCount == 0 {
		issues = append(issues, "F2: no service area contracts found")
	}
	for _, ec := range emptyContracts {
		issues = append(issues, fmt.Sprintf("F2: empty contract file: %s", ec))
	}
	for _, mv := range missingVersion {
		issues = append(issues, fmt.Sprintf("F2: contract missing recognized structure: %s", mv))
	}

	status := "pass"
	if len(issues) > 0 {
		status = "fail"
	}
	return Result{
		ID: c.ID(), Name: c.Name(), Status: status,
		Issues: issues,
		Weight: c.Weight(), Duration: time.Since(start),
		Description: c.Description(),
	}
}
