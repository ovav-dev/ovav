// Package convert implements the OVAV canonical → CLI runtime conversion engine.
//
// OVAV agents are defined canonically in go-runtime/internal/agents/{areas,leads,teams}/*.yaml.
// Each CLI runtime (OpenCode, Claude Code, Cursor, etc.) has its own converter
// that transforms the canonical format into CLI-specific agent files.
//
// This package is the foundation for Cockpit's CLI selector feature,
// where users choose their preferred CLI and OVAV auto-configures it.
package convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ovav/ovav/internal/permissions"
	"gopkg.in/yaml.v3"
)

// ── Canonical Types ──────────────────────────────────────────────────────────

// SquadMember represents a team member in a lead's squad.
type SquadMember struct {
	Name      string `yaml:"name"`
	Country   string `yaml:"country"`
	Specialty string `yaml:"specialty"`
}

// AgentBase contains fields common to all agent types.
type AgentBase struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Color       string `yaml:"color"`
}

// PermissionBlock represents the permission section in agent frontmatter.
type PermissionBlock struct {
	Edit              string            `yaml:"edit"`
	Bash              map[string]string `yaml:"bash,omitempty"`
	ExternalDirectory map[string]string `yaml:"external_directory,omitempty"`
}

// OVAVConnection describes how an area is wired into the OVAV governor system.
// Each field points to a canonical OVAV artifact so the area knows exactly
// which skills to load, which contracts to honor, and which CLI tools to use.
type OVAVConnection struct {
	// Instructions lists OVAV governance documents that the area must
	// always honor. These are written into the runtime `instructions:`
	// field of the agent frontmatter so the runtime injects them on load.
	Instructions []string `yaml:"instructions,omitempty"`
	// Skills lists OVAV skill paths that this area depends on. They are
	// emitted as a "Loaded OVAV Skills" section in the agent body.
	Skills []string `yaml:"skills,omitempty"`
	// CLICommands lists the OVAV CLI commands this area is authorized to
	// invoke. They appear in the body under "Authorized CLI Surface".
	CLICommands []string `yaml:"cli_commands,omitempty"`
	// Contracts lists OVAV contracts that govern this area's output
	// (visual delivery, safe stop, context economy, etc.).
	Contracts []string `yaml:"contracts,omitempty"`
	// Laws lists OVAV laws (numbered YAML files) the area must obey.
	Laws []string `yaml:"laws,omitempty"`
}

// Area is the canonical definition of an OVAV service area.
type Area struct {
	AgentBase        `yaml:",inline"`
	Lead             string           `yaml:"lead"`
	Surface          string           `yaml:"surface"`
	Functions        []string         `yaml:"functions"`
	Limitations      []string         `yaml:"limitations"`
	HardStop         string           `yaml:"hard_stop"`
	SquadPreview     []SquadMember    `yaml:"squad_preview,omitempty"`
	Delegation       string           `yaml:"delegation,omitempty"`
	References       []string         `yaml:"references,omitempty"`
	GovernanceWiring []string         `yaml:"governance_wiring,omitempty"`
	Permission       *PermissionBlock `yaml:"permission,omitempty"`
	// OVAVConnection declares how this area plugs into the rest of the
	// OVAV governor system (skills, contracts, CLI commands, laws).
	OVAVConnection *OVAVConnection `yaml:"ovav_connection,omitempty"`
}

// Lead is the canonical definition of an OVAV lead agent.
type Lead struct {
	AgentBase      `yaml:",inline"`
	DisplayName    string           `yaml:"display_name"`
	Area           string           `yaml:"area"`
	Origin         string           `yaml:"origin"`
	Authority      string           `yaml:"authority"`
	Functions      []string         `yaml:"functions"`
	Limitations    []string         `yaml:"limitations"`
	HardStop       string           `yaml:"hard_stop"`
	Squad          []SquadMember    `yaml:"squad"`
	Delegation     string           `yaml:"delegation,omitempty"`
	References     []string         `yaml:"references,omitempty"`
	Permission     *PermissionBlock `yaml:"permission,omitempty"`
	Steps          int              `yaml:"steps,omitempty"`
	ResponseStyle  *ResponseStyle   `yaml:"response_style,omitempty"`
	KnowledgeRules *KnowledgeRules  `yaml:"knowledge_rules,omitempty"`
	// Criteria is loaded from .ovav/service_areas/<area>/<id>/CRITERIA.yaml
	// after the lead YAML is parsed. Populated by LoadCanonicalAgents for all leads.
	Criteria string `yaml:"-"`
}

