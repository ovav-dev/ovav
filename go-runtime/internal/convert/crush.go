package convert

import (
	"fmt"
	"strings"
)

// CrushConverter transforms OVAV canonical agents into Crush .md files.
//
// Verified via SDK: internal/agent/types.gen.go AgentV2Info
//
// Frontmatter fields (VERIFIED):
//
//	id, mode ("subagent"|"primary"|"all"), hidden (bool),
//	permissions (action/resource/effect), description, system,
//	color, steps, model
//
// IMPORTANT: use `id` NOT `name` — name is NOT a valid field in Crush SDK.
// IMPORTANT: `hidden` is a REQUIRED boolean field.
// IMPORTANT: permission format is action/resource/effect (SDK format).
type CrushConverter struct{}

func (c *CrushConverter) FileExtension() string { return ".md" }
func (c *CrushConverter) OutputDir() string     { return "clients/crush/agents" }

// AreasOnly returns false: Crush generates full hierarchy.
func (c *CrushConverter) AreasOnly() bool { return false }

// ConvertArea generates an area agent .md for Crush.
func (c *CrushConverter) ConvertArea(area *Area, leadForArea map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	// Frontmatter — use id (NOT name), hidden=false for areas
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %q\n", area.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: false\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	// Permissions — SDK format: action/resource/effect
	if area.Permission != nil {
		b.WriteString("permissions:\n")
		b.WriteString(fmt.Sprintf("  - action: \"file.edit\"\n"))
		if area.Permission.Edit == "allow" {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"allow\"\n")
		} else {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"deny\"\n")
		}
		if len(area.Permission.Bash) > 0 {
			b.WriteString("  - action: \"bash\"\n")
			b.WriteString("    resource: \"*\"\n")
			// Check if any bash permission is allow
			hasAllow := false
			for _, v := range area.Permission.Bash {
				if v == "allow" {
					hasAllow = true
					break
				}
			}
			if hasAllow {
				b.WriteString("    effect: \"allow\"\n")
			} else {
				b.WriteString("    effect: \"deny\"\n")
			}
		}
	}
	// OVAV instructions
	b.WriteString("instructions:\n")
	b.WriteString("  - \"crush_AGENTS.md\"\n")
	if area.OVAVConnection != nil {
		for _, inst := range area.OVAVConnection.Instructions {
			b.WriteString(fmt.Sprintf("  - %q\n", inst))
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("**Lead:** %s\n", area.Lead))
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("**Color:** %s\n", area.Color))
	}
	if area.Surface != "" {
		b.WriteString(fmt.Sprintf("**Superficie:** %s\n", area.Surface))
	}
	b.WriteString("\n---\n\n")

	// OVAV Connection
	if area.OVAVConnection != nil {
		b.WriteString("## Conexión OVAV (Governor System)\n\n")
		b.WriteString("Este área está cableada al sistema administrador OVAV.\n\n")

		if len(area.OVAVConnection.Skills) > 0 {
			b.WriteString("### Skills cargadas\n\n")
			for _, s := range area.OVAVConnection.Skills {
				b.WriteString(fmt.Sprintf("- `%s`\n", s))
			}
			b.WriteString("\n")
		}

		if len(area.OVAVConnection.CLICommands) > 0 {
			b.WriteString("### Comandos CLI autorizados\n\n")
			b.WriteString("```bash\n")
			b.WriteString(`export OVAV_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"` + "\n\n")
			for _, c := range area.OVAVConnection.CLICommands {
				b.WriteString(fmt.Sprintf("(cd \"$OVAV_ROOT\" && %s)\n", c))
			}
			b.WriteString("```\n\n")
		}

		if len(area.OVAVConnection.Contracts) > 0 {
			b.WriteString("### Contratos OVAV\n\n")
			for _, c := range area.OVAVConnection.Contracts {
				b.WriteString(fmt.Sprintf("- `%s`\n", c))
			}
			b.WriteString("\n")
		}

		if len(area.OVAVConnection.Laws) > 0 {
			b.WriteString("### Leyes OVAV\n\n")
			for _, l := range area.OVAVConnection.Laws {
				b.WriteString(fmt.Sprintf("- `%s`\n", l))
			}
			b.WriteString("\n")
		}
		b.WriteString("---\n\n")
	}

	// Lead intelligence
	leadAreaKey := strings.ReplaceAll(area.ID, "-", "_")
	lead := leadForArea[leadAreaKey]
	if lead != nil {
		if lead.Criteria != "" {
			b.WriteString("## Decision Criteria\n\n")
			b.WriteString(lead.Criteria)
			b.WriteString("\n---\n\n")
		}
		if lead.KnowledgeRules != nil {
			b.WriteString("## Reglas de Conocimiento\n\n")
			b.WriteString(fmt.Sprintf("**Dominio:** %s\n\n", lead.KnowledgeRules.Domain))
			for _, rule := range lead.KnowledgeRules.Rules {
				b.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			b.WriteString("\n---\n\n")
		}
		if lead.ResponseStyle != nil {
			b.WriteString("## Estilo de Respuesta\n\n")
			b.WriteString(fmt.Sprintf("**Formato:** %s | **Máx palabras:** %d\n\n", lead.ResponseStyle.Format, lead.ResponseStyle.MaxWords))
			for _, rule := range lead.ResponseStyle.Rules {
				b.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			b.WriteString("\n---\n\n")
		}
	}

	// Contracts
	b.WriteString("## Contratos de Gobernanza\n\n")
	b.WriteString("- **visual_delivery_contract.yaml** — 50% shorter, result first\n")
	b.WriteString("- **safe_stop_contract.yaml** — PARTIAL/SAFE_STOP/READY_FOR_COMMIT\n")
	b.WriteString("- **context_economy_contract.yaml** — Tiers T0-T5\n\n")

	// Functions
	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HACE)\n\n")
	for i, fn := range area.Functions {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, fn))
	}
	b.WriteString("\n---\n\n")

	// Limitations
	b.WriteString("## Limitaciones Explícitas (LO QUE NO HACE)\n\n")
	for _, lim := range area.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n---\n\n")

	// Hard Stop
	b.WriteString("## Respuesta de Hard Stop\n\n```\n")
	b.WriteString(area.HardStop)
	b.WriteString("\n```\n\n---\n\n")

	// Squad Preview
	if len(area.SquadPreview) > 0 {
		b.WriteString("## Squad Members\n\n")
		b.WriteString("| Miembro | País | Especialidad |\n")
		b.WriteString("|---------|------|-------------|\n")
		for _, m := range area.SquadPreview {
			b.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", m.Name, m.Country, m.Specialty))
		}
		b.WriteString("\n---\n\n")
	}

	// Delegation
	if area.Delegation != "" {
		b.WriteString("## Protocolo de Delegación\n\n")
		b.WriteString(area.Delegation)
		b.WriteString("\n\n")
	}

	// Crush delegation
	b.WriteString("## Sistema de Delegación (OVAV — Crush)\n\n")
	b.WriteString("**Regla absoluta:** Para delegar trabajo a otro agente OVAV, usa el **agent tool** nativo de Crush:\n\n")
	b.WriteString("```\nagent(prompt: \"<detalle del task para el agente destinatario>\")\n```\n\n")
	b.WriteString("**OVAV agent IDs:**\n")
	b.WriteString("- `area-<id>` — agentes de área (visibles en picker)\n")
	b.WriteString("- `lead-<id>` — leads OVAV\n")
	b.WriteString("- `team-<id>` — miembros del squad\n\n")

	// References
	if len(area.References) > 0 {
		b.WriteString("## Referencias Canónicas\n\n")
		for _, ref := range area.References {
			// Format: "Label: path" → "- **Label**: path"
			parts := strings.SplitN(ref, ":", 2)
			if len(parts) == 2 {
				b.WriteString(fmt.Sprintf("- **%s**:%s\n", strings.TrimSpace(parts[0]), parts[1]))
			} else {
				b.WriteString(fmt.Sprintf("- **%s**\n", ref))
			}
		}
		b.WriteString("\n")
	}

	// Governance Wiring
	if len(area.GovernanceWiring) > 0 {
		b.WriteString("## Governance Wiring (DO NOT REMOVE)\n\n")
		b.WriteString("This area es gobernado por los siguientes validators y gates:\n\n")
		for _, gw := range area.GovernanceWiring {
			b.WriteString(fmt.Sprintf("- %s\n", gw))
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — Área %s — Lead: %s*\n", area.Name, area.Lead))
	return []byte(b.String()), nil
}

// ConvertLead generates a lead .md for Crush.
func (c *CrushConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %q\n", lead.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", lead.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: true\n")
	if lead.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", lead.Color))
	}
	if lead.Permission != nil {
		b.WriteString("permissions:\n")
		b.WriteString(fmt.Sprintf("  - action: \"file.edit\"\n"))
		if lead.Permission.Edit == "allow" {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"allow\"\n")
		} else {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"deny\"\n")
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("**Área:** %s\n", lead.DisplayName))
	b.WriteString(fmt.Sprintf("**Origen:** %s\n", lead.Origin))
	if lead.Authority != "" {
		b.WriteString(fmt.Sprintf("**Autoridad:** `%s`\n", lead.Authority))
	}
	b.WriteString("\n---\n\n")

	// Functions
	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HAGO)\n\n")
	for i, fn := range lead.Functions {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, fn))
	}
	b.WriteString("\n---\n\n")

	// Limitations
	b.WriteString("## Limitaciones Explícitas (LO QUE NO HAGO)\n\n")
	for _, lim := range lead.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n---\n\n")

	// Hard Stop
	b.WriteString("## Respuesta de Hard Stop\n\n```\n")
	b.WriteString(lead.HardStop)
	b.WriteString("\n```\n\n---\n\n")

	// Squad
	if len(lead.Squad) > 0 {
		b.WriteString("## Squad\n\n")
		b.WriteString("| Miembro | País | Especialidad |\n")
		b.WriteString("|---------|------|-------------|\n")
		for _, m := range lead.Squad {
			b.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", m.Name, m.Country, m.Specialty))
		}
		b.WriteString("\n---\n\n")
	}

	// Delegation
	if lead.Delegation != "" {
		b.WriteString("## Protocolo de Delegación\n\n")
		b.WriteString(lead.Delegation)
		b.WriteString("\n\n")
	}

	// Crush delegation
	b.WriteString("## Sistema de Delegación (OVAV — Crush)\n\n")
	b.WriteString("**Regla absoluta:** Usa el **agent tool** nativo de Crush:\n\n")
	b.WriteString("```\nagent(prompt: \"<detalle del task para el miembro del squad>\")\n```\n\n")
	b.WriteString("**Team members:** ver tabla Squad Members arriba.\n\n")

	// CRITERIA
	if lead.Criteria != "" {
		b.WriteString("## Decision Criteria\n\n")
		b.WriteString(lead.Criteria)
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — %s, Lead de %s*\n", lead.Name, lead.DisplayName))
	return []byte(b.String()), nil
}

