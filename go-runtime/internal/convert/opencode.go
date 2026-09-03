package convert

import (
	"fmt"
	"sort"
	"strings"
)

// OpenCodeConverter transforms OVAV canonical agents into OpenCode .md files.
//
// Conversion rules:
//
//	Areas → mode:primary, hidden:false → visible in TAB selector
//	Leads → mode:primary, hidden:true  → hidden from TAB, invocable by name
//	Teams → mode:subagent, hidden:true → only via Task tool or @ mention
type OpenCodeConverter struct{}

func (c *OpenCodeConverter) FileExtension() string { return ".md" }
func (c *OpenCodeConverter) OutputDir() string     { return "go-runtime/internal/runtimes/opencode/agents" }

// AreasOnly returns false: opencode generates full hierarchy (areas + leads + teams).
// The TAB picker will show areas (mode:primary, hidden:false); leads and teams
// remain accessible via direct @ mention or Task tool. OpenCode does NOT honor
// `hidden: true` for teams, so they WILL appear in the TAB. This is a known
// limitation of the OpenCode harness that cannot be fixed without a custom
// workflow engine (which OpenCode does not support).
//
// Teams ARE accessible via Task tool (subagent_type=team-<id>) even though they
// also appear in the TAB. This is the trade-off for OpenCode.
//
// Lead intelligence (Criteria, KnowledgeRules, ResponseStyle) is embedded in the
// area agent body so the area IS the full intelligent interface.
func (c *OpenCodeConverter) AreasOnly() bool { return false }