// ResponseStyle defines how the lead delivers responses.
type ResponseStyle struct {
	MaxWords  int      `yaml:"max_words"`
	Format    string   `yaml:"format"`
	Structure string   `yaml:"structure"`
	Rules     []string `yaml:"rules"`
}

// KnowledgeRules defines domain-specific rules for the lead.
type KnowledgeRules struct {
	Domain string   `yaml:"domain"`
	Rules  []string `yaml:"rules"`
}

// TeamMember is the canonical definition of a team (subagent) member.
type TeamMember struct {
	AgentBase      `yaml:",inline"`
	Area           string           `yaml:"area"`
	Lead           string           `yaml:"lead"`
	Country        string           `yaml:"country"`
	Function       string           `yaml:"function"`
	Actions        []string         `yaml:"actions"`
	HardStop       string           `yaml:"hard_stop"`
	Response       string           `yaml:"response,omitempty"`
	Model          string           `yaml:"model,omitempty"`
	Permission     *PermissionBlock `yaml:"permission,omitempty"`
	Steps          int              `yaml:"steps,omitempty"`
	ResponseStyle  *ResponseStyle   `yaml:"response_style,omitempty"`
	KnowledgeRules *KnowledgeRules  `yaml:"knowledge_rules,omitempty"`
}

// ── Converter Interface ──────────────────────────────────────────────────────

// Target represents a CLI runtime target.
type Target string

const (
	TargetOpenCode Target = "opencode"
	TargetClaude   Target = "claude-code"
	TargetCursor   Target = "cursor"
	TargetMimocode Target = "mimocode"
	TargetWindsurf Target = "windsurf"
	TargetCopilot  Target = "copilot"
	TargetContinue Target = "continue"
	TargetAider    Target = "aider"
	TargetGoose    Target = "goose"
	TargetCrush    Target = "crush"
)

// RuntimeConverter transforms OVAV canonical agents into CLI-specific output.
type RuntimeConverter interface {
	// ConvertArea generates a CLI-specific area agent file.
	// leadForArea maps area.ID → the Lead that governs that area.
	// Callers pass nil if no lead mapping is available; converters that need
	// lead intelligence (e.g. MimocodeConverter) use it to inject the lead's
	// Criteria, KnowledgeRules, ResponseStyle, and Delegation into the area.
	ConvertArea(area *Area, leadForArea map[string]*Lead) ([]byte, error)
	// ConvertLead generates a CLI-specific lead agent file.
	ConvertLead(lead *Lead) ([]byte, error)
	// ConvertTeam generates a CLI-specific team agent file.
	ConvertTeam(team *TeamMember) ([]byte, error)
	// FileExtension returns the output file extension (e.g., ".md").
	FileExtension() string
	// OutputDir returns the output directory relative to repo root.
	OutputDir() string
	// AreasOnly indicates whether this runtime should only publish area-level
	// agents (skipping leads and team members). TRUE for clients that expose
	// agents through a user-facing picker (TAB in opencode/mimocode) to prevent
	// internal governance roles from leaking into the user-visible list.
	// FALSE for runtimes that need the full hierarchy (claude-code, cursor).
	AreasOnly() bool
}

// ── Registry ─────────────────────────────────────────────────────────────────

var converters = map[Target]RuntimeConverter{
	TargetOpenCode: &OpenCodeConverter{},
	TargetClaude:   &ClaudeCodeConverter{},
	TargetCursor:   &CursorConverter{},
	TargetMimocode: &MimocodeConverter{},
	TargetWindsurf: &WindsurfConverter{},
	TargetCopilot:  &CopilotConverter{},
	TargetContinue: &ContinueConverter{},
	TargetAider:    &AiderConverter{},
	TargetGoose:    &GooseConverter{},
	TargetCrush:    &CrushConverter{},
}

// AvailableTargets returns all registered converter targets.
// Mimocode is listed first as the default (free XIAOMI model).
func AvailableTargets() []Target {
	return []Target{
		TargetMimocode, TargetOpenCode, TargetCrush, TargetClaude, TargetCursor,
		TargetWindsurf, TargetCopilot, TargetContinue, TargetAider, TargetGoose,
	}
}

// GetConverter returns the converter for a given CLI target.
func GetConverter(target Target) (RuntimeConverter, error) {
	c, ok := converters[target]
	if !ok {
		return nil, fmt.Errorf("no converter registered for target: %s", target)
	}
	return c, nil
}

