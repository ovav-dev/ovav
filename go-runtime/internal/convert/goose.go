package convert

import (
	"fmt"
	"strings"
)

// GooseConverter transforms OVAV canonical agents into Goose session prompts.
//
// Goose is a newly open-sourced CLI agent focused on task automation.
type GooseConverter struct{}

func (c *GooseConverter) FileExtension() string { return ".md" }
func (c *GooseConverter) OutputDir() string     { return "runtimes/goose/agents" }
func (c *GooseConverter) AreasOnly() bool        { return true }

func (c *GooseConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("# " + area.Name + "\n\n")
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

func (c *GooseConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil
}

func (c *GooseConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil
}
