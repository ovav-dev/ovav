package convert

import (
	"fmt"
	"strings"
)

// ContinueConverter transforms OVAV canonical agents into Continue.dev agent files.
//
// Verified format: https://docs.continue.dev
//
// Directory: .continue/agents/
// File extension: .md
// Fields: name, description ONLY
//
// OVAV extends with body content for Functions, Limitations, Hard Stop.
type ContinueConverter struct{}

func (c *ContinueConverter) FileExtension() string { return ".md" }
func (c *ContinueConverter) OutputDir() string     { return ".continue/agents" }
func (c *ContinueConverter) AreasOnly() bool        { return true }

func (c *ContinueConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("---\n\n")

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

func (c *ContinueConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil
}

func (c *ContinueConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil
}
