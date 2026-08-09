package convert

import (
	"fmt"
	"strings"
)

// AiderConverter transforms OVAV canonical agents into Aider session prompts.
//
// Aider is CLI-focused and uses system prompts for agent behavior.
type AiderConverter struct{}

func (c *AiderConverter) FileExtension() string { return ".md" }
func (c *AiderConverter) OutputDir() string     { return "runtimes/aider/agents" }
func (c *AiderConverter) AreasOnly() bool        { return true }

func (c *AiderConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
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

func (c *AiderConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil
}

func (c *AiderConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil
}
