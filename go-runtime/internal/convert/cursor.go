package convert

import (
	"fmt"
	"strings"
)

// CursorConverter transforms OVAV canonical agents into Cursor rules.
//
// Cursor uses .md files with YAML frontmatter in .cursor/rules/.
//
// Verified format: https://cursor.com/docs/customcursor
//
// Frontmatter fields for rules (.mdc):
//   description, alwaysApply (bool), globs (string array)
//
// Frontmatter fields for agents (.cursor/agents/*.md):
//   name, description, model, readonly (bool), is_background (bool)
//
// Since OVAV generates areas as rules AND leads/teams as agents,
// we output everything as .md files with the appropriate frontmatter.
// The frontmatter `name` field distinguishes agents from rules (rules
// omit it, agents include it).
type CursorConverter struct{}

func (c *CursorConverter) FileExtension() string { return ".md" }
func (c *CursorConverter) OutputDir() string     { return "runtimes/cursor" }

// AreasOnly returns false: leads and teams are generated as agents.
func (c *CursorConverter) AreasOnly() bool { return false }

// ConvertArea generates a Cursor rule (.md file).
func (c *CursorConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("alwaysApply: false\n")
	// Permission block — Cursor permission format
	if area.Permission != nil {
		b.WriteString("permission:\n")
		b.WriteString(fmt.Sprintf("  edit: %q\n", area.Permission.Edit))
		if len(area.Permission.Bash) > 0 {
			b.WriteString("  bash:\n")
			for k, v := range area.Permission.Bash {
				if strings.ContainsAny(k, " *:") {
					b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
				} else {
					b.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
				}
			}
		}
		if len(area.Permission.ExternalDirectory) > 0 {
			b.WriteString("  external_directory:\n")
			for k, v := range area.Permission.ExternalDirectory {
				b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
			}
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# OVAV Area: %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("Lead: %s\n\n", area.Lead))
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

// ConvertLead generates a Cursor agent (.md file).
func (c *CursorConverter) ConvertLead(lead *Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", lead.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", lead.Description))
	b.WriteString("readonly: false\n")
	b.WriteString("is_background: false\n")
	// Permission block
	if lead.Permission != nil {
		b.WriteString("permission:\n")
		b.WriteString(fmt.Sprintf("  edit: %q\n", lead.Permission.Edit))
		if len(lead.Permission.Bash) > 0 {
			b.WriteString("  bash:\n")
			for k, v := range lead.Permission.Bash {
				if strings.ContainsAny(k, " *:") {
					b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
				} else {
					b.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
				}
			}
		}
		if len(lead.Permission.ExternalDirectory) > 0 {
			b.WriteString("  external_directory:\n")
			for k, v := range lead.Permission.ExternalDirectory {
				b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
			}
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, lead.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s — %s\n\n", lead.Name, lead.DisplayName))
	b.WriteString(fmt.Sprintf("Origin: %s\n", lead.Origin))
	if lead.Authority != "" {
		b.WriteString(fmt.Sprintf("Authority: %s\n", lead.Authority))
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

	if lead.Criteria != "" {
		b.WriteString("\n## Decision Criteria\n\n")
		b.WriteString(lead.Criteria)
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}

// ConvertTeam generates a Cursor team agent (.md file).
func (c *CursorConverter) ConvertTeam(team *TeamMember) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", team.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", team.Function))
	b.WriteString("readonly: true\n")
	b.WriteString("is_background: false\n")
	if team.Model != "" {
		b.WriteString(fmt.Sprintf("model: %q\n", team.Model))
	}
	// Permission block
	if team.Permission != nil {
		b.WriteString("permission:\n")
		b.WriteString(fmt.Sprintf("  edit: %q\n", team.Permission.Edit))
		if len(team.Permission.Bash) > 0 {
			b.WriteString("  bash:\n")
			for k, v := range team.Permission.Bash {
				if strings.ContainsAny(k, " *:") {
					b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
				} else {
					b.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
				}
			}
		}
		if len(team.Permission.ExternalDirectory) > 0 {
			b.WriteString("  external_directory:\n")
			for k, v := range team.Permission.ExternalDirectory {
				b.WriteString(fmt.Sprintf("    %q: %q\n", k, v))
			}
		}
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, team.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", team.Name))
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
