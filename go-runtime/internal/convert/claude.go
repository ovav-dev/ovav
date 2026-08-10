package convert

import (
	"fmt"
	"strings"
)

// ClaudeCodeConverter transforms OVAV canonical agents into Claude Code .md files.
//
// Claude Code uses .claude/agents/ directory with .md files.
// Verified format: https://docs.anthropic.com/en/docs/claude-code/agents
//
// Frontmatter fields (VERIFIED):
//   name, description, tools, disallowedTools, model, permissionMode,
//   maxTurns, skills, mcpServers, hooks, memory, background, effort,
//   isolation, color, initialPrompt
//
// NO type:, hidden:, mode:, permission: blocks — these are OVAV extensions
// NOT supported by Claude Code's agent format.
type ClaudeCodeConverter struct{}

func (c *ClaudeCodeConverter) FileExtension() string { return ".md" }
func (c *ClaudeCodeConverter) OutputDir() string {
	return "go-runtime/internal/runtimes/claude-code/agents"
}

// AreasOnly returns false: full hierarchy needed (area/lead/subagent via type:).
func (c *ClaudeCodeConverter) AreasOnly() bool { return false }

// ConvertArea generates a Claude Code area agent.
func (c *ClaudeCodeConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", hexToClaudeColor(area.Color)))
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("**Lead:** %s\n", area.Lead))
	if area.Surface != "" {
		b.WriteString(fmt.Sprintf("**Surface:** %s\n", area.Surface))
	}
	b.WriteString("\n## Functions\n\n")
	for _, fn := range area.Functions {
		b.WriteString(fmt.Sprintf("- %s\n", fn))
	}
	b.WriteString("\n## Limitations\n\n")
	for _, lim := range area.Limitations {
		b.WriteString(fmt.Sprintf("- %s\n", lim))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(area.HardStop)
	b.WriteString("\n")

	return []byte(b.String()), nil
}

// ConvertLead generates a Claude Code lead agent.
func (c *ClaudeCodeConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", lead.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", lead.Description))
	if lead.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", hexToClaudeColor(lead.Color)))
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", lead.Name))
	b.WriteString(fmt.Sprintf("**Display Name:** %s\n", lead.DisplayName))
	b.WriteString(fmt.Sprintf("**Origin:** %s\n", lead.Origin))
	if lead.Authority != "" {
		b.WriteString(fmt.Sprintf("**Authority:** %s\n", lead.Authority))
	}
	b.WriteString("\n## Authorized Functions\n\n")
	for _, fn := range lead.Functions {
		b.WriteString(fmt.Sprintf("- %s\n", fn))
	}
	b.WriteString("\n## Limitations\n\n")
	for _, lim := range lead.Limitations {
		b.WriteString(fmt.Sprintf("- %s\n", lim))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(lead.HardStop)
	b.WriteString("\n")

	// Decision Criteria from GAP-3
	if lead.Criteria != "" {
		b.WriteString("\n## Decision Criteria\n\n")
		b.WriteString(lead.Criteria)
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}

// ConvertTeam generates a Claude Code team subagent.
func (c *ClaudeCodeConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", team.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", team.Function))
	if team.Model != "" {
		b.WriteString(fmt.Sprintf("model: %s\n", team.Model))
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, team.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", team.Name))
	b.WriteString(fmt.Sprintf("**Country:** %s\n", team.Country))
	b.WriteString(fmt.Sprintf("**Reports to:** %s\n", team.Lead))
	b.WriteString(fmt.Sprintf("**Area:** %s\n\n", team.Area))
	b.WriteString(fmt.Sprintf("## Function\n\n%s\n\n", team.Function))
	b.WriteString("## Actions\n\n")
	for _, action := range team.Actions {
		b.WriteString(fmt.Sprintf("- %s\n", action))
	}

	return []byte(b.String()), nil
}

// hexToClaudeColor converts hex colors to Claude Code's named colors.
// Claude Code supports: red, blue, green, yellow, purple, orange, pink, cyan
func hexToClaudeColor(hex string) string {
	hex = strings.ToLower(strings.TrimPrefix(hex, "#"))
	colorMap := map[string]string{
		"ef4444": "red",
		"f97316": "orange",
		"eab308": "yellow",
		"22c55e": "green",
		"06b6d4": "cyan",
		"3b82f6": "blue",
		"8b5cf6": "purple",
		"ec4899": "pink",
		"dc2626": "red",
		"ea580c": "orange",
		"ca8a04": "yellow",
		"16a34a": "green",
		"0891b2": "cyan",
		"2563eb": "blue",
		"7c3aed": "purple",
		"db2777": "pink",
	}
	if named, ok := colorMap[hex]; ok {
		return named
	}
	return hex
}
