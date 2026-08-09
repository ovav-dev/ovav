package convert

import (
	"fmt"
	"strings"
)

// MimocodeConverter transforms OVAV canonical agents into Mimocode .md files.
//
// Mimocode is an OpenCode fork developed by XIOAMI with a free model tier.
// Agent format mirrors OpenCode: .md files with YAML frontmatter (mode, hidden, color).
//
// Conversion rules:
//
//	Areas → mode:primary, hidden:false → visible in TAB selector
//	Leads → mode:primary, hidden:true  → hidden from TAB, invocable by name
//	Teams → mode:subagent, hidden:true → only via Task tool or @ mention
type MimocodeConverter struct{}

func (c *MimocodeConverter) FileExtension() string { return ".md" }
func (c *MimocodeConverter) OutputDir() string     { return "runtimes/mimocode/agents" }

// AreasOnly returns true: mimocode exposes agents through a TAB picker.
// Its fork of OpenCode does not honor the `hidden: true` frontmatter field
// (different TUI), so it leaks every agent file into the picker. To prevent
// internal leads/team members from being exposed to end users, the converter
// only publishes area-level agents and cleans up lead/team leftovers.
func (c *MimocodeConverter) AreasOnly() bool { return true }

// ConvertArea generates an area .md file from canonical YAML.
// It merges the corresponding lead's intelligence (Criteria, KnowledgeRules,
// ResponseStyle, Delegation, Squad) into the area so the area IS the full
// intelligent lead interface in the MiMoCode picker.
func (c *MimocodeConverter) ConvertArea(area *Area, leadForArea map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	// Look up the lead for this area to merge lead intelligence
	lead := leadForArea[area.ID]

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.Name))
	b.WriteString(fmt.Sprintf("description: \"◆ %s\"\n", area.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: false\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	if area.Permission != nil {
		writePermissionBlock(&b, area.Permission)
	}
	// OVAV instructions: always include the root AGENTS.md (gates,
	// identity seal, session protocol). Then layer the area-specific
	// OVAVConnection.Instructions on top.
	b.WriteString("instructions:\n")
	b.WriteString("  - \"AGENTS.md\"\n")
	if area.OVAVConnection != nil {
		for _, inst := range area.OVAVConnection.Instructions {
			b.WriteString(fmt.Sprintf("  - %q\n", inst))
		}
	}
	b.WriteString("---\n\n")

	// Identity guard — suppresses native model meta-identity
	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	// Body
	b.WriteString(fmt.Sprintf("**Lead:** %s\n", area.Lead))
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("**Color:** %s\n", area.Color))
	}
	if area.Surface != "" {
		b.WriteString(fmt.Sprintf("**Superficie:** %s\n", area.Surface))
	}
	b.WriteString("\n---\n\n")

	// OVAV Connection — declared wiring to the governor system. This is the
	// canonical place where an area tells the runtime which skills, contracts,
	// laws, and CLI commands it depends on. Anything missing here is not
	// honored by the runtime.
	if area.OVAVConnection != nil {
		b.WriteString("## Conexión OVAV (Governor System)\n\n")
		b.WriteString("Este área está cableada al sistema gobernador OVAV mediante los siguientes puntos de integración. **No remover** — cualquier desvío rompe el contrato global.\n\n")

		if len(area.OVAVConnection.Skills) > 0 {
			b.WriteString("### Skills cargadas\n\n")
			for _, s := range area.OVAVConnection.Skills {
				b.WriteString(fmt.Sprintf("- `%s`\n", s))
			}
			b.WriteString("\n")
		}

		if len(area.OVAVConnection.CLICommands) > 0 {
			b.WriteString("### Comandos CLI autorizados\n\n")
			b.WriteString("Estos son los únicos comandos del CLI OVAV que este área puede invocar. **Ejecutar desde la raíz del repo OVAV** (`$OVAV_ROOT` se reemplaza por la ruta real al cargar el área):\n\n")
			b.WriteString("```bash\n")
			b.WriteString("# Atajo universal — todos los comandos asumen estar en $OVAV_ROOT\n")
			b.WriteString(`export OVAV_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"` + "\n\n")
			for _, cmd := range area.OVAVConnection.CLICommands {
				b.WriteString(fmt.Sprintf("(cd \"$OVAV_ROOT\" && %s)\n", cmd))
			}
			b.WriteString("```\n")
		}

		if len(area.OVAVConnection.Contracts) > 0 {
			b.WriteString("### Contratos OVAV que aplica\n\n")
			for _, ct := range area.OVAVConnection.Contracts {
				b.WriteString(fmt.Sprintf("- `%s`\n", ct))
			}
			b.WriteString("\n")
		}

		if len(area.OVAVConnection.Laws) > 0 {
			b.WriteString("### Leyes OVAV que obedece\n\n")
			for _, l := range area.OVAVConnection.Laws {
				b.WriteString(fmt.Sprintf("- `%s`\n", l))
			}
			b.WriteString("\n")
		}
		b.WriteString("---\n\n")
	}

	b.WriteString("## Contratos de Gobernanza\n\n")
	b.WriteString("Esta área opera bajo los siguientes contratos OVAV:\n\n")
	b.WriteString("- **visual_delivery_contract.yaml** — Entrega visual: 50% shorter, no visible reasoning, result first, half_length_response\n")
	b.WriteString("- **safe_stop_contract.yaml** — Safe Stop Report: PARTIAL/SAFE_STOP/READY_FOR_COMMIT, Host Runtime vs OVAV Runtime distinction\n")
	b.WriteString("- **context_economy_contract.yaml** — Tiers T0-T5, escalation rules, must not load repo/internal OVAV context by default\n")
	b.WriteString("\n---\n\n")

	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HACE)\n\n")
	for i, fn := range area.Functions {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, fn))
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Limitaciones Explícitas (LO QUE NO HACE)\n\n")
	for _, lim := range area.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n---\n\n")

	// Lead intelligence: merge the lead's brain INTO the area so the area IS
	// the full intelligent lead interface in the MiMoCode picker.
	if lead != nil {
		// Decision Criteria from lead
		if lead.Criteria != "" {
			b.WriteString("## Decision Criteria\n\n")
			b.WriteString(lead.Criteria)
			b.WriteString("\n---\n\n")
		}

		// Knowledge Rules from lead
		if lead.KnowledgeRules != nil {
			b.WriteString("## Reglas de Conocimiento\n\n")
			b.WriteString(fmt.Sprintf("**Dominio:** %s\n\n", lead.KnowledgeRules.Domain))
			for _, rule := range lead.KnowledgeRules.Rules {
				b.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			b.WriteString("\n---\n\n")
		}

		// Response Style from lead
		if lead.ResponseStyle != nil {
			b.WriteString("## Estilo de Respuesta\n\n")
			b.WriteString(fmt.Sprintf("**Formato:** %s | **Máx palabras:** %d\n\n", lead.ResponseStyle.Format, lead.ResponseStyle.MaxWords))
			for _, rule := range lead.ResponseStyle.Rules {
				b.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			b.WriteString("\n---\n\n")
		}
	}

	b.WriteString("## Respuesta de Hard Stop\n\n")
	b.WriteString("```\n")
	b.WriteString(area.HardStop)
	b.WriteString("\n```\n\n---\n\n")

	// Use lead's Squad if available, fall back to area's SquadPreview
	if lead != nil && len(lead.Squad) > 0 {
		b.WriteString("## Squad Members\n\n")
		b.WriteString("| Miembro | País | Especialidad |\n")
		b.WriteString("|---------|------|-------------|\n")
		for _, m := range lead.Squad {
			b.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", m.Name, m.Country, m.Specialty))
		}
		b.WriteString("\n---\n\n")
	} else if len(area.SquadPreview) > 0 {
		b.WriteString("## Squad Members\n\n")
		b.WriteString("| Miembro | País | Especialidad |\n")
		b.WriteString("|---------|------|-------------|\n")
		for _, m := range area.SquadPreview {
			b.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", m.Name, m.Country, m.Specialty))
		}
		b.WriteString("\n---\n\n")
	}

	// Use lead's Delegation if available, fall back to area's
	if lead != nil && lead.Delegation != "" {
		b.WriteString("## Protocolo de Delegación\n\n")
		b.WriteString(lead.Delegation)
		b.WriteString("\n\n")
	} else if area.Delegation != "" {
		b.WriteString("## Protocolo de Delegación\n\n")
		b.WriteString(area.Delegation)
		b.WriteString("\n\n")
	}

	// HARD WIRED: delegation must use workflow("ovav-delegate")
	// actor tool is LIMITED to explore/general — NEVER use actor spawn for OVAV agents.
	// This section is the clean injection point: OVAV controls the delegation path.
	// When OVAV is removed, this section disappears with the generated agent file.
	b.WriteString("## Sistema de Delegación (OVAV)\n\n")
	b.WriteString("**Regla absoluta:** Para delegar trabajo a otro agente OVAV, usa:\n\n")
	b.WriteString("```\nworkflow(\"ovav-delegate\", {\n  agent_id: \"<agent-id>\",\n  task: \"<task-description>\",\n  context: {<context>}\n})\n```\n\n")
	b.WriteString("**No uses `actor spawn`** — el tool `actor` solo acepta tipos `explore` o `general`. Cualquier agent_id OVAV (ej: `lead-thavren`, `team-clara`) hace fallback silencioso a `general`, perdiendo identidad y permisos OVAV.\n\n")
	b.WriteString("**Directorio de agentes OVAV:** Los agentes OVAV se identifican con prefijos:\n")
	b.WriteString("- `area-<id>` — agentes de área (visible en TAB)\n")
	b.WriteString("- `lead-<id>` — leads OVAV (delegación primaria)\n")
	b.WriteString("- `team-<id>` — miembros del squad (delegación granular)\n\n")
	b.WriteString("El workflow `ovav-delegate` inyecta permisos OVAV, identidad del agente, y protocolos de handoff automáticamente.\n\n")

	if len(area.References) > 0 {
		b.WriteString("## Referencias Canónicas\n\n")
		for _, ref := range area.References {
			b.WriteString(fmt.Sprintf("- **%s**\n", ref))
		}
		b.WriteString("\n")
	}

	if len(area.GovernanceWiring) > 0 {
		b.WriteString("## Governance Wiring (DO NOT REMOVE)\n\n")
		b.WriteString("This area is governed by the following validators and gates. Removing these references will cause CI/CD failures:\n\n")
		for _, gw := range area.GovernanceWiring {
			b.WriteString(fmt.Sprintf("- %s\n", gw))
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — Área %s — Lead: %s*\n", area.Name, area.Lead))
	return []byte(b.String()), nil
}

// ConvertLead generates a lead .md file from canonical YAML.
func (c *MimocodeConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", lead.Name))
	b.WriteString(fmt.Sprintf("description: \"✦ %s\"\n", lead.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: true\n")
	if lead.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", lead.Color))
	}
	if lead.Permission != nil {
		writePermissionBlock(&b, lead.Permission)
	}
	if lead.Steps > 0 {
		b.WriteString(fmt.Sprintf("steps: %d\n", lead.Steps))
	}
	b.WriteString("---\n\n")

	// Identity guard — suppresses native model meta-identity
	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("**Área:** %s\n", lead.DisplayName))
	b.WriteString(fmt.Sprintf("**Origen:** %s\n", lead.Origin))
	if lead.Authority != "" {
		b.WriteString(fmt.Sprintf("**Autoridad:** `%s`\n", lead.Authority))
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Funciones Autorizadas (LO QUE SÍ HAGO)\n\n")
	for i, fn := range lead.Functions {
		b.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, fn))
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Limitaciones Explícitas (LO QUE NO HAGO)\n\n")
	for _, lim := range lead.Limitations {
		b.WriteString(fmt.Sprintf("- ❌ %s\n", lim))
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Respuesta de Hard Stop\n\n")
	b.WriteString("```\n")
	b.WriteString(lead.HardStop)
	b.WriteString("\n```\n\n---\n\n")

	if len(lead.Squad) > 0 {
		b.WriteString("## Squad\n\n")
		b.WriteString("| Miembro | País | Especialidad |\n")
		b.WriteString("|---------|------|-------------|\n")
		for _, m := range lead.Squad {
			b.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", m.Name, m.Country, m.Specialty))
		}
		b.WriteString("\n---\n\n")
	}

	if lead.Delegation != "" {
		b.WriteString("## Protocolo de Delegación\n\n")
		b.WriteString(lead.Delegation)
		b.WriteString("\n\n")
	}

	// HARD WIRED: delegation must use workflow("ovav-delegate")
	b.WriteString("## Sistema de Delegación (OVAV)\n\n")
	b.WriteString("**Regla absoluta:** Para delegar trabajo a un miembro del squad, usa:\n\n")
	b.WriteString("```\nworkflow(\"ovav-delegate\", {\n  agent_id: \"team-<member-id>\",\n  task: \"<task-description>\",\n  context: {<context>}\n})\n```\n\n")
	b.WriteString("**No uses `actor spawn`** — spawnea solo `explore` o `general`, haciendo fallback silencioso y perdiendo la identidad OVAV del team member.\n\n")
	b.WriteString("El workflow inyecta permisos, identidad y protocolos de handoff automáticamente.\n\n")

	if len(lead.References) > 0 {
		b.WriteString("## Referencias Canónicas\n\n")
		for _, ref := range lead.References {
			b.WriteString(fmt.Sprintf("- **%s**\n", ref))
		}
		b.WriteString("\n")
	}

	if lead.ResponseStyle != nil {
		b.WriteString("## Estilo de Respuesta\n\n")
		b.WriteString(fmt.Sprintf("**Formato:** %s | **Máx palabras:** %d\n\n", lead.ResponseStyle.Format, lead.ResponseStyle.MaxWords))
		for _, rule := range lead.ResponseStyle.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
		b.WriteString("\n")
	}

	if lead.KnowledgeRules != nil {
		b.WriteString("## Reglas de Conocimiento\n\n")
		b.WriteString(fmt.Sprintf("**Dominio:** %s\n\n", lead.KnowledgeRules.Domain))
		for _, rule := range lead.KnowledgeRules.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
		b.WriteString("\n")
	}

	// GAP-3: CRITERIA from .ovav/service_areas/ (all converters except Mimocode already had this)
	if lead.Criteria != "" {
		b.WriteString("## Decision Criteria\n\n")
		b.WriteString(lead.Criteria)
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — %s, Lead de %s*\n", lead.Name, lead.DisplayName))
	return []byte(b.String()), nil
}

// ConvertTeam generates a team .md file from canonical YAML.
func (c *MimocodeConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", team.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", team.Function))
	b.WriteString("mode: subagent\n")
	if team.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", team.Model))
	}
	b.WriteString("hidden: true\n")
	if team.Lead != "" {
		b.WriteString(fmt.Sprintf("lead: %q\n", team.Lead))
	}
	if team.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", team.Color))
	}
	if team.Permission != nil {
		writePermissionBlock(&b, team.Permission)
	}
	if team.Steps > 0 {
		b.WriteString(fmt.Sprintf("steps: %d\n", team.Steps))
	}
	b.WriteString("---\n\n")

	// Identity guard — suppresses native model meta-identity
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

	// Response style injection
	if team.ResponseStyle != nil {
		b.WriteString("\n## Estilo de Respuesta\n\n")
		b.WriteString(fmt.Sprintf("**Formato:** %s | **Máx palabras:** %d\n\n", team.ResponseStyle.Format, team.ResponseStyle.MaxWords))
		for _, rule := range team.ResponseStyle.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
	}

	// Knowledge rules injection
	if team.KnowledgeRules != nil {
		b.WriteString("\n## Reglas de Conocimiento\n\n")
		b.WriteString(fmt.Sprintf("**Dominio:** %s\n\n", team.KnowledgeRules.Domain))
		for _, rule := range team.KnowledgeRules.Rules {
			b.WriteString(fmt.Sprintf("- %s\n", rule))
		}
	}

	b.WriteString("\n---\n")
	b.WriteString(fmt.Sprintf("*OVAV Governor System — %s, %s*\n", team.Name, team.Function))
	if team.Lead != "" {
		b.WriteString(fmt.Sprintf("*Reporta a: %s · Área: %s*\n", team.Lead, team.Area))
	}
	return []byte(b.String()), nil
}
