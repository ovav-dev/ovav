package convert

import (
	"fmt"
	"strings"
)

// WindsurfConverter transforms OVAV canonical agents into Windsurf/Devin .md files.
//
// Verified format: https://docs.devin.ai
//
// Windsurf legacy (.windsurf/agents/):  name, description, type, color
// Devin modern (.devin/agents/):        name, description, model, allowed-tools, max-nesting
//
// OVAV targets Windsurf legacy format since it's the established convention.
// For Devin, the same .md format works — just copy to .devin/agents/.
//
// AreasOnly=true: Windsurf/Devin don't support hierarchical lead/team agents.
type WindsurfConverter struct{}

func (c *WindsurfConverter) FileExtension() string { return ".md" }
func (c *WindsurfConverter) OutputDir() string     { return "runtimes/windsurf/agents" }
func (c *WindsurfConverter) AreasOnly() bool        { return true }

func (c *WindsurfConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("type: agent\n")
	if area.Color != "" {
		b.WriteString(fmt.Sprintf("color: %q\n", hexToNamedColor(area.Color)))
	}
	// Permission block — Windsurf uses simple permission.edit format
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

func (c *WindsurfConverter) ConvertLead(_ *Lead) ([]byte, error) {
	return nil, nil // Windsurf AreasOnly mode
}

func (c *WindsurfConverter) ConvertTeam(_ *TeamMember) ([]byte, error) {
	return nil, nil // Windsurf AreasOnly mode
}

// hexToNamedColor converts hex colors to Windsurf/Devin named colors.
// Supported: red, blue, green, yellow, purple, orange, pink, cyan, black
func hexToNamedColor(hex string) string {
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
		"000000": "black",
		"1f2937": "black",
	}
	if named, ok := colorMap[hex]; ok {
		return named
	}
	return "blue" // default fallback
}
