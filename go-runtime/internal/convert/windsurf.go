package convert

import (
	"fmt"
	"strings"
)

// WindsurfConverter transforms OVAV canonical agents into Windsurf .md files.
//
// Windsurf uses .windsurf/agents/ directory with .md files.
// Format follows Windsurf's agent definition conventions.
type WindsurfConverter struct{}

func (c *WindsurfConverter) FileExtension() string { return ".md" }
func (c *WindsurfConverter) OutputDir() string     { return "runtimes/windsurf/agents" }
func (c *WindsurfConverter) AreasOnly() bool        { return true }

func (c *WindsurfConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.Name))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("type: agent\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", area.Color))
	}
	if area.Permission != nil {
		writePermissionBlock(&b, area.Permission)
	}
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

func (c *WindsurfConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil // Windsurf uses AreasOnly mode
}

func (c *WindsurfConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil // Windsurf uses AreasOnly mode
}
