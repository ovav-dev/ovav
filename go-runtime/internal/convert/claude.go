package convert

import (
	"fmt"
	"strings"
)

// ClaudeCodeConverter transforms OVAV canonical agents into Claude Code .md files.
//
// Claude Code uses .claude/agents/ directory with .md files.
// Format is similar to OpenCode but with Claude-specific frontmatter conventions.
type ClaudeCodeConverter struct{}

func (c *ClaudeCodeConverter) FileExtension() string { return ".md" }
func (c *ClaudeCodeConverter) OutputDir() string     { return "runtimes/claude-code/agents" }

// AreasOnly returns false: Claude Code's agent system is hierarchical and
// distinguishes areas / leads / subagents via the `type:` frontmatter field,
// respecting `hidden: true`. The full hierarchy is needed.
func (c *ClaudeCodeConverter) AreasOnly() bool { return false }

func (c *ClaudeCodeConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("type: area\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	// Permission block
	if area.Permission != nil {
		writePermissionBlock(&b, area.Permission)
	}
	b.WriteString("---\n\n")

	// OVAV Identity Guard — same as opencode/mimocode
	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("**Lead:** %s\n\n", area.Lead))
	b.WriteString("## Functions\n\n")
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

func (c *ClaudeCodeConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", lead.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", lead.Description))
	b.WriteString("type: lead\n")
	b.WriteString("hidden: true\n")
	if lead.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", lead.Color))
	}
	// Permission block
	if lead.Permission != nil {
		writePermissionBlock(&b, lead.Permission)
	}
	b.WriteString("---\n\n")

	// OVAV Identity Guard
	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s — %s\n\n", lead.Name, lead.DisplayName))
	b.WriteString(fmt.Sprintf("**Origin:** %s\n\n", lead.Origin))
	b.WriteString("## Authorized Functions\n\n")
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

	// GAP-3: CRITERIA from .ovav/service_areas/
	if lead.Criteria != "" {
		b.WriteString("\n## Decision Criteria\n\n")
		b.WriteString(lead.Criteria)
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}

func (c *ClaudeCodeConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", team.Name))
	b.WriteString("type: subagent\n")
	b.WriteString("hidden: true\n")
	if team.Permission != nil {
		writePermissionBlock(&b, team.Permission)
	}
	b.WriteString("---\n\n")

	// OVAV Identity Guard
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