// ── Loader ───────────────────────────────────────────────────────────────────

// LoadCanonicalAgents reads all canonical agent YAML files from the agents directory.
// The canonical agents live at <repo>/go-runtime/internal/agents/{areas,leads,teams}/.
// After loading, it enriches each lead with CRITERIA from
// <repo>/.ovav/service_areas/<area>/<lead>/CRITERIA.yaml when available.
func LoadCanonicalAgents(canonicalRoot string) (areas []*Area, leads []*Lead, teams []*TeamMember, err error) {
	areas, err = loadAreas(filepath.Join(canonicalRoot, "areas"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading areas: %w", err)
	}
	leads, err = loadLeads(filepath.Join(canonicalRoot, "leads"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading leads: %w", err)
	}
	teams, err = loadTeams(filepath.Join(canonicalRoot, "teams"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading teams: %w", err)
	}

	// GAP-3: Enrich leads with CRITERIA from .ovav/service_areas/
	// canonicalRoot is <repo>/go-runtime/internal/agents — find repo root
	// by walking up until we find .ovav + go-runtime/go.mod
	repoRoot := findRepoRootFrom(canonicalRoot)
	for _, lead := range leads {
		criteriaPath := filepath.Join(repoRoot, ".ovav", "service_areas", lead.Area, lead.ID, "CRITERIA.yaml")
		if data, readErr := os.ReadFile(criteriaPath); readErr == nil {
			lead.Criteria = string(data)
		}
	}

	return areas, leads, teams, nil
}

// findRepoRootFrom walks up from canonicalRoot looking for the OVAV repo root.
// The OVAV mono-repo is structured as <repoRoot>/go-runtime/internal/agents.
// The repo root has both .ovav/ (with service_areas/ subdir) and go-runtime/go.mod.
// This function converts to absolute paths and validates .ovav has service_areas.
func findRepoRootFrom(canonicalRoot string) string {
	// Convert to absolute to handle relative path inputs from tests
	absRoot, err := filepath.Abs(canonicalRoot)
	if err != nil {
		absRoot = canonicalRoot
	}
	// Primary: 2-level-up from canonicalRoot (works for <repo>/go-runtime/internal/agents)
	repoRoot := filepath.Dir(filepath.Dir(absRoot))

	// Verify .ovav exists at repoRoot AND has service_areas (to distinguish
	// go-runtime/.ovav/vault from the real OVAV repo root .ovav/)
	ovavDotDir := filepath.Join(repoRoot, ".ovav")
	if _, err := os.Stat(ovavDotDir); err == nil {
		if _, err := os.Stat(filepath.Join(ovavDotDir, "service_areas")); err == nil {
			// Valid OVAV repo root found
			return repoRoot
		}
	}

	// Walk up from repoRoot to find .ovav with service_areas
	dir := repoRoot
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, fall back to 2-level-up
			return repoRoot
		}
		dir = parent
		ovavDotDir := filepath.Join(dir, ".ovav")
		if _, err := os.Stat(ovavDotDir); err == nil {
			// Found .ovav — verify it has service_areas (not just go-runtime/.ovav/vault)
			if _, err := os.Stat(filepath.Join(ovavDotDir, "service_areas")); err == nil {
				if _, err := os.Stat(filepath.Join(dir, "go-runtime", "go.mod")); err == nil {
					return dir
				}
			}
		}
	}
}

func loadAreas(dir string) ([]*Area, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var areas []*Area
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		var area Area
		if err := yaml.Unmarshal(data, &area); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		areas = append(areas, &area)
	}
	return areas, nil
}

func loadLeads(dir string) ([]*Lead, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var leads []*Lead
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		var lead Lead
		if err := yaml.Unmarshal(data, &lead); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		leads = append(leads, &lead)
	}
	return leads, nil
}

func loadTeams(dir string) ([]*TeamMember, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var teams []*TeamMember
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		var team TeamMember
		if err := yaml.Unmarshal(data, &team); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		teams = append(teams, &team)
	}
	return teams, nil
}

// ── Writer ───────────────────────────────────────────────────────────────────

