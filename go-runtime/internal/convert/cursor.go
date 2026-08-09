package convert

import (
	"fmt"
	"strings"
)

// CursorConverter transforms OVAV canonical agents into Cursor .cursorrules format.
//
// Cursor uses .cursor/rules/ directory with .mdc files.
// The format embeds agent instructions as markdown rules that Cursor's AI can reference.
type CursorConverter struct{}

func (c *CursorConverter) FileExtension() string { return ".mdc" }
func (c *CursorConverter) OutputDir() string     { return "runtimes/cursor/rules" }

// AreasOnly returns false: Cursor rules are applied through .mdc files where
// the hierarchy (area / lead / subagent) is flattened into rule content. The
// full hierarchy is needed to compose the rule bodies.
func (c *CursorConverter) AreasOnly() bool { return false }

func (c *CursorConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("alwaysApply: false\n")
	// Permission block
	if area.Permission != nil {
		writePermissionBlock(&b, area.Permission)
	}
	b.WriteString("---\n\n")
	// OVAV Identity Guard — HTML comments work in .mdc format
	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# OVAV Area: %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("Lead: %s\n\n", area.Lead))
	b.WriteString("## Authorized Functions\n\n")
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

func (c *CursorConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: \"%s — %s\"\n", lead.Name, lead.DisplayName))
	b.WriteString("alwaysApply: false\n")
	// Permission block
	if lead.Permission != nil {
		writePermissionBlock(&b, lead.Permission)
	}
	b.WriteString("---\n\n")

	// OVAV Identity Guard
	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s — %s\n\n", lead.Name, lead.DisplayName))
	b.WriteString(fmt.Sprintf("Origin: %s\n\n", lead.Origin))
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

func (c *CursorConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: \"Team: %s — %s\"\n", team.Name, team.Country))
	b.WriteString("alwaysApply: false\n")
	if team.Permission != nil {
		writePermissionBlock(&b, team.Permission)
	}
	b.WriteString("---\n\n")

	// OVAV Identity Guard
	WriteIdentityGuard(&b, team.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# Team: %s\n\n", team.Name))
	b.WriteString(fmt.Sprintf("Country: %s\n", team.Country))
	b.WriteString(fmt.Sprintf("Reports to: %s\n", team.Lead))
	b.WriteString(fmt.Sprintf("Area: %s\n\n", team.Area))
	b.WriteString(fmt.Sprintf("## Function\n\n%s\n\n", team.Function))
	b.WriteString("## Actions\n\n")
	for _, action := range team.Actions {
		b.WriteString(fmt.Sprintf("- %s\n", action))
	}

	return []byte(b.String()), nil
}