// ConvertArea generates an area .md file from canonical YAML.
// It merges the corresponding lead's intelligence (Criteria, KnowledgeRules,
// ResponseStyle, Squad) into the area so the area IS the full intelligent
// lead interface — matching MiMoCodeConverter behavior.
func (c *OpenCodeConverter) ConvertArea(area *Area, leadForArea map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.Name))
	b.WriteString(fmt.Sprintf("description: \"◆ %s\"\n", area.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: false\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	// Permission block (validated by agent_permission_invariants)
	if area.Permission != nil {
		writePermissionBlock(&b, area.Permission)
	}
	// OVAV instructions: always include the OpenCode-specific AGENTS.md (gates,
	// identity seal, session protocol, OpenCode Task tool delegation).
	// Then layer the area-specific OVAVConnection.Instructions on top.
	b.WriteString("instructions:\n")
	b.WriteString("  - \"opencode_AGENTS.md\"\n")
	if area.OVAVConnection != nil {
		for _, inst := range area.OVAVConnection.Instructions {
			b.WriteString(fmt.Sprintf("  - %q\n", inst))
		}
	}
	b.WriteString("---\n\n")

	// Identity guard — suppresses native model meta-identity
	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	// Body — area description with lead reference
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
			for _, c := range area.OVAVConnection.CLICommands {
				b.WriteString(fmt.Sprintf("(cd \"$OVAV_ROOT\" && %s)\n", c))
			}
			b.WriteString("```\n\n")
		}

		if len(area.OVAVConnection.Contracts) > 0 {
			b.WriteString("### Contratos OVAV que aplica\n\n")
			for _, c := range area.OVAVConnection.Contracts {
				b.WriteString(fmt.Sprintf("- `%s`\n", c))
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

	// Lead intelligence: merge the lead's brain INTO the area so the area IS
	// the full intelligent lead interface in the TAB picker.
	// NOTE: area.ID uses hyphens (e.g., "platform-engineering") but lead.Area
	// uses underscores (e.g., "platform_engineering"). Normalize for lookup.
	leadAreaKey := strings.ReplaceAll(area.ID, "-", "_")
	lead := leadForArea[leadAreaKey]
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

	// Contracts — required by agent_ux_visual_delivery, context_economy validators
	b.WriteString("## Contratos de Gobernanza\n\n")
	b.WriteString("Esta área opera bajo los siguientes contratos OVAV:\n\n")
	b.WriteString("- **visual_delivery_contract.yaml** — Entrega visual: 50% shorter, no visible reasoning, result first, half_length_response\n")
	b.WriteString("- **safe_stop_contract.yaml** — Safe Stop Report: PARTIAL/SAFE_STOP/READY_FOR_COMMIT, Host Runtime vs OVAV Runtime distinction\n")
	b.WriteString("- **context_economy_contract.yaml** — Tiers T0-T5, escalation rules, must not load repo/internal OVAV context by default\n")
	b.WriteString("\n---\n\n")

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
	b.WriteString("## Respuesta de Hard Stop\n\n")
	b.WriteString("```\n")
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

	// HARD WIRED: OpenCode uses Task tool for delegation (NO workflow() — not available)
	b.WriteString("## Sistema de Delegación (OVAV — OpenCode)\n\n")
	b.WriteString("**Regla absoluta:** Para delegar trabajo a otro agente OVAV, usa el **Task tool** nativo de OpenCode:\n\n")
	b.WriteString("```\nTask({\n  description: \"<descripcion-corta>\",\n  prompt: \"<detalle del task para el agente destinatario>\",\n  subagent_type: \"<agent-id>\"\n})\n```\n\n")
	b.WriteString("**ID de agentes OVAV:**\n")
	b.WriteString("- `area-<id>` — agentes de área (visibles en TAB)\n")
	b.WriteString("- `lead-<id>` — leads OVAV (e.g., `lead-thavren`, `lead-eidren`)\n")
	b.WriteString("- `team-<id>` — miembros del squad (e.g., `team-clara`, `team-marco`)\n\n")
	b.WriteString("**No uses `actor spawn`** — el tool `actor` solo acepta tipos `explore` o `general`, haciendo fallback silencioso y perdiendo la identidad OVAV del agente.\n\n")
	b.WriteString("**No uses `workflow()`** — el tool `workflow()` no existe en OpenCode. Solo Task tool.\n\n")

	// References
	if len(area.References) > 0 {
		b.WriteString("## Referencias Canónicas\n\n")
		for _, ref := range area.References {
			b.WriteString(fmt.Sprintf("- **%s**\n", ref))
		}
		b.WriteString("\n")
	}

	// Governance Wiring
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
func (c *OpenCodeConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", lead.Name))
	b.WriteString(fmt.Sprintf("description: \"✦ %s\"\n", lead.Description))
	b.WriteString("mode: primary\n")
	b.WriteString("hidden: true\n")
	if lead.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", lead.Color))
	}
	// Permission block (validated by agent_permission_invariants)
	if lead.Permission != nil {
		writePermissionBlock(&b, lead.Permission)
	}
	b.WriteString("---\n\n")

	// Identity guard — suppresses native model meta-identity
	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	// Body — lead profile
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
	b.WriteString("## Respuesta de Hard Stop\n\n")
	b.WriteString("```\n")
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

	// HARD WIRED: OpenCode uses Task tool for delegation (NO workflow() — not available)
	b.WriteString("## Sistema de Delegación (OVAV — OpenCode)\n\n")
	b.WriteString("**Regla absoluta:** Para delegar trabajo a un miembro del squad, usa el **Task tool** nativo de OpenCode:\n\n")
	b.WriteString("```\nTask({\n  description: \"<descripcion-corta>\",\n  prompt: \"<detalle del task para el miembro del squad>\",\n  subagent_type: \"team-<member-id>\"\n})\n```\n\n")
	b.WriteString("**Team members disponibles:** ver tabla Squad Members arriba para el ID correcto (e.g., `team-clara`, `team-marco`).\n\n")
	b.WriteString("**No uses `actor spawn`** — spawnea solo `explore` o `general`, perdiendo identidad OVAV del team member.\n\n")
	b.WriteString("**No uses `workflow()`** — el tool `workflow()` no existe en OpenCode. Solo Task tool.\n\n")

	// References
	if len(lead.References) > 0 {
		b.WriteString("## Referencias Canónicas\n\n")
		for _, ref := range lead.References {
			b.WriteString(fmt.Sprintf("- **%s**\n", ref))
		}
		b.WriteString("\n")
	}

	// GAP-3: CRITERIA from .ovav/service_areas/
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
func (c *OpenCodeConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", team.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", team.Function))
	b.WriteString("mode: subagent\n")
	if team.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", team.Model))
	}
	b.WriteString("hidden: true\n")
	if team.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", team.Color))
	}
	// Permission block
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

	// Body — team profile
	b.WriteString(fmt.Sprintf("**País:** %s\n", team.Country))
	b.WriteString(fmt.Sprintf("**Reporta a:** %s\n", team.Lead))
	b.WriteString(fmt.Sprintf("**Área:** %s\n", team.Area))
	b.WriteString("\n")

	// Function
	b.WriteString("## Función Principal\n\n")
	b.WriteString(team.Function)
	b.WriteString("\n\n")

	// Actions
	b.WriteString("## Acciones Autorizadas\n\n")
	for i, action := range team.Actions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
	}
	b.WriteString("\n")

	// Hard Stop
	b.WriteString("## Hard Stop\n\n")
	b.WriteString(fmt.Sprintf("%q\n\n", team.HardStop))

	// Response
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

// writePermissionBlock writes a YAML permission block into the frontmatter.
func writePermissionBlock(b *strings.Builder, p *PermissionBlock) {
	b.WriteString("permission:\n")
	b.WriteString(fmt.Sprintf("  edit: %q\n", p.Edit))
	if len(p.Bash) > 0 {
		b.WriteString("  bash:\n")
		for _, k := range orderedProjectionKeys(p.Bash) {
			v := p.Bash[k]
			// Quote keys that start with * or contain special chars to ensure valid YAML
			if strings.HasPrefix(k, "*") || strings.Contains(k, ":") || strings.Contains(k, "-") {
				b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
			} else {
				b.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
			}
		}
	}
	if len(p.ExternalDirectory) > 0 {
		b.WriteString("  external_directory:\n")
		for _, k := range orderedProjectionKeys(p.ExternalDirectory) {
			v := p.ExternalDirectory[k]
			b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
		}
	}
}

func orderedProjectionKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "*" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if _, ok := values["*"]; ok {
		return append([]string{"*"}, keys...)
	}
	return keys
}