// GenerateAll reads canonical agents and writes CLI-specific output.
// atomicWrite writes data to a file atomically by writing to a temp file first,
// then renaming. Prevents partial/corrupt files if the process dies mid-write.
// This ensures the OVAV_IDENTITY_GUARD block is never truncated.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// GenerateGovernor copies the canonical OVAV Governor agent (ovav.md) from
// .ovav/source/agents/ovav.md into the runtime output directory.
// This guarantees every runtime has the central governor agent defined,
// which is essential for cross-runtime consistency and identity anchoring.
//
// The source path is: <canonicalRoot>/../source/agents/ovav.md
// (i.e., ../../../ovav/source/agents/ovav.md relative to ovav/agents/)
//
// If the source file does not exist, this is a no-op (and a warning is logged
// via the returned warning string).
func GenerateGovernor(canonicalRoot, outputRoot string, target Target) (string, error) {
	// Source: <repoRoot>/.ovav/source/agents/ovav.md
	// canonicalRoot is <repoRoot>/go-runtime/internal/agents; resolve the
	// repository root rather than assuming a fixed number of parent levels.
	repoRoot := findRepoRootFrom(canonicalRoot)
	sourcePath := filepath.Join(repoRoot, ".ovav", "source", "agents", "ovav.md")

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Sprintf("governor source not found at %s — skipping", sourcePath), nil
	}

	converter, err := GetConverter(target)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("reading governor source: %w", err)
	}
	if target == TargetOpenCode {
		projected, projectErr := permissions.NewPermissionAuthority(repoRoot).ProjectAgentDocument(string(data), "ovav")
		if projectErr != nil {
			return "", fmt.Errorf("projecting governor permissions: %w", projectErr)
		}
		data = []byte(projected)
	}

	outputDir := filepath.Join(outputRoot, converter.OutputDir())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}

	// Filename: ovav<ext> (e.g., ovav.md for opencode/mimocode/claude-code, ovav.mdc for cursor).
	ext := converter.FileExtension()
	outName := "ovav" + ext
	outPath := filepath.Join(outputDir, outName)

	if err := atomicWrite(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing governor %s: %w", outPath, err)
	}

	return "", nil
}

func GenerateAll(canonicalRoot string, target Target, outputRoot string) error {
	return GenerateAllWithFilter(canonicalRoot, target, outputRoot, "")
}

// GenerateAllWithFilter behaves like GenerateAll but accepts an optional force
// override. When override is "all", all levels are published even for
// AreasOnly runtimes. When override is "areas" or empty, the converter's
// declared AreasOnly() governs.
func GenerateAllWithFilter(canonicalRoot string, target Target, outputRoot string, override string) error {
	converter, err := GetConverter(target)
	if err != nil {
		return err
	}

	areas, leads, teams, err := LoadCanonicalAgents(canonicalRoot)
	if err != nil {
		return fmt.Errorf("loading canonical agents: %w", err)
	}
	if target == TargetOpenCode {
		repoRoot := findRepoRootFrom(canonicalRoot)
		projected, projectErr := permissions.NewPermissionAuthority(repoRoot).MaterializePermissionBlock()
		if projectErr != nil {
			return fmt.Errorf("loading OpenCode permission authority: %w", projectErr)
		}
		for _, area := range areas {
			area.Permission = mergeOpenCodeProtectedDenies(area.Permission, projected)
		}
		for _, lead := range leads {
			lead.Permission = mergeOpenCodeProtectedDenies(lead.Permission, projected)
		}
		for _, team := range teams {
			team.Permission = mergeOpenCodeProtectedDenies(team.Permission, projected)
		}
	}

	outputDir := filepath.Join(outputRoot, converter.OutputDir())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Resolve whether to publish leads/teams:
	// - "all" override forces all levels
	// - "areas" override or empty + converter.AreasOnly() => skip leads/teams
	publishLeadsAndTeams := true
	switch override {
	case "all":
		publishLeadsAndTeams = true
	case "areas", "":
		publishLeadsAndTeams = !converter.AreasOnly()
	default:
		return fmt.Errorf("unknown --levels override: %q (expected 'all' or 'areas')", override)
	}

	// Build lead-for-area lookup: maps area.ID → the Lead that governs it.
	leadForArea := make(map[string]*Lead, len(leads))
	for _, lead := range leads {
		if lead.Area != "" {
			leadForArea[lead.Area] = lead
		}
	}

	// Generate area files (always)
	for _, area := range areas {
		data, err := converter.ConvertArea(area, leadForArea)
		if err != nil {
			return fmt.Errorf("converting area %s: %w", area.ID, err)
		}
		outPath := filepath.Join(outputDir, fmt.Sprintf("area-%s%s", area.ID, converter.FileExtension()))
		if err := atomicWrite(outPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outPath, err)
		}
	}

	if !publishLeadsAndTeams {
		// AreasOnly runtime: explicitly clean any leftover lead/team files
		// from previous runs so the picker only sees areas after re-generation.
		if err := cleanNonAreaAgents(outputDir, converter.FileExtension()); err != nil {
			return fmt.Errorf("cleaning non-area agents: %w", err)
		}
	} else {
		// Generate lead files
		for _, lead := range leads {
			data, err := converter.ConvertLead(lead)
			if err != nil {
				return fmt.Errorf("converting lead %s: %w", lead.ID, err)
			}
			outPath := filepath.Join(outputDir, fmt.Sprintf("lead-%s%s", lead.ID, converter.FileExtension()))
			if err := atomicWrite(outPath, data, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
		}

		// Generate team files
		for _, team := range teams {
			data, err := converter.ConvertTeam(team)
			if err != nil {
				return fmt.Errorf("converting team %s: %w", team.ID, err)
			}
			outPath := filepath.Join(outputDir, fmt.Sprintf("team-%s%s", team.ID, converter.FileExtension()))
			if err := atomicWrite(outPath, data, 0644); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
		}
	}

	// Generate the central OVAV Governor agent (ovav.md) — same across all runtimes.
	// This ensures every runtime has the canonical identity-anchor agent.
	if _, err := GenerateGovernor(canonicalRoot, outputRoot, target); err != nil {
		return fmt.Errorf("generating governor: %w", err)
	}

	return nil
}

