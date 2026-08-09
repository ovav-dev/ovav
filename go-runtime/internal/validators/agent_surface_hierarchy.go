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

// AgentSurfaceHierarchy validates agent surface hierarchy (TAB areas, @ leads, squads).
// Replaces: check_agent_surface_hierarchy.py
type AgentSurfaceHierarchy struct{}

func NewAgentSurfaceHierarchy() *AgentSurfaceHierarchy { return &AgentSurfaceHierarchy{} }

func (a *AgentSurfaceHierarchy) ID() string   { return "agent_surface_hierarchy" }
func (a *AgentSurfaceHierarchy) Name() string { return "Agent Surface Hierarchy" }
func (a *AgentSurfaceHierarchy) Description() string {
	return "Validates agent surface hierarchy: TAB areas, @ leads, and hidden squads"
}
func (a *AgentSurfaceHierarchy) Weight() int { return 10 }

// Expected area and lead names.
var areaNames = map[string]bool{
	"Platform Engineering":     true,
	"Research Intelligence":    true,
	"Commercial Growth":        true,
	"Digital Product":          true,
	"Education Career":         true,
	"Health Performance":       true,
	"Devops Infrastructure":    true,
	"Ux Design":                true,
	"Legal Compliance":         true,
	"Adversarial Intelligence": true,
}

var leadNames = map[string]bool{
	"Thavren": true, "Eidren": true, "Sofia": true, "Dante": true,
	"Renata": true, "Valeria": true, "Uriel": true, "Elena": true,
	"Camila": true, "Kenji": true,
}

const governorName = "OVAV"

func (a *AgentSurfaceHierarchy) parseFrontmatter(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return nil
	}
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil
	}
	return fm
}

