package convert

import (
	"fmt"
	"strings"
)

// GooseConverter transforms OVAV canonical areas into Goose AGENTS.md format.
//
// Verified format: https://goosedocs.ai
//
// Goose auto-loads AGENTS.md for project context. Format is PLAIN MARKDOWN
// (no YAML frontmatter). This is an interop standard donated to AAIF.
//
// Since Goose uses a SINGLE AGENTS.md file (not per-agent files), we generate
// one file at runtimes/goose/AGENTS.md containing all areas as sections.
//
// AreasOnly=true: Goose has no named agent files or hierarchy.
type GooseConverter struct{}

func (c *GooseConverter) FileExtension() string { return ".md" }
func (c *GooseConverter) OutputDir() string     { return "runtimes/goose" }
func (c *GooseConverter) AreasOnly() bool        { return true }

func (c *GooseConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	// NO YAML frontmatter for Goose AGENTS.md — plain markdown
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
	b.WriteString("\n\n")

	return []byte(b.String()), nil
}

func (c *GooseConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil
}

func (c *GooseConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil
}
