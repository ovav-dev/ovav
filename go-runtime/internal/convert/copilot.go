package convert

import (
	"fmt"
	"strings"
)

// CopilotConverter transforms OVAV canonical agents into GitHub Copilot Chat templates.
//
// Copilot uses VS Code's agent chat format with instructions embedded
// in the active instruction set.
type CopilotConverter struct{}

func (c *CopilotConverter) FileExtension() string { return ".md" }
func (c *CopilotConverter) OutputDir() string     { return "runtimes/copilot/agents" }
func (c *CopilotConverter) AreasOnly() bool        { return true }

func (c *CopilotConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", area.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("type: agent\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	b.WriteString("---\n\n")

	WriteIdentityGuard(&b, area.Name)
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("# %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("**Lead:** %s\n\n", area.Lead))
	b.WriteString("## Capabilities\n\n")
	for _, fn := range area.Functions {
		b.WriteString(fmt.Sprintf("- %s\n", fn))
	}
	b.WriteString("\n## Constraints\n\n")
	for _, lim := range area.Limitations {
		b.WriteString(fmt.Sprintf("- %s\n", lim))
	}
	b.WriteString("\n## Hard Stop\n\n")
	b.WriteString(area.HardStop)
	b.WriteString("\n")

	return []byte(b.String()), nil
}

func (c *CopilotConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil
}

func (c *CopilotConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil
}