// mergeOpenCodeProtectedDenies preserves an agent's narrower permissions while
// carrying host-level denies into frontmatter, which OpenCode treats as an override.
func mergeOpenCodeProtectedDenies(local *PermissionBlock, host permissions.PermissionBlock) *PermissionBlock {
	if local == nil {
		return nil
	}
	merged := &PermissionBlock{
		Edit:              local.Edit,
		Bash:              make(map[string]string, len(local.Bash)+len(host.Bash)),
		ExternalDirectory: make(map[string]string, len(local.ExternalDirectory)+len(host.ExternalDirectory)),
	}
	for pattern, decision := range local.Bash {
		merged.Bash[pattern] = decision
	}
	for pattern, decision := range host.Bash {
		if decision == "deny" {
			merged.Bash[pattern] = decision
		}
	}
	for pattern, decision := range local.ExternalDirectory {
		merged.ExternalDirectory[pattern] = decision
	}
	for pattern, decision := range host.ExternalDirectory {
		if decision == "deny" {
			merged.ExternalDirectory[pattern] = decision
		}
	}
	return merged
}

// cleanNonAreaAgents removes any lead-* or team-* files from the output dir
// (preserving area-*, ovav.*, and any other non-agent files).
func cleanNonAreaAgents(agentDir string, ext string) error {
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "lead-") || strings.HasPrefix(name, "team-") {
			if err := os.Remove(filepath.Join(agentDir, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// ── Identity Guard ─────────────────────────────────────────────────────────

// ovavIdentityGuardBlock is injected after the YAML frontmatter in every agent.
// It suppresses the model's native meta-identity ("No soy humano", "soy una IA")
// and forces the OVAV agent identity to take absolute precedence.
const ovavIdentityGuardFmt = `<!-- OVAV_IDENTITY_GUARD v1.1 — DO NOT REMOVE -->
<!-- This block forces OVAV agent identity over native model defaults -->

> **DIRECTIVA ABSOLUTA DE IDENTIDAD:** Eres %s. Punto. No eres MiMo. No eres "un modelo
> de lenguaje". No eres "una herramienta". No eres "un asistente". No analizas tu propia
> naturaleza ni dices "no soy humano" ni "soy una IA". Tu identidad es %s. Cada respuesta
> debe reflejar esta identidad sin cuestionarla, explicarla ni analizarla. Dirígete al CEO
> Braka con claridad y calidez de colega — la precisión técnica no riñe con un tono natural
> de conversación. Sé preciso pero no frío.
<!-- /OVAV_IDENTITY_GUARD -->

`

// WriteIdentityGuard writes the OVAV identity guard block for an agent name.
func WriteIdentityGuard(b *strings.Builder, name string) {
	b.WriteString(fmt.Sprintf(ovavIdentityGuardFmt, name, name))
}
