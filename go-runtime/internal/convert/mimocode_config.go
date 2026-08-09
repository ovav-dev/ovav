package convert

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MimocodeConfigConverter generates a JSON config for MiMoCode's config.json.
// MiMoCode loads agents from config.json (not .md files).
// Format: { "$schema": "...", "agent": { "name": { prompt, description, mode, hidden, permission } } }
type MimocodeConfigConverter struct {
	AreasOnly bool // true = only areas, false = areas + leads + teams
}

func (c *MimocodeConfigConverter) FileExtension() string { return ".json" }
func (c *MimocodeConfigConverter) OutputDir() string     { return "runtimes/mimocode" }
func (c *MimocodeConfigConverter) IsAreasOnly() bool     { return c.AreasOnly }

// ConfigOutput is the top-level config.json structure
type ConfigOutput struct {
	Schema string               `json:"$schema"`
	Agent  map[string]*AgentDef `json:"agent"`
}

// AgentDef is a single agent entry in config.json
type AgentDef struct {
	Prompt      string `json:"prompt,omitempty"`
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Hidden      *bool  `json:"hidden,omitempty"`
	Color       string `json:"color,omitempty"`
}

// ConvertArea generates config.json entry for an area agent
func (c *MimocodeConfigConverter) ConvertArea(area *Area) (*AgentDef, error) {
	hidden := false
	prompt := buildAreaPrompt(area)

	return &AgentDef{
		Prompt:      prompt,
		Description: fmt.Sprintf("◆ %s", area.Description),
		Mode:        "primary",
		Hidden:      &hidden,
		Color:       area.Color,
	}, nil
}

// ConvertLead generates config.json entry for a lead agent
func (c *MimocodeConfigConverter) ConvertLead(lead *Lead) (*AgentDef, error) {
	hidden := true
	prompt := buildLeadPrompt(lead)

	return &AgentDef{
		Prompt:      prompt,
		Description: fmt.Sprintf("✦ %s", lead.Description),
		Mode:        "primary",
		Hidden:      &hidden,
		Color:       lead.Color,
	}, nil
}

// ConvertTeam generates config.json entry for a team agent
func (c *MimocodeConfigConverter) ConvertTeam(team *TeamMember) (*AgentDef, error) {
	hidden := true
	prompt := buildTeamPrompt(team)

	return &AgentDef{
		Prompt:      prompt,
		Description: team.Function,
		Mode:        "subagent",
		Hidden:      &hidden,
		Color:       team.Color,
	}, nil
}

// buildAreaPrompt constructs the system prompt for an area agent
func buildAreaPrompt(area *Area) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Lead:** %s\n", area.Lead))
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("**Color:** %s\n", area.Color))
	}
	if area.Surface != "" {
		b.WriteString(fmt.Sprintf("**Superficie:** %s\n", area.Surface))
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Contratos de Gobernanza\n\n")
	b.WriteString("- **visual_delivery_contract.yaml** — Entrega visual: 50% shorter, result first\n")
	b.WriteString("- **safe_stop_contract.yaml** — Safe Stop Report: PARTIAL/SAFE_STOP/READY_FOR_COMMIT\n")
	b.WriteString("- **context_economy_contract.yaml** — Tiers T0-T5, must not load repo/internal context by default\n\n")
	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HACE)\n\n")
	for i, fn := range area.Functions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, fn))
	}
	b.WriteString("\n## Limitaciones Explícitas (LO QUE NO HACE)\n\n")
	for _, lim := range area.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(area.HardStop)
	b.WriteString("\n\n")
	if len(area.SquadPreview) > 0 {
		b.WriteString("## Squad Members\n\n")
		for _, m := range area.SquadPreview {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — %s\n", m.Name, m.Country, m.Specialty))
		}
	}
	return b.String()
}

// buildLeadPrompt constructs the system prompt for a lead agent
func buildLeadPrompt(lead *Lead) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Área:** %s\n", lead.DisplayName))
	b.WriteString(fmt.Sprintf("**Origen:** %s\n", lead.Origin))
	if lead.Authority != "" {
		b.WriteString(fmt.Sprintf("**Autoridad:** `%s`\n", lead.Authority))
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HAGO)\n\n")
	for i, fn := range lead.Functions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, fn))
	}
	b.WriteString("\n## Limitaciones Explícitas (LO QUE NO HAGO)\n\n")
	for _, lim := range lead.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(lead.HardStop)
	if len(lead.Squad) > 0 {
		b.WriteString("\n\n## Squad\n\n")
		for _, m := range lead.Squad {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — %s\n", m.Name, m.Country, m.Specialty))
		}
	}
	return b.String()
}

// buildTeamPrompt constructs the system prompt for a team agent
func buildTeamPrompt(team *TeamMember) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**País:** %s\n", team.Country))
	b.WriteString(fmt.Sprintf("**Reporta a:** %s\n", team.Lead))
	b.WriteString(fmt.Sprintf("**Área:** %s\n", team.Area))
	b.WriteString("\n---\n\n")
	b.WriteString("## Función Principal\n\n")
	b.WriteString(team.Function)
	b.WriteString("\n\n## Acciones Autorizadas\n\n")
	for i, action := range team.Actions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(team.HardStop)
	return b.String()
}

// GenerateConfigJSON creates a full config.json from areas, leads, and teams
func (c *MimocodeConfigConverter) GenerateConfigJSON(areas []*Area, leads []*Lead, teams []*TeamMember) ([]byte, error) {
	cfg := &ConfigOutput{
		Schema: "https://mimocode.ai/config.json",
		Agent:  make(map[string]*AgentDef),
	}

	// Add areas
	for _, area := range areas {
		agent, err := c.ConvertArea(area)
		if err != nil {
			return nil, fmt.Errorf("convert area %s: %w", area.Name, err)
		}
		cfg.Agent[area.Name] = agent
	}

	// Add leads (unless AreasOnly)
	if !c.IsAreasOnly() {
		for _, lead := range leads {
			agent, err := c.ConvertLead(lead)
			if err != nil {
				return nil, fmt.Errorf("convert lead %s: %w", lead.Name, err)
			}
			cfg.Agent[lead.Name] = agent
		}
	}

	// Add teams (unless AreasOnly)
	if !c.IsAreasOnly() {
		for _, team := range teams {
			agent, err := c.ConvertTeam(team)
			if err != nil {
				return nil, fmt.Errorf("convert team %s: %w", team.Name, err)
			}
			cfg.Agent[team.Name] = agent
		}
	}

	return json.MarshalIndent(cfg, "", "  ")
}
