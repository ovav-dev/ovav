package convert

import (
	"fmt"
	"strings"
)

// AiderConverter transforms OVAV canonical areas into Aider system prompts.
//
// Verified format: https://aider.chat/docs
//
// Aider does NOT have named agent files. It uses system prompts via
// the --system-prompts-from CLI flag. These are PLAIN MARKDOWN files
// with NO YAML frontmatter.
//
// Output: runtimes/aider/agents/ as .md files (system prompt templates).
// AreasOnly=true: Aider has no hierarchical agent system.
//
// Usage: aider --system-prompts-from runtimes/aider/agents/
type AiderConverter struct{}

func (c *AiderConverter) FileExtension() string { return ".md" }
func (c *AiderConverter) OutputDir() string     { return "runtimes/aider/agents" }
func (c *AiderConverter) AreasOnly() bool       { return true }

func (c *AiderConverter) ConvertArea(area *Area, _ map[string]*Lead) ([]byte, error) {
	var b strings.Builder

	// NO YAML frontmatter for Aider — plain markdown system prompt
	b.WriteString(fmt.Sprintf("# %s\n\n", area.Name))
	b.WriteString(fmt.Sprintf("**Lead:** %s\n\n", area.Lead))
	if area.Surface != "" {
		b.WriteString(fmt.Sprintf("**Surface:** %s\n\n", area.Surface))
	}
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