// ConvertTeam generates a team .md for Crush.
func (c *CrushConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %q\n", team.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", team.Function))
	b.WriteString("mode: subagent\n")
	b.WriteString("hidden: true\n")
	if team.Model != "" {
		b.WriteString(fmt.Sprintf("model:\n"))
		b.WriteString(fmt.Sprintf("  id: %q\n", team.Model))
	}
	if team.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", team.Color))
	}
	if team.Steps > 0 {
		b.WriteString(fmt.Sprintf("steps: %d\n", team.Steps))
	}
	if team.Permission != nil {
		b.WriteString("permissions:\n")
		b.WriteString(fmt.Sprintf("  - action: \"file.edit\"\n"))
		if team.Permission.Edit == "allow" {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"allow\"\n")
		} else {
			b.WriteString("    resource: \"*\"\n")
			b.WriteString("    effect: \"deny\"\n")
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, team.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("**País:** %s\n", team.Country))
	b.WriteString(fmt.Sprintf("**Reporta a:** %s\n", team.Lead))
	b.WriteString(fmt.Sprintf("**Área:** %s\n", team.Area))
	b.WriteString("\n")

	b.WriteString("## Función Principal\n\n")
	b.WriteString(team.Function)
	b.WriteString("\n\n")

	b.WriteString("## Acciones Autorizadas\n\n")
	for i, action := range team.Actions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
	}
	b.WriteString("\n")

	b.WriteString("## Hard Stop\n\n")
	b.WriteString(fmt.Sprintf("%q\n\n", team.HardStop))

	if team.Response != "" {
		b.WriteString("## Respuesta Fuera de Alcance\n\n")
		b.WriteString("```\n")
		b.WriteString(team.Response)
		b.WriteString("\n```\n")
	}

	if team.ResponseStyle != nil {
		b.WriteString("\n## Estilo de Respuesta\n\n")
		b.WriteString(fmt.Sprintf("**Formato:** %s | **Máx palabras:** %d\n\n", team.ResponseStyle.Format, team.ResponseStyle.MaxWords))
		for _, rule := range team.ResponseStyle.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
	}

	if team.KnowledgeRules != nil {
		b.WriteString("\n## Reglas de Conocimiento\n\n")
		b.WriteString(fmt.Sprintf("**Dominio:** %s\n\n", team.KnowledgeRules.Domain))
		for _, rule := range team.KnowledgeRules.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
	}

	b.WriteString("\n---\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — %s, %s*\n", team.Name, team.Function))
	b.WriteString(fmt.Sprintf("*Reporta a: %s · Área: %s*\n", team.Lead, team.Area))
	return []byte(b.String()), nil
}
