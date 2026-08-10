package convert

import (
	"fmt"
	"strings"
)

// CopilotConverter transforms OVAV canonical agents into GitHub Copilot agent files.
//
// Verified format: https://docs.github.com/en/copilot/building-copilot-extensions
//
// Agent files go to .github/agents/ (NOT runtimes/copilot/).
// File extension: .agent.md
// Fields: description, name, argument-hint, tools, agents, model,
//          user-invocable, disable-model-invocation, target,
//          mcp-servers, handoffs, hooks
//
// AreasOnly=true: Copilot doesn't support hierarchical agent hierarchies.
type CopilotConverter struct{}

func (c *CopilotConverter) FileExtension() string { return ".agent.md" }
func (c *CopilotConverter) OutputDir() string     { return ".github/agents" }
func (c *CopilotConverter) AreasOnly() bool        { return true }

func (c *CopilotConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("name: %q\n", area.ID))
	b.WriteString(fmt.Sprintf("description: %q\n", area.Description))
	b.WriteString("user-invocable: true\n")
	b.WriteString("disable-model-invocation: false\n")
	b.WriteString("target: vscode\n")
	// Permission block
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