func (a *AgentSurfaceHierarchy) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	agentsDir := filepath.Join(root, ".opencode", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil || len(entries) == 0 {
		issues = append(issues, "No agent files found in .opencode/agents/")
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  "FAIL agent surface hierarchy — no agents found",
			Issues:   issues,
			Duration: time.Since(start),
		}
	}

	type agentCfg struct {
		file        string
		mode        string
		hidden      interface{}
		description string
	}
	agents := make(map[string]agentCfg)

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fullPath := filepath.Join(agentsDir, e.Name())
		fm := a.parseFrontmatter(fullPath)
		if fm == nil {
			issues = append(issues, fmt.Sprintf("Cannot parse frontmatter: %s", e.Name()))
			continue
		}

		name, ok := fm["name"].(string)
		if !ok || name == "" {
			issues = append(issues, fmt.Sprintf("Missing 'name' in frontmatter: %s", e.Name()))
			continue
		}

		mode, _ := fm["mode"].(string)
		desc, _ := fm["description"].(string)
		hidden := fm["hidden"]

		agents[name] = agentCfg{
			file:        e.Name(),
			mode:        mode,
			hidden:      hidden,
			description: desc,
		}
	}

	// Classify agents
	var areaAgents, atAgents, squadAgents []string
	for name, cfg := range agents {
		isHidden := false
		switch v := cfg.hidden.(type) {
		case bool:
			isHidden = v
		}

		if isHidden {
			if cfg.mode == "subagent" || cfg.mode == "primary" || cfg.mode == "lead" {
				squadAgents = append(squadAgents, name)
			} else {
				issues = append(issues, fmt.Sprintf("[%s] %s: hidden:true with mode:%s — must be mode:primary, mode:subagent, or mode:lead", cfg.file, name, cfg.mode))
			}
			continue
		}

		// Not hidden
		switch cfg.mode {
		case "all", "primary":
			areaAgents = append(areaAgents, name)
		case "subagent":
			atAgents = append(atAgents, name)
		default:
			issues = append(issues, fmt.Sprintf("[%s] %s: hidden:false with mode:%s — must be mode:primary, mode:all or mode:subagent", cfg.file, name, cfg.mode))
		}
	}

	// Validate area agents
	for _, name := range areaAgents {
		cfg := agents[name]
		if !areaNames[name] {
			issues = append(issues, fmt.Sprintf("[%s] %s: configured as area (mode:all) but not in known areas list", cfg.file, name))
		}
		if !strings.Contains(cfg.description, "◆") {
			issues = append(issues, fmt.Sprintf("[%s] %s: area must have ◆ in description", cfg.file, name))
		}
	}

	// Validate @ agents (leads + governor)
	for _, name := range atAgents {
		cfg := agents[name]
		if name == governorName {
			if !strings.Contains(cfg.description, "Governor") {
				issues = append(issues, fmt.Sprintf("[%s] %s: governor must include 'Governor' in description", cfg.file, name))
			}
			continue
		}
		if !leadNames[name] {
			issues = append(issues, fmt.Sprintf("[%s] %s: visible in @ but not a lead. Only leads (✦) should be visible.", cfg.file, name))
		}
		if !strings.Contains(cfg.description, "✦") {
			issues = append(issues, fmt.Sprintf("[%s] %s: lead in @ must have ✦ in description", cfg.file, name))
		}
	}

	// Validate squad agents — ensure they're not areas or leads
	for _, name := range squadAgents {
		cfg := agents[name]
		if areaNames[name] && cfg.mode == "subagent" {
			issues = append(issues, fmt.Sprintf("[%s] %s: is an area but configured as squad (hidden:true)", agents[name].file, name))
		}
		if leadNames[name] && cfg.mode == "subagent" && !strings.HasPrefix(cfg.file, "lead-") {
			// Only flag non-lead files that share lead names
		}
	}

	// Check opencode.json consistency
	ocPath := filepath.Join(root, "opencode.json")
	if data, err := os.ReadFile(ocPath); err == nil {
		var config map[string]interface{}
		if json.Unmarshal(data, &config) == nil {
			// Check no OVAV agents defined in opencode.json
			if ojAgents, ok := config["agent"].(map[string]interface{}); ok {
				// GOV-006: Host builtin subagent types. These are MiMoCode-native agent types
				// that exist OUTSIDE the OVAV governance system. When spawned directly via
				// actor("explore") or actor("general"), they bypass:
				//   - OVAV identity & area boundaries (LAW-001)
				//   - Hard stop rules from permission_authority.json
				//   - Delegation Router audit trail
				// PREFER ovav_squad_routing in model_routing.json which maps task types
				// to OVAV squad members (Irene, Helena, Pablo, Diana, etc.)
				builtins := map[string]bool{"build": true, "plan": true, "general": true, "explore": true, "task": true}
				for k := range ojAgents {
					if !builtins[k] && (areaNames[k] || leadNames[k] || k == governorName) {
						issues = append(issues, fmt.Sprintf("OVAV agent '%s' defined in opencode.json — must be in .opencode/agents/*.md only", k))
					}
				}
			}
			// Check default_agent
			if def, ok := config["default_agent"].(string); ok && def != "" {
				if !areaNames[def] {
					issues = append(issues, fmt.Sprintf("default_agent '%s' is not a valid area", def))
				}
			}
		}
	}

	// GOV-006-C: Check model_routing.json for generic subagent overrides that bypass OVAV governance
	mrPath := filepath.Join(root, ".mimocode", "model_routing.json")
	if data, err := os.ReadFile(mrPath); err == nil {
		var mrConfig map[string]interface{}
		if json.Unmarshal(data, &mrConfig) == nil {
			if overrides, ok := mrConfig["subagent_model_override"].(map[string]interface{}); ok {
				genericKeys := map[string]bool{"explore": true, "general": true, "general_coding": true, "general_review": true, "task": true, "build": true, "plan": true}
				genericFound := []string{}
				for k := range overrides {
					if k == "description" || k == "_comment" {
						continue
					}
					if genericKeys[k] || (!strings.HasPrefix(k, "_") && !strings.Contains(k, "DISABLED")) {
						genericFound = append(genericFound, k)
					}
				}
				if len(genericFound) > 0 {
					issues = append(issues, fmt.Sprintf("GOV-006-C: model_routing.json subagent_model_override contains generic keys %v — these bypass OVAV squad identity and LAW-001 boundaries. Use ovav_squad_routing instead.", genericFound))
				}
			}
			// Verify ovav_squad_routing exists
			if _, ok := mrConfig["ovav_squad_routing"]; !ok {
				issues = append(issues, "GOV-006-C: model_routing.json missing ovav_squad_routing — OVAV squad delegation router not configured.")
			}
		}
	}

	// Cardinality checks
	missingAreas := []string{}
	for name := range areaNames {
		found := false
		for _, a := range areaAgents {
			if a == name {
				found = true
				break
			}
		}
		if !found {
			missingAreas = append(missingAreas, name)
		}
	}
	if len(missingAreas) > 0 {
		issues = append(issues, fmt.Sprintf("Areas faltantes en TAB: %v — deben tener mode:all + hidden:false + ◆", missingAreas))
	}

	if len(issues) > 0 {
		return Result{
			ID: a.ID(), Name: a.Name(), Status: "fail", Weight: a.Weight(),
			Message:  fmt.Sprintf("FAIL agent surface hierarchy — %d issue(s). Areas=%d Leads=@%d Squads=%d", len(issues), len(areaAgents), len(atAgents), len(squadAgents)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: a.ID(), Name: a.Name(), Status: "pass", Weight: a.Weight(),
		Message:  fmt.Sprintf("PASS agent surface hierarchy — Areas=%d Leads=@%d Squads=%d", len(areaAgents), len(atAgents), len(squadAgents)),
		Duration: time.Since(start),
	}
}

var _ Validator = (*AgentSurfaceHierarchy)(nil)
